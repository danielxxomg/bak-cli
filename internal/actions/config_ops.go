package actions

import "github.com/danielxxomg/bak-cli/internal/config"

// SaveSetting applies a single TUI settings key-value pair to the config
// in memory. The caller is responsible for loading and saving config.
// Unknown keys are silently ignored.
func SaveSetting(cfg *config.Config, key string, value any) {
	switch key {
	case "auto_sync":
		if v, ok := value.(bool); ok {
			cfg.Settings.AutoSync = v
		}
	case "verbose_default":
		if v, ok := value.(bool); ok {
			cfg.Settings.VerboseDefault = v
		}
	case "confirm_destructive":
		if v, ok := value.(bool); ok {
			cfg.Settings.ConfirmDestructive = &v
		}
	case "default_provider":
		if v, ok := value.(bool); ok {
			if v {
				cfg.Settings.DefaultProvider = "github"
			} else {
				cfg.Settings.DefaultProvider = ""
			}
		}
	}
}

// SaveProfileFromInfo persists a profile with the given name, provider, and
// preset in-memory. The caller is responsible for saving config. If a
// profile with the same name already exists, it is overwritten.
func SaveProfileFromInfo(cfg *config.Config, name string, provider string, preset string) {
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]config.ProfileConfig)
	}
	cfg.Profiles[name] = config.ProfileConfig{
		Provider: provider,
		Preset:   preset,
	}
}

// DeleteProfileSilent removes a profile by name from memory. The caller is
// responsible for saving config. Deleting a non-existent profile is a no-op.
func DeleteProfileSilent(cfg *config.Config, name string) {
	delete(cfg.Profiles, name)
}

// SetActiveProfile marks the named profile as active in memory. The caller
// is responsible for saving config.
func SetActiveProfile(cfg *config.Config, name string) {
	cfg.ActiveProfile = name
}

// GetCloudProviderStatus returns the default provider name and whether a
// token (or rclone remote) is configured, indicating connectivity.
func GetCloudProviderStatus(cfg *config.Config) (provider string, connected bool) {
	provider = cfg.Settings.DefaultProvider
	if provider == "" {
		provider = "github"
	}
	if p, ok := cfg.Providers[provider]; ok {
		connected = p.Token != ""
	}
	return
}

// ProfileInfo holds displayable profile data for the profiles screen.
type ProfileInfo struct {
	Name     string
	Provider string
	Preset   string
	Active   bool
}

// ListProfileInfos returns all configured profiles as a slice of ProfileInfo
// suitable for display in the TUI profiles screen.
func ListProfileInfos(cfg *config.Config) []ProfileInfo {
	var result []ProfileInfo
	for name, p := range cfg.Profiles {
		result = append(result, ProfileInfo{
			Name:     name,
			Provider: p.Provider,
			Preset:   p.Preset,
			Active:   name == cfg.ActiveProfile,
		})
	}
	return result
}
