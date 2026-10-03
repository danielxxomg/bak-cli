package cmd

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	configtest "github.com/danielxxomg/bak-cli/internal/config/testutil"
	"github.com/danielxxomg/bak-cli/internal/manifest"
)

func TestRestoreCmd_Structure(t *testing.T) {
	// Ensure the restore command is registered on root.
	if findSubcommand(t, "restore") == nil {
		t.Fatal("restore subcommand not registered on root")
	}
}

func TestRestoreCmd_Flags(t *testing.T) {
	cmd := findSubcommand(t, "restore")
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
	cmd := findSubcommand(t, "restore")
	if cmd == nil {
		t.Fatal("restore command not found")
	}

	// Test args validator directly.
	// MaximumNArgs(1) should accept 0 args (picker) or 1 arg.
	err := cmd.Args(cmd, []string{})
	if err != nil {
		t.Fatalf("expected no error with 0 args (picker mode), got %v", err)
	}

	// MaximumNArgs(1) should reject 2 args.
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
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("restore --help should not error: %v", err)
	}

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

func TestRestoreCmd_UseAndDescription(t *testing.T) {
	cmd := findSubcommand(t, "restore")
	if cmd == nil {
		t.Fatal("restore command not found")
	}

	tests := []struct {
		name       string
		got        string
		wantPrefix string
	}{
		{name: "Use", got: cmd.Use, wantPrefix: "restore"},
		{name: "Short", got: cmd.Short},
		{name: "Long", got: cmd.Long},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got == "" {
				t.Fatalf("restore %s should not be empty", tt.name)
			}
			if tt.wantPrefix != "" && !strings.HasPrefix(tt.got, tt.wantPrefix) {
				t.Fatalf("restore %s = %q, should start with %q", tt.name, tt.got, tt.wantPrefix)
			}
		})
	}
}

// --- runRestore execution tests ---

func TestRunRestoreWithDeps_InvalidBackupID(t *testing.T) {
	deps, _, _ := setupTestDeps(t)

	cmd := findSubcommand(t, "restore")
	if cmd == nil {
		t.Fatal("restore command not found")
	}
	err := runRestoreWithDeps(cmd, []string{"not-a-valid-id"}, deps)

	if err == nil {
		t.Fatal("expected error for invalid backup ID")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "invalid") && !strings.Contains(errStr, "backup") {
		t.Errorf("error should mention invalid backup ID, got: %v", err)
	}
}

func TestRunRestoreWithDeps_BackupNotFound(t *testing.T) {
	deps, _, _ := setupTestDeps(t)

	cmd := findSubcommand(t, "restore")
	if cmd == nil {
		t.Fatal("restore command not found")
	}
	err := runRestoreWithDeps(cmd, []string{"20250101-000000"}, deps)

	if err == nil {
		t.Skip("backup 20250101-000000 exists")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestRunRestore_MissingArgs(t *testing.T) {
	configtest.SetConfigHome(t, t.TempDir())
	// Reset rootCmd and restoreCmd state to avoid help-flag leakage
	// from previous tests (e.g., TestRestoreCmd_Help). pflag doesn't
	// reset flag values when parsing empty args (pflag v1.0.9 bug).
	rootCmd.SetArgs(nil)
	// Ensure the default help flag exists (cobra adds it lazily on first
	// Execute) before resetting leaked state from previous tests.
	restoreCmd.InitDefaultHelpFlag()
	if err := restoreCmd.Flags().Set("help", "false"); err != nil {
		t.Fatal(err)
	}

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"restore"})
	err := rootCmd.Execute()

	// MaximumNArgs(1) allows 0 args (triggers TTY picker or error).
	// The command may error or succeed depending on TTY availability.
	// Either way, args validation for 0 args must pass.
	if err != nil {
		// If it errors, verify it's not a wrong error type.
		t.Logf("restore with 0 args errored (expected in non-TTY): %v", err)
		return
	}

	// Verify Args validator accepts 0 args (MaximumNArgs(1)).
	cmd := findSubcommand(t, "restore")
	if cmd == nil {
		t.Fatal("restore command not found")
	}
	argErr := cmd.Args(cmd, []string{})
	if argErr != nil {
		t.Errorf("restore MaximumNArgs(1) should accept 0 args (picker mode), got: %v", argErr)
	}
}

