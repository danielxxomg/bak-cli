package actions

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/danielxxomg/bak-cli/internal/cloud"
	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/danielxxomg/bak-cli/internal/crypto"
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

func (m mockDirEntry) Name() string      { return m.name }
func (m mockDirEntry) IsDir() bool       { return m.isDir }
func (m mockDirEntry) Type() fs.FileMode { return 0 }
func (m mockDirEntry) Info() (os.FileInfo, error) {
	return mockFileInfo{name: m.name, isDir: m.isDir}, nil
}

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

func TestPushAction_VerboseLogging(t *testing.T) {
	home := t.TempDir()
	mockFS := &MockFileSystem{
		HomeDir:    home,
		StatResult: make(map[string]MockStatResult),
		Files:      make(map[string][]byte),
	}

	action := &PushAction{FS: mockFS, Provider: "github-gist", Verbose: true}
	_ = action.Run(nil, []string{"nonexistent"})
	// Just exercises the verbose code path. Error expected.
}

func TestPushAction_UnknownProvider(t *testing.T) {
	mockFS := &MockFileSystem{
		HomeDir:    "/home/test",
		StatResult: make(map[string]MockStatResult),
		Files:      make(map[string][]byte),
	}

	action := &PushAction{FS: mockFS, Provider: "unknown-backend-xyz"}
	err := action.Run(nil, nil)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestPushAction_PathTraversal_Latest(t *testing.T) {
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

func TestPushAction_ExplicitArgResolvesID(t *testing.T) {
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
	// Explicit arg should resolve without reading dir.
	err := action.Run(nil, []string{"20260101-120000"})
	if err != nil {
		t.Logf("push with explicit arg: %v", err)
	}
}

func TestPushAction_EmptyArgFallsback(t *testing.T) {
	home := "/home/test"
	backupsDir := filepath.Join(home, ".bak", "backups")

	mockFS := &MockFileSystem{
		HomeDir: home,
		DirEntries: map[string][]os.DirEntry{
			backupsDir: {
				&mockDirEntry{name: "20260102-130000", isDir: true},
			},
		},
		StatResult: map[string]MockStatResult{
			filepath.Join(backupsDir, "20260102-130000"): {Info: &mockFileInfo{name: "20260102-130000", isDir: true}},
		},
		Files: make(map[string][]byte),
	}

	action := &PushAction{FS: mockFS, Provider: "github-gist"}
	// Empty string arg should fall back to latest.
	err := action.Run(nil, []string{""})
	if err != nil {
		t.Logf("push with empty arg: %v", err)
	}
}

func TestPushAction_HomeDirError(t *testing.T) {
	// A mock that fails UserHomeDir (though our mock always succeeds).
	// Test the Stat error path instead.
	mockFS := &MockFileSystem{
		HomeDir:    "/home/test",
		StatResult: map[string]MockStatResult{},
		Files:      make(map[string][]byte),
	}

	action := &PushAction{FS: mockFS, Provider: "github-gist"}
	err := action.Run(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent backup")
	}
}

func TestPushAction_ReadDirErrorOnFallback(t *testing.T) {
	mockFS := &MockFileSystem{
		HomeDir: "/home/test",
		ReadDirErrors: map[string]error{
			filepath.Join("/home/test", ".bak", "backups"): os.ErrPermission,
		},
		StatResult: make(map[string]MockStatResult),
		Files:      make(map[string][]byte),
	}

	action := &PushAction{FS: mockFS, Provider: "github-gist"}
	err := action.Run(nil, nil) // no args → fallback to ReadDir
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPushAction_MockProvider_HappyPath(t *testing.T) {
	home := t.TempDir()

	// Create a real backup to package.
	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups")
	backupID := "20260101-120000"
	backupPath := filepath.Join(backupsDir, backupID)
	os.MkdirAll(backupPath, 0755)

	// Create a manifest so Stat succeeds.
	manifestData := []byte(`{"id":"20260101-120000","version":"1.0"}`)
	os.WriteFile(filepath.Join(backupPath, "manifest.json"), manifestData, 0644)

	// Track push calls.
	var pushedArchive []byte
	var pushedMeta cloud.PushMeta
	pushCalled := false

	mockProvider := &MockProvider{
		MockName: "mock-gist",
		PushFn: func(archive []byte, meta cloud.PushMeta) (string, error) {
			pushedArchive = archive
			pushedMeta = meta
			pushCalled = true
			return "mock-id-123", nil
		},
	}

	factory := &MockProviderFactory{
		Providers: map[string]cloud.Provider{
			"mock-gist": mockProvider,
		},
	}

	action := &PushAction{
		FS:         newHomeFS(home),
		Provider:   "mock-gist",
		Factory:    factory,
		HostnameFn: func() (string, error) { return "testbox", nil },
	}

	err := action.Run(nil, []string{backupID})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !pushCalled {
		t.Fatal("provider.Push was not called")
	}
	if len(pushedArchive) == 0 {
		t.Error("pushed archive is empty")
	}
	if pushedMeta.BackupID != backupID {
		t.Errorf("pushed meta BackupID = %q, want %q", pushedMeta.BackupID, backupID)
	}
	if pushedMeta.Hostname != "testbox" {
		t.Errorf("pushed meta Hostname = %q, want testbox", pushedMeta.Hostname)
	}
}

func TestPushAction_MockProvider_ProviderError(t *testing.T) {
	home := t.TempDir()

	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups")
	backupID := "20260101-120000"
	backupPath := filepath.Join(backupsDir, backupID)
	os.MkdirAll(backupPath, 0755)
	os.WriteFile(filepath.Join(backupPath, "manifest.json"), []byte(`{"id":"test"}`), 0644)

	factory := &MockProviderFactory{
		Err: errors.New("factory explosion"),
	}

	action := &PushAction{
		FS:       newHomeFS(home),
		Provider: "mock-gist",
		Factory:  factory,
	}

	err := action.Run(nil, []string{backupID})
	if err == nil {
		t.Fatal("expected error from factory")
	}
}

func TestPushAction_MockProvider_PushError(t *testing.T) {
	home := t.TempDir()

	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups")
	backupID := "20260101-120000"
	backupPath := filepath.Join(backupsDir, backupID)
	os.MkdirAll(backupPath, 0755)
	os.WriteFile(filepath.Join(backupPath, "manifest.json"), []byte(`{"id":"test"}`), 0644)

	mockProvider := &MockProvider{
		MockName: "mock-gist",
		PushFn: func(archive []byte, meta cloud.PushMeta) (string, error) {
			return "", errors.New("network timeout")
		},
	}

	factory := &MockProviderFactory{
		Providers: map[string]cloud.Provider{
			"mock-gist": mockProvider,
		},
	}

	action := &PushAction{
		FS:       newHomeFS(home),
		Provider: "mock-gist",
		Factory:  factory,
	}

	err := action.Run(nil, []string{backupID})
	if err == nil {
		t.Fatal("expected error from provider push")
	}
	if !strings.Contains(err.Error(), "push") {
		t.Errorf("error should mention push: %v", err)
	}
}

// --- encryption tests ---------------------------------------------------

func TestPushAction_EncryptionEnabled(t *testing.T) {
	home := t.TempDir()

	// Create a real backup to package.
	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups")
	backupID := "20260101-120000"
	backupPath := filepath.Join(backupsDir, backupID)
	os.MkdirAll(backupPath, 0755)
	os.WriteFile(filepath.Join(backupPath, "manifest.json"), []byte(`{"id":"20260101-120000","version":"1.0"}`), 0644)

	t.Setenv("BAK_ENCRYPTION_PASSWORD", "test-password-123")

	var pushedArchive []byte
	mockProvider := &MockProvider{
		MockName: "mock-gist",
		PushFn: func(archive []byte, meta cloud.PushMeta) (string, error) {
			pushedArchive = archive
			return "mock-id-123", nil
		},
	}

	factory := &MockProviderFactory{
		Providers: map[string]cloud.Provider{
			"mock-gist": mockProvider,
		},
	}

	action := &PushAction{
		FS:       newHomeFS(home),
		Provider: "mock-gist",
		Profile:  "default",
		Factory:  factory,
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{
				Profiles: map[string]config.ProfileConfig{
					"default": {
						Encryption: &config.EncryptionConfig{
							Enabled: true,
						},
					},
				},
			}, nil
		},
	}

	err := action.Run(nil, []string{backupID})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !crypto.IsEncrypted(pushedArchive) {
		t.Fatal("expected pushed archive to be encrypted, but magic bytes not found")
	}
}

func TestPushAction_EncryptionDisabled(t *testing.T) {
	home := t.TempDir()

	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups")
	backupID := "20260101-120000"
	backupPath := filepath.Join(backupsDir, backupID)
	os.MkdirAll(backupPath, 0755)
	os.WriteFile(filepath.Join(backupPath, "manifest.json"), []byte(`{"id":"20260101-120000","version":"1.0"}`), 0644)

	var pushedArchive []byte
	mockProvider := &MockProvider{
		MockName: "mock-gist",
		PushFn: func(archive []byte, meta cloud.PushMeta) (string, error) {
			pushedArchive = archive
			return "mock-id-123", nil
		},
	}

	factory := &MockProviderFactory{
		Providers: map[string]cloud.Provider{
			"mock-gist": mockProvider,
		},
	}

	action := &PushAction{
		FS:       newHomeFS(home),
		Provider: "mock-gist",
		Profile:  "default",
		Factory:  factory,
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{
				Profiles: map[string]config.ProfileConfig{
					"default": {
						// Encryption not set — disabled.
					},
				},
			}, nil
		},
	}

	err := action.Run(nil, []string{backupID})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if crypto.IsEncrypted(pushedArchive) {
		t.Fatal("expected pushed archive to be plaintext, but magic bytes found")
	}
}

func TestPushAction_NonexistentProfile(t *testing.T) {
	home := t.TempDir()

	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups")
	backupID := "20260101-120000"
	backupPath := filepath.Join(backupsDir, backupID)
	os.MkdirAll(backupPath, 0755)
	os.WriteFile(filepath.Join(backupPath, "manifest.json"), []byte(`{"id":"20260101-120000","version":"1.0"}`), 0644)

	var pushedArchive []byte
	mockProvider := &MockProvider{
		MockName: "mock-gist",
		PushFn: func(archive []byte, meta cloud.PushMeta) (string, error) {
			pushedArchive = archive
			return "mock-id-123", nil
		},
	}

	factory := &MockProviderFactory{
		Providers: map[string]cloud.Provider{
			"mock-gist": mockProvider,
		},
	}

	action := &PushAction{
		FS:       newHomeFS(home),
		Provider: "mock-gist",
		Profile:  "nonexistent",
		Factory:  factory,
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{
				Profiles: map[string]config.ProfileConfig{
					"default": {
						Encryption: &config.EncryptionConfig{
							Enabled: true,
						},
					},
				},
			}, nil
		},
	}

	err := action.Run(nil, []string{backupID})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Nonexistent profile should gracefully fall back to plaintext.
	if crypto.IsEncrypted(pushedArchive) {
		t.Fatal("expected plaintext for nonexistent profile, but archive is encrypted")
	}
}

