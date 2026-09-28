package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/anoland/reviewbot/pkg/client"
	"github.com/anoland/reviewbot/pkg/evaluator"
	"github.com/anoland/reviewbot/pkg/gitnotes"
	"github.com/anoland/reviewbot/pkg/logger"
	"github.com/anoland/reviewbot/pkg/models"
)

type githubEventPayload struct {
	PullRequest struct {
		Number int `json:"number"`
		Head   struct {
			SHA string `json:"sha"`
		} `json:"head"`
	} `json:"pull_request"`
}

func getEnv(keys ...string) string {
	for _, key := range keys {
		if val := os.Getenv(key); val != "" {
			return val
		}
	}
	return ""
}

func parseEventPayload() (int, string) {
	eventPath := getEnv("GITHUB_EVENT_PATH", "FORGEJO_EVENT_PATH")
	if eventPath == "" {
		return 0, ""
	}

	data, err := os.ReadFile(eventPath)
	if err != nil {
		return 0, ""
	}

	var payload githubEventPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return 0, ""
	}

	return payload.PullRequest.Number, payload.PullRequest.Head.SHA
}

func appendStepSummary(summaryReport string) {
	summaryFile := getEnv("GITHUB_STEP_SUMMARY", "FORGEJO_STEP_SUMMARY")
	if summaryFile == "" {
		return
	}
	f, err := os.OpenFile(summaryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Debug("Could not open step summary file %s: %v", summaryFile, err)
		return
	}
	defer f.Close()
	_, _ = f.WriteString(summaryReport + "\n")
}

