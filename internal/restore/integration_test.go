package restore

import (
	"os"
	"path/filepath"
	"testing"

	pathsutil "github.com/danielxxomg/bak-cli/internal/paths"
)

func TestIntegration_RestoreRoundTrip(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// Simulates: backup on one machine → restore on another.
	// The backup directory has files with a manifest referencing them.
	// The restore engine copies them to the target home.

	t.Run("backup then restore files match", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		homeDir := t.TempDir()
		backupDir := t.TempDir()

		// Set up backup files with known content.
		backupFiles := map[string]string{
			"opencode/opencode.json":              `{"theme":"dark","version":"1.0"}`,
			"opencode/skills/go/SKILL.md":         "# Go Skill\n\nGo development patterns.",
			"opencode/skills/typescript/SKILL.md": "# TypeScript Skill",
			"opencode/commands/build.md":          "## Build Command\n\nRuns the build.",
			"opencode/AGENTS.md":                  "You are a helpful assistant.",
		}

		manifestItems := make([]manifestItem, 0, len(backupFiles))
		for backupPath, content := range backupFiles {
			fullPath := filepath.Join(backupDir, backupPath)
			mustWrite(t, fullPath, content)
			// Derive canonical source path: opencode/<rest> → ~/.config/<rest>
			sourcePath := "~/.config/" + pathsutil.Slash(backupPath)
			manifestItems = append(manifestItems, manifestItem{
				source: sourcePath,
				backup: backupPath,
			})
		}

		m := buildTestManifest(t, backupDir, manifestItems)

		// Run restore engine.
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

		if result.Restored != len(backupFiles) {
			t.Fatalf("expected %d restored, got %d", len(backupFiles), result.Restored)
		}

		// Verify every file was written correctly.
		for backupPath, expectedContent := range backupFiles {
			// Convert backup path to target: opencode/skills/go/SKILL.md → .config/opencode/skills/go/SKILL.md
			relPath := filepath.FromSlash(backupPath)
			targetPath := filepath.Join(homeDir, ".config", relPath)

			got, err := os.ReadFile(targetPath)
			if err != nil {
				t.Fatalf("target file %s not created: %v", targetPath, err)
			}
			if string(got) != expectedContent {
				t.Fatalf("file %s: got %q, want %q", targetPath, string(got), expectedContent)
			}
		}
	})

	t.Run("restore with path translation cross-platform", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		// Simulate: backup created on Linux (paths in canonical form)
		// being restored on Windows (paths with backslashes).
		homeDir := t.TempDir()
		backupDir := t.TempDir()

		// Write a backup file.
		mustWrite(t, filepath.Join(backupDir, "opencode", "opencode.json"),
			`{"theme":"dark"}`)

		m := buildTestManifest(t, backupDir, []manifestItem{
			{
				source: "~/.config/opencode/opencode.json",
				backup: "opencode/opencode.json",
			},
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

		// The file should exist in the target, with OS-appropriate path.
		targetPath := filepath.Join(homeDir, ".config", "opencode", "opencode.json")
		got, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("target file not created at %s: %v", targetPath, err)
		}
		if string(got) != `{"theme":"dark"}` {
			t.Fatalf("wrong content: %q", string(got))
		}
	})

	t.Run("restore with nested directories preserves structure", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		homeDir := t.TempDir()
		backupDir := t.TempDir()

		// Deeply nested paths.
		mustWrite(t, filepath.Join(backupDir, "opencode", "skills", "a", "b", "c", "deep.md"),
			"deep content")

		m := buildTestManifest(t, backupDir, []manifestItem{
			{
				source: "~/.config/opencode/skills/a/b/c/deep.md",
				backup: "opencode/skills/a/b/c/deep.md",
			},
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

		targetPath := filepath.Join(homeDir, ".config", "opencode", "skills", "a", "b", "c", "deep.md")
		got, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("target file not created: %v", err)
		}
		if string(got) != "deep content" {
			t.Fatalf("wrong content: %q", string(got))
		}
	})

	t.Run("restore with overwrite of existing target", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		homeDir := t.TempDir()
		backupDir := t.TempDir()

		// Write original content on target.
		targetDir := filepath.Join(homeDir, ".config", "opencode")
		mustWrite(t, filepath.Join(targetDir, "opencode.json"), `{"theme":"light"}`)

		// Write backup with different content.
		mustWrite(t, filepath.Join(backupDir, "opencode", "opencode.json"),
			`{"theme":"dark"}`)

		m := buildTestManifest(t, backupDir, []manifestItem{
			{
				source: "~/.config/opencode/opencode.json",
				backup: "opencode/opencode.json",
			},
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

		// Content should be the backup version (overwritten).
		got, err := os.ReadFile(filepath.Join(targetDir, "opencode.json"))
		if err != nil {
			t.Fatalf("target file not created: %v", err)
		}
		if string(got) != `{"theme":"dark"}` {
			t.Fatalf("expected overwrite with backup content, got %q", string(got))
		}
	})
}
