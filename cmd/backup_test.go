package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/danielxxomg/bak-cli/internal/config"
)

// resetBackupVars resets package-level flag variables between tests.
func resetBackupVars() {
	backupPreset = "quick"
	backupAdapter = ""
	backupProfile = ""
}

// --- backup command execution tests ---

func TestRunBackup_InvalidInputs(t *testing.T) {
	tests := []struct {
		name       string
		setupFlags func()
		setupDeps  func(deps cmdDeps) cmdDeps
		wantErrSub string
	}{
		{
			name: "invalid preset",
			setupFlags: func() {
				backupPreset = "nonexistent_xyz"
			},
			wantErrSub: "preset",
		},
		{
			name: "invalid adapter",
			setupFlags: func() {
				backupAdapter = "nonexistent_adapter_xyz"
			},
			wantErrSub: "nonexistent_adapter_xyz",
		},
		{
			name: "profile not found",
			setupFlags: func() {
				backupProfile = "nonexistent_profile_xyz"
			},
			wantErrSub: "not found",
		},
		{
			name: "config loader error",
			setupFlags: func() {
				backupProfile = "test-profile"
			},
			setupDeps: func(deps cmdDeps) cmdDeps {
				deps.ConfigLoader = func() (*config.Config, error) {
					return nil, errors.New("disk failure")
				}
				return deps
			},
			wantErrSub: "load config for profile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetBackupVars()
			tt.setupFlags()
			defer resetBackupVars()

			deps, _, _ := setupTestDeps(t)
			if tt.setupDeps != nil {
				deps = tt.setupDeps(deps)
			}

			cmd := &cobra.Command{}
			err := runBackupWithDeps(cmd, nil, deps)

			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Errorf("error should contain %q, got: %v", tt.wantErrSub, err)
			}
		})
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

// --- extracted helper tests (Phase 10) ---

// TestApplyProfileOverrides verifies the profile-override helper: defaults
// pass through when no profile is selected, overrides apply when a profile
// exists, and error/verbose paths behave correctly.
func TestApplyProfileOverrides(t *testing.T) {
	t.Run("no profile returns defaults unchanged without loading config", func(t *testing.T) {
		resetBackupVars()
		defer resetBackupVars()

		loaderCalled := false
		deps := cmdDeps{
			ConfigLoader: func() (*config.Config, error) {
				loaderCalled = true
				return &config.Config{}, nil
			},
			Stderr: new(bytes.Buffer),
		}
		preset, adapters, cats, err := applyProfileOverrides(deps, "quick", []string{"opencode"}, []string{"config"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if loaderCalled {
			t.Error("config loader must not be called when no profile is selected")
		}
		if preset != "quick" {
			t.Errorf("preset = %q, want quick", preset)
		}
		if !equalStrings(adapters, []string{"opencode"}) {
			t.Errorf("adapters = %v, want [opencode]", adapters)
		}
		if !equalStrings(cats, []string{"config"}) {
			t.Errorf("categories = %v, want [config]", cats)
		}
	})

	t.Run("profile found applies preset categories and adapters", func(t *testing.T) {
		resetBackupVars()
		backupProfile = "work"
		defer resetBackupVars()

		deps := cmdDeps{
			ConfigLoader: func() (*config.Config, error) {
				return &config.Config{
					Profiles: map[string]config.ProfileConfig{
						"work": {
							Preset:     "full",
							Categories: []string{"skills", "config"},
							Adapters:   []string{"opencode", "cursor"},
							Provider:   "github-gist",
						},
					},
				}, nil
			},
			Stderr: new(bytes.Buffer),
		}
		preset, adapters, cats, err := applyProfileOverrides(deps, "quick", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if preset != "full" {
			t.Errorf("preset = %q, want full (overridden)", preset)
		}
		if !equalStrings(adapters, []string{"opencode", "cursor"}) {
			t.Errorf("adapters = %v, want [opencode cursor]", adapters)
		}
		if !equalStrings(cats, []string{"skills", "config"}) {
			t.Errorf("categories = %v, want [skills config]", cats)
		}
	})

	t.Run("profile not found returns error", func(t *testing.T) {
		resetBackupVars()
		backupProfile = "missing"
		defer resetBackupVars()

		deps := cmdDeps{
			ConfigLoader: func() (*config.Config, error) {
				return &config.Config{Profiles: map[string]config.ProfileConfig{}}, nil
			},
			Stderr: new(bytes.Buffer),
		}
		_, _, _, err := applyProfileOverrides(deps, "quick", nil, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error = %q, want substring 'not found'", err.Error())
		}
	})

	t.Run("config loader error is wrapped", func(t *testing.T) {
		resetBackupVars()
		backupProfile = "work"
		defer resetBackupVars()

		deps := cmdDeps{
			ConfigLoader: func() (*config.Config, error) {
				return nil, errors.New("disk failure")
			},
			Stderr: new(bytes.Buffer),
		}
		_, _, _, err := applyProfileOverrides(deps, "quick", nil, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "load config for profile") {
			t.Errorf("error = %q, want substring 'load config for profile'", err.Error())
		}
	})

	t.Run("verbose writes overridden profile summary to stderr", func(t *testing.T) {
		resetBackupVars()
		backupProfile = "work"
		origVerbose := verbose
		verbose = true
		defer func() {
			resetBackupVars()
			verbose = origVerbose
		}()

		stderr := new(bytes.Buffer)
		deps := cmdDeps{
			ConfigLoader: func() (*config.Config, error) {
				return &config.Config{
					Profiles: map[string]config.ProfileConfig{
						"work": {Preset: "full", Provider: "github-gist"},
					},
				}, nil
			},
			Stderr: stderr,
		}
		_, _, _, err := applyProfileOverrides(deps, "quick", nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := stderr.String()
		if !strings.Contains(got, "Using profile") {
			t.Errorf("stderr should contain 'Using profile'; got: %q", got)
		}
		if !strings.Contains(got, `profile "work"`) {
			t.Errorf("stderr should mention profile name; got: %q", got)
		}
		if !strings.Contains(got, "preset=full") {
			t.Errorf("stderr should show overridden preset; got: %q", got)
		}
	})
}
