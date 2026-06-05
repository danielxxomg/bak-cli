// Package manifest defines the backup manifest schema and provides
// serialization, deserialization, and validation routines.
package manifest

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ManifestVersion is the current schema version written by this tool.
const ManifestVersion = "0.1.0"

// AdapterManifest records the items backed up by a single adapter.
type AdapterManifest struct {
	VersionDetected string `json:"version_detected,omitempty"`
	ConfigDir       string `json:"config_dir"`
	Items           []Item `json:"items"`
}

// Item describes one backed-up file or directory.
type Item struct {
	Category   string `json:"category"`
	SourcePath string `json:"source_path"`
	BackupPath string `json:"backup_path"`
	Hash       string `json:"hash"`
	Size       int64  `json:"size"`
}

// Manifest is the top-level backup descriptor.
type Manifest struct {
	Version         string                     `json:"version"`
	ID              string                     `json:"id"`
	CreatedAt       time.Time                  `json:"created_at"`
	OSSource        string                     `json:"os_source"`
	Hostname        string                     `json:"hostname,omitempty"`
	BakVersion      string                     `json:"bak_version"`
	Preset          string                     `json:"preset"`
	Categories      []string                   `json:"categories"`
	Adapters        map[string]AdapterManifest `json:"adapters"`
	SecretsExcluded bool                       `json:"secrets_excluded"`
	FileCount       int                        `json:"file_count"`
	TotalSize       int64                      `json:"total_size"`
}

// New creates a Manifest pre-populated with metadata.
func New(id, osSource, hostname, bakVersion, preset string, categories []string) *Manifest {
	return &Manifest{
		Version:    ManifestVersion,
		ID:         id,
		CreatedAt:  time.Now().UTC(),
		OSSource:   osSource,
		Hostname:   hostname,
		BakVersion: bakVersion,
		Preset:     preset,
		Categories: categories,
		Adapters:   make(map[string]AdapterManifest),
	}
}

// Save writes the manifest as JSON to the given directory.
func (m *Manifest) Save(dir string) error {
	path := filepath.Join(dir, "manifest.json")
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	return nil
}

// Load reads a manifest from a directory containing manifest.json.
func Load(dir string) (*Manifest, error) {
	path := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

// AddAdapter appends (or replaces) adapter-level data in the manifest.
func (m *Manifest) AddAdapter(name, versionDetected, configDir string, items []Item) {
	m.Adapters[name] = AdapterManifest{
		VersionDetected: versionDetected,
		ConfigDir:       configDir,
		Items:           items,
	}
	m.recount()
}

// Validate checks the manifest for structural correctness and verifies
// every backed-up file's SHA-256 hash against the actual file on disk.
// Returns nil when all checks pass.
func (m *Manifest) Validate(backupDir string) error {
	if m.Version == "" {
		return fmt.Errorf("manifest version is empty")
	}
	if len(m.Adapters) == 0 {
		return fmt.Errorf("manifest contains no adapters")
	}

	for adapterName, am := range m.Adapters {
		for _, item := range am.Items {
			diskPath := filepath.Join(backupDir, item.BackupPath)
			actualHash, err := hashFile(diskPath)
			if err != nil {
				return fmt.Errorf("adapter %q, file %q: %w", adapterName, item.BackupPath, err)
			}
			if actualHash != item.Hash {
				return fmt.Errorf("adapter %q, file %q: hash mismatch (expected %s, got %s)",
					adapterName, item.BackupPath, item.Hash, actualHash)
			}
		}
	}
	return nil
}

// recount updates file_count and total_size from the adapter contents.
func (m *Manifest) recount() {
	count := 0
	var total int64
	for _, am := range m.Adapters {
		count += len(am.Items)
		for _, item := range am.Items {
			total += item.Size
		}
	}
	m.FileCount = count
	m.TotalSize = total
}

// hashFile computes the SHA-256 hex digest of a file.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash: %w", err)
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}
