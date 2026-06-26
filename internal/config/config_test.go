package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	configtest "github.com/danielxxomg/bak-cli/internal/config/testutil"
)

func TestLoadPath_NonExistent(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestLoadPath_ValidFile(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Write a v0.3.0 config to avoid migration.
	data := `{"schema_version":"0.3.0","providers":{"github":{"token":"ghp_test123","gist_id":"abc123"}}}`
	if err := os.WriteFile(cfgPath, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}
	if cfg.SchemaVersion != "0.3.0" {
		t.Errorf("SchemaVersion = %q, want 0.3.0", cfg.SchemaVersion)
	}
	githubCfg, ok := cfg.Providers["github"]
	if !ok {
		t.Fatal("expected providers.github")
	}
	if githubCfg.Token != "ghp_test123" {
		t.Errorf("Expected Token 'ghp_test123', got %q", githubCfg.Token)
	}
	if githubCfg.GistID != "abc123" {
		t.Errorf("Expected GistID 'abc123', got %q", githubCfg.GistID)
	}
}

func TestLoadPath_InvalidJSON(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestSave_RoundTrip(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Create and save a v0.3.0 config using the nested format.
	cfg := &Config{
		path:          cfgPath,
		SchemaVersion: "0.3.0",
		Providers: map[string]ProviderConfig{
			"github": {Token: "ghp_roundtrip", GistID: "gist_roundtrip"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Load and verify.
	loaded, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}
	if loaded.SchemaVersion != "0.3.0" {
		t.Errorf("SchemaVersion = %q, want 0.3.0", loaded.SchemaVersion)
	}
	githubCfg, ok := loaded.Providers["github"]
	if !ok {
		t.Fatal("expected providers.github")
	}
	if githubCfg.Token != "ghp_roundtrip" {
		t.Errorf("Expected Token 'ghp_roundtrip', got %q", githubCfg.Token)
	}
	if githubCfg.GistID != "gist_roundtrip" {
		t.Errorf("Expected GistID 'gist_roundtrip', got %q", githubCfg.GistID)
	}
}

func TestSave_CreatesDirectories(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestGet_KnownKeys(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.key, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

func TestGet_UnknownKey(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	cfg := &Config{}
	_, err := cfg.Get("unknown.key")
	if err == nil {
		t.Error("Expected error for unknown key, got nil")
	}
}

func TestSet_KnownKeys(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	cfg := &Config{path: cfgPath}

	// Set and verify using legacy keys (still supported).
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

	// Verify persistence via providers (Set writes to both flat and nested).
	loaded, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}
	githubCfg, ok := loaded.Providers["github"]
	if !ok {
		t.Fatal("expected providers.github after set")
	}
	if githubCfg.Token != "ghp_new" {
		t.Errorf("Expected persisted Token 'ghp_new', got %q", githubCfg.Token)
	}
	if githubCfg.GistID != "gist_new" {
		t.Errorf("Expected persisted GistID 'gist_new', got %q", githubCfg.GistID)
	}
}

func TestSet_UnknownKey(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	cfg := &Config{}
	err := cfg.Set("unknown.key", "value")
	if err == nil {
		t.Error("Expected error for unknown key, got nil")
	}
}

func TestDefaultPath(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestMarshalIndent(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestLoadPath_EmptyFile(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestLoadPath_WhitespaceOnly(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestLoadPath_EmptyObject(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestLoadPath_PartialJSON(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Write v0.3.0 JSON with only token set.
	if err := os.WriteFile(cfgPath, []byte(`{"schema_version":"0.3.0","providers":{"github":{"token":"ghp_partial"}}}`), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}
	githubCfg, ok := cfg.Providers["github"]
	if !ok {
		t.Fatal("expected providers.github")
	}
	if githubCfg.Token != "ghp_partial" {
		t.Errorf("Expected Token 'ghp_partial', got %q", githubCfg.Token)
	}
}

func TestLoadPath_ExtraFields(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Write v0.3.0 JSON with unknown fields — should be silently ignored.
	data := `{"schema_version":"0.3.0","providers":{"github":{"token":"ghp_extra"}},"unknown_field":"should_be_ignored"}`
	if err := os.WriteFile(cfgPath, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}
	githubCfg, ok := cfg.Providers["github"]
	if !ok {
		t.Fatal("expected providers.github")
	}
	if githubCfg.Token != "ghp_extra" {
		t.Errorf("Expected Token 'ghp_extra', got %q", githubCfg.Token)
	}
}

func TestSave_PreservesIndentation(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestSave_ExcludesOmitempty(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestConfig_ImmutabilityAfterLoad(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	original := `{"schema_version":"0.3.0","providers":{"github":{"token":"ghp_immutable","gist_id":"gist_immutable"}}}`
	if err := os.WriteFile(cfgPath, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}

	// Modify the returned config.
	cfg.Providers["github"] = ProviderConfig{Token: "ghp_modified"}

	// Reload from disk — should still have original value.
	reloaded, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error on reload: %v", err)
	}
	githubCfg, ok := reloaded.Providers["github"]
	if !ok {
		t.Fatal("expected providers.github on reload")
	}
	if githubCfg.Token != "ghp_immutable" {
		t.Errorf("Expected reloaded token 'ghp_immutable', got %q — modification leaked to disk", githubCfg.Token)
	}
}

func TestSave_ValidOutputFile(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

// --- ScheduleConfig round-trip ---

func TestSave_ScheduleConfig_RoundTrip(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	cfg := &Config{
		path:          cfgPath,
		SchemaVersion: "0.3.0",
		Profiles: map[string]ProfileConfig{
			"work": {
				Provider: "github-gist",
				Preset:   "full",
				Schedule: &ScheduleConfig{
					Enabled:  true,
					Interval: "daily",
				},
			},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Load and verify schedule config survived.
	loaded, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}

	pc, ok := loaded.Profiles["work"]
	if !ok {
		t.Fatal("expected profile 'work'")
	}
	if pc.Schedule == nil {
		t.Fatal("expected Schedule to be non-nil")
	}
	if !pc.Schedule.Enabled {
		t.Error("Schedule.Enabled should be true")
	}
	if pc.Schedule.Interval != "daily" {
		t.Errorf("Schedule.Interval = %q, want 'daily'", pc.Schedule.Interval)
	}
}

func TestSave_ScheduleConfig_DisabledOmitted(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Profile without schedule should have no schedule key in JSON.
	cfg := &Config{
		path:          cfgPath,
		SchemaVersion: "0.3.0",
		Profiles: map[string]ProfileConfig{
			"minimal": {
				Provider: "github-gist",
			},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	// Verify schedule key is not present in JSON output.
	if strings.Contains(string(data), "schedule") {
		t.Error("schedule key should be omitted when nil (omitempty)")
	}

	// Load and verify.
	loaded, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}
	pc, ok := loaded.Profiles["minimal"]
	if !ok {
		t.Fatal("expected profile 'minimal'")
	}
	if pc.Schedule != nil {
		t.Error("Schedule should be nil for profile without scheduling")
	}
}

func TestSave_ScheduleConfig_MultipleIntervals(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	intervals := []string{"daily", "weekly", "every-12h", "every-6h"}

	for _, iv := range intervals { //nolint:paralleltest // subtests share table/struct state
		t.Run(iv, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			dir := t.TempDir()
			cfgPath := filepath.Join(dir, "config.json")

			cfg := &Config{
				path:          cfgPath,
				SchemaVersion: "0.3.0",
				Profiles: map[string]ProfileConfig{
					"test": {
						Provider: "github-gist",
						Schedule: &ScheduleConfig{
							Enabled:  true,
							Interval: iv,
						},
					},
				},
			}
			if err := cfg.Save(); err != nil {
				t.Fatalf("Save() error: %v", err)
			}

			loaded, err := LoadPath(cfgPath)
			if err != nil {
				t.Fatalf("LoadPath() error: %v", err)
			}

			sc := loaded.Profiles["test"].Schedule
			if sc == nil {
				t.Fatal("Schedule should not be nil")
			}
			if sc.Interval != iv {
				t.Errorf("Interval = %q, want %q", sc.Interval, iv)
			}
		})
	}
}

func TestLoad_ViaEnvVar(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// Use configtest.SetConfigHome so Load() finds our config on every OS.
	dir := t.TempDir()
	configtest.SetConfigHome(t, dir)

	// Use DefaultPath() so the file lands where Load() actually looks,
	// which differs per OS (e.g. macOS: $HOME/Library/Application Support/bak).
	cfgPath, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	cfgDir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Use v0.3.0 format to avoid migration during test.
	data := `{"schema_version":"0.3.0","providers":{"github":{"token":"ghp_load_test","gist_id":"gist_load"}}}`
	if err := os.WriteFile(cfgPath, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	githubCfg, ok := cfg.Providers["github"]
	if !ok {
		t.Fatal("expected providers.github")
	}
	if githubCfg.Token != "ghp_load_test" {
		t.Errorf("Expected Token 'ghp_load_test', got %q", githubCfg.Token)
	}
	if githubCfg.GistID != "gist_load" {
		t.Errorf("Expected GistID 'gist_load', got %q", githubCfg.GistID)
	}
}

func TestLoad_NonExistentConfig(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// Load with a config dir that has no config.json should return defaults.
	dir := t.TempDir()
	configtest.SetConfigHome(t, dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.GitHubToken != "" {
		t.Errorf("Expected empty GitHubToken, got %q", cfg.GitHubToken)
	}
}

func TestSave_EmptyPath(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// Save with empty path should use DefaultPath().
	// We override env vars so DefaultPath() goes to our temp dir.
	dir := t.TempDir()
	configtest.SetConfigHome(t, dir)

	cfg := &Config{GitHubToken: "ghp_empty_path"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Verify the file was created at the default path.
	defaultPath, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath() error: %v", err)
	}
	if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		t.Errorf("Config file should exist at default path %s", defaultPath)
	}
}

// =============================================================================
// Settings tests — RED (Settings struct does not exist yet)
// =============================================================================

func TestSettings_Defaults(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	s := DefaultSettings()

	if s.DefaultPreset != "quick" {
		t.Errorf("DefaultPreset = %q, want %q", s.DefaultPreset, "quick")
	}
	if s.AutoSync != false {
		t.Errorf("AutoSync = %v, want false", s.AutoSync)
	}
	if s.MaxFileSize != 1048576 {
		t.Errorf("MaxFileSize = %d, want %d", s.MaxFileSize, 1048576)
	}
	if s.ConfirmDestructive == nil || *s.ConfirmDestructive != true {
		t.Errorf("ConfirmDestructive = %v, want true", s.ConfirmDestructive)
	}
	if s.VerboseDefault != false {
		t.Errorf("VerboseDefault = %v, want false", s.VerboseDefault)
	}
	if s.DefaultProvider != "" {
		t.Errorf("DefaultProvider = %q, want empty", s.DefaultProvider)
	}
	if len(s.ExcludePatterns) != 0 {
		t.Errorf("ExcludePatterns = %v, want nil/empty", s.ExcludePatterns)
	}
}

func TestSettings_SaveLoadRoundTrip(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	cfg := &Config{
		path:          cfgPath,
		SchemaVersion: "0.3.0",
		Settings: Settings{
			DefaultPreset:      "full",
			AutoSync:           true,
			ExcludePatterns:    []string{"node_modules", "*.log"},
			MaxFileSize:        2097152,
			ConfirmDestructive: boolPtr(false),
			VerboseDefault:     true,
			DefaultProvider:    "github",
		},
		ActiveProfile: "work",
	}

	// Save.
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Load and verify.
	loaded, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}

	s := loaded.Settings
	if s.DefaultPreset != "full" {
		t.Errorf("DefaultPreset = %q, want %q", s.DefaultPreset, "full")
	}
	if !s.AutoSync {
		t.Error("AutoSync = false, want true")
	}
	if len(s.ExcludePatterns) != 2 || s.ExcludePatterns[0] != "node_modules" {
		t.Errorf("ExcludePatterns = %v, want [node_modules *.log]", s.ExcludePatterns)
	}
	if s.MaxFileSize != 2097152 {
		t.Errorf("MaxFileSize = %d, want %d", s.MaxFileSize, 2097152)
	}
	if s.ConfirmDestructive == nil || *s.ConfirmDestructive != false {
		t.Errorf("ConfirmDestructive = %v, want false", s.ConfirmDestructive)
	}
	if !s.VerboseDefault {
		t.Error("VerboseDefault = false, want true")
	}
	if s.DefaultProvider != "github" {
		t.Errorf("DefaultProvider = %q, want %q", s.DefaultProvider, "github")
	}
	if loaded.ActiveProfile != "work" {
		t.Errorf("ActiveProfile = %q, want %q", loaded.ActiveProfile, "work")
	}
}

func TestLoad_AppliesDefaultsWhenMissing(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Write a config WITHOUT settings key — LoadPath must apply DefaultSettings().
	data := `{"schema_version":"0.3.0","providers":{"github":{"token":"t"}}}`
	if err := os.WriteFile(cfgPath, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}

	// Settings fields MUST have DefaultSettings values applied.
	if cfg.Settings.DefaultPreset != "quick" {
		t.Errorf("DefaultPreset = %q, want %q", cfg.Settings.DefaultPreset, "quick")
	}
	if cfg.Settings.MaxFileSize != 1048576 {
		t.Errorf("MaxFileSize = %d, want %d", cfg.Settings.MaxFileSize, 1048576)
	}
	if cfg.Settings.ConfirmDestructive == nil || *cfg.Settings.ConfirmDestructive != true {
		t.Errorf("ConfirmDestructive = %v, want true", cfg.Settings.ConfirmDestructive)
	}
	if cfg.Settings.AutoSync != false {
		t.Errorf("AutoSync = %v, want false", cfg.Settings.AutoSync)
	}

	// Non-Settings fields MUST be preserved (providers, etc.).
	if cfg.Providers["github"].Token != "t" {
		t.Errorf("Providers[github].Token = %q, want %q", cfg.Providers["github"].Token, "t")
	}
}

func TestSettings_LoadDefaultsWhenMissing(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Write a config WITHOUT settings — settings should default on load.
	data := `{"schema_version":"0.3.0","providers":{"github":{"token":"t"}}}`
	if err := os.WriteFile(cfgPath, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}

	// Settings fields should have DefaultSettings applied.
	if cfg.Settings.DefaultPreset != "quick" {
		t.Errorf("DefaultPreset = %q, want %q", cfg.Settings.DefaultPreset, "quick")
	}
	if cfg.ActiveProfile != "" {
		t.Errorf("ActiveProfile = %q, want empty", cfg.ActiveProfile)
	}
}

// triangulate: existing non-zero settings are NOT overwritten.
func TestLoad_DefaultsRespectExistingSettings(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Config WITH an explicit non-zero settings section.
	data := `{"schema_version":"0.3.0","providers":{"github":{"token":"t"}},"settings":{"default_preset":"full","max_file_size":2097152}}`
	if err := os.WriteFile(cfgPath, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}

	// Non-zero user settings MUST be preserved.
	if cfg.Settings.DefaultPreset != "full" {
		t.Errorf("DefaultPreset = %q, want %q (should NOT be overwritten)", cfg.Settings.DefaultPreset, "full")
	}
	if cfg.Settings.MaxFileSize != 2097152 {
		t.Errorf("MaxFileSize = %d, want %d (should NOT be overwritten)", cfg.Settings.MaxFileSize, 2097152)
	}
	// ConfirmDestructive was NOT explicitly set, so it should default to true.
	if cfg.Settings.ConfirmDestructive == nil || *cfg.Settings.ConfirmDestructive != true {
		t.Errorf("ConfirmDestructive = %v, want true (default applies since not set)", cfg.Settings.ConfirmDestructive)
	}
}

// boolPtr returns a pointer to a bool — useful for constructing Settings
// structs with *bool fields in table-driven tests.
func boolPtr(b bool) *bool {
	return &b
}

// =============================================================================
// Settings-key helpers — getSettingsField / setSettingsField / parseBool
// =============================================================================

// TestGetSettingsField covers every documented settings alias resolving to
// its canonical value through the Get("settings.*") path.
func TestGetSettingsField(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	confirm := false
	cfg := &Config{
		Settings: Settings{
			DefaultPreset:      "full",
			AutoSync:           true,
			MaxFileSize:        2048,
			VerboseDefault:     true,
			DefaultProvider:    "github",
			ConfirmDestructive: &confirm,
		},
	}

	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "default_preset", key: "settings.default_preset", want: "full"},
		{name: "auto_sync", key: "settings.auto_sync", want: "true"},
		{name: "max_file_size", key: "settings.max_file_size", want: "2048"},
		{name: "verbose_default", key: "settings.verbose_default", want: "true"},
		{name: "default_provider", key: "settings.default_provider", want: "github"},
		{name: "confirm_destructive set false", key: "settings.confirm_destructive", want: "false"},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			got, err := cfg.Get(tt.key)
			if err != nil {
				t.Fatalf("Get(%q) error: %v", tt.key, err)
			}
			if got != tt.want {
				t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

// TestGetSettingsField_ConfirmDestructiveNilDefault covers the nil branch:
// when ConfirmDestructive is unset, the getter returns the "true" default.
func TestGetSettingsField_ConfirmDestructiveNilDefault(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	cfg := &Config{Settings: Settings{ConfirmDestructive: nil}}
	got, err := cfg.Get("settings.confirm_destructive")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got != "true" {
		t.Errorf("Get(settings.confirm_destructive) = %q, want %q (nil default)", got, "true")
	}
}

func TestGetSettingsField_UnknownKey(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	cfg := &Config{}
	_, err := cfg.Get("settings.nonexistent")
	if err == nil {
		t.Error("expected error for unknown settings key, got nil")
	}
}

// TestSetSettingsField covers writing each settings field through the
// Set("settings.*") path and verifying the in-memory Config is updated.
func TestSetSettingsField(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name   string
		key    string
		value  string
		verify func(t *testing.T, c *Config)
	}{
		{
			name:  "default_preset",
			key:   "settings.default_preset",
			value: "full",
			verify: func(t *testing.T, c *Config) {
				if c.Settings.DefaultPreset != "full" {
					t.Errorf("DefaultPreset = %q, want full", c.Settings.DefaultPreset)
				}
			},
		},
		{
			name:  "auto_sync",
			key:   "settings.auto_sync",
			value: "true",
			verify: func(t *testing.T, c *Config) {
				if !c.Settings.AutoSync {
					t.Error("AutoSync = false, want true")
				}
			},
		},
		{
			name:  "max_file_size",
			key:   "settings.max_file_size",
			value: "4096",
			verify: func(t *testing.T, c *Config) {
				if c.Settings.MaxFileSize != 4096 {
					t.Errorf("MaxFileSize = %d, want 4096", c.Settings.MaxFileSize)
				}
			},
		},
		{
			name:  "verbose_default",
			key:   "settings.verbose_default",
			value: "1",
			verify: func(t *testing.T, c *Config) {
				if !c.Settings.VerboseDefault {
					t.Error("VerboseDefault = false, want true")
				}
			},
		},
		{
			name:  "default_provider",
			key:   "settings.default_provider",
			value: "codeberg",
			verify: func(t *testing.T, c *Config) {
				if c.Settings.DefaultProvider != "codeberg" {
					t.Errorf("DefaultProvider = %q, want codeberg", c.Settings.DefaultProvider)
				}
			},
		},
		{
			name:  "confirm_destructive",
			key:   "settings.confirm_destructive",
			value: "false",
			verify: func(t *testing.T, c *Config) {
				if c.Settings.ConfirmDestructive == nil || *c.Settings.ConfirmDestructive != false {
					t.Errorf("ConfirmDestructive = %v, want false", c.Settings.ConfirmDestructive)
				}
			},
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			dir := t.TempDir()
			cfg := &Config{path: filepath.Join(dir, "config.json")}
			if err := cfg.Set(tt.key, tt.value); err != nil {
				t.Fatalf("Set(%q, %q) error: %v", tt.key, tt.value, err)
			}
			tt.verify(t, cfg)

			// The value must persist to disk and reload identically.
			loaded, err := LoadPath(cfg.path)
			if err != nil {
				t.Fatalf("LoadPath error: %v", err)
			}
			got, err := loaded.Get(tt.key)
			if err != nil {
				t.Fatalf("reloaded Get(%q) error: %v", tt.key, err)
			}
			want, err := cfg.Get(tt.key)
			if err != nil {
				t.Fatalf("Get(%q) error: %v", tt.key, err)
			}
			if got != want {
				t.Errorf("reloaded %q = %q, want %q", tt.key, got, want)
			}
		})
	}
}

func TestSetSettingsField_InvalidBool(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	cfg := &Config{path: filepath.Join(dir, "config.json")}

	boolKeys := []string{"settings.auto_sync", "settings.verbose_default", "settings.confirm_destructive"}
	for _, key := range boolKeys { //nolint:paralleltest // subtests share table/struct state
		t.Run(key, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			if err := cfg.Set(key, "notabool"); err == nil {
				t.Errorf("Set(%q, %q) expected error for invalid bool, got nil", key, "notabool")
			}
		})
	}
}

func TestSetSettingsField_InvalidMaxFileSize(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	cfg := &Config{path: filepath.Join(dir, "config.json")}
	if err := cfg.Set("settings.max_file_size", "notanumber"); err == nil {
		t.Error("Set(settings.max_file_size, notanumber) expected error, got nil")
	}
}

func TestSetSettingsField_UnknownKey(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	cfg := &Config{}
	if err := cfg.Set("settings.nonexistent", "x"); err == nil {
		t.Error("expected error for unknown settings key, got nil")
	}
}

// TestParseBool locks the accepted boolean token set: true/false, 1/0,
// yes/no (all case-insensitive). Any other token returns an error.
func TestParseBool(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name    string
		input   string
		want    bool
		wantErr bool
	}{
		{name: "true", input: "true", want: true},
		{name: "1", input: "1", want: true},
		{name: "yes accepted", input: "yes", want: true},
		{name: "mixed case TRUE", input: "TRUE", want: true},
		{name: "false", input: "false", want: false},
		{name: "0", input: "0", want: false},
		{name: "no accepted", input: "no", want: false},
		{name: "numeric 2 rejected", input: "2", wantErr: true},
		{name: "empty rejected", input: "", wantErr: true},
		{name: "arbitrary string rejected", input: "maybe", wantErr: true},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			got, err := parseBool(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseBool(%q) err = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("parseBool(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// =============================================================================
// Load / Save / Get / Set error paths
// =============================================================================

func TestLoadPath_UnreadableFile(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	if runtime.GOOS == "windows" {
		t.Skip("chmod 000 does not block reads on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses chmod 000 permissions")
	}
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(cfgPath, []byte(`{"schema_version":"0.3.0"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(cfgPath, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cerr := os.Chmod(cfgPath, 0644); cerr != nil {
			t.Logf("cleanup chmod: %v", cerr)
		}
	})

	_, err := LoadPath(cfgPath)
	if err == nil {
		t.Error("expected error for unreadable config file, got nil")
	}
}

func TestSave_UnwritableDirectory(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	if runtime.GOOS == "windows" {
		t.Skip("chmod 0500 does not block writes on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses chmod permissions")
	}
	dir := t.TempDir()
	roDir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(roDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(roDir, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cerr := os.Chmod(roDir, 0755); cerr != nil {
			t.Logf("cleanup chmod: %v", cerr)
		}
	})

	// Target a NEW subdirectory under the read-only dir so MkdirAll fails.
	cfg := &Config{path: filepath.Join(roDir, "sub", "config.json")}
	if err := cfg.Save(); err == nil {
		t.Error("expected error saving under read-only directory, got nil")
	}
}

func TestGet_NestedProviderKeys(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	cfg := &Config{
		Providers: map[string]ProviderConfig{
			"github": {Token: "tok-gh", GistID: "gist-gh"},
			"gitea":  {Repo: "me/repo", BaseURL: "https://gitea.example", Remote: "origin"},
		},
	}

	tests := []struct {
		key  string
		want string
	}{
		{"providers.github.token", "tok-gh"},
		{"providers.github.gist_id", "gist-gh"},
		{"providers.gitea.repo", "me/repo"},
		{"providers.gitea.base_url", "https://gitea.example"},
		{"providers.gitea.remote", "origin"},
	}
	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.key, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			got, err := cfg.Get(tt.key)
			if err != nil {
				t.Fatalf("Get(%q) error: %v", tt.key, err)
			}
			if got != tt.want {
				t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestGet_NestedProvider_Errors(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	cfg := &Config{Providers: map[string]ProviderConfig{"github": {Token: "t"}}}

	t.Run("unconfigured provider", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		if _, err := cfg.Get("providers.codeberg.token"); err == nil {
			t.Error("expected error for unconfigured provider, got nil")
		}
	})
	t.Run("unsupported field", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		if _, err := cfg.Get("providers.github.bogus"); err == nil {
			t.Error("expected error for unsupported provider field, got nil")
		}
	})
}

func TestSet_NestedProviderKeys(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	cfg := &Config{path: filepath.Join(dir, "config.json")}

	pairs := []struct{ key, value string }{
		{"providers.codeberg.token", "tok-cb"},
		{"providers.codeberg.repo", "user/backup"},
		{"providers.rclone.remote", "gdrive"},
		{"providers.gitea.base_url", "https://gitea.local"},
	}
	for _, p := range pairs {
		if err := cfg.Set(p.key, p.value); err != nil {
			t.Fatalf("Set(%q, %q) error: %v", p.key, p.value, err)
		}
	}

	loaded, err := LoadPath(cfg.path)
	if err != nil {
		t.Fatalf("LoadPath error: %v", err)
	}
	for _, p := range pairs {
		got, err := loaded.Get(p.key)
		if err != nil {
			t.Fatalf("reloaded Get(%q) error: %v", p.key, err)
		}
		if got != p.value {
			t.Errorf("reloaded %q = %q, want %q", p.key, got, p.value)
		}
	}
}

func TestSet_NestedProvider_UnsupportedField(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	cfg := &Config{path: filepath.Join(dir, "config.json")}
	if err := cfg.Set("providers.github.bogus", "x"); err == nil {
		t.Error("expected error for unsupported provider field, got nil")
	}
}

func TestSettings_ActiveProfileRoundTrip(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	cfg := &Config{
		path:          cfgPath,
		SchemaVersion: "0.3.0",
		ActiveProfile: "personal",
		Profiles: map[string]ProfileConfig{
			"personal": {Preset: "quick", Provider: "github"},
			"work":     {Preset: "full", Provider: "github"},
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath() error: %v", err)
	}

	if loaded.ActiveProfile != "personal" {
		t.Errorf("ActiveProfile = %q, want %q", loaded.ActiveProfile, "personal")
	}
	if len(loaded.Profiles) != 2 {
		t.Errorf("Profiles count = %d, want 2", len(loaded.Profiles))
	}
}
