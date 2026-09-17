package gitnotes

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/anoland/reviewbot/pkg/models"
)

const (
	NotesRef      = "refs/notes/ai-test-bot"
	BotAuthorName = "Forgejo AI Test Bot"
	BotEmail      = "ai-test-bot@forgejo.local"
)

type GitRunner interface {
	Run(args ...string) (string, error)
}

type ExecGitRunner struct{}

func (e *ExecGitRunner) Run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w (stderr: %s)", strings.Join(args, " "), err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

type Manager struct {
	runner GitRunner
}

func NewManager(runner GitRunner) *Manager {
	if runner == nil {
		runner = &ExecGitRunner{}
	}
	return &Manager{runner: runner}
}

// FetchNotes fetches the ai-test-bot git notes ref from origin.
func (m *Manager) FetchNotes() error {
	_, err := m.runner.Run("fetch", "origin", fmt.Sprintf("%s:%s", NotesRef, NotesRef))
	if err != nil {
		// Non-zero exit is expected if notes ref does not exist yet on origin
		return nil
	}
	return nil
}

// ReadState reads and unmarshals the JSON state attached to the given commit SHA.
func (m *Manager) ReadState(commitSHA string) (*models.State, error) {
	if commitSHA == "" {
		return nil, fmt.Errorf("commit SHA cannot be empty")
	}

	out, err := m.runner.Run("notes", fmt.Sprintf("--ref=%s", NotesRef), "show", commitSHA)
	if err != nil {
		return nil, fmt.Errorf("no git notes state found for commit %s: %w", commitSHA, err)
	}

	return models.FromJSON(out)
}

// ReadPriorState walks back commit history from commitSHA to find the most recent prior git note.
func (m *Manager) ReadPriorState(commitSHA string) (*models.State, error) {
	if commitSHA == "" {
		commitSHA = "HEAD"
	}

	// Retrieve list of parent commit SHAs
	out, err := m.runner.Run("rev-list", fmt.Sprintf("%s~1", commitSHA), "-n", "10")
	if err != nil || strings.TrimSpace(out) == "" {
		return nil, fmt.Errorf("no prior commits found before %s", commitSHA)
	}

	parentSHAs := strings.Split(strings.TrimSpace(out), "\n")
	for _, sha := range parentSHAs {
		sha = strings.TrimSpace(sha)
		if sha == "" {
			continue
		}
		state, err := m.ReadState(sha)
		if err == nil && state != nil {
			return state, nil
		}
	}

	return nil, fmt.Errorf("no prior git notes state found in history before %s", commitSHA)
}

// WriteState marshals the state to JSON, attaches it to commitSHA, and pushes to origin.
func (m *Manager) WriteState(commitSHA string, state *models.State) error {
	if commitSHA == "" {
		return fmt.Errorf("commit SHA cannot be empty")
	}

	jsonStr, err := state.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Set bot git committer/author identity environment
	_, err = m.runner.Run(
		"-c", fmt.Sprintf("user.name=%s", BotAuthorName),
		"-c", fmt.Sprintf("user.email=%s", BotEmail),
		"notes", fmt.Sprintf("--ref=%s", NotesRef),
		"add", "-f", "-m", jsonStr, commitSHA,
	)
	if err != nil {
		return fmt.Errorf("failed to add git note to commit %s: %w", commitSHA, err)
	}

	// Push notes ref to origin
	_, err = m.runner.Run("push", "origin", NotesRef)
	if err != nil {
		return fmt.Errorf("failed to push git notes ref to origin: %w", err)
	}

	return nil
}
