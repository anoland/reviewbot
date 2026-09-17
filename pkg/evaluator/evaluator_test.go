package evaluator_test

import (
	"testing"

	"github.com/anoland/reviewbot/pkg/client"
	"github.com/anoland/reviewbot/pkg/evaluator"
	"github.com/anoland/reviewbot/pkg/models"
)

type mockForgejoClient struct {
	client.ForgejoClient
}

func (m *mockForgejoClient) GetPRFileContent(owner, repo, ref, filepath string) (string, error) {
	if filepath == "pkg/auth/login.go" {
		return "package auth\nfunc Login() {}\n", nil
	}
	if filepath == "pkg/auth/login_test.go" {
		return "package auth\nfunc TestLogin(t *testing.T) {}\n", nil
	}
	return "", nil
}

func TestAnalyzerContextAssembly(t *testing.T) {
	mockCli := &mockForgejoClient{}
	diff := `diff --git a/pkg/auth/login.go b/pkg/auth/login.go
--- a/pkg/auth/login.go
+++ b/pkg/auth/login.go
@@ -1 +1 @@
-old
+new
diff --git a/pkg/auth/login_test.go b/pkg/auth/login_test.go
--- a/pkg/auth/login_test.go
+++ b/pkg/auth/login_test.go
@@ -1 +1 @@
-old
+new`

	ctx, err := evaluator.AssembleContext(mockCli, "owner", "repo", "main", diff)
	if err != nil {
		t.Fatalf("AssembleContext failed: %v", err)
	}

	if len(ctx.AppFiles) != 1 || ctx.AppFiles[0].Path != "pkg/auth/login.go" {
		t.Errorf("unexpected AppFiles: %+v", ctx.AppFiles)
	}

	if len(ctx.TestFiles) != 1 || ctx.TestFiles[0].Path != "pkg/auth/login_test.go" {
		t.Errorf("unexpected TestFiles: %+v", ctx.TestFiles)
	}

	prompt := evaluator.BuildUserPrompt(ctx, nil)
	if len(prompt) == 0 {
		t.Errorf("expected non-empty user prompt")
	}
}

func TestCalculateDelta(t *testing.T) {
	priorState := &models.State{
		CommitSHA: "sha1",
		ActionItems: []models.ActionItem{
			{
				ID:          "item-1",
				File:        "pkg/auth/login.go",
				Line:        20,
				Description: "Add missing test assertion",
				Severity:    "HIGH",
				Status:      models.StatusUnresolved,
			},
			{
				ID:          "item-2",
				File:        "pkg/auth/login.go",
				Line:        40,
				Description: "Check nil pointer exception",
				Severity:    "MEDIUM",
				Status:      models.StatusUnresolved,
			},
		},
	}

	currentResult := &models.EvaluationResult{
		OverallRating:  3,
		RatingCategory: "Level 3: Robust Functional",
		Summary:        "Improved test suite",
		ActionItems: []models.ActionItem{
			{
				ID:          "item-1",
				File:        "pkg/auth/login.go",
				Line:        20,
				Description: "Add missing test assertion",
				Severity:    "HIGH",
			},
		},
	}

	comments := []client.PRComment{}

	updated := evaluator.CalculateDelta(currentResult, priorState, comments)

	// item-1 should be PARTIALLY_ADDRESSED
	// item-2 should be RESOLVED (since it's absent from current evaluation)
	if len(updated.ActionItems) != 2 {
		t.Fatalf("expected 2 action items, got %d", len(updated.ActionItems))
	}

	var item1, item2 *models.ActionItem
	for i := range updated.ActionItems {
		if updated.ActionItems[i].ID == "item-1" {
			item1 = &updated.ActionItems[i]
		}
		if updated.ActionItems[i].ID == "item-2" {
			item2 = &updated.ActionItems[i]
		}
	}

	if item1 == nil || item1.Status != models.StatusPartiallyAddressed {
		t.Errorf("expected item-1 to be PARTIALLY_ADDRESSED, got %+v", item1)
	}

	if item2 == nil || item2.Status != models.StatusResolved {
		t.Errorf("expected item-2 to be RESOLVED, got %+v", item2)
	}

	summary := evaluator.FormatPRSummaryComment(updated, "sha2")
	if len(summary) == 0 {
		t.Errorf("expected non-empty formatted summary")
	}
}
