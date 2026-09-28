package logger_test

import (
	"testing"

	"github.com/anoland/reviewbot/pkg/logger"
)

func TestSetLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected logger.Level
	}{
		{"none", logger.LevelNone},
		{"off", logger.LevelNone},
		{"0", logger.LevelNone},
		{"info", logger.LevelInfo},
		{"INFO", logger.LevelInfo},
		{"debug", logger.LevelDebug},
		{"DEBUG", logger.LevelDebug},
		{"unknown", logger.LevelInfo},
	}

	for _, tt := range tests {
		logger.SetLevel(tt.input)
		if logger.GetLevel() != tt.expected {
			t.Errorf("SetLevel(%q) = %v; expected %v", tt.input, logger.GetLevel(), tt.expected)
		}
	}
}
