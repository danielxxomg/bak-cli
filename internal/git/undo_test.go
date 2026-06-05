package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestUndo(t *testing.T) {
	t.Run("reverts last commit restoring previous state", func(t *testing.T) {
		dir := t.TempDir()
		repo := mustInit(t, dir)

		// Create initial state and commit.
		filePath := filepath.Join(dir, "config.json")
		mustWrite(t, filePath, `{"version": 1}`)
		mustCommit(t, repo, "bak: initial state — 2026-06-04 20:00:00")

		// Modify and commit.
		mustWrite(t, filePath, `{"version": 2}`)
		mustCommit(t, repo, "bak: modified — 2026-06-04 20:01:00")

		// Undo the last commit.
		if err := Undo(repo); err != nil {
			t.Fatalf("Undo failed: %v", err)
		}

		// Verify file content is back to version 1.
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("read file: %v", err)
		}
		if string(content) != `{"version": 1}` {
			t.Fatalf("file content = %q, want {\"version\": 1}", string(content))
		}

		// Verify history has 3 commits (initial + modified + revert).
		head, _ := repo.Head()
		commitIter, err := repo.Log(&gogit.LogOptions{From: head.Hash()})
		if err != nil {
			t.Fatalf("log: %v", err)
		}
		count := 0
		commitIter.ForEach(func(c *object.Commit) error {
			count++
			return nil
		})
		if count != 3 {
			t.Fatalf("commit count = %d, want 3 (initial + modified + revert)", count)
		}
	})

	t.Run("undo with multiple files preserves all original content", func(t *testing.T) {
		dir := t.TempDir()
		repo := mustInit(t, dir)

		// Initial commit with two files.
		mustWrite(t, filepath.Join(dir, "a.txt"), "A1")
		mustWrite(t, filepath.Join(dir, "b.txt"), "B1")
		mustCommit(t, repo, "bak: initial — 2026-06-04 20:00:00")

		// Change both files.
		mustWrite(t, filepath.Join(dir, "a.txt"), "A2")
		mustWrite(t, filepath.Join(dir, "b.txt"), "B2")
		mustCommit(t, repo, "bak: modified both — 2026-06-04 20:01:00")

		// Undo.
		if err := Undo(repo); err != nil {
			t.Fatalf("Undo: %v", err)
		}

		// Both files should be back to original.
		a, _ := os.ReadFile(filepath.Join(dir, "a.txt"))
		b, _ := os.ReadFile(filepath.Join(dir, "b.txt"))
		if string(a) != "A1" {
			t.Fatalf("a.txt = %q, want A1", string(a))
		}
		if string(b) != "B1" {
			t.Fatalf("b.txt = %q, want B1", string(b))
		}
	})

	t.Run("undo revert commit message contains bak prefix", func(t *testing.T) {
		dir := t.TempDir()
		repo := mustInit(t, dir)

		mustWrite(t, filepath.Join(dir, "data"), "v1")
		mustCommit(t, repo, "bak: first — 2026-06-04 20:00:00")
		mustWrite(t, filepath.Join(dir, "data"), "v2")
		mustCommit(t, repo, "bak: second — 2026-06-04 20:01:00")

		if err := Undo(repo); err != nil {
			t.Fatalf("Undo: %v", err)
		}

		head, _ := repo.Head()
		commit, _ := repo.CommitObject(head.Hash())
		if !strings.HasPrefix(commit.Message, "bak: undo") {
			t.Fatalf("revert commit message = %q, want prefix \"bak: undo\"", commit.Message)
		}
	})

	t.Run("undo on initial commit returns error", func(t *testing.T) {
		dir := t.TempDir()
		repo := mustInit(t, dir)

		mustWrite(t, filepath.Join(dir, "only.txt"), "only content")
		mustCommit(t, repo, "bak: only commit — 2026-06-04 20:00:00")

		err := Undo(repo)
		if err == nil {
			t.Fatal("expected error undoing initial commit, got nil")
		}
	})
}
