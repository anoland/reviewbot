package evaluator

import (
	"fmt"
	"strings"

	"forgejo-test-evaluator/pkg/client"
	"forgejo-test-evaluator/pkg/models"
)

type ChangedFile struct {
	Path        string
	IsTest      bool
	Diff        string
	FullContent string
}

type AnalysisContext struct {
	AppFiles  []ChangedFile
	TestFiles []ChangedFile
}

// IsTestFile checks if a file path is likely a test file.
func IsTestFile(path string) bool {
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, "_test.go") ||
		strings.HasSuffix(lower, ".test.js") ||
		strings.HasSuffix(lower, ".test.ts") ||
		strings.HasSuffix(lower, ".spec.js") ||
		strings.HasSuffix(lower, ".spec.ts") ||
		strings.Contains(lower, "/tests/") ||
		strings.Contains(lower, "/test/") {
		return true
	}
	return false
}

// ParseDiffFiles parses raw unified diff output into file diff map.
func ParseDiffFiles(diff string) map[string]string {
	fileDiffs := make(map[string]string)
	lines := strings.Split(diff, "\n")

	var currentFile string
	var currentDiff []string

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			if currentFile != "" {
				fileDiffs[currentFile] = strings.Join(currentDiff, "\n")
			}
			parts := strings.Split(line, " ")
			if len(parts) >= 4 {
				// e.g. b/path/to/file.go
				currentFile = strings.TrimPrefix(parts[3], "b/")
			} else {
				currentFile = ""
			}
			currentDiff = []string{line}
		} else if currentFile != "" {
			currentDiff = append(currentDiff, line)
		}
	}

	if currentFile != "" {
		fileDiffs[currentFile] = strings.Join(currentDiff, "\n")
	}

	return fileDiffs
}

// AssembleContext fetches file contents for modified files and separates them into App vs Test files.
func AssembleContext(forgejoClient client.ForgejoClient, owner, repo, ref, diff string) (*AnalysisContext, error) {
	fileDiffs := ParseDiffFiles(diff)
	ctx := &AnalysisContext{}

	for path, fileDiff := range fileDiffs {
		if path == "" || path == "/dev/null" {
			continue
		}

		fullContent, _ := forgejoClient.GetPRFileContent(owner, repo, ref, path)

		cf := ChangedFile{
			Path:        path,
			IsTest:      IsTestFile(path),
			Diff:        fileDiff,
			FullContent: fullContent,
		}

		if cf.IsTest {
			ctx.TestFiles = append(ctx.TestFiles, cf)
		} else {
			ctx.AppFiles = append(ctx.AppFiles, cf)
		}
	}

	return ctx, nil
}

// BuildUserPrompt generates the user prompt given the context and prior state.
func BuildUserPrompt(ctx *AnalysisContext, priorState *models.State) string {
	var sb strings.Builder

	sb.WriteString("## PR Changeset Context\n\n")

	sb.WriteString("### Application Code Files:\n")
	for _, f := range ctx.AppFiles {
		sb.WriteString(fmt.Sprintf("#### File: `%s`\n", f.Path))
		sb.WriteString("```diff\n")
		sb.WriteString(f.Diff)
		sb.WriteString("\n```\n")
		if f.FullContent != "" {
			sb.WriteString("<details><summary>Full File Content</summary>\n\n```\n")
			sb.WriteString(f.FullContent)
			sb.WriteString("\n```\n</details>\n\n")
		}
	}

	sb.WriteString("### Test Code Files:\n")
	for _, f := range ctx.TestFiles {
		sb.WriteString(fmt.Sprintf("#### File: `%s`\n", f.Path))
		sb.WriteString("```diff\n")
		sb.WriteString(f.Diff)
		sb.WriteString("\n```\n")
		if f.FullContent != "" {
			sb.WriteString("<details><summary>Full File Content</summary>\n\n```\n")
			sb.WriteString(f.FullContent)
			sb.WriteString("\n```\n</details>\n\n")
		}
	}

	if priorState != nil {
		sb.WriteString("### Prior Evaluation State:\n")
		priorJSON, _ := priorState.ToJSON()
		sb.WriteString("```json\n")
		sb.WriteString(priorJSON)
		sb.WriteString("\n```\n")
	}

	return sb.String()
}
