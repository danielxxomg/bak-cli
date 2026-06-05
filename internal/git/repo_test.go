package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitRepo(t *testing.T) {
	t.Run("creates new git repository in empty directory", func(t *testing.T) {
		dir := t.TempDir()

		repo, err := InitRepo(dir)
		if err != nil {
			t.Fatalf("InitRepo failed: %v", err)
		}
		if repo == nil {
			t.Fatal("InitRepo returned nil repository")
		}

		// Verify .git directory exists.
		gitDir := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitDir); err != nil || !info.IsDir() {
			t.Fatalf(".git directory not created at %s", gitDir)
		}
	})

	t.Run("returns error when path is a file not a directory", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "notadir")
		if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
			t.Fatalf("setup: %v", err)
		}

		_, err := InitRepo(filePath)
		if err == nil {
			t.Fatal("expected error when path is a file, got nil")
		}
	})
}

func TestOpenRepo(t *testing.T) {
	t.Run("opens existing repository", func(t *testing.T) {
		dir := t.TempDir()

		// Create a repo first.
		if _, err := InitRepo(dir); err != nil {
			t.Fatalf("setup: %v", err)
		}

		repo, err := OpenRepo(dir)
		if err != nil {
			t.Fatalf("OpenRepo failed: %v", err)
		}
		if repo == nil {
			t.Fatal("OpenRepo returned nil repository")
		}
	})

	t.Run("returns error for non-repository directory", func(t *testing.T) {
		dir := t.TempDir()

		_, err := OpenRepo(dir)
		if err == nil {
			t.Fatal("expected error for non-repo directory, got nil")
		}
	})
}

func TestInitRepo_ExistingRepo(t *testing.T) {
	t.Run("re-initializing already initialized repo returns error", func(t *testing.T) {
		dir := t.TempDir()

		if _, err := InitRepo(dir); err != nil {
			t.Fatalf("setup: %v", err)
		}

		// Re-init on existing repo should fail.
		_, err := InitRepo(dir)
		if err == nil {
			t.Fatal("expected error re-initializing existing repo, got nil")
		}
	})
}

func TestOpenRepo_RemovedRepo(t *testing.T) {
	t.Run("returns error after .git is deleted", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := InitRepo(dir); err != nil {
			t.Fatalf("setup: %v", err)
		}

		// Delete .git directory.
		if err := os.RemoveAll(filepath.Join(dir, ".git")); err != nil {
			t.Fatalf("setup remove: %v", err)
		}

		_, err := OpenRepo(dir)
		if err == nil {
			t.Fatal("expected error after deleting .git, got nil")
		}
	})
}

func TestIsRepo(t *testing.T) {
	t.Run("returns true for initialized repository", func(t *testing.T) {
		dir := t.TempDir()

		if _, err := InitRepo(dir); err != nil {
			t.Fatalf("setup: %v", err)
		}

		if !IsRepo(dir) {
			t.Fatal("IsRepo returned false for initialized repo")
		}
	})

	t.Run("returns false for non-repository directory", func(t *testing.T) {
		dir := t.TempDir()

		if IsRepo(dir) {
			t.Fatal("IsRepo returned true for non-repo directory")
		}
	})

	t.Run("returns false for nonexistent path", func(t *testing.T) {
		if IsRepo("/nonexistent/path") {
			t.Fatal("IsRepo returned true for nonexistent path")
		}
	})
}
