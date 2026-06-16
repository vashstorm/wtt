package cli

import (
	"errors"
	"fmt"
	"io"

	"wtt/internal/git"
	"wtt/internal/run"
)

// handleDelete implements the delete-worktree and delete-worktree-and-branch flows.
// callerCwd is the working directory of the calling process, captured once at startup.
// It returns 0 on success, 1 on error.
// All output goes to stderr only — delete modes never emit to stdout.
func handleDelete(parsed *ParsedArgs, stderr io.Writer, callerCwd string) int {
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

	wt, err := svc.FindWorktree(repoRoot, parsed.Name)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if wt == nil {
		fmt.Fprintf(stderr, "worktree not found: %s\n", parsed.Name)
		return 1
	}

	// Remove the worktree (no --force).
	if err := svc.RemoveWorktree(repoRoot, wt.Path); err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	// For -D: also delete the branch.
	// If branch deletion fails, report the error but do NOT undo the worktree removal.
	if parsed.Op == OpDeleteWorktreeAndBranch {
		if err := svc.DeleteBranch(repoRoot, parsed.Name); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
	}

	return 0
}
