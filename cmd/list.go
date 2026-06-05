package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"

	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/danielxxomg/bak-cli/internal/manifest"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all local backups",
	Long: `Scans ~/.bak/backups/ and displays a table of all local backups
with their ID, date, preset, file count, and size.

Use 'bak restore <id>' to restore a backup.
Use 'bak push <id>' to upload a backup to GitHub Gist.

Examples:
  bak list`,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
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
