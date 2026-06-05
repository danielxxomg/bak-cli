package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// --- profile command structure ---

func TestProfileCmd_Registered(t *testing.T) {
	cmd := findSubcommand(t, "profile")
	if cmd == nil {
		t.Fatal("profile command not registered on root")
	}
}

func TestProfileCmd_Structure(t *testing.T) {
	cmd := findSubcommand(t, "profile")
	if cmd == nil {
		t.Fatal("profile command not registered")
	}
	if cmd.Use != "profile" {
		t.Errorf("Use = %q, want 'profile'", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("profile should have a short description")
	}
	if cmd.RunE != nil {
		t.Error("profile parent should not have RunE (subcommands do)")
	}
}

func TestProfileCmd_HasSubcommands(t *testing.T) {
	cmd := findSubcommand(t, "profile")
	if cmd == nil {
		t.Fatal("profile command not found")
	}

	subs := cmd.Commands()
	names := make(map[string]bool)
	for _, s := range subs {
		names[s.Name()] = true
	}

	expected := []string{"create", "list", "show", "delete"}
	for _, want := range expected {
		if !names[want] {
			t.Errorf("profile should have subcommand %q", want)
		}
	}
}

func TestProfileCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"profile", "--help"})
	rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "profile") {
		t.Fatal("help output should mention 'profile'")
	}
}

// --- profile list tests ---

func TestProfileList_NoProfiles(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"profile", "list"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("profile list should not error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No profiles") {
		t.Errorf("expected 'No profiles' message, got: %s", output)
	}
}

func TestProfileList_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"profile", "list", "--help"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("profile list --help should not error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "profile") {
		t.Fatal("help output should mention 'profile'")
	}
}

// --- profile show tests ---

func TestProfileShow_MissingArgs(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"profile", "show"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("profile show without name should error")
	}
}

func TestProfileShow_NotFound(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"profile", "show", "nonexistent_xyz"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("profile show with nonexistent name should error")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "not found") && !strings.Contains(errStr, "nonexistent_xyz") {
		t.Errorf("error should mention not found or the profile name, got: %v", err)
	}
}

func TestProfileShow_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"profile", "show", "--help"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("profile show --help should not error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "profile") {
		t.Fatal("help output should mention 'profile'")
	}
}

// --- profile delete tests ---

func TestProfileDelete_MissingArgs(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"profile", "delete"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("profile delete without name should error")
	}
}

func TestProfileDelete_NotFound(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"profile", "delete", "nonexistent_xyz"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("profile delete with nonexistent name should error")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "not found") && !strings.Contains(errStr, "nonexistent_xyz") {
		t.Errorf("error should mention not found or the profile name, got: %v", err)
	}
}

func TestProfileDelete_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"profile", "delete", "--help"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("profile delete --help should not error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "profile") {
		t.Fatal("help output should mention 'profile'")
	}
}

// --- profile create tests ---

func TestProfileCreate_MissingProvider(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"profile", "create", "test-profile"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("profile create without --provider should error (required flag)")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "provider") {
		t.Errorf("error should mention provider, got: %v", err)
	}
}

func TestProfileCreate_NoArgs(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"profile", "create"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("profile create without name should error")
	}
}

func TestProfileCreate_TooManyArgs(t *testing.T) {
	cmd := findSubcommand(t, "profile")
	if cmd == nil {
		t.Fatal("profile command not found")
	}
	createCmd := findSubcommandIn(t, "create", cmd)
	if createCmd == nil {
		t.Fatal("profile create not found")
	}

	err := createCmd.Args(createCmd, []string{"a", "b"})
	if err == nil {
		t.Fatal("expected error for 2 args")
	}
}

func TestProfileCreate_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"profile", "create", "--help"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("profile create --help should not error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "provider") {
		t.Fatal("help output should mention --provider")
	}
}

// --- profile flags ---

func TestProfileCreateCmd_Flags(t *testing.T) {
	cmd := findSubcommand(t, "profile")
	if cmd == nil {
		t.Fatal("profile command not found")
	}
	createCmd := findSubcommandIn(t, "create", cmd)
	if createCmd == nil {
		t.Fatal("profile create command not found")
	}

	providerFlag := createCmd.Flags().Lookup("provider")
	if providerFlag == nil {
		t.Fatal("--provider flag not defined")
	}

	presetFlag := createCmd.Flags().Lookup("preset")
	if presetFlag == nil {
		t.Fatal("--preset flag not defined")
	}
	if presetFlag.DefValue != "quick" {
		t.Errorf("--preset default = %q, want 'quick'", presetFlag.DefValue)
	}

	adaptersFlag := createCmd.Flags().Lookup("adapters")
	if adaptersFlag == nil {
		t.Fatal("--adapters flag not defined")
	}

	categoriesFlag := createCmd.Flags().Lookup("categories")
	if categoriesFlag == nil {
		t.Fatal("--categories flag not defined")
	}

	encryptFlag := createCmd.Flags().Lookup("encrypt")
	if encryptFlag == nil {
		t.Fatal("--encrypt flag not defined")
	}
	if encryptFlag.DefValue != "false" {
		t.Errorf("--encrypt default = %q, want 'false'", encryptFlag.DefValue)
	}

	interactiveFlag := createCmd.Flags().Lookup("interactive")
	if interactiveFlag == nil {
		t.Fatal("--interactive flag not defined")
	}
	if interactiveFlag.DefValue != "false" {
		t.Errorf("--interactive default = %q, want 'false'", interactiveFlag.DefValue)
	}
}

// --- profile integration tests (creates real profile in temp config) ---

// findSubcommandIn finds a subcommand within a parent command.
func findSubcommandIn(t *testing.T, name string, parent *cobra.Command) *cobra.Command {
	t.Helper()
	for _, sub := range parent.Commands() {
		if sub.Name() == name {
			return sub
		}
	}
	return nil
}
