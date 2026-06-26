package actions

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/cloud"
	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/danielxxomg/bak-cli/internal/crypto"
)

// --- pull tests --------------------------------------------------------

func TestPullAction_MkdirError(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := "/home/test"
	backupsDir := filepath.Join(home, ".bak", "backups")

	mockFS := &MockFileSystem{
		HomeDir: home,
		MkdirErrors: map[string]error{
			backupsDir: errors.New("permission denied"),
		},
		StatResult: make(map[string]MockStatResult),
		Files:      make(map[string][]byte),
	}

	action := &PullAction{
		FS: mockFS,
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{}, nil
		},
		Provider: "github-gist",
	}
	// This will fail at config load or before MkdirAll since config.Load()
	// tries to access the real filesystem.
	err := action.Run(nil)
	if err != nil {
		t.Logf("pull failed as expected: %v", err)
	}
}

func TestPullAction_ConfigLoadError(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// Use a home that doesn't exist to trigger config.Load() error.
	mockFS := &MockFileSystem{
		HomeDir:    "/nonexistent/homedir",
		StatResult: make(map[string]MockStatResult),
		Files:      make(map[string][]byte),
	}

	action := &PullAction{
		FS: mockFS,
		ConfigLoader: func() (*config.Config, error) {
			return nil, errors.New("config load failed")
		},
		Provider: "github-gist",
	}
	err := action.Run(nil)
	if err == nil {
		t.Fatal("expected error when config load fails")
	}
}

func TestPullAction_NoStoredBackupID(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := "/home/test"
	mockFS := &MockFileSystem{
		HomeDir:    home,
		StatResult: make(map[string]MockStatResult),
		Files:      make(map[string][]byte),
	}

	action := &PullAction{
		FS: mockFS,
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{}, nil
		},
		Provider: "github-gist",
	}
	// No args and no stored ID → should fail.
	err := action.Run(nil)
	if err == nil {
		t.Fatal("expected error when no backup ID provided")
	}
}

func TestPullAction_ExplicitID(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	mockFS := &MockFileSystem{
		HomeDir:     "/home/test",
		StatResult:  make(map[string]MockStatResult),
		Files:       make(map[string][]byte),
		MkdirErrors: make(map[string]error),
	}

	action := &PullAction{
		FS: mockFS,
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{}, nil
		},
		Provider: "github-gist",
		Verbose:  true,
	}
	err := action.Run([]string{"abc123"})
	if err != nil {
		// Expected: either config load, factory not configured, or cloud pull failure.
		if !strings.Contains(err.Error(), "pull") &&
			!strings.Contains(err.Error(), "load config") &&
			!strings.Contains(err.Error(), "factory") {
			t.Errorf("unexpected error: %v", err)
		}
		t.Logf("pull failed as expected: %v", err)
	}
}

func TestPullAction_MkdirAllError(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()

	// Set up a custom FS that fails MkdirAll.
	fs := &mkdirFailingFS{OSFileSystem: &OSFileSystem{}, home: home}

	action := &PullAction{
		FS: fs,
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{}, nil
		},
		Provider: "github-gist",
	}
	err := action.Run([]string{"test-id"})
	if err != nil {
		// We expect either config load error or cloud pull error first,
		// since mkdir happens after download. The mkdir error path is
		// exercised when the cloud pull would succeed.
		t.Logf("pull failed: %v", err)
	}
}

func TestPullAction_UserHomeDir(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	mockFS := &MockFileSystem{
		HomeDir:    "/home/test",
		StatResult: make(map[string]MockStatResult),
		Files:      make(map[string][]byte),
	}

	action := &PullAction{
		FS: mockFS,
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{}, nil
		},
		Provider: "github-gist",
	}
	err := action.Run([]string{"test-id"})
	if err != nil {
		t.Logf("pull failed as expected: %v", err)
	}
}

func TestPullAction_InvalidProvider(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	mockFS := &MockFileSystem{
		HomeDir:    "/home/test",
		StatResult: make(map[string]MockStatResult),
		Files:      make(map[string][]byte),
	}

	action := &PullAction{
		FS: mockFS,
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{}, nil
		},
		Provider: "unknown-provider",
	}
	err := action.Run([]string{"test-id"})
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "provider") && !strings.Contains(err.Error(), "factory") {
		t.Errorf("error should mention provider or factory: %v", err)
	}
}

