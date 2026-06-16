package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"wtt/internal/git"
	"wtt/internal/run"
	"wtt/internal/tmux"
)

// handleCreate implements the create-worktree and existing-worktree flows.
// callerCwd is the working directory of the calling process, captured once at startup.
// It returns 0 on success, 1 on error.
func handleCreate(parsed *ParsedArgs, stdout, stderr io.Writer, callerCwd string) int {
	svc := git.NewService(run.NewRealRunner())

	repoRoot, err := svc.RepoRoot(callerCwd)
	if err != nil {
		if errors.Is(err, git.ErrNotRepo) {
			fmt.Fprintln(stderr, "not inside a git repository")
			return 1
		}
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	// Existing worktree flow (-e flag).
	if parsed.Existing {
		wt, err := svc.FindWorktree(repoRoot, parsed.Name)
		if err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		if wt == nil {
			fmt.Fprintf(stderr, "worktree not found: %s\n", parsed.Name)
			return 1
		}
		if !parsed.WithTmux {
			fmt.Fprintln(stdout, FormatCDCommand(wt.Path))
		} else {
			sess := tmux.NewSession(run.NewRealRunner())
			if err := sess.EnsureSession(parsed.Name, wt.Path); err != nil {
				fmt.Fprintf(stderr, "tmux error: %v\n", err)
				return 1
			}
		}
		return 0
	}

	// Create new worktree flow.

	// Compute base directory.
	var baseDir string
	if parsed.CustomBase != "" {
		// Resolve custom base: absolute paths used as-is, relative paths resolved
		// against callerCwd (not os.Getwd, which may differ in tests).
		if filepath.IsAbs(parsed.CustomBase) {
			baseDir = filepath.Clean(parsed.CustomBase)
		} else {
			baseDir = filepath.Clean(filepath.Join(callerCwd, parsed.CustomBase))
		}
	} else {
		baseDir = filepath.Join(repoRoot, ".claude", "worktree")
	}

	finalPath := filepath.Join(baseDir, parsed.Name)

	syncEntriesToCopy, err := resolveSyncEntries(callerCwd, parsed.SyncSpecs)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	// Ensure base directory exists before creating worktree.
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "error creating base directory: %v\n", err)
		return 1
	}

	// Check if worktree already exists (duplicate without -e is an error).
	existing, err := svc.FindWorktree(repoRoot, parsed.Name)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if existing != nil {
		fmt.Fprintf(stderr, "worktree already exists: %s\n", parsed.Name)
		return 1
	}

	// Validate branch name before asking git to create it.
	if err := svc.ValidateBranchName(parsed.Name); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	// Create the worktree.
	if err := svc.CreateWorktree(repoRoot, finalPath, parsed.Name); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	if err := syncEntries(syncEntriesToCopy, finalPath); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	if !parsed.WithTmux {
		fmt.Fprintln(stdout, FormatCDCommand(finalPath))
	} else {
		sess := tmux.NewSession(run.NewRealRunner())
		if err := sess.EnsureSession(parsed.Name, finalPath); err != nil {
			fmt.Fprintf(stderr, "tmux error: %v\n", err)
			return 1
		}
	}
	return 0
}
