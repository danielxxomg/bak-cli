package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/danielxxomg/bak-cli/internal/actions"
	"github.com/spf13/cobra"
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
	action := &actions.PushAction{
		FS:       &actions.OSFileSystem{},
		Provider: pushProvider,
		Profile:  pushProfile,
		Verbose:  verbose,
	}

	return action.Run(cmd, args)
}

// resolveBackupID returns the backup ID from args or finds the most
// recent backup when no argument is given. Kept as a package-level
// helper for testability; PushAction has its own resolver internally.
func resolveBackupID(backupsDir string, args []string) (string, error) {
	if len(args) > 0 && args[0] != "" {
		return args[0], nil
	}

	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		return "", fmt.Errorf("read backups dir: %w", err)
	}

	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}

	if len(ids) == 0 {
		return "", fmt.Errorf("no backups found — run 'bak backup' first")
	}

	sort.Slice(ids, func(i, j int) bool {
		return ids[i] > ids[j]
	})

	return ids[0], nil
}
