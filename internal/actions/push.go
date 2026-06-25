package actions

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/danielxxomg/bak-cli/internal/cloud"
	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/danielxxomg/bak-cli/internal/crypto"
	"github.com/danielxxomg/bak-cli/internal/paths"
)

// PushAction encapsulates the push-to-cloud workflow with injectable
// filesystem and provider factory.
type PushAction struct {
	FS       FileSystem
	Provider string
	Profile  string
	Verbose  bool

	// ProgressFn is an optional callback invoked at coarse milestones during push.
	// When nil (default), no progress is reported.
	ProgressFn func(step string, done, total int)

	// Stdout receives informational output. Nil falls back to os.Stdout.
	Stdout io.Writer
	// Stderr receives warnings and error diagnostics. Nil falls back to os.Stderr.
	Stderr io.Writer

	// Factory creates cloud providers on demand. When nil, the action
	// falls back to the real cloud provider registry (backward compat).
	Factory ProviderFactory

	// HostnameFn returns the current hostname. Nil falls back to os.Hostname.
	HostnameFn HostnameFunc

	// ConfigLoader loads the bak-cli configuration. When nil, falls back
	// to config.Load(). Injected via struct field for testability.
	ConfigLoader func() (*config.Config, error)
}

// Run packages a local backup and pushes it to a cloud backend.
func (a *PushAction) Run(args []string) error {
	out := a.Stdout
	if out == nil {
		out = os.Stdout
	}
	errOut := a.Stderr
	if errOut == nil {
		errOut = os.Stderr
	}
	// 1. Determine backups directory.
	if a.FS == nil {
		return fmt.Errorf("filesystem not configured")
	}
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
	cleanBackup := paths.CanonicalPath(backupPath)
	cleanBase := paths.CanonicalPath(backupsDir) + "/"
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
		warnf(errOut, "Using provider: %s\n", provider.Name())
	}

	// 4-6. Package, optionally encrypt, and push the backup to the provider.
	id, err := a.publishArchive(provider, backupPath, backupID, out, errOut)
	if err != nil {
		return err
	}

	infof(out, "✅ Pushed to %s: %s\n", provider.Name(), id)
	if a.ProgressFn != nil {
		a.ProgressFn("Complete", 2, 2)
	}
	return nil
}

// publishArchive performs the push workflow's I/O-heavy phase: package the
// backup directory as a tar.gz, base64-decode it, apply optional encryption,
// and upload to the provider, reporting progress along the way. It returns
// the uploaded ID. Extracted from Run to keep PushAction.Run within the funlen
// statement budget.
func (a *PushAction) publishArchive(
	provider cloud.Provider,
	backupPath, backupID string,
	out, errOut io.Writer,
) (string, error) {
	// 4. Package backup as tar.gz.
	infof(out, "Packaging backup %s...\n", backupID)
	if a.ProgressFn != nil {
		a.ProgressFn("Packaging", 0, 2)
	}
	archiveData, err := cloud.TarGzDirectory(backupPath)
	if err != nil {
		return "", fmt.Errorf("package backup: %w", err)
	}
	if a.ProgressFn != nil {
		a.ProgressFn("Packaging", 1, 2)
	}

	hostname := backup.ResolveHostname(a.HostnameFn, a.Verbose, errOut)
	rawArchive, err := base64.StdEncoding.DecodeString(archiveData)
	if err != nil {
		return "", fmt.Errorf("decode archive: %w", err)
	}

	// 5. Encrypt archive if the profile has encryption enabled.
	rawArchive, err = a.encryptArchiveIfNeeded(rawArchive, errOut)
	if err != nil {
		return "", err
	}

	// 6. Push via provider.
	if a.ProgressFn != nil {
		a.ProgressFn("Uploading", 1, 2)
	}
	id, err := provider.Push(rawArchive, cloud.PushMeta{
		BackupID:  backupID,
		CreatedAt: time.Now().UTC(),
		Hostname:  hostname,
		OS:        runtime.GOOS,
	})
	if err != nil {
		return "", fmt.Errorf("push: %w", err)
	}
	return id, nil
}

// encryptArchiveIfNeeded encrypts the raw archive bytes when the configured
// profile has encryption enabled, prompting for the password and returning the
// ciphertext. When encryption is disabled it returns rawArchive unchanged.
// Extracted from Run to keep PushAction.Run within the funlen statement budget.
func (a *PushAction) encryptArchiveIfNeeded(rawArchive []byte, errOut io.Writer) ([]byte, error) {
	encrypt, err := a.shouldEncrypt()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if !encrypt {
		return rawArchive, nil
	}

	password, err := crypto.GetPassword("Enter encryption password: ")
	if err != nil {
		return nil, fmt.Errorf("encryption password: %w", err)
	}

	encrypted, err := crypto.Encrypt(rawArchive, password)
	if err != nil {
		return nil, fmt.Errorf("encrypt archive: %w", err)
	}

	if a.Verbose {
		warnf(errOut, "Archive encrypted\n")
	}

	return encrypted, nil
}

// shouldEncrypt checks whether the configured profile has encryption
// enabled. It returns (true, nil) when the profile exists and has
// Encryption.Enabled set, (false, nil) when the profile is missing or
// encryption is not enabled, and (false, err) when config loading fails.
func (a *PushAction) shouldEncrypt() (bool, error) {
	cfg, err := loadConfigOr(a.ConfigLoader)
	if err != nil {
		return false, err
	}

	profile, ok := cfg.Profiles[a.Profile]
	if !ok {
		return false, nil
	}

	if profile.Encryption != nil && profile.Encryption.Enabled {
		return true, nil
	}

	return false, nil
}

// resolveBackupID returns the backup ID from args or finds the most
// recent backup when no argument is given. The sort/dedup of backup IDs
// delegates to backup.SortedBackupIDs (the canonical resolver core); the
// ReadDir stays on the injected FileSystem so error paths stay testable.
func (a *PushAction) resolveBackupID(backupsDir string, args []string) (string, error) {
	if len(args) > 0 && args[0] != "" {
		return args[0], nil
	}

	entries, err := a.FS.ReadDir(backupsDir)
	if err != nil {
		return "", fmt.Errorf("read backups dir: %w", err)
	}

	ids := backup.SortedBackupIDs(entries)
	if len(ids) == 0 {
		return "", fmt.Errorf("no backups found — run 'bak backup' first")
	}

	if a.Verbose {
		stderr := a.Stderr
		if stderr == nil {
			stderr = os.Stderr
		}
		warnf(stderr, "Found %d backup(s), using latest: %s\n", len(ids), ids[0])
	}

	return ids[0], nil
}
