package run

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// RunOpts holds optional configuration for a command execution.
type RunOpts struct {
	Dir   string    // Working directory for the command.
	Env   []string  // Environment variables (overrides parent env if set).
	Stdin io.Reader // Optional stdin for the command.
}

// RecordedCall captures a single invocation recorded by FakeRunner.
type RecordedCall struct {
	Cmd  string
	Args []string
	Dir  string
	Env  []string
}

// Runner defines the interface for executing shell commands.
type Runner interface {
	// Run executes a command with default options and returns combined
	// stdout+stderr output trimmed of trailing whitespace.
	Run(cmd string, args ...string) (string, error)

	// RunWithOpts executes a command with the given options and returns
	// combined stdout+stderr output trimmed of trailing whitespace.
	RunWithOpts(opts RunOpts, cmd string, args ...string) (string, error)

	// RunInteractive executes a command with inherited stdin/stdout/stderr,
	// giving the process direct terminal control. Returns error on failure.
	RunInteractive(cmd string, args ...string) error
}

// RealRunner executes commands via os/exec.
type RealRunner struct{}

// NewRealRunner returns a Runner that executes real commands via os/exec.
func NewRealRunner() Runner {
	return &RealRunner{}
}

// Run executes a command with default options.
func (r *RealRunner) Run(cmd string, args ...string) (string, error) {
	return r.RunWithOpts(RunOpts{}, cmd, args...)
}

// RunWithOpts executes a command with the given options.
// A fresh exec.Cmd is constructed per call; it is never reused.
func (r *RealRunner) RunWithOpts(opts RunOpts, cmd string, args ...string) (string, error) {
	c := exec.Command(cmd, args...)

	if opts.Dir != "" {
		c.Dir = opts.Dir
	}
	if opts.Env != nil {
		c.Env = opts.Env
	}
	if opts.Stdin != nil {
		c.Stdin = opts.Stdin
	}

	// Capture stderr separately so we can include it in error messages.
	var stderr bytes.Buffer
	c.Stderr = &stderr

	out, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("run %s: %w: %s", cmd, err, strings.TrimRight(stderr.String(), "\n"))
	}

	return strings.TrimRight(string(out), "\n"), nil
}

// RunInteractive executes a command with inherited stdin/stdout/stderr,
// giving the process direct terminal control (e.g. for tmux attach).
func (r *RealRunner) RunInteractive(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// FakeRunner records calls and returns scripted outputs/errors for testing.
type FakeRunner struct {
	Calls            []RecordedCall
	InteractiveCalls []RecordedCall
	outputs          map[string]string
	interactiveErrs  map[string]error
	errors           map[string]error
}

// NewFakeRunner returns a Runner that records calls and returns scripted
// results. The outputs map is keyed by command name; the errors map is also
// keyed by command name.
func NewFakeRunner(outputs map[string]string, errors map[string]error) Runner {
	return &FakeRunner{
		outputs:         outputs,
		errors:          errors,
		interactiveErrs: make(map[string]error),
	}
}

// Run records the call and returns the scripted output or error.
func (f *FakeRunner) Run(cmd string, args ...string) (string, error) {
	return f.RunWithOpts(RunOpts{}, cmd, args...)
}

// RunWithOpts records the call with full options and returns the scripted
// output or error.
func (f *FakeRunner) RunWithOpts(opts RunOpts, cmd string, args ...string) (string, error) {
	f.Calls = append(f.Calls, RecordedCall{
		Cmd:  cmd,
		Args: args,
		Dir:  opts.Dir,
		Env:  opts.Env,
	})

	if err, ok := f.errors[cmd]; ok {
		return "", err
	}
	if out, ok := f.outputs[cmd]; ok {
		return out, nil
	}
	return "", nil
}

// SetInteractiveError scripts an error to be returned by RunInteractive for
// the given command name.
func (f *FakeRunner) SetInteractiveError(cmd string, err error) {
	f.interactiveErrs[cmd] = err
}

// RunInteractive records the call and returns the scripted error.
func (f *FakeRunner) RunInteractive(cmd string, args ...string) error {
	f.InteractiveCalls = append(f.InteractiveCalls, RecordedCall{
		Cmd:  cmd,
		Args: args,
	})

	if err, ok := f.interactiveErrs[cmd]; ok {
		return err
	}
	return nil
}
