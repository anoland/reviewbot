package models_test

import (
	"testing"
	"time"

	"github.com/anoland/reviewbot/pkg/models"
)

func TestStateJSONSerialization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	original := &models.State{
		CommitSHA:      "abc123456789def0",
		Timestamp:      now,
		OverallRating:  3,
		RatingCategory: "Level 3: Robust Functional",
		Summary:        "Good test coverage for core business logic.",
		SecurityCoverage: models.SecurityCoverage{
			SensitiveAreasIdentified: []string{"Input Validation", "Auth"},
			GapsFound:                []string{"Missing check for expired session"},
		},
		ActionItems: []models.ActionItem{
			{
				ID:          "item-1",
				File:        "pkg/auth/login.go",
				Line:        25,
				Description: "Add test for expired session token",
				Severity:    "HIGH",
				Status:      models.StatusUnresolved,
			},
		},
		PreviousSHA: "0000000000000000",
	}

	jsonStr, err := original.ToJSON()
	if err != nil {
		t.Fatalf("failed to marshal State to JSON: %v", err)
	}

	parsed, err := models.FromJSON(jsonStr)
	if err != nil {
		t.Fatalf("failed to unmarshal State from JSON: %v", err)
	}

	if parsed.CommitSHA != original.CommitSHA {
		t.Errorf("expected CommitSHA %s, got %s", original.CommitSHA, parsed.CommitSHA)
	}

	if parsed.OverallRating != original.OverallRating {
		t.Errorf("expected Rating %d, got %d", original.OverallRating, parsed.OverallRating)
	}

	if len(parsed.ActionItems) != 1 || parsed.ActionItems[0].ID != "item-1" {
		t.Errorf("action items mismatch: %+v", parsed.ActionItems)
	}
}
