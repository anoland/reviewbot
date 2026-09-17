package main

import (
	"embed"
	"fmt"
	"os"
)

//go:embed prompts/system_instruction.md
var defaultPromptFS embed.FS

// LoadSystemPrompt returns the system prompt, overriding with promptFilePath if specified and non-empty.
func LoadSystemPrompt(promptFilePath string) (string, error) {
	if promptFilePath != "" {
		data, err := os.ReadFile(promptFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to read custom prompt file %s: %w", promptFilePath, err)
		}
		return string(data), nil
	}

	data, err := defaultPromptFS.ReadFile("prompts/system_instruction.md")
	if err != nil {
		return "", fmt.Errorf("failed to load embedded system prompt: %w", err)
	}

	return string(data), nil
}
