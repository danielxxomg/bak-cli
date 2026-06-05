package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
