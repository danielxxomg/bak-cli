package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/danielxxomg/bak-cli/internal/cloud"
	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/spf13/cobra"
)

var pushProvider string

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
	rootCmd.AddCommand(pushCmd)
}

func runPush(cmd *cobra.Command, args []string) error {
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

	provider, err := reg.Get(pushProvider)
	if err != nil {
		return fmt.Errorf("provider: %w", err)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "Using provider: %s\n", provider.Name())
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

	// Security: validate resolved path stays under backupsDir.
	cleanBackup := path.Clean(filepath.ToSlash(backupPath))
	cleanBase := path.Clean(filepath.ToSlash(backupsDir)) + "/"
	if !strings.HasPrefix(cleanBackup, cleanBase) {
		return fmt.Errorf("backup ID %q resolves outside backups directory", backupID)
	}

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup %q not found", backupID)
	}

	// 3. Package backup as tar.gz.
	fmt.Printf("Packaging backup %s...\n", backupID)
	archiveData, err := cloud.TarGzDirectory(backupPath)
	if err != nil {
		return fmt.Errorf("package backup: %w", err)
	}

	// 4. Push via provider.
	hostname, err := os.Hostname()
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "warning: hostname: %v\n", err)
		}
		hostname = "unknown"
	}
	rawArchive, err := base64.StdEncoding.DecodeString(archiveData)
	if err != nil {
		return fmt.Errorf("decode archive: %w", err)
	}

	id, err := provider.Push(rawArchive, cloud.PushMeta{
		BackupID:  backupID,
		CreatedAt: time.Now().UTC(),
		Hostname:  hostname,
		OS:        runtime.GOOS,
	})
	if err != nil {
		return fmt.Errorf("push: %w", err)
	}

	fmt.Printf("✅ Pushed to %s: %s\n", provider.Name(), id)
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
		return "", fmt.Errorf("no backups found — run 'bak backup' first")
	}

	// Sort descending (newest first) by timestamp ID format.
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] > ids[j]
	})

	if verbose {
		fmt.Fprintf(os.Stderr, "Found %d backup(s), using latest: %s\n", len(ids), ids[0])
	}

	return ids[0], nil
}
