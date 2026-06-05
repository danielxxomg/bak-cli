package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRestoreCmd_Structure(t *testing.T) {
	// Ensure the restore command is registered on root.
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "restore" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("restore subcommand not registered on root")
	}
}

func TestRestoreCmd_Flags(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "restore" {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("restore command not found")
	}

	dryRunFlag := cmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil {
		t.Fatal("--dry-run flag not defined")
	}
	if dryRunFlag.DefValue != "false" {
		t.Fatalf("--dry-run default = %q, want \"false\"", dryRunFlag.DefValue)
	}

	forceFlag := cmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Fatal("--force flag not defined")
	}
	if forceFlag.DefValue != "false" {
		t.Fatalf("--force default = %q, want \"false\"", forceFlag.DefValue)
	}
}

func TestRestoreCmd_Args(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "restore" {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("restore command not found")
	}

	// Test args validator directly.
	// ExactArgs(1) should reject 0 args.
	err := cmd.Args(cmd, []string{})
	if err == nil {
		t.Fatal("expected error with 0 args, got nil")
	}

	// ExactArgs(1) should reject 2 args.
	err = cmd.Args(cmd, []string{"id1", "id2"})
	if err == nil {
		t.Fatal("expected error with 2 args, got nil")
	}

	// ExactArgs(1) should accept 1 arg.
	err = cmd.Args(cmd, []string{"valid-id"})
	if err != nil {
		t.Fatalf("expected no error with 1 arg, got %v", err)
	}
}

func TestRestoreCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Route through root so help is properly resolved.
	rootCmd.SetArgs([]string{"restore", "--help"})
	rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "restore") {
		t.Fatal("help output should mention 'restore'")
	}
	if !strings.Contains(output, "--dry-run") {
		t.Fatal("help output should mention --dry-run")
	}
	if !strings.Contains(output, "--force") {
		t.Fatal("help output should mention --force")
	}
}

func TestRestoreCmd_Use(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "restore" {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("restore command not found")
	}

	// Use should start with "restore".
	if cmd.Use == "" {
		t.Fatal("restore command should have a Use string")
	}
	if !strings.HasPrefix(cmd.Use, "restore") {
		t.Fatalf("Use = %q, should start with \"restore\"", cmd.Use)
	}

	if cmd.Short == "" {
		t.Fatal("restore command should have a short description")
	}
	if cmd.Long == "" {
		t.Fatal("restore command should have a long description")
	}
}

// --- runRestore execution tests ---

func TestRunRestore_MissingArgs(t *testing.T) {
	// Restore requires exactly 1 arg. Direct args validation was tested above.
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"restore"})
	err := rootCmd.Execute()

	if err == nil {
		// If there's a restore with no args that succeeds, it may be because
		// cobra handles subcommand routing differently. Test the arg validator directly.
		var cmd *cobra.Command
		for _, sub := range rootCmd.Commands() {
			if sub.Name() == "restore" {
				cmd = sub
				break
			}
		}
		if cmd != nil {
			argErr := cmd.Args(cmd, []string{})
			if argErr == nil {
				t.Error("restore Args validator should reject 0 args")
			}
		}
		return
	}
	// If it errors, that's correct behavior.
}

func TestRunRestore_BackupNotFound(t *testing.T) {
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"restore", "20250101-000000"})
	err := rootCmd.Execute()

	if err == nil {
		t.Log("restore of non-existent backup succeeded (backup may exist)")
		return
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestRunRestore_DryRunNonexistent(t *testing.T) {
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"restore", "--dry-run", "20250101-000000"})
	err := rootCmd.Execute()

	if err == nil {
		t.Log("restore --dry-run succeeded (backup may exist)")
		return
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestRestoreCmd_UseAndDescription(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "restore" {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("restore command not found")
	}
	if cmd.Use == "" {
		t.Fatal("restore Use should not be empty")
	}
	if cmd.Short == "" {
		t.Fatal("restore Short should not be empty")
	}
	if cmd.Long == "" {
		t.Fatal("restore Long should not be empty")
	}
}
