package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// --- push command tests ---

func TestPushCmd_Structure(t *testing.T) {
	cmd := findSubcommand(t, "push")
	if cmd == nil {
		t.Fatal("push subcommand not registered on root")
	}
	if cmd.Use != "push [backup-id]" {
		t.Errorf("Use = %q, want 'push [backup-id]'", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("push should have a short description")
	}
	if cmd.Long == "" {
		t.Error("push should have a long description")
	}
}

func TestPushCmd_Args(t *testing.T) {
	cmd := findSubcommand(t, "push")
	if cmd == nil {
		t.Fatal("push command not found")
	}

	// No args should be fine (uses latest backup).
	err := cmd.Args(cmd, []string{})
	if err != nil {
		t.Fatalf("push with 0 args should be ok: %v", err)
	}

	// One arg should be fine.
	err = cmd.Args(cmd, []string{"20260604-150405"})
	if err != nil {
		t.Fatalf("push with 1 arg should be ok: %v", err)
	}

	// Two args should error.
	err = cmd.Args(cmd, []string{"a", "b"})
	if err == nil {
		t.Fatal("expected error for 2 args")
	}
}

func TestPushCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"push", "--help"})
	rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "push") {
		t.Fatal("help output should mention 'push'")
	}
	if !strings.Contains(output, "Gist") {
		t.Fatal("help output should mention 'Gist'")
	}
}

// --- pull command tests ---

func TestPullCmd_Structure(t *testing.T) {
	cmd := findSubcommand(t, "pull")
	if cmd == nil {
		t.Fatal("pull subcommand not registered on root")
	}
	if !strings.HasPrefix(cmd.Use, "pull") {
		t.Errorf("Use = %q, should start with 'pull'", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("pull should have a short description")
	}
}

func TestPullCmd_Args(t *testing.T) {
	cmd := findSubcommand(t, "pull")
	if cmd == nil {
		t.Fatal("pull command not found")
	}

	// No args should be fine (uses stored gist ID).
	err := cmd.Args(cmd, []string{})
	if err != nil {
		t.Fatalf("pull with 0 args should be ok: %v", err)
	}

	// One arg (gist ID) should be fine.
	err = cmd.Args(cmd, []string{"abc123"})
	if err != nil {
		t.Fatalf("pull with 1 arg should be ok: %v", err)
	}

	// Two args should error.
	err = cmd.Args(cmd, []string{"a", "b"})
	if err == nil {
		t.Fatal("expected error for 2 args")
	}
}

func TestPullCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"pull", "--help"})
	rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "pull") {
		t.Fatal("help output should mention 'pull'")
	}
}

// --- login command tests ---

func TestLoginCmd_Structure(t *testing.T) {
	cmd := findSubcommand(t, "login")
	if cmd == nil {
		t.Fatal("login subcommand not registered on root")
	}
	if cmd.Use != "login" {
		t.Errorf("Use = %q, want 'login'", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("login should have a short description")
	}
}

func TestLoginCmd_Args(t *testing.T) {
	cmd := findSubcommand(t, "login")
	if cmd == nil {
		t.Fatal("login command not found")
	}

	// login takes no args.
	err := cmd.Args(cmd, []string{})
	if err != nil {
		t.Fatalf("login with 0 args should be ok: %v", err)
	}

	// Extra arg should error.
	err = cmd.Args(cmd, []string{"extra"})
	if err == nil {
		t.Fatal("expected error for extra arg")
	}
}

func TestLoginCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"login", "--help"})
	rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "login") {
		t.Fatal("help output should mention 'login'")
	}
	if !strings.Contains(output, "GitHub") {
		t.Fatal("help output should mention 'GitHub'")
	}
}

// findSubcommand finds a registered subcommand by name.
func findSubcommand(t *testing.T, name string) *cobra.Command {
	t.Helper()
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == name {
			return sub
		}
	}
	return nil
}