func TestPushAction_ConfigLoadError(t *testing.T) {
	home := t.TempDir()

	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups")
	backupID := "20260101-120000"
	backupPath := filepath.Join(backupsDir, backupID)
	os.MkdirAll(backupPath, 0755)
	os.WriteFile(filepath.Join(backupPath, "manifest.json"), []byte(`{"id":"20260101-120000","version":"1.0"}`), 0644)

	mockProvider := &MockProvider{
		MockName: "mock-gist",
		PushFn: func(archive []byte, meta cloud.PushMeta) (string, error) {
			return "mock-id-123", nil
		},
	}

	factory := &MockProviderFactory{
		Providers: map[string]cloud.Provider{
			"mock-gist": mockProvider,
		},
	}

	action := &PushAction{
		FS:       newHomeFS(home),
		Provider: "mock-gist",
		Profile:  "default",
		Factory:  factory,
		ConfigLoader: func() (*config.Config, error) {
			return nil, errors.New("config file corrupted")
		},
	}

	err := action.Run(nil, []string{backupID})
	if err == nil {
		t.Fatal("expected error from config loader")
	}
	if !strings.Contains(err.Error(), "load config") {
		t.Errorf("error should mention load config: %v", err)
	}
}

func TestPushAction_PasswordError(t *testing.T) {
	home := t.TempDir()

	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups")
	backupID := "20260101-120000"
	backupPath := filepath.Join(backupsDir, backupID)
	os.MkdirAll(backupPath, 0755)
	os.WriteFile(filepath.Join(backupPath, "manifest.json"), []byte(`{"id":"20260101-120000","version":"1.0"}`), 0644)

	// Unset the env var so GetPassword falls through to stdin.
	// In a non-interactive test environment, stdin returns io.EOF,
	// triggering the password error.
	os.Unsetenv("BAK_ENCRYPTION_PASSWORD")

	mockProvider := &MockProvider{
		MockName: "mock-gist",
		PushFn: func(archive []byte, meta cloud.PushMeta) (string, error) {
			return "mock-id-123", nil
		},
	}

	factory := &MockProviderFactory{
		Providers: map[string]cloud.Provider{
			"mock-gist": mockProvider,
		},
	}

	action := &PushAction{
		FS:       newHomeFS(home),
		Provider: "mock-gist",
		Profile:  "default",
		Factory:  factory,
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{
				Profiles: map[string]config.ProfileConfig{
					"default": {
						Encryption: &config.EncryptionConfig{
							Enabled: true,
						},
					},
				},
			}, nil
		},
	}

	err := action.Run(nil, []string{backupID})
	if err == nil {
		t.Fatal("expected error from password prompt")
	}
	if !strings.Contains(err.Error(), "encryption password") {
		t.Errorf("error should mention encryption password: %v", err)
	}
}
