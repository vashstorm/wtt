package cli

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantOp     OpType
		wantName   string
		wantBase   string
		wantSync   []string
		wantTmux   bool
		wantExist  bool
		wantHelp   bool
		wantErr    bool
		errContain string
	}{
		// Flag order independence: wtt name -w ≡ wtt -w name
		{
			name:     "flag after name",
			args:     []string{"name", "-w"},
			wantOp:   OpCreate,
			wantName: "name",
			wantTmux: true,
		},
		{
			name:     "flag before name",
			args:     []string{"-w", "name"},
			wantOp:   OpCreate,
			wantName: "name",
			wantTmux: true,
		},

		// Mutual exclusion: -d + -D
		{
			name:       "-d and -D together fails",
			args:       []string{"-d", "name", "-D"},
			wantErr:    true,
			errContain: "mutually exclusive",
		},
		{
			name:       "-D and -d together fails",
			args:       []string{"-D", "name", "-d"},
			wantErr:    true,
			errContain: "mutually exclusive",
		},

		// Delete mode with incompatible flags
		{
			name:       "-d with -w fails",
			args:       []string{"-d", "name", "-w"},
			wantErr:    true,
			errContain: "-w cannot be used with delete",
		},
		{
			name:       "-d with -c path fails",
			args:       []string{"-d", "-c", "path", "name"},
			wantErr:    true,
			errContain: "-c cannot be used with delete",
		},
		{
			name:       "-d with -e fails",
			args:       []string{"-d", "name", "-e"},
			wantErr:    true,
			errContain: "-e cannot be used with delete",
		},
		{
			name:       "-d with -s fails",
			args:       []string{"-d", "-s", "config.json", "name"},
			wantErr:    true,
			errContain: "-s cannot be used with delete",
		},

		// No arguments
		{
			name:       "no args fails",
			args:       []string{},
			wantErr:    true,
			errContain: "worktree name is required",
		},

		// Missing name
		{
			name:       "-e without name fails",
			args:       []string{"-e"},
			wantErr:    true,
			errContain: "requires a worktree name",
		},
		{
			name:       "-d without name fails",
			args:       []string{"-d"},
			wantErr:    true,
			errContain: "requires a worktree name",
		},

		// help
		{
			args:     []string{"init"},
			wantOp:   OpInit,
			wantName: "",
		},
		{
			name:     "wtt help returns OpHelp",
			args:     []string{"help"},
			wantOp:   OpHelp,
			wantHelp: true,
		},
		{
			name:     "wtt -h returns OpHelp",
			args:     []string{"-h"},
			wantOp:   OpHelp,
			wantHelp: true,
		},
		{
			name:     "wtt --help returns OpHelp",
			args:     []string{"--help"},
			wantOp:   OpHelp,
			wantHelp: true,
		},

		// -c flag captures CustomBase
		{
			name:     "-c path name captures CustomBase",
			args:     []string{"-c", "path", "name"},
			wantOp:   OpCreate,
			wantName: "name",
			wantBase: "path",
		},
		{
			name:     "-s path name captures SyncSpecs",
			args:     []string{"-s", "config.json", "name"},
			wantOp:   OpCreate,
			wantName: "name",
			wantSync: []string{"config.json"},
		},
		{
			name:     "-s comma list captures SyncSpecs",
			args:     []string{"name", "-s", "config.json,data/*"},
			wantOp:   OpCreate,
			wantName: "name",
			wantSync: []string{"config.json", "data/*"},
		},
		{
			name:     "-s attached value captures SyncSpecs",
			args:     []string{"-sconfig.json,data/*", "name"},
			wantOp:   OpCreate,
			wantName: "name",
			wantSync: []string{"config.json", "data/*"},
		},
		{
			name:     "repeated -s appends SyncSpecs",
			args:     []string{"-s", "config.json", "-s", "data/*", "name"},
			wantOp:   OpCreate,
			wantName: "name",
			wantSync: []string{"config.json", "data/*"},
		},
		{
			name:       "-s without path value fails",
			args:       []string{"-s"},
			wantErr:    true,
			errContain: "requires a path argument",
		},
		{
			name:       "-s with empty comma item fails",
			args:       []string{"-s", "config.json,,data/*", "name"},
			wantErr:    true,
			errContain: "requires a path argument",
		},

		// -e and -w together
		{
			name:      "-e -w name sets both Existing and WithTmux",
			args:      []string{"name", "-e", "-w"},
			wantOp:    OpCreate,
			wantName:  "name",
			wantExist: true,
			wantTmux:  true,
		},

		// Unknown flag
		{
			name:       "unknown flag -x fails",
			args:       []string{"-x", "name"},
			wantErr:    true,
			errContain: "unknown flag",
		},

		// Too many positional arguments
		{
			name:       "too many positionals fails",
			args:       []string{"name1", "name2"},
			wantErr:    true,
			errContain: "too many arguments",
		},

		// Delete operations
		{
			name:     "-d name returns OpDeleteWorktree",
			args:     []string{"-d", "name"},
			wantOp:   OpDeleteWorktree,
			wantName: "name",
		},
		{
			name:     "-D name returns OpDeleteWorktreeAndBranch",
			args:     []string{"-D", "name"},
			wantOp:   OpDeleteWorktreeAndBranch,
			wantName: "name",
		},

		// Basic create
		{
			name:     "simple name returns OpCreate",
			args:     []string{"name"},
			wantOp:   OpCreate,
			wantName: "name",
		},

		// -c with -e is OK
		{
			name:      "-c path -e name is valid",
			args:      []string{"-c", "path", "-e", "name"},
			wantOp:    OpCreate,
			wantName:  "name",
			wantBase:  "path",
			wantExist: true,
		},
		{
			name:       "-s with -e fails",
			args:       []string{"-s", "config.json", "-e", "name"},
			wantErr:    true,
			errContain: "-s cannot be used with -e",
		},

		// -c path after name
		{
			name:     "name -c path captures CustomBase",
			args:     []string{"name", "-c", "path"},
			wantOp:   OpCreate,
			wantName: "name",
			wantBase: "path",
		},

		// -c without path value
		{
			name:       "-c without path fails",
			args:       []string{"-c"},
			wantErr:    true,
			errContain: "requires a path argument",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse(%v) expected error, got nil (parsed: %+v)", tt.args, got)
				}
				if tt.errContain != "" && !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("Parse(%v) error = %q, want to contain %q", tt.args, err.Error(), tt.errContain)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%v) unexpected error: %v", tt.args, err)
			}
			if got.Op != tt.wantOp {
				t.Errorf("Op = %v, want %v", got.Op, tt.wantOp)
			}
			if got.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantName)
			}
			if got.CustomBase != tt.wantBase {
				t.Errorf("CustomBase = %q, want %q", got.CustomBase, tt.wantBase)
			}
			if !stringSlicesEqual(got.SyncSpecs, tt.wantSync) {
				t.Errorf("SyncSpecs = %v, want %v", got.SyncSpecs, tt.wantSync)
			}
			if got.WithTmux != tt.wantTmux {
				t.Errorf("WithTmux = %v, want %v", got.WithTmux, tt.wantTmux)
			}
			if got.Existing != tt.wantExist {
				t.Errorf("Existing = %v, want %v", got.Existing, tt.wantExist)
			}
			if got.Help != tt.wantHelp {
				t.Errorf("Help = %v, want %v", got.Help, tt.wantHelp)
			}
		})
	}
}

func stringSlicesEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestParseFlagOrderEquivalence(t *testing.T) {
	// Explicitly verify wtt name -w ≡ wtt -w name produce identical ParsedArgs
	a1, err1 := Parse([]string{"name", "-w"})
	a2, err2 := Parse([]string{"-w", "name"})

	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected errors: %v, %v", err1, err2)
	}

	if a1.Op != a2.Op || a1.Name != a2.Name || a1.WithTmux != a2.WithTmux {
		t.Errorf("flag order mismatch: after=%+v, before=%+v", a1, a2)
	}
}
