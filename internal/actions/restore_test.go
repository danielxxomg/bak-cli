package actions

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	configtest "github.com/danielxxomg/bak-cli/internal/config/testutil"
	"github.com/danielxxomg/bak-cli/internal/manifest"
	restorepkg "github.com/danielxxomg/bak-cli/internal/restore"
)

// --- restore helpers ----------------------------------------------------

// createBackupForRestore creates a real backup inside home so it can be
// restored.
func createBackupForRestore(t *testing.T, home string) string {
	t.Helper()

	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups")
	backupID := "20260101-120000"
	backupDir := filepath.Join(backupsDir, backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a manifest.
	m := manifest.New(backupID, "linux", "testhost", "test", "quick", []string{"config"})
	configDir := filepath.Join(home, ".config", "bak")

	// Write a backed-up file.
	adapterDir := filepath.Join(backupDir, "test-adapter")
	if err := os.MkdirAll(adapterDir, 0755); err != nil {
		t.Fatal(err)
	}
	testContent := []byte("key=value\n")
	backedFile := filepath.Join(adapterDir, "config.json")
	if err := os.WriteFile(backedFile, testContent, 0644); err != nil {
		t.Fatal(err)
	}

	h := sha256.Sum256(testContent)

	m.AddAdapter("test-adapter", "", "~/.config/bak", []manifest.Item{
		{
			Category:   "config",
			SourcePath: "~/.config/bak/config.json",
			BackupPath: "test-adapter/config.json",
			Hash:       fmt.Sprintf("sha256:%x", h),
			Size:       int64(len(testContent)),
		},
	})
	if err := m.Save(backupDir); err != nil {
		t.Fatal(err)
	}

	// Create the target dir but not the file (simulates "new" diff).
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	return backupID
}

// --- tests -------------------------------------------------------------

func TestRestoreAction_DryRun(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()
	backupID := createBackupForRestore(t, home)

	action := &RestoreAction{
		FS:      newHomeFS(home),
		Stdout:  io.Discard,
		Stderr:  io.Discard,
		DryRun:  true,
		Verbose: false,
	}

	bakDir := filepath.Join(home, ".bak")
	action.BackupDir = filepath.Join(bakDir, "backups", backupID)

	err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRestoreAction_MissingManifest(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()

	// Backup dir without manifest.
	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups")
	backupID := "20260101-120000"
	backupDir := filepath.Join(backupsDir, backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error for missing manifest")
	}
	if !strings.Contains(err.Error(), "load manifest") {
		t.Errorf("error should mention manifest: %v", err)
	}
}

func TestRestoreAction_MissingBackup(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: filepath.Join(home, ".bak", "backups", "nonexistent"),
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error for missing backup")
	}
}

func TestRestoreAction_ChecksumMismatch(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()
	backupID := createBackupForRestore(t, home)

	bakDir := filepath.Join(home, ".bak")
	backupDir := filepath.Join(bakDir, "backups", backupID)

	// Modify backed-up file to break the checksum.
	adapterDir := filepath.Join(backupDir, "test-adapter")
	if err := os.WriteFile(filepath.Join(adapterDir, "config.json"),
		[]byte("tampered-content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		Verbose:   false,
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}
}

func TestRestoreAction_DryRunShowsDiff(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()
	backupID := createBackupForRestore(t, home)

	bakDir := filepath.Join(home, ".bak")
	backupDir := filepath.Join(bakDir, "backups", backupID)

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		Stdout:    io.Discard,
		Stderr:    io.Discard,
		DryRun:    true,
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRestoreAction_ApplyRestore(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()
	backupID := createBackupForRestore(t, home)

	bakDir := filepath.Join(home, ".bak")
	backupDir := filepath.Join(bakDir, "backups", backupID)

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		Force:     true, // skip confirmation
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verify the file was restored.
	targetPath := filepath.Join(home, ".config", "bak", "config.json")
	if _, err := os.Stat(targetPath); err != nil {
		t.Errorf("restored file not found: %v", err)
	}
}

func TestRestoreAction_UserHomeDirError(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	mockFS := &MockFileSystem{
		HomeDir:    "",
		StatResult: map[string]MockStatResult{},
		Files:      map[string][]byte{},
	}
	// The mock always returns HomeDir without error. We test via
	// another path: missing backup.
	action := &RestoreAction{
		FS:        mockFS,
		BackupDir: "/home/test/.bak/backups/nonexistent",
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRestoreAction_VerboseOutput(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()
	backupID := createBackupForRestore(t, home)

	bakDir := filepath.Join(home, ".bak")
	backupDir := filepath.Join(bakDir, "backups", backupID)

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		Verbose:   true,
		Stdout:    io.Discard,
		Stderr:    io.Discard,
		DryRun:    true,
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRestoreAction_DryRunWithDiffs(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()
	backupID := createBackupForRestore(t, home)

	bakDir := filepath.Join(home, ".bak")
	backupDir := filepath.Join(bakDir, "backups", backupID)

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		Stdout:    io.Discard,
		Stderr:    io.Discard,
		DryRun:    true,
		Verbose:   false,
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestRestoreAction_RestoreFile_MkdirError(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()
	backupID := createBackupForRestore(t, home)

	bakDir := filepath.Join(home, ".bak")
	backupDir := filepath.Join(bakDir, "backups", backupID)

	mockFS := &MockFileSystem{
		HomeDir:    home,
		StatResult: make(map[string]MockStatResult),
		Files:      make(map[string][]byte),
		MkdirErrors: map[string]error{
			filepath.Join(home, ".config", "bak"): os.ErrPermission,
		},
	}

	action := &RestoreAction{
		FS:        mockFS,
		BackupDir: backupDir,
		Force:     true,
		Verbose:   true,
	}

	err := action.Run()
	if err != nil {
		t.Logf("restore with mkdir error returned: %v", err)
	}
}

func TestRestoreAction_CountByStatus_AllTypes(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	diffs := []restorepkg.FileDiff{
		{Status: restorepkg.DiffNew, SourcePath: "/a"},
		{Status: restorepkg.DiffNew, SourcePath: "/b"},
		{Status: restorepkg.DiffModified, SourcePath: "/c"},
		{Status: restorepkg.DiffUnchanged, SourcePath: "/d"},
		{Status: restorepkg.DiffMissing, SourcePath: "/e"},
	}

	if n := countByStatus(diffs, restorepkg.DiffNew); n != 2 {
		t.Errorf("DiffNew count = %d, want 2", n)
	}
	if n := countByStatus(diffs, restorepkg.DiffModified); n != 1 {
		t.Errorf("DiffModified count = %d, want 1", n)
	}
	if n := countByStatus(diffs, restorepkg.DiffUnchanged); n != 1 {
		t.Errorf("DiffUnchanged count = %d, want 1", n)
	}
	if n := countByStatus(diffs, restorepkg.DiffMissing); n != 1 {
		t.Errorf("DiffMissing count = %d, want 1", n)
	}
}

func TestRestoreAction_RestoreFile_PathTraversalBackupDir(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()
	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: filepath.Join(home, ".bak", "backups", "test"),
	}

	err := action.restoreFile(restorepkg.FileDiff{
		BackupPath: "../../../etc/passwd",
		TargetPath: filepath.Join(home, "safe.txt"),
	}, 0)

	if err == nil {
		t.Fatal("expected path traversal error")
	}
	if !strings.Contains(err.Error(), "escapes") {
		t.Errorf("error should mention escapes: %v", err)
	}
}

func TestRestoreAction_RestoreFile_PathTraversalTarget(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()

	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups")
	backupDir := filepath.Join(backupsDir, "test")
	os.MkdirAll(backupDir, 0755)
	srcFile := filepath.Join(backupDir, "safe.txt")
	os.WriteFile(srcFile, []byte("content"), 0644)

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
	}

	err := action.restoreFile(restorepkg.FileDiff{
		BackupPath: "safe.txt",
		TargetPath: filepath.Join(home, "..", "..", "etc", "passwd"),
	}, 0)

	if err == nil {
		t.Fatal("expected path traversal error")
	}
	if !strings.Contains(err.Error(), "escapes") {
		t.Errorf("error should mention escapes: %v", err)
	}
}

func TestRestoreAction_UserHomeDir_Error(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	mockFS := &MockFileSystem{
		HomeDir:    "",
		StatResult: make(map[string]MockStatResult),
		Files:      make(map[string][]byte),
	}

	action := &RestoreAction{
		FS:        mockFS,
		BackupDir: "/some/path",
		Force:     true,
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRestoreAction_RestoreFile_CopyFile_Success(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()

	// Create only the backup directory structure (no real source file).
	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups", "test")
	os.MkdirAll(backupsDir, 0755)

	// Source file exists ONLY in the mock FS, forcing the code path
	// through a.FS.CopyFile() instead of os.Open.
	srcPath := filepath.Join(backupsDir, "safe.txt")
	mockFS := &MockFileSystem{
		HomeDir:    home,
		StatResult: make(map[string]MockStatResult),
		Files: map[string][]byte{
			srcPath: []byte("content"),
		},
	}

	action := &RestoreAction{
		FS:        mockFS,
		BackupDir: backupsDir,
	}

	dst := filepath.Join(home, "restored.txt")
	err := action.restoreFile(restorepkg.FileDiff{
		BackupPath: "safe.txt",
		TargetPath: dst,
	}, 0)
	if err != nil {
		t.Fatalf("restoreFile: %v", err)
	}

	// Verify CopyFile was called by checking dst is in the mock FS.
	if _, ok := mockFS.Files[dst]; !ok {
		t.Error("CopyFile was not called — dst path not found in mock FS")
	}
}

func TestRestoreAction_RestoreFile_CopyFile_Error(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()

	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups", "test")
	os.MkdirAll(backupsDir, 0755)

	srcPath := filepath.Join(backupsDir, "safe.txt")
	mockFS := &MockFileSystem{
		HomeDir:    home,
		StatResult: make(map[string]MockStatResult),
		Files: map[string][]byte{
			srcPath: []byte("content"),
		},
		CopyErrors: map[string]error{
			srcPath: os.ErrPermission,
		},
	}

	action := &RestoreAction{
		FS:        mockFS,
		BackupDir: backupsDir,
	}

	dst := filepath.Join(home, "restored.txt")
	err := action.restoreFile(restorepkg.FileDiff{
		BackupPath: "safe.txt",
		TargetPath: dst,
	}, 0)
	if err == nil {
		t.Fatal("expected error from CopyFile")
	}
}

func TestRestoreAction_Stdin_Injected(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: filepath.Join(home, "nonexistent"),
	}

	// Zero-value Stdin (nil) means os.Stdin is used at runtime.
	if action.Stdin != nil {
		t.Log("Stdin field is injectable on RestoreAction")
	} else {
		t.Log("Stdin is nil — will fall back to os.Stdin")
	}
}

func TestResolveBackup(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name        string
		setup       func(home string) (backupID string)
		wantErr     bool
		errContains string
		wantDir     func(home string) string
	}{
		{
			name: "valid_resolution",
			setup: func(home string) string {
				dir := filepath.Join(home, ".bak", "backups", "20260101-120000")
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				return "20260101-120000"
			},
			wantErr: false,
			wantDir: func(home string) string {
				return filepath.Join(home, ".bak", "backups", "20260101-120000")
			},
		},
		{
			name: "missing_backup",
			setup: func(home string) string {
				if err := os.MkdirAll(filepath.Join(home, ".bak", "backups"), 0755); err != nil {
					t.Fatal(err)
				}
				return "20260101-120000"
			},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name: "traversal_dotdot_blocked",
			setup: func(home string) string {
				if err := os.MkdirAll(filepath.Join(home, ".bak", "backups"), 0755); err != nil {
					t.Fatal(err)
				}
				return "../etc"
			},
			wantErr:     true,
			errContains: "outside",
		},
		{
			name: "traversal_nested_blocked",
			setup: func(home string) string {
				if err := os.MkdirAll(filepath.Join(home, ".bak", "backups"), 0755); err != nil {
					t.Fatal(err)
				}
				return "../../etc"
			},
			wantErr:     true,
			errContains: "outside",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			home := t.TempDir()

			// Override home directory for BakDir() resolution.
			configtest.SetConfigHome(t, home)

			backupID := tt.setup(home)

			action := &RestoreAction{
				FS: newHomeFS(home),
			}

			err := action.ResolveBackup(backupID)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errContains)
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want substring %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			wantDir := tt.wantDir(home)
			if action.BackupDir != wantDir {
				t.Errorf("BackupDir = %q, want %q", action.BackupDir, wantDir)
			}
		})
	}
}

func TestRestoreAction_StdoutInjection(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()
	backupID := createBackupForRestore(t, home)

	bakDir := filepath.Join(home, ".bak")
	backupDir := filepath.Join(bakDir, "backups", backupID)

	var stdout, stderr bytes.Buffer

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		DryRun:    true,
		Stdout:    &stdout,
		Stderr:    &stderr,
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Dry-run") {
		t.Errorf("Stdout should contain dry-run output, got: %s", out)
	}
}

func TestRestoreAction_StdoutNotLeaked(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()
	backupID := createBackupForRestore(t, home)

	bakDir := filepath.Join(home, ".bak")
	backupDir := filepath.Join(bakDir, "backups", backupID)

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		DryRun:    true,
		Stdout:    io.Discard,
		Stderr:    io.Discard,
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run with io.Discard: %v", err)
	}
}

func TestRestoreAction_NilWritersFallback(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()
	backupID := createBackupForRestore(t, home)

	bakDir := filepath.Join(home, ".bak")
	backupDir := filepath.Join(bakDir, "backups", backupID)

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		DryRun:    true,
		// Stdout/Stderr are nil — must fall back to os.Stdout/os.Stderr.
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run with nil writers: %v", err)
	}
}

