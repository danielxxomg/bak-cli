package restore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/manifest"
)

func TestComputeDryRun(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	t.Run("new file — backup exists, target does not", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

	t.Run("modified file — backup and target differ", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

	t.Run("unchanged file — identical content", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

	t.Run("missing backup file — manifest references file not on disk", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

	t.Run("mixed batch — new, modified, unchanged, missing", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

func TestCountByStatus(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name     string
		diffs    []FileDiff
		status   DiffStatus
		expected int
	}{
		{
			name:     "empty slice",
			diffs:    []FileDiff{},
			status:   DiffNew,
			expected: 0,
		},
		{
			name: "single match",
			diffs: []FileDiff{
				{Status: DiffNew},
			},
			status:   DiffNew,
			expected: 1,
		},
		{
			name: "single mismatch",
			diffs: []FileDiff{
				{Status: DiffNew},
			},
			status:   DiffUnchanged,
			expected: 0,
		},
		{
			name: "mixed statuses — count new",
			diffs: []FileDiff{
				{Status: DiffNew},
				{Status: DiffModified},
				{Status: DiffNew},
				{Status: DiffUnchanged},
				{Status: DiffMissing},
			},
			status:   DiffNew,
			expected: 2,
		},
		{
			name: "mixed statuses — count missing",
			diffs: []FileDiff{
				{Status: DiffNew},
				{Status: DiffModified},
				{Status: DiffMissing},
				{Status: DiffUnchanged},
				{Status: DiffMissing},
			},
			status:   DiffMissing,
			expected: 2,
		},
		{
			name: "all same status",
			diffs: []FileDiff{
				{Status: DiffUnchanged},
				{Status: DiffUnchanged},
				{Status: DiffUnchanged},
			},
			status:   DiffUnchanged,
			expected: 3,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			got := CountByStatus(tt.diffs, tt.status)
			if got != tt.expected {
				t.Fatalf("CountByStatus = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestWriteRestoreLog_ErrorPaths(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	t.Run("valid write — creates log entry", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		backupDir := t.TempDir()
		engine := &Engine{
			GitDir:    "/some/git/dir",
			BackupDir: backupDir,
		}
		result := &RestoreResult{
			Restored: 5,
			Skipped:  2,
			Failed:   1,
		}

		engine.writeRestoreLog("test-id", result)

		logPath := filepath.Join(backupDir, "restore-log.jsonl")
		data, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("expected log file to exist: %v", err)
		}
		want := `{"id":"test-id","restored":5,"skipped":2,"failed":1}` + "\n"
		if string(data) != want {
			t.Fatalf("log content = %q, want %q", string(data), want)
		}
	})

	t.Run("empty GitDir — no-op, no file created", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		backupDir := t.TempDir()
		engine := &Engine{
			GitDir:    "",
			BackupDir: backupDir,
		}
		result := &RestoreResult{
			Restored: 1,
			Skipped:  0,
			Failed:   0,
		}

		engine.writeRestoreLog("no-git-dir", result)

		logPath := filepath.Join(backupDir, "restore-log.jsonl")
		if _, err := os.Stat(logPath); err == nil {
			t.Fatal("log file should not exist when GitDir is empty")
		}
	})

	t.Run("mkdir blocked — BackupDir is a file, not a directory", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		dir := t.TempDir()
		// Create backupDir path as a FILE, so MkdirAll on its parent fails.
		blockFile := filepath.Join(dir, "backup-file")
		if err := os.WriteFile(blockFile, []byte("x"), 0644); err != nil {
			t.Fatalf("setup: %v", err)
		}

		engine := &Engine{
			GitDir:    "/some/git/dir",
			BackupDir: blockFile, // this is a file, not a directory
			Verbose:   true,
		}
		result := &RestoreResult{
			Restored: 1,
			Skipped:  0,
			Failed:   0,
		}

		// Should not panic; should silently skip.
		engine.writeRestoreLog("mkdir-fail", result)

		logPath := filepath.Join(blockFile, "restore-log.jsonl")
		if _, err := os.Stat(logPath); err == nil {
			t.Fatal("log file should not exist when mkdir fails")
		}
	})

	t.Run("write failure — restore-log.jsonl is a directory", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		backupDir := t.TempDir()
		// Create restore-log.jsonl as a DIRECTORY, so WriteFile fails.
		logDir := filepath.Join(backupDir, "restore-log.jsonl")
		if err := os.MkdirAll(logDir, 0755); err != nil {
			t.Fatalf("setup: %v", err)
		}

		engine := &Engine{
			GitDir:    "/some/git/dir",
			BackupDir: backupDir,
			Verbose:   true,
		}
		result := &RestoreResult{
			Restored: 1,
			Skipped:  0,
			Failed:   0,
		}

		// Should not panic; should silently skip.
		engine.writeRestoreLog("write-fail", result)
	})
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
