package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"wtt/internal/run"
)

func TestIntegrationDeleteKeepsBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	repoDir := t.TempDir()
	initTestRepo(t, repoDir)

	// Create a worktree named "testdel".
	runner := run.NewRealRunner()
	wtPath := filepath.Join(t.TempDir(), "testdel-wt")
	if _, err := runner.RunWithOpts(run.RunOpts{Dir: repoDir}, "git", "worktree", "add", wtPath, "-b", "testdel"); err != nil {
		t.Fatalf("git worktree add: %v", err)
	}

	parsed := &ParsedArgs{
		Op:   OpDeleteWorktree,
		Name: "testdel",
	}

	var stderr bytes.Buffer
	exitCode := handleDelete(parsed, &stderr, repoDir)

	if exitCode != 0 {
		t.Fatalf("exit code: got %d, want 0; stderr: %s", exitCode, stderr.String())
	}

	// Verify worktree path is removed from git worktree list.
	out, err := runner.RunWithOpts(run.RunOpts{Dir: repoDir}, "git", "worktree", "list", "--porcelain")
	if err != nil {
		t.Fatalf("git worktree list: %v", err)
	}
	if strings.Contains(out, wtPath) {
		t.Errorf("worktree path %q still appears in worktree list", wtPath)
	}

	// Verify worktree directory is gone from disk.
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("worktree directory %q still exists on disk", wtPath)
	}

	// Verify branch still exists.
	branchOut, err := runner.RunWithOpts(run.RunOpts{Dir: repoDir}, "git", "branch", "--list", "testdel")
	if err != nil {
		t.Fatalf("git branch --list: %v", err)
	}
	if !strings.Contains(branchOut, "testdel") {
		t.Errorf("branch testdel should still exist after -d, but not found; output: %q", branchOut)
	}
}

func TestIntegrationDeleteWithBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	repoDir := t.TempDir()
	initTestRepo(t, repoDir)

	// Create a worktree named "testdel2".
	runner := run.NewRealRunner()
	wtPath := filepath.Join(t.TempDir(), "testdel2-wt")
	if _, err := runner.RunWithOpts(run.RunOpts{Dir: repoDir}, "git", "worktree", "add", wtPath, "-b", "testdel2"); err != nil {
		t.Fatalf("git worktree add: %v", err)
	}

	parsed := &ParsedArgs{
		Op:   OpDeleteWorktreeAndBranch,
		Name: "testdel2",
	}

	var stderr bytes.Buffer
	exitCode := handleDelete(parsed, &stderr, repoDir)

	if exitCode != 0 {
		t.Fatalf("exit code: got %d, want 0; stderr: %s", exitCode, stderr.String())
	}

	// Verify worktree path is removed from git worktree list.
	out, err := runner.RunWithOpts(run.RunOpts{Dir: repoDir}, "git", "worktree", "list", "--porcelain")
	if err != nil {
		t.Fatalf("git worktree list: %v", err)
	}
	if strings.Contains(out, wtPath) {
		t.Errorf("worktree path %q still appears in worktree list", wtPath)
	}

	// Verify worktree directory is gone from disk.
	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("worktree directory %q still exists on disk", wtPath)
	}

	// Verify branch is gone.
	branchOut, err := runner.RunWithOpts(run.RunOpts{Dir: repoDir}, "git", "branch", "--list", "testdel2")
	if err != nil {
		t.Fatalf("git branch --list: %v", err)
	}
	if strings.Contains(branchOut, "testdel2") {
		t.Errorf("branch testdel2 should be deleted after -D, but still found; output: %q", branchOut)
	}
}

func TestIntegrationDeleteMissingWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	repoDir := t.TempDir()
	initTestRepo(t, repoDir)

	parsed := &ParsedArgs{
		Op:   OpDeleteWorktree,
		Name: "nonexistent",
	}

	var stderr bytes.Buffer
	exitCode := handleDelete(parsed, &stderr, repoDir)

	if exitCode == 0 {
		t.Fatal("expected non-zero exit code for missing worktree, got 0")
	}

	if !strings.Contains(stderr.String(), "not found") {
		t.Errorf("stderr should contain 'not found', got: %q", stderr.String())
	}
}

func TestIntegrationDeleteDirtyWorktreeFails(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	repoDir := t.TempDir()
	initTestRepo(t, repoDir)

	// Create a worktree named "dirty".
	runner := run.NewRealRunner()
	wtPath := filepath.Join(t.TempDir(), "dirty-wt")
	if _, err := runner.RunWithOpts(run.RunOpts{Dir: repoDir}, "git", "worktree", "add", wtPath, "-b", "dirty"); err != nil {
		t.Fatalf("git worktree add: %v", err)
	}

	// Create an uncommitted file in the worktree.
	dirtyFile := filepath.Join(wtPath, "dirty.txt")
	if err := os.WriteFile(dirtyFile, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	parsed := &ParsedArgs{
		Op:   OpDeleteWorktree,
		Name: "dirty",
	}

	var stderr bytes.Buffer
	exitCode := handleDelete(parsed, &stderr, repoDir)

	if exitCode == 0 {
		t.Fatal("expected non-zero exit code for dirty worktree, got 0")
	}

	// Verify worktree path still exists on disk (git protects dirty worktrees).
	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		t.Errorf("dirty worktree directory %q should still exist after failed removal", wtPath)
	}
}

func TestIntegrationDeleteDirtyWorktreeForce(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	repoDir := t.TempDir()
	initTestRepo(t, repoDir)

	runner := run.NewRealRunner()
	wtPath := filepath.Join(t.TempDir(), "dirty-force-wt")
	if _, err := runner.RunWithOpts(run.RunOpts{Dir: repoDir}, "git", "worktree", "add", wtPath, "-b", "dirty-force"); err != nil {
		t.Fatalf("git worktree add: %v", err)
	}

	dirtyFile := filepath.Join(wtPath, "dirty.txt")
	if err := os.WriteFile(dirtyFile, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	parsed := &ParsedArgs{
		Op:    OpDeleteWorktree,
		Name:  "dirty-force",
		Force: true,
	}

	var stderr bytes.Buffer
	exitCode := handleDelete(parsed, &stderr, repoDir)

	if exitCode != 0 {
		t.Fatalf("exit code: got %d, want 0; stderr: %s", exitCode, stderr.String())
	}

	if _, err := os.Stat(wtPath); !os.IsNotExist(err) {
		t.Errorf("dirty worktree directory %q still exists after forced removal", wtPath)
	}
}
