package git

import (
	"fmt"
	"strings"

	"wtt/internal/run"
)

// WorktreeInfo holds parsed information about a single git worktree.
type WorktreeInfo struct {
	Path      string // Absolute path of the worktree (from "worktree" field)
	Head      string // Current HEAD commit SHA (from "HEAD" field)
	BranchRef string // Full branch ref e.g. "refs/heads/main" (from "branch" field)
	Bare      bool   // True if this is a bare worktree
	Detached  bool   // True if HEAD is detached
}

// ListWorktrees runs git worktree list --porcelain from repoRoot and
// parses the output into a slice of WorktreeInfo.
func (s *Service) ListWorktrees(repoRoot string) ([]WorktreeInfo, error) {
	out, err := s.runner.RunWithOpts(run.RunOpts{Dir: repoRoot}, "git", "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("list worktrees: %w", err)
	}
	return parsePorcelain(out), nil
}

// FindWorktree searches for a worktree by branch name or path.
// Priority: first match by branch ref refs/heads/<name>, then exact path match
// when branch info is missing. Returns nil if not found.
// Returns an error if multiple worktrees match the same branch ref.
func (s *Service) FindWorktree(repoRoot string, name string) (*WorktreeInfo, error) {
	trees, err := s.ListWorktrees(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("find worktree %q: %w", name, err)
	}

	wantRef := "refs/heads/" + name
	var branchMatch *WorktreeInfo
	var pathMatches []WorktreeInfo

	for i := range trees {
		wt := &trees[i]
		if wt.BranchRef == wantRef {
			if branchMatch != nil {
				return nil, fmt.Errorf("ambiguous: multiple worktrees on branch %q", name)
			}
			branchMatch = wt
		}
		if wt.Path == name {
			pathMatches = append(pathMatches, *wt)
		}
	}

	if branchMatch != nil {
		return branchMatch, nil
	}
	if len(pathMatches) == 1 {
		return &pathMatches[0], nil
	}
	if len(pathMatches) > 1 {
		return nil, fmt.Errorf("ambiguous: multiple worktrees at path %q", name)
	}
	return nil, nil
}

// CreateWorktree runs git worktree add <absPath> -b <branchName> from repoRoot.
func (s *Service) CreateWorktree(repoRoot string, absPath string, branchName string) error {
	_, err := s.runner.RunWithOpts(run.RunOpts{Dir: repoRoot}, "git", "worktree", "add", absPath, "-b", branchName)
	if err != nil {
		return fmt.Errorf("create worktree %q with branch %q: %w", absPath, branchName, err)
	}
	return nil
}

// RemoveWorktree runs git worktree remove <absPath> from repoRoot.
// It does NOT pass --force.
func (s *Service) RemoveWorktree(repoRoot string, absPath string) error {
	_, err := s.runner.RunWithOpts(run.RunOpts{Dir: repoRoot}, "git", "worktree", "remove", absPath)
	if err != nil {
		return fmt.Errorf("remove worktree %q: %w", absPath, err)
	}
	return nil
}

// DeleteBranch runs git branch -D <branchName> from repoRoot.
func (s *Service) DeleteBranch(repoRoot string, branchName string) error {
	_, err := s.runner.RunWithOpts(run.RunOpts{Dir: repoRoot}, "git", "branch", "-D", branchName)
	if err != nil {
		return fmt.Errorf("delete branch %q: %w", branchName, err)
	}
	return nil
}

// parsePorcelain parses the output of git worktree list --porcelain.
// Each worktree block starts with "worktree <path>" and ends with an empty line.
// Known fields: worktree, HEAD, branch, bare, detached.
func parsePorcelain(output string) []WorktreeInfo {
	if output == "" {
		return nil
	}

	var result []WorktreeInfo
	var current *WorktreeInfo

	flush := func() {
		if current != nil {
			result = append(result, *current)
			current = nil
		}
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			flush()
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			flush()
			current = &WorktreeInfo{Path: strings.TrimPrefix(line, "worktree ")}
			continue
		}

		if current == nil {
			continue
		}

		if strings.HasPrefix(line, "HEAD ") {
			current.Head = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") {
			current.BranchRef = strings.TrimPrefix(line, "branch ")
		} else if line == "bare" {
			current.Bare = true
		} else if line == "detached" {
			current.Detached = true
		}
	}
	flush()

	return result
}
