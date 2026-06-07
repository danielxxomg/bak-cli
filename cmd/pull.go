package cmd

import (
	"github.com/danielxxomg/bak-cli/internal/actions"
	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/spf13/cobra"
)

var pullProvider string
var pullProfile string

// pullCmd represents the pull command.
var pullCmd = &cobra.Command{
	Use:   "pull [gist-id]",
	Short: "Pull a backup from the cloud",
	Long: `Download a backup from a cloud backend and reconstruct it
locally in ~/.bak/backups/.

If no ID is provided, the ID stored from a previous push is used.

Supported providers:
  github-gist (default) — pull from a private GitHub Gist

Requires a token configured via 'bak login' or the appropriate
environment variable.

Examples:
  bak pull                          # pull from stored ID
  bak pull abc123def456             # pull from specific ID
  bak pull --provider github-gist   # explicit provider`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPull,
}

func init() {
	pullCmd.Flags().StringVar(&pullProvider, "provider", "github-gist",
		"cloud provider to use (github-gist)")
	pullCmd.Flags().StringVar(&pullProfile, "profile", "default",
		"decryption profile to use from config")
	rootCmd.AddCommand(pullCmd)
}

func runPull(cmd *cobra.Command, args []string) error {
	return runPullWithDeps(cmd, args, depsFromCmd(cmd))
}

// runPullWithDeps follows the *WithDeps pattern for testability.
// deps is accepted for consistency even if not directly used here
// (the action wires its own FS/Factory).
func runPullWithDeps(cmd *cobra.Command, args []string, _ cmdDeps) error {
	action := &actions.PullAction{
		FS:           &actions.OSFileSystem{},
		ConfigLoader: config.Load,
		Provider:     pullProvider,
		Profile:      pullProfile,
		Verbose:      verbose,
		Factory:      &actions.RealProviderFactory{},
	}

	return action.Run(args)
}
