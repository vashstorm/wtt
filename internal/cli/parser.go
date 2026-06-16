package cli

import (
	"io"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// OpType represents the operation type determined from CLI arguments.
type OpType int

const (
	OpCreate                  OpType = iota // create a new worktree
	OpDeleteWorktree                        // delete worktree only
	OpDeleteWorktreeAndBranch               // delete worktree and its branch
	OpInit                                  // print shell wrapper function
	OpHelp                                  // show usage
)

// ParsedArgs holds the result of parsing CLI arguments.
type ParsedArgs struct {
	Op         OpType
	Name       string // worktree name (required for create/delete, empty for init/help)
	CustomBase string // custom base path from -c flag
	SyncSpecs  []string
	WithTmux   bool // -w flag: open in tmux session
	Existing   bool // -e flag: attach to existing worktree
	Help       bool // -h/--help/help subcommand
}

// Parse parses CLI arguments into a ParsedArgs struct.
// Viper reads values from a pflag FlagSet so flags can appear before OR after
// the worktree name:
//
//	wtt name -w  ≡  wtt -w name
func Parse(args []string) (*ParsedArgs, error) {
	pa, positionals, err := parseWithViper(args)
	if err != nil {
		return nil, err
	}

	// Determine operation from positionals
	if len(positionals) > 0 {
		switch positionals[0] {
		case "init":
			if len(positionals) > 1 {
				return nil, newParseError("init takes no arguments")
			}
			pa.Op = OpInit
			return pa, nil
		case "help":
			if len(positionals) > 1 {
				return nil, newParseError("too many arguments")
			}
			pa.Op = OpHelp
			pa.Help = true
			return pa, nil
		default:
			pa.Name = positionals[0]
		}
	}

	if len(positionals) > 1 {
		return nil, newParseError("too many arguments")
	}

	// Determine Op from flags
	if pa.Help {
		pa.Op = OpHelp
		return pa, nil
	}

	if pa.Op == OpDeleteWorktreeAndBranch || pa.Op == OpDeleteWorktree {
		// Validate: delete mode cannot use -c, -w, -e
		if pa.CustomBase != "" {
			return nil, newParseError("-c cannot be used with delete operations")
		}
		if pa.WithTmux {
			return nil, newParseError("-w cannot be used with delete operations")
		}
		if pa.Existing {
			return nil, newParseError("-e cannot be used with delete operations")
		}
		if len(pa.SyncSpecs) > 0 {
			return nil, newParseError("-s cannot be used with delete operations")
		}
		// Name is required for delete
		if pa.Name == "" {
			return nil, newParseError("delete requires a worktree name")
		}
		return pa, nil
	}

	// Default: OpCreate
	pa.Op = OpCreate

	// -e requires a name
	if pa.Existing && pa.Name == "" {
		return nil, newParseError("-e requires a worktree name")
	}
	if pa.Existing && len(pa.SyncSpecs) > 0 {
		return nil, newParseError("-s cannot be used with -e")
	}

	// Create requires a name (unless it's help/init which is already handled)
	if pa.Name == "" {
		return nil, newParseError("worktree name is required")
	}

	return pa, nil
}

func parseWithViper(args []string) (*ParsedArgs, []string, error) {
	flags := pflag.NewFlagSet("wtt", pflag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.SortFlags = false

	flags.StringP("custom-base", "c", "", "custom worktree base directory")
	flags.StringArrayP("sync", "s", nil, "paths to sync into the new worktree")
	flags.BoolP("tmux", "w", false, "open in tmux")
	flags.BoolP("existing", "e", false, "attach to existing worktree")
	flags.BoolP("delete", "d", false, "delete worktree")
	flags.BoolP("delete-branch", "D", false, "delete worktree and branch")
	flags.BoolP("help", "h", false, "show help")

	if err := flags.Parse(args); err != nil {
		return nil, nil, normalizeFlagError(err)
	}

	cfg := viper.New()
	if err := cfg.BindPFlags(flags); err != nil {
		return nil, nil, err
	}

	pa := &ParsedArgs{
		CustomBase: cfg.GetString("custom-base"),
		WithTmux:   cfg.GetBool("tmux"),
		Existing:   cfg.GetBool("existing"),
		Help:       cfg.GetBool("help"),
	}

	for _, value := range cfg.GetStringSlice("sync") {
		if err := addSyncSpecs(pa, value); err != nil {
			return nil, nil, err
		}
	}

	deleteWorktree := cfg.GetBool("delete")
	deleteBranch := cfg.GetBool("delete-branch")
	if deleteWorktree && deleteBranch {
		return nil, nil, newParseError("-d and -D are mutually exclusive")
	}
	if deleteBranch {
		pa.Op = OpDeleteWorktreeAndBranch
	} else if deleteWorktree {
		pa.Op = OpDeleteWorktree
	}

	return pa, flags.Args(), nil
}

func normalizeFlagError(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "flag needs an argument: 'c'"):
		return newParseError("-c requires a path argument")
	case strings.Contains(msg, "flag needs an argument: 's'"):
		return newParseError("-s requires a path argument")
	case strings.Contains(msg, "unknown shorthand flag:"):
		if _, flag, ok := strings.Cut(msg, " in "); ok {
			return newParseError("unknown flag: " + flag)
		}
		return newParseError("unknown flag")
	default:
		return newParseError(msg)
	}
}

func addSyncSpecs(pa *ParsedArgs, value string) error {
	for _, spec := range strings.Split(value, ",") {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			return newParseError("-s requires a path argument")
		}
		pa.SyncSpecs = append(pa.SyncSpecs, spec)
	}
	return nil
}

// parseError is a sentinel error type for parse failures.
type parseError struct {
	msg string
}

func newParseError(msg string) *parseError {
	return &parseError{msg: msg}
}

func (e *parseError) Error() string {
	return e.msg
}
