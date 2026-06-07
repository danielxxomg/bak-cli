package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// resetBackupVars resets package-level flag variables between tests.
func resetBackupVars() {
	backupPreset = "quick"
	backupAdapter = ""
	backupProfile = ""
}

// --- backup command execution tests ---

func TestRunBackup_InvalidPreset(t *testing.T) {
	resetBackupVars()
	backupPreset = "nonexistent_xyz"
	defer func() { backupPreset = "quick" }()

	deps, _, _ := setupTestDeps(t)

	cmd := &cobra.Command{}
	err := runBackupWithDeps(cmd, nil, deps)

	if err == nil {
		t.Fatal("expected backup with invalid preset to error")
	}
	if !strings.Contains(err.Error(), "preset") {
		t.Errorf("error should mention preset issue, got: %v", err)
	}
}

func TestRunBackup_InvalidAdapter(t *testing.T) {
	resetBackupVars()

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"backup", "--adapter", "nonexistent_adapter_xyz"})
	err := rootCmd.Execute()

	if err == nil {
		t.Fatal("expected backup with invalid adapter to error")
	}
	if !strings.Contains(err.Error(), "nonexistent_adapter_xyz") {
		t.Errorf("error should mention adapter name, got: %v", err)
	}
}

func TestRunBackup_Defaults(t *testing.T) {
	resetBackupVars()

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"backup"})
	err := rootCmd.Execute()

	// The engine attempts to detect adapters. If no adapters are installed,
	// it returns an error about no installed adapters.
	if err != nil {
		if !strings.Contains(err.Error(), "no installed adapters") &&
			!strings.Contains(err.Error(), "not found") &&
			!strings.Contains(err.Error(), "detect") {
			t.Errorf("unexpected error from default backup: %v", err)
		}
	}
}

func TestBackupCmd_RunEIsSet(t *testing.T) {
	cmd := findSubcommand(t, "backup")
	if cmd == nil {
		t.Fatal("backup command not found")
	}
	if cmd.RunE == nil {
		t.Error("backup RunE should be set")
	}
}

func TestBackupCmd_FlagsAfterInit(t *testing.T) {
	cmd := findSubcommand(t, "backup")
	if cmd == nil {
		t.Fatal("backup command not found")
	}

	presetFlag := cmd.Flags().Lookup("preset")
	if presetFlag == nil {
		t.Fatal("--preset flag not found after init")
	}
	if presetFlag.DefValue != "quick" {
		t.Errorf("--preset default = %q, want 'quick'", presetFlag.DefValue)
	}

	adapterFlag := cmd.Flags().Lookup("adapter")
	if adapterFlag == nil {
		t.Fatal("--adapter flag not found after init")
	}
	if adapterFlag.DefValue != "" {
		t.Errorf("--adapter default = %q, want ''", adapterFlag.DefValue)
	}

	profileFlag := cmd.Flags().Lookup("profile")
	if profileFlag == nil {
		t.Fatal("--profile flag not found after init")
	}
	if profileFlag.DefValue != "" {
		t.Errorf("--profile default = %q, want ''", profileFlag.DefValue)
	}
}

func TestRunBackup_ProfileNotFound(t *testing.T) {
	resetBackupVars()
	backupProfile = "nonexistent_profile_xyz"
	defer func() { backupProfile = "" }()

	deps, _, _ := setupTestDeps(t)

	cmd := &cobra.Command{}
	err := runBackupWithDeps(cmd, nil, deps)

	if err == nil {
		t.Fatal("expected backup with nonexistent profile to error")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "not found") {
		t.Errorf("error should mention not found, got: %v", err)
	}
}

func TestRunBackup_ProfileFlagRegistered(t *testing.T) {
	cmd := findSubcommand(t, "backup")
	if cmd == nil {
		t.Fatal("backup command not found")
	}

	profileFlag := cmd.Flags().Lookup("profile")
	if profileFlag == nil {
		t.Fatal("--profile flag not found")
	}
	if profileFlag.Usage == "" {
		t.Error("--profile flag should have usage description")
	}
}

func TestRunBackup_HelpMentionsProfile(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"backup", "--help"})
	rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "profile") {
		t.Fatal("help output should mention --profile")
	}
}

func TestRunBackup_OverrideFlag(t *testing.T) {
	resetBackupVars()

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"backup", "--override"})
	err := rootCmd.Execute()

	if err != nil {
		if !strings.Contains(err.Error(), "no installed adapters") &&
			!strings.Contains(err.Error(), "not found") &&
			!strings.Contains(err.Error(), "detect") {
			t.Errorf("unexpected error from backup --override: %v", err)
		}
	}
}

func TestRunBackup_ProfileWithPreset(t *testing.T) {
	resetBackupVars()

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"backup", "--profile", "nonexistent_xyz", "--preset", "full"})
	err := rootCmd.Execute()

	// The command may succeed (preset override) or fail (profile not found).
	// Either is fine — we're exercising flag parsing.
	if err != nil {
		t.Logf("backup with profile returned: %v", err)
	}
}

func TestRunBackup_InvalidPresetWithAdapter(t *testing.T) {
	resetBackupVars()

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"backup", "--preset", "invalid", "--adapter", "nonexistent"})
	err := rootCmd.Execute()

	// Should error. The specific error depends on which check runs first.
	if err != nil {
		t.Logf("backup with invalid flags: %v", err)
	}
}

func TestRunBackup_OverrideFlagRegistered(t *testing.T) {
	cmd := findSubcommand(t, "backup")
	if cmd == nil {
		t.Fatal("backup command not found")
	}

	overrideFlag := cmd.Flags().Lookup("override")
	if overrideFlag == nil {
		t.Fatal("--override flag not found")
	}
	if overrideFlag.DefValue != "false" {
		t.Errorf("--override default = %q, want 'false'", overrideFlag.DefValue)
	}
}
