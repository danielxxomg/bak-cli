package actions

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/config"
)

// setupConfigDir creates a temporary config directory with a config.json
// pre-populated with the given providers. Returns the config directory path,
// the config file path, and a loaded *config.Config.
func setupConfigDir(t *testing.T, providers map[string]config.ProviderConfig) (string, *config.Config) {
	t.Helper()

	dir := t.TempDir()
	cfgDir := filepath.Join(dir, ".config", "bak")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(cfgDir, "config.json")
	cfg := &config.Config{
		SchemaVersion: "0.3.0",
		Providers:     providers,
		Profiles:      make(map[string]config.ProfileConfig),
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	loaded, err := config.LoadPath(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	return dir, loaded
}

func TestProfileCreate_HappyPath(t *testing.T) {
	_, cfg := setupConfigDir(t, map[string]config.ProviderConfig{
		"github-gist": {Token: "ghp_test123"},
	})

	var out bytes.Buffer
	err := ProfileCreate(cfg, "work", ProfileCreateOptions{
		Provider: "github-gist",
		Preset:   "full",
		Encrypt:  true,
	}, &out)
	if err != nil {
		t.Fatalf("ProfileCreate: %v", err)
	}

	// Verify profile was saved.
	p, ok := cfg.Profiles["work"]
	if !ok {
		t.Fatal("expected profile 'work' to exist")
	}
	if p.Provider != "github-gist" {
		t.Errorf("provider = %q, want github-gist", p.Provider)
	}
	if p.Preset != "full" {
		t.Errorf("preset = %q, want full", p.Preset)
	}
	if p.Encryption == nil || !p.Encryption.Enabled {
		t.Error("expected encryption to be enabled")
	}

	// Verify output message.
	if !strings.Contains(out.String(), `Profile "work" created`) {
		t.Errorf("output should contain creation message, got: %q", out.String())
	}
}

func TestProfileCreate_DuplicateName(t *testing.T) {
	_, cfg := setupConfigDir(t, map[string]config.ProviderConfig{
		"github-gist": {Token: "ghp_test123"},
	})

	// Create first profile (Profiles may be nil from omitempty — init it).
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]config.ProfileConfig)
	}
	cfg.Profiles["existing"] = config.ProfileConfig{
		Provider: "github-gist",
		Preset:   "quick",
	}

	var out bytes.Buffer
	err := ProfileCreate(cfg, "existing", ProfileCreateOptions{
		Provider: "github-gist",
		Preset:   "full",
	}, &out)
	if err == nil {
		t.Fatal("expected error for duplicate profile")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention already exists: %v", err)
	}
}

func TestProfileCreate_MissingProvider(t *testing.T) {
	_, cfg := setupConfigDir(t, map[string]config.ProviderConfig{
		"github-gist": {Token: "ghp_test123"},
	})

	var out bytes.Buffer
	err := ProfileCreate(cfg, "missing-prov", ProfileCreateOptions{
		Provider: "nonexistent",
		Preset:   "quick",
	}, &out)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("error should mention not configured: %v", err)
	}
}

func TestProfileCreate_NoProviders(t *testing.T) {
	_, cfg := setupConfigDir(t, nil)

	var out bytes.Buffer
	err := ProfileCreate(cfg, "test", ProfileCreateOptions{
		Provider: "github-gist",
		Preset:   "quick",
	}, &out)
	if err == nil {
		t.Fatal("expected error when no providers configured")
	}
	if !strings.Contains(err.Error(), "no providers configured") {
		t.Errorf("error should mention no providers: %v", err)
	}
}

func TestProfileCreate_NoToken(t *testing.T) {
	_, cfg := setupConfigDir(t, map[string]config.ProviderConfig{
		"github-gist": {Token: ""},
	})

	var out bytes.Buffer
	err := ProfileCreate(cfg, "test", ProfileCreateOptions{
		Provider: "github-gist",
		Preset:   "quick",
	}, &out)
	if err == nil {
		t.Fatal("expected error for provider without token")
	}
	if !strings.Contains(err.Error(), "no token") {
		t.Errorf("error should mention no token: %v", err)
	}
}

