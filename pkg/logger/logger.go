package logger

import (
	"fmt"
	"log"
	"strings"
)

type Level int

const (
	LevelNone Level = iota
	LevelInfo
	LevelDebug
)

var currentLevel = LevelInfo

// SetLevel sets the global logging level based on string input: "none", "info", "debug".
func SetLevel(levelStr string) {
	switch strings.ToLower(strings.TrimSpace(levelStr)) {
	case "none", "off", "0", "false":
		currentLevel = LevelNone
	case "debug", "2", "verbose":
		currentLevel = LevelDebug
	case "info", "1", "true", "":
		currentLevel = LevelInfo
	default:
		currentLevel = LevelInfo
	}
}

// GetLevel returns the current log level.
func GetLevel() Level {
	return currentLevel
}

// Info logs a message if the log level is Info or Debug.
func Info(format string, v ...interface{}) {
	if currentLevel >= LevelInfo {
		log.Printf("[INFO] "+format, v...)
	}
}

// Debug logs a message if the log level is Debug.
func Debug(format string, v ...interface{}) {
	if currentLevel >= LevelDebug {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// Print logs raw messages regardless of log level, used for outputs.
func Print(format string, v ...interface{}) {
	fmt.Printf(format+"\n", v...)
}
