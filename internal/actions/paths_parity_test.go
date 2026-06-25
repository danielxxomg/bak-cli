package actions

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/danielxxomg/bak-cli/internal/manifest"
)

// TestCLIAndTUIPathsProduceIdenticalManifest asserts the spec scenario
// "CLI and TUI paths use same implementation": BackupAction.Run (CLI path)
// and backup.Engine.Run (TUI path) both delegate to the canonical backup.Run,
// so over identical fixtures their manifests MUST carry identical Items —
// same BackupPath, Hash, Size, Category, and ordering.
//
// This is the [VERIFY] test for task 1.5. It lives in package actions (not
// backup) because the backup package cannot import actions (import direction),
// and a both-paths test needs to construct a BackupAction.
func TestCLIAndTUIPathsProduceIdenticalManifest(t *testing.T) {
	const preset = "full"

	runCLI := func(t *testing.T) []manifest.Item {
		t.Helper()
		home := t.TempDir()
		createOpenCodeFixture(t, home)

		action := &BackupAction{
			FS:         newHomeFS(home),
			Registry:   setupBackupRegistry(),
			Stdout:     io.Discard,
			Stderr:     io.Discard,
			Preset:     preset,
			BakVersion: "test",
		}
		if err := action.Run(); err != nil {
			t.Fatalf("BackupAction.Run: %v", err)
		}
		return loadManifestItems(t, home)
	}

	runTUI := func(t *testing.T) []manifest.Item {
		t.Helper()
		home := t.TempDir()
		createOpenCodeFixture(t, home)

		engine := &backup.Engine{
			HomeDir:    home,
			BakDir:     filepath.Join(home, ".bak"),
			Registry:   setupBackupRegistry(),
			Preset:     preset,
			BakVersion: "test",
			FS:         newHomeFS(home),
		}
		result, err := engine.Run()
		if err != nil {
			t.Fatalf("Engine.Run: %v", err)
		}
		m, err := manifest.Load(result.BackupDir)
		if err != nil {
			t.Fatalf("load manifest: %v", err)
		}
		var items []manifest.Item
		for _, am := range m.Adapters {
			items = append(items, am.Items...)
		}
		if len(items) == 0 {
			t.Fatal("TUI path produced no manifest items — fixture not exercised")
		}
		return items
	}

	cliItems := runCLI(t)
	tuiItems := runTUI(t)

	if len(cliItems) != len(tuiItems) {
		t.Fatalf("item count differs: CLI=%d, TUI=%d", len(cliItems), len(tuiItems))
	}
	for i, want := range cliItems {
		got := tuiItems[i]
		switch {
		case got.BackupPath != want.BackupPath:
			t.Errorf("item[%d] BackupPath: CLI=%q TUI=%q", i, want.BackupPath, got.BackupPath)
		case got.Hash != want.Hash:
			t.Errorf("item[%d] Hash: CLI=%q TUI=%q", i, want.Hash, got.Hash)
		case got.Size != want.Size:
			t.Errorf("item[%d] Size: CLI=%d TUI=%d", i, want.Size, got.Size)
		case got.Category != want.Category:
			t.Errorf("item[%d] Category: CLI=%q TUI=%q", i, want.Category, got.Category)
		}
	}
}
