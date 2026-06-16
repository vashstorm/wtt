# wtt — Worktree Tool

A small Go CLI for creating, entering, and deleting git worktrees, with optional tmux integration.

## How It Works

`wtt` emits a `cd '<path>'` line to stdout after successful create/enter operations. A shell function can evaluate that line so `wtt feature-login` changes the caller shell's current directory.

## Installation

```sh
make install
```

This builds `wtt`, copies the binary to `~/.local/bin/wtt`, copies `.wtt` to `~/.wtt`, and appends `source ~/.wtt` to your shell rc file (`~/.zshrc` or `~/.bashrc`).

Do not use `sudo`; Makefile rejects `sudo make ...`. If needed, override shell detection with `make install USER_SHELL=/bin/zsh` or `make install USER_SHELL=/bin/bash`.

To uninstall:

```sh
make uninstall
```

This removes `~/.local/bin/wtt`, removes `~/.wtt`, and removes the exact `source ~/.wtt` line from your zsh or bash rc file when present.

Or build only:

```sh
make build
```

For a development binary with debug symbols:

```sh
make dev
```

## Usage

```
wtt [FLAGS] <name>       Create or attach to a worktree
wtt help | -h | --help   Show this help
```

### Flags

| Flag        | Description                                                        |
|-------------|--------------------------------------------------------------------|
| `-c <path>` | Use `<path>` as the base directory instead of the default          |
| `-s <paths>` | Sync files or directories into the new worktree. Use commas for multiple paths |
| `-w`        | Open the worktree in a new tmux session                            |
| `-e`        | Attach to an existing worktree (do not create a new one)           |
| `-d <name>` | Delete a worktree (keeps the branch)                               |
| `-D <name>` | Delete a worktree and its branch                                   |
| `-f`, `--force` | Force delete a worktree with `-d` or `-D`                      |
| `-h`, `--help` | Show help                                                       |

### Shell Integration

A standalone binary cannot change the current directory of its parent shell, so `make install` installs a shell function in `~/.wtt` and sources it from your shell rc file. The function calls the real `~/.local/bin/wtt` binary, then evaluates its `cd '<path>'` output so `wtt feature-login` leaves your current shell inside the worktree.

### Examples

```sh
wtt feature-login          # Create worktree "feature-login"
wtt -c ../worktrees feat   # Create worktree using ../worktrees as base
wtt -s config.json feat    # Sync config.json into the new worktree
wtt -s 'config.json,data/*' feat
                           # Sync multiple paths, preserving their structure
wtt -e feature-login       # Attach to existing worktree
wtt -w feature-login       # Create worktree in a tmux session
wtt -d feature-login       # Delete worktree, keep branch
wtt -D feature-login       # Delete worktree and branch
wtt -d -f feature-login    # Force delete worktree, keep branch
```

## Architecture

```
cmd/main.go
    └── internal/cli.Main
            ├── internal/cli.Parse
            └── handlers
                    ├── handleCreate  → internal/git.Service, internal/tmux.Session
                    ├── handleDelete  → internal/git.Service
                    └── PrintUsage
```

### Packages

| Package | Description |
|---------|-------------|
| `internal/cli` | Argument parsing, operation dispatch, and command handlers. The only package that wires the application together. |
| `internal/git` | Git operations: `RepoRoot`, `ListWorktrees`, `FindWorktree`, `CreateWorktree`, `RemoveWorktree`, `DeleteBranch`, `ValidateBranchName`. Parses `git worktree list --porcelain` output. |
| `internal/tmux` | Tmux session management: `IsAvailable`, `HasSession`, `NewSession`, `AttachSession`, `EnsureSession`. Uses `switch-client` when already inside tmux, `attach` otherwise. |
| `internal/run` | Process-running abstraction. `RealRunner` executes commands via `os/exec`; `FakeRunner` records calls and returns scripted outputs for tests. All git and tmux packages depend on this interface, not `os/exec` directly. |

### Design Decisions

- **pflag-backed argument parser**: CLI flags are defined and read with `spf13/pflag`, while still allowing flags before or after the worktree name (`wtt name -w` and `wtt -w name` are equivalent).
- **No config files or persistent state**: All behavior is driven by CLI flags and the current git repository.
- **stdout contract**: Successful create/enter operations print a single `cd '<path>'` line to stdout. Errors and delete output go to stderr.
- **Sync paths**: `-s` copies files, directories, or glob matches from the caller's current directory into a newly created worktree while preserving relative paths.
- **Forced deletion is explicit**: `-f` passes `--force` to `git worktree remove`. Without `-f`, dirty worktrees fail.

## Development

### Prerequisites

- Go 1.26.4+
- `git` in `PATH`
- `golangci-lint` (for `make lint`)

### Commands

```sh
make build         # Build optimized release binary
make dev           # Build with debug symbols
make test          # Run all tests
make test-cover    # Generate and view coverage
make vet           # Run go vet
make fmt           # Check formatting
make lint          # Run golangci-lint
make install       # Install to ~/.local/bin and configure ~/.wtt
make uninstall     # Remove ~/.local/bin/wtt and shell integration
make clean         # Remove build artifacts
```

Run a single test:

```sh
go test ./internal/cli -run TestIntegrationCreateDefaultWorktree
```

## Guardrails

- Must run inside a git repository
- No fish shell support
- `make install` updates the current zsh or bash rc file to source `~/.wtt`
- `make uninstall` removes the installed binary and zsh/bash shell integration
- Forced deletion of dirty worktrees requires `-f`
- No app config files or persistent worktree state; shell integration lives in `~/.wtt`
