package cmd

import (
	"github.com/danielxxomg/bak-cli/internal/actions"
	"github.com/spf13/cobra"
)

// verifyCmd represents the verify command.
var verifyCmd = &cobra.Command{
	Use:   "verify <backup-id>",
	Short: "Verify a backup's integrity",
	Long: `Verifies the integrity of a backup by checking SHA-256 hashes
of every file in the manifest against the actual files on disk.

Exits 0 if all files pass, exits 1 on the first hash mismatch.

Example:
  bak verify 20260604-150405
  bak verify --verbose 20260604-150405`,
	Args: cobra.ExactArgs(1),
	RunE: runVerify,
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}

func runVerify(cmd *cobra.Command, args []string) error {
	return runVerifyWithDeps(cmd, args, depsFromCmd(cmd))
}

func runVerifyWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	action := &actions.VerifyBackupAction{
		Stdout:  deps.Stdout,
		Stderr:  deps.Stderr,
		Verbose: verbose,
	}
	return action.Run(args[0])
}
