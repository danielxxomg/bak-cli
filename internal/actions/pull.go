package actions

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/danielxxomg/bak-cli/internal/cloud"
	"github.com/danielxxomg/bak-cli/internal/config"
	"github.com/danielxxomg/bak-cli/internal/crypto"
	"github.com/spf13/cobra"
)

// PullAction encapsulates the pull-from-cloud workflow with injectable
// filesystem for directory creation and path resolution.
type PullAction struct {
	FS       FileSystem
	Provider string
	Profile  string
	Verbose  bool

	// Factory creates cloud providers on demand.
	Factory ProviderFactory
}

// Run downloads a backup from a cloud backend and reconstructs it locally.
func (a *PullAction) Run(cmd *cobra.Command, args []string) error {
	// 1. Determine home and bak directories.
	homeDir, err := a.FS.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}
	bakDir := filepath.Join(homeDir, ".bak")

	// 2. Load config (for stored backup ID resolution).
	cfg, err := config.Load()
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
		fmt.Fprintf(os.Stderr, "Using provider: %s\n", provider.Name())
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
	fmt.Printf("Downloading backup %s...\n", remoteID)
	archiveData, err := provider.Pull(remoteID)
	if err != nil {
		return fmt.Errorf("pull: %w", err)
	}

	archiveStr := string(archiveData)

	// 5. Decrypt if encrypted.
	if rawBytes, decErr := base64.StdEncoding.DecodeString(archiveStr); decErr == nil && crypto.IsEncrypted(rawBytes) {
		password, err := crypto.GetPassword("Enter decryption password: ")
		if err != nil {
			return fmt.Errorf("decryption password: %w", err)
		}

		decrypted, err := crypto.Decrypt(rawBytes, password)
		if err != nil {
			return fmt.Errorf("decrypt archive: %w", err)
		}

		archiveStr = base64.StdEncoding.EncodeToString(decrypted)

		if a.Verbose {
			fmt.Fprintf(os.Stderr, "Decrypted archive\n")
		}
	}

	// 6. Extract to local bak dir.
	backupID := time.Now().UTC().Format("20060102-150405")
	backupPath := filepath.Join(bakDir, "backups", backupID)

	if err := a.FS.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("create backup dir: %w", err)
	}

	fmt.Printf("Extracting backup %s...\n", backupID)
	if err := cloud.UntarGz(archiveStr, backupPath); err != nil {
		return fmt.Errorf("extract backup: %w", err)
	}

	fmt.Printf("✅ Backup pulled: %s\n", backupID)
	fmt.Printf("   Run 'bak restore %s' to apply it.\n", backupID)

	return nil
}
