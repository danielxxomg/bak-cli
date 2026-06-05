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
