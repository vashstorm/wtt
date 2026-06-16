package tmux

import (
	"fmt"
	"os"

	"wtt/internal/run"
)

// Session provides tmux session operations using a run.Runner.
type Session struct {
	runner run.Runner
}

// NewSession creates a new tmux Session backed by the given runner.
func NewSession(runner run.Runner) *Session {
	return &Session{runner: runner}
}

// IsAvailable checks whether tmux is installed by running `tmux -V`.
// Returns a descriptive error if tmux is not found or the check fails.
func (s *Session) IsAvailable() error {
	_, err := s.runner.Run("tmux", "-V")
	if err != nil {
		return fmt.Errorf("tmux is not available: %w", err)
	}
	return nil
}

// HasSession reports whether a tmux session with the given name exists.
// It runs `tmux has-session -t <name>`. If the session exists (exit 0),
// it returns true. If the session does not exist (non-zero exit), it
// returns false, nil. Other errors are propagated.
func (s *Session) HasSession(name string) (bool, error) {
	_, err := s.runner.Run("tmux", "has-session", "-t", name)
	if err != nil {
		// tmux has-session exits non-zero when the session doesn't exist.
		// Treat any error as "session not found" rather than a fatal error.
		return false, nil
	}
	return true, nil
}

// NewSession creates a detached tmux session with the given name, rooted
// at worktreePath. It runs `tmux new-session -d -s <name> -c <worktreePath>`.
func (s *Session) NewSession(name string, worktreePath string) error {
	_, err := s.runner.Run("tmux", "new-session", "-d", "-s", name, "-c", worktreePath)
	if err != nil {
		return fmt.Errorf("failed to create tmux session %q: %w", name, err)
	}
	return nil
}

// AttachSession connects the current terminal to the named tmux session.
// If the TMUX environment variable is set (inside tmux), it uses
// `tmux switch-client -t <name>`. Otherwise, it uses `tmux attach -t <name>`,
// which is blocking and inherits stdin/stdout/stderr for terminal control.
func (s *Session) AttachSession(name string) error {
	if os.Getenv("TMUX") != "" {
		return s.runner.RunInteractive("tmux", "switch-client", "-t", name)
	}
	return s.runner.RunInteractive("tmux", "attach", "-t", name)
}

// EnsureSession guarantees that a tmux session with the given name exists
// and the terminal is attached to it. It:
//  1. Checks that tmux is available.
//  2. Checks if the session already exists; creates it if not.
//  3. Attaches or switches to the session.
func (s *Session) EnsureSession(name string, worktreePath string) error {
	if err := s.IsAvailable(); err != nil {
		return fmt.Errorf("tmux unavailable: %w", err)
	}

	exists, err := s.HasSession(name)
	if err != nil {
		return fmt.Errorf("failed to check tmux session: %w", err)
	}
	if !exists {
		if err := s.NewSession(name, worktreePath); err != nil {
			return fmt.Errorf("create tmux session: %w", err)
		}
	}

	return s.AttachSession(name)
}
