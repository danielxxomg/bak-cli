package cmd

import (
	"fmt"
	"os"

	"github.com/danielxxomg/bak-cli/internal/actions"
	"github.com/danielxxomg/bak-cli/internal/adapters"
	"github.com/danielxxomg/bak-cli/internal/adapters/register"
	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/danielxxomg/bak-cli/internal/presets"
	"github.com/spf13/cobra"
)

var backupPreset string
var backupAdapter string
var backupProfile string
var backupOverride bool

// backupCmd represents the backup command.
var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create a backup of your AI coding setup",
	Long: `Scans for installed AI coding tools (currently OpenCode), resolves the
requested preset, copies configuration files to ~/.bak/backups/<id>/,
detects and redacts secrets, and writes a manifest.

Examples:
  bak backup                    # quick backup (config files only)
  bak backup --preset full      # everything: skills, commands, plugins, agents, config
  bak backup --preset skills    # skills only
  bak backup --adapter opencode # force a specific adapter
  bak backup --profile work     # use profile settings (preset, categories, adapters)
  bak backup --override         # prefer custom YAML presets/adapters over built-ins`,
	RunE: runBackup,
}

func init() {
	backupCmd.Flags().StringVarP(&backupPreset, "preset", "p", "quick",
		"backup preset: quick, full, or skills")
	backupCmd.Flags().StringVarP(&backupAdapter, "adapter", "a", "",
		"run only the named adapter (default: all detected)")
	backupCmd.Flags().StringVar(&backupProfile, "profile", "",
		"use named profile from config (overrides --preset, --adapter)")
	backupCmd.Flags().BoolVar(&backupOverride, "override", false,
		"prefer custom YAML presets and adapters over built-ins")

	rootCmd.AddCommand(backupCmd)
}

func runBackup(cmd *cobra.Command, args []string) error {
	// --- Build injectable dependencies -------------------------------------
	fs := &actions.OSFileSystem{}
	cfgLoader := &actions.RealConfigLoader{}

	// --- Wire adapters (built-in + YAML) -----------------------------------
	reg := adapters.NewRegistry()
	if err := register.All(reg); err != nil {
		return fmt.Errorf("register adapters: %w", err)
	}
	if err := register.LoadYAMLAdapters(reg, backupOverride); err != nil {
		return fmt.Errorf("load yaml adapters: %w", err)
	}

	// --- Resolve profile (overrides CLI flags) ----------------------------
	preset := backupPreset
	adapterFilter := backupAdapter
	var customCategories []string

	if backupProfile != "" {
		cfg, loadErr := config.Load()
		if loadErr != nil {
			return fmt.Errorf("load config for profile: %w", loadErr)
		}

		p, ok := cfg.Profiles[backupProfile]
		if !ok {
			return fmt.Errorf("profile %q not found — create it with 'bak profile create %s --provider <name>'", backupProfile, backupProfile)
		}

		if p.Preset != "" {
			preset = p.Preset
		}
		if len(p.Categories) > 0 {
			customCategories = p.Categories
		}
		if len(p.Adapters) > 0 {
			adapterFilter = p.Adapters[0]
		}

		if verbose {
			enc := "disabled"
			if p.Encryption != nil {
				enc = "enabled"
			}
			fmt.Fprintf(os.Stderr, "Using profile %q (provider=%s, preset=%s, encryption=%s)\n",
				backupProfile, p.Provider, preset, enc)
		}
	}

	// --- Resolve categories (YAML-aware) -----------------------------------
	if len(customCategories) == 0 {
		cats, err := presets.ResolveAll(preset, backupOverride)
		if err != nil {
			return fmt.Errorf("resolve preset %q: %w", preset, err)
		}
		customCategories = cats
	}

	// --- Build and run action ----------------------------------------------
	action := &actions.BackupAction{
		FS:               fs,
		Config:           cfgLoader,
		Registry:         reg,
		Preset:           preset,
		AdapterFilter:    adapterFilter,
		Verbose:          verbose,
		BakVersion:       Version,
		CustomCategories: customCategories,
	}

	return action.Run(cmd, args)
}
