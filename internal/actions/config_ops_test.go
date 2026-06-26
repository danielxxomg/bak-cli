package actions

import (
	"testing"

	"github.com/danielxxomg/bak-cli/internal/config"
)

func TestSaveSetting(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name     string
		cfg      *config.Config
		key      string
		value    any
		validate func(t *testing.T, cfg *config.Config)
	}{
		{
			name:  "set auto_sync to true",
			cfg:   &config.Config{Settings: config.DefaultSettings()},
			key:   "auto_sync",
			value: true,
			validate: func(t *testing.T, cfg *config.Config) {
				if !cfg.Settings.AutoSync {
					t.Error("AutoSync should be true")
				}
			},
		},
		{
			name:  "set verbose_default to true",
			cfg:   &config.Config{Settings: config.DefaultSettings()},
			key:   "verbose_default",
			value: true,
			validate: func(t *testing.T, cfg *config.Config) {
				if !cfg.Settings.VerboseDefault {
					t.Error("VerboseDefault should be true")
				}
			},
		},
		{
			name:  "set confirm_destructive to false",
			cfg:   &config.Config{Settings: config.DefaultSettings()},
			key:   "confirm_destructive",
			value: false,
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Settings.ConfirmDestructive == nil || *cfg.Settings.ConfirmDestructive {
					t.Error("ConfirmDestructive should be false")
				}
			},
		},
		{
			name:  "set default_provider to github",
			cfg:   &config.Config{Settings: config.DefaultSettings()},
			key:   "default_provider",
			value: true,
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Settings.DefaultProvider != "github" {
					t.Errorf("DefaultProvider = %q, want github", cfg.Settings.DefaultProvider)
				}
			},
		},
		{
			name:  "set default_provider to false clears it",
			cfg:   &config.Config{Settings: config.Settings{DefaultProvider: "github"}},
			key:   "default_provider",
			value: false,
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Settings.DefaultProvider != "" {
					t.Errorf("DefaultProvider = %q, want empty", cfg.Settings.DefaultProvider)
				}
			},
		},
		{
			name:  "unknown key is no-op",
			cfg:   &config.Config{Settings: config.DefaultSettings()},
			key:   "nonexistent",
			value: true,
			validate: func(t *testing.T, cfg *config.Config) {
				// No changes expected.
			},
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			SaveSetting(tt.cfg, tt.key, tt.value)
			if tt.validate != nil {
				tt.validate(t, tt.cfg)
			}
		})
	}
}

func TestSaveProfileFromInfo(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	cfg := &config.Config{}
	SaveProfileFromInfo(cfg, "test-profile", "github-gist", "quick")

	p, ok := cfg.Profiles["test-profile"]
	if !ok {
		t.Fatal("profile not saved")
	}
	if p.Provider != "github-gist" {
		t.Errorf("Provider = %q, want github-gist", p.Provider)
	}
	if p.Preset != "quick" {
		t.Errorf("Preset = %q, want quick", p.Preset)
	}

	// Overwrite with different values.
	SaveProfileFromInfo(cfg, "test-profile", "codeberg", "full")
	p = cfg.Profiles["test-profile"]
	if p.Provider != "codeberg" {
		t.Errorf("Provider = %q, want codeberg", p.Provider)
	}
}

func TestDeleteProfileSilent(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	cfg := &config.Config{
		Profiles: map[string]config.ProfileConfig{
			"keep":   {Provider: "github-gist"},
			"remove": {Provider: "codeberg"},
		},
	}
	DeleteProfileSilent(cfg, "remove")
	if _, ok := cfg.Profiles["remove"]; ok {
		t.Error("profile 'remove' should be deleted")
	}
	if _, ok := cfg.Profiles["keep"]; !ok {
		t.Error("profile 'keep' should still exist")
	}

	// Deleting non-existent profile is a no-op (no panic).
	DeleteProfileSilent(cfg, "nonexistent")
}

func TestSetActiveProfile(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	cfg := &config.Config{
		Profiles: map[string]config.ProfileConfig{
			"github": {Provider: "github-gist"},
		},
	}
	SetActiveProfile(cfg, "github")
	if cfg.ActiveProfile != "github" {
		t.Errorf("ActiveProfile = %q, want github", cfg.ActiveProfile)
	}
}

func TestGetCloudProviderStatus(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name          string
		cfg           *config.Config
		wantProvider  string
		wantConnected bool
	}{
		{
			name: "github with token",
			cfg: &config.Config{
				Settings: config.Settings{DefaultProvider: "github"},
				Providers: map[string]config.ProviderConfig{
					"github": {Token: "ghp_test123"},
				},
			},
			wantProvider:  "github",
			wantConnected: true,
		},
		{
			name: "github without token",
			cfg: &config.Config{
				Settings: config.Settings{DefaultProvider: "github"},
				Providers: map[string]config.ProviderConfig{
					"github": {},
				},
			},
			wantProvider:  "github",
			wantConnected: false,
		},
		{
			name: "no default provider falls back to github",
			cfg: &config.Config{
				Settings: config.Settings{},
			},
			wantProvider:  "github",
			wantConnected: false,
		},
		{
			name: "rclone has remote but no token",
			cfg: &config.Config{
				Settings: config.Settings{DefaultProvider: "rclone"},
				Providers: map[string]config.ProviderConfig{
					"rclone": {Remote: "myremote"},
				},
			},
			wantProvider:  "rclone",
			wantConnected: false, // rclone uses Remote not Token
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			provider, connected := GetCloudProviderStatus(tt.cfg)
			if provider != tt.wantProvider {
				t.Errorf("provider = %q, want %q", provider, tt.wantProvider)
			}
			if connected != tt.wantConnected {
				t.Errorf("connected = %v, want %v", connected, tt.wantConnected)
			}
		})
	}
}

func TestListProfileInfos(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	cfg := &config.Config{
		Profiles: map[string]config.ProfileConfig{
			"default": {Provider: "github-gist", Preset: "quick"},
			"full":    {Provider: "codeberg", Preset: "full"},
		},
		ActiveProfile: "default",
	}

	infos := ListProfileInfos(cfg)
	if len(infos) != 2 {
		t.Fatalf("len(infos) = %d, want 2", len(infos))
	}

	for _, info := range infos {
		switch info.Name {
		case "default":
			if info.Provider != "github-gist" || info.Preset != "quick" || !info.Active {
				t.Errorf("default profile: %+v", info)
			}
		case "full":
			if info.Provider != "codeberg" || info.Preset != "full" || info.Active {
				t.Errorf("full profile: %+v", info)
			}
		default:
			t.Errorf("unexpected profile name: %s", info.Name)
		}
	}

	// Empty config.
	cfg2 := &config.Config{}
	infos2 := ListProfileInfos(cfg2)
	if len(infos2) != 0 {
		t.Errorf("len(infos2) = %d, want 0", len(infos2))
	}
}
