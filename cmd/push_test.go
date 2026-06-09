package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/actions"
)

// --- resolveBackupID tests ---

func TestResolveBackupID_ExplicitArg(t *testing.T) {
	id, err := actions.ResolveBackupID("/nonexistent", []string{"20260604-150405"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "20260604-150405" {
		t.Errorf("got %q, want '20260604-150405'", id)
	}
}

func TestResolveBackupID_Latest(t *testing.T) {
	backupsDir := t.TempDir()

	ids := []string{"20260601-120000", "20260604-150405", "20260603-080000"}
	for _, id := range ids {
		if err := os.MkdirAll(filepath.Join(backupsDir, id), 0755); err != nil {
			t.Fatal(err)
		}
	}

	id, err := actions.ResolveBackupID(backupsDir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "20260604-150405" {
		t.Errorf("got %q, want '20260604-150405'", id)
	}
}

func TestResolveBackupID_EmptyArg(t *testing.T) {
	backupsDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(backupsDir, "20260604-150405"), 0755); err != nil {
		t.Fatal(err)
	}

	id, err := actions.ResolveBackupID(backupsDir, []string{""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "20260604-150405" {
		t.Errorf("got %q, want '20260604-150405'", id)
	}
}

func TestResolveBackupID_NoBackups(t *testing.T) {
	backupsDir := t.TempDir()

	_, err := actions.ResolveBackupID(backupsDir, nil)
	if err == nil {
		t.Fatal("expected error for empty backups dir")
	}
	if !strings.Contains(err.Error(), "no backups found") {
		t.Errorf("error should mention 'no backups found', got: %v", err)
	}
}

func TestResolveBackupID_DirNotFound(t *testing.T) {
	nonexistentDir := filepath.Join(t.TempDir(), "definitely-not-real")
	_, err := actions.ResolveBackupID(nonexistentDir, nil)
	if err == nil {
		t.Fatal("expected error for non-existent dir")
	}
}

func TestResolveBackupID_OnlyDirsConsidered(t *testing.T) {
	backupsDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(backupsDir, "not-a-backup.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(backupsDir, "20260604-150405"), 0755); err != nil {
		t.Fatal(err)
	}

	id, err := actions.ResolveBackupID(backupsDir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "20260604-150405" {
		t.Errorf("got %q, want '20260604-150405'", id)
	}
}

func TestResolveBackupID_SingleBackup(t *testing.T) {
	backupsDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(backupsDir, "20260101-000000"), 0755); err != nil {
		t.Fatal(err)
	}

	id, err := actions.ResolveBackupID(backupsDir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "20260101-000000" {
		t.Errorf("got %q, want '20260101-000000'", id)
	}
}

// --- push command execution tests ---

func TestRunPushWithDeps_Delegation(t *testing.T) {
	// Verify runPushWithDeps creates PushAction and delegates without panic.
	// The actual Run() result depends on token/config state — any error is fine.
	deps, _, _ := setupTestDeps(t)

	cmd := findSubcommand(t, "push")
	if cmd == nil {
		t.Fatal("push command not found")
	}
	err := runPushWithDeps(cmd, []string{"20250101-000000"}, deps)

	// We expect an error (no real token, backup doesn't exist), but not a panic.
	// The key assertion: the wrapper successfully created the action and called Run.
	if err == nil {
		t.Skip("push succeeded — valid GitHub token configured")
	}
	// Verify error is from the actions layer, not a nil pointer or wiring issue.
	errStr := err.Error()
	if !strings.Contains(errStr, "not found") &&
		!strings.Contains(errStr, "token") &&
		!strings.Contains(errStr, "backup") &&
		!strings.Contains(errStr, "resolve") {
		t.Errorf("unexpected error from push delegation: %v", err)
	}
}

func TestRunPush_ErrorsAppropriately(t *testing.T) {
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	// Reset cobra internal state by setting args to nil first.
	rootCmd.SetArgs(nil)
	rootCmd.SetArgs([]string{"push"})

	err := rootCmd.Execute()
	if err == nil {
		t.Skip("push succeeded — valid GitHub token configured")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "token") &&
		!strings.Contains(errStr, "gist") &&
		!strings.Contains(errStr, "Gist") &&
		!strings.Contains(errStr, "backup") {
		t.Errorf("unexpected push error: %v", err)
	}
}

func TestRunPush_BackupNotFound(t *testing.T) {
	// Push with an explicit backup ID that doesn't exist should fail
	// with a "not found" error, before even checking the token.
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"push", "20250101-000000"})
	err := rootCmd.Execute()

	if err == nil {
		t.Skip("push succeeded (token configured, backup somehow exists)")
	}

	errStr := err.Error()
	// Should fail with "not found" or "token" depending on which check runs first.
	if !strings.Contains(errStr, "not found") &&
		!strings.Contains(errStr, "token") {
		t.Errorf("unexpected push error: %v", err)
	}
}

func TestRunPush_TooManyArgs(t *testing.T) {
	// push accepts Max 1 arg (cobra.MaximumNArgs(1)).
	// Test the arg validator directly since cobra.Execute may handle routing differently.
	cmd := findSubcommand(t, "push")
	if cmd == nil {
		t.Fatal("push command not found")
	}
	err := cmd.Args(cmd, []string{"abc", "xyz"})
	if err == nil {
		t.Fatal("expected push command to reject 2 args")
	}
}