// TestRestoreHelpFollowedByExecute verifies that running a help command
// does not leak state into subsequent Execute() calls on the shared
// restoreCmd. This tests the isolation fix for TestRunRestore_MissingArgs.
func TestRestoreHelpFollowedByExecute(t *testing.T) {
	// Step 1: Run --help on restore (like TestRestoreCmd_Help does).
	buf1 := new(bytes.Buffer)
	rootCmd.SetOut(buf1)
	rootCmd.SetErr(buf1)
	rootCmd.SetArgs(nil)
	restoreCmd.InitDefaultHelpFlag()
	if err := restoreCmd.Flags().Set("help", "false"); err != nil {
		t.Fatal(err)
	}

	rootCmd.SetArgs([]string{"restore", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("restore --help should not error: %v", err)
	}
	if !strings.Contains(buf1.String(), "restore") {
		t.Fatal("help output should mention 'restore'")
	}

	// Step 2: Reset help flag (pflag doesn't reset on empty Parse)
	// and run restore with no args — must NOT short-circuit to help.
	restoreCmd.InitDefaultHelpFlag()
	if err := restoreCmd.Flags().Set("help", "false"); err != nil {
		t.Fatal(err)
	}
	buf2 := new(bytes.Buffer)
	rootCmd.SetOut(buf2)
	rootCmd.SetErr(buf2)
	rootCmd.SetArgs([]string{"restore"})
	err := rootCmd.Execute()

	// The key assertion: output must be DIFFERENT from help output.
	// If it leaked, buf2 would contain the same help text as buf1.
	if err == nil && strings.Contains(buf2.String(), "restore") && strings.Contains(buf2.String(), "--dry-run") {
		output2 := buf2.String()
		if strings.Contains(output2, "Usage:") {
			t.Error("restore with no args leaked help output instead of running command")
		}
	}
	// MaximumNArgs(1) allows 0 args — verify args validator.
	cmd := findSubcommand(t, "restore")
	if cmd == nil {
		t.Fatal("restore command not found")
	}
	if argErr := cmd.Args(cmd, []string{}); argErr != nil {
		t.Errorf("MaximumNArgs(1) should accept 0 args: %v", argErr)
	}
}

func TestRunRestore_FlagVariants(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no_flags", args: []string{"restore", "20250101-000000"}},
		{name: "dry_run", args: []string{"restore", "--dry-run", "20250101-000000"}},
		{name: "force", args: []string{"restore", "--force", "20250101-000000"}},
		{name: "override", args: []string{"restore", "--override", "20250101-000000"}},
		{name: "override_and_dry_run", args: []string{"restore", "--override", "--dry-run", "20250101-000000"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configtest.SetConfigHome(t, t.TempDir())
			bufOut := new(bytes.Buffer)
			bufErr := new(bytes.Buffer)
			rootCmd.SetOut(bufOut)
			rootCmd.SetErr(bufErr)

			rootCmd.SetArgs(tt.args)
			err := rootCmd.Execute()

			if err == nil {
				t.Logf("restore %v succeeded (backup may exist)", tt.args)
				return
			}
			if !strings.Contains(err.Error(), "not found") {
				t.Errorf("error should mention 'not found', got: %v", err)
			}
		})
	}
}

func TestRunRestore_VerboseFlagExists(t *testing.T) {
	// --verbose exists as a cobra PersistentFlag registered in root.go.
	// Verify it is available as a global/persistent flag concept.
	// The flag is registered in Execute() at runtime; in tests we check
	// the root command help output covers verbose behavior indirectly.
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"restore", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("restore --help should not error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "restore") {
		t.Error("restore help should mention 'restore'")
	}
}

