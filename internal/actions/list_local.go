package actions

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/danielxxomg/bak-cli/internal/manifest"
)

// RunListLocal scans the local backups directory and writes a table of
// all backups to out. Verbose warnings (e.g., skipped corrupt backups)
// are written to errOut when verbose is true.
func RunListLocal(bakDir string, verbose bool, out, errOut io.Writer) error {
	backupDirs, err := readBackupDirs(bakDir, out)
	if err != nil {
		return err
	}
	if backupDirs == nil {
		return nil // "No backups found" already written by readBackupDirs
	}

	// Create tabwriter for formatted output.
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "ID\tDATE\tPRESET\tFILES\tSIZE\tADAPTERS"); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	if _, err := fmt.Fprintln(w, "--\t----\t------\t-----\t----\t--------"); err != nil {
		return fmt.Errorf("write separator: %w", err)
	}

	for _, entry := range backupDirs {
		backupID := entry.Name()
		backupPath := filepath.Join(filepath.Join(bakDir, "backups"), backupID)

		// Try to load manifest.
		m, err := manifest.Load(backupPath)
		if err != nil {
			if verbose {
				// Best-effort verbose warning — write errors are non-fatal diagnostics.
				fmt.Fprintf(errOut, "warning: skipping %s: %v\n", backupID, err) //nolint:errcheck
			}
			continue
		}

		if _, err := fmt.Fprint(w, formatBackupRow(backupID, m)); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}

	return nil
}

// readBackupDirs returns the backup subdirectories under bakDir/backups.
// When the directory is missing or contains no backups it writes the
// "No backups found" guidance to out and returns (nil, nil) so the caller
// can return cleanly. A non-nil error is returned only for unexpected
// filesystem failures.
func readBackupDirs(bakDir string, out io.Writer) ([]os.DirEntry, error) {
	backupsDir := filepath.Join(bakDir, "backups")

	if _, err := os.Stat(backupsDir); err != nil {
		if os.IsNotExist(err) {
			if err := writeNoBackupsFound(out); err != nil {
				return nil, fmt.Errorf("write output: %w", err)
			}
			return nil, nil
		}
		return nil, fmt.Errorf("stat backups dir: %w", err)
	}

	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		return nil, fmt.Errorf("read backups dir: %w", err)
	}

	// Filter to directories only (backup IDs are directory names).
	var backupDirs []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			backupDirs = append(backupDirs, e)
		}
	}

	if len(backupDirs) == 0 {
		if err := writeNoBackupsFound(out); err != nil {
			return nil, fmt.Errorf("write output: %w", err)
		}
		return nil, nil
	}

	return backupDirs, nil
}

// FormatSizeBytes formats bytes into a human-readable string.
// Supports magnitudes up to exabytes (EB). Negative values are
// formatted with the raw byte count and " B" suffix.
func FormatSizeBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit && bytes > -unit {
		return fmt.Sprintf("%d B", bytes)
	}
	// Work with absolute value for the magnitude computation, then
	// reapply the sign in the formatted result.
	abs := bytes
	sign := ""
	if bytes < 0 {
		abs = -bytes
		sign = "-"
	}
	div, exp := int64(unit), 0
	for n := abs / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%s%.1f %cB", sign, float64(abs)/float64(div), "KMGTPE"[exp])
}

// formatBackupDate parses a backup ID of the form YYYYMMDD-HHMMSS into a
// readable date string. Returns an empty string when the ID is too short
// to parse (fewer than 15 characters).
func formatBackupDate(backupID string) string {
	if len(backupID) < 15 {
		return ""
	}
	return fmt.Sprintf("%s-%s-%s %s:%s:%s",
		backupID[:4], backupID[4:6], backupID[6:8],
		backupID[9:11], backupID[11:13], backupID[13:15])
}

// formatBackupRow formats a single backup into a tab-separated table row
// (ID, date, preset, file count, size, adapters). Adapter names are sorted
// and joined for deterministic output. The returned string ends with a
// newline.
func formatBackupRow(backupID string, m *manifest.Manifest) string {
	date := formatBackupDate(backupID)
	adapterNames := make([]string, 0, len(m.Adapters))
	for name := range m.Adapters {
		adapterNames = append(adapterNames, name)
	}
	sort.Strings(adapterNames)
	adapterStr := strings.Join(adapterNames, ", ")
	sizeStr := FormatSizeBytes(m.TotalSize)
	return fmt.Sprintf("%s\t%s\t%s\t%d\t%s\t%s\n",
		backupID, date, m.Preset, m.FileCount, sizeStr, adapterStr)
}

// writeNoBackupsFound writes the "No backups found" guidance message to out
// and returns any write error. Shared by the early-return branches in
// RunListLocal to keep the message in one place.
func writeNoBackupsFound(out io.Writer) error {
	_, err := fmt.Fprintln(out, "No backups found. Run 'bak backup' first.")
	return err
}
