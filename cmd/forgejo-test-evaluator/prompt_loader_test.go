package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSystemPromptEmbedded(t *testing.T) {
	prompt, err := LoadSystemPrompt("")
	if err != nil {
		t.Fatalf("unexpected error loading embedded system prompt: %v", err)
	}

	if len(prompt) == 0 {
		t.Fatal("expected non-empty embedded system prompt")
	}
}

func TestLoadSystemPromptFileOverride(t *testing.T) {
	tmpDir := t.TempDir()
	customFile := filepath.Join(tmpDir, "custom.md")
	customContent := "Custom prompt instruction override"

	if err := os.WriteFile(customFile, []byte(customContent), 0644); err != nil {
		t.Fatalf("failed to write temp custom prompt: %v", err)
	}

	prompt, err := LoadSystemPrompt(customFile)
	if err != nil {
		t.Fatalf("unexpected error loading custom prompt: %v", err)
	}

	if prompt != customContent {
		t.Errorf("expected '%s', got '%s'", customContent, prompt)
	}
}
