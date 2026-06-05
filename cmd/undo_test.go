package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestUndoCmd_Structure(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "undo" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("undo subcommand not registered on root")
	}
}

func TestUndoCmd_Flags(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "undo" {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("undo command not found")
	}

	// undo should accept no arguments.
	err := cmd.Args(cmd, []string{})
	if err != nil {
		t.Fatalf("undo should accept 0 args, got error: %v", err)
	}
}

func TestUndoCmd_Args(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "undo" {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("undo command not found")
	}

	// undo takes no args — passing one should error.
	err := cmd.Args(cmd, []string{"extra"})
	if err == nil {
		t.Fatal("expected error with extra arg, got nil")
	}
}

func TestUndoCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"undo", "--help"})
	rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "undo") {
		t.Fatal("help output should mention 'undo'")
	}
	if !strings.Contains(output, "revert") || !strings.Contains(output, "Revert") {
		t.Fatal("help output should mention revert")
	}
}

func TestUndoCmd_Use(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "undo" {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("undo command not found")
	}

	if cmd.Use == "" {
		t.Fatal("undo command should have a Use string")
	}
	if !strings.HasPrefix(cmd.Use, "undo") {
		t.Fatalf("Use = %q, should start with \"undo\"", cmd.Use)
	}
	if cmd.Short == "" {
		t.Fatal("undo command should have a short description")
	}
	if cmd.Long == "" {
		t.Fatal("undo command should have a long description")
	}
}
