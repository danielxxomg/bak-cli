package restore

import (
	"path/filepath"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestRestoreWithGitSafety(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	t.Run("git-backed restore creates pre and post commits", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		homeDir := t.TempDir()
		backupDir := t.TempDir()

		// Initialize git in backup directory (simulating ~/.bak/).
		initTestRepo(t, backupDir)
		initialCommits := countCommits(t, backupDir)

		// Set up backup with known files.
		mustWrite(t, filepath.Join(backupDir, "opencode", "opencode.json"), `{"theme":"dark"}`)

		m := buildTestManifest(t, backupDir, []manifestItem{
			{source: "~/.config/opencode/opencode.json", backup: "opencode/opencode.json"},
		})

		engine := &Engine{
			HomeDir:   homeDir,
			BackupDir: backupDir,
			DryRun:    false,
			Force:     true,
			GitDir:    backupDir,
		}

		result, err := engine.Run(m)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		if result.Restored != 1 {
			t.Fatalf("expected 1 restored, got %d", result.Restored)
		}

		// Two new commits should have been created: pre-restore + post-restore.
		finalCommits := countCommits(t, backupDir)
		expected := initialCommits + 2
		if finalCommits != expected {
			t.Fatalf("commit count = %d, want %d (initial + pre + post)", finalCommits, expected)
		}

		// Verify commit messages.
		messages := listCommitMessages(t, backupDir)
		foundPre := false
		foundPost := false
		for _, msg := range messages {
			if strings.Contains(msg, "bak: pre-restore snapshot") {
				foundPre = true
			}
			if strings.Contains(msg, "bak: restored:") {
				foundPost = true
			}
		}
		if !foundPre {
			t.Fatal("pre-restore commit not found")
		}
		if !foundPost {
			t.Fatal("post-restore commit not found")
		}
	})

	t.Run("git safety gracefully skipped when not a git repo", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		homeDir := t.TempDir()
		backupDir := t.TempDir()

		// Backup dir is NOT a git repo.
		mustWrite(t, filepath.Join(backupDir, "opencode", "opencode.json"), `{"theme":"dark"}`)

		m := buildTestManifest(t, backupDir, []manifestItem{
			{source: "~/.config/opencode/opencode.json", backup: "opencode/opencode.json"},
		})

		engine := &Engine{
			HomeDir:   homeDir,
			BackupDir: backupDir,
			DryRun:    false,
			Force:     true,
			GitDir:    backupDir, // Points to non-repo dir.
		}

		result, err := engine.Run(m)
		if err != nil {
			t.Fatalf("Run failed unexpectedly: %v", err)
		}

		// Should still restore files even without git.
		if result.Restored != 1 {
			t.Fatalf("expected 1 restored even without git, got %d", result.Restored)
		}
	})

	t.Run("git safety skipped when GitDir is empty", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		homeDir := t.TempDir()
		backupDir := t.TempDir()

		mustWrite(t, filepath.Join(backupDir, "opencode", "opencode.json"), `{"theme":"dark"}`)

		m := buildTestManifest(t, backupDir, []manifestItem{
			{source: "~/.config/opencode/opencode.json", backup: "opencode/opencode.json"},
		})

		engine := &Engine{
			HomeDir:   homeDir,
			BackupDir: backupDir,
			DryRun:    false,
			Force:     true,
			GitDir:    "", // No git configured.
		}

		result, err := engine.Run(m)
		if err != nil {
			t.Fatalf("Run failed: %v", err)
		}

		if result.Restored != 1 {
			t.Fatalf("expected 1 restored, got %d", result.Restored)
		}
	})
}

// --- git test helpers ---

func initTestRepo(t *testing.T, dir string) {
	t.Helper()
	_, err := gogit.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init test repo: %v", err)
	}
}

func countCommits(t *testing.T, dir string) int {
	t.Helper()
	repo, err := gogit.PlainOpen(dir)
	if err != nil {
		// No repo = 0 commits.
		return 0
	}
	head, err := repo.Head()
	if err != nil {
		return 0
	}
	iter, err := repo.Log(&gogit.LogOptions{From: head.Hash()})
	if err != nil {
		t.Fatalf("log: %v", err)
	}
	count := 0
	iter.ForEach(func(c *object.Commit) error {
		count++
		return nil
	})
	return count
}

func listCommitMessages(t *testing.T, dir string) []string {
	t.Helper()
	repo, err := gogit.PlainOpen(dir)
	if err != nil {
		return nil
	}
	head, err := repo.Head()
	if err != nil {
		return nil
	}
	iter, err := repo.Log(&gogit.LogOptions{From: head.Hash()})
	if err != nil {
		t.Fatalf("log: %v", err)
	}
	var msgs []string
	iter.ForEach(func(c *object.Commit) error {
		msgs = append(msgs, c.Message)
		return nil
	})
	return msgs
}
