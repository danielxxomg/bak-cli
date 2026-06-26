package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/danielxxomg/bak-cli/internal/actions"
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
var profileCreateInteractive bool

var profileCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new backup profile",
	Long: `Create a named profile that groups backup settings for a machine.

The --provider flag is required unless --interactive is used to launch
the step-by-step wizard.

Examples:
  bak profile create work-laptop --provider github-gist --preset full
  bak profile create home-pc --provider github-repo --preset quick --encrypt
  bak profile create dev-box --provider codeberg --adapters opencode,cursor
  bak profile create my-profile --interactive`,
	Args: cobra.MaximumNArgs(1),
	RunE: runProfileCreate,
}

func init() {
	profileCreateCmd.Flags().StringVar(&profileCreateProvider, "provider", "",
		"cloud provider name (required unless --interactive)")
	profileCreateCmd.Flags().StringVar(&profileCreatePreset, "preset", "quick",
		"backup preset: quick, full, or skills")
	profileCreateCmd.Flags().StringVar(&profileCreateAdapters, "adapters", "",
		"comma-separated adapter names (e.g. opencode,cursor)")
	profileCreateCmd.Flags().StringVar(&profileCreateCategories, "categories", "",
		"comma-separated categories (e.g. config,skills)")
	profileCreateCmd.Flags().BoolVar(&profileCreateEncrypt, "encrypt", false,
		"enable encryption for this profile")
	profileCreateCmd.Flags().BoolVar(&profileCreateInteractive, "interactive", false,
		"launch interactive wizard for profile creation")

	profileCmd.AddCommand(profileCreateCmd)
	rootCmd.AddCommand(profileCmd)
}

func runProfileCreate(cmd *cobra.Command, args []string) error {
	return runProfileCreateWithDeps(cmd, args, depsFromCmd(cmd))
}

func runProfileCreateWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	var name string
	if len(args) > 0 {
		name = args[0]
	}
	out := deps.Stdout

	// Interactive wizard mode.
	if profileCreateInteractive {
		return runProfileCreateInteractiveWithDeps(cmd, name, deps)
	}

	// No name and not interactive: error.
	if name == "" {
		return fmt.Errorf("provide a profile name or use --interactive")
	}

	// --provider is required in non-interactive mode.
	if profileCreateProvider == "" {
		return fmt.Errorf("required flag \"--provider\" not set (or use --interactive)")
	}

	cfg, err := deps.ConfigLoader()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Parse adapters and categories from comma-separated flags.
	adapters := actions.ParseCSV(profileCreateAdapters)
	categories := actions.ParseCSV(profileCreateCategories)

	return actions.ProfileCreate(cfg, name, actions.ProfileCreateOptions{
		Provider:   profileCreateProvider,
		Preset:     profileCreatePreset,
		Adapters:   adapters,
		Categories: categories,
		Encrypt:    profileCreateEncrypt,
	}, out)
}

// --- list ---

var profileListCmd = &cobra.Command{
	Use:   listAction,
	Short: "List all configured profiles",
	Long:  "Display a table of all configured machine profiles with their provider, preset, and encryption status.",
	Args:  cobra.NoArgs,
	RunE:  runProfileList,
}

func init() {
	profileCmd.AddCommand(profileListCmd)
}

func runProfileList(cmd *cobra.Command, args []string) error {
	return runProfileListWithDeps(cmd, args, depsFromCmd(cmd))
}

func runProfileListWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	cfg, err := deps.ConfigLoader()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return actions.ProfileList(cfg, deps.Stdout)
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
	return runProfileShowWithDeps(cmd, args, depsFromCmd(cmd))
}

func runProfileShowWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	cfg, err := deps.ConfigLoader()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return actions.ProfileShow(cfg, args[0], deps.Stdout)
}

// --- delete ---

var profileDeleteDryRun bool

var profileDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a backup profile",
	Long:  "Remove a named profile from the configuration. This does not delete any backups.",
	Args:  cobra.ExactArgs(1),
	RunE:  runProfileDelete,
}

func init() {
	profileDeleteCmd.Flags().BoolVar(&profileDeleteDryRun, "dry-run", false,
		"preview what would be deleted without making changes")
	profileCmd.AddCommand(profileDeleteCmd)
}

func runProfileDelete(cmd *cobra.Command, args []string) error {
	return runProfileDeleteWithDeps(cmd, args, depsFromCmd(cmd))
}

func runProfileDeleteWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	cfg, err := deps.ConfigLoader()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	return actions.ProfileDelete(cfg, args[0], deps.Stdout, profileDeleteDryRun)
}

// --- interactive profile creation ---

func runProfileCreateInteractiveWithDeps(cmd *cobra.Command, name string, deps cmdDeps) error {
	out := deps.Stdout

	cfg, err := deps.ConfigLoader()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// When a name is supplied, validate it up front and pre-resolve the
	// provider list for the wizard. The wizard then runs with those
	// providers; the supplied name is used directly.
	if name != "" {
		providers, err := actions.ProfileValidateForCreation(cfg, name)
		if err != nil {
			return err
		}
		fromWizard, _, err := launchWizard(providers)
		if err != nil {
			return err
		}
		return actions.ProfileCreateInteractive(cfg, name, fromWizard, out)
	}

	// No name supplied — the wizard collects it via the NameStep and
	// auto-detects providers (nil providers). Use the wizard's name.
	fromWizard, wm, err := launchWizard(nil) // nil providers → wizard auto-detects
	if err != nil {
		return err
	}
	wizardName := wm.ProfileName()
	if wizardName == "" {
		return fmt.Errorf("profile name is required")
	}
	return actions.ProfileCreateInteractive(cfg, wizardName, fromWizard, out)
}
