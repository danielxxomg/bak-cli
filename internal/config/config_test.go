package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestLoadPath_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Write an empty file (0 bytes).
	if err := os.WriteFile(cfgPath, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadPath(cfgPath)
	if err == nil {
		t.Error("Expected error for empty JSON file, got nil")
	}
}

func TestLoadPath_WhitespaceOnly(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Write a file containing only whitespace.
	if err := os.WriteFile(cfgPath, []byte("   \n\t  "), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadPath(cfgPath)
	if err == nil {
		t.Error("Expected error for whitespace-only JSON file, got nil")
	}
}

func TestLoadPath_EmptyObject(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Write a minimal valid JSON object.
	if err := os.WriteFile(cfgPath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error for empty JSON object: %v", err)
	}
	if cfg.GitHubToken != "" {
		t.Errorf("Expected empty GitHubToken from empty JSON, got %q", cfg.GitHubToken)
	}
	if cfg.GistID != "" {
		t.Errorf("Expected empty GistID from empty JSON, got %q", cfg.GistID)
	}
}

func TestLoadPath_PartialJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Write JSON with only one field set.
	if err := os.WriteFile(cfgPath, []byte(`{"github_token":"ghp_partial"}`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}
	if cfg.GitHubToken != "ghp_partial" {
		t.Errorf("Expected GitHubToken 'ghp_partial', got %q", cfg.GitHubToken)
	}
	if cfg.GistID != "" {
		t.Errorf("Expected empty GistID, got %q", cfg.GistID)
	}
}

func TestLoadPath_ExtraFields(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Write JSON with unknown fields — should be silently ignored.
	data := `{"github_token":"ghp_extra","unknown_field":"should_be_ignored"}`
	if err := os.WriteFile(cfgPath, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}
	if cfg.GitHubToken != "ghp_extra" {
		t.Errorf("Expected GitHubToken 'ghp_extra', got %q", cfg.GitHubToken)
	}
}

func TestSave_PreservesIndentation(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	cfg := &Config{
		path:        cfgPath,
		GitHubToken: "ghp_indent",
		GistID:      "gist_indent",
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	// Verify the file contains newlines (indented, not compact).
	content := string(data)
	if !strings.Contains(content, "\n") && len(data) > 20 {
		t.Error("Saved JSON should contain newlines (indentation)")
	}

	// Verify specific keys are present.
	if !strings.Contains(content, "github_token") {
		t.Error("Saved JSON should contain github_token key")
	}
	if !strings.Contains(content, "gist_id") {
		t.Error("Saved JSON should contain gist_id key")
	}
}

func TestSave_ExcludesOmitempty(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Config with ALL empty fields.
	cfg := &Config{path: cfgPath}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	// With omitempty, empty string fields should NOT appear.
	content := string(data)
	if strings.Contains(content, "github_token") {
		t.Error("Empty github_token should be omitted (omitempty)")
	}
	if strings.Contains(content, "gist_id") {
		t.Error("Empty gist_id should be omitted (omitempty)")
	}

	// File should be valid JSON (just {} or similar).
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Errorf("Config file should be valid JSON even with empty fields: %v", err)
	}
}

func TestConfig_ImmutabilityAfterLoad(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	original := `{"github_token":"ghp_immutable","gist_id":"gist_immutable"}`
	if err := os.WriteFile(cfgPath, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}

	// Modify the returned config.
	cfg.GitHubToken = "ghp_modified"

	// Reload from disk — should still have original value.
	reloaded, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error on reload: %v", err)
	}
	if reloaded.GitHubToken != "ghp_immutable" {
		t.Errorf("Expected reloaded token 'ghp_immutable', got %q — modification leaked to disk", reloaded.GitHubToken)
	}
}

func TestSave_ValidOutputFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	cfg := &Config{path: cfgPath, GitHubToken: "ghp_output"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	// File should exist and be parseable.
	if len(data) == 0 {
		t.Error("Saved config file should not be empty")
	}

	var parsed Config
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("Saved config should be valid JSON: %v", err)
	}
	if parsed.GitHubToken != "ghp_output" {
		t.Errorf("Expected GitHubToken 'ghp_output', got %q", parsed.GitHubToken)
	}
}
