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

// pullCmd represents the pull command.
var pullCmd = &cobra.Command{
	Use:   "pull [gist-id]",
	Short: "Pull a backup from GitHub Gist",
	Long: `Download a backup from a private GitHub Gist and reconstruct it
locally in ~/.bak/backups/.

If no Gist ID is provided, the ID stored from a previous push is used.

Requires a GitHub token configured via 'bak login' or the
GITHUB_TOKEN environment variable.

Examples:
  bak pull                          # pull from saved gist ID
  bak pull abc123def456             # pull from specific gist`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPull,
}

func init() {
	rootCmd.AddCommand(pullCmd)
}

func runPull(cmd *cobra.Command, args []string) error {
	// 1. Resolve token.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	token, source := cloud.ResolveToken(cfg)
	if token == "" {
		return fmt.Errorf("no GitHub token found — run 'bak login' or set GITHUB_TOKEN")
	}
	if verbose {
		fmt.Printf("Using token from %s\n", source)
	}

	// 2. Resolve gist ID.
	var gistID string
	if len(args) > 0 && args[0] != "" {
		gistID = args[0]
	} else {
		id, err := cfg.Get("github.gist_id")
		if err != nil || id == "" {
			return fmt.Errorf("no stored Gist ID — provide one as argument or run 'bak push' first")
		}
		gistID = id
	}

	// 3. Download gist.
	fmt.Printf("Downloading Gist %s...\n", gistID)
	files, err := cloud.GetGist(token, gistID)
	if err != nil {
		return fmt.Errorf("get gist: %w", err)
	}

	// Find the backup archive in the gist files.
	var archiveData string
	for _, f := range files {
		if f.Filename == "backup.tar.gz" {
			archiveData = f.Content
			break
		}
	}
	if archiveData == "" {
		return fmt.Errorf("no backup.tar.gz found in Gist %s", gistID)
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

	fmt.Printf("Extracting to %s...\n", backupPath)
	if err := cloud.UntarGz(archiveData, backupPath); err != nil {
		return fmt.Errorf("extract backup: %w", err)
	}

	fmt.Printf("✅ Backup pulled to: %s\n", backupPath)
	fmt.Printf("   Run 'bak restore %s' to apply it.\n", backupID)

	return nil
}