func main() {
	promptFile := flag.String("prompt-file", "", "Path to custom prompt system instruction file")
	forgejoToken := flag.String("forgejo-token", getEnv("FORGEJO_TOKEN", "GITHUB_TOKEN"), "Forgejo API Token")
	serverURL := flag.String("forgejo-server-url", getEnv("FORGEJO_SERVER_URL", "GITHUB_SERVER_URL"), "Forgejo Server Base URL")
	repository := flag.String("repository", getEnv("FORGEJO_REPOSITORY", "GITHUB_REPOSITORY"), "Repository owner/repo")
	prNumberStr := flag.String("pr-number", getEnv("FORGEJO_PR_NUMBER", "GITHUB_PR_NUMBER"), "Pull Request Number")
	ref := flag.String("ref", getEnv("FORGEJO_REF", "GITHUB_REF"), "Git Ref or Branch")
	headSHA := flag.String("head-sha", getEnv("FORGEJO_HEAD_SHA", "GITHUB_SHA"), "Head Commit SHA")
	geminiAPIKey := flag.String("gemini-api-key", getEnv("GEMINI_API_KEY"), "Gemini API Key")
	logLevel := flag.String("log-level", getEnv("LOG_LEVEL", "FORGEJO_LOG_LEVEL"), "Log level verbosity (none, info, debug)")

	flag.Parse()

	logger.SetLevel(*logLevel)

	logger.Info("Starting Forgejo Test Evaluator...")

	eventPRNum, eventHeadSHA := parseEventPayload()

	if *prNumberStr == "" && eventPRNum > 0 {
		*prNumberStr = strconv.Itoa(eventPRNum)
	}
	if (*headSHA == "" || len(*headSHA) < 40) && eventHeadSHA != "" {
		*headSHA = eventHeadSHA
	}

	if *forgejoToken == "" {
		log.Fatalf("Error: forgejo-token is required. Please set --forgejo-token parameter or FORGEJO_TOKEN / GITHUB_TOKEN environment variable.")
	}

	if *geminiAPIKey == "" {
		log.Fatalf("Error: gemini-api-key is required. Please set --gemini-api-key parameter or GEMINI_API_KEY environment variable.")
	}

	if *repository == "" {
		log.Fatalf("Error: repository is a required parameter or environment variable (repo: '%s')", *repository)
	}

	repoParts := strings.Split(*repository, "/")
	if len(repoParts) != 2 {
		log.Fatalf("Invalid repository format %s. Expected owner/repo", *repository)
	}
	owner, repo := repoParts[0], repoParts[1]

	var prNum int
	isPR := false
	if *prNumberStr != "" {
		num, err := strconv.Atoi(*prNumberStr)
		if err == nil && num > 0 {
			prNum = num
			isPR = true
		}
	}

	targetRef := *ref
	if targetRef == "" {
		targetRef = *headSHA
	}

	// Fail fast if neither PR nor ref/branch/headSHA was provided
	if !isPR && targetRef == "" {
		log.Fatalf("Error: Neither pr-number nor ref/branch/head-sha was provided. Cannot identify target for review.")
	}

	systemPrompt, err := LoadSystemPrompt(*promptFile)
	if err != nil {
		log.Fatalf("Error loading system prompt: %v", err)
	}

	gitNotesMgr := gitnotes.NewManager(nil)
	_ = gitNotesMgr.FetchNotes()

	var priorState *models.State
	if *headSHA != "" {
		// Look up prior state from parent commits in Git history
		priorState, _ = gitNotesMgr.ReadPriorState(*headSHA)
		if priorState != nil {
			logger.Debug("Found prior state for commit %s (rating: %d)", priorState.CommitSHA, priorState.OverallRating)
		}
	}

	forgejoCli := client.NewForgejoClient(*serverURL, *forgejoToken)

	var diff string
	var contextRef string

	if isPR {
		logger.Info("Reviewing Pull Request #%d in repository %s", prNum, *repository)
		diff, err = forgejoCli.GetPRDiff(owner, repo, prNum)
		if err != nil {
			logger.Info("Warning: Failed to fetch PR diff: %v", err)
		}
		if *headSHA != "" {
			contextRef = *headSHA
		} else {
			contextRef = fmt.Sprintf("refs/pull/%d/head", prNum)
		}
	} else {
		logger.Info("Reviewing ref/branch/commit '%s' in repository %s", targetRef, *repository)
		diff, err = forgejoCli.GetCommitDiff(owner, repo, targetRef)
		if err != nil {
			logger.Info("Warning: Failed to fetch commit diff for ref '%s': %v", targetRef, err)
		}
		contextRef = targetRef
	}

	logger.Debug("Fetched diff size: %d bytes", len(diff))

	analysisCtx, err := evaluator.AssembleContext(forgejoCli, owner, repo, contextRef, diff)
	if err != nil {
		log.Fatalf("Failed to assemble analysis context: %v", err)
	}

	logger.Info("Context assembled: %d app files, %d test files", len(analysisCtx.AppFiles), len(analysisCtx.TestFiles))

	userPrompt := evaluator.BuildUserPrompt(analysisCtx, priorState)

	logger.Info("Calling review agent (Gemini)...")
	geminiCli := client.NewGeminiClient(*geminiAPIKey)
	evalResult, err := geminiCli.EvaluateChanges(context.Background(), systemPrompt, userPrompt)
	if err != nil {
		logger.Info("Gemini evaluation error (using default fallback state if needed): %v", err)
		evalResult = &models.EvaluationResult{
			OverallRating:  2,
			RatingCategory: "Level 2: Basic Happy-Path",
			Summary:        fmt.Sprintf("Evaluation completed. Notice: %v", err),
		}
	}

	logger.Info("Review agent returned result: Rating %d/4 (%s)", evalResult.OverallRating, evalResult.RatingCategory)

	var prComments []client.PRComment
	if isPR {
		prComments, _ = forgejoCli.GetPRComments(owner, repo, prNum)
	}

	updatedResult := evaluator.CalculateDelta(evalResult, priorState, prComments)

	newState := &models.State{
		CommitSHA:        *headSHA,
		Timestamp:        time.Now().UTC(),
		OverallRating:    updatedResult.OverallRating,
		RatingCategory:   updatedResult.RatingCategory,
		Summary:          updatedResult.Summary,
		SecurityCoverage: updatedResult.SecurityCoverage,
		ActionItems:      updatedResult.ActionItems,
	}
	if priorState != nil {
		newState.PreviousSHA = priorState.CommitSHA
	}

	if *headSHA != "" {
		if err := gitNotesMgr.WriteState(*headSHA, newState); err != nil {
			logger.Info("Warning: Failed to write state to Git Notes: %v", err)
		} else {
			logger.Debug("State successfully written to Git Notes for SHA %s", *headSHA)
		}
	}

	reportSHA := *headSHA
	if reportSHA == "" {
		reportSHA = targetRef
	}

	summaryComment := evaluator.FormatPRSummaryComment(updatedResult, reportSHA)

	if isPR {
		logger.Info("Posting PR summary comment to PR #%d...", prNum)
		if err := forgejoCli.PostPRComment(owner, repo, prNum, summaryComment); err != nil {
			logger.Info("Warning: Failed to post PR summary comment: %v", err)
		}

		for _, inline := range updatedResult.InlineComments {
			_ = forgejoCli.PostPRInlineComment(owner, repo, prNum, inline.Comment, *headSHA, inline.File, inline.Line)
		}
	} else {
		logger.Info("Displaying review results in output logs...")
		logger.Print("\n=================== REVIEW RESULTS ===================")
		logger.Print(summaryComment)
		if len(updatedResult.InlineComments) > 0 {
			logger.Print("### 💬 Inline Comments:")
			for _, inline := range updatedResult.InlineComments {
				logger.Print("- **%s:%d**: %s", inline.File, inline.Line, inline.Comment)
			}
		}
		logger.Print("======================================================\n")
	}

	appendStepSummary(summaryComment)

	logger.Info("Forgejo Test Evaluator completed successfully.")
}
