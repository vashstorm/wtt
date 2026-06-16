package tmux

import (
	"errors"
	"os"
	"testing"

	"wtt/internal/run"
)

// TestHasSession_Exists verifies that HasSession returns true when
// the tmux has-session command succeeds (exit 0).
func TestHasSession_Exists(t *testing.T) {
	fr := run.NewFakeRunner(map[string]string{
		"tmux": "tmux 3.4",
	}, nil)

	sess := NewSession(fr)
	got, err := sess.HasSession("mysession")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Error("expected HasSession to return true for existing session")
	}

	// Verify the recorded call.
	fake := fr.(*run.FakeRunner)
	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(fake.Calls))
	}
	c := fake.Calls[0]
	if c.Cmd != "tmux" {
		t.Errorf("cmd: got %q, want %q", c.Cmd, "tmux")
	}
	wantArgs := []string{"has-session", "-t", "mysession"}
	if len(c.Args) != len(wantArgs) {
		t.Fatalf("args length: got %d, want %d", len(c.Args), len(wantArgs))
	}
	for i, a := range c.Args {
		if a != wantArgs[i] {
			t.Errorf("arg[%d]: got %q, want %q", i, a, wantArgs[i])
		}
	}
}

// TestHasSession_NotExists verifies that HasSession returns false, nil
// when the tmux has-session command fails (session doesn't exist).
func TestHasSession_NotExists(t *testing.T) {
	fr := run.NewFakeRunner(nil, map[string]error{
		"tmux": errors.New("exit status 1"),
	})

	sess := NewSession(fr)
	got, err := sess.HasSession("mysession")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Error("expected HasSession to return false for non-existent session")
	}
}

// TestNewSession verifies that the NewSession method runs the correct
// tmux command with -d, -s, and -c flags.
func TestNewSession(t *testing.T) {
	fr := run.NewFakeRunner(map[string]string{
		"tmux": "",
	}, nil)

	sess := NewSession(fr)
	err := sess.NewSession("feature-x", "/path/to/worktree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fake := fr.(*run.FakeRunner)
	if len(fake.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(fake.Calls))
	}
	c := fake.Calls[0]
	if c.Cmd != "tmux" {
		t.Errorf("cmd: got %q, want %q", c.Cmd, "tmux")
	}
	wantArgs := []string{"new-session", "-d", "-s", "feature-x", "-c", "/path/to/worktree"}
	if len(c.Args) != len(wantArgs) {
		t.Fatalf("args length: got %d, want %d", len(c.Args), len(wantArgs))
	}
	for i, a := range c.Args {
		if a != wantArgs[i] {
			t.Errorf("arg[%d]: got %q, want %q", i, a, wantArgs[i])
		}
	}
}

