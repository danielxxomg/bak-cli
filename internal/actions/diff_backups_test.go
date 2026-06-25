package actions

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	configtest "github.com/danielxxomg/bak-cli/internal/config/testutil"
	"github.com/danielxxomg/bak-cli/internal/diff"
	"github.com/danielxxomg/bak-cli/internal/manifest"
)

// setupDiffFixture creates a backup directory with the given files and returns
// its backup ID. Files are written to disk and a manifest.json with correct
// SHA-256 hashes is created.
func setupDiffFixture(t *testing.T, homeDir string, backupID string, files map[string]string) {
	t.Helper()

	configtest.SetConfigHome(t, homeDir)

	bakDir := filepath.Join(homeDir, ".bak")
	backupDir := filepath.Join(bakDir, "backups", backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	var items []manifest.Item
	for relPath, content := range files {
		fullPath := filepath.Join(backupDir, relPath)
		if dir := filepath.Dir(fullPath); dir != backupDir {
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		hash := sha256.Sum256([]byte(content))
		items = append(items, manifest.Item{
			Category:   "config",
			SourcePath: relPath,
			BackupPath: relPath,
			Hash:       "sha256:" + hex.EncodeToString(hash[:]),
			Size:       int64(len(content)),
		})
	}

	totalSize := int64(0)
	for _, it := range items {
		totalSize += it.Size
	}

	m := &manifest.Manifest{
		Version:    manifest.ManifestVersion,
		ID:         backupID,
		Preset:     "quick",
		FileCount:  len(items),
		TotalSize:  totalSize,
		Categories: []string{"config"},
		Adapters: map[string]manifest.AdapterManifest{
			"opencode": {
				ConfigDir: "opencode",
				Items:     items,
			},
		},
	}

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "manifest.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDiffBackupsAction_Added(t *testing.T) {
	homeDir := t.TempDir()

	// Backup 1: has config.yaml only.
	setupDiffFixture(t, homeDir, "backup-v1", map[string]string{
		"config.yaml": "version: 1",
	})
	// Backup 2: has config.yaml AND new skill.md.
	setupDiffFixture(t, homeDir, "backup-v2", map[string]string{
		"config.yaml":   "version: 1",
		"skills/new.md": "# New Skill",
	})

	var out strings.Builder
	action := &DiffBackupsAction{
		Stdout: &out,
	}

	err := action.Run("backup-v1", "backup-v2")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Added:") {
		t.Errorf("output should contain 'Added:', got: %q", output)
	}
	if !strings.Contains(output, "skills/new.md") {
		t.Errorf("output should mention added file: %q", output)
	}
}

func TestDiffBackupsAction_Removed(t *testing.T) {
	homeDir := t.TempDir()

	// Backup 1: has old-file.txt and config.yaml.
	setupDiffFixture(t, homeDir, "backup-v1", map[string]string{
		"config.yaml":  "version: 1",
		"old-file.txt": "old content",
	})
	// Backup 2: only has config.yaml (old-file.txt was deleted).
	setupDiffFixture(t, homeDir, "backup-v2", map[string]string{
		"config.yaml": "version: 1",
	})

	var out strings.Builder
	action := &DiffBackupsAction{
		Stdout: &out,
	}

	err := action.Run("backup-v1", "backup-v2")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Removed:") {
		t.Errorf("output should contain 'Removed:', got: %q", output)
	}
	if !strings.Contains(output, "old-file.txt") {
		t.Errorf("output should mention removed file: %q", output)
	}
}

func TestDiffBackupsAction_Modified(t *testing.T) {
	homeDir := t.TempDir()

	// Both backups have config.yaml but with different content (different hashes).
	setupDiffFixture(t, homeDir, "backup-v1", map[string]string{
		"config.yaml": "version: 1\nsetting: old-value",
	})
	setupDiffFixture(t, homeDir, "backup-v2", map[string]string{
		"config.yaml": "version: 1\nsetting: new-value",
	})

	var out strings.Builder
	action := &DiffBackupsAction{
		Stdout: &out,
	}

	err := action.Run("backup-v1", "backup-v2")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Modified:") {
		t.Errorf("output should contain 'Modified:', got: %q", output)
	}
	if !strings.Contains(output, "config.yaml") {
		t.Errorf("output should mention modified file: %q", output)
	}
}