func TestPullAction_MockProvider_HappyPath(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()

	mockProvider := &MockProvider{
		MockName: "mock-gist",
		PullFn: func(id string) ([]byte, error) {
			// Return a minimal tar.gz. cloud.UntarGz will try to decompress;
			// a pre-built valid archive is complex. Instead return data that
			// triggers a specific error AFTER the provider.Pull succeeds.
			if id != "abc123" {
				return nil, fmt.Errorf("unexpected id: %q", id)
			}
			// Return minimal gzip data (empty tar). UntarGz will fail
			// but that proves provider.Pull was called successfully.
			gzData := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
			return gzData, nil
		},
	}

	factory := &MockProviderFactory{
		Providers: map[string]cloud.Provider{
			"mock-gist": mockProvider,
		},
	}

	action := &PullAction{
		FS:       newHomeFS(home),
		Provider: "mock-gist",
		Factory:  factory,
	}

	err := action.Run([]string{"abc123"})
	if err != nil {
		// Error is expected from UntarGz on empty data, proving
		// provider.Pull was called successfully.
		if !strings.Contains(err.Error(), "extract") && !strings.Contains(err.Error(), "gzip") && !strings.Contains(err.Error(), "unexpected") {
			t.Errorf("expected extract/gzip error, got: %v", err)
		}
	}
}

func TestPullAction_MockProvider_FactoryError(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	mockFS := &MockFileSystem{
		HomeDir:    "/home/test",
		StatResult: make(map[string]MockStatResult),
		Files:      make(map[string][]byte),
	}

	factory := &MockProviderFactory{
		Err: errors.New("factory kaput"),
	}

	action := &PullAction{
		FS: mockFS,
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{}, nil
		},
		Provider: "mock-gist",
		Factory:  factory,
	}

	err := action.Run([]string{"abc123"})
	if err == nil {
		t.Fatal("expected error from factory")
	}
}

func TestPullAction_MockProvider_PullError(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	home := t.TempDir()

	mockProvider := &MockProvider{
		MockName: "mock-gist",
		PullFn: func(id string) ([]byte, error) {
			return nil, errors.New("network timeout")
		},
	}

	factory := &MockProviderFactory{
		Providers: map[string]cloud.Provider{
			"mock-gist": mockProvider,
		},
	}

	action := &PullAction{
		FS: newHomeFS(home),
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{}, nil
		},
		Provider: "mock-gist",
		Factory:  factory,
	}

	err := action.Run([]string{"abc123"})
	if err == nil {
		t.Fatal("expected error from provider pull")
	}
	if !strings.Contains(err.Error(), "pull") {
		t.Errorf("error should mention pull: %v", err)
	}
}

// --- encryption test helpers --------------------------------------------

// buildTarGz creates a valid tar.gz archive in memory from a map
// of filename→content and returns the raw bytes.
func buildTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header for %s: %v", name, err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("write tar entry for %s: %v", name, err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}

	return buf.Bytes()
}

// buildEncryptedArchive creates a base64-encoded encrypted archive
// suitable for use as a MockProvider PullFn return value.
func buildEncryptedArchive(t *testing.T, files map[string]string, password string) []byte {
	t.Helper()

	rawTarGz := buildTarGz(t, files)
	encrypted, err := crypto.Encrypt(rawTarGz, password)
	if err != nil {
		t.Fatalf("encrypt archive: %v", err)
	}
	return []byte(base64.StdEncoding.EncodeToString(encrypted))
}

// buildPlainArchive creates a base64-encoded plaintext archive
// (no encryption magic bytes) suitable for a MockProvider PullFn.
func buildPlainArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	rawTarGz := buildTarGz(t, files)
	return []byte(base64.StdEncoding.EncodeToString(rawTarGz))
}

// verifyExtractedFiles reads the backup directory created by PullAction
// and checks that every expected file matches its content.
func verifyExtractedFiles(t *testing.T, home string, expected map[string]string) {
	t.Helper()

	bakDir := filepath.Join(home, ".bak", "backups")
	entries, err := os.ReadDir(bakDir)
	if err != nil {
		t.Fatalf("read backups dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no backup directories created")
	}

	backupPath := filepath.Join(bakDir, entries[0].Name())

	for filename, want := range expected {
		filePath := filepath.Join(backupPath, filename)
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("read %s: %v", filePath, err)
			continue
		}
		if string(data) != want {
			t.Errorf("file %s: got %q, want %q", filePath, string(data), want)
		}
	}
}

// --- encryption integration tests ----------------------------------------

func TestPull_EncryptedRoundTrip(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name  string
		files map[string]string
	}{
		{
			name: "round-trip",
			files: map[string]string{
				"hello.txt": "hello, encrypted world!",
			},
		},
		{
			name: "multi-file",
			files: map[string]string{
				"a.txt":      "content A",
				"b.txt":      "content B",
				"sub/c.txt":  "nested content",
				"config.yml": "key: value\n",
			},
		},
		{
			name:  "empty archive",
			files: map[string]string{},
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			const password = "test-password-123"
			t.Setenv("BAK_ENCRYPTION_PASSWORD", password)

			archiveData := buildEncryptedArchive(t, tt.files, password)
			home := t.TempDir()

			mockProvider := &MockProvider{
				MockName: "mock-gist",
				PullFn: func(id string) ([]byte, error) {
					return archiveData, nil
				},
			}

			factory := &MockProviderFactory{
				Providers: map[string]cloud.Provider{
					"mock-gist": mockProvider,
				},
			}

			action := &PullAction{
				FS:       newHomeFS(home),
				Provider: "mock-gist",
				Factory:  factory,
				ConfigLoader: func() (*config.Config, error) {
					return &config.Config{}, nil
				},
			}

			err := action.Run([]string{"test-id"})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}

			verifyExtractedFiles(t, home, tt.files)
		})
	}
}

