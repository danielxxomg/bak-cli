package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/danielxxomg/bak-cli/internal/manifest"
)

// BackupInfo represents summary metadata for a single local backup.
type BackupInfo struct {
	ID     string
	Date   string
	Size   string
	Status string
	Cloud  string
}

// ListBackupsAction scans the backups directory and loads manifests into
// BackupInfo summaries. It supports struct-field injection and provides a
// usable zero value.
type ListBackupsAction struct {
	// FS is the filesystem interface used for directory traversal and reading
	// manifests. Defaults to OSFileSystem when nil.
	FS FileSystem

	// BakDir is the root bak data directory containing the "backups" subfolder.
	// When empty, defaults to backup.BakDir().
	BakDir string

	// FilterInvalidIDs restricts results to valid backup IDs (YYYYMMDD-HHMMSS).
	// When false (default), any directory containing a valid manifest is included.
	FilterInvalidIDs bool

	// SortDescending orders backups newest-first when true. When false (default),
	// backups are ordered ascending by directory name.
	SortDescending bool
}

// Run executes the backup listing workflow and returns the slice of backups.
func (a *ListBackupsAction) Run() ([]BackupInfo, error) {
	fsys := a.FS
	if fsys == nil {
		fsys = &OSFileSystem{}
	}

	bakDir := a.BakDir
	if bakDir == "" {
		var err error
		bakDir, err = backup.BakDir()
		if err != nil {
			return nil, fmt.Errorf("bak dir: %w", err)
		}
	}

	backupsDir := filepath.Join(bakDir, "backups")
	entries, err := fsys.ReadDir(backupsDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read backups dir: %w", err)
	}

	sortedEntries := make([]os.DirEntry, len(entries))
	copy(sortedEntries, entries)
	if a.SortDescending {
		sort.Slice(sortedEntries, func(i, j int) bool {
			return sortedEntries[i].Name() > sortedEntries[j].Name()
		})
	} else {
		sort.Slice(sortedEntries, func(i, j int) bool {
			return sortedEntries[i].Name() < sortedEntries[j].Name()
		})
	}

	var result []BackupInfo
	for _, entry := range sortedEntries {
		if !entry.IsDir() {
			continue
		}
		info, ok := a.loadBackupInfo(fsys, backupsDir, entry.Name())
		if !ok {
			continue
		}
		result = append(result, info)
	}

	return result, nil
}

// loadBackupInfo attempts to load and parse a manifest for entryName. Returns false
// when the directory should be skipped (filtered out, missing, or corrupt).
func (a *ListBackupsAction) loadBackupInfo(fsys FileSystem, backupsDir, entryName string) (BackupInfo, bool) {
	if a.FilterInvalidIDs && !IsValidBackupID(entryName) {
		return BackupInfo{}, false
	}

	manifestPath := filepath.Join(backupsDir, entryName, "manifest.json")
	data, err := fsys.ReadFile(manifestPath)
	if err != nil {
		return BackupInfo{}, false
	}

	var m manifest.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return BackupInfo{}, false
	}

	backupID := m.ID
	if backupID == "" {
		backupID = entryName
	}

	if a.FilterInvalidIDs && !IsValidBackupID(backupID) {
		return BackupInfo{}, false
	}

	return BackupInfo{
		ID:     backupID,
		Date:   formatBackupDate(backupID),
		Size:   FormatSizeBytes(m.TotalSize),
		Status: "ok",
		Cloud:  resolveCloudHint(m.Adapters),
	}, true
}

// resolveCloudHint returns the first adapter name sorted alphabetically, or "none".
func resolveCloudHint(adapters map[string]manifest.AdapterManifest) string {
	if len(adapters) == 0 {
		return "none"
	}
	adapterNames := make([]string, 0, len(adapters))
	for name := range adapters {
		adapterNames = append(adapterNames, name)
	}
	sort.Strings(adapterNames)
	return adapterNames[0]
}
