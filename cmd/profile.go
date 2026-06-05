package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/spf13/cobra"
)

// profileCmd is the parent command for profile management.
var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage machine-specific backup profiles",
	Long: `Profiles let you scope backups to specific machines or use cases.

Each profile defines which adapters, categories, and preset to use,
which cloud provider to sync with, and whether to encrypt backups.

Examples:
  bak profile create work --provider github-gist --preset full
  bak profile list
  bak profile show work
  bak profile delete work`,
}

// --- create ---

var profileCreateProvider string
var profileCreatePreset string
var profileCreateAdapters string
var profileCreateCategories string
var profileCreateEncrypt bool

var profileCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new backup profile",
	Long: `Create a named profile that groups backup settings for a machine.

The --provider flag is required and must reference a configured cloud provider
with a valid token.

Examples:
  bak profile create work-laptop --provider github-gist --preset full
  bak profile create home-pc --provider github-repo --preset quick --encrypt
  bak profile create dev-box --provider codeberg --adapters opencode,cursor`,
	Args: cobra.ExactArgs(1),
	RunE: runProfileCreate,
}

func init() {
	profileCreateCmd.Flags().StringVar(&profileCreateProvider, "provider", "",
		"cloud provider name (required)")
	profileCreateCmd.Flags().StringVar(&profileCreatePreset, "preset", "quick",
		"backup preset: quick, full, or skills")
	profileCreateCmd.Flags().StringVar(&profileCreateAdapters, "adapters", "",
		"comma-separated adapter names (e.g. opencode,cursor)")
	profileCreateCmd.Flags().StringVar(&profileCreateCategories, "categories", "",
		"comma-separated categories (e.g. config,skills)")
	profileCreateCmd.Flags().BoolVar(&profileCreateEncrypt, "encrypt", false,
		"enable encryption for this profile")
	profileCreateCmd.MarkFlagRequired("provider")

	profileCmd.AddCommand(profileCreateCmd)
	rootCmd.AddCommand(profileCmd)
}

func runProfileCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	out := cmd.OutOrStdout()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Validate provider exists.
	providerName := profileCreateProvider
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

	// Parse adapters and categories from comma-separated flags.
	var adapters []string
	if profileCreateAdapters != "" {
		for _, a := range strings.Split(profileCreateAdapters, ",") {
			a = strings.TrimSpace(a)
			if a != "" {
				adapters = append(adapters, a)
			}
		}
	}

	var categories []string
	if profileCreateCategories != "" {
		for _, c := range strings.Split(profileCreateCategories, ",") {
			c = strings.TrimSpace(c)
			if c != "" {
				categories = append(categories, c)
			}
		}
	}

	// Build profile config.
	profile := config.ProfileConfig{
		Adapters:   adapters,
		Categories: categories,
		Preset:     profileCreatePreset,
		Provider:   providerName,
	}

	if profileCreateEncrypt {
		profile.Encryption = &config.EncryptionConfig{
			Enabled: true,
		}
	}

	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]config.ProfileConfig)
	}
	cfg.Profiles[name] = profile

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(out, "Profile %q created.\n", name)
	if profileCreateEncrypt {
		fmt.Fprintln(out, "  Encryption: enabled (password will be prompted during push)")
	}
	fmt.Fprintf(out, "  Provider:   %s\n", providerName)
	if profileCreatePreset != "" {
		fmt.Fprintf(out, "  Preset:     %s\n", profileCreatePreset)
	}
	if len(adapters) > 0 {
		fmt.Fprintf(out, "  Adapters:   %s\n", strings.Join(adapters, ", "))
	}
	if len(categories) > 0 {
		fmt.Fprintf(out, "  Categories: %s\n", strings.Join(categories, ", "))
	}

	return nil
}

// --- list ---

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured profiles",
	Long:  "Display a table of all configured machine profiles with their provider, preset, and encryption status.",
	Args:  cobra.NoArgs,
	RunE:  runProfileList,
}

func init() {
	profileCmd.AddCommand(profileListCmd)
}

func runProfileList(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

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
			provider = "—"
		}
		encryption := "disabled"
		if p.Encryption != nil {
			encryption = "enabled"
		}
		fmt.Fprintf(out, "%-20s %-15s %-10s %s\n", name, provider, preset, encryption)
	}

	return nil
}

// --- show ---

var profileShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show detailed profile configuration",
	Long:  "Display the full configuration of a named profile including adapters, categories, preset, provider, and encryption status.",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileShow,
}

func init() {
	profileCmd.AddCommand(profileShowCmd)
}

func runProfileShow(cmd *cobra.Command, args []string) error {
	name := args[0]
	out := cmd.OutOrStdout()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

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

// --- delete ---

var profileDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a backup profile",
	Long:  "Remove a named profile from the configuration. This does not delete any backups.",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileDelete,
}

func init() {
	profileCmd.AddCommand(profileDeleteCmd)
}

func runProfileDelete(cmd *cobra.Command, args []string) error {
	name := args[0]
	out := cmd.OutOrStdout()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("profile %q not found — use 'bak profile list' to see configured profiles", name)
	}

	delete(cfg.Profiles, name)

	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(out, "Profile %q deleted.\n", name)
	return nil
}

// --- helpers ---

// profileList writes a profile name, its provider, and its encryption status
// to os.Stderr for verbose output (used by backup/push/pull when --profile
// resolves a profile).
func profileVerbose(name string, p config.ProfileConfig) {
	enc := "disabled"
	if p.Encryption != nil {
		enc = "enabled"
	}
	fmt.Fprintf(os.Stderr, "Using profile %q (provider=%s, preset=%s, encryption=%s)\n",
		name, p.Provider, p.Preset, enc)
}