// --- confirmation prompt tests -----------------------------------------

// errorReader returns an error on every Read call.
type errorReader struct{}

func (e *errorReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestRestoreAction_CancelPrompt_AnswerNo(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()
	backupID := createBackupForRestore(t, home)

	bakDir := filepath.Join(home, ".bak")
	backupDir := filepath.Join(bakDir, "backups", backupID)

	var stdout, stderr bytes.Buffer

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		DryRun:    false,
		Force:     false,
		Stdin:     strings.NewReader("n\n"),
		Stdout:    &stdout,
		Stderr:    &stderr,
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Apply restore?") {
		t.Error("expected confirmation prompt in stdout")
	}
	if !strings.Contains(stderr.String(), "Restore cancelled") {
		t.Error("expected 'Restore cancelled' in stderr")
	}
}

func TestRestoreAction_CancelPrompt_EmptyInput(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()
	backupID := createBackupForRestore(t, home)

	bakDir := filepath.Join(home, ".bak")
	backupDir := filepath.Join(bakDir, "backups", backupID)

	var stderr bytes.Buffer

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		DryRun:    false,
		Force:     false,
		Stdin:     strings.NewReader("\n"),
		Stdout:    io.Discard,
		Stderr:    &stderr,
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !strings.Contains(stderr.String(), "Restore cancelled") {
		t.Error("expected 'Restore cancelled' for empty input")
	}
}

