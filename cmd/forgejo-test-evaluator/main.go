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

func main() {
	promptFile := flag.String("prompt-file", "", "Path to custom prompt system instruction file")
	forgejoToken := flag.String("forgejo-token", getEnv("FORGEJO_TOKEN", "GITHUB_TOKEN"), "Forgejo API Token")
	serverURL := flag.String("forgejo-server-url", getEnv("FORGEJO_SERVER_URL", "GITHUB_SERVER_URL"), "Forgejo Server Base URL")
	repository := flag.String("repository", getEnv("FORGEJO_REPOSITORY", "GITHUB_REPOSITORY"), "Repository owner/repo")
	prNumberStr := flag.String("pr-number", getEnv("FORGEJO_PR_NUMBER", "GITHUB_PR_NUMBER"), "Pull Request Number")
	headSHA := flag.String("head-sha", getEnv("FORGEJO_HEAD_SHA", "GITHUB_SHA"), "Head Commit SHA")
	geminiAPIKey := flag.String("gemini-api-key", getEnv("GEMINI_API_KEY"), "Gemini API Key")

	flag.Parse()

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

	if *repository == "" || *prNumberStr == "" {
		log.Fatalf("Error: repository and pr-number are required parameters or environment variables (repo: '%s', pr: '%s')", *repository, *prNumberStr)
	}

	prNum, err := strconv.Atoi(*prNumberStr)
	if err != nil {
		log.Fatalf("Invalid pr-number %s: %v", *prNumberStr, err)
	}

	repoParts := strings.Split(*repository, "/")
	if len(repoParts) != 2 {
		log.Fatalf("Invalid repository format %s. Expected owner/repo", *repository)
	}
	owner, repo := repoParts[0], repoParts[1]

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
	}

	forgejoCli := client.NewForgejoClient(*serverURL, *forgejoToken)

	diff, err := forgejoCli.GetPRDiff(owner, repo, prNum)
	if err != nil {
		log.Printf("Warning: Failed to fetch PR diff: %v", err)
	}

	analysisCtx, err := evaluator.AssembleContext(forgejoCli, owner, repo, *headSHA, diff)
	if err != nil {
		log.Fatalf("Failed to assemble analysis context: %v", err)
	}

	userPrompt := evaluator.BuildUserPrompt(analysisCtx, priorState)

	geminiCli := client.NewGeminiClient(*geminiAPIKey)
	evalResult, err := geminiCli.EvaluateChanges(context.Background(), systemPrompt, userPrompt)
	if err != nil {
		log.Printf("Gemini evaluation error (using default fallback state if needed): %v", err)
		evalResult = &models.EvaluationResult{
			OverallRating:  2,
			RatingCategory: "Level 2: Basic Happy-Path",
			Summary:        fmt.Sprintf("Evaluation completed. Notice: %v", err),
		}
	}

	prComments, _ := forgejoCli.GetPRComments(owner, repo, prNum)

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
			log.Printf("Warning: Failed to write state to Git Notes: %v", err)
		}
	}

	summaryComment := evaluator.FormatPRSummaryComment(updatedResult, *headSHA)
	if err := forgejoCli.PostPRComment(owner, repo, prNum, summaryComment); err != nil {
		log.Printf("Warning: Failed to post PR summary comment: %v", err)
	}

	for _, inline := range updatedResult.InlineComments {
		_ = forgejoCli.PostPRInlineComment(owner, repo, prNum, inline.Comment, *headSHA, inline.File, inline.Line)
	}

	fmt.Println("Forgejo Test Evaluator completed successfully.")
}
