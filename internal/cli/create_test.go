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

func TestIntegrationCreateDefaultWorktree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	repoDir := t.TempDir()
	initTestRepo(t, repoDir)

	parsed := &ParsedArgs{
		Op:   OpCreate,
		Name: "feature-a",
	}

	var stdout, stderr bytes.Buffer
	exitCode := handleCreate(parsed, &stdout, &stderr, repoDir)

	if exitCode != 0 {
		t.Fatalf("exit code: got %d, want 0; stderr: %s", exitCode, stderr.String())
	}

	// Verify worktree exists on disk at <repo>/.claude/worktree/feature-a.
	expectedPath := filepath.Join(repoDir, ".claude", "worktree", "feature-a")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("worktree path %q does not exist", expectedPath)
	}

	// Verify branch exists.
	runner := run.NewRealRunner()
	out, err := runner.RunWithOpts(run.RunOpts{Dir: repoDir}, "git", "branch", "--list", "feature-a")
	if err != nil {
		t.Fatalf("git branch --list: %v", err)
	}
	if !strings.Contains(out, "feature-a") {
		t.Errorf("branch feature-a not found; output: %q", out)
	}

	// Verify stdout is a single cd command line.
	outStr := strings.TrimRight(stdout.String(), "\n")
	if !strings.HasPrefix(outStr, "cd '") {
		t.Errorf("stdout should start with \"cd '\", got: %q", outStr)
	}
	lines := strings.Count(stdout.String(), "\n")
	if strings.HasSuffix(stdout.String(), "\n") {
		lines--
	}
	if lines != 0 {
		t.Errorf("stdout should have exactly 1 line, got %d lines: %q", lines+1, stdout.String())
	}
}

func TestIntegrationCreateCustomBase(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	repoDir := t.TempDir()
	initTestRepo(t, repoDir)

	customBase := t.TempDir()

	parsed := &ParsedArgs{
		Op:         OpCreate,
		Name:       "feature-b",
		CustomBase: customBase,
	}

	var stdout, stderr bytes.Buffer
	exitCode := handleCreate(parsed, &stdout, &stderr, repoDir)

	if exitCode != 0 {
		t.Fatalf("exit code: got %d, want 0; stderr: %s", exitCode, stderr.String())
	}

	// Verify worktree exists at <customBase>/feature-b.
	expectedPath := filepath.Join(customBase, "feature-b")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("worktree path %q does not exist", expectedPath)
	}
}

func TestIntegrationCreateSyncsFilesAndGlob(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	repoDir := t.TempDir()
	initTestRepo(t, repoDir)

	writeTestFile(t, filepath.Join(repoDir, "config.json"), `{"env":"local"}`)
	writeTestFile(t, filepath.Join(repoDir, "data", "one.txt"), "one")
	writeTestFile(t, filepath.Join(repoDir, "data", "nested", "two.txt"), "two")

	parsed := &ParsedArgs{
		Op:        OpCreate,
		Name:      "feature-sync",
		SyncSpecs: []string{"config.json", "data/*"},
	}

	var stdout, stderr bytes.Buffer
	exitCode := handleCreate(parsed, &stdout, &stderr, repoDir)

	if exitCode != 0 {
		t.Fatalf("exit code: got %d, want 0; stderr: %s", exitCode, stderr.String())
	}

	expectedPath := filepath.Join(repoDir, ".claude", "worktree", "feature-sync")
	assertFileContent(t, filepath.Join(expectedPath, "config.json"), `{"env":"local"}`)
	assertFileContent(t, filepath.Join(expectedPath, "data", "one.txt"), "one")
	assertFileContent(t, filepath.Join(expectedPath, "data", "nested", "two.txt"), "two")
}

func TestIntegrationCreateSyncMissingSourceFailsBeforeCreate(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	repoDir := t.TempDir()
	initTestRepo(t, repoDir)

	parsed := &ParsedArgs{
		Op:        OpCreate,
		Name:      "feature-missing-sync",
		SyncSpecs: []string{"missing.json"},
	}

	var stdout, stderr bytes.Buffer
	exitCode := handleCreate(parsed, &stdout, &stderr, repoDir)

	if exitCode == 0 {
		t.Fatal("expected non-zero exit code, got 0")
	}
	if !strings.Contains(stderr.String(), "sync source not found") {
		t.Errorf("stderr should contain sync source error, got: %q", stderr.String())
	}

	expectedPath := filepath.Join(repoDir, ".claude", "worktree", "feature-missing-sync")
	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Errorf("worktree path should not exist after sync validation failure: %q", expectedPath)
	}
}

