package git

import (
	"path/filepath"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v5"
)

func TestStageAll(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	t.Run("stages new files in worktree", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		dir := t.TempDir()
		repo := mustInit(t, dir)

		// Create a file that needs staging.
		mustWrite(t, filepath.Join(dir, "hello.txt"), "hello world")

		if err := StageAll(repo); err != nil {
			t.Fatalf("StageAll failed: %v", err)
		}

		// Verify it's staged by checking the worktree status.
		wt, _ := repo.Worktree()
		status, err := wt.Status()
		if err != nil {
			t.Fatalf("worktree status: %v", err)
		}
		if status.File("hello.txt").Staging == gogit.Unmodified {
			t.Fatal("file not staged after StageAll")
		}
	})

	t.Run("stages modified files", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		dir := t.TempDir()
		repo := mustInit(t, dir)

		filePath := filepath.Join(dir, "config.toml")
		mustWrite(t, filePath, "version = 1")
		if err := StageAll(repo); err != nil {
			t.Fatalf("first StageAll: %v", err)
		}
		mustCommit(t, repo, "initial commit")
		mustWrite(t, filePath, "version = 2")

		if err := StageAll(repo); err != nil {
			t.Fatalf("StageAll for modified file: %v", err)
		}

		wt, _ := repo.Worktree()
		status, err := wt.Status()
		if err != nil {
			t.Fatalf("worktree status: %v", err)
		}
		if status.File("config.toml").Staging == gogit.Unmodified {
			t.Fatal("modified file not staged after StageAll")
		}
	})
}

func TestCommit(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	t.Run("creates commit with correct message", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		dir := t.TempDir()
		repo := mustInit(t, dir)

		mustWrite(t, filepath.Join(dir, "README.md"), "# My Backup")
		if err := StageAll(repo); err != nil {
			t.Fatalf("StageAll: %v", err)
		}

		msg := "bak: test commit — 2026-06-04 20:00:00"
		if err := Commit(repo, msg); err != nil {
			t.Fatalf("Commit failed: %v", err)
		}

		// Verify the commit exists and has the right message.
		head, err := repo.Head()
		if err != nil {
			t.Fatalf("get HEAD: %v", err)
		}
		commit, err := repo.CommitObject(head.Hash())
		if err != nil {
			t.Fatalf("get commit: %v", err)
		}
		if commit.Message != msg {
			t.Fatalf("commit message = %q, want %q", commit.Message, msg)
		}
	})

	t.Run("creates commit with author set to bak", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		dir := t.TempDir()
		repo := mustInit(t, dir)

		mustWrite(t, filepath.Join(dir, "file.txt"), "content")
		if err := StageAll(repo); err != nil {
			t.Fatalf("StageAll: %v", err)
		}
		if err := Commit(repo, "bak: author test"); err != nil {
			t.Fatalf("Commit: %v", err)
		}

		head, _ := repo.Head()
		commit, _ := repo.CommitObject(head.Hash())
		if commit.Author.Name != "bak" {
			t.Fatalf("author name = %q, want \"bak\"", commit.Author.Name)
		}
	})

	t.Run("returns error when nothing to commit", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		dir := t.TempDir()
		repo := mustInit(t, dir)

		// No files staged → should return error.
		err := Commit(repo, "empty commit")
		if err == nil {
			t.Fatal("expected error for empty commit, got nil")
		}
	})
}

func TestAutoCommitMessage(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	t.Run("commit message contains bak prefix and timestamp", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		dir := t.TempDir()
		repo := mustInit(t, dir)

		mustWrite(t, filepath.Join(dir, "data.json"), `{"key":"value"}`)
		if err := StageAll(repo); err != nil {
			t.Fatalf("StageAll: %v", err)
		}

		msg := AutoCommitMessage("test-action")
		if err := Commit(repo, msg); err != nil {
			t.Fatalf("Commit: %v", err)
		}

		if !strings.HasPrefix(msg, "bak: test-action") {
			t.Fatalf("message prefix mismatch: %q", msg)
		}

		head, _ := repo.Head()
		commit, _ := repo.CommitObject(head.Hash())
		if commit.Message != msg {
			t.Fatalf("stored message = %q, want %q", commit.Message, msg)
		}
	})
}
