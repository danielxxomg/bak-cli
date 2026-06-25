package actions

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/danielxxomg/bak-cli/internal/cloud"
	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/danielxxomg/bak-cli/internal/crypto"
)

// PullAction encapsulates the pull-from-cloud workflow with injectable
// filesystem for directory creation and path resolution.
type PullAction struct {
	FS       FileSystem
	Provider string
	Profile  string
	Verbose  bool

	// Stdout receives informational output. When nil, defaults to os.Stdout.
	Stdout io.Writer
	// Stderr receives diagnostic and error output. When nil, defaults to os.Stderr.
	Stderr io.Writer

	// ProgressFn is an optional callback invoked at coarse milestones during pull.
	// When nil (default), no progress is reported.
	ProgressFn func(step string, done, total int)

	// ConfigLoader loads the bak-cli configuration. Defaults to config.Load.
	ConfigLoader func() (*config.Config, error)

	// Factory creates cloud providers on demand.
	Factory ProviderFactory
}

// stdout returns the Stdout writer or os.Stdout when nil.
func (a *PullAction) stdout() io.Writer {
	if a.Stdout != nil {
		return a.Stdout
	}
	return os.Stdout
}

// stderr returns the Stderr writer or os.Stderr when nil.
func (a *PullAction) stderr() io.Writer {
	if a.Stderr != nil {
		return a.Stderr
	}
	return os.Stderr
}

// Run downloads a backup from a cloud backend and reconstructs it locally.
func (a *PullAction) Run(args []string) error {
	if a.FS == nil {
		return fmt.Errorf("filesystem is not configured")
	}

	// 1. Determine home and bak directories.
	homeDir, err := a.FS.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}
	bakDir := filepath.Join(homeDir, ".bak")

	// 2. Load config (for stored backup ID resolution).
	cfg, err := loadConfigOr(a.ConfigLoader)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
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
		fmt.Fprintf(a.stderr(), "Using provider: %s\n", provider.Name()) //nolint:errcheck
	}

	// 4. Resolve remote backup ID.
	var remoteID string
	if len(args) > 0 && args[0] != "" {
		remoteID = args[0]
	} else {
		id, err := cfg.Get("github.gist_id")
		if err != nil || id == "" {
			return fmt.Errorf("no stored backup ID — provide one as argument or run 'bak push' first")
		}
		remoteID = id
	}

	// 4. Download from provider.
	fmt.Fprintf(a.stdout(), "Downloading backup %s...\n", remoteID) //nolint:errcheck
	if a.ProgressFn != nil {
		a.ProgressFn("Downloading", 0, 2)
	}
	archiveData, err := provider.Pull(remoteID)
	if err != nil {
		return fmt.Errorf("pull: %w", err)
	}
	if a.ProgressFn != nil {
		a.ProgressFn("Downloading", 1, 2)
	}

	archiveStr := string(archiveData)

	// 5. Decrypt if encrypted.
	archiveStr, err = a.decryptArchiveIfNeeded(archiveStr)
	if err != nil {
		return err
	}

	// 6. Extract to local bak dir.
	backupID := time.Now().UTC().Format("20060102-150405")
	backupPath := filepath.Join(bakDir, "backups", backupID)

	if err := a.FS.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("create backup dir: %w", err)
	}

	fmt.Fprintf(a.stdout(), "Extracting backup %s...\n", backupID) //nolint:errcheck
	if a.ProgressFn != nil {
		a.ProgressFn("Extracting", 1, 2)
	}
	if err := cloud.UntarGz(archiveStr, backupPath); err != nil {
		return fmt.Errorf("extract backup: %w", err)
	}

	fmt.Fprintf(a.stdout(), "✅ Backup pulled: %s\n", backupID)                  //nolint:errcheck
	fmt.Fprintf(a.stdout(), "   Run 'bak restore %s' to apply it.\n", backupID) //nolint:errcheck
	if a.ProgressFn != nil {
		a.ProgressFn("Complete", 2, 2)
	}

	return nil
}

// decryptArchiveIfNeeded decodes + decrypts the archive when it is recognized
// as encrypted, prompting for the password and returning the re-base64-encoded
// plaintext archive string. When the archive is not encrypted (decode failed or
// not recognized) it returns the archiveString unchanged. Extracted from Run to
// keep PullAction.Run within the funlen statement budget.
func (a *PullAction) decryptArchiveIfNeeded(archiveStr string) (string, error) {
	rawBytes, decErr := base64.StdEncoding.DecodeString(archiveStr)
	if decErr != nil {
		// Not base64-encoded, so it cannot be an encrypted archive blob; return
		// the archive unchanged so the caller proceeds with extraction.
		return archiveStr, nil //nolint:nilerr // intentional: decode failure means the archive is not an encrypted blob
	}
	if !crypto.IsEncrypted(rawBytes) {
		return archiveStr, nil
	}

	password, err := crypto.GetPassword("Enter decryption password: ")
	if err != nil {
		return "", fmt.Errorf("decryption password: %w", err)
	}

	decrypted, err := crypto.Decrypt(rawBytes, password)
	if err != nil {
		return "", fmt.Errorf("decrypt archive: %w", err)
	}

	if a.Verbose {
		fmt.Fprintf(a.stderr(), "Decrypted archive\n") //nolint:errcheck
	}

	return base64.StdEncoding.EncodeToString(decrypted), nil
}