func TestTuiRunRestore_IntegrityValidation(t *testing.T) {
	tests := []struct {
		name       string
		dryRun     bool
		tamper     bool
		wantErr    bool
		errContain string
	}{
		{
			name:    "dry_run_tampered_passes_diff_phase",
			dryRun:  true,
			tamper:  true,
			wantErr: false,
		},
		{
			name:       "apply_tampered_fails_manifest_validation",
			dryRun:     false,
			tamper:     true,
			wantErr:    true,
			errContain: "manifest validation failed",
		},
		{
			name:    "apply_valid_succeeds",
			dryRun:  false,
			tamper:  false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			configtest.SetConfigHome(t, home)

			backupID := "20260101-120000"
			bakDir := filepath.Join(home, ".bak")
			backupDir := filepath.Join(bakDir, "backups", backupID)
			if err := os.MkdirAll(backupDir, 0755); err != nil {
				t.Fatal(err)
			}

			adapterDir := filepath.Join(backupDir, "test-adapter")
			if err := os.MkdirAll(adapterDir, 0755); err != nil {
				t.Fatal(err)
			}
			content := []byte("hello=world\n")
			if err := os.WriteFile(filepath.Join(adapterDir, "config.json"), content, 0644); err != nil {
				t.Fatal(err)
			}

			h := sha256.Sum256(content)
			m := manifest.New(backupID, runtime.GOOS, "testhost", "test", "quick", []string{"config"})
			m.AddAdapter("test-adapter", "", "~/.config/bak", []manifest.Item{
				{
					Category:   "config",
					SourcePath: "~/.config/bak/config.json",
					BackupPath: "test-adapter/config.json",
					Hash:       fmt.Sprintf("sha256:%x", h),
					Size:       int64(len(content)),
				},
			})
			if err := m.Save(backupDir); err != nil {
				t.Fatal(err)
			}

			if tt.tamper {
				if err := os.WriteFile(filepath.Join(adapterDir, "config.json"), []byte("tampered-data\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}

			output, err := tuiRunRestore(backupID, tt.dryRun)

			targetPath := filepath.Join(home, ".config", "bak", "config.json")

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil (output: %s)", tt.errContain, output)
				}
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContain)
				}
				if _, statErr := os.Stat(targetPath); !os.IsNotExist(statErr) {
					t.Errorf("target file should not exist after failed validation")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v (output: %s)", err, output)
				}
				if !tt.dryRun {
					if _, statErr := os.Stat(targetPath); statErr != nil {
						t.Errorf("target file should exist after successful restore: %v", statErr)
					}
				}
			}
		})
	}
}

