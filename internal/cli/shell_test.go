package cli

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestFormatCDCommand_SimplePath(t *testing.T) {
	got := FormatCDCommand("/home/user/project")
	want := `cd '/home/user/project'`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatCDCommand_PathWithSpace(t *testing.T) {
	got := FormatCDCommand("/tmp/wtt feature")
	want := `cd '/tmp/wtt feature'`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatCDCommand_PathWithSingleQuote(t *testing.T) {
	got := FormatCDCommand("/tmp/a'b")
	want := `cd '/tmp/a'\''b'`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatCDCommand_PathWithMultipleSingleQuotes(t *testing.T) {
	got := FormatCDCommand("/tmp/it's a'test")
	want := `cd '/tmp/it'\''s a'\''test'`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatCDCommand_EvalSimplePath(t *testing.T) {
	cmd := FormatCDCommand("/tmp")
	verifyEval(t, cmd, "/tmp")
}

func TestFormatCDCommand_EvalPathWithSpace(t *testing.T) {
	// Create a temp dir with a space in the name for a real eval test.
	dir := t.TempDir()
	spaceDir := dir + "/wtt feature"
	if err := mkdirAll(spaceDir); err != nil {
		t.Skipf("cannot create dir with space: %v", err)
	}
	cmd := FormatCDCommand(spaceDir)
	verifyEval(t, cmd, spaceDir)
}

func TestFormatCDCommand_EvalPathWithSingleQuote(t *testing.T) {
	dir := t.TempDir()
	quoteDir := dir + "/a'b"
	if err := mkdirAll(quoteDir); err != nil {
		t.Skipf("cannot create dir with quote: %v", err)
	}
	cmd := FormatCDCommand(quoteDir)
	verifyEval(t, cmd, quoteDir)
}

// verifyEval runs the cd command via sh -c and checks that PWD matches the
// expected directory. This proves the output is syntactically valid POSIX.
func verifyEval(t *testing.T, cdCmd, expectedDir string) {
	t.Helper()
	// Use sh -c to eval the cd command and then print PWD.
	script := cdCmd + " && pwd"
	c := exec.Command("sh", "-c", script)
	out, err := c.Output()
	if err != nil {
		t.Fatalf("sh -c %q failed: %v: %s", script, err, out)
	}
	got := strings.TrimRight(string(out), "\n")
	if got != expectedDir {
		t.Errorf("eval: pwd got %q, want %q", got, expectedDir)
	}
}

// mkdirAll creates a directory and all parents.
func mkdirAll(path string) error {
	return os.MkdirAll(path, 0o755)
}
