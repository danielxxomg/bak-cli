package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/danielxxomg/bak-cli/internal/tui"
)

var verbose bool

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "bak",
	Short: "Backup and restore your AI coding setup",
	Long: `bak packs, restores, and syncs your OpenCode configuration
across machines with safety guarantees.

Run 'bak backup' to create a backup.
Run 'bak restore --dry-run <id>' to preview before applying.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Launch the interactive TUI when bak is invoked with no arguments
		// and stdout is a terminal. Cobra handles --help before RunE, so
		// `bak --help` still shows cobra help regardless of TTY.
		if len(args) == 0 && isTTY() {
			deps := tui.Deps{
				Version:      Version,
				ConfigExists: configExists,
			}
			return runTUI(deps)
		}
		// Fall through to cobra help output (non-TTY or non-empty args).
		return cmd.Help()
	},
}

// configExists returns true if the bak configuration file exists on disk.
// Used by the TUI for first-run welcome detection.
func configExists() bool {
	cfgPath, err := config.DefaultPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(cfgPath)
	return err == nil
}

// Execute runs the root command.
func Execute() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	if err := rootCmd.Execute(); err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}

// formatSize returns a human-readable byte count.
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
