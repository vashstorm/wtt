package git

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"wtt/internal/run"
)

// --- helpers ---

// fakeRunner returns a *run.FakeRunner from a Runner interface.
func fakeRunner(r run.Runner) *run.FakeRunner {
	return r.(*run.FakeRunner)
}

// lastCall returns the most recent RecordedCall, or fails the test.
func lastCall(t *testing.T, fr *run.FakeRunner) run.RecordedCall {
	t.Helper()
	if len(fr.Calls) == 0 {
		t.Fatal("expected at least one recorded call, got none")
	}
	return fr.Calls[len(fr.Calls)-1]
}

// --- Unit tests with FakeRunner ---

func TestCreateWorktree(t *testing.T) {
	fr := run.NewFakeRunner(nil, nil)
	svc := NewService(fr)

	err := svc.CreateWorktree("/repo", "/repo-wt", "feature-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	call := lastCall(t, fakeRunner(fr))
	if call.Cmd != "git" {
		t.Errorf("cmd: got %q, want %q", call.Cmd, "git")
	}
	wantArgs := []string{"worktree", "add", "/repo-wt", "-b", "feature-x"}
	if len(call.Args) != len(wantArgs) {
		t.Fatalf("args length: got %d, want %d", len(call.Args), len(wantArgs))
	}
	for i, got := range call.Args {
		if got != wantArgs[i] {
			t.Errorf("args[%d]: got %q, want %q", i, got, wantArgs[i])
		}
	}
	if call.Dir != "/repo" {
		t.Errorf("dir: got %q, want %q", call.Dir, "/repo")
	}
}

func TestRemoveWorktree(t *testing.T) {
	fr := run.NewFakeRunner(nil, nil)
	svc := NewService(fr)

	err := svc.RemoveWorktree("/repo", "/repo-wt", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	call := lastCall(t, fakeRunner(fr))
	if call.Cmd != "git" {
		t.Errorf("cmd: got %q, want %q", call.Cmd, "git")
	}
	wantArgs := []string{"worktree", "remove", "/repo-wt"}
	if len(call.Args) != len(wantArgs) {
		t.Fatalf("args length: got %d, want %d", len(call.Args), len(wantArgs))
	}
	for i, got := range call.Args {
		if got != wantArgs[i] {
			t.Errorf("args[%d]: got %q, want %q", i, got, wantArgs[i])
		}
	}
	if call.Dir != "/repo" {
		t.Errorf("dir: got %q, want %q", call.Dir, "/repo")
	}
	// Verify --force is NOT passed.
	for _, arg := range call.Args {
		if arg == "--force" {
			t.Error("--force flag should not be passed to git worktree remove")
		}
	}
}

func TestRemoveWorktreeForce(t *testing.T) {
	fr := run.NewFakeRunner(nil, nil)
	svc := NewService(fr)

	err := svc.RemoveWorktree("/repo", "/repo-wt", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	call := lastCall(t, fakeRunner(fr))
	if call.Cmd != "git" {
		t.Errorf("cmd: got %q, want %q", call.Cmd, "git")
	}
	wantArgs := []string{"worktree", "remove", "--force", "/repo-wt"}
	if len(call.Args) != len(wantArgs) {
		t.Fatalf("args length: got %d, want %d", len(call.Args), len(wantArgs))
	}
	for i, got := range call.Args {
		if got != wantArgs[i] {
			t.Errorf("args[%d]: got %q, want %q", i, got, wantArgs[i])
		}
	}
	if call.Dir != "/repo" {
		t.Errorf("dir: got %q, want %q", call.Dir, "/repo")
	}
}

func TestDeleteBranch(t *testing.T) {
	fr := run.NewFakeRunner(nil, nil)
	svc := NewService(fr)

	err := svc.DeleteBranch("/repo", "feature-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	call := lastCall(t, fakeRunner(fr))
	if call.Cmd != "git" {
		t.Errorf("cmd: got %q, want %q", call.Cmd, "git")
	}
	wantArgs := []string{"branch", "-D", "feature-x"}
	if len(call.Args) != len(wantArgs) {
		t.Fatalf("args length: got %d, want %d", len(call.Args), len(wantArgs))
	}
	for i, got := range call.Args {
		if got != wantArgs[i] {
			t.Errorf("args[%d]: got %q, want %q", i, got, wantArgs[i])
		}
	}
	if call.Dir != "/repo" {
		t.Errorf("dir: got %q, want %q", call.Dir, "/repo")
	}
}

func TestListWorktrees(t *testing.T) {
	porcelain := `worktree /home/user/project
HEAD abc123def456
branch refs/heads/main

worktree /home/user/project-feature
HEAD def456abc123
branch refs/heads/feature-x

worktree /home/user/project-detached
HEAD 111222333444
detached

worktree /home/user/project-bare
HEAD 555666777888
bare

`
	fr := run.NewFakeRunner(map[string]string{"git": porcelain}, nil)
	svc := NewService(fr)

	trees, err := svc.ListWorktrees("/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(trees) != 4 {
		t.Fatalf("expected 4 worktrees, got %d", len(trees))
	}

	// Main worktree.
	if trees[0].Path != "/home/user/project" {
		t.Errorf("trees[0].Path: got %q, want %q", trees[0].Path, "/home/user/project")
	}
	if trees[0].Head != "abc123def456" {
		t.Errorf("trees[0].Head: got %q, want %q", trees[0].Head, "abc123def456")
	}
	if trees[0].BranchRef != "refs/heads/main" {
		t.Errorf("trees[0].BranchRef: got %q, want %q", trees[0].BranchRef, "refs/heads/main")
	}
	if trees[0].Detached {
		t.Error("trees[0] should not be detached")
	}
	if trees[0].Bare {
		t.Error("trees[0] should not be bare")
	}

	// Feature worktree.
	if trees[1].BranchRef != "refs/heads/feature-x" {
		t.Errorf("trees[1].BranchRef: got %q, want %q", trees[1].BranchRef, "refs/heads/feature-x")
	}

	// Detached worktree.
	if !trees[2].Detached {
		t.Error("trees[2] should be detached")
	}
	if trees[2].BranchRef != "" {
		t.Errorf("trees[2].BranchRef: got %q, want empty", trees[2].BranchRef)
	}

	// Bare worktree.
	if !trees[3].Bare {
		t.Error("trees[3] should be bare")
	}
}

func TestFindWorktree_ByBranchRef(t *testing.T) {
	porcelain := `worktree /repo
HEAD aaa111
branch refs/heads/main

worktree /repo-feature
HEAD bbb222
branch refs/heads/feature-x

`
	fr := run.NewFakeRunner(map[string]string{"git": porcelain}, nil)
	svc := NewService(fr)

	wt, err := svc.FindWorktree("/repo", "feature-x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wt == nil {
		t.Fatal("expected worktree, got nil")
	}
	if wt.Path != "/repo-feature" {
		t.Errorf("path: got %q, want %q", wt.Path, "/repo-feature")
	}
	if wt.BranchRef != "refs/heads/feature-x" {
		t.Errorf("branch: got %q, want %q", wt.BranchRef, "refs/heads/feature-x")
	}
}

func TestFindWorktree_ByPathMatch(t *testing.T) {
	// Worktree with no branch info (detached HEAD), match by path.
	porcelain := `worktree /repo
HEAD aaa111
branch refs/heads/main

worktree /repo-detached
HEAD bbb222
detached

`
	fr := run.NewFakeRunner(map[string]string{"git": porcelain}, nil)
	svc := NewService(fr)

	wt, err := svc.FindWorktree("/repo", "/repo-detached")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wt == nil {
		t.Fatal("expected worktree via path match, got nil")
	}
	if wt.Path != "/repo-detached" {
		t.Errorf("path: got %q, want %q", wt.Path, "/repo-detached")
	}
}

func TestFindWorktree_NotFound(t *testing.T) {
	porcelain := `worktree /repo
HEAD aaa111
branch refs/heads/main

`
	fr := run.NewFakeRunner(map[string]string{"git": porcelain}, nil)
	svc := NewService(fr)

	wt, err := svc.FindWorktree("/repo", "nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wt != nil {
		t.Errorf("expected nil, got %+v", wt)
	}
}

func TestFindWorktree_Ambiguous(t *testing.T) {
	// Two worktrees with same branch ref — should error.
	porcelain := `worktree /repo-a
HEAD aaa111
branch refs/heads/feature

worktree /repo-b
HEAD bbb222
branch refs/heads/feature

`
	fr := run.NewFakeRunner(map[string]string{"git": porcelain}, nil)
	svc := NewService(fr)

	_, err := svc.FindWorktree("/repo", "feature")
	if err == nil {
		t.Fatal("expected ambiguous error, got nil")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("error should mention ambiguous, got: %v", err)
	}
}

func TestRepoRoot_NotInRepo(t *testing.T) {
	fr := run.NewFakeRunner(nil, map[string]error{
		"git": errors.New("fatal: not a git repository: /nope"),
	})
	svc := NewService(fr)

	_, err := svc.RepoRoot("/nope")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotRepo) {
		t.Errorf("expected ErrNotRepo, got: %v", err)
	}
}

func TestRepoRoot_Success(t *testing.T) {
	fr := run.NewFakeRunner(map[string]string{"git": "/home/user/project"}, nil)
	svc := NewService(fr)

	root, err := svc.RepoRoot("/home/user/project/sub")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != "/home/user/project" {
		t.Errorf("got %q, want %q", root, "/home/user/project")
	}

	call := lastCall(t, fakeRunner(fr))
	if call.Dir != "/home/user/project/sub" {
		t.Errorf("dir: got %q, want %q", call.Dir, "/home/user/project/sub")
	}
}

func TestValidateBranchName(t *testing.T) {
	tests := []struct {
		name    string
		branch  string
		wantErr bool
	}{
		{"valid branch", "feature-x", false},
		{"valid with slash", "feature/x", false},
		{"invalid double dot", "feature..x", true},
		{"invalid starts with dash", "-bad", true},
		{"invalid with space", "bad name", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fr run.Runner
			if tt.wantErr {
				fr = run.NewFakeRunner(nil, map[string]error{
					"git": errors.New("invalid branch name"),
				})
			} else {
				fr = run.NewFakeRunner(nil, nil)
			}
			svc := NewService(fr)

			err := svc.ValidateBranchName(tt.branch)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateBranchName_ValidNoError(t *testing.T) {
	// No error scripted → FakeRunner returns empty string, nil error.
	fr := run.NewFakeRunner(nil, nil)
	svc := NewService(fr)

	err := svc.ValidateBranchName("feature-x")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Integration tests with real git ---

func TestIntegrationCreateWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	runner := run.NewRealRunner()
	svc := NewService(runner)

	repoDir := t.TempDir()

	// Init repo and make initial commit (worktree add needs at least one commit).
	if _, err := runner.RunWithOpts(run.RunOpts{Dir: repoDir}, "git", "init"); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if _, err := runner.RunWithOpts(run.RunOpts{Dir: repoDir}, "git", "commit", "--allow-empty", "-m", "init"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	// Create worktree.
	wtPath := filepath.Join(t.TempDir(), "feature-wt")
	if err := svc.CreateWorktree(repoDir, wtPath, "feature-integration"); err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	// Verify worktree directory exists on disk.
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Errorf("worktree path %q does not exist", wtPath)
	}

	// Verify via git worktree list.
	trees, err := svc.ListWorktrees(repoDir)
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}

	found := false
	for _, wt := range trees {
		if wt.BranchRef == "refs/heads/feature-integration" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("feature-integration worktree not found in list: %+v", trees)
	}
}

func TestIntegrationListWorktrees(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	runner := run.NewRealRunner()
	svc := NewService(runner)

	repoDir := t.TempDir()

	if _, err := runner.RunWithOpts(run.RunOpts{Dir: repoDir}, "git", "init"); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if _, err := runner.RunWithOpts(run.RunOpts{Dir: repoDir}, "git", "commit", "--allow-empty", "-m", "init"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	trees, err := svc.ListWorktrees(repoDir)
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}

	if len(trees) < 1 {
		t.Fatal("expected at least 1 worktree (main)")
	}

	// The main branch worktree should exist.
	hasMain := false
	for _, wt := range trees {
		if wt.BranchRef == "refs/heads/main" || wt.BranchRef == "refs/heads/master" {
			hasMain = true
			if wt.Path == "" {
				t.Error("main worktree should have a path")
			}
			if wt.Head == "" {
				t.Error("main worktree should have a HEAD")
			}
		}
	}
	if !hasMain {
		t.Errorf("expected main/master branch in worktrees, got: %+v", trees)
	}
}

func TestIntegrationRemoveWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	runner := run.NewRealRunner()
	svc := NewService(runner)

	repoDir := t.TempDir()

	if _, err := runner.RunWithOpts(run.RunOpts{Dir: repoDir}, "git", "init"); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if _, err := runner.RunWithOpts(run.RunOpts{Dir: repoDir}, "git", "commit", "--allow-empty", "-m", "init"); err != nil {
		t.Fatalf("git commit: %v", err)
	}

	// Create worktree.
	wtPath := filepath.Join(t.TempDir(), "to-remove-wt")
	if err := svc.CreateWorktree(repoDir, wtPath, "to-remove"); err != nil {
		t.Fatalf("CreateWorktree: %v", err)
	}

	// Remove worktree.
	if err := svc.RemoveWorktree(repoDir, wtPath, false); err != nil {
		t.Fatalf("RemoveWorktree: %v", err)
	}

	// Verify it's gone from list.
	trees, err := svc.ListWorktrees(repoDir)
	if err != nil {
		t.Fatalf("ListWorktrees: %v", err)
	}
	for _, wt := range trees {
		if wt.Path == wtPath {
			t.Errorf("worktree %q still in list after removal", wtPath)
		}
	}

	// Verify directory is gone.
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("worktree directory %q still exists after removal", wtPath)
	}
}