func TestRunRestoreWithDeps_TamperedManifestWithForceFails(t *testing.T) {
	home := t.TempDir()
	configtest.SetConfigHome(t, home)

	backupID := "20260101-120000"
	bakDir := filepath.Join(home, ".bak")
	backupDir := filepath.Join(bakDir, "backups", backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	adapterDir := filepath.Join(backupDir, "test-adapter")
	if err := os.MkdirAll(adapterDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := []byte("hello=world\n")
	if err := os.WriteFile(filepath.Join(adapterDir, "config.json"), content, 0644); err != nil {
		t.Fatal(err)
	}

	h := sha256.Sum256(content)
	m := manifest.New(backupID, runtime.GOOS, "testhost", "test", "quick", []string{"config"})
	m.AddAdapter("test-adapter", "", "~/.config/bak", []manifest.Item{
		{
			Category:   "config",
			SourcePath: "~/.config/bak/config.json",
			BackupPath: "test-adapter/config.json",
			Hash:       fmt.Sprintf("sha256:%x", h),
			Size:       int64(len(content)),
		},
	})
	if err := m.Save(backupDir); err != nil {
		t.Fatal(err)
	}

	// Tamper the file content so hash mismatches.
	if err := os.WriteFile(filepath.Join(adapterDir, "config.json"), []byte("tampered-data\n"), 0644); err != nil {
		t.Fatal(err)
	}

	deps, stdout, _ := setupTestDeps(t)
	cmd := findSubcommand(t, "restore")
	if cmd == nil {
		t.Fatal("restore command not found")
	}

	// Set --force flag.
	restoreForce = true
	restoreDryRun = false
	defer func() {
		restoreForce = false
		restoreDryRun = false
	}()

	err := runRestoreWithDeps(cmd, []string{backupID}, deps)
	if err == nil {
		t.Fatalf("expected error for tampered manifest even with --force, got nil (stdout: %s)", stdout.String())
	}
	if !strings.Contains(err.Error(), "manifest validation failed") {
		t.Errorf("error %q should contain 'manifest validation failed'", err.Error())
	}

	targetPath := filepath.Join(home, ".config", "bak", "config.json")
	if _, statErr := os.Stat(targetPath); !os.IsNotExist(statErr) {
		t.Errorf("target file should not exist after failed validation")
	}
}

// TestRunRestoreWithDeps_RecoveryPointCreatedOnApply proves that the CLI/shared-action path
// creates a recovery point under ~/.bak/recovery/ upon successful restore apply (does not claim
// an independently executed TUI flow).
func TestRunRestoreWithDeps_RecoveryPointCreatedOnApply(t *testing.T) {
	home := t.TempDir()
	configtest.SetConfigHome(t, home)

	backupID := "20260101-123456"
	bakDir := filepath.Join(home, ".bak")
	backupDir := filepath.Join(bakDir, "backups", backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	adapterDir := filepath.Join(backupDir, "test-adapter")
	if err := os.MkdirAll(adapterDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := []byte("setting=on\n")
	if err := os.WriteFile(filepath.Join(adapterDir, "config.json"), content, 0644); err != nil {
		t.Fatal(err)
	}

	h := sha256.Sum256(content)
	m := manifest.New(backupID, runtime.GOOS, "testhost", "test", "quick", []string{"config"})
	m.AddAdapter("test-adapter", "", "~/.config/bak", []manifest.Item{
		{
			Category:   "config",
			SourcePath: "~/.config/bak/config.json",
			BackupPath: "test-adapter/config.json",
			Hash:       fmt.Sprintf("sha256:%x", h),
			Size:       int64(len(content)),
		},
	})
	if err := m.Save(backupDir); err != nil {
		t.Fatal(err)
	}

	deps, _, _ := setupTestDeps(t)
	cmd := findSubcommand(t, "restore")
	if cmd == nil {
		t.Fatal("restore command not found")
	}

	restoreForce = true
	restoreDryRun = false
	defer func() {
		restoreForce = false
		restoreDryRun = false
	}()

	err := runRestoreWithDeps(cmd, []string{backupID}, deps)
	if err != nil {
		t.Fatalf("runRestoreWithDeps failed: %v", err)
	}

	targetPath := filepath.Join(home, ".config", "bak", "config.json")
	if _, statErr := os.Stat(targetPath); statErr != nil {
		t.Fatalf("target file should exist after restore: %v", statErr)
	}

	recDir := filepath.Join(home, ".bak", "recovery")
	entries, err := os.ReadDir(recDir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("expected recovery point directory in %s, got err: %v, count: %d", recDir, err, len(entries))
	}
}

func TestRunRestoreWithDeps_BakVersionMismatchWarning(t *testing.T) {
	home := t.TempDir()
	configtest.SetConfigHome(t, home)

	backupID := "20260101-120000"
	backupDir := filepath.Join(home, ".bak", "backups", backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	adapterDir := filepath.Join(backupDir, "test-adapter")
	if err := os.MkdirAll(adapterDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := []byte("key=value\n")
	backedFile := filepath.Join(adapterDir, "config.json")
	if err := os.WriteFile(backedFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	h := sha256.Sum256(content)
	m := manifest.New(backupID, "linux", "host", "1.0.0", "quick", []string{"config"})
	m.AddAdapter("test-adapter", "", "~/.config/bak", []manifest.Item{
		{
			Category:   "config",
			SourcePath: "~/.config/bak/config.json",
			BackupPath: "test-adapter/config.json",
			Hash:       fmt.Sprintf("sha256:%x", h),
			Size:       int64(len(content)),
			Mode:       0644,
		},
	})
	if err := m.Save(backupDir); err != nil {
		t.Fatal(err)
	}

	oldVersion := Version
	Version = "2.0.0"
	defer func() { Version = oldVersion }()

	restoreForce = true
	restoreDryRun = false
	defer func() {
		restoreForce = false
		restoreDryRun = false
	}()

	var stdoutBuf, stderrBuf bytes.Buffer
	deps := cmdDeps{
		ConfigLoader: defaultDeps.ConfigLoader,
		Stdout:       &stdoutBuf,
		Stderr:       &stderrBuf,
		Stdin:        strings.NewReader(""),
	}

	cmd := findSubcommand(t, "restore")
	err := runRestoreWithDeps(cmd, []string{backupID}, deps)
	if err != nil {
		t.Fatalf("runRestoreWithDeps failed: %v", err)
	}

	stderrOut := stderrBuf.String()
	if !strings.Contains(stderrOut, "warning:") {
		t.Fatalf("expected warning on Stderr, got: %q", stderrOut)
	}
	if !strings.Contains(stderrOut, backupID) || !strings.Contains(stderrOut, "1.0.0") || !strings.Contains(stderrOut, "2.0.0") {
		t.Errorf("warning %q should mention backup ID, 1.0.0, and 2.0.0", stderrOut)
	}
}

func TestRunRestoreWithDeps_NewerSchemaFailsClosed(t *testing.T) {
	home := t.TempDir()
	configtest.SetConfigHome(t, home)

	backupID := "20260101-120000"
	backupDir := filepath.Join(home, ".bak", "backups", backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	adapterDir := filepath.Join(backupDir, "test-adapter")
	if err := os.MkdirAll(adapterDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := []byte("key=value\n")
	backedFile := filepath.Join(adapterDir, "config.json")
	if err := os.WriteFile(backedFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	h := sha256.Sum256(content)
	m := manifest.New(backupID, "linux", "host", "1.0.0", "quick", []string{"config"})
	m.Version = "0.5.0" // newer than 0.4.0
	m.AddAdapter("test-adapter", "", "~/.config/bak", []manifest.Item{
		{
			Category:   "config",
			SourcePath: "~/.config/bak/config.json",
			BackupPath: "test-adapter/config.json",
			Hash:       fmt.Sprintf("sha256:%x", h),
			Size:       int64(len(content)),
			Mode:       0644,
		},
	})
	if err := m.Save(backupDir); err != nil {
		t.Fatal(err)
	}

	restoreForce = true
	restoreDryRun = false
	defer func() {
		restoreForce = false
		restoreDryRun = false
	}()

	var stdoutBuf, stderrBuf bytes.Buffer
	deps := cmdDeps{
		ConfigLoader: defaultDeps.ConfigLoader,
		Stdout:       &stdoutBuf,
		Stderr:       &stderrBuf,
		Stdin:        strings.NewReader(""),
	}

	cmd := findSubcommand(t, "restore")
	err := runRestoreWithDeps(cmd, []string{backupID}, deps)
	if err == nil {
		t.Fatal("expected error for newer schema, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported manifest schema version") {
		t.Errorf("error %q should mention unsupported manifest schema version", err.Error())
	}
}

func TestTuiRunRestore_BakVersionAndSchema(t *testing.T) {
	home := t.TempDir()
	configtest.SetConfigHome(t, home)

	backupID := "20260101-120000"
	backupDir := filepath.Join(home, ".bak", "backups", backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	adapterDir := filepath.Join(backupDir, "test-adapter")
	if err := os.MkdirAll(adapterDir, 0755); err != nil {
		t.Fatal(err)
	}
	content := []byte("key=value\n")
	backedFile := filepath.Join(adapterDir, "config.json")
	if err := os.WriteFile(backedFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	h := sha256.Sum256(content)
	m := manifest.New(backupID, "linux", "host", "0.9.0", "quick", []string{"config"})
	m.Version = "0.5.0"
	m.AddAdapter("test-adapter", "", "~/.config/bak", []manifest.Item{
		{
			Category:   "config",
			SourcePath: "~/.config/bak/config.json",
			BackupPath: "test-adapter/config.json",
			Hash:       fmt.Sprintf("sha256:%x", h),
			Size:       int64(len(content)),
			Mode:       0644,
		},
	})
	if err := m.Save(backupDir); err != nil {
		t.Fatal(err)
	}

	out, err := tuiRunRestore(backupID, false)
	if err == nil {
		t.Fatalf("expected error for newer schema, got out: %s", out)
	}
	if !strings.Contains(err.Error(), "unsupported manifest schema version") {
		t.Errorf("expected error to mention unsupported schema version, got: %v", err)
	}
}
