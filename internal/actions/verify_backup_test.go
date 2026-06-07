package actions

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/config/testutil"
	"github.com/danielxxomg/bak-cli/internal/manifest"
)

// setupVerifyFixture creates a complete test backup fixture under the given
// home directory. It writes real files, computes their SHA-256 hashes, and
// creates a manifest.json with correct checksums. Returns the backup dir path.
func setupVerifyFixture(t *testing.T, homeDir string, backupID string, files map[string]string) string {
	t.Helper()

	configtest.SetConfigHome(t, homeDir)

	bakDir := filepath.Join(homeDir, ".bak")
	backupDir := filepath.Join(bakDir, "backups", backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	var items []manifest.Item
	for relPath, content := range files {
		fullPath := filepath.Join(backupDir, relPath)
		// Ensure parent directory exists.
		if dir := filepath.Dir(fullPath); dir != backupDir {
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		hash := sha256.Sum256([]byte(content))
		items = append(items, manifest.Item{
			Category:   "config",
			SourcePath: relPath,
			BackupPath: relPath,
			Hash:       "sha256:" + hex.EncodeToString(hash[:]),
			Size:       int64(len(content)),
		})
	}

	m := &manifest.Manifest{
		Version:    manifest.ManifestVersion,
		ID:         backupID,
		Preset:     "quick",
		FileCount:  len(items),
		TotalSize:  sumSizes(items),
		Categories: []string{"config"},
		Adapters: map[string]manifest.AdapterManifest{
			"opencode": {
				ConfigDir: "opencode",
				Items:     items,
			},
		},
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "manifest.json"), data, 0644); err != nil {
		t.Fatal(err)
	}

	return backupDir
}

func sumSizes(items []manifest.Item) int64 {
	var total int64
	for _, it := range items {
		total += it.Size
	}
	return total
}

func TestVerifyBackupAction_Success(t *testing.T) {
	homeDir := t.TempDir()
	files := map[string]string{
		"skills/skill-one.md":  "# Skill One\n\nContent here.",
		"commands/cmd-one.toml": "[command]\nname = \"test\"",
	}
	setupVerifyFixture(t, homeDir, "20260101-120000", files)

	var out, errOut strings.Builder
	action := &VerifyBackupAction{
		Stdout:  &out,
		Stderr:  &errOut,
		Verbose: false,
	}

	err := action.Run("20260101-120000")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "verified") {
		t.Errorf("output should contain 'verified', got: %q", output)
	}
	if !strings.Contains(output, "20260101-120000") {
		t.Errorf("output should contain backup ID: %q", output)
	}
	if !strings.Contains(output, "2 files") {
		t.Errorf("output should mention file count: %q", output)
	}
}

func TestVerifyBackupAction_ChecksumMismatch(t *testing.T) {
	homeDir := t.TempDir()
	files := map[string]string{
		"config.yaml": "original: content",
	}
	backupDir := setupVerifyFixture(t, homeDir, "20260101-130000", files)

	// Corrupt the file on disk — change its content so hash no longer matches.
	corrupted := []byte("modified: content")
	if err := os.WriteFile(filepath.Join(backupDir, "config.yaml"), corrupted, 0644); err != nil {
		t.Fatal(err)
	}

	var out, errOut strings.Builder
	action := &VerifyBackupAction{
		Stdout:  &out,
		Stderr:  &errOut,
		Verbose: false,
	}

	err := action.Run("20260101-130000")
	if err == nil {
		t.Fatal("expected hash mismatch error")
	}
	if !strings.Contains(err.Error(), "hash mismatch") {
		t.Errorf("error should mention hash mismatch: %v", err)
	}
}

func TestVerifyBackupAction_MissingManifest(t *testing.T) {
	homeDir := t.TempDir()
	configtest.SetConfigHome(t, homeDir)

	bakDir := filepath.Join(homeDir, ".bak")
	backupDir := filepath.Join(bakDir, "backups", "20260101-140000")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	// No manifest.json — the backup directory exists but has no manifest.

	var out, errOut strings.Builder
	action := &VerifyBackupAction{
		Stdout:  &out,
		Stderr:  &errOut,
		Verbose: false,
	}

	err := action.Run("20260101-140000")
	if err == nil {
		t.Fatal("expected manifest load error")
	}
	if !strings.Contains(err.Error(), "load manifest") {
		t.Errorf("error should mention load manifest: %v", err)
	}
}

func TestVerifyBackupAction_MissingBackup(t *testing.T) {
	homeDir := t.TempDir()
	configtest.SetConfigHome(t, homeDir)

	// No backup directory at all.

	var out, errOut strings.Builder
	action := &VerifyBackupAction{
		Stdout:  &out,
		Stderr:  &errOut,
		Verbose: false,
	}

	err := action.Run("nonexistent-backup")
	if err == nil {
		t.Fatal("expected error for nonexistent backup")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestVerifyBackupAction_MissingFileOnDisk(t *testing.T) {
	homeDir := t.TempDir()
	files := map[string]string{
		"will-be-deleted.md": "# I will be deleted",
	}
	backupDir := setupVerifyFixture(t, homeDir, "20260101-150000", files)

	// Remove the file from disk but keep the manifest referencing it.
	if err := os.Remove(filepath.Join(backupDir, "will-be-deleted.md")); err != nil {
		t.Fatal(err)
	}

	var out, errOut strings.Builder
	action := &VerifyBackupAction{
		Stdout:  &out,
		Stderr:  &errOut,
		Verbose: false,
	}

	err := action.Run("20260101-150000")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	// The error should come from manifest.Validate trying to hash the missing file.
	if !strings.Contains(err.Error(), "open") {
		t.Errorf("error should mention file open failure: %v", err)
	}
}

func TestVerifyBackupAction_VerboseOutput(t *testing.T) {
	homeDir := t.TempDir()
	files := map[string]string{
		"skills/skill-one.md": "# Skill One",
	}
	setupVerifyFixture(t, homeDir, "20260101-160000", files)

	var out, errOut strings.Builder
	action := &VerifyBackupAction{
		Stdout:  &out,
		Stderr:  &errOut,
		Verbose: true,
	}

	err := action.Run("20260101-160000")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Verbose progress goes to stderr.
	stderrOutput := errOut.String()
	if !strings.Contains(stderrOutput, "Verifying") {
		t.Errorf("stderr should contain 'Verifying', got: %q", stderrOutput)
	}
	if !strings.Contains(stderrOutput, "verifying") {
		t.Errorf("stderr should contain per-file 'verifying', got: %q", stderrOutput)
	}

	// Success message still goes to stdout.
	if !strings.Contains(out.String(), "verified") {
		t.Errorf("stdout should contain success: %q", out.String())
	}
}
