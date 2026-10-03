// Package manifest defines the backup manifest schema and provides
// serialization, deserialization, and validation routines.
package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/danielxxomg/bak-cli/internal/paths"
)

// ManifestVersion is the current schema version written by this tool.
const ManifestVersion = "0.4.0"

// parseSemver splits a version into [major, minor, patch] integers and optional prerelease string.
func parseSemver(v string) ([]int, string, error) {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	if v == "" {
		return nil, "", fmt.Errorf("version string is empty")
	}

	// Strip build metadata (+...)
	main, _, _ := strings.Cut(v, "+")
	main, prerelease, _ := strings.Cut(main, "-")

	parts := strings.Split(main, ".")
	if len(parts) > 3 {
		return nil, "", fmt.Errorf("too many parts in version %q", v)
	}

	nums := make([]int, 3)
	for i, p := range parts {
		if p == "" {
			return nil, "", fmt.Errorf("empty part in version %q", v)
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return nil, "", fmt.Errorf("invalid numeric version part %q in %q", p, v)
		}
		nums[i] = n
	}
	return nums, prerelease, nil
}

// CompareVersions compares two semver-like version strings v1 and v2.
// It returns -1 if v1 < v2, 0 if v1 == v2, and 1 if v1 > v2.
func CompareVersions(v1, v2 string) (int, error) {
	nums1, pre1, err := parseSemver(v1)
	if err != nil {
		return 0, err
	}
	nums2, pre2, err := parseSemver(v2)
	if err != nil {
		return 0, err
	}

	for i := 0; i < 3; i++ {
		if nums1[i] < nums2[i] {
			return -1, nil
		}
		if nums1[i] > nums2[i] {
			return 1, nil
		}
	}

	// Numbers are equal. Check prerelease.
	// Semver rule: 1.0.0-alpha < 1.0.0 (release version has higher precedence than prerelease)
	if pre1 != "" && pre2 == "" {
		return -1, nil
	}
	if pre1 == "" && pre2 != "" {
		return 1, nil
	}
	if pre1 < pre2 {
		return -1, nil
	}
	if pre1 > pre2 {
		return 1, nil
	}
	return 0, nil
}

// ValidateSchemaVersion checks that a manifest schema version is known and supported
// by this version of bak (i.e. not newer than ManifestVersion).
func ValidateSchemaVersion(version string) error {
	if version == "" {
		return fmt.Errorf("manifest version is empty")
	}
	cmp, err := CompareVersions(version, ManifestVersion)
	if err != nil {
		return fmt.Errorf("unsupported manifest schema version %q: %w", version, err)
	}
	if cmp > 0 {
		return fmt.Errorf("unsupported manifest schema version %s (maximum supported is %s): upgrade bak to restore this backup", version, ManifestVersion)
	}
	return nil
}

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
	Mode       uint32 `json:"mode,omitempty"`
}

// Encryption holds encryption metadata for an encrypted backup.
type Encryption struct {
	Algorithm   string `json:"algorithm"`   // "AES-256-GCM"
	KDF         string `json:"kdf"`         // "Argon2id"
	Salt        string `json:"salt"`        // hex-encoded salt
	Nonce       string `json:"nonce"`       // hex-encoded nonce
	Iterations  int    `json:"iterations"`  // Argon2id iterations (3)
	MemoryKB    int    `json:"memory_kb"`   // Argon2id memory (65536)
	Parallelism int    `json:"parallelism"` // Argon2id parallelism (4)
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
	Encryption      *Encryption                `json:"encryption,omitempty"`
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

// SetEncryption populates the Encryption metadata on the manifest.
func (m *Manifest) SetEncryption(algorithm, kdf string, salt, nonce []byte, iter, memKB, parallel int) {
	m.Encryption = &Encryption{
		Algorithm:   algorithm,
		KDF:         kdf,
		Salt:        hex.EncodeToString(salt),
		Nonce:       hex.EncodeToString(nonce),
		Iterations:  iter,
		MemoryKB:    memKB,
		Parallelism: parallel,
	}
}

// Validate checks the manifest for structural correctness and verifies
// every backed-up file's SHA-256 hash against the actual file on disk.
// progressFn is called for each file being verified (receives the backup path).
// Returns nil when all checks pass.
func (m *Manifest) Validate(backupDir string, progressFn func(string)) error {
	if m.Version == "" {
		return fmt.Errorf("manifest version is empty")
	}
	if err := ValidateSchemaVersion(m.Version); err != nil {
		return err
	}
	if len(m.Adapters) == 0 {
		return fmt.Errorf("manifest contains no adapters")
	}

	for adapterName, am := range m.Adapters {
		for _, item := range am.Items {
			if progressFn != nil {
				progressFn(item.BackupPath)
			}
			diskPath := filepath.Join(backupDir, item.BackupPath)
			// Prevent path traversal: ensure resolved path stays under backupDir.
			cleanDisk := paths.CanonicalPath(diskPath)
			cleanBackup := paths.CanonicalPath(backupDir) + "/"
			if !strings.HasPrefix(strings.ToLower(cleanDisk), strings.ToLower(cleanBackup)) {
				return fmt.Errorf("adapter %q, file %q: path traversal detected", adapterName, item.BackupPath)
			}
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
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("close: %w", err)
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}
