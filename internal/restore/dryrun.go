package restore

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/danielxxomg/bak-cli/internal/manifest"
)

// DiffStatus classifies the relationship between a backup file and its
// corresponding target file on disk.
type DiffStatus string

const (
	DiffNew       DiffStatus = "new"       // exists in backup, not on target
	DiffModified  DiffStatus = "modified"  // both exist, content differs
	DiffUnchanged DiffStatus = "unchanged" // both exist, identical content
	DiffMissing   DiffStatus = "missing"   // referenced in manifest, not on backup disk
)

// FileDiff describes the difference between one backed-up file and the
// current target file on disk.
type FileDiff struct {
	SourcePath string     // canonical source path from manifest
	TargetPath string     // resolved absolute path on target OS
	BackupPath string     // path within backup directory
	Status     DiffStatus // classification
	Diff       string     // unified diff for modified files; empty otherwise
}

// CountByStatus returns the number of diffs with the given status.
func CountByStatus(diffs []FileDiff, status DiffStatus) int {
	count := 0
	for _, d := range diffs {
		if d.Status == status {
			count++
		}
	}
	return count
}

// ComputeDryRun reads the manifest, resolves every item's target path,
// and compares backup content with the current target files. Returns a
// sorted list of diffs without modifying the filesystem.
func ComputeDryRun(m *manifest.Manifest, backupDir, homeDir string) ([]FileDiff, error) {
	var diffs []FileDiff

	for _, am := range m.Adapters {
		for _, item := range am.Items {
			targetPath, err := ResolvePath(item.SourcePath, homeDir)
			if err != nil {
				return nil, fmt.Errorf("resolve %q: %w", item.SourcePath, err)
			}

			backupFilePath := filepath.Join(backupDir, item.BackupPath)

			d := FileDiff{
				SourcePath: item.SourcePath,
				TargetPath: targetPath,
				BackupPath: item.BackupPath,
			}

			// Check if backup file exists on disk.
			backupData, err := os.ReadFile(backupFilePath)
			if err != nil {
				d.Status = DiffMissing
				diffs = append(diffs, d)
				continue
			}

			// Check if target file exists.
			targetData, err := os.ReadFile(targetPath)
			if err != nil {
				// Target doesn't exist — this is a new file.
				d.Status = DiffNew
				diffs = append(diffs, d)
				continue
			}

			// Both exist — compare content.
			if string(backupData) == string(targetData) {
				d.Status = DiffUnchanged
			} else {
				d.Status = DiffModified
				d.Diff = unifiedDiff(targetPath, string(targetData), string(backupData))
			}

			diffs = append(diffs, d)
		}
	}

	return diffs, nil
}

// unifiedDiff produces a minimal unified diff between the current
// content on disk and the backup content.
func unifiedDiff(path, current, incoming string) string {
	curLines := splitLines(current)
	incLines := splitLines(incoming)

	if len(curLines) == 0 && len(incLines) == 0 {
		return ""
	}

	var out strings.Builder
	out.WriteString(fmt.Sprintf("--- a/%s\n", path))
	out.WriteString(fmt.Sprintf("+++ b/%s\n", path))

	// Simple line-by-line diff: mark lines unique to current as removed,
	// lines unique to incoming as added.
	maxN := len(curLines)
	if len(incLines) > maxN {
		maxN = len(incLines)
	}

	for i := 0; i < maxN; i++ {
		cur := ""
		inc := ""
		if i < len(curLines) {
			cur = curLines[i]
		}
		if i < len(incLines) {
			inc = incLines[i]
		}

		if cur == inc {
			out.WriteString(fmt.Sprintf("  %s\n", cur))
		} else {
			if cur != "" || inc != "" {
				if cur != "" {
					out.WriteString(fmt.Sprintf("- %s\n", cur))
				}
				if inc != "" {
					out.WriteString(fmt.Sprintf("+ %s\n", inc))
				}
			}
		}
	}

	return out.String()
}

// splitLines splits content into lines preserving empty trailing line.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// sha256Hex returns the hex-encoded SHA-256 digest of data.
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:])
}
