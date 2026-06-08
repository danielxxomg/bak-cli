package cloud

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/danielxxomg/bak-cli/internal/config"
)

const rcloneTimeout = 5 * time.Minute

// execCommandContext is the function used to create exec.Cmd instances.
// It is swappable for testing.
var execCommandContext = exec.CommandContext

// RcloneProvider implements Provider by shelling out to the rclone
// binary. Backups are stored as files in the configured rclone remote
// path using "rclone copyto", "rclone cat", and "rclone lsf".
type RcloneProvider struct {
	cfg       *config.Config
	remote    string
	rcloneBin string
}

// NewRcloneProvider creates a provider that shells out to rclone.
// The remote parameter is the rclone remote path (e.g., "gdrive:bak").
// rclone must be installed and available in PATH.
func NewRcloneProvider(cfg *config.Config, remote string) *RcloneProvider {
	return &RcloneProvider{
		cfg:       cfg,
		remote:    remote,
		rcloneBin: "rclone",
	}
}

// Name returns "rclone".
func (p *RcloneProvider) Name() string {
	return "rclone"
}

// Push uploads a backup archive to the rclone remote.
// The archive is written to a temp file, copied via "rclone copyto",
// and then the temp file is cleaned up.
func (p *RcloneProvider) Push(archive []byte, meta PushMeta) (string, error) {
	if p.remote == "" {
		return "", fmt.Errorf("push rclone: remote is required")
	}

	filename := fmt.Sprintf("%s.tar.gz", meta.BackupID)
	remotePath := fmt.Sprintf("%s/%s", p.remote, filename)

	tmpFile, err := os.CreateTemp("", "bak-push-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("push rclone: create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmpFile.Write(archive); err != nil {
		if closeErr := tmpFile.Close(); closeErr != nil {
			return "", fmt.Errorf("push rclone: write temp file: %w (close error: %v)", err, closeErr)
		}
		return "", fmt.Errorf("push rclone: write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("push rclone: close temp file: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), rcloneTimeout)
	defer cancel()

	cmd := execCommandContext(ctx, p.rcloneBin, "copyto", tmpPath, remotePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		stderr := strings.TrimSpace(string(output))
		if stderr == "" {
			stderr = err.Error()
		}
		return "", fmt.Errorf("push rclone: %s: %w", stderr, err)
	}

	return meta.BackupID, nil
}

// Pull downloads a backup archive from the rclone remote by its
// backup ID using "rclone cat".
func (p *RcloneProvider) Pull(id string) ([]byte, error) {
	if p.remote == "" {
		return nil, fmt.Errorf("pull rclone: remote is required")
	}
	if id == "" {
		return nil, fmt.Errorf("pull rclone: backup ID is required")
	}

	filename := fmt.Sprintf("%s.tar.gz", id)
	remotePath := fmt.Sprintf("%s/%s", p.remote, filename)

	ctx, cancel := context.WithTimeout(context.Background(), rcloneTimeout)
	defer cancel()

	cmd := execCommandContext(ctx, p.rcloneBin, "cat", remotePath)
	output, err := cmd.Output()
	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(exitErr.Stderr))
		}
		if stderr == "" {
			stderr = err.Error()
		}
		return nil, fmt.Errorf("pull rclone: %s: %w", stderr, err)
	}

	// Return raw bytes — rclone cat returns exact file content.
	return output, nil
}

// List returns metadata for all bak backups stored in the rclone
// remote using "rclone lsf".
func (p *RcloneProvider) List() ([]BackupMeta, error) {
	if p.remote == "" {
		return nil, fmt.Errorf("list rclone: remote is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), rcloneTimeout)
	defer cancel()

	cmd := execCommandContext(ctx, p.rcloneBin, "lsf", p.remote)
	output, err := cmd.Output()
	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(exitErr.Stderr))
		}
		if stderr == "" {
			stderr = err.Error()
		}
		return nil, fmt.Errorf("list rclone: %s: %w", stderr, err)
	}

	out := strings.TrimSpace(string(output))
	if out == "" {
		return nil, nil
	}

	lines := strings.Split(out, "\n")
	metas := make([]BackupMeta, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		backupID := strings.TrimSuffix(line, ".tar.gz")
		metas = append(metas, BackupMeta{
			ID:       backupID,
			BackupID: backupID,
			URL:      fmt.Sprintf("%s/%s", p.remote, line),
		})
	}

	return metas, nil
}
