package evaluator

import (
	"fmt"
	"strings"

	"github.com/anoland/reviewbot/pkg/client"
	"github.com/anoland/reviewbot/pkg/models"
)

// CalculateDelta compares the current evaluation result with the prior commit state and PR comments.
// It tracks item resolution status ([RESOLVED], [PARTIALLY ADDRESSED], [UNRESOLVED], [JUSTIFIED]).
func CalculateDelta(current *models.EvaluationResult, priorState *models.State, prComments []client.PRComment) *models.EvaluationResult {
	if priorState == nil || len(priorState.ActionItems) == 0 {
		// Mark all current action items as UNRESOLVED by default
		for i := range current.ActionItems {
			if current.ActionItems[i].Status == "" {
				current.ActionItems[i].Status = models.StatusUnresolved
			}
		}
		return current
	}

	priorMap := make(map[string]models.ActionItem)
	for _, item := range priorState.ActionItems {
		priorMap[item.ID] = item
		if item.File != "" && item.Line > 0 {
			priorMap[fmt.Sprintf("%s:%d", item.File, item.Line)] = item
		}
	}

	currentMap := make(map[string]bool)
	for i := range current.ActionItems {
		item := &current.ActionItems[i]
		currentMap[item.ID] = true
		if item.File != "" && item.Line > 0 {
			currentMap[fmt.Sprintf("%s:%d", item.File, item.Line)] = true
		}

		// Check if it matches a prior item
		if priorItem, exists := priorMap[item.ID]; exists {
			if priorItem.Status == models.StatusJustified {
				item.Status = models.StatusJustified
				item.Note = "Justified by user comment"
			} else {
				item.Status = models.StatusPartiallyAddressed
				item.Note = "Still present in latest evaluation"
			}
		} else {
			item.Status = models.StatusUnresolved
		}
	}

	// Check if prior items are now missing (meaning resolved or addressed in code)
	for id, priorItem := range priorMap {
		if strings.Contains(id, ":") {
			continue // Skip composite key duplicates
		}
		if !currentMap[id] {
			// Check if user replied with a justification comment
			hasJustification := false
			for _, comment := range prComments {
				if priorItem.File != "" && comment.Path == priorItem.File {
					if priorItem.Line == 0 || comment.Line == priorItem.Line {
						if len(comment.Body) > 0 {
							hasJustification = true
							break
						}
					}
				}
			}

			resolvedStatus := models.StatusResolved
			note := "Addressed in code"
			if hasJustification {
				resolvedStatus = models.StatusJustified
				note = "Justified via user comment"
			}

			current.ActionItems = append(current.ActionItems, models.ActionItem{
				ID:          priorItem.ID,
				File:        priorItem.File,
				Line:        priorItem.Line,
				Description: priorItem.Description,
				Severity:    priorItem.Severity,
				Status:      resolvedStatus,
				Note:        note,
			})
		}
	}

	return current
}

// FormatPRSummaryComment generates the formatted markdown PR summary report.
func FormatPRSummaryComment(result *models.EvaluationResult, commitSHA string) string {
	var sb strings.Builder

	sb.WriteString("## 🧪 Test Effectiveness & Security Evaluation Report\n\n")
	sb.WriteString(fmt.Sprintf("**Commit SHA:** `%s`  \n", commitSHA))
	sb.WriteString(fmt.Sprintf("**Overall Quality Rating:** **%d/4** (%s)  \n\n", result.OverallRating, result.RatingCategory))

	sb.WriteString("### Summary\n")
	sb.WriteString(result.Summary)
	sb.WriteString("\n\n")

	sb.WriteString("### 🔒 Security Surface Coverage\n")
	if len(result.SecurityCoverage.SensitiveAreasIdentified) > 0 {
		sb.WriteString("- **Sensitive Areas Identified:** " + strings.Join(result.SecurityCoverage.SensitiveAreasIdentified, ", ") + "\n")
	} else {
		sb.WriteString("- **Sensitive Areas Identified:** None\n")
	}

	if len(result.SecurityCoverage.GapsFound) > 0 {
		sb.WriteString("- **Security Coverage Gaps:**\n")
		for _, gap := range result.SecurityCoverage.GapsFound {
			sb.WriteString(fmt.Sprintf("  - ⚠️ %s\n", gap))
		}
	} else {
		sb.WriteString("- **Security Coverage Gaps:** None identified\n")
	}
	sb.WriteString("\n")

	sb.WriteString("### 📋 Action Items & Delta Tracking\n")
	if len(result.ActionItems) == 0 {
		sb.WriteString("No outstanding action items.\n")
	} else {
		sb.WriteString("| Status | Severity | Location | Description | Note |\n")
		sb.WriteString("| :--- | :--- | :--- | :--- | :--- |\n")
		for _, item := range result.ActionItems {
			loc := "General"
			if item.File != "" {
				if item.Line > 0 {
					loc = fmt.Sprintf("`%s:%d`", item.File, item.Line)
				} else {
					loc = fmt.Sprintf("`%s`", item.File)
				}
			}
			sb.WriteString(fmt.Sprintf("| **[%s]** | %s | %s | %s | %s |\n",
				item.Status, item.Severity, loc, item.Description, item.Note))
		}
	}

	return sb.String()
}
