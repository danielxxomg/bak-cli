package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	configtest "github.com/danielxxomg/bak-cli/internal/config/testutil"
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
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"undo", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("undo --help should not error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "undo") {
		t.Fatal("help output should mention 'undo'")
	}
	if !strings.Contains(output, "revert") && !strings.Contains(output, "Revert") {
		t.Fatal("help output should mention revert")
	}

	cmd := findSubcommand(t, "undo")
	if cmd != nil {
		if err := cmd.Flags().Set("help", "false"); err != nil {
			t.Fatal(err)
		}
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
	if !strings.Contains(cmd.Long, "target configuration") {
		t.Errorf("expected Long description to mention target configuration, got: %s", cmd.Long)
	}
	if !strings.Contains(cmd.Long, "drift") {
		t.Errorf("expected Long description to mention drift, got: %s", cmd.Long)
	}
}

// --- runUndo execution tests ---

func TestRunUndoWithDeps_Delegation(t *testing.T) {
	configtest.SetConfigHome(t, t.TempDir())
	// Verify runUndoWithDeps creates UndoAction and delegates without panic.
	deps, _, _ := setupTestDeps(t)

	cmd := findSubcommand(t, "undo")
	if cmd == nil {
		t.Fatal("undo command not found")
	}
	err := runUndoWithDeps(cmd, nil, deps)

	// In an isolated home without .bak repo, undo must return a repository error.
	if err == nil {
		t.Fatal("expected error from undo delegation in empty isolated home, got nil")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "repository") &&
		!strings.Contains(errStr, "repo") &&
		!strings.Contains(errStr, "bak") &&
		!strings.Contains(errStr, "git") &&
		!strings.Contains(errStr, "undo") {
		t.Errorf("unexpected error from undo delegation: %v", err)
	}
}

func TestRunUndo_NoBakRepo(t *testing.T) {
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})
	configtest.SetConfigHome(t, t.TempDir())
	cmd := findSubcommand(t, "undo")
	if cmd != nil {
		cmd.InitDefaultHelpFlag()
		if err := cmd.Flags().Set("help", "false"); err != nil {
			t.Fatal(err)
		}
	}
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"undo"})
	err := rootCmd.Execute()

	if err == nil {
		t.Fatalf("expected error in empty isolated home without bak repo, got nil (stdout: %q, stderr: %q)", bufOut.String(), bufErr.String())
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "repository") && !strings.Contains(errStr, "repo") && !strings.Contains(errStr, "bak") {
		t.Errorf("unexpected undo error: %v", err)
	}
}

func TestRunUndo_ExtraArgs(t *testing.T) {
	t.Cleanup(func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	})
	configtest.SetConfigHome(t, t.TempDir())
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"undo", "extra"})
	err := rootCmd.Execute()

	if err == nil {
		// Test arg validator directly.
		var cmd *cobra.Command
		for _, sub := range rootCmd.Commands() {
			if sub.Name() == "undo" {
				cmd = sub
				break
			}
		}
		if cmd != nil {
			argErr := cmd.Args(cmd, []string{"extra"})
			if argErr == nil {
				t.Error("undo Args validator should reject extra args")
			}
		}
		return
	}
	// Error is expected (cobra.NoArgs).
}
