package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/danielxxomg/bak-cli/internal/actions"
	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/danielxxomg/bak-cli/internal/cloud"
	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/spf13/cobra"
)

var listProvider string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all local backups",
	Long: `Scans ~/.bak/backups/ and displays a table of all local backups
with their ID, date, preset, file count, and size.

Use 'bak restore <id>' to restore a backup.
Use 'bak push <id>' to upload a backup to GitHub Gist.

When --provider is set, lists backups from the specified cloud backend
instead of local backups.

Examples:
  bak list
  bak list --provider github-gist`,
	RunE: runList,
}

func init() {
	listCmd.Flags().StringVar(&listProvider, "provider", "",
		"list backups from a cloud provider instead of local (github-gist, codeberg, gitea, github-repo, rclone)")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	// When --provider is set, list from cloud backend.
	if listProvider != "" {
		return runListCloud(listProvider)
	}

	// Local listing (existing behavior).
	return runListLocal(cmd)
}

func runListLocal(cmd *cobra.Command) error {
	bakDir, err := backup.BakDir()
	if err != nil {
		return fmt.Errorf("bak dir: %w", err)
	}

	return actions.RunListLocal(bakDir, verbose, cmd.OutOrStdout(), os.Stderr)
}

// runListCloud lists backups from a named cloud provider.
func runListCloud(providerName string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	reg := cloud.NewProviderRegistry()

	// Register all available providers (they'll fail at runtime if not configured).
	reg.Register(cloud.NewGitHubGistProvider(cfg, ""))
	reg.Register(cloud.NewGitHubRepoProvider(cfg, "", cfg.Providers["github"].Repo))
	reg.Register(cloud.NewCodebergProvider(cfg, "", cfg.Providers["codeberg"].Repo))
	reg.Register(cloud.NewGiteaProvider(cfg, "", cfg.Providers["gitea"].BaseURL, cfg.Providers["gitea"].Repo))
	reg.Register(cloud.NewRcloneProvider(cfg, cfg.Providers["rclone"].Remote))
	reg.SetDefault("github-gist")

	provider, err := reg.Get(providerName)
	if err != nil {
		return fmt.Errorf("provider: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Using provider: %s\n", provider.Name())
	}

	backups, err := provider.List()
	if err != nil {
		return fmt.Errorf("list from %s: %w", providerName, err)
	}

	if len(backups) == 0 {
		fmt.Printf("No backups found on %s.\n", providerName)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDATE\tHOST\tSIZE\tURL")
	fmt.Fprintln(w, "--\t----\t----\t----\t---")

	for _, b := range backups {
		date := b.CreatedAt.Format(time.RFC3339)
		sizeStr := formatSize(b.Size)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			b.ID, date, b.Hostname, sizeStr, b.URL)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}

	return nil
}

// formatSizeBytes delegates to actions.formatSizeBytes for backward compatibility.
func formatSizeBytes(bytes int64) string {
	return actions.FormatSizeBytes(bytes)
}
