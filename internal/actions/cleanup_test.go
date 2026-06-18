package actions

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCleanupAction_KeepsNewest verifies keep=N logic.
func TestCleanupAction_KeepsNewest(t *testing.T) {
	dir := t.TempDir()
	createBackupDirs(t, dir, 10)
	mockFS := &MockFileSystem{
		DirEntries: map[string][]os.DirEntry{
			dir: readDirEntries(t, dir),
		},
	}

	stdout := new(bytes.Buffer)
	action := &CleanupAction{
		FS:         mockFS,
		BackupsDir: dir,
		Keep:       3,
		Force:      true,
		Stdout:     stdout,
		Stderr:     new(bytes.Buffer),
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// 10 backups, keep 3 → 7 deleted.
	if mockFS.RemoveAllCalls != 7 {
		t.Errorf("RemoveAllCalls = %d, want 7", mockFS.RemoveAllCalls)
	}
}

// TestCleanupAction_DryRun verifies 0 deletions in dry-run mode.
func TestCleanupAction_DryRun(t *testing.T) {
	dir := t.TempDir()
	createBackupDirs(t, dir, 5)
	mockFS := &MockFileSystem{
		DirEntries: map[string][]os.DirEntry{
			dir: readDirEntries(t, dir),
		},
	}

	stdout := new(bytes.Buffer)
	action := &CleanupAction{
		FS:         mockFS,
		BackupsDir: dir,
		Keep:       2,
		DryRun:     true,
		Stdout:     stdout,
		Stderr:     new(bytes.Buffer),
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if mockFS.RemoveAllCalls != 0 {
		t.Error("dry-run should delete 0 backups")
	}
	if !strings.Contains(stdout.String(), "Would delete") {
		t.Error("dry-run output should contain 'Would delete'")
	}
}

// TestCleanupAction_KeepAboveCount deletes nothing.
func TestCleanupAction_KeepAboveCount(t *testing.T) {
	dir := t.TempDir()
	createBackupDirs(t, dir, 3)
	mockFS := &MockFileSystem{
		DirEntries: map[string][]os.DirEntry{
			dir: readDirEntries(t, dir),
		},
	}

	stdout := new(bytes.Buffer)
	action := &CleanupAction{
		FS:         mockFS,
		BackupsDir: dir,
		Keep:       10,
		Force:      true,
		Stdout:     stdout,
		Stderr:     new(bytes.Buffer),
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if mockFS.RemoveAllCalls != 0 {
		t.Error("should not delete when keep > count")
	}
}

// TestCleanupAction_ConfirmFalse cancels.
func TestCleanupAction_ConfirmFalse(t *testing.T) {
	dir := t.TempDir()
	createBackupDirs(t, dir, 5)
	mockFS := &MockFileSystem{
		DirEntries: map[string][]os.DirEntry{
			dir: readDirEntries(t, dir),
		},
	}

	stdout := new(bytes.Buffer)
	action := &CleanupAction{
		FS:         mockFS,
		BackupsDir: dir,
		Keep:       2,
		Confirm:    func() bool { return false },
		Stdout:     stdout,
		Stderr:     new(bytes.Buffer),
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if mockFS.RemoveAllCalls != 0 {
		t.Error("confirmed false should delete nothing")
	}
}

// TestCleanupAction_ConfirmNilNoForce errors.
func TestCleanupAction_ConfirmNilNoForce(t *testing.T) {
	dir := t.TempDir()
	createBackupDirs(t, dir, 5)
	mockFS := &MockFileSystem{
		DirEntries: map[string][]os.DirEntry{
			dir: readDirEntries(t, dir),
		},
	}

	action := &CleanupAction{
		FS:         mockFS,
		BackupsDir: dir,
		Keep:       2,
		Stdout:     new(bytes.Buffer),
		Stderr:     new(bytes.Buffer),
	}

	err := action.Run()
	if err == nil {
		t.Error("nil confirm without force should error")
	}
}

// TestCleanupAction_EmptyDir is a no-op.
func TestCleanupAction_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	mockFS := &MockFileSystem{
		DirEntries: map[string][]os.DirEntry{
			dir: nil,
		},
	}
	stdout := new(bytes.Buffer)

	action := &CleanupAction{
		FS:         mockFS,
		BackupsDir: dir,
		Keep:       3,
		Force:      true,
		Stdout:     stdout,
		Stderr:     new(bytes.Buffer),
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

// --- test helpers ---

// createBackupDirs creates n backup subdirectories under dir.
func createBackupDirs(t *testing.T, dir string, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("202606%02d-120000", 17-i)
		if err := os.MkdirAll(filepath.Join(dir, id), 0755); err != nil {
			t.Fatal(err)
		}
	}
}

// readDirEntries reads directory entries from dir.
func readDirEntries(t *testing.T, dir string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	return entries
}
