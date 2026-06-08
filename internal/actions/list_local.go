package actions

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"

	"github.com/danielxxomg/bak-cli/internal/manifest"
)

// RunListLocal scans the local backups directory and writes a table of
// all backups to out. Verbose warnings (e.g., skipped corrupt backups)
// are written to errOut when verbose is true.
func RunListLocal(bakDir string, verbose bool, out, errOut io.Writer) error {
	backupsDir := filepath.Join(bakDir, "backups")

	// Check if backups directory exists.
	if _, err := os.Stat(backupsDir); err != nil {
		if os.IsNotExist(err) {
			_, _ = fmt.Fprintln(out, "No backups found. Run 'bak backup' first.")
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
		_, _ = fmt.Fprintln(out, "No backups found. Run 'bak backup' first.")
		return nil
	}

	// Create tabwriter for formatted output.
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tDATE\tPRESET\tFILES\tSIZE\tADAPTERS")
	_, _ = fmt.Fprintln(w, "--\t----\t------\t-----\t----\t--------")

	for _, entry := range backupDirs {
		backupID := entry.Name()
		backupPath := filepath.Join(backupsDir, backupID)

		// Try to load manifest.
		m, err := manifest.Load(backupPath)
		if err != nil {
			if verbose {
				_, _ = fmt.Fprintf(errOut, "warning: skipping %s: %v\n", backupID, err)
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

		// Format size.
		sizeStr := FormatSizeBytes(m.TotalSize)

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
			backupID, date, m.Preset, m.FileCount, sizeStr, adapterStr)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}

	return nil
}

// FormatSizeBytes formats bytes into a human-readable string.
func FormatSizeBytes(bytes int64) string {
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
