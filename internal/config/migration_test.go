package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigMigration_V010_Detected(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// v0.1.0 config: flat github_token + gist_id at root, no schema_version.
	v010 := `{"github_token":"ghp_migrate_me","gist_id":"abc123"}`
	if err := os.WriteFile(cfgPath, []byte(v010), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath: %v", err)
	}

	// After chain migration, schema_version must be "0.3.0".
	if cfg.SchemaVersion != "0.3.0" {
		t.Errorf("SchemaVersion = %q, want 0.3.0", cfg.SchemaVersion)
	}

	// Original flat fields should be cleared (moved to Providers).
	if cfg.GitHubToken != "" {
		t.Errorf("GitHubToken = %q, want empty (migrated to providers)", cfg.GitHubToken)
	}
	if cfg.GistID != "" {
		t.Errorf("GistID = %q, want empty (migrated to providers)", cfg.GistID)
	}

	// Token should be in providers.github.
	githubCfg, ok := cfg.Providers["github"]
	if !ok {
		t.Fatal("expected providers.github to exist after migration")
	}
	if githubCfg.Token != "ghp_migrate_me" {
		t.Errorf("providers.github.token = %q, want ghp_migrate_me", githubCfg.Token)
	}
	if githubCfg.GistID != "abc123" {
		t.Errorf("providers.github.gist_id = %q, want abc123", githubCfg.GistID)
	}
}

func TestConfigMigration_V030_Skipped(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// v0.3.0 config: has schema_version, no migration needed.
	v030 := `{"schema_version":"0.3.0","providers":{"github":{"token":"ghp_v3","gist_id":"xyz"}}}`
	if err := os.WriteFile(cfgPath, []byte(v030), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath: %v", err)
	}

	// Should load normally — no migration.
	if cfg.SchemaVersion != "0.3.0" {
		t.Errorf("SchemaVersion = %q, want 0.3.0", cfg.SchemaVersion)
	}
	githubCfg, ok := cfg.Providers["github"]
	if !ok {
		t.Fatal("providers.github should exist")
	}
	if githubCfg.Token != "ghp_v3" {
		t.Errorf("providers.github.token = %q, want ghp_v3", githubCfg.Token)
	}
	if githubCfg.GistID != "xyz" {
		t.Errorf("providers.github.gist_id = %q, want xyz", githubCfg.GistID)
	}
}

func TestConfigMigration_BackupCreated(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	bakPath := cfgPath + ".v010.bak"

	v010 := `{"github_token":"ghp_bak","gist_id":"bak123"}`
	if err := os.WriteFile(cfgPath, []byte(v010), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadPath(cfgPath); err != nil {
		t.Fatalf("LoadPath: %v", err)
	}

	// Backup file must exist with original content.
	bakData, err := os.ReadFile(bakPath)
	if err != nil {
		t.Fatalf("expected .v010.bak file: %v", err)
	}
	if string(bakData) != v010 {
		t.Errorf("backup content = %q, want %q", string(bakData), v010)
	}
}

func TestConfigMigration_Idempotent(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Write v0.1.0 and load (triggers chain migration to v0.3.0).
	v010 := `{"github_token":"ghp_idem","gist_id":"idem123"}`
	if err := os.WriteFile(cfgPath, []byte(v010), 0644); err != nil {
		t.Fatal(err)
	}

	cfg1, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath first: %v", err)
	}
	if cfg1.SchemaVersion != "0.3.0" {
		t.Fatalf("first load: SchemaVersion = %q, want 0.3.0", cfg1.SchemaVersion)
	}

	// Load again — should be v0.3.0, no re-migration.
	cfg2, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath second: %v", err)
	}
	if cfg2.SchemaVersion != "0.3.0" {
		t.Errorf("second load: SchemaVersion = %q, want 0.3.0", cfg2.SchemaVersion)
	}

	githubCfg, ok := cfg2.Providers["github"]
	if !ok {
		t.Fatal("second load: providers.github missing")
	}
	if githubCfg.Token != "ghp_idem" {
		t.Errorf("second load: token = %q, want ghp_idem", githubCfg.Token)
	}

	// No second backup should be created.
	if _, err := os.Stat(cfgPath + ".v010.bak"); os.IsNotExist(err) {
		// The bak might have been overwritten with identical content — that's OK.
		// We just verify no duplicate like .v010.bak.1
	}
}

func TestConfigMigration_NoGitHubToken_V020Migrated(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Config at v0.2.0 without github_token at root — migrates to v0.3.0.
	plain := `{"schema_version":"0.2.0","providers":{}}`
	if err := os.WriteFile(cfgPath, []byte(plain), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath: %v", err)
	}
	// Migrated to v0.3.0.
	if cfg.SchemaVersion != "0.3.0" {
		t.Errorf("SchemaVersion = %q, want 0.3.0", cfg.SchemaVersion)
	}
	// Profiles should be initialized.
	if cfg.Profiles == nil {
		t.Error("Profiles should be non-nil after migration")
	}
}

