package git

import (
	"os"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v5"
)

// mustInit creates a git repo at dir and fails the test on error.
func mustInit(t *testing.T, dir string) *gogit.Repository {
	t.Helper()
	repo, err := InitRepo(dir)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}
	return repo
}

// mustWrite creates a file with content and fails the test on error.
func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

// mustCommit stages all, commits with message, and fails the test on error.
func mustCommit(t *testing.T, repo *gogit.Repository, msg string) *gogit.Repository {
	t.Helper()
	if err := StageAll(repo); err != nil {
		t.Fatalf("stage all: %v", err)
	}
	if err := Commit(repo, msg); err != nil {
		t.Fatalf("commit: %v", err)
	}
	return repo
}