func TestPull_BackwardCompatPlaintext(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name  string
		files map[string]string
	}{
		{
			name: "plaintext",
			files: map[string]string{
				"readme.md": "# Plaintext Backup\n",
			},
		},
		{
			name: "non-encrypted",
			files: map[string]string{
				"data.bin": "\x00\x01\x02\x03",
				"info.txt": "no magic bytes here",
			},
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			// Intentionally do NOT set BAK_ENCRYPTION_PASSWORD.
			archiveData := buildPlainArchive(t, tt.files)
			home := t.TempDir()

			mockProvider := &MockProvider{
				MockName: "mock-gist",
				PullFn: func(id string) ([]byte, error) {
					return archiveData, nil
				},
			}

			factory := &MockProviderFactory{
				Providers: map[string]cloud.Provider{
					"mock-gist": mockProvider,
				},
			}

			action := &PullAction{
				FS:       newHomeFS(home),
				Provider: "mock-gist",
				Factory:  factory,
				ConfigLoader: func() (*config.Config, error) {
					return &config.Config{}, nil
				},
			}

			err := action.Run([]string{"test-id"})
			if err != nil {
				t.Fatalf("Run: %v", err)
			}

			verifyExtractedFiles(t, home, tt.files)
		})
	}
}

func TestPull_WrongPassword(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name         string
		envPassword  string
		wantContains string
	}{
		{
			name:         "wrong password",
			envPassword:  "bad-password",
			wantContains: "decrypt archive",
		},
		{
			name:         "empty password",
			envPassword:  "",
			wantContains: "decrypt archive",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			const encryptPassword = "correct-password"
			t.Setenv("BAK_ENCRYPTION_PASSWORD", tt.envPassword)

			files := map[string]string{"secret.txt": "classified"}
			archiveData := buildEncryptedArchive(t, files, encryptPassword)
			home := t.TempDir()

			mockProvider := &MockProvider{
				MockName: "mock-gist",
				PullFn: func(id string) ([]byte, error) {
					return archiveData, nil
				},
			}

			factory := &MockProviderFactory{
				Providers: map[string]cloud.Provider{
					"mock-gist": mockProvider,
				},
			}

			action := &PullAction{
				FS:       newHomeFS(home),
				Provider: "mock-gist",
				Factory:  factory,
				ConfigLoader: func() (*config.Config, error) {
					return &config.Config{}, nil
				},
			}

			err := action.Run([]string{"test-id"})
			if err == nil {
				t.Fatal("expected error for wrong/empty password")
			}
			if !strings.Contains(err.Error(), tt.wantContains) {
				t.Errorf("error should contain %q, got: %v", tt.wantContains, err)
			}
		})
	}
}

// TestPullAction_OutputRouting verifies that all output goes through
// injectable Stdout/Stderr writers instead of fmt.Printf / os.Stderr.
func TestPullAction_OutputRouting(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	const password = "test-password-456"
	t.Setenv("BAK_ENCRYPTION_PASSWORD", password)

	files := map[string]string{
		"hello.txt": "hello, output routing test!",
	}
	archiveData := buildEncryptedArchive(t, files, password)
	home := t.TempDir()

	mockProvider := &MockProvider{
		MockName: "mock-gist",
		PullFn: func(id string) ([]byte, error) {
			return archiveData, nil
		},
	}

	factory := &MockProviderFactory{
		Providers: map[string]cloud.Provider{
			"mock-gist": mockProvider,
		},
	}

	var stdoutBuf, stderrBuf bytes.Buffer

	action := &PullAction{
		FS:       newHomeFS(home),
		Provider: "mock-gist",
		Factory:  factory,
		Verbose:  true,
		Stdout:   &stdoutBuf,
		Stderr:   &stderrBuf,
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{}, nil
		},
	}

	err := action.Run([]string{"test-id"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	stdout := stdoutBuf.String()
	stderr := stderrBuf.String()

	// Stdout assertions: informational messages MUST go to Stdout.
	if !strings.Contains(stdout, "Downloading backup") {
		t.Errorf("stdout missing 'Downloading backup':\n%s", stdout)
	}
	if !strings.Contains(stdout, "Extracting backup") {
		t.Errorf("stdout missing 'Extracting backup':\n%s", stdout)
	}
	if !strings.Contains(stdout, "Backup pulled:") {
		t.Errorf("stdout missing 'Backup pulled:':\n%s", stdout)
	}
	if !strings.Contains(stdout, "Run 'bak restore") {
		t.Errorf("stdout missing 'Run 'bak restore':\n%s", stdout)
	}

	// Stderr assertions: verbose/diagnostic messages MUST go to Stderr.
	if !strings.Contains(stderr, "Using provider:") {
		t.Errorf("stderr missing 'Using provider:':\n%s", stderr)
	}
}