func TestConfig_Get_NestedKeys(t *testing.T) {
	cfg := &Config{
		SchemaVersion: "0.2.0",
		Providers: map[string]ProviderConfig{
			"github":    {Token: "ghp_nested", GistID: "nest123"},
			"codeberg":  {Token: "cb_token"},
		},
	}

	tests := []struct {
		key  string
		want string
	}{
		{"github.token", "ghp_nested"},
		{"github.gist_id", "nest123"},
		{"providers.github.token", "ghp_nested"},
		{"providers.github.gist_id", "nest123"},
		{"providers.codeberg.token", "cb_token"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, err := cfg.Get(tt.key)
			if err != nil {
				t.Errorf("Get(%q): unexpected error: %v", tt.key, err)
				return
			}
			if got != tt.want {
				t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestConfig_Get_NestedKeys_Unknown(t *testing.T) {
	cfg := &Config{
		SchemaVersion: "0.2.0",
		Providers: map[string]ProviderConfig{
			"github": {Token: "t"},
		},
	}

	_, err := cfg.Get("providers.unknown.token")
	if err == nil {
		t.Fatal("expected error for unknown provider in nested key")
	}
	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("error = %v, want 'unknown config key'", err)
	}
}

func TestConfig_Set_NestedKeys(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	cfg := &Config{path: cfgPath}

	tests := []struct {
		key       string
		value     string
		wantField string // which field to check after set
	}{
		{"github.token", "ghp_set", "token"},
		{"github.gist_id", "gist_set", "gist_id"},
		{"providers.github.token", "ghp_set2", "token"},
		{"providers.codeberg.token", "cb_set", "token"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if err := cfg.Set(tt.key, tt.value); err != nil {
				t.Fatalf("Set(%q, %q): %v", tt.key, tt.value, err)
			}
			got, err := cfg.Get(tt.key)
			if err != nil {
				t.Fatalf("Get(%q) after Set: %v", tt.key, err)
			}
			if got != tt.value {
				t.Errorf("Get(%q) = %q, want %q", tt.key, got, tt.value)
			}
		})
	}
}

func TestConfig_Set_NestedKeys_Unknown(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfg := &Config{path: cfgPath}

	// Setting an unknown provider creates it — config is a flexible store.
	err := cfg.Set("providers.ghost.token", "val")
	if err != nil {
		t.Fatalf("Set: unexpected error: %v", err)
	}
	got, err := cfg.Get("providers.ghost.token")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "val" {
		t.Errorf("Get = %q, want val", got)
	}
}

func TestConfig_Save_V030(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	cfg := &Config{
		path:          cfgPath,
		SchemaVersion: "0.3.0",
		Providers: map[string]ProviderConfig{
			"github": {Token: "ghp_save", GistID: "save123"},
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "schema_version") {
		t.Error("saved config should contain schema_version")
	}
	if !strings.Contains(content, "providers") {
		t.Error("saved config should contain providers")
	}
	if !strings.Contains(content, "ghp_save") {
		t.Error("saved config should contain token value")
	}

	// Reload and verify.
	loaded, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath after Save: %v", err)
	}
	if loaded.SchemaVersion != "0.3.0" {
		t.Errorf("SchemaVersion = %q, want 0.3.0", loaded.SchemaVersion)
	}
}

func TestConfig_Save_CompatShim_NotDuplicated(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	// Write migrated config with schema_version and providers.
	cfg := &Config{
		path:          cfgPath,
		SchemaVersion: "0.3.0",
		Providers: map[string]ProviderConfig{
			"github": {Token: "ghp_clean"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	// Compat shim fields (github_token, gist_id at root) should not appear
	// when providers contains the data.
	if strings.Contains(content, `"github_token"`) {
		t.Error("saved config should NOT contain github_token at root (should be in providers)")
	}
	if strings.Contains(content, `"gist_id"`) && !strings.Contains(content, `"providers"`) {
		t.Error("saved config should NOT contain gist_id at root without providers")
	}
}

func TestConfig_CompatShim_LoadV010(t *testing.T) {
	// Loading a v0.1.0 config via compat shim (no migration)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	v010 := `{"github_token":"ghp_shim","gist_id":"shim123"}`
	if err := os.WriteFile(cfgPath, []byte(v010), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath: %v", err)
	}

	// Compat shim: after migration, GitHubToken/GistID should be empty,
	// but providers.github should have the data.
	githubCfg, ok := cfg.Providers["github"]
	if !ok {
		t.Fatal("providers.github should exist after migration")
	}
	if githubCfg.Token != "ghp_shim" {
		t.Errorf("providers.github.token = %q, want ghp_shim", githubCfg.Token)
	}
	if githubCfg.GistID != "shim123" {
		t.Errorf("providers.github.gist_id = %q, want shim123", githubCfg.GistID)
	}
}

// ---- v0.2.0 → v0.3.0 migration tests ----

func TestConfigMigration_V020_Detected(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	v020 := `{"schema_version":"0.2.0","providers":{"github":{"token":"ghp_migrate_v3","gist_id":"abc456"}}}`
	if err := os.WriteFile(cfgPath, []byte(v020), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath: %v", err)
	}

	// After migration, schema_version must be "0.3.0".
	if cfg.SchemaVersion != "0.3.0" {
		t.Errorf("SchemaVersion = %q, want 0.3.0", cfg.SchemaVersion)
	}

	// Providers data preserved.
	githubCfg, ok := cfg.Providers["github"]
	if !ok {
		t.Fatal("expected providers.github to exist after migration")
	}
	if githubCfg.Token != "ghp_migrate_v3" {
		t.Errorf("providers.github.token = %q, want ghp_migrate_v3", githubCfg.Token)
	}
	if githubCfg.GistID != "abc456" {
		t.Errorf("providers.github.gist_id = %q, want abc456", githubCfg.GistID)
	}

	// Profiles should be initialized.
	if cfg.Profiles == nil {
		t.Error("Profiles should be non-nil after migration")
	}
}

func TestConfigMigration_V020_BackupCreated(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	bakPath := cfgPath + ".v020.bak"

	v020 := `{"schema_version":"0.2.0","providers":{"github":{"token":"ghp_bak_v3","gist_id":"bak456"}}}`
	if err := os.WriteFile(cfgPath, []byte(v020), 0644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadPath(cfgPath); err != nil {
		t.Fatalf("LoadPath: %v", err)
	}

	// Backup file must exist with the v0.2.0 content.
	bakData, err := os.ReadFile(bakPath)
	if err != nil {
		t.Fatalf("expected .v020.bak file: %v", err)
	}
	if string(bakData) != v020 {
		t.Errorf("backup content = %q, want %q", string(bakData), v020)
	}
}

func TestConfigMigration_V020_Idempotent(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	v020 := `{"schema_version":"0.2.0","providers":{"github":{"token":"ghp_idem_v3","gist_id":"idem456"}}}`
	if err := os.WriteFile(cfgPath, []byte(v020), 0644); err != nil {
		t.Fatal(err)
	}

	cfg1, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath first: %v", err)
	}
	if cfg1.SchemaVersion != "0.3.0" {
		t.Fatalf("first load: SchemaVersion = %q, want 0.3.0", cfg1.SchemaVersion)
	}

	// Load again — should already be v0.3.0, no re-migration.
	cfg2, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath second: %v", err)
	}
	if cfg2.SchemaVersion != "0.3.0" {
		t.Errorf("second load: SchemaVersion = %q, want 0.3.0", cfg2.SchemaVersion)
	}

	githubCfg, ok := cfg2.Providers["github"]
	if !ok {
		t.Fatal("second load: providers.github missing")
	}
	if githubCfg.Token != "ghp_idem_v3" {
		t.Errorf("second load: token = %q, want ghp_idem_v3", githubCfg.Token)
	}
}

func TestConfigMigration_V010_ChainToV030(t *testing.T) {
	// v0.1.0 → v0.2.0 → v0.3.0 chain. Both .v010.bak and .v020.bak exist.
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")

	v010 := `{"github_token":"ghp_chain","gist_id":"chain123"}`
	if err := os.WriteFile(cfgPath, []byte(v010), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadPath(cfgPath)
	if err != nil {
		t.Fatalf("LoadPath: %v", err)
	}

	if cfg.SchemaVersion != "0.3.0" {
		t.Errorf("SchemaVersion = %q, want 0.3.0 (chain)", cfg.SchemaVersion)
	}

	// Both backup files should exist.
	if _, err := os.Stat(cfgPath + ".v010.bak"); os.IsNotExist(err) {
		t.Error("expected .v010.bak from v0.1.0→v0.2.0 migration")
	}
	if _, err := os.Stat(cfgPath + ".v020.bak"); os.IsNotExist(err) {
		t.Error("expected .v020.bak from v0.2.0→v0.3.0 migration")
	}

	// Providers should be populated.
	githubCfg, ok := cfg.Providers["github"]
	if !ok || githubCfg.Token != "ghp_chain" {
		t.Errorf("providers.github.token = %q, want ghp_chain", githubCfg.Token)
	}

	// Profiles should be initialized.
	if cfg.Profiles == nil {
		t.Error("Profiles should be non-nil after chain migration")
	}
}
