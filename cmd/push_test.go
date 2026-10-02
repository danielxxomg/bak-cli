package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/actions"
	configtest "github.com/danielxxomg/bak-cli/internal/config/testutil"
)

// --- resolveBackupID tests ---

func TestResolveBackupID(t *testing.T) {
	setup := func(t *testing.T, dirs []string, files []string) string {
		t.Helper()
		backupsDir := t.TempDir()
		for _, dir := range dirs {
			if err := os.MkdirAll(filepath.Join(backupsDir, dir), 0755); err != nil {
				t.Fatal(err)
			}
		}
		for _, file := range files {
			if err := os.WriteFile(filepath.Join(backupsDir, file), []byte("hello"), 0644); err != nil {
				t.Fatal(err)
			}
		}
		return backupsDir
	}

	tests := []struct {
		name        string
		dirs        []string
		files       []string
		args        []string
		fixedDir    string
		wantID      string
		wantErr     bool
		errContains string
	}{
		{
			name:   "explicit arg",
			args:   []string{"20260604-150405"},
			wantID: "20260604-150405",
		},
		{
			name:   "latest of three",
			dirs:   []string{"20260601-120000", "20260604-150405", "20260603-080000"},
			wantID: "20260604-150405",
		},
		{
			name:   "empty arg resolves latest",
			dirs:   []string{"20260604-150405"},
			args:   []string{""},
			wantID: "20260604-150405",
		},
		{
			name:        "no backups errors",
			wantErr:     true,
			errContains: "no backups found",
		},
		{
			name:     "missing dir errors",
			fixedDir: "missing",
			wantErr:  true,
		},
		{
			name:   "only dirs considered",
			dirs:   []string{"20260604-150405"},
			files:  []string{"not-a-backup.txt"},
			wantID: "20260604-150405",
		},
		{
			name:   "single backup",
			dirs:   []string{"20260101-000000"},
			wantID: "20260101-000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backupsDir := setup(t, tt.dirs, tt.files)
			if tt.fixedDir == "missing" {
				backupsDir = filepath.Join(t.TempDir(), "definitely-not-real")
			}
			if tt.fixedDir == "" && len(tt.args) > 0 && tt.args[0] != "" {
				backupsDir = "/nonexistent"
			}
			id, err := actions.ResolveBackupID(backupsDir, tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error should mention %q, got: %v", tt.errContains, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tt.wantID {
				t.Errorf("got %q, want %q", id, tt.wantID)
			}
		})
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
	configtest.SetConfigHome(t, t.TempDir())
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
	configtest.SetConfigHome(t, t.TempDir())
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

func TestRunPush_EmptyProfile(t *testing.T) {
	deps, _, _ := setupTestDeps(t)

	cmd := findSubcommand(t, "push")
	if cmd == nil {
		t.Fatal("push command not found")
	}

	// Reset pushProfile flag after test.
	origProfile := pushProfile
	defer func() { pushProfile = origProfile }()

	pushProfile = ""
	err := runPushWithDeps(cmd, []string{"20250101-000000"}, deps)
	if err == nil {
		t.Fatal("expected error for empty profile, got nil")
	}
	if !strings.Contains(err.Error(), "profile") || !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention empty profile, got: %v", err)
	}
}
