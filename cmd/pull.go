package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/danielxxomg/bak-cli/internal/cloud"
	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/spf13/cobra"
)

var pullProvider string

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
	rootCmd.AddCommand(pullCmd)
}

func runPull(cmd *cobra.Command, args []string) error {
	// 1. Load config and build provider registry.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	reg := cloud.NewProviderRegistry()
	defaultProvider := cloud.NewGitHubGistProvider(cfg, "")
	if err := reg.Register(defaultProvider); err != nil {
		return fmt.Errorf("register provider: %w", err)
	}
	reg.SetDefault("github-gist")

	provider, err := reg.Get(pullProvider)
	if err != nil {
		return fmt.Errorf("provider: %w", err)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "Using provider: %s\n", provider.Name())
	}

	// 2. Resolve backup ID.
	var remoteID string
	if len(args) > 0 && args[0] != "" {
		remoteID = args[0]
	} else {
		id, err := cfg.Get("github.gist_id")
		if err != nil || id == "" {
			return fmt.Errorf("no stored backup ID — provide one as argument or run 'bak push' first")
		}
		remoteID = id
	}

	// 3. Download from provider.
	fmt.Printf("Downloading backup %s...\n", remoteID)
	archiveData, err := provider.Pull(remoteID)
	if err != nil {
		return fmt.Errorf("pull: %w", err)
	}

	// 4. Extract to local bak dir.
	bakDir, err := backup.BakDir()
	if err != nil {
		return fmt.Errorf("bak dir: %w", err)
	}

	backupID := time.Now().UTC().Format("20060102-150405")
	backupPath := filepath.Join(bakDir, "backups", backupID)

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("create backup dir: %w", err)
	}

	fmt.Printf("Extracting backup %s...\n", backupID)
	if err := cloud.UntarGz(string(archiveData), backupPath); err != nil {
		return fmt.Errorf("extract backup: %w", err)
	}

	fmt.Printf("✅ Backup pulled: %s\n", backupID)
	fmt.Printf("   Run 'bak restore %s' to apply it.\n", backupID)

	return nil
}
