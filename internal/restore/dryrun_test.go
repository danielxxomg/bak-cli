package restore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/manifest"
)

func TestComputeDryRun(t *testing.T) {
	t.Run("new file — backup exists, target does not", func(t *testing.T) {
		homeDir := t.TempDir()
		backupDir := t.TempDir()

		// Write a file into the backup directory.
		backupFilePath := filepath.Join(backupDir, "opencode", "skills", "go", "SKILL.md")
		mustWrite(t, backupFilePath, "# Go Skill\n\nGo development skill.")

		m := &manifest.Manifest{
			Adapters: map[string]manifest.AdapterManifest{
				"opencode": {
					ConfigDir: "~/.config/opencode",
					Items: []manifest.Item{
						{
							SourcePath: "~/.config/opencode/skills/go/SKILL.md",
							BackupPath: "opencode/skills/go/SKILL.md",
							Hash:       mustHash(t, backupFilePath),
						},
					},
				},
			},
		}

		diffs, err := ComputeDryRun(m, backupDir, homeDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(diffs) != 1 {
			t.Fatalf("expected 1 diff, got %d", len(diffs))
		}

		d := diffs[0]
		if d.Status != DiffNew {
			t.Fatalf("expected status 'new', got %q", d.Status)
		}
		if d.SourcePath != "~/.config/opencode/skills/go/SKILL.md" {
			t.Fatalf("expected SourcePath preserved, got %q", d.SourcePath)
		}
	})

	t.Run("modified file — backup and target differ", func(t *testing.T) {
		homeDir := t.TempDir()
		backupDir := t.TempDir()

		// Write backup version.
		backupFilePath := filepath.Join(backupDir, "opencode", "opencode.json")
		mustWrite(t, backupFilePath, `{"theme":"dark"}`)

		// Write a different target version.
		targetPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
		mustWrite(t, targetPath, `{"theme":"light"}`)

		m := &manifest.Manifest{
			Adapters: map[string]manifest.AdapterManifest{
				"opencode": {
					ConfigDir: "~/.config/opencode",
					Items: []manifest.Item{
						{
							SourcePath: "~/.config/opencode/opencode.json",
							BackupPath: "opencode/opencode.json",
							Hash:       mustHash(t, backupFilePath),
						},
					},
				},
			},
		}

		diffs, err := ComputeDryRun(m, backupDir, homeDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(diffs) != 1 {
			t.Fatalf("expected 1 diff, got %d", len(diffs))
		}

		d := diffs[0]
		if d.Status != DiffModified {
			t.Fatalf("expected status 'modified', got %q", d.Status)
		}
		if d.Diff == "" {
			t.Fatal("expected non-empty diff text for modified file")
		}
	})

	t.Run("unchanged file — identical content", func(t *testing.T) {
		homeDir := t.TempDir()
		backupDir := t.TempDir()

		content := `{"theme":"dark"}`

		backupFilePath := filepath.Join(backupDir, "opencode", "opencode.json")
		mustWrite(t, backupFilePath, content)

		targetPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
		mustWrite(t, targetPath, content)

		m := &manifest.Manifest{
			Adapters: map[string]manifest.AdapterManifest{
				"opencode": {
					ConfigDir: "~/.config/opencode",
					Items: []manifest.Item{
						{
							SourcePath: "~/.config/opencode/opencode.json",
							BackupPath: "opencode/opencode.json",
							Hash:       mustHash(t, backupFilePath),
						},
					},
				},
			},
		}

		diffs, err := ComputeDryRun(m, backupDir, homeDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(diffs) != 1 {
			t.Fatalf("expected 1 diff, got %d", len(diffs))
		}

		d := diffs[0]
		if d.Status != DiffUnchanged {
			t.Fatalf("expected status 'unchanged', got %q", d.Status)
		}
	})

	t.Run("missing backup file — manifest references file not on disk", func(t *testing.T) {
		homeDir := t.TempDir()
		backupDir := t.TempDir()

		m := &manifest.Manifest{
			Adapters: map[string]manifest.AdapterManifest{
				"opencode": {
					ConfigDir: "~/.config/opencode",
					Items: []manifest.Item{
						{
							SourcePath: "~/.config/opencode/skills/missing/SKILL.md",
							BackupPath: "opencode/skills/missing/SKILL.md",
							Hash:       "sha256:0000000000000000000000000000000000000000000000000000000000000000",
						},
					},
				},
			},
		}

		diffs, err := ComputeDryRun(m, backupDir, homeDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(diffs) != 1 {
			t.Fatalf("expected 1 diff, got %d", len(diffs))
		}

		d := diffs[0]
		if d.Status != DiffMissing {
			t.Fatalf("expected status 'missing', got %q", d.Status)
		}
	})

	t.Run("mixed batch — new, modified, unchanged, missing", func(t *testing.T) {
		homeDir := t.TempDir()
		backupDir := t.TempDir()

		// New file: only in backup.
		newPath := filepath.Join(backupDir, "opencode", "skills", "new", "SKILL.md")
		mustWrite(t, newPath, "# New Skill")

		// Modified: exists in both, different content.
		modBackup := filepath.Join(backupDir, "opencode", "commands", "test.md")
		mustWrite(t, modBackup, "backup version")
		modTarget := filepath.Join(homeDir, ".config", "opencode", "commands", "test.md")
		mustWrite(t, modTarget, "target version")

		// Unchanged: identical in both.
		unchangedContent := "same"
		unchBackup := filepath.Join(backupDir, "opencode", "AGENTS.md")
		mustWrite(t, unchBackup, unchangedContent)
		unchTarget := filepath.Join(homeDir, ".config", "opencode", "AGENTS.md")
		mustWrite(t, unchTarget, unchangedContent)

		m := &manifest.Manifest{
			Adapters: map[string]manifest.AdapterManifest{
				"opencode": {
					ConfigDir: "~/.config/opencode",
					Items: []manifest.Item{
						{SourcePath: "~/.config/opencode/skills/new/SKILL.md", BackupPath: "opencode/skills/new/SKILL.md", Hash: mustHash(t, newPath)},
						{SourcePath: "~/.config/opencode/commands/test.md", BackupPath: "opencode/commands/test.md", Hash: mustHash(t, modBackup)},
						{SourcePath: "~/.config/opencode/AGENTS.md", BackupPath: "opencode/AGENTS.md", Hash: mustHash(t, unchBackup)},
						{SourcePath: "~/.config/opencode/skills/missing/SKILL.md", BackupPath: "opencode/skills/missing/SKILL.md", Hash: "sha256:deadbeef"},
					},
				},
			},
		}

		diffs, err := ComputeDryRun(m, backupDir, homeDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(diffs) != 4 {
			t.Fatalf("expected 4 diffs, got %d", len(diffs))
		}

		statuses := make(map[DiffStatus]int)
		for _, d := range diffs {
			statuses[d.Status]++
		}

		if statuses[DiffNew] != 1 {
			t.Fatalf("expected 1 'new', got %d", statuses[DiffNew])
		}
		if statuses[DiffModified] != 1 {
			t.Fatalf("expected 1 'modified', got %d", statuses[DiffModified])
		}
		if statuses[DiffUnchanged] != 1 {
			t.Fatalf("expected 1 'unchanged', got %d", statuses[DiffUnchanged])
		}
		if statuses[DiffMissing] != 1 {
			t.Fatalf("expected 1 'missing', got %d", statuses[DiffMissing])
		}
	})
}

// mustWrite creates parent directories and writes content to path.
func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// mustHash returns the SHA-256 hex digest of a file.
func mustHash(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	h := sha256Hex(data)
	return "sha256:" + h
}
