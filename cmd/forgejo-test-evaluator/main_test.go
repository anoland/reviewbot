package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_KEY_1", "")
	os.Setenv("TEST_KEY_2", "value2")
	defer func() {
		os.Unsetenv("TEST_KEY_1")
		os.Unsetenv("TEST_KEY_2")
	}()

	val := getEnv("TEST_KEY_1", "TEST_KEY_2", "TEST_KEY_3")
	if val != "value2" {
		t.Errorf("expected value2, got %s", val)
	}

	valEmpty := getEnv("NON_EXISTENT_1", "NON_EXISTENT_2")
	if valEmpty != "" {
		t.Errorf("expected empty string, got %s", valEmpty)
	}
}

func TestParseEventPayload(t *testing.T) {
	tmpDir := t.TempDir()
	eventFile := filepath.Join(tmpDir, "event.json")
	payloadJSON := `{
		"pull_request": {
			"number": 42,
			"head": {
				"sha": "1234567890123456789012345678901234567890"
			}
		}
	}`

	if err := os.WriteFile(eventFile, []byte(payloadJSON), 0644); err != nil {
		t.Fatalf("failed to write event payload file: %v", err)
	}

	os.Setenv("GITHUB_EVENT_PATH", eventFile)
	defer os.Unsetenv("GITHUB_EVENT_PATH")

	prNum, headSHA := parseEventPayload()
	if prNum != 42 {
		t.Errorf("expected PR number 42, got %d", prNum)
	}
	if headSHA != "1234567890123456789012345678901234567890" {
		t.Errorf("expected sha '1234567890123456789012345678901234567890', got '%s'", headSHA)
	}
}

func TestAppendStepSummary(t *testing.T) {
	tmpDir := t.TempDir()
	summaryFile := filepath.Join(tmpDir, "step_summary.md")

	os.Setenv("GITHUB_STEP_SUMMARY", summaryFile)
	defer os.Unsetenv("GITHUB_STEP_SUMMARY")

	appendStepSummary("## Evaluation Report")
	appendStepSummary("Summary content")

	data, err := os.ReadFile(summaryFile)
	if err != nil {
		t.Fatalf("failed to read step summary file: %v", err)
	}

	expected := "## Evaluation Report\nSummary content\n"
	if string(data) != expected {
		t.Errorf("expected summary content %q, got %q", expected, string(data))
	}
}
