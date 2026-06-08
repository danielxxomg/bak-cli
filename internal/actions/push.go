package actions

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

	"github.com/spf13/cobra"

	"github.com/danielxxomg/bak-cli/internal/cloud"
)

// PushAction encapsulates the push-to-cloud workflow with injectable
// filesystem and provider factory.
type PushAction struct {
	FS       FileSystem
	Provider string
	Profile  string
	Verbose  bool

	// Factory creates cloud providers on demand. When nil, the action
	// falls back to the real cloud provider registry (backward compat).
	Factory ProviderFactory

	// HostnameFn returns the current hostname. Nil falls back to os.Hostname.
	HostnameFn HostnameFunc
}

// Run packages a local backup and pushes it to a cloud backend.
func (a *PushAction) Run(cmd *cobra.Command, args []string) error {
	// 1. Determine backups directory.
	homeDir, err := a.FS.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}
	bakDir := filepath.Join(homeDir, ".bak")
	backupsDir := filepath.Join(bakDir, "backups")

	// 2. Resolve backup ID.
	backupID, err := a.resolveBackupID(backupsDir, args)
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

	if _, err := a.FS.Stat(backupPath); err != nil {
		return fmt.Errorf("backup %q not found", backupID)
	}

	// 3. Resolve provider via injected factory.
	if a.Factory == nil {
		return fmt.Errorf("provider factory is not configured")
	}

	provider, err := a.Factory.CreateProvider(a.Provider)
	if err != nil {
		return fmt.Errorf("provider: %w", err)
	}
	if a.Verbose {
		fmt.Fprintf(os.Stderr, "Using provider: %s\n", provider.Name())
	}

	// 4. Package backup as tar.gz.
	fmt.Printf("Packaging backup %s...\n", backupID)
	archiveData, err := cloud.TarGzDirectory(backupPath)
	if err != nil {
		return fmt.Errorf("package backup: %w", err)
	}

	// 5. Push via provider.
	hostname := "unknown"
	if a.HostnameFn != nil {
		if h, err := a.HostnameFn(); err == nil {
			hostname = h
		} else if a.Verbose {
			fmt.Fprintf(os.Stderr, "warning: hostname: %v\n", err)
		}
	} else {
		if h, err := os.Hostname(); err == nil {
			hostname = h
		} else if a.Verbose {
			fmt.Fprintf(os.Stderr, "warning: hostname: %v\n", err)
		}
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
func (a *PushAction) resolveBackupID(backupsDir string, args []string) (string, error) {
	if len(args) > 0 && args[0] != "" {
		return args[0], nil
	}

	entries, err := a.FS.ReadDir(backupsDir)
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

	sort.Slice(ids, func(i, j int) bool {
		return ids[i] > ids[j]
	})

	if a.Verbose {
		fmt.Fprintf(os.Stderr, "Found %d backup(s), using latest: %s\n", len(ids), ids[0])
	}

	return ids[0], nil
}
