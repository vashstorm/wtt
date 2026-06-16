package cli

import (
	"fmt"
	"io"
)

// PrintUsage writes the usage text to w.
func PrintUsage(w io.Writer) error {
	_, err := fmt.Fprintln(w, `wtt — worktree tool

USAGE
    wtt [FLAGS] <name>       Create or attach to a worktree
    wtt help | -h | --help   Show this help

FLAGS
    -c <path>  Use <path> as the base directory instead of the default
    -s <paths> Sync files or directories into the new worktree (comma-separated)
    -w         Open the worktree in a new tmux session
    -e         Attach to an existing worktree (do not create a new one)
    -d <name>  Delete a worktree (keeps the branch)
    -D <name>  Delete a worktree and its branch
    -f, --force
               Force delete a worktree with -d or -D
    -h, --help Show this help

EXAMPLES
    wtt feature-login          Create worktree "feature-login"
    wtt -w feature-login       Create worktree in a tmux session
    wtt -e feature-login       Attach to existing worktree
    wtt -c /repo feature-login Create worktree using /repo as base
    wtt -s config.json feature-login
                               Sync config.json into the new worktree
    wtt -s config.json,data/* feature-login
                               Sync multiple paths into the new worktree
    wtt -d feature-login       Delete worktree, keep branch
    wtt -D feature-login       Delete worktree and branch
    wtt -d -f feature-login    Force delete worktree, keep branch`)
	return err
}
