// Package git provides go-git wrappers for repository management,
// auto-commit, and safe undo via git revert. All operations are
// local-only — no remotes, no force-push, no history rewrite.
package git

import (
	"fmt"

	gogit "github.com/go-git/go-git/v5"
)

// InitRepo initializes a new git repository at the given path.
func InitRepo(path string) (*gogit.Repository, error) {
	repo, err := gogit.PlainInit(path, false)
	if err != nil {
		return nil, fmt.Errorf("init repo at %s: %w", path, err)
	}
	return repo, nil
}

// OpenRepo opens an existing git repository at the given path.
func OpenRepo(path string) (*gogit.Repository, error) {
	repo, err := gogit.PlainOpen(path)
	if err != nil {
		return nil, fmt.Errorf("open repo at %s: %w", path, err)
	}
	return repo, nil
}

// IsRepo returns true when path contains a git repository.
func IsRepo(path string) bool {
	_, err := gogit.PlainOpen(path)
	return err == nil
}
