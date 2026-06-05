package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/danielxxomg/bak-cli/internal/cloud"
	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/danielxxomg/bak-cli/internal/manifest"
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
	return runListLocal()
}

func runListLocal() error {
	bakDir, err := backup.BakDir()
	if err != nil {
		return fmt.Errorf("bak dir: %w", err)
	}

	backupsDir := filepath.Join(bakDir, "backups")

	// Check if backups directory exists.
	if _, err := os.Stat(backupsDir); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No backups found. Run 'bak backup' first.")
			return nil
		}
		return fmt.Errorf("stat backups dir: %w", err)
	}

	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		return fmt.Errorf("read backups dir: %w", err)
	}

	// Filter to directories only (backup IDs are directory names).
	var backupDirs []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			backupDirs = append(backupDirs, e)
		}
	}

	if len(backupDirs) == 0 {
		fmt.Println("No backups found. Run 'bak backup' first.")
		return nil
	}

	// Create tabwriter for formatted output.
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDATE\tPRESET\tFILES\tSIZE\tADAPTERS")
	fmt.Fprintln(w, "--\t----\t------\t-----\t----\t--------")

	for _, entry := range backupDirs {
		backupID := entry.Name()
		backupPath := filepath.Join(backupsDir, backupID)

		// Try to load manifest.
		m, err := manifest.Load(backupPath)
		if err != nil {
			// Skip backups without valid manifest.
			if verbose {
				fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", backupID, err)
			}
			continue
		}

		// Format date from backup ID (YYYYMMDD-HHMMSS).
		date := ""
		if len(backupID) >= 15 {
			date = fmt.Sprintf("%s-%s-%s %s:%s:%s",
				backupID[:4], backupID[4:6], backupID[6:8],
				backupID[9:11], backupID[11:13], backupID[13:15])
		}

		// Count total files across all adapters.
		totalFiles := 0
		for _, adapter := range m.Adapters {
			totalFiles += len(adapter.Items)
		}

		// Format size.
		sizeStr := formatSizeBytes(m.TotalSize)

		// Get adapter names (sorted for deterministic output).
		adapterNames := make([]string, 0, len(m.Adapters))
		for name := range m.Adapters {
			adapterNames = append(adapterNames, name)
		}
		sort.Strings(adapterNames)
		adapterStr := ""
		for i, name := range adapterNames {
			if i > 0 {
				adapterStr += ", "
			}
			adapterStr += name
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
			backupID, date, m.Preset, totalFiles, sizeStr, adapterStr)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}

	return nil
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

// formatSizeBytes formats bytes into a human-readable string.
func formatSizeBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