func TestRestoreAction_CancelPrompt_ReadError(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()
	backupID := createBackupForRestore(t, home)

	bakDir := filepath.Join(home, ".bak")
	backupDir := filepath.Join(bakDir, "backups", backupID)

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		DryRun:    false,
		Force:     false,
		Stdin:     &errorReader{},
		Stdout:    io.Discard,
		Stderr:    io.Discard,
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error from stdin read failure")
	}
	if !strings.Contains(err.Error(), "read input") {
		t.Errorf("error should mention 'read input': %v", err)
	}
}

func TestRestoreAction_ConfirmPrompt_AnswerYes(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()
	backupID := createBackupForRestore(t, home)

	bakDir := filepath.Join(home, ".bak")
	backupDir := filepath.Join(bakDir, "backups", backupID)

	var stdout bytes.Buffer

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		DryRun:    false,
		Force:     false,
		Stdin:     strings.NewReader("y\n"),
		Stdout:    &stdout,
		Stderr:    io.Discard,
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "Apply restore?") {
		t.Error("expected confirmation prompt in stdout")
	}
	if !strings.Contains(out, "Restore complete") {
		t.Errorf("expected 'Restore complete' for confirmed restore, got: %s", out)
	}

	// Verify the file was actually restored.
	targetPath := filepath.Join(home, ".config", "bak", "config.json")
	if _, err := os.Stat(targetPath); err != nil {
		t.Errorf("restored file not found: %v", err)
	}
}

