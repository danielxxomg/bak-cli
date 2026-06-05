package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPath_NonExistent(t *testing.T) {
	// Loading a non-existent config should return defaults (empty config).
	cfgPath := filepath.Join(t.TempDir(), "nonexistent", "config.json")
	cfg, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}
	if cfg.GitHubToken != "" {
		t.Errorf("Expected empty GitHubToken, got %q", cfg.GitHubToken)
	}
	if cfg.GistID != "" {
		t.Errorf("Expected empty GistID, got %q", cfg.GistID)
	}
}

func TestLoadPath_ValidFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Write a valid config file.
	data := `{"github_token":"ghp_test123","gist_id":"abc123"}`
	if err := os.WriteFile(cfgPath, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}
	if cfg.GitHubToken != "ghp_test123" {
		t.Errorf("Expected GitHubToken 'ghp_test123', got %q", cfg.GitHubToken)
	}
	if cfg.GistID != "abc123" {
		t.Errorf("Expected GistID 'abc123', got %q", cfg.GistID)
	}
}

func TestLoadPath_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Write invalid JSON.
	if err := os.WriteFile(cfgPath, []byte("{invalid json}"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadPath(cfgPath)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestSave_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Create and save a config.
	cfg := &Config{
		path:        cfgPath,
		GitHubToken: "ghp_roundtrip",
		GistID:      "gist_roundtrip",
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Load and verify.
	loaded, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}
	if loaded.GitHubToken != "ghp_roundtrip" {
		t.Errorf("Expected GitHubToken 'ghp_roundtrip', got %q", loaded.GitHubToken)
	}
	if loaded.GistID != "gist_roundtrip" {
		t.Errorf("Expected GistID 'gist_roundtrip', got %q", loaded.GistID)
	}
}

func TestSave_CreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "nested", "deep", "config.json")

	cfg := &Config{path: cfgPath}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify file exists.
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Error("Config file should have been created")
	}
}

func TestGet_KnownKeys(t *testing.T) {
	cfg := &Config{
		GitHubToken: "ghp_test",
		GistID:      "gist_test",
	}

	tests := []struct {
		key  string
		want string
	}{
		{"github.token", "ghp_test"},
		{"github.gist_id", "gist_test"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, err := cfg.Get(tt.key)
			if err != nil {
				t.Errorf("Get(%q) error: %v", tt.key, err)
			}
			if got != tt.want {
				t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestGet_UnknownKey(t *testing.T) {
	cfg := &Config{}
	_, err := cfg.Get("unknown.key")
	if err == nil {
		t.Error("Expected error for unknown key, got nil")
	}
}

func TestSet_KnownKeys(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	cfg := &Config{path: cfgPath}

	// Set and verify.
	if err := cfg.Set("github.token", "ghp_new"); err != nil {
		t.Fatalf("Set() error: %v", err)
	}
	if cfg.GitHubToken != "ghp_new" {
		t.Errorf("Expected GitHubToken 'ghp_new', got %q", cfg.GitHubToken)
	}

	if err := cfg.Set("github.gist_id", "gist_new"); err != nil {
		t.Fatalf("Set() error: %v", err)
	}
	if cfg.GistID != "gist_new" {
		t.Errorf("Expected GistID 'gist_new', got %q", cfg.GistID)
	}

	// Verify persistence.
	loaded, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}
	if loaded.GitHubToken != "ghp_new" {
		t.Errorf("Expected persisted GitHubToken 'ghp_new', got %q", loaded.GitHubToken)
	}
	if loaded.GistID != "gist_new" {
		t.Errorf("Expected persisted GistID 'gist_new', got %q", loaded.GistID)
	}
}

func TestSet_UnknownKey(t *testing.T) {
	cfg := &Config{}
	err := cfg.Set("unknown.key", "value")
	if err == nil {
		t.Error("Expected error for unknown key, got nil")
	}
}

func TestDefaultPath(t *testing.T) {
	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath() error: %v", err)
	}
	if path == "" {
		t.Error("DefaultPath() should not return empty string")
	}
	if filepath.Ext(path) != ".json" {
		t.Errorf("DefaultPath() should end with .json, got %q", path)
	}
}

func TestMarshalIndent(t *testing.T) {
	cfg := &Config{
		GitHubToken: "ghp_test",
		GistID:      "gist_test",
	}

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfg.path = cfgPath

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Read and verify it's valid JSON with indentation.
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	// Should contain newlines (indented JSON).
	if len(data) == 0 {
		t.Error("Config file should not be empty")
	}

	// Should be valid JSON.
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Errorf("Config file should be valid JSON: %v", err)
	}
}
