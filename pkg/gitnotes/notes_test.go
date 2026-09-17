package gitnotes_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/anoland/reviewbot/pkg/gitnotes"
	"github.com/anoland/reviewbot/pkg/models"
)

type MockGitRunner struct {
	notes map[string]string
	pushed bool
}

func NewMockGitRunner() *MockGitRunner {
	return &MockGitRunner{
		notes: make(map[string]string),
	}
}

func (m *MockGitRunner) Run(args ...string) (string, error) {
	cmdStr := strings.Join(args, " ")

	if strings.HasPrefix(cmdStr, "fetch") {
		return "", nil
	}

	if strings.Contains(cmdStr, "notes") && strings.Contains(cmdStr, "show") {
		sha := args[len(args)-1]
		val, ok := m.notes[sha]
		if !ok {
			return "", fmt.Errorf("no note found for SHA %s", sha)
		}
		return val, nil
	}

	if strings.Contains(cmdStr, "notes") && strings.Contains(cmdStr, "add") {
		// args structure: [-c, user.name=..., -c, user.email=..., notes, --ref=..., add, -f, -m, <json>, <sha>]
		var message string
		var sha string
		for i, arg := range args {
			if arg == "-m" && i+1 < len(args) {
				message = args[i+1]
			}
		}
		sha = args[len(args)-1]
		m.notes[sha] = message
		return "", nil
	}

	if strings.HasPrefix(cmdStr, "push") {
		m.pushed = true
		return "", nil
	}

	return "", nil
}

func TestGitNotesManager(t *testing.T) {
	mockRunner := NewMockGitRunner()
	mgr := gitnotes.NewManager(mockRunner)

	if err := mgr.FetchNotes(); err != nil {
		t.Fatalf("FetchNotes failed: %v", err)
	}

	sha := "1234567890abcdef"
	state := &models.State{
		CommitSHA:     sha,
		Timestamp:     time.Now(),
		OverallRating: 3,
		Summary:       "Test summary",
	}

	if err := mgr.WriteState(sha, state); err != nil {
		t.Fatalf("WriteState failed: %v", err)
	}

	if !mockRunner.pushed {
		t.Errorf("expected git push to have been executed")
	}

	readBack, err := mgr.ReadState(sha)
	if err != nil {
		t.Fatalf("ReadState failed: %v", err)
	}

	if readBack.CommitSHA != sha {
		t.Errorf("expected SHA %s, got %s", sha, readBack.CommitSHA)
	}
	if readBack.OverallRating != 3 {
		t.Errorf("expected rating 3, got %d", readBack.OverallRating)
	}
}