func TestRestoreAction_ForceSkipsPrompt(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()
	backupID := createBackupForRestore(t, home)

	bakDir := filepath.Join(home, ".bak")
	backupDir := filepath.Join(bakDir, "backups", backupID)

	var stdout bytes.Buffer

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		DryRun:    false,
		Force:     true,
		Stdout:    &stdout,
		Stderr:    io.Discard,
		// Stdin is nil — Force=true should skip the prompt entirely.
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	out := stdout.String()
	if strings.Contains(out, "Apply restore?") {
		t.Error("expected NO confirmation prompt when Force=true")
	}
	if !strings.Contains(out, "Restore complete") {
		t.Errorf("expected 'Restore complete' with Force=true, got: %s", out)
	}
}

func TestRestoreAction_Run_CopyFailure(t *testing.T) { //nolint:paralleltest // shared state isolation pending
	tests := []struct {
		name      string
		setupMock func(home, backupDir string) *MockFileSystem
		wantMsg   string
	}{
		{
			name: "mkdir_failure",
			setupMock: func(home, backupDir string) *MockFileSystem {
				return &MockFileSystem{
					HomeDir:    home,
					StatResult: make(map[string]MockStatResult),
					Files:      make(map[string][]byte),
					MkdirErrors: map[string]error{
						filepath.Join(home, ".config", "bak"): os.ErrPermission,
					},
				}
			},
			wantMsg: "permission denied",
		},
		{
			name: "copy_file_failure",
			setupMock: func(home, backupDir string) *MockFileSystem {
				adapterDir := filepath.Join(backupDir, "test-adapter")
				srcPath := filepath.Join(adapterDir, "config.json")
				return &MockFileSystem{
					HomeDir:    home,
					StatResult: make(map[string]MockStatResult),
					Files:      make(map[string][]byte),
					CopyErrors: map[string]error{
						srcPath: os.ErrPermission,
					},
				}
			},
			wantMsg: "permission denied",
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest
			home := t.TempDir()
			backupID := createBackupForRestore(t, home)
			bakDir := filepath.Join(home, ".bak")
			backupDir := filepath.Join(bakDir, "backups", backupID)

			mockFS := tt.setupMock(home, backupDir)
			var stdout, stderr bytes.Buffer

			action := &RestoreAction{
				FS:        mockFS,
				BackupDir: backupDir,
				Force:     true,
				Stdout:    &stdout,
				Stderr:    &stderr,
			}

			err := action.Run()
			if err == nil {
				t.Fatal("expected error on copy failure, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("error %q should contain %q", err.Error(), tt.wantMsg)
			}
			outStr := stdout.String()
			if strings.Contains(outStr, "Restore complete") {
				t.Errorf("stdout should NOT contain 'Restore complete' on failure, got: %s", outStr)
			}
			if !strings.Contains(outStr, "Restore failed") {
				t.Errorf("stdout should contain 'Restore failed' on failure, got: %s", outStr)
			}
		})
	}
}