// TestAttachSession_OutsideTmux verifies that when TMUX env var is not set,
// AttachSession uses `tmux attach -t <name>`.
func TestAttachSession_OutsideTmux(t *testing.T) {
	fr := run.NewFakeRunner(nil, nil)

	// Ensure TMUX is not set.
	origTMUX := os.Getenv("TMUX")
	os.Unsetenv("TMUX")
	defer os.Setenv("TMUX", origTMUX)

	sess := NewSession(fr)
	err := sess.AttachSession("myfeature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fake := fr.(*run.FakeRunner)
	if len(fake.InteractiveCalls) != 1 {
		t.Fatalf("expected 1 interactive call, got %d", len(fake.InteractiveCalls))
	}
	c := fake.InteractiveCalls[0]
	if c.Cmd != "tmux" {
		t.Errorf("cmd: got %q, want %q", c.Cmd, "tmux")
	}
	wantArgs := []string{"attach", "-t", "myfeature"}
	if len(c.Args) != len(wantArgs) {
		t.Fatalf("args length: got %d, want %d", len(c.Args), len(wantArgs))
	}
	for i, a := range c.Args {
		if a != wantArgs[i] {
			t.Errorf("arg[%d]: got %q, want %q", i, a, wantArgs[i])
		}
	}
}

// TestAttachSession_InsideTmux verifies that when TMUX env var is set,
// AttachSession uses `tmux switch-client -t <name>`.
func TestAttachSession_InsideTmux(t *testing.T) {
	fr := run.NewFakeRunner(nil, nil)

	// Set TMUX env var to simulate being inside tmux.
	origTMUX := os.Getenv("TMUX")
	os.Setenv("TMUX", "/tmp/tmux-1000/default,1234,0")
	defer os.Setenv("TMUX", origTMUX)

	sess := NewSession(fr)
	err := sess.AttachSession("myfeature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fake := fr.(*run.FakeRunner)
	if len(fake.InteractiveCalls) != 1 {
		t.Fatalf("expected 1 interactive call, got %d", len(fake.InteractiveCalls))
	}
	c := fake.InteractiveCalls[0]
	if c.Cmd != "tmux" {
		t.Errorf("cmd: got %q, want %q", c.Cmd, "tmux")
	}
	wantArgs := []string{"switch-client", "-t", "myfeature"}
	if len(c.Args) != len(wantArgs) {
		t.Fatalf("args length: got %d, want %d", len(c.Args), len(wantArgs))
	}
	for i, a := range c.Args {
		if a != wantArgs[i] {
			t.Errorf("arg[%d]: got %q, want %q", i, a, wantArgs[i])
		}
	}
}

// TestEnsureSession_NewSession verifies the full flow when the session
// does not exist: IsAvailable → HasSession (not found) → NewSession → AttachSession.
func TestEnsureSession_NewSession(t *testing.T) {
	// FakeRunner: "tmux" command errors for has-session (session not found),
	// but succeeds for -V and new-session.
	// We need a runner that behaves differently for different tmux subcommands.
	sfr := newSeqFakeRunner()
	sfr.results = []seqResult{
		{out: "tmux 3.4", err: nil},                 // tmux -V (IsAvailable)
		{out: "", err: errors.New("exit status 1")}, // tmux has-session (HasSession → not found)
		{out: "", err: nil},                         // tmux new-session (NewSession)
		// AttachSession uses RunInteractive, recorded separately
	}
	sfr.interactiveErr = nil // attach succeeds

	sess := &Session{runner: sfr}
	err := sess.EnsureSession("feature-x", "/path/to/worktree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the sequence of Run calls.
	if sfr.runIdx != 3 {
		t.Errorf("expected 3 Run calls, got %d", sfr.runIdx)
	}

	// Verify the interactive call for attach.
	if len(sfr.interactiveCalls) != 1 {
		t.Fatalf("expected 1 interactive call, got %d", len(sfr.interactiveCalls))
	}
	ic := sfr.interactiveCalls[0]
	if ic.Cmd != "tmux" {
		t.Errorf("interactive cmd: got %q, want %q", ic.Cmd, "tmux")
	}
}

// TestEnsureSession_ExistingSession verifies that when the session already
// exists, EnsureSession skips NewSession and only attaches.
func TestEnsureSession_ExistingSession(t *testing.T) {
	sfr := newSeqFakeRunner()
	sfr.results = []seqResult{
		{out: "tmux 3.4", err: nil}, // tmux -V (IsAvailable)
		{out: "", err: nil},         // tmux has-session (HasSession → exists)
		// No new-session call expected
		// AttachSession uses RunInteractive
	}
	sfr.interactiveErr = nil

	sess := &Session{runner: sfr}
	err := sess.EnsureSession("feature-x", "/path/to/worktree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only 2 Run calls: IsAvailable + HasSession. No NewSession.
	if sfr.runIdx != 2 {
		t.Errorf("expected 2 Run calls (no new-session), got %d", sfr.runIdx)
	}

	// Verify the interactive call for attach.
	if len(sfr.interactiveCalls) != 1 {
		t.Fatalf("expected 1 interactive call, got %d", len(sfr.interactiveCalls))
	}
}

// TestTmuxNotAvailable verifies that IsAvailable returns an error
// when tmux is not installed.
func TestTmuxNotAvailable(t *testing.T) {
	fr := run.NewFakeRunner(nil, map[string]error{
		"tmux": errors.New("exec: not found"),
	})

	sess := NewSession(fr)
	err := sess.IsAvailable()
	if err == nil {
		t.Fatal("expected error when tmux is not available, got nil")
	}
	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}
}

// --- Sequence-aware fake runner for multi-call tests ---

type seqResult struct {
	out string
	err error
}

type seqFakeRunner struct {
	results          []seqResult
	runIdx           int
	interactiveErr   error
	interactiveCalls []run.RecordedCall
}

func newSeqFakeRunner() *seqFakeRunner {
	return &seqFakeRunner{}
}

func (s *seqFakeRunner) Run(cmd string, args ...string) (string, error) {
	if s.runIdx >= len(s.results) {
		return "", nil
	}
	r := s.results[s.runIdx]
	s.runIdx++
	return r.out, r.err
}

func (s *seqFakeRunner) RunWithOpts(opts run.RunOpts, cmd string, args ...string) (string, error) {
	return s.Run(cmd, args...)
}

func (s *seqFakeRunner) RunInteractive(cmd string, args ...string) error {
	s.interactiveCalls = append(s.interactiveCalls, run.RecordedCall{
		Cmd:  cmd,
		Args: args,
	})
	return s.interactiveErr
}
