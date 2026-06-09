package cmd

import (
	"github.com/spf13/cobra"

	"github.com/danielxxomg/bak-cli/internal/actions"
	"github.com/danielxxomg/bak-cli/internal/config"
)

var pushProvider string
var pushProfile string

// pushCmd represents the push command.
var pushCmd = &cobra.Command{
	Use:   "push [backup-id]",
	Short: "Push a backup to the cloud",
	Long: `Package a local backup and push it to a cloud backend for
sync across machines.

If no backup ID is provided, the most recent backup is used.

Supported providers:
  github-gist (default) — push to a private GitHub Gist

Requires a token configured via 'bak login' or the appropriate
environment variable.

Examples:
  bak push                          # push latest backup
  bak push 20260604-150405          # push a specific backup
  bak push --provider github-gist   # explicit provider`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPush,
}

func init() {
	pushCmd.Flags().StringVar(&pushProvider, "provider", "github-gist",
		"cloud provider to use (github-gist)")
	pushCmd.Flags().StringVar(&pushProfile, "profile", "default",
		"encryption profile to use from config")
	rootCmd.AddCommand(pushCmd)
}

func runPush(cmd *cobra.Command, args []string) error {
	return runPushWithDeps(cmd, args, depsFromCmd(cmd))
}

// runPushWithDeps follows the *WithDeps pattern for testability.
func runPushWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	action := &actions.PushAction{
		FS:           &actions.OSFileSystem{},
		Provider:     pushProvider,
		Profile:      pushProfile,
		Verbose:      verbose,
		Stdout:       deps.Stdout,
		Stderr:       deps.Stderr,
		Factory:      &actions.RealProviderFactory{},
		ConfigLoader: config.Load,
	}

	return action.Run(args)
}
