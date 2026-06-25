package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/danielxxomg/bak-cli/internal/paths"
)

// ResolveBackupID validates the id, builds the backup directory path,
// enforces path traversal prevention, and checks existence.
func ResolveBackupID(id string) (backupDir string, err error) {
	bakDir, err := BakDir()
	if err != nil {
		return "", fmt.Errorf("bak dir: %w", err)
	}

	backupsDir := filepath.Join(bakDir, "backups")
	backupDir = filepath.Join(backupsDir, id)

	// Security: validate the resolved path stays under backupsDir.
	cleanBackup := paths.CanonicalPath(backupDir)
	cleanBase := paths.CanonicalPath(backupsDir) + "/"
	if !strings.HasPrefix(cleanBackup, cleanBase) && cleanBackup != paths.CanonicalPath(backupsDir) {
		return "", fmt.Errorf("backup ID %q resolves outside backups directory", id)
	}

	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return "", fmt.Errorf("backup %q not found", id)
	} else if err != nil {
		return "", fmt.Errorf("stat backup dir: %w", err)
	}

	return backupDir, nil
}

// SortedBackupIDs collects the directory names from a ReadDir result and
// returns them sorted descending (lexicographic order matches the
// chronological timestamp order, so index 0 is the most recent backup).
// Non-directory entries are ignored. The result is nil when no backup
// directories are present.
//
// This is the pure core of backup-ID resolution: callers that read the
// backups directory through an injected filesystem (PushAction,
// CleanupAction) call this after their own ReadDir so FS injection and
// error handling stay testable, while the sort/dedup logic stays canonical.
func SortedBackupIDs(entries []os.DirEntry) []string {
	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ids)))
	return ids
}

// ListBackupIDs returns the backup IDs in backupsDir sorted descending
// (most recent first), using os.ReadDir. It returns an error wrapping the
// underlying ReadDir failure; an empty directory yields an empty (non-nil
// assumption) slice and no error, so callers decide how to handle "no
// backups".
func ListBackupIDs(backupsDir string) ([]string, error) {
	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		return nil, fmt.Errorf("read backups dir: %w", err)
	}
	return SortedBackupIDs(entries), nil
}

// LatestBackupID returns the most recent backup ID in backupsDir, or an
// error if the directory contains no backups (or cannot be read).
func LatestBackupID(backupsDir string) (string, error) {
	ids, err := ListBackupIDs(backupsDir)
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("no backups found — run 'bak backup' first")
	}
	return ids[0], nil
}
