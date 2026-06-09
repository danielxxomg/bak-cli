package backup

import (
	"fmt"
	"os"
	"path/filepath"
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
