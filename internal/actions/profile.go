package actions

import (
	"fmt"
	"io"
	"strings"

	"github.com/danielxxomg/bak-cli/internal/config"
)

// ProfileCreateOptions holds the non-interactive parameters for creating
// a backup profile.
type ProfileCreateOptions struct {
	Provider   string
	Preset     string
	Adapters   []string
	Categories []string
	Encrypt    bool
}

// ProfileCreate creates a named backup profile from the given options.
// It validates that the provider is configured and has a token (or rclone
// remote), checks for duplicate names, builds the profile config, saves
// it through cfg, and writes confirmation to out.
func ProfileCreate(cfg *config.Config, name string, opts ProfileCreateOptions, out io.Writer) error {
	// Validate provider exists.
	providerName := opts.Provider
	if _, ok := cfg.Providers[providerName]; !ok {
		known := make([]string, 0, len(cfg.Providers))
		for k := range cfg.Providers {
			known = append(known, k)
		}
		if len(known) == 0 {
			return fmt.Errorf("no providers configured — run 'bak login' or 'bak config set' first")
		}
		return fmt.Errorf("provider %q not configured (known: %s)", providerName, strings.Join(known, ", "))
	}

	// Validate provider has a token set (or remote for rclone).
	pc := cfg.Providers[providerName]
	if providerName == "rclone" {
		if pc.Remote == "" {
			return fmt.Errorf("rclone remote not configured — run 'bak config set providers.rclone.remote <name>'")
		}
	} else {
		if pc.Token == "" {
			return fmt.Errorf("provider %q has no token — run 'bak login' or set a token first", providerName)
		}
	}

	// Check for duplicate name.
	if _, exists := cfg.Profiles[name]; exists {
		return fmt.Errorf("profile %q already exists — use 'bak profile delete %s' first or choose a different name", name, name)
	}

	profile := config.ProfileConfig{
		Adapters:   opts.Adapters,
		Categories: opts.Categories,
		Preset:     opts.Preset,
		Provider:   providerName,
	}

	if opts.Encrypt {
		profile.Encryption = &config.EncryptionConfig{
			Enabled: true,
		}
	}

	return saveProfile(cfg, name, profile, out)
}

// saveProfile is the shared helper that stores a profile in config and
// writes confirmation output.
func saveProfile(cfg *config.Config, name string, profile config.ProfileConfig, out io.Writer) error {
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]config.ProfileConfig)
	}
	cfg.Profiles[name] = profile

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(out, "Profile %q created.\n", name)
	if profile.Encryption != nil {
		fmt.Fprintln(out, "  Encryption: enabled (password will be prompted during push)")
	}
	fmt.Fprintf(out, "  Provider:   %s\n", profile.Provider)
	if profile.Preset != "" {
		fmt.Fprintf(out, "  Preset:     %s\n", profile.Preset)
	}
	if len(profile.Adapters) > 0 {
		fmt.Fprintf(out, "  Adapters:   %s\n", strings.Join(profile.Adapters, ", "))
	}
	if len(profile.Categories) > 0 {
		fmt.Fprintf(out, "  Categories: %s\n", strings.Join(profile.Categories, ", "))
	}

	return nil
}

// ProfileList writes a table of all configured profiles to out.
func ProfileList(cfg *config.Config, out io.Writer) error {
	if len(cfg.Profiles) == 0 {
		fmt.Fprintln(out, "No profiles configured.")
		fmt.Fprintln(out, "Create one with: bak profile create <name> --provider <name>")
		return nil
	}

	// Header.
	fmt.Fprintf(out, "%-20s %-15s %-10s %s\n", "NAME", "PROVIDER", "PRESET", "ENCRYPTION")
	fmt.Fprintln(out, strings.Repeat("-", 65))

	for name, p := range cfg.Profiles {
		preset := p.Preset
		if preset == "" {
			preset = "quick"
		}
		provider := p.Provider
		if provider == "" {
			provider = "\u2014"
		}
		encryption := "disabled"
		if p.Encryption != nil {
			encryption = "enabled"
		}
		fmt.Fprintf(out, "%-20s %-15s %-10s %s\n", name, provider, preset, encryption)
	}

	return nil
}