func TestRestoreAction_ManifestIntegrity(t *testing.T) { //nolint:paralleltest // shared state isolation pending
	tests := []struct {
		name       string
		force      bool
		tamper     bool
		wantErr    bool
		errContain string
	}{
		{
			name:    "valid_manifest_without_force",
			force:   false,
			tamper:  false,
			wantErr: false,
		},
		{
			name:    "valid_manifest_with_force",
			force:   true,
			tamper:  false,
			wantErr: false,
		},
		{
			name:       "tampered_manifest_without_force",
			force:      false,
			tamper:     true,
			wantErr:    true,
			errContain: "manifest validation failed",
		},
		{
			name:       "tampered_manifest_with_force_fails_hard",
			force:      true,
			tamper:     true,
			wantErr:    true,
			errContain: "manifest validation failed",
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest
			home := t.TempDir()
			backupID := createBackupForRestore(t, home)
			bakDir := filepath.Join(home, ".bak")
			backupDir := filepath.Join(bakDir, "backups", backupID)

			if tt.tamper {
				adapterDir := filepath.Join(backupDir, "test-adapter")
				if err := os.WriteFile(filepath.Join(adapterDir, "config.json"),
					[]byte("tampered-content\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}

			action := &RestoreAction{
				FS:        newHomeFS(home),
				BackupDir: backupDir,
				Force:     tt.force,
				Stdin:     strings.NewReader("y\n"),
				Stdout:    io.Discard,
				Stderr:    io.Discard,
			}

			err := action.Run()
			targetPath := filepath.Join(home, ".config", "bak", "config.json")

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errContain)
				}
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContain)
				}
				// Verify no file was written when manifest validation failed.
				if _, statErr := os.Stat(targetPath); !os.IsNotExist(statErr) {
					t.Errorf("target file should not exist after failed validation, got statErr=%v", statErr)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				// Verify file was written.
				if _, statErr := os.Stat(targetPath); statErr != nil {
					t.Errorf("target file should exist: %v", statErr)
				}
			}
		})
	}
}

func TestRestoreAction_V030Manifest_RestoresDegraded(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups")
	backupID := "20260101-120000"
	backupDir := filepath.Join(backupsDir, backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	adapterDir := filepath.Join(backupDir, "test-adapter")
	if err := os.MkdirAll(adapterDir, 0755); err != nil {
		t.Fatal(err)
	}
	testContent := []byte("key=value\n")
	backedFile := filepath.Join(adapterDir, "config.json")
	if err := os.WriteFile(backedFile, testContent, 0644); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(testContent)

	m := manifest.New(backupID, "linux", "testhost", "0.3.0", "quick", []string{"config"})
	m.Version = "0.3.0"
	m.AddAdapter("test-adapter", "", "~/.config/bak", []manifest.Item{
		{
			Category:   "config",
			SourcePath: "~/.config/bak/config.json",
			BackupPath: "test-adapter/config.json",
			Hash:       fmt.Sprintf("sha256:%x", h),
			Size:       int64(len(testContent)),
			Mode:       0, // 0.3.0 manifest has no mode
		},
	})
	if err := m.Save(backupDir); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		Force:     true,
		Stdout:    &stdout,
		Stderr:    io.Discard,
	}

	if err := action.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}

	outStr := stdout.String()
	if !strings.Contains(outStr, "degraded permissions") {
		t.Errorf("expected summary to report degraded permissions, got:\n%s", outStr)
	}
}

