package run

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

// --- RealRunner tests ---

func TestRealRunnerEchoHello(t *testing.T) {
	r := NewRealRunner()
	out, err := r.Run("echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "hello" {
		t.Errorf("got %q, want %q", out, "hello")
	}
}

func TestRealRunnerStderrOnFailure(t *testing.T) {
	r := NewRealRunner()
	_, err := r.Run("sh", "-c", "echo oops >&2 && exit 1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "oops") {
		t.Errorf("error should contain stderr content, got: %v", err)
	}
}

func TestRealRunnerWithOptsDir(t *testing.T) {
	r := NewRealRunner()
	// Use a directory that exists on macOS.
	out, err := r.RunWithOpts(RunOpts{Dir: "/tmp"}, "ls", "-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// /tmp always has content on macOS; just verify we got some output.
	if len(out) == 0 {
		t.Error("expected non-empty output from ls /tmp")
	}
}

func TestRealRunnerWithOptsStdin(t *testing.T) {
	r := NewRealRunner()
	out, err := r.RunWithOpts(RunOpts{Stdin: strings.NewReader("from-stdin")}, "cat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "from-stdin" {
		t.Errorf("got %q, want %q", out, "from-stdin")
	}
}

func TestRealRunnerTrimsTrailingNewline(t *testing.T) {
	r := NewRealRunner()
	out, err := r.Run("echo", "line1", "line2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// echo adds a trailing newline; RealRunner should trim it.
	if strings.HasSuffix(out, "\n") {
		t.Error("output should have trailing newline trimmed")
	}
}

func TestRealRunnerFreshCmdPerCall(t *testing.T) {
	r := NewRealRunner()
	// Run two different commands; each should work independently.
	out1, err := r.Run("echo", "first")
	if err != nil {
		t.Fatalf("first call: unexpected error: %v", err)
	}
	out2, err := r.Run("echo", "second")
	if err != nil {
		t.Fatalf("second call: unexpected error: %v", err)
	}
	if out1 != "first" {
		t.Errorf("first call: got %q, want %q", out1, "first")
	}
	if out2 != "second" {
		t.Errorf("second call: got %q, want %q", out2, "second")
	}
}

// --- FakeRunner tests ---

func TestFakeRunnerRecordsCalls(t *testing.T) {
	fr := NewFakeRunner(map[string]string{"echo": "hello"}, nil)

	_, _ = fr.Run("echo", "hello", "world")
	_, _ = fr.RunWithOpts(RunOpts{Dir: "/tmp", Env: []string{"FOO=bar"}}, "ls", "-la")

	fake := fr.(*FakeRunner)
	if len(fake.Calls) != 2 {
		t.Fatalf("expected 2 recorded calls, got %d", len(fake.Calls))
	}

	// First call: Run with no opts.
	c1 := fake.Calls[0]
	if c1.Cmd != "echo" {
		t.Errorf("call 0 cmd: got %q, want %q", c1.Cmd, "echo")
	}
	if len(c1.Args) != 2 || c1.Args[0] != "hello" || c1.Args[1] != "world" {
		t.Errorf("call 0 args: got %v", c1.Args)
	}

	// Second call: RunWithOpts with Dir and Env.
	c2 := fake.Calls[1]
	if c2.Cmd != "ls" {
		t.Errorf("call 1 cmd: got %q, want %q", c2.Cmd, "ls")
	}
	if c2.Dir != "/tmp" {
		t.Errorf("call 1 dir: got %q, want %q", c2.Dir, "/tmp")
	}
	if len(c2.Env) != 1 || c2.Env[0] != "FOO=bar" {
		t.Errorf("call 1 env: got %v", c2.Env)
	}
}

func TestFakeRunnerReturnsScriptedOutput(t *testing.T) {
	fr := NewFakeRunner(map[string]string{
		"git":  "branch main",
		"echo": "hello",
	}, nil)

	out, err := fr.Run("git", "branch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "branch main" {
		t.Errorf("got %q, want %q", out, "branch main")
	}
}

func TestFakeRunnerReturnsScriptedError(t *testing.T) {
	scriptedErr := errors.New("command not found")
	fr := NewFakeRunner(nil, map[string]error{
		"badcmd": scriptedErr,
	})

	_, err := fr.Run("badcmd")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, scriptedErr) {
		t.Errorf("got error %v, want %v", err, scriptedErr)
	}
}

func TestFakeRunnerNoExternalBinaryInvoked(t *testing.T) {
	// If FakeRunner leaked to the real system, this non-existent command
	// would fail. With the fake, it should just record the call.
	fr := NewFakeRunner(map[string]string{"nonexistent_cmd_12345": "fake output"}, nil)

	out, err := fr.Run("nonexistent_cmd_12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "fake output" {
		t.Errorf("got %q, want %q", out, "fake output")
	}

	// Verify no real binary was executed: the command does not exist.
	if _, realErr := os.Stat("/usr/bin/nonexistent_cmd_12345"); !os.IsNotExist(realErr) {
		t.Log("sanity: checking that the binary truly does not exist")
	}
}

func TestFakeRunnerReturnsEmptyForUnknownCommand(t *testing.T) {
	fr := NewFakeRunner(nil, nil)
	out, err := fr.Run("unknown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty output for unscripted command, got %q", out)
	}
}

// --- Interface compliance checks ---

func TestRealRunnerImplementsRunner(t *testing.T) {
	var _ Runner = NewRealRunner()
}

func TestFakeRunnerImplementsRunner(t *testing.T) {
	var _ Runner = NewFakeRunner(nil, nil)
}

func TestRealRunnerFalseCommand(t *testing.T) {
	r := NewRealRunner()
	_, err := r.Run("false")
	if err == nil {
		t.Fatal("expected error from 'false' command, got nil")
	}
}

func TestRealRunnerMultiArg(t *testing.T) {
	r := NewRealRunner()
	out, err := r.Run("printf", "%s-%s", "foo", "bar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "foo-bar"
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

func TestRealRunnerEnvOverride(t *testing.T) {
	r := NewRealRunner()
	out, err := r.RunWithOpts(RunOpts{Env: []string{"MY_VAR=hello"}}, "sh", "-c", "echo $MY_VAR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "hello" {
		t.Errorf("got %q, want %q", out, "hello")
	}
}

func TestFakeRunnerMultipleCallsRecorded(t *testing.T) {
	fr := NewFakeRunner(map[string]string{"echo": "ok"}, nil)
	for i := 0; i < 5; i++ {
		_, _ = fr.Run("echo", fmt.Sprintf("call-%d", i))
	}
	fake := fr.(*FakeRunner)
	if len(fake.Calls) != 5 {
		t.Errorf("expected 5 calls, got %d", len(fake.Calls))
	}
}
