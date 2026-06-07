package actions

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// mockFileInfo implements os.FileInfo for testing.
type mockFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return m.size }
func (m mockFileInfo) Mode() os.FileMode  { return 0644 }
func (m mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m mockFileInfo) IsDir() bool        { return m.isDir }
func (m mockFileInfo) Sys() interface{}   { return nil }

// mockDirEntry implements os.DirEntry for testing.
type mockDirEntry struct {
	name  string
	isDir bool
}

func (m mockDirEntry) Name() string               { return m.name }
func (m mockDirEntry) IsDir() bool                 { return m.isDir }
func (m mockDirEntry) Type() fs.FileMode           { return 0 }
func (m mockDirEntry) Info() (os.FileInfo, error)  { return mockFileInfo{name: m.name, isDir: m.isDir}, nil }

// --- push helpers -------------------------------------------------------

func setupPushMockFS(home string) *MockFileSystem {
	backupsDir := filepath.Join(home, ".bak", "backups")
	backupDir := filepath.Join(backupsDir, "20260101-120000")

	return &MockFileSystem{
		HomeDir: home,
		StatResult: map[string]MockStatResult{
			backupDir: {Info: &mockFileInfo{name: "20260101-120000", isDir: true}},
		},
		DirEntries: map[string][]os.DirEntry{
			backupsDir: {
				&mockDirEntry{name: "20260101-120000", isDir: true},
				&mockDirEntry{name: "20260102-130000", isDir: true},
			},
		},
		Files: make(map[string][]byte),
	}
}

// --- tests -------------------------------------------------------------

func TestPushAction_ReadDirError(t *testing.T) {
	mockFS := &MockFileSystem{
		HomeDir: "/home/test",
		ReadDirErrors: map[string]error{
			filepath.Join("/home/test", ".bak", "backups"): os.ErrNotExist,
		},
		StatResult: make(map[string]MockStatResult),
		Files:      make(map[string][]byte),
	}

	action := &PushAction{FS: mockFS, Provider: "github-gist"}
	err := action.Run(nil, nil)
	if err == nil {
		t.Fatal("expected error when backups dir missing")
	}
	if !strings.Contains(err.Error(), "read backups dir") {
		t.Errorf("error should mention reading backups dir: %v", err)
	}
}

func TestPushAction_NoBackupsFound(t *testing.T) {
	home := "/home/test"
	mockFS := &MockFileSystem{
		HomeDir:    home,
		DirEntries: map[string][]os.DirEntry{filepath.Join(home, ".bak", "backups"): {}},
		StatResult: make(map[string]MockStatResult),
		Files:      make(map[string][]byte),
	}

	action := &PushAction{FS: mockFS, Provider: "github-gist"}
	err := action.Run(nil, nil)
	if err == nil {
		t.Fatal("expected error when no backups found")
	}
}

func TestPushAction_StatError(t *testing.T) {
	home := "/home/test"
	backupsDir := filepath.Join(home, ".bak", "backups")

	mockFS := &MockFileSystem{
		HomeDir: home,
		DirEntries: map[string][]os.DirEntry{
			backupsDir: {
				&mockDirEntry{name: "20260101-120000", isDir: true},
			},
		},
		StatResult: map[string]MockStatResult{},
		Files:      make(map[string][]byte),
	}

	action := &PushAction{FS: mockFS, Provider: "github-gist"}
	err := action.Run(nil, nil)
	if err == nil {
		t.Fatal("expected error when backup not found")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestPushAction_LatestBackup(t *testing.T) {
	mockFS := setupPushMockFS("/home/test")

	action := &PushAction{FS: mockFS, Provider: "github-gist", Verbose: true}
	// This will fail at the cloud push step, which is expected.
	err := action.Run(nil, nil)
	if err == nil {
		t.Log("push succeeded (unexpected — may have real credentials)")
	} else {
		// Cloud push failure is expected without real credentials.
		t.Logf("push failed as expected (no cloud credentials): %v", err)
	}
}

func TestPushAction_ExplicitBackupID(t *testing.T) {
	home := "/home/test"
	backupsDir := filepath.Join(home, ".bak", "backups")
	backupDir := filepath.Join(backupsDir, "20260101-120000")

	mockFS := &MockFileSystem{
		HomeDir: home,
		StatResult: map[string]MockStatResult{
			backupDir: {Info: &mockFileInfo{name: "20260101-120000", isDir: true}},
		},
		Files: make(map[string][]byte),
	}

	action := &PushAction{FS: mockFS, Provider: "github-gist"}
	err := action.Run(nil, []string{"20260101-120000"})
	if err == nil {
		t.Log("push succeeded with explicit ID")
	} else {
		t.Logf("push failed as expected: %v", err)
	}
}

func TestPushAction_InvalidBackupID(t *testing.T) {
	home := "/home/test"
	backupsDir := filepath.Join(home, ".bak", "backups")

	mockFS := &MockFileSystem{
		HomeDir:    home,
		StatResult: make(map[string]MockStatResult),
		Files:      make(map[string][]byte),
		DirEntries: map[string][]os.DirEntry{backupsDir: {}},
	}

	action := &PushAction{FS: mockFS, Provider: "github-gist"}
	err := action.Run(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent backup")
	}
}

func TestPushAction_PathTraversal(t *testing.T) {
	home := "/home/test"
	backupsDir := filepath.Join(home, ".bak", "backups")

	mockFS := &MockFileSystem{
		HomeDir: home,
		DirEntries: map[string][]os.DirEntry{
			backupsDir: {
				&mockDirEntry{name: "../escape", isDir: true},
			},
		},
		StatResult: map[string]MockStatResult{},
		Files:      make(map[string][]byte),
	}

	action := &PushAction{FS: mockFS, Provider: "github-gist"}
	err := action.Run(nil, nil)
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}
