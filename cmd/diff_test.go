package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/manifest"
	"github.com/spf13/cobra"
)

// stageDiffBackups creates two backup directories and returns their IDs.
func stageDiffBackups(t *testing.T, tmpDir string) (id1, id2 string) {
	t.Helper()

	backupsDir := filepath.Join(tmpDir, ".bak", "backups")
	os.MkdirAll(backupsDir, 0755)

	// Shared hash for common.md (Unchanged between both).
	commonHash := "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	// Different hash for modified.md.
	oldHash := "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	newHash := "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	makeBackup := func(id string, files map[string]string) {
		dir := filepath.Join(backupsDir, id)
		os.MkdirAll(dir, 0755)

		items := make([]manifest.Item, 0, len(files))
		for fname, hash := range files {
			fpath := filepath.Join(dir, "opencode", fname)
			os.MkdirAll(filepath.Dir(fpath), 0755)
			os.WriteFile(fpath, []byte(fmt.Sprintf("content-%s", fname)), 0644)

			items = append(items, manifest.Item{
				Category:   "config",
				SourcePath: fmt.Sprintf("~/.config/opencode/%s", fname),
				BackupPath: fmt.Sprintf("opencode/%s", fname),
				Hash:       hash,
				Size:       int64(len(fmt.Sprintf("content-%s", fname))),
			})
		}

		m := &manifest.Manifest{
			Version:    "0.3.0",
			ID:         id,
			BakVersion: "0.3.0",
			Preset:     "quick",
			Adapters: map[string]manifest.AdapterManifest{
				"opencode": {Items: items},
			},
			FileCount: len(files),
		}
		m.Save(dir)
	}

	id1 = "20250101-120000"
	id2 = "20250102-130000"

	// id1 has: common.md, removed.md, modified.md (old hash)
	makeBackup(id1, map[string]string{
		"common.md":   commonHash,
		"removed.md":  "sha256:rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr",
		"modified.md": oldHash,
	})

	// id2 has: common.md, modified.md (new hash), added.md
	makeBackup(id2, map[string]string{
		"common.md":   commonHash,
		"modified.md": newHash,
		"added.md":    "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab",
	})

	return id1, id2
}

func TestDiffCmd_Registered(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "diff" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("diff subcommand not registered on root")
	}
}

func TestDiffCmd_Args(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "diff" {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("diff command not found")
	}

	// Diff requires exactly 2 args.
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("expected error with 0 args")
	}
	if err := cmd.Args(cmd, []string{"a"}); err == nil {
		t.Error("expected error with 1 arg")
	}
	if err := cmd.Args(cmd, []string{"a", "b"}); err != nil {
		t.Errorf("expected no error with 2 args, got %v", err)
	}
	if err := cmd.Args(cmd, []string{"a", "b", "c"}); err == nil {
		t.Error("expected error with 3 args")
	}
}

func TestRunDiff_AllCategories(t *testing.T) {
	tmpDir := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("USERPROFILE", tmpDir)
	default:
		t.Setenv("HOME", tmpDir)
	}

	id1, id2 := stageDiffBackups(t, tmpDir)

	deps, stdout, _ := setupTestDeps(t)

	cmd := &cobra.Command{}
	err := runDiffWithDeps(cmd, []string{id1, id2}, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	// Verify headings present.
	for _, heading := range []string{"Added", "Removed", "Modified", "Unchanged"} {
		if !strings.Contains(output, heading) {
			t.Errorf("output should contain %q heading\ngot: %s", heading, output)
		}
	}
	// Verify specific files.
	if !strings.Contains(output, "removed.md") {
		t.Error("output should mention removed.md")
	}
	if !strings.Contains(output, "added.md") {
		t.Error("output should mention added.md")
	}
	if !strings.Contains(output, "modified.md") {
		t.Error("output should mention modified.md")
	}
	if !strings.Contains(output, "common.md") {
		t.Error("output should mention common.md")
	}
}

