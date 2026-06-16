# wtt — Worktree Tool

A small Go CLI for creating, entering, and deleting git worktrees, with optional tmux integration.

## How It Works

`wtt` emits a `cd '<path>'` line to stdout so that an eval'd shell wrapper can change the caller's working directory. This lets you jump into worktrees from your current shell without manually `cd`-ing.

## Installation

```sh
make install
```

Or build manually:

```sh
make build
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
| `-h`, `--help` | Show help                                                       |

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

- **Viper-backed argument parser**: CLI flags are defined with `spf13/pflag` and read through `spf13/viper`, while still allowing flags before or after the worktree name (`wtt name -w` and `wtt -w name` are equivalent).
- **No config files or persistent state**: All behavior is driven by CLI flags and the current git repository.
- **stdout contract**: Successful create/enter operations print a single `cd '<path>'` line to stdout. Errors and delete output go to stderr.
- **Sync paths**: `-s` copies files, directories, or glob matches from the caller's current directory into a newly created worktree while preserving relative paths.
- **No forced deletion**: `RemoveWorktree` calls `git worktree remove` without `--force`. Dirty worktrees will fail.

## Development

### Prerequisites

- Go 1.26.4+
- `git` in `PATH`
- `golangci-lint` (for `make lint`)

### Commands

```sh
make build       # Build the binary
make test        # Run all tests
make test-cover  # Generate and view coverage
make vet         # Run go vet
make fmt         # Check formatting
make lint        # Run golangci-lint
make install     # Install to $GOPATH/bin
make clean       # Remove build artifacts
```

Run a single test:

```sh
go test ./internal/cli -run TestIntegrationCreateDefaultWorktree
```

## Guardrails

- Must run inside a git repository
- No fish shell support
- No auto-modification of shell config files
- No forced deletion of dirty worktrees
- No config files or persistent state
