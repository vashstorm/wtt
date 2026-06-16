package cli

import (
	"testing"

	"wtt/internal/run"
)

// initTestRepo creates a git repo with an initial commit in dir.
// It uses a real runner, so tests that call it require git in PATH.
func initTestRepo(t *testing.T, dir string) {
	t.Helper()
	runner := run.NewRealRunner()
	if _, err := runner.RunWithOpts(run.RunOpts{Dir: dir}, "git", "init"); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if _, err := runner.RunWithOpts(run.RunOpts{Dir: dir}, "git", "commit", "--allow-empty", "-m", "init"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
}
