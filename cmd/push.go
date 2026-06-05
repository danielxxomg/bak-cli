package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/danielxxomg/bak-cli/internal/cloud"
	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/spf13/cobra"
)

// pushCmd represents the push command.
var pushCmd = &cobra.Command{
	Use:   "push [backup-id]",
	Short: "Push a backup to GitHub Gist",
	Long: `Package a local backup and push it to a private GitHub Gist for
cloud sync across machines.

If no backup ID is provided, the most recent backup is used.

Requires a GitHub token configured via 'bak login' or the
GITHUB_TOKEN environment variable.

Examples:
  bak push                          # push latest backup
  bak push 20260604-150405          # push a specific backup`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPush,
}

func init() {
	rootCmd.AddCommand(pushCmd)
}

func runPush(cmd *cobra.Command, args []string) error {
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

	// 2. Find backup.
	bakDir, err := backup.BakDir()
	if err != nil {
		return fmt.Errorf("bak dir: %w", err)
	}

	backupsDir := filepath.Join(bakDir, "backups")
	backupID, err := resolveBackupID(backupsDir, args)
	if err != nil {
		return err
	}

	backupPath := filepath.Join(backupsDir, backupID)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup %q not found at %s", backupID, backupPath)
	}

	// 3. Package backup as tar.gz + base64.
	fmt.Printf("Packaging backup %s...\n", backupID)
	archiveData, err := cloud.TarGzDirectory(backupPath)
	if err != nil {
		return fmt.Errorf("package backup: %w", err)
	}

	// 4. Push to Gist.
	desc := fmt.Sprintf("bak backup %s — %s", backupID, time.Now().UTC().Format(time.RFC3339))

	var gistID string
	existingID, err := cfg.Get("github.gist_id")
	if err != nil {
		// No existing gist ID — will create new.
		existingID = ""
	}

	if existingID != "" {
		fmt.Printf("Updating existing Gist %s...\n", existingID)
		if err := cloud.UpdateGist(token, existingID, desc, []cloud.GistFile{
			{Filename: "backup.tar.gz", Content: archiveData},
		}); err != nil {
			return fmt.Errorf("update gist: %w", err)
		}
		gistID = existingID
	} else {
		fmt.Print("Creating new private Gist... ")
		id, err := cloud.CreateGist(token, desc, []cloud.GistFile{
			{Filename: "backup.tar.gz", Content: archiveData},
		})
		if err != nil {
			return fmt.Errorf("create gist: %w", err)
		}
		gistID = id
		fmt.Println("done")

		// Save gist ID for future updates.
		if err := cfg.Set("github.gist_id", gistID); err != nil {
			return fmt.Errorf("save gist ID: %w", err)
		}
	}

	fmt.Printf("✅ Pushed to GitHub Gist: https://gist.github.com/%s\n", gistID)
	return nil
}

// resolveBackupID returns the backup ID from args or finds the most
// recent backup when no argument is given.
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
		return "", fmt.Errorf("no backups found in %s — run 'bak backup' first", backupsDir)
	}

	// Sort descending (newest first) by timestamp ID format.
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] > ids[j]
	})

	if verbose {
		fmt.Printf("Found %d backup(s), using latest: %s\n", len(ids), ids[0])
	}

	return ids[0], nil
}