func TestDiffBackupsAction_Unchanged(t *testing.T) {
	homeDir := t.TempDir()

	// Both backups have config.yaml with identical content.
	setupDiffFixture(t, homeDir, "backup-v1", map[string]string{
		"config.yaml": "version: 1",
	})
	setupDiffFixture(t, homeDir, "backup-v2", map[string]string{
		"config.yaml": "version: 1",
	})

	var out strings.Builder
	action := &DiffBackupsAction{
		Stdout: &out,
	}

	err := action.Run("backup-v1", "backup-v2")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Unchanged:") {
		t.Errorf("output should contain 'Unchanged:', got: %q", output)
	}
	if !strings.Contains(output, "config.yaml") {
		t.Errorf("output should mention unchanged file: %q", output)
	}
}

func TestDiffBackupsAction_Identical(t *testing.T) {
	homeDir := t.TempDir()

	// Two backups with exactly the same files AND content.
	setupDiffFixture(t, homeDir, "backup-v1", map[string]string{
		"config.yaml": "version: 1",
	})
	setupDiffFixture(t, homeDir, "backup-v2", map[string]string{
		"config.yaml": "version: 1",
	})

	var out strings.Builder
	action := &DiffBackupsAction{
		Stdout: &out,
	}

	err := action.Run("backup-v1", "backup-v2")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	output := out.String()
	// When all files match (Unchanged), diff.Compare still produces entries.
	// The generic "are identical" message only triggers when there are zero
	// entries at all (empty manifests).
	if !strings.Contains(output, "Unchanged:") {
		t.Errorf("output should contain 'Unchanged:', got: %q", output)
	}
	if !strings.Contains(output, "config.yaml") {
		t.Errorf("output should mention file: %q", output)
	}
}