func TestProfileCreate_WithAdapters(t *testing.T) {
	_, cfg := setupConfigDir(t, map[string]config.ProviderConfig{
		"github-gist": {Token: "ghp_test123"},
	})

	var out bytes.Buffer
	err := ProfileCreate(cfg, "work", ProfileCreateOptions{
		Provider: "github-gist",
		Preset:   "full",
		Adapters: []string{"opencode", "cursor"},
	}, &out)
	if err != nil {
		t.Fatalf("ProfileCreate: %v", err)
	}

	p, ok := cfg.Profiles["work"]
	if !ok {
		t.Fatal("expected profile 'work' to exist")
	}
	if len(p.Adapters) != 2 || p.Adapters[0] != "opencode" || p.Adapters[1] != "cursor" {
		t.Errorf("adapters = %v, want [opencode cursor]", p.Adapters)
	}
	if !strings.Contains(out.String(), "Adapters:   opencode, cursor") {
		t.Errorf("output should list adapters: %q", out.String())
	}
}

func TestProfileList_Empty(t *testing.T) {
	_, cfg := setupConfigDir(t, nil)

	var out bytes.Buffer
	err := ProfileList(cfg, &out)
	if err != nil {
		t.Fatalf("ProfileList: %v", err)
	}
	if !strings.Contains(out.String(), "No profiles configured") {
		t.Errorf("output should mention no profiles: %q", out.String())
	}
}

func TestProfileList_Populated(t *testing.T) {
	_, cfg := setupConfigDir(t, map[string]config.ProviderConfig{
		"github-gist": {Token: "ghp_test"},
	})

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]config.ProfileConfig)
	}
	cfg.Profiles["work"] = config.ProfileConfig{
		Provider: "github-gist",
		Preset:   "full",
	}
	cfg.Profiles["home"] = config.ProfileConfig{
		Provider:   "codeberg",
		Preset:     "quick",
		Encryption: &config.EncryptionConfig{Enabled: true},
	}

	var out bytes.Buffer
	err := ProfileList(cfg, &out)
	if err != nil {
		t.Fatalf("ProfileList: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "work") {
		t.Error("list output should include 'work'")
	}
	if !strings.Contains(output, "home") {
		t.Error("list output should include 'home'")
	}
	if !strings.Contains(output, "enabled") {
		t.Error("list output should show encryption status")
	}
}

func TestProfileShow_Exists(t *testing.T) {
	_, cfg := setupConfigDir(t, map[string]config.ProviderConfig{
		"github-gist": {Token: "ghp_test"},
	})

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]config.ProfileConfig)
	}
	cfg.Profiles["work"] = config.ProfileConfig{
		Provider: "github-gist",
		Preset:   "full",
		Adapters: []string{"opencode"},
	}

	var out bytes.Buffer
	err := ProfileShow(cfg, "work", &out)
	if err != nil {
		t.Fatalf("ProfileShow: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "work") {
		t.Error("show output should include profile name")
	}
	if !strings.Contains(output, "github-gist") {
		t.Error("show output should include provider")
	}
	if !strings.Contains(output, "full") {
		t.Error("show output should include preset")
	}
}

func TestProfileShow_Missing(t *testing.T) {
	_, cfg := setupConfigDir(t, nil)

	var out bytes.Buffer
	err := ProfileShow(cfg, "nonexistent", &out)
	if err == nil {
		t.Fatal("expected error for missing profile")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestProfileDelete_Exists(t *testing.T) {
	_, cfg := setupConfigDir(t, map[string]config.ProviderConfig{
		"github-gist": {Token: "ghp_test"},
	})

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]config.ProfileConfig)
	}
	cfg.Profiles["work"] = config.ProfileConfig{
		Provider: "github-gist",
		Preset:   "quick",
	}

	var out bytes.Buffer
	err := ProfileDelete(cfg, "work", &out, false)
	if err != nil {
		t.Fatalf("ProfileDelete: %v", err)
	}

	if _, ok := cfg.Profiles["work"]; ok {
		t.Error("profile 'work' should have been deleted")
	}
	if !strings.Contains(out.String(), `Profile "work" deleted`) {
		t.Errorf("output should confirm deletion: %q", out.String())
	}
}

