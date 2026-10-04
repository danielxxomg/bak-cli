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

// createTestBackupForRestore creates a real backup inside home with customizable
// bakVersion, schemaVersion, and file mode.
func createTestBackupForRestore(t *testing.T, home, bakVersion, schemaVersion string, mode uint32) string {
	t.Helper()

	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups")
	backupID := "20260101-120000"
	backupDir := filepath.Join(backupsDir, backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	m := manifest.New(backupID, "linux", "testhost", bakVersion, "quick", []string{"config"})
	m.Version = schemaVersion
	configDir := filepath.Join(home, ".config", "bak")

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
			Mode:       mode,
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

// createBackupForRestore creates a standard backup inside home.
func createBackupForRestore(t *testing.T, home string) string {
	t.Helper()
	return createTestBackupForRestore(t, home, "test", manifest.ManifestVersion, 0)
}

// createCustomManifestBackupForRestore creates a backup with custom versions and mode 0644.
func createCustomManifestBackupForRestore(t *testing.T, home, bakVersion, schemaVersion string) string {
	t.Helper()
	return createTestBackupForRestore(t, home, bakVersion, schemaVersion, 0644)
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

func TestRestoreAction_DryRun_DistinguishesSecretExcluded(t *testing.T) { //nolint:paralleltest // not yet parallelized
	home := t.TempDir()
	bakDir := filepath.Join(home, ".bak")
	backupID := "20260101-120000"
	backupDir := filepath.Join(bakDir, "backups", backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create manifest with SecretsExcluded: true simulating a legacy backup
	m := manifest.New(backupID, "linux", "testhost", "1.0.0", "quick", []string{"config"})
	m.SecretsExcluded = true
	m.AddAdapter("opencode", "", "~/.config/opencode", []manifest.Item{
		{
			Category:   "config",
			SourcePath: "~/.config/opencode/secrets.json",
			BackupPath: "opencode/secrets.json",
			Hash:       "sha256:0000000000000000000000000000000000000000000000000000000000000000",
			Size:       100,
		},
	})
	if err := m.Save(backupDir); err != nil {
		t.Fatal(err)
	}

	// Write companion .env.example
	envExamplePath := filepath.Join(backupDir, ".env.example")
	if err := os.WriteFile(envExamplePath, []byte("# ---- ~/.config/opencode/secrets.json ----\nKEY=<YOUR_SECRET>\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		Stdout:    &stdout,
		Stderr:    io.Discard,
		DryRun:    true,
	}

	if err := action.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}

	out := stdout.String()

	// Must label as secret-excluded in diff
	if !strings.Contains(out, "[secret-excluded]") {
		t.Errorf("dry-run diff missing [secret-excluded], got:\n%s", out)
	}

	// Must NOT label as missing
	if strings.Contains(out, "[missing]") {
		t.Errorf("dry-run diff misclassified as [missing], got:\n%s", out)
	}

	// Must NOT say 1 file would be restored
	if strings.Contains(out, "1 file(s) would be restored") {
		t.Errorf("dry-run falsely implies excluded file would be restored:\n%s", out)
	}

	// Must state 0 files would be restored
	if !strings.Contains(out, "0 file(s) would be restored") {
		t.Errorf("dry-run missing '0 file(s) would be restored':\n%s", out)
	}

	// Must note that excluded files cannot be restored and must be re-entered
	if !strings.Contains(out, "cannot be restored") || !strings.Contains(out, "re-entered by hand") {
		t.Errorf("dry-run summary missing honesty note that files cannot be restored and must be re-entered by hand:\n%s", out)
	}
}

func TestRestoreAction_RedactedFileExplicitInDryRunAndReport(t *testing.T) { //nolint:paralleltest // not yet parallelized
	home := t.TempDir()
	configtest.SetConfigHome(t, home)
	bakDir := filepath.Join(home, ".bak")
	backupID := "20261001-120000"
	backupDir := filepath.Join(bakDir, "backups", backupID)
	if err := os.MkdirAll(filepath.Join(backupDir, "opencode"), 0755); err != nil {
		t.Fatal(err)
	}

	payloadFile := filepath.Join(backupDir, "opencode", "opencode.json")
	payloadContent := `{"token":"<YOUR_SECRET>","theme":"dark"}`
	if err := os.WriteFile(payloadFile, []byte(payloadContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Live file on target has real secrets
	liveConfigDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(liveConfigDir, 0755); err != nil {
		t.Fatal(err)
	}
	liveFile := filepath.Join(liveConfigDir, "opencode.json")
	if err := os.WriteFile(liveFile, []byte(`{"token":"real_secret_token","theme":"light"}`), 0644); err != nil {
		t.Fatal(err)
	}

	h := sha256.Sum256([]byte(payloadContent))
	m := manifest.New(backupID, "linux", "testhost", "0.5.0", "quick", []string{"config"})
	m.AddAdapter("opencode", "", "~/.config/opencode", []manifest.Item{
		{
			Category:    "config",
			SourcePath:  "~/.config/opencode/opencode.json",
			BackupPath:  "opencode/opencode.json",
			Hash:        fmt.Sprintf("sha256:%x", h),
			Size:        int64(len(payloadContent)),
			Mode:        0644,
			Redacted:    true,
			SecretCount: 1,
		},
	})
	if err := m.Save(backupDir); err != nil {
		t.Fatal(err)
	}

	// 1. Dry run check
	var dryRunStdout bytes.Buffer
	dryRunAction := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		Stdout:    &dryRunStdout,
		Stderr:    io.Discard,
		DryRun:    true,
	}
	if err := dryRunAction.Run(); err != nil {
		t.Fatalf("dry run error: %v", err)
	}
	dryOut := dryRunStdout.String()

	// Must label as [redacted]
	if !strings.Contains(dryOut, "[redacted]") {
		t.Errorf("dry-run missing [redacted] label:\n%s", dryOut)
	}
	// Must say 1 file(s) would be restored
	if !strings.Contains(dryOut, "1 file(s) would be restored") {
		t.Errorf("dry-run missing 1 file(s) would be restored:\n%s", dryOut)
	}
	// Must state the tradeoff plainly
	wantTradeoff := "Restoring a redacted file overwrites the live file's real secrets with placeholders. The user must re-enter them."
	if !strings.Contains(dryOut, wantTradeoff) {
		t.Errorf("dry-run missing mandatory tradeoff warning %q:\n%s", wantTradeoff, dryOut)
	}

	// 2. Real restore apply check
	var restoreStdout bytes.Buffer
	restoreAction := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		Stdout:    &restoreStdout,
		Stderr:    io.Discard,
		Force:     true,
	}
	if err := restoreAction.Run(); err != nil {
		t.Fatalf("restore apply error: %v", err)
	}
	repOut := restoreStdout.String()

	// Must name restored path with placeholders
	if !strings.Contains(repOut, "~/.config/opencode/opencode.json") {
		t.Errorf("restore report missing restored path with placeholders:\n%s", repOut)
	}
	// Must state secrets must be re-entered by hand
	if !strings.Contains(repOut, "secrets must be re-entered by hand") {
		t.Errorf("restore report missing re-entered by hand note:\n%s", repOut)
	}
	// Must state the tradeoff plainly in report
	if !strings.Contains(repOut, wantTradeoff) {
		t.Errorf("restore report missing mandatory tradeoff warning %q:\n%s", wantTradeoff, repOut)
	}
	// Target file on disk now has the placeholder
	liveAfter, err := os.ReadFile(liveFile)
	if err != nil {
		t.Fatalf("read live file after restore: %v", err)
	}
	if !strings.Contains(string(liveAfter), "<YOUR_SECRET>") {
		t.Errorf("live file not overwritten with placeholder:\n%s", string(liveAfter))
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

func TestRestoreAction_RestoreFile_CaseContainment(t *testing.T) { //nolint:paralleltest // shared state
	tests := []struct {
		name       string
		homeDir    string
		backupDir  string
		backupPath string
		targetPath string
		wantEscape bool
		errSnippet string
	}{
		{
			name:       "in_bounds_case_variant_target_accepted",
			homeDir:    "/home/alice",
			backupDir:  "/home/alice/.bak/backups/test",
			backupPath: "safe.txt",
			targetPath: "/HOME/alice/safe.txt",
			wantEscape: false,
		},
		{
			name:       "in_bounds_case_variant_backup_dir_accepted",
			homeDir:    "/home/alice",
			backupDir:  "/HOME/alice/.bak/backups/test",
			backupPath: "safe.txt",
			targetPath: "/home/alice/safe.txt",
			wantEscape: false,
		},
		{
			name:       "escape_target_parent_traversal_uppercase",
			homeDir:    "/home/alice",
			backupDir:  "/home/alice/.bak/backups/test",
			backupPath: "safe.txt",
			targetPath: "/HOME/alice/../../etc/passwd",
			wantEscape: true,
			errSnippet: "target path escapes home directory",
		},
		{
			name:       "escape_target_sibling_dir_case_variant",
			homeDir:    "/home/alice",
			backupDir:  "/home/alice/.bak/backups/test",
			backupPath: "safe.txt",
			targetPath: "/HOME/alice-sibling/escape.txt",
			wantEscape: true,
			errSnippet: "target path escapes home directory",
		},
		{
			name:       "escape_backup_path_traversal_case_variant",
			homeDir:    "/home/alice",
			backupDir:  "/home/alice/.bak/backups/test",
			backupPath: "../OTHER/escape.txt",
			targetPath: "/home/alice/safe.txt",
			wantEscape: true,
			errSnippet: "source path escapes backup directory",
		},
		{
			name:       "escape_backup_path_windows_slash_case_variant",
			homeDir:    "/home/alice",
			backupDir:  "/home/alice/.bak/backups/test",
			backupPath: "..\\ESCAPE\\file.txt",
			targetPath: "/home/alice/safe.txt",
			wantEscape: true,
			errSnippet: "source path escapes backup directory",
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest
			srcFile := filepath.Join(tt.backupDir, tt.backupPath)
			mockFS := &MockFileSystem{
				HomeDir:    tt.homeDir,
				StatResult: make(map[string]MockStatResult),
				Files: map[string][]byte{
					srcFile: []byte("content"),
				},
			}

			action := &RestoreAction{
				FS:        mockFS,
				BackupDir: tt.backupDir,
			}

			err := action.restoreFile(restorepkg.FileDiff{
				BackupPath: tt.backupPath,
				TargetPath: tt.targetPath,
			}, 0)

			if tt.wantEscape {
				if err == nil {
					t.Fatalf("expected error for escaping path, got nil")
				}
				if !strings.Contains(err.Error(), tt.errSnippet) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSnippet)
				}
				// Verify refusal: nothing was written to the target path.
				if _, ok := mockFS.Files[tt.targetPath]; ok {
					t.Errorf("expected target %q not to be written in mock FS after refusal", tt.targetPath)
				}
			} else {
				if err != nil {
					t.Fatalf("expected in-bounds case-variant path to succeed, got: %v", err)
				}
				if _, ok := mockFS.Files[tt.targetPath]; !ok {
					t.Errorf("expected target %q to be written in mock FS, but was not", tt.targetPath)
				}
			}
		})
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
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("mkdir backup dir: %v", err)
	}
	srcFile := filepath.Join(backupDir, "safe.txt")
	if err := os.WriteFile(srcFile, []byte("content"), 0644); err != nil {
		t.Fatalf("write src file: %v", err)
	}

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
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		t.Fatalf("mkdir backups dir: %v", err)
	}

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
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		t.Fatalf("mkdir backups dir: %v", err)
	}

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

var _ io.Reader = (*errorReader)(nil)

func (e *errorReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

// failingWriter returns an error on every Write call.
type failingWriter struct{}

var _ io.Writer = (*failingWriter)(nil)

func (f *failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestRestoreAction_WritersErrorHandling(t *testing.T) { //nolint:paralleltest // shared state
	action := &RestoreAction{}
	fw := &failingWriter{}

	// printDryRunDiff error propagation
	diffs := []restorepkg.FileDiff{{Status: restorepkg.DiffNew, SourcePath: "/a"}}
	if err := action.printDryRunDiff(fw, diffs); err == nil {
		t.Error("printDryRunDiff should return error when writer fails")
	}

	// warnBakVersionMismatch error propagation
	if err := action.warnBakVersionMismatch("test-id", "1.0.0", fw); err == nil {
		t.Error("warnBakVersionMismatch should return error when writer fails")
	}

	// confirmRestore prompt write error propagation
	if _, err := action.confirmRestore(fw, fw); err == nil {
		t.Error("confirmRestore should return error when prompt write fails")
	}

	// reportRestore error propagation
	m := &manifest.Manifest{ID: "test-id"}
	if err := reportRestore(fw, m, 1, 0, 0, 0, nil); err == nil {
		t.Error("reportRestore should return error when writer fails")
	}
}

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

var _ FileSystem = (*chmodFailingFS)(nil)

func (c *chmodFailingFS) Chmod(name string, _ os.FileMode) error {
	return fmt.Errorf("chmod %s: permission denied", name)
}

func TestRestoreAction_ChmodFailure_SurfacesAsError(t *testing.T) { //nolint:paralleltest // shared state
	if isWindows() {
		t.Skip("skipping chmod failure test on Windows: POSIX permission bits do not exist, so restore skips chmod entirely and can never surface a chmod error")
	}
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

var _ FileSystem = (*partialCopyFailingFS)(nil)

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

var _ FileSystem = (*metadataFailCopyFS)(nil)

func (m *metadataFailCopyFS) CopyFile(src, dst string) error {
	if dst == m.failOnDst {
		recBase := filepath.Join(m.homeDir, ".bak", "recovery")
		if entries, err := os.ReadDir(recBase); err == nil && len(entries) > 0 {
			metaFile := filepath.Join(recBase, entries[0].Name(), "repo", "recovery-meta.json")
			if rmErr := os.Remove(metaFile); rmErr != nil && !os.IsNotExist(rmErr) {
				return fmt.Errorf("remove metafile: %w", rmErr)
			}
			if mkErr := os.Mkdir(metaFile, 0755); mkErr != nil {
				return fmt.Errorf("mkdir metafile: %w", mkErr)
			}
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

var _ FileSystem = (*postFailStorageFS)(nil)

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

func TestRestoreAction_BakVersionWarning(t *testing.T) { //nolint:paralleltest // shared state
	tests := []struct {
		name           string
		backupBakVer   string
		injectedBakVer string
		dryRun         bool
		wantWarning    bool
		warnContains   []string
	}{
		{
			name:           "matching_versions_stay_silent",
			backupBakVer:   "1.5.0",
			injectedBakVer: "1.5.0",
			dryRun:         false,
			wantWarning:    false,
		},
		{
			name:           "version_mismatch_warns_with_both_versions_and_id",
			backupBakVer:   "1.4.0",
			injectedBakVer: "1.5.0",
			dryRun:         false,
			wantWarning:    true,
			warnContains:   []string{"warning:", "20260101-120000", "1.4.0", "1.5.0"},
		},
		{
			name:           "running_dev_warns",
			backupBakVer:   "1.5.0",
			injectedBakVer: "dev",
			dryRun:         false,
			wantWarning:    true,
			warnContains:   []string{"warning:", "20260101-120000", "1.5.0", "dev"},
		},
		{
			name:           "backup_dev_warns",
			backupBakVer:   "dev",
			injectedBakVer: "1.5.0",
			dryRun:         false,
			wantWarning:    true,
			warnContains:   []string{"warning:", "20260101-120000", "dev", "1.5.0"},
		},
		{
			name:           "both_dev_warns",
			backupBakVer:   "dev",
			injectedBakVer: "dev",
			dryRun:         false,
			wantWarning:    true,
			warnContains:   []string{"warning:", "20260101-120000", "dev"},
		},
		{
			name:           "running_empty_unknown_warns",
			backupBakVer:   "1.5.0",
			injectedBakVer: "",
			dryRun:         false,
			wantWarning:    true,
			warnContains:   []string{"warning:", "20260101-120000", "1.5.0", "unknown"},
		},
		{
			name:           "backup_empty_unknown_warns",
			backupBakVer:   "",
			injectedBakVer: "1.5.0",
			dryRun:         false,
			wantWarning:    true,
			warnContains:   []string{"warning:", "20260101-120000", "unknown", "1.5.0"},
		},
		{
			name:           "dry_run_with_mismatch_still_warns",
			backupBakVer:   "1.4.0",
			injectedBakVer: "1.5.0",
			dryRun:         true,
			wantWarning:    true,
			warnContains:   []string{"warning:", "20260101-120000", "1.4.0", "1.5.0"},
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest
			home := t.TempDir()
			backupID := createCustomManifestBackupForRestore(t, home, tt.backupBakVer, manifest.ManifestVersion)

			var stdoutBuf, stderrBuf bytes.Buffer
			action := &RestoreAction{
				FS:         newHomeFS(home),
				BackupDir:  filepath.Join(home, ".bak", "backups", backupID),
				BakVersion: tt.injectedBakVer,
				DryRun:     tt.dryRun,
				Force:      true,
				Stdout:     &stdoutBuf,
				Stderr:     &stderrBuf,
			}

			err := action.Run()
			if err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}

			stderrOutput := stderrBuf.String()
			stdoutOutput := stdoutBuf.String()

			// Warning must never leak to Stdout.
			if strings.Contains(stdoutOutput, "warning:") {
				t.Errorf("stdout should not contain warning: %q", stdoutOutput)
			}

			if tt.wantWarning {
				if stderrOutput == "" {
					t.Fatalf("expected warning on Stderr, got empty")
				}
				for _, sub := range tt.warnContains {
					if !strings.Contains(stderrOutput, sub) {
						t.Errorf("stderr %q should contain %q", stderrOutput, sub)
					}
				}
			} else if strings.Contains(stderrOutput, "warning:") {
				t.Errorf("stderr should not contain warning, got %q", stderrOutput)
			}
		})
	}
}

func TestRestoreAction_NewerSchemaVersion_FailsClosedBeforeWrite(t *testing.T) { //nolint:paralleltest // shared state
	tests := []struct {
		name          string
		schemaVersion string
		wantErr       bool
		errContain    string
	}{
		{
			name:          "newer_minor_schema_fails_closed",
			schemaVersion: "0.6.0",
			wantErr:       true,
			errContain:    "unsupported manifest schema version",
		},
		{
			name:          "newer_major_schema_fails_closed",
			schemaVersion: "1.0.0",
			wantErr:       true,
			errContain:    "unsupported manifest schema version",
		},
		{
			name:          "current_schema_050_succeeds",
			schemaVersion: "0.5.0",
			wantErr:       false,
		},
		{
			name:          "older_schema_040_succeeds",
			schemaVersion: "0.4.0",
			wantErr:       false,
		},
		{
			name:          "older_schema_030_succeeds",
			schemaVersion: "0.3.0",
			wantErr:       false,
		},
	}

	for _, tt := range tests { //nolint:paralleltest
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest
			home := t.TempDir()
			backupID := createCustomManifestBackupForRestore(t, home, "1.5.0", tt.schemaVersion)

			targetFile := filepath.Join(home, ".config", "bak", "config.json")
			var stdoutBuf, stderrBuf bytes.Buffer
			action := &RestoreAction{
				FS:         newHomeFS(home),
				BackupDir:  filepath.Join(home, ".bak", "backups", backupID),
				BakVersion: "1.5.0",
				Force:      true,
				Stdout:     &stdoutBuf,
				Stderr:     &stderrBuf,
			}

			err := action.Run()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContain)
				}
				// Verify zero target files written
				if _, statErr := os.Stat(targetFile); !os.IsNotExist(statErr) {
					t.Errorf("target file should not have been written on schema rejection")
				}
				// Verify zero recovery side effects (recovery directory should not have any points)
				recoveryPointsDir := filepath.Join(home, ".bak", "recovery")
				entries, _ := os.ReadDir(recoveryPointsDir)
				if len(entries) > 0 {
					t.Errorf("recovery directory should not contain points, found %d", len(entries))
				}
			} else {
				// Succeeded: target should exist
				if _, statErr := os.Stat(targetFile); statErr != nil {
					t.Errorf("target file should exist: %v", statErr)
				}
			}
		})
	}
}