func TestIntegrationCreateBaseDirAutoCreated(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	repoDir := t.TempDir()
	initTestRepo(t, repoDir)

	nonExistentBase := filepath.Join(t.TempDir(), "nested", "dir")

	parsed := &ParsedArgs{
		Op:         OpCreate,
		Name:       "feature-c",
		CustomBase: nonExistentBase,
	}

	var stdout, stderr bytes.Buffer
	exitCode := handleCreate(parsed, &stdout, &stderr, repoDir)

	if exitCode != 0 {
		t.Fatalf("exit code: got %d, want 0; stderr: %s", exitCode, stderr.String())
	}

	// Verify the base directory was created.
	if _, err := os.Stat(nonExistentBase); os.IsNotExist(err) {
		t.Errorf("base directory %q was not created", nonExistentBase)
	}

	// Verify worktree exists.
	expectedPath := filepath.Join(nonExistentBase, "feature-c")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("worktree path %q does not exist", expectedPath)
	}
}

func TestIntegrationExistingWorktreeEmitCd(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	repoDir := t.TempDir()
	initTestRepo(t, repoDir)

	// Create a worktree named "existing" first.
	runner := run.NewRealRunner()
	wtPath := filepath.Join(t.TempDir(), "existing-wt")
	if _, err := runner.RunWithOpts(run.RunOpts{Dir: repoDir}, "git", "worktree", "add", wtPath, "-b", "existing"); err != nil {
		t.Fatalf("git worktree add: %v", err)
	}

	parsed := &ParsedArgs{
		Op:       OpCreate,
		Name:     "existing",
		Existing: true,
	}

	var stdout, stderr bytes.Buffer
	exitCode := handleCreate(parsed, &stdout, &stderr, repoDir)

	if exitCode != 0 {
		t.Fatalf("exit code: got %d, want 0; stderr: %s", exitCode, stderr.String())
	}

	outStr := stdout.String()
	if !strings.HasPrefix(outStr, "cd '") {
		t.Errorf("stdout should start with \"cd '\", got: %q", outStr)
	}

	// Verify exactly one line in stdout.
	trimmed := strings.TrimRight(outStr, "\n")
	if strings.Count(trimmed, "\n") != 0 {
		t.Errorf("stdout should have exactly 1 line, got: %q", outStr)
	}
}

func TestIntegrationExistingWorktreeMissing(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	repoDir := t.TempDir()
	initTestRepo(t, repoDir)

	parsed := &ParsedArgs{
		Op:       OpCreate,
		Name:     "nonexistent",
		Existing: true,
	}

	var stdout, stderr bytes.Buffer
	exitCode := handleCreate(parsed, &stdout, &stderr, repoDir)

	if exitCode == 0 {
		t.Fatal("expected non-zero exit code, got 0")
	}

	if strings.Contains(stdout.String(), "cd ") {
		t.Errorf("stdout should not contain cd command, got: %q", stdout.String())
	}

	if !strings.Contains(stderr.String(), "not found") {
		t.Errorf("stderr should contain 'not found', got: %q", stderr.String())
	}
}

func TestIntegrationCreateDuplicateWithoutE(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}

	repoDir := t.TempDir()
	initTestRepo(t, repoDir)

	// Create a worktree named "dup" first.
	runner := run.NewRealRunner()
	wtPath := filepath.Join(t.TempDir(), "dup-wt")
	if _, err := runner.RunWithOpts(run.RunOpts{Dir: repoDir}, "git", "worktree", "add", wtPath, "-b", "dup"); err != nil {
		t.Fatalf("git worktree add: %v", err)
	}

	// Try creating the same worktree again without -e.
	parsed := &ParsedArgs{
		Op:   OpCreate,
		Name: "dup",
	}

	var stdout, stderr bytes.Buffer
	exitCode := handleCreate(parsed, &stdout, &stderr, repoDir)

	if exitCode == 0 {
		t.Fatal("expected non-zero exit code for duplicate worktree, got 0")
	}

	if !strings.Contains(stderr.String(), "already exists") {
		t.Errorf("stderr should contain 'already exists', got: %q", stderr.String())
	}
}

func writeTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", path, err)
	}
	if string(got) != want {
		t.Errorf("content of %q = %q, want %q", path, string(got), want)
	}
}