func TestProfileDelete_DryRun(t *testing.T) {
	_, cfg := setupConfigDir(t, map[string]config.ProviderConfig{
		"github-gist": {Token: "ghp_test"},
	})

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]config.ProfileConfig)
	}
	cfg.Profiles["work"] = config.ProfileConfig{
		Provider: "github-gist",
		Preset:   "quick",
	}

	var out bytes.Buffer
	err := ProfileDelete(cfg, "work", &out, true)
	if err != nil {
		t.Fatalf("ProfileDelete dry-run: %v", err)
	}

	// Profile should NOT be deleted in dry-run mode.
	if _, ok := cfg.Profiles["work"]; !ok {
		t.Error("profile 'work' should NOT be deleted in dry-run mode")
	}
	if !strings.Contains(out.String(), "[dry-run]") {
		t.Errorf("output should mention dry-run: %q", out.String())
	}
}

func TestProfileValidateForCreation(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *config.Config
		profileName   string
		wantErr       bool
		errContains   string
		wantProviders int // minimum number of providers expected
	}{
		{
			name: "duplicate name",
			cfg: &config.Config{
				Profiles: map[string]config.ProfileConfig{
					"work": {Provider: "github-gist"},
				},
				Providers: map[string]config.ProviderConfig{
					"github-gist": {Token: "ghp_test"},
				},
			},
			profileName: "work",
			wantErr:     true,
			errContains: "already exists",
		},
		{
			name: "no providers",
			cfg: &config.Config{
				Profiles:  map[string]config.ProfileConfig{},
				Providers: map[string]config.ProviderConfig{},
			},
			profileName: "test",
			wantErr:     true,
			errContains: "no providers configured",
		},
		{
			name: "valid with single provider",
			cfg: &config.Config{
				Profiles: map[string]config.ProfileConfig{},
				Providers: map[string]config.ProviderConfig{
					"github-gist": {Token: "ghp_test"},
				},
			},
			profileName:   "new-profile",
			wantErr:       false,
			wantProviders: 1,
		},
		{
			name: "valid with multiple providers",
			cfg: &config.Config{
				Profiles: map[string]config.ProfileConfig{},
				Providers: map[string]config.ProviderConfig{
					"github-gist": {Token: "ghp_test"},
					"codeberg":    {Token: "cb_test"},
				},
			},
			profileName:   "new-profile",
			wantErr:       false,
			wantProviders: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers, err := ProfileValidateForCreation(tt.cfg, tt.profileName)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errContains)
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want substring %q", err, tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(providers) < tt.wantProviders {
				t.Errorf("providers count = %d, want at least %d (got %v)", len(providers), tt.wantProviders, providers)
			}
		})
	}
}

func TestParseCSV(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty string", "", nil},
		{"single token", "opencode", []string{"opencode"}},
		{"multiple tokens", "opencode,cursor", []string{"opencode", "cursor"}},
		{"whitespace trimming", " opencode , cursor ", []string{"opencode", "cursor"}},
		{"empty middle token", "opencode,,cursor", []string{"opencode", "cursor"}},
		{"trailing comma", "opencode,", []string{"opencode"}},
		{"whitespace only", "  ,  ", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseCSV(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("ParseCSV(%q) = %v, want nil", tt.input, got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("ParseCSV(%q) = %v (len=%d), want %v (len=%d)", tt.input, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("ParseCSV(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestProfileDelete_Missing(t *testing.T) {
	_, cfg := setupConfigDir(t, nil)

	var out bytes.Buffer
	err := ProfileDelete(cfg, "nonexistent", &out, false)
	if err == nil {
		t.Fatal("expected error for missing profile")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}