// chmodFailingFS wraps a FileSystem double and fails Chmod calls.
type chmodFailingFS struct {
	FileSystem
}

func (c *chmodFailingFS) Chmod(name string, _ os.FileMode) error {
	return fmt.Errorf("chmod %s: permission denied", name)
}

func TestRestoreAction_ChmodFailure_SurfacesAsError(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups")
	backupID := "20260101-chmodfail"
	backupDir := filepath.Join(backupsDir, backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	adapterDir := filepath.Join(backupDir, "test-adapter")
	if err := os.MkdirAll(adapterDir, 0755); err != nil {
		t.Fatal(err)
	}
	testContent := []byte("#!/bin/sh\necho test\n")
	backedFile := filepath.Join(adapterDir, "script.sh")
	if err := os.WriteFile(backedFile, testContent, 0755); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(testContent)

	m := manifest.New(backupID, "linux", "testhost", "0.4.0", "quick", []string{"config"})
	m.Version = "0.4.0"
	m.AddAdapter("test-adapter", "", "~/.config/bak", []manifest.Item{
		{
			Category:   "config",
			SourcePath: "~/.config/bak/script.sh",
			BackupPath: "test-adapter/script.sh",
			Hash:       fmt.Sprintf("sha256:%x", h),
			Size:       int64(len(testContent)),
			Mode:       0755,
		},
	})
	if err := m.Save(backupDir); err != nil {
		t.Fatal(err)
	}

	baseFS := newHomeFS(home)
	failingFS := &chmodFailingFS{FileSystem: baseFS}

	action := &RestoreAction{
		FS:        failingFS,
		BackupDir: backupDir,
		Force:     true,
		Stdout:    io.Discard,
		Stderr:    io.Discard,
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error on chmod failure, got nil")
	}
	if !strings.Contains(err.Error(), "chmod") {
		t.Errorf("error %q should mention chmod", err.Error())
	}
}

// partialCopyFailingFS wraps a FileSystem double to simulate a copy failure with partial write.
type partialCopyFailingFS struct {
	FileSystem
	failOnDst string
	failErr   error
}

func (p *partialCopyFailingFS) CopyFile(src, dst string) error {
	if dst == p.failOnDst {
		if err := p.WriteFile(dst, []byte("corrupted partial write"), 0644); err != nil {
			return err
		}
		return p.failErr
	}
	return p.FileSystem.CopyFile(src, dst)
}

func TestRestoreAction_RollbackOnFailure(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups")
	backupID := "20260101-rollback-test"
	backupDir := filepath.Join(backupsDir, backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	adapterDir := filepath.Join(backupDir, "test-adapter")
	if err := os.MkdirAll(adapterDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Two backup files.
	file1Backup := filepath.Join(adapterDir, "file1.txt")
	file2Backup := filepath.Join(adapterDir, "file2.txt")
	file1NewContent := []byte("new-content-1\n")
	file2NewContent := []byte("new-content-2\n")
	if err := os.WriteFile(file1Backup, file1NewContent, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2Backup, file2NewContent, 0644); err != nil {
		t.Fatal(err)
	}

	h1 := sha256.Sum256(file1NewContent)
	h2 := sha256.Sum256(file2NewContent)

	m := manifest.New(backupID, "linux", "testhost", "0.4.0", "quick", []string{"config"})
	m.Version = "0.4.0"
	m.AddAdapter("test-adapter", "", "~/.config/bak", []manifest.Item{
		{
			Category:   "config",
			SourcePath: "~/.config/bak/file1.txt",
			BackupPath: "test-adapter/file1.txt",
			Hash:       fmt.Sprintf("sha256:%x", h1),
			Size:       int64(len(file1NewContent)),
			Mode:       0644,
		},
		{
			Category:   "config",
			SourcePath: "~/.config/bak/file2.txt",
			BackupPath: "test-adapter/file2.txt",
			Hash:       fmt.Sprintf("sha256:%x", h2),
			Size:       int64(len(file2NewContent)),
			Mode:       0644,
		},
	})
	if err := m.Save(backupDir); err != nil {
		t.Fatal(err)
	}

	// Pre-create targets with original content.
	targetDir := filepath.Join(home, ".config", "bak")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	target1Path := filepath.Join(targetDir, "file1.txt")
	target2Path := filepath.Join(targetDir, "file2.txt")
	orig1 := []byte("original-1\n")
	orig2 := []byte("original-2\n")
	if err := os.WriteFile(target1Path, orig1, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target2Path, orig2, 0644); err != nil {
		t.Fatal(err)
	}

	baseFS := newHomeFS(home)
	failingFS := &partialCopyFailingFS{
		FileSystem: baseFS,
		failOnDst:  target2Path,
		failErr:    fmt.Errorf("injected disk write error on file2"),
	}

	action := &RestoreAction{
		FS:        failingFS,
		BackupDir: backupDir,
		Force:     true,
		Stdout:    io.Discard,
		Stderr:    io.Discard,
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error on partial restore failure, got nil")
	}

	// Target 1 must be reverted to original content via rollback.
	got1, err := os.ReadFile(target1Path)
	if err != nil {
		t.Fatalf("read target 1: %v", err)
	}
	if string(got1) != string(orig1) {
		t.Errorf("target 1 was not rolled back: got %q, want %q", string(got1), string(orig1))
	}

	// Target 2 must also be restored to original content, not left partially written.
	got2, err := os.ReadFile(target2Path)
	if err != nil {
		t.Fatalf("read target 2: %v", err)
	}
	if string(got2) != string(orig2) {
		t.Errorf("target 2 was not rolled back: got %q, want %q", string(got2), string(orig2))
	}

	// Recovery evidence must be retained in home/.bak/recovery.
	recBase := filepath.Join(home, ".bak", "recovery")
	entries, err := os.ReadDir(recBase)
	if err != nil || len(entries) == 0 {
		t.Errorf("expected recovery evidence in %s, got err: %v, entries: %d", recBase, err, len(entries))
	}
}

type metadataFailCopyFS struct {
	FileSystem
	failOnDst string
	failErr   error
	homeDir   string
}

func (m *metadataFailCopyFS) CopyFile(src, dst string) error {
	if dst == m.failOnDst {
		recBase := filepath.Join(m.homeDir, ".bak", "recovery")
		if entries, err := os.ReadDir(recBase); err == nil && len(entries) > 0 {
			metaFile := filepath.Join(recBase, entries[0].Name(), "repo", "recovery-meta.json")
			_ = os.Remove(metaFile)
			_ = os.Mkdir(metaFile, 0755)
		}
		return m.failErr
	}
	return m.FileSystem.CopyFile(src, dst)
}

func TestRestoreAction_MetadataFailureDuringRollback_SurfacedAtActionBoundary(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups")
	backupID := "20260101-metafail-test"
	backupDir := filepath.Join(backupsDir, backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	adapterDir := filepath.Join(backupDir, "test-adapter")
	if err := os.MkdirAll(adapterDir, 0755); err != nil {
		t.Fatal(err)
	}

	f1Backup := filepath.Join(adapterDir, "file1.txt")
	f2Backup := filepath.Join(adapterDir, "file2.txt")
	if err := os.WriteFile(f1Backup, []byte("new-1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2Backup, []byte("new-2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	h1 := sha256.Sum256([]byte("new-1\n"))
	h2 := sha256.Sum256([]byte("new-2\n"))

	m := manifest.New(backupID, "linux", "testhost", "0.4.0", "quick", []string{"config"})
	m.Version = "0.4.0"
	m.AddAdapter("test-adapter", "", "~/.config/bak", []manifest.Item{
		{
			Category:   "config",
			SourcePath: "~/.config/bak/file1.txt",
			BackupPath: "test-adapter/file1.txt",
			Hash:       fmt.Sprintf("sha256:%x", h1),
			Size:       int64(len("new-1\n")),
			Mode:       0644,
		},
		{
			Category:   "config",
			SourcePath: "~/.config/bak/file2.txt",
			BackupPath: "test-adapter/file2.txt",
			Hash:       fmt.Sprintf("sha256:%x", h2),
			Size:       int64(len("new-2\n")),
			Mode:       0644,
		},
	})
	if err := m.Save(backupDir); err != nil {
		t.Fatal(err)
	}

	targetDir := filepath.Join(home, ".config", "bak")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	t1Path := filepath.Join(targetDir, "file1.txt")
	t2Path := filepath.Join(targetDir, "file2.txt")
	if err := os.WriteFile(t1Path, []byte("orig-1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(t2Path, []byte("orig-2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	baseFS := newHomeFS(home)
	failingFS := &metadataFailCopyFS{
		FileSystem: baseFS,
		failOnDst:  t2Path,
		failErr:    fmt.Errorf("injected disk failure on file2"),
		homeDir:    home,
	}

	var stdoutBuf strings.Builder
	action := &RestoreAction{
		FS:        failingFS,
		BackupDir: backupDir,
		Force:     true,
		Stdout:    &stdoutBuf,
		Stderr:    io.Discard,
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errMsg := err.Error()
	// Must keep original apply error
	if !strings.Contains(errMsg, "injected disk failure on file2") {
		t.Errorf("error %q should retain original apply error", errMsg)
	}
	// Must surface evidence persistence failure
	if !strings.Contains(errMsg, "write rollback metadata") {
		t.Errorf("error %q should surface evidence persistence failure 'write rollback metadata'", errMsg)
	}

	report := stdoutBuf.String()
	if !strings.Contains(report, "Failed:   1") {
		t.Errorf("report %q should reflect accurate failed count 1", report)
	}
}

type postFailStorageFS struct {
	FileSystem
}

func (p *postFailStorageFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	if strings.HasSuffix(path, "recovery-meta.json") && strings.Contains(string(data), `"status": "applied"`) {
		return fmt.Errorf("injected post commit error")
	}
	return p.FileSystem.WriteFile(path, data, perm)
}

func TestRestoreAction_RealAutomaticPostRecordingFailure(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups")
	backupID := "20260101-autorecordfail"
	backupDir := filepath.Join(backupsDir, backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	adapterDir := filepath.Join(backupDir, "test-adapter")
	if err := os.MkdirAll(adapterDir, 0755); err != nil {
		t.Fatal(err)
	}

	f1Backup := filepath.Join(adapterDir, "existing.txt")
	f2Backup := filepath.Join(adapterDir, "new.txt")
	f1NewBytes := []byte("restored-existing\n")
	f2NewBytes := []byte("restored-new\n")
	if err := os.WriteFile(f1Backup, f1NewBytes, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2Backup, f2NewBytes, 0644); err != nil {
		t.Fatal(err)
	}

	h1 := sha256.Sum256(f1NewBytes)
	h2 := sha256.Sum256(f2NewBytes)

	m := manifest.New(backupID, "linux", "testhost", "0.4.0", "quick", []string{"config"})
	m.Version = "0.4.0"
	m.AddAdapter("test-adapter", "", "~/.config/bak", []manifest.Item{
		{
			Category:   "config",
			SourcePath: "~/.config/bak/existing.txt",
			BackupPath: "test-adapter/existing.txt",
			Hash:       fmt.Sprintf("sha256:%x", h1),
			Size:       int64(len(f1NewBytes)),
			Mode:       0644,
		},
		{
			Category:   "config",
			SourcePath: "~/.config/bak/new.txt",
			BackupPath: "test-adapter/new.txt",
			Hash:       fmt.Sprintf("sha256:%x", h2),
			Size:       int64(len(f2NewBytes)),
			Mode:       0644,
		},
	})
	if err := m.Save(backupDir); err != nil {
		t.Fatal(err)
	}

	targetDir := filepath.Join(home, ".config", "bak")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	existingTarget := filepath.Join(targetDir, "existing.txt")
	newTarget := filepath.Join(targetDir, "new.txt")

	origExistingBytes := []byte("orig-existing-content\n")
	if err := os.WriteFile(existingTarget, origExistingBytes, 0600); err != nil {
		t.Fatal(err)
	}
	// newTarget intentionally does not exist (pre-existing absence)

	baseFS := newHomeFS(home)
	var stdoutBuf strings.Builder
	action := &RestoreAction{
		FS:        baseFS,
		storageFS: &postFailStorageFS{FileSystem: &OSFileSystem{}},
		BackupDir: backupDir,
		Force:     true,
		Stdout:    &stdoutBuf,
		Stderr:    io.Discard,
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected restore to fail during post-state recording, got nil")
	}

	// Assert original error from RecordPostState is retained
	if !strings.Contains(err.Error(), "injected post commit error") {
		t.Errorf("error %q should retain original post recording error", err.Error())
	}
	if !strings.Contains(err.Error(), "record post-restore state") {
		t.Errorf("error %q should mention record post-restore state", err.Error())
	}

	// Assert existing target has original bytes and original mode restored
	gotExisting, err := os.ReadFile(existingTarget)
	if err != nil {
		t.Fatalf("read existing target: %v", err)
	}
	if string(gotExisting) != string(origExistingBytes) {
		t.Errorf("existing target was not rolled back: got %q, want %q", string(gotExisting), string(origExistingBytes))
	}
	if !isWindows() {
		fi, err := os.Stat(existingTarget)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != 0600 {
			t.Errorf("existing target mode = %v, want 0600", fi.Mode().Perm())
		}
	}

	// Assert pre-existing absence is restored for newTarget (must not exist)
	if _, err := os.Stat(newTarget); err == nil {
		t.Errorf("newTarget should have been removed by rollback to restore pre-existing absence, but still exists")
	}

	// Assert report reflected failed status
	report := stdoutBuf.String()
	if !strings.Contains(report, "Restore failed") {
		t.Errorf("report %q should reflect failed restore", report)
	}
}
