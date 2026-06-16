package actions

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/cloud"
	"github.com/danielxxomg/bak-cli/internal/config"
)

// TestCloudSync_PushPullRoundTrip exercises the full push→pull cycle
// through a MockProvider: create a backup directory with known files,
// push it through the mock, pull to a new home, and verify the extracted
// files match the originals.
func TestCloudSync_PushPullRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
	}{
		{
			name: "single file",
			files: map[string]string{
				"opencode.json": `{"version":"1.0","theme":"dark"}`,
			},
		},
		{
			name: "multiple files",
			files: map[string]string{
				"opencode.json":  `{"version":"2.0"}`,
				"AGENTS.md":      "# AGENTS\n\nRules here.",
				"settings.yml":   "key: value\n",
				"sub/plugin.cfg": "enabled: true\n",
			},
		},
		{
			name:  "empty backup dir",
			files: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// --- Setup: create source backup directory ---
			home1 := t.TempDir()
			backupID := "20260101-120000"
			backupPath := filepath.Join(home1, ".bak", "backups", backupID)
			if err := os.MkdirAll(backupPath, 0755); err != nil {
				t.Fatalf("mkdir backup dir: %v", err)
			}

			// Write a manifest so PushAction's Stat check passes.
			manifestData := fmt.Sprintf(`{"id":"%s","version":"1.0"}`, backupID)
			if err := os.WriteFile(filepath.Join(backupPath, "manifest.json"), []byte(manifestData), 0644); err != nil {
				t.Fatalf("write manifest: %v", err)
			}

			// Write the fixture files that we'll verify after round-trip.
			for name, content := range tt.files {
				fullPath := filepath.Join(backupPath, name)
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					t.Fatalf("mkdir parent for %s: %v", name, err)
				}
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatalf("write %s: %v", name, err)
				}
			}

			// --- Setup: MockProvider with shared state ---
			// The mock stores the base64-encoded archive on push and returns
			// it on pull, simulating a cloud backend round-trip.
			var storedArchive string
			mockProvider := &MockProvider{
				MockName: "mock-gist",
				PushFn: func(archive []byte, meta cloud.PushMeta) (string, error) {
					storedArchive = base64.StdEncoding.EncodeToString(archive)
					return "mock-gist-id-123", nil
				},
				PullFn: func(id string) ([]byte, error) {
					if id != "mock-gist-id-123" {
						return nil, fmt.Errorf("not found: %s", id)
					}
					return []byte(storedArchive), nil
				},
			}

			factory := &MockProviderFactory{
				Providers: map[string]cloud.Provider{
					"mock-gist": mockProvider,
				},
			}

			// --- Push from home1 ---
			pushAction := &PushAction{
				FS:         newHomeFS(home1),
				Provider:   "mock-gist",
				Stdout:     os.Stdout,
				Stderr:     os.Stderr,
				Factory:    factory,
				HostnameFn: func() (string, error) { return "testbox", nil },
			}

			if err := pushAction.Run([]string{backupID}); err != nil {
				t.Fatalf("Push: %v", err)
			}

			if storedArchive == "" {
				t.Fatal("mock provider did not store any archive — Push was not called")
			}

			// --- Pull to home2 ---
			home2 := t.TempDir()
			pullAction := &PullAction{
				FS: newHomeFS(home2),
				ConfigLoader: func() (*config.Config, error) {
					return &config.Config{}, nil
				},
				Provider: "mock-gist",
				Factory:  factory,
			}

			if err := pullAction.Run([]string{"mock-gist-id-123"}); err != nil {
				t.Fatalf("Pull: %v", err)
			}

			// --- Verify: extracted files match originals ---
			verifyExtractedFiles(t, home2, tt.files)
		})
	}
}

// TestCloudSync_PushInvalidToken verifies that a PushAction with a
// mock provider that returns a 401-style error produces an error
// containing "401" or "unauthorized".
func TestCloudSync_PushInvalidToken(t *testing.T) {
	home := t.TempDir()

	// Create a minimal backup directory so PushAction can stat it.
	backupID := "20260101-120000"
	backupPath := filepath.Join(home, ".bak", "backups", backupID)
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backupPath, "manifest.json"), []byte(`{"id":"20260101-120000"}`), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	mockProvider := &MockProvider{
		MockName: "mock-gist",
		PushFn: func(archive []byte, meta cloud.PushMeta) (string, error) {
			return "", fmt.Errorf("401 Unauthorized: invalid token")
		},
	}

	factory := &MockProviderFactory{
		Providers: map[string]cloud.Provider{
			"mock-gist": mockProvider,
		},
	}

	pushAction := &PushAction{
		FS:         newHomeFS(home),
		Provider:   "mock-gist",
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Factory:    factory,
		HostnameFn: func() (string, error) { return "testbox", nil },
	}

	err := pushAction.Run([]string{backupID})
	if err == nil {
		t.Fatal("expected error for invalid token, got nil")
	}

	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "401") && !strings.Contains(errStr, "unauthorized") {
		t.Errorf("error should contain '401' or 'unauthorized', got: %v", err)
	}
}

// TestCloudSync_Pull_NotFound verifies that a PullAction with a
// mock provider that returns a not-found error produces an error
// containing "not found" or "404".
func TestCloudSync_Pull_NotFound(t *testing.T) {
	home := t.TempDir()

	mockProvider := &MockProvider{
		MockName: "mock-gist",
		PullFn: func(id string) ([]byte, error) {
			return nil, fmt.Errorf("not found: 404 the gist %q does not exist", id)
		},
	}

	factory := &MockProviderFactory{
		Providers: map[string]cloud.Provider{
			"mock-gist": mockProvider,
		},
	}

	pullAction := &PullAction{
		FS: newHomeFS(home),
		ConfigLoader: func() (*config.Config, error) {
			return &config.Config{}, nil
		},
		Provider: "mock-gist",
		Factory:  factory,
	}

	err := pullAction.Run([]string{"nonexistent-id"})
	if err == nil {
		t.Fatal("expected error for nonexistent gist, got nil")
	}

	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "not found") && !strings.Contains(errStr, "404") {
		t.Errorf("error should contain 'not found' or '404', got: %v", err)
	}
}
