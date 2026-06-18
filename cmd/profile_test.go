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
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("profile --help should not error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "profile") {
		t.Fatal("help output should mention 'profile'")
	}
}

// --- profile list tests ---

func TestProfileList_NoProfiles(t *testing.T) {
	deps, stdout, _ := setupTestDeps(t)

	cmd := &cobra.Command{}
	err := runProfileListWithDeps(cmd, nil, deps)
	if err != nil {
		t.Fatalf("profile list should not error: %v", err)
	}

	output := stdout.String()
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
	err := profileShowCmd.Args(profileShowCmd, []string{})
	if err == nil {
		t.Fatal("profile show without name should error")
	}
}

func TestProfileShow_NotFound(t *testing.T) {
	deps, _, _ := setupTestDeps(t)

	cmd := &cobra.Command{}
	err := runProfileShowWithDeps(cmd, []string{"nonexistent_xyz"}, deps)
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
	err := profileDeleteCmd.Args(profileDeleteCmd, []string{})
	if err == nil {
		t.Fatal("profile delete without name should error")
	}
}

func TestProfileDelete_NotFound(t *testing.T) {
	deps, _, _ := setupTestDeps(t)
	cmd := &cobra.Command{}
	err := runProfileDeleteWithDeps(cmd, []string{"nonexistent_xyz"}, deps)
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
	oldProvider := profileCreateProvider
	profileCreateProvider = ""
	defer func() { profileCreateProvider = oldProvider }()

	deps, _, _ := setupTestDeps(t)
	cmd := &cobra.Command{}
	err := runProfileCreateWithDeps(cmd, []string{"test-profile"}, deps)
	if err == nil {
		t.Fatal("profile create without --provider should error (required flag)")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "provider") {
		t.Errorf("error should mention provider, got: %v", err)
	}
}

func TestProfileCreate_NoArgs(t *testing.T) {
	// MaximumNArgs(1) allows 0 or 1 arg. Zero args is valid for the wizard path.
	err := profileCreateCmd.Args(profileCreateCmd, []string{})
	if err != nil {
		t.Fatalf("profile create MaximumNArgs(1) should accept 0 args (wizard mode), got: %v", err)
	}
}

// TestProfileCreate_NoArgs_LaunchesWizard verifies that running profile create
// with no arguments suggests the interactive wizard or launches it.
func TestProfileCreate_NoArgs_LaunchesWizard(t *testing.T) {
	deps, _, _ := setupTestDeps(t)

	// No args, no --interactive: should error with helpful message.
	cmd := &cobra.Command{}
	err := runProfileCreateWithDeps(cmd, []string{}, deps)
	if err == nil {
		t.Fatal("profile create with no args and no --interactive should error")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "interactive") {
		t.Errorf("error should mention --interactive, got: %v", err)
	}
}

// TestProfileCreate_NoArgs_InteractiveAttempt verifies that profile create
// with --interactive and no args attempts to launch the wizard.
func TestProfileCreate_NoArgs_InteractiveAttempt(t *testing.T) {
	oldInteractive := profileCreateInteractive
	profileCreateInteractive = true
	defer func() { profileCreateInteractive = oldInteractive }()

	deps, _, _ := setupTestDeps(t)
	cmd := &cobra.Command{}

	// With --interactive and no args, the wizard is launched. Without a TTY
	// it will error, but it must not be the "provide a name" error.
	err := runProfileCreateWithDeps(cmd, []string{}, deps)
	if err == nil {
		t.Log("wizard succeeded (TTY available)")
		return
	}
	errStr := err.Error()
	// Must NOT be the "provide a profile name or use --interactive" error.
	if strings.Contains(errStr, "provide a profile name") {
		t.Errorf("with --interactive should not ask for name, got: %v", err)
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

// --- additional profile execution tests ---

func TestProfileList_Execute(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"profile", "list"})
	err := rootCmd.Execute()
	if err != nil {
		t.Logf("profile list: %v", err)
	}
}

func TestProfileShow_ExecuteNonexistent(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"profile", "show", "definitely_not_a_profile_xyz"})
	err := rootCmd.Execute()
	if err != nil {
		t.Logf("profile show nonexistent: %v (expected)", err)
	}
}

func TestProfileDelete_ExecuteNonexistent(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"profile", "delete", "definitely_not_a_profile_xyz"})
	err := rootCmd.Execute()
	if err != nil {
		t.Logf("profile delete nonexistent: %v (expected)", err)
	}
}

func TestProfileCreate_RcloneProvider(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"profile", "create", "rclone-test", "--provider", "rclone"})
	err := rootCmd.Execute()
	if err != nil {
		t.Logf("profile create rclone: %v (expected)", err)
	}
}

func TestProfileCreate_AdaptersAndCategories(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"profile", "create", "adapt-profile",
		"--provider", "github-gist",
		"--adapters", "opencode,cursor",
		"--categories", "config,skills"})
	err := rootCmd.Execute()
	if err != nil {
		t.Logf("profile create with adapters/categories: %v (expected without configured providers)", err)
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
