package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestCmdFlagRegistration verifies that all expected flags are registered
// on their respective commands. This is a compile-time-plus check —
// faster than running full command execution but confirms wiring integrity.
func TestCmdFlagRegistration(t *testing.T) {
	tests := []struct {
		name     string
		cmd      *cobra.Command
		flags    []string
		defaults map[string]string
	}{
		{
			name:  "push flags",
			cmd:   pushCmd,
			flags: []string{"provider", "profile"},
			defaults: map[string]string{
				"provider": "github-gist",
			},
		},
		{
			name:  "pull flags",
			cmd:   pullCmd,
			flags: []string{"provider", "profile"},
			defaults: map[string]string{
				"provider": "github-gist",
			},
		},
		{
			name:  "backup flags",
			cmd:   backupCmd,
			flags: []string{"preset", "profile", "adapter"},
		},
		{
			name:  "restore flags",
			cmd:   restoreCmd,
			flags: []string{"dry-run", "force"},
			defaults: map[string]string{
				"dry-run": "false",
				"force":   "false",
			},
		},
		{
			name:  "list flags",
			cmd:   listCmd,
			flags: []string{"provider"},
		},
		{
			name:  "login flags",
			cmd:   loginCmd,
			flags: []string{"provider"},
		},
		{
			name:  "export flags",
			cmd:   exportCmd,
			flags: []string{"output"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, flagName := range tt.flags {
				flag := tt.cmd.Flags().Lookup(flagName)
				if flag == nil {
					t.Errorf("--%s flag not defined on %s", flagName, tt.cmd.Name())
					continue
				}
				if expected, ok := tt.defaults[flagName]; ok {
					if flag.DefValue != expected {
						t.Errorf("--%s default = %q, want %q", flagName, flag.DefValue, expected)
					}
				}
			}
		})
	}
}

// TestCmdStructure verifies that all expected subcommands are registered on root.
func TestCmdStructure(t *testing.T) {
	expected := []string{
		"backup", "restore", "undo", "list", "pick",
		"push", "pull", "export", "login", "profile",
		"verify", "diff", "version", "schedule",
	}

	registered := make(map[string]bool)
	for _, sub := range rootCmd.Commands() {
		registered[sub.Name()] = true
	}

	for _, name := range expected {
		if !registered[name] {
			t.Errorf("subcommand %q not registered on root", name)
		}
	}
}
