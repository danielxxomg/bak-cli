package restore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/manifest"
)

func TestRestoreEngine_Run(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	t.Run("successful restore — dry-run then apply", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		homeDir := t.TempDir()
		backupDir := t.TempDir()

		// Set up backup with known files.
		mustWrite(t, filepath.Join(backupDir, "opencode", "opencode.json"), `{"theme":"dark"}`)
		mustWrite(t, filepath.Join(backupDir, "opencode", "skills", "go", "SKILL.md"), "# Go Skill")

		m := buildTestManifest(t, backupDir, []manifestItem{
			{source: "~/.config/opencode/opencode.json", backup: "opencode/opencode.json"},
			{source: "~/.config/opencode/skills/go/SKILL.md", backup: "opencode/skills/go/SKILL.md"},
		})

		engine := &Engine{
			HomeDir:   homeDir,
			BackupDir: backupDir,
			DryRun:    false,
			Force:     true,
		}

		result, err := engine.Run(m)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Restored != 2 {
			t.Fatalf("expected 2 restored, got %d", result.Restored)
		}
		if result.Failed != 0 {
			t.Fatalf("expected 0 failed, got %d", result.Failed)
		}
		if result.Skipped != 0 {
			t.Fatalf("expected 0 skipped, got %d", result.Skipped)
		}

		// Verify files were actually written.
		got, err := os.ReadFile(filepath.Join(homeDir, ".config", "opencode", "opencode.json"))
		if err != nil {
			t.Fatalf("target file not created: %v", err)
		}
		if string(got) != `{"theme":"dark"}` {
			t.Fatalf("wrong content: %q", string(got))
		}
	})

	t.Run("dry-run only — does not modify files", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		homeDir := t.TempDir()
		backupDir := t.TempDir()

		mustWrite(t, filepath.Join(backupDir, "opencode", "opencode.json"), `{"theme":"dark"}`)

		m := buildTestManifest(t, backupDir, []manifestItem{
			{source: "~/.config/opencode/opencode.json", backup: "opencode/opencode.json"},
		})

		engine := &Engine{
			HomeDir:   homeDir,
			BackupDir: backupDir,
			DryRun:    true,
			Force:     true,
		}

		result, err := engine.Run(m)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should report diffs but not write.
		if len(result.Diffs) == 0 {
			t.Fatal("expected diffs in dry-run mode")
		}
		if result.Restored != 0 {
			t.Fatalf("expected 0 restored in dry-run, got %d", result.Restored)
		}

		// Target file should NOT exist.
		targetPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
		if _, err := os.Stat(targetPath); err == nil {
			t.Fatal("target file should not exist after dry-run")
		}
	})

	t.Run("missing backup file — graceful skip", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		homeDir := t.TempDir()
		backupDir := t.TempDir()

		// Only create one of the two files.
		mustWrite(t, filepath.Join(backupDir, "opencode", "opencode.json"), `{"theme":"dark"}`)

		m := buildTestManifest(t, backupDir, []manifestItem{
			{source: "~/.config/opencode/opencode.json", backup: "opencode/opencode.json"},
			{source: "~/.config/opencode/skills/missing/SKILL.md", backup: "opencode/skills/missing/SKILL.md", hash: "sha256:0000000000000000000000000000000000000000000000000000000000000000"},
		})

		engine := &Engine{
			HomeDir:   homeDir,
			BackupDir: backupDir,
			DryRun:    false,
			Force:     true,
		}

		result, err := engine.Run(m)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Restored != 1 {
			t.Fatalf("expected 1 restored, got %d", result.Restored)
		}
		if result.Skipped != 1 {
			t.Fatalf("expected 1 skipped, got %d", result.Skipped)
		}
	})

	t.Run("path outside home — rejected", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		backupDir := t.TempDir()

		mustWrite(t, filepath.Join(backupDir, "opencode", "passwd"), "root:x:0:0:root")

		m := buildTestManifest(t, backupDir, []manifestItem{
			{source: "~/../../../etc/passwd", backup: "opencode/passwd"},
		})

		engine := &Engine{
			HomeDir:   "/home/alice",
			BackupDir: backupDir,
			DryRun:    false,
			Force:     true,
		}

		_, err := engine.Run(m)
		if err == nil {
			t.Fatal("expected error for path outside home, got nil")
		}
	})

	t.Run("dry-run mandatory — force dryRun when not forcing", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		homeDir := t.TempDir()
		backupDir := t.TempDir()

		mustWrite(t, filepath.Join(backupDir, "opencode", "opencode.json"), `{"theme":"dark"}`)

		m := buildTestManifest(t, backupDir, []manifestItem{
			{source: "~/.config/opencode/opencode.json", backup: "opencode/opencode.json"},
		})

		// DryRun=false, Force=false: should still produce diffs (dry-run is mandatory).
		engine := &Engine{
			HomeDir:   homeDir,
			BackupDir: backupDir,
			DryRun:    false,
			Force:     false,
		}

		result, err := engine.Run(m)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Even without force, should have computed diffs.
		if len(result.Diffs) == 0 {
			t.Fatal("expected diffs computed")
		}

		// In non-interactive mode (Force=false without terminal), should still restore.
		if result.Restored != 1 {
			t.Fatalf("expected 1 restored, got %d", result.Restored)
		}
	})
}