func TestDiffBackupsAction_MissingBackup1(t *testing.T) {
	homeDir := t.TempDir()
	configtest.SetConfigHome(t, homeDir)

	setupDiffFixture(t, homeDir, "backup-v2", map[string]string{
		"config.yaml": "version: 1",
	})

	var out strings.Builder
	action := &DiffBackupsAction{
		Stdout: &out,
	}

	err := action.Run("nonexistent", "backup-v2")
	if err == nil {
		t.Fatal("expected error for nonexistent first backup")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestDiffBackupsAction_MissingBackup2(t *testing.T) {
	homeDir := t.TempDir()
	configtest.SetConfigHome(t, homeDir)

	setupDiffFixture(t, homeDir, "backup-v1", map[string]string{
		"config.yaml": "version: 1",
	})

	var out strings.Builder
	action := &DiffBackupsAction{
		Stdout: &out,
	}

	err := action.Run("backup-v1", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent second backup")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestDiffBackupsAction_Summary(t *testing.T) {
	homeDir := t.TempDir()

	// Create two backups with all categories represented.
	setupDiffFixture(t, homeDir, "backup-v1", map[string]string{
		"config.yaml":  "version: 1",
		"old-file.txt": "will be removed",
		"shared.json":  `{"key": "old"}`,
	})
	setupDiffFixture(t, homeDir, "backup-v2", map[string]string{
		"config.yaml": "version: 1",
		"new-file.go": "package main",
		"shared.json": `{"key": "new"}`,
	})

	var out strings.Builder
	action := &DiffBackupsAction{
		Stdout: &out,
	}

	err := action.Run("backup-v1", "backup-v2")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	output := out.String()
	// Summary line should include counts.
	if !strings.Contains(output, "Summary:") {
		t.Errorf("output should contain 'Summary:', got: %q", output)
	}
	// 1 added (new-file.go), 1 removed (old-file.txt),
	// 1 modified (shared.json), 1 unchanged (config.yaml)
	if !strings.Contains(output, "1 added") {
		t.Errorf("output should say '1 added': %q", output)
	}
	if !strings.Contains(output, "1 removed") {
		t.Errorf("output should say '1 removed': %q", output)
	}
	if !strings.Contains(output, "1 modified") {
		t.Errorf("output should say '1 modified': %q", output)
	}
	if !strings.Contains(output, "1 unchanged") {
		t.Errorf("output should say '1 unchanged': %q", output)
	}
}

// --- extracted helper tests (Phase 9) ---

// TestGroupDiffEntries verifies the pure grouping/counting helper.
func TestGroupDiffEntries(t *testing.T) {
	entries := []diff.DiffEntry{
		{SourcePath: "a.go", Category: diff.CategoryAdded, Adapter: "opencode"},
		{SourcePath: "b.go", Category: diff.CategoryAdded, Adapter: "cursor"},
		{SourcePath: "c.txt", Category: diff.CategoryRemoved, Adapter: "opencode"},
		{SourcePath: "d.json", Category: diff.CategoryModified, Adapter: "codex"},
		{SourcePath: "e.md", Category: diff.CategoryUnchanged, Adapter: "opencode"},
	}

	groups, counts := groupDiffEntries(entries)

	if len(groups[diff.CategoryAdded]) != 2 {
		t.Errorf("Added group len = %d, want 2", len(groups[diff.CategoryAdded]))
	}
	if counts[diff.CategoryAdded] != 2 {
		t.Errorf("Added count = %d, want 2", counts[diff.CategoryAdded])
	}
	if counts[diff.CategoryRemoved] != 1 {
		t.Errorf("Removed count = %d, want 1", counts[diff.CategoryRemoved])
	}
	if counts[diff.CategoryModified] != 1 {
		t.Errorf("Modified count = %d, want 1", counts[diff.CategoryModified])
	}
	if counts[diff.CategoryUnchanged] != 1 {
		t.Errorf("Unchanged count = %d, want 1", counts[diff.CategoryUnchanged])
	}
	// Empty input still yields all four category keys initialized.
	emptyGroups, emptyCounts := groupDiffEntries(nil)
	for _, cat := range []diff.Category{diff.CategoryAdded, diff.CategoryRemoved, diff.CategoryModified, diff.CategoryUnchanged} {
		if _, ok := emptyGroups[cat]; !ok {
			t.Errorf("empty: missing group key %s", cat)
		}
		if emptyCounts[cat] != 0 {
			t.Errorf("empty: count %s = %d, want 0", cat, emptyCounts[cat])
		}
	}
}

// TestPrintDiffGroups verifies grouped output is emitted in stable order
// with empty groups skipped and a trailing blank line per group.
func TestPrintDiffGroups(t *testing.T) {
	groups := map[diff.Category][]diff.DiffEntry{
		diff.CategoryAdded: {
			{SourcePath: "new.go", Adapter: "opencode"},
		},
		diff.CategoryModified: {
			{SourcePath: "cfg.yaml", Adapter: "cursor"},
		},
		// Removed and Unchanged intentionally absent/empty.
	}

	var out bytes.Buffer
	printDiffGroups(&out, groups)

	got := out.String()
	// Added must appear before Modified (stable order).
	addedIdx := strings.Index(got, "Added:")
	modIdx := strings.Index(got, "Modified:")
	if addedIdx < 0 || modIdx < 0 {
		t.Fatalf("missing group headings; got: %q", got)
	}
	if addedIdx > modIdx {
		t.Errorf("Added should precede Modified; added=%d mod=%d", addedIdx, modIdx)
	}
	if !strings.Contains(got, "new.go (opencode)") {
		t.Errorf("output should list added entry; got: %q", got)
	}
	if !strings.Contains(got, "cfg.yaml (cursor)") {
		t.Errorf("output should list modified entry; got: %q", got)
	}
	// No Removed/Unchanged heading should be emitted (empty groups skipped).
	if strings.Contains(got, "Removed:") {
		t.Errorf("empty Removed group should be skipped; got: %q", got)
	}
	if strings.Contains(got, "Unchanged:") {
		t.Errorf("empty Unchanged group should be skipped; got: %q", got)
	}
}

// TestPrintDiffSummary verifies the summary line counts and total.
func TestPrintDiffSummary(t *testing.T) {
	counts := map[diff.Category]int{
		diff.CategoryAdded:     2,
		diff.CategoryRemoved:   1,
		diff.CategoryModified:  3,
		diff.CategoryUnchanged: 4,
	}
	var out bytes.Buffer
	printDiffSummary(&out, counts, 10)

	got := out.String()
	for _, want := range []string{"Summary:", "2 added", "1 removed", "3 modified", "4 unchanged", "10 total"} {
		if !strings.Contains(got, want) {
			t.Errorf("summary missing %q; got: %q", want, got)
		}
	}
}
