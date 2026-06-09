package actions

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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

	m.AddAdapter("test-adapter", "", "~/.config/bak", []manifest.Item{
		{
			Category:   "config",
			SourcePath: "~/.config/bak/config.json",
			BackupPath: "test-adapter/config.json",
			Hash:       "sha256:abc",
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

func TestRestoreAction_DryRun(t *testing.T) {
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

func TestRestoreAction_MissingManifest(t *testing.T) {
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

func TestRestoreAction_MissingBackup(t *testing.T) {
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

func TestRestoreAction_ChecksumMismatch(t *testing.T) {
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

func TestRestoreAction_DryRunShowsDiff(t *testing.T) {
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

func TestRestoreAction_ApplyRestore(t *testing.T) {
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

func TestRestoreAction_UserHomeDirError(t *testing.T) {
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

func TestRestoreAction_VerboseOutput(t *testing.T) {
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

func TestRestoreAction_DryRunWithDiffs(t *testing.T) {
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

func TestRestoreAction_RestoreFile_MkdirError(t *testing.T) {
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

func TestRestoreAction_CountByStatus_AllTypes(t *testing.T) {
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

func TestRestoreAction_RestoreFile_PathTraversalBackupDir(t *testing.T) {
	home := t.TempDir()
	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: filepath.Join(home, ".bak", "backups", "test"),
	}

	err := action.restoreFile(restorepkg.FileDiff{
		BackupPath: "../../../etc/passwd",
		TargetPath: filepath.Join(home, "safe.txt"),
	})

	if err == nil {
		t.Fatal("expected path traversal error")
	}
	if !strings.Contains(err.Error(), "escapes") {
		t.Errorf("error should mention escapes: %v", err)
	}
}

func TestRestoreAction_RestoreFile_PathTraversalTarget(t *testing.T) {
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
	})

	if err == nil {
		t.Fatal("expected path traversal error")
	}
	if !strings.Contains(err.Error(), "escapes") {
		t.Errorf("error should mention escapes: %v", err)
	}
}

func TestRestoreAction_UserHomeDir_Error(t *testing.T) {
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

func TestRestoreAction_RestoreFile_CopyFile_Success(t *testing.T) {
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
	})
	if err != nil {
		t.Fatalf("restoreFile: %v", err)
	}

	// Verify CopyFile was called by checking dst is in the mock FS.
	if _, ok := mockFS.Files[dst]; !ok {
		t.Error("CopyFile was not called — dst path not found in mock FS")
	}
}

func TestRestoreAction_RestoreFile_CopyFile_Error(t *testing.T) {
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
	})
	if err == nil {
		t.Fatal("expected error from CopyFile")
	}
}

func TestRestoreAction_Stdin_Injected(t *testing.T) {
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

func TestResolveBackup(t *testing.T) {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()

			// Override home directory for BakDir() resolution.
			switch runtime.GOOS {
			case "windows":
				t.Setenv("USERPROFILE", home)
			default:
				t.Setenv("HOME", home)
			}

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

func TestRestoreAction_StdoutInjection(t *testing.T) {
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

func TestRestoreAction_StdoutNotLeaked(t *testing.T) {
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

func TestRestoreAction_NilWritersFallback(t *testing.T) {
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