func TestEngine_Run_ChecksumMismatch(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	homeDir := t.TempDir()
	backupDir := t.TempDir()

	mustWrite(t, filepath.Join(backupDir, "opencode", "opencode.json"), `{"theme":"dark"}`)

	// Build a manifest with a WRONG hash for the backup file.
	m := &manifest.Manifest{
		Version:    "0.1.0",
		ID:         "test-backup",
		OSSource:   "linux",
		BakVersion: "0.1.0",
		Preset:     "full",
		Categories: []string{"config"},
		Adapters: map[string]manifest.AdapterManifest{
			"opencode": {
				ConfigDir: "~/.config/opencode",
				Items: []manifest.Item{
					{
						SourcePath: "~/.config/opencode/opencode.json",
						BackupPath: "opencode/opencode.json",
						Hash:       "sha256:0000000000000000000000000000000000000000000000000000000000000000",
					},
				},
			},
		},
	}

	engine := &Engine{
		HomeDir:   homeDir,
		BackupDir: backupDir,
		DryRun:    false,
		Force:     false, // Force is off → checksum mismatch should cause error
	}

	_, err := engine.Run(m)
	if err == nil {
		t.Fatal("expected error for checksum mismatch, got nil")
	}
}

func TestEngine_Run_ChecksumMismatch_ForceOverride(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	homeDir := t.TempDir()
	backupDir := t.TempDir()

	mustWrite(t, filepath.Join(backupDir, "opencode", "opencode.json"), `{"theme":"dark"}`)

	// Build a manifest with a WRONG hash — but Force is true.
	m := &manifest.Manifest{
		Version:    "0.1.0",
		ID:         "test-backup",
		OSSource:   "linux",
		BakVersion: "0.1.0",
		Preset:     "full",
		Categories: []string{"config"},
		Adapters: map[string]manifest.AdapterManifest{
			"opencode": {
				ConfigDir: "~/.config/opencode",
				Items: []manifest.Item{
					{
						SourcePath: "~/.config/opencode/opencode.json",
						BackupPath: "opencode/opencode.json",
						Hash:       "sha256:0000000000000000000000000000000000000000000000000000000000000000",
					},
				},
			},
		},
	}

	engine := &Engine{
		HomeDir:   homeDir,
		BackupDir: backupDir,
		DryRun:    false,
		Force:     true, // Force overrides checksum validation
	}

	result, err := engine.Run(m)
	if err != nil {
		t.Fatalf("unexpected error with Force=true: %v", err)
	}
	if result.Restored != 1 {
		t.Fatalf("expected 1 restored with Force=true, got %d", result.Restored)
	}

	// Verify the file was actually written.
	targetPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("target file not created: %v", err)
	}
	if string(data) != `{"theme":"dark"}` {
		t.Fatalf("wrong content: %q", string(data))
	}
}

func TestEngine_Run_WriteFailure(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	homeDir := t.TempDir()
	backupDir := t.TempDir()

	mustWrite(t, filepath.Join(backupDir, "opencode", "opencode.json"), `{"theme":"dark"}`)

	// Create a file where a directory component should be, so MkdirAll fails.
	// The resolved target path will be: homeDir/.config/opencode/opencode.json
	// We block the ".config" path by making it a file.
	blockPath := filepath.Join(homeDir, ".config")
	if err := os.WriteFile(blockPath, []byte("blocked"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	m := buildTestManifest(t, backupDir, []manifestItem{
		{source: "~/.config/opencode/opencode.json", backup: "opencode/opencode.json"},
	})

	engine := &Engine{
		HomeDir:   homeDir,
		BackupDir: backupDir,
		DryRun:    false,
		Force:     true,
		Verbose:   true,
	}

	result, err := engine.Run(m)
	if err != nil {
		t.Fatalf("unexpected top-level error: %v", err)
	}

	// The write should have failed; Failed count should be 1.
	if result.Failed != 1 {
		t.Fatalf("expected 1 failed, got %d", result.Failed)
	}
	if result.Restored != 0 {
		t.Fatalf("expected 0 restored, got %d", result.Restored)
	}
}

// --- Helpers for restore engine tests ---

type manifestItem struct {
	source string
	backup string
	hash   string // empty = compute from backup file
}

// buildTestManifest creates a manifest with the given items under an
// "opencode" adapter, computing hashes from the actual backup files
// unless an explicit hash is provided.
func buildTestManifest(t *testing.T, backupDir string, items []manifestItem) *manifest.Manifest {
	t.Helper()

	manifestItems := make([]manifest.Item, 0, len(items))
	for _, mi := range items {
		h := mi.hash
		if h == "" {
			backupPath := filepath.Join(backupDir, mi.backup)
			h = mustHash(t, backupPath)
		}
		manifestItems = append(manifestItems, manifest.Item{
			SourcePath: mi.source,
			BackupPath: mi.backup,
			Hash:       h,
		})
	}

	return &manifest.Manifest{
		Version:    "0.1.0",
		ID:         "test-backup",
		OSSource:   "linux",
		BakVersion: "0.1.0",
		Preset:     "full",
		Categories: []string{"config", "skills"},
		Adapters: map[string]manifest.AdapterManifest{
			"opencode": {
				ConfigDir: "~/.config/opencode",
				Items:     manifestItems,
			},
		},
	}
}
