package models

import (
	"encoding/json"
	"time"
)

// ItemStatus represents the delta status of an action item across commit iterations.
type ItemStatus string

const (
	StatusUnresolved        ItemStatus = "UNRESOLVED"
	StatusPartiallyAddressed ItemStatus = "PARTIALLY_ADDRESSED"
	StatusResolved          ItemStatus = "RESOLVED"
	StatusJustified         ItemStatus = "JUSTIFIED"
)

// SecurityCoverage contains identified sensitive areas and missing security test coverage.
type SecurityCoverage struct {
	SensitiveAreasIdentified []string `json:"sensitive_areas_identified"`
	GapsFound                []string `json:"gaps_found"`
}

// ActionItem represents a recommended improvement or test coverage gap.
type ActionItem struct {
	ID          string     `json:"id"`
	File        string     `json:"file,omitempty"`
	Line        int        `json:"line,omitempty"`
	Description string     `json:"description"`
	Severity    string     `json:"severity"` // "CRITICAL", "HIGH", "MEDIUM", "LOW"
	Status      ItemStatus `json:"status,omitempty"`
	Note        string     `json:"note,omitempty"`
}

// InlineComment represents a specific code comment recommendation.
type InlineComment struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Comment string `json:"comment"`
}

// EvaluationResult represents the direct AI response structured payload.
type EvaluationResult struct {
	OverallRating    int              `json:"overall_rating"`
	RatingCategory   string           `json:"rating_category"`
	Summary          string           `json:"summary"`
	SecurityCoverage SecurityCoverage `json:"security_coverage"`
	ActionItems      []ActionItem     `json:"action_items"`
	InlineComments   []InlineComment  `json:"inline_comments"`
}

// State represents the complete JSON structure persisted in Git Notes (refs/notes/ai-test-bot).
type State struct {
	CommitSHA        string           `json:"commit_sha"`
	Timestamp        time.Time        `json:"timestamp"`
	OverallRating    int              `json:"overall_rating"`
	RatingCategory   string           `json:"rating_category"`
	Summary          string           `json:"summary"`
	SecurityCoverage SecurityCoverage `json:"security_coverage"`
	ActionItems      []ActionItem     `json:"action_items"`
	PreviousSHA      string           `json:"previous_sha,omitempty"`
}

// ToJSON marshals State into formatted JSON string.
func (s *State) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// FromJSON unmarshals JSON string into State.
func FromJSON(data string) (*State, error) {
	var state State
	err := json.Unmarshal([]byte(data), &state)
	if err != nil {
		return nil, err
	}
	return &state, nil
}
