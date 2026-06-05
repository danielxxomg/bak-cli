package backup

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
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
	cleanBackup := path.Clean(filepath.ToSlash(backupDir))
	cleanBase := path.Clean(filepath.ToSlash(backupsDir)) + "/"
	if !strings.HasPrefix(cleanBackup, cleanBase) && cleanBackup != path.Clean(filepath.ToSlash(backupsDir)) {
		return "", fmt.Errorf("backup ID %q resolves outside backups directory", id)
	}

	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return "", fmt.Errorf("backup %q not found", id)
	} else if err != nil {
		return "", fmt.Errorf("stat backup dir: %w", err)
	}

	return backupDir, nil
}
