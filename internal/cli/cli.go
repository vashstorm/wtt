package cli

import (
	"fmt"
	"io"
	"os"
)

// Main is the entry point for the wtt CLI.
func Main(args []string, stdout, stderr io.Writer) int {
	parsed, err := Parse(args)
	if err != nil {
		PrintUsage(stderr)
		return 2
	}

	callerCwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	switch parsed.Op {
	case OpCreate:
		return handleCreate(parsed, stdout, stderr, callerCwd)
	case OpDeleteWorktree, OpDeleteWorktreeAndBranch:
		return handleDelete(parsed, stderr, callerCwd)
	case OpHelp:
		if err := PrintUsage(stdout); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "error: unknown operation %v\n", parsed.Op)
		return 1
	}
}
