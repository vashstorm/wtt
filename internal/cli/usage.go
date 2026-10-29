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
    -c <path>  Use <path> as the base directory (default: <repo>/.worktree/<name>)
    -s <paths> Sync files or directories into the new worktree (comma-separated)
    -w         Open the worktree in a new tmux session
    -e         Attach to an existing worktree (do not create a new one)
    -b         Create the worktree on a new branch named <name>
               (default: detached HEAD at the current commit)
    -d <name>  Delete a worktree (keeps the branch)
    -D <name>  Delete a worktree and its branch
    -f, --force
               Force delete a worktree with -d or -D
    -h, --help Show this help

EXAMPLES
    wtt feature                 Create worktree "feature"
    wtt -w feature              Create worktree in a tmux session
    wtt -e feature              Attach to existing worktree
    wtt -c /repo feature        Create worktree using /repo as base
    wtt -b feature              Create worktree "feature" on a new branch
    wtt -s config.json feature  Sync config.json into the new worktree
    wtt -s config.json,data/* feature
                                Sync multiple paths into the new worktree
    wtt -d feature              Delete worktree, keep branch
    wtt -D feature              Delete worktree and branch
    wtt -d -f feature           Force delete worktree, keep branch

    Flags can be combined and placed before or after the worktree name:
    wtt -b -w feature           Create a new branch and open the worktree in tmux
    wtt -b feature -c ../worktrees -s 'config.json,data/*'
                                Create a new branch under a custom base and sync multiple paths
    wtt -e feature -w           Open an existing worktree in tmux
    wtt -D feature --force      Force delete the worktree and its branch

    Short flags can be grouped:
    wtt -df feature            Same as -d -f: force delete worktree, keep branch
    wtt -Df feature            Same as -D -f: force delete worktree and branch
    wtt -bw feature            Same as -b -w: create a new branch and open in tmux
    wtt -ew feature            Same as -e -w: open an existing worktree in tmux`)
	return err
}
