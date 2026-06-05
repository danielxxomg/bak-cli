package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// resetBackupVars resets package-level flag variables between tests.
func resetBackupVars() {
	backupPreset = "quick"
	backupAdapter = ""
}

// --- backup command execution tests ---

func TestRunBackup_InvalidPreset(t *testing.T) {
	resetBackupVars()

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"backup", "--preset", "nonexistent_xyz"})
	err := rootCmd.Execute()

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
}