// ProfileShow writes detailed configuration for a single profile to out.
func ProfileShow(cfg *config.Config, name string, out io.Writer) error {
	p, ok := cfg.Profiles[name]
	if !ok {
		return fmt.Errorf("profile %q not found — use 'bak profile list' to see configured profiles", name)
	}

	fmt.Fprintf(out, "Profile: %s\n", name)
	fmt.Fprintf(out, "  Provider:    %s\n", p.Provider)

	preset := p.Preset
	if preset == "" {
		preset = "quick"
	}
	fmt.Fprintf(out, "  Preset:      %s\n", preset)

	if len(p.Adapters) > 0 {
		fmt.Fprintf(out, "  Adapters:    %s\n", strings.Join(p.Adapters, ", "))
	} else {
		fmt.Fprintln(out, "  Adapters:    all detected")
	}

	if len(p.Categories) > 0 {
		fmt.Fprintf(out, "  Categories:  %s\n", strings.Join(p.Categories, ", "))
	} else {
		fmt.Fprintln(out, "  Categories:  from preset")
	}

	if p.Encryption != nil {
		fmt.Fprintln(out, "  Encryption:  enabled")
	} else {
		fmt.Fprintln(out, "  Encryption:  disabled")
	}

	return nil
}

// ParseCSV splits a comma-separated string into trimmed, non-empty tokens.
func ParseCSV(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, token := range strings.Split(s, ",") {
		token = strings.TrimSpace(token)
		if token != "" {
			result = append(result, token)
		}
	}
	return result
}

// ProfileCreateFromWizard holds the raw selections from the interactive
// wizard. cmd/ passes these after the bubbletea program finishes.
type ProfileCreateFromWizard struct {
	Confirmed        bool
	SelectedProvider string
	SelectedPreset   string
	AdapterNames     []string
	CategoryNames    []string
}

// ProfileValidateForCreation checks that the profile name is unique and
// providers are configured. Returns the provider list for the wizard.
func ProfileValidateForCreation(cfg *config.Config, name string) ([]string, error) {
	if _, exists := cfg.Profiles[name]; exists {
		return nil, fmt.Errorf("profile %q already exists — use 'bak profile delete %s' first or choose a different name", name, name)
	}

	providers := make([]string, 0, len(cfg.Providers))
	for k := range cfg.Providers {
		providers = append(providers, k)
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers configured — run 'bak login' first")
	}

	return providers, nil
}

// ProfileCreateInteractive validates the wizard results, builds the profile,
// saves it through cfg, and writes confirmation to out.
func ProfileCreateInteractive(cfg *config.Config, name string, wiz ProfileCreateFromWizard, out io.Writer) error {
	if !wiz.Confirmed {
		fmt.Fprintln(out, "Profile creation cancelled.")
		return nil
	}

	profile := config.ProfileConfig{
		Adapters:   wiz.AdapterNames,
		Categories: wiz.CategoryNames,
		Preset:     wiz.SelectedPreset,
		Provider:   wiz.SelectedProvider,
	}

	return saveProfile(cfg, name, profile, out)
}

// ProfileDelete removes a named profile from the configuration and saves.
// If dryRun is true, it only reports what would be deleted without saving.
func ProfileDelete(cfg *config.Config, name string, out io.Writer, dryRun bool) error {
	_, ok := cfg.Profiles[name]
	if !ok {
		return fmt.Errorf("profile %q not found — use 'bak profile list' to see configured profiles", name)
	}

	if dryRun {
		fmt.Fprintf(out, "[dry-run] Would delete profile %q.\n", name)
		return nil
	}

	delete(cfg.Profiles, name)

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(out, "Profile %q deleted.\n", name)
	return nil
}
