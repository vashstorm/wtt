package git

import (
	"errors"
	"fmt"
	"strings"

	"wtt/internal/run"
)

// ErrNotRepo indicates the current directory is not inside a git repository.
var ErrNotRepo = errors.New("not inside a git repository")

// Service provides git operations via a Runner.
type Service struct {
	runner run.Runner
}

// NewService creates a git Service backed by the given Runner.
func NewService(runner run.Runner) *Service {
	return &Service{runner: runner}
}

// RepoRoot discovers the git repository root from callerCwd.
// It runs git rev-parse --show-toplevel from callerCwd.
// Returns ErrNotRepo if the directory is not inside a git repository.
func (s *Service) RepoRoot(callerCwd string) (string, error) {
	out, err := s.runner.RunWithOpts(run.RunOpts{Dir: callerCwd}, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		if strings.Contains(err.Error(), "not a git repository") {
			return "", ErrNotRepo
		}
		return "", fmt.Errorf("discover repo root from %q: %w", callerCwd, err)
	}
	if out == "" {
		return "", ErrNotRepo
	}
	return out, nil
}

// ValidateBranchName checks whether name is a valid git branch name.
// It runs git check-ref-format --branch <name>.
func (s *Service) ValidateBranchName(name string) error {
	_, err := s.runner.Run("git", "check-ref-format", "--branch", name)
	if err != nil {
		return fmt.Errorf("invalid branch name %q: %w", name, err)
	}
	return nil
}