func TestRunDiff_IdenticalBackups(t *testing.T) {
	tmpDir := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("USERPROFILE", tmpDir)
	default:
		t.Setenv("HOME", tmpDir)
	}

	backupsDir := filepath.Join(tmpDir, ".bak", "backups")
	os.MkdirAll(backupsDir, 0755)

	makeSimple := func(id string) {
		dir := filepath.Join(backupsDir, id)
		os.MkdirAll(dir, 0755)
		m := &manifest.Manifest{
			Version:    "0.3.0",
			ID:         id,
			BakVersion: "0.3.0",
			Preset:     "quick",
			Adapters: map[string]manifest.AdapterManifest{
				"opencode": {Items: []manifest.Item{
					{SourcePath: "a.md", BackupPath: "opencode/a.md", Hash: "sha256:abc", Size: 10},
				}},
			},
			FileCount: 1,
		}
		m.Save(dir)
	}
	makeSimple("20250101-120000")
	makeSimple("20250102-130000")

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"diff", "20250101-120000", "20250102-130000"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := bufOut.String()
	if !strings.Contains(output, "Unchanged") {
		t.Errorf("identical backups should show Unchanged heading\ngot: %s", output)
	}
}

func TestRunDiff_MissingBackup(t *testing.T) {
	tmpDir := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("USERPROFILE", tmpDir)
	default:
		t.Setenv("HOME", tmpDir)
	}

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"diff", "valid-id", "nonexistent"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing backup")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestRunDiff_TraversalBlocked(t *testing.T) {
	tmpDir := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("USERPROFILE", tmpDir)
	default:
		t.Setenv("HOME", tmpDir)
	}

	// Create a valid backup so the first ID resolves successfully.
	backupsDir := filepath.Join(tmpDir, ".bak", "backups")
	os.MkdirAll(filepath.Join(backupsDir, "20250101-120000"), 0755)
	m := &manifest.Manifest{
		Version:    "0.3.0",
		ID:         "20250101-120000",
		BakVersion: "0.3.0",
		Preset:     "quick",
		Adapters:   map[string]manifest.AdapterManifest{},
	}
	m.Save(filepath.Join(backupsDir, "20250101-120000"))

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"diff", "20250101-120000", "../etc"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for traversal")
	}
	if !strings.Contains(err.Error(), "outside") {
		t.Errorf("error should mention 'outside', got: %v", err)
	}
}

func TestRunDiff_EmptyManifests(t *testing.T) {
	tmpDir := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("USERPROFILE", tmpDir)
	default:
		t.Setenv("HOME", tmpDir)
	}

	backupsDir := filepath.Join(tmpDir, ".bak", "backups")
	os.MkdirAll(backupsDir, 0755)

	makeEmpty := func(id string) {
		dir := filepath.Join(backupsDir, id)
		os.MkdirAll(dir, 0755)
		m := &manifest.Manifest{
			Version:    "0.3.0",
			ID:         id,
			BakVersion: "0.3.0",
			Preset:     "quick",
			Adapters:   map[string]manifest.AdapterManifest{},
		}
		m.Save(dir)
	}
	makeEmpty("20250101-120000")
	makeEmpty("20250102-130000")

	deps, stdout, _ := setupTestDeps(t)

	cmd := &cobra.Command{}
	err := runDiffWithDeps(cmd, []string{"20250101-120000", "20250102-130000"}, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "identical") || !strings.Contains(output, "no differences") {
		// Accept either message indicating no diffs.
		if !strings.Contains(strings.ToLower(output), "no") {
			t.Errorf("empty manifests should indicate no differences\ngot: %s", output)
		}
	}
}

func TestDiffCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"diff", "--help"})
	rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "diff") {
		t.Fatal("help output should mention 'diff'")
	}
}

func TestDiffCmd_UseAndDescription(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "diff" {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("diff command not found")
	}
	if cmd.Use == "" {
		t.Fatal("diff Use should not be empty")
	}
	if cmd.Short == "" {
		t.Fatal("diff Short should not be empty")
	}
}
