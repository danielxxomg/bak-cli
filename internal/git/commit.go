package git

import (
	"fmt"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// StageAll stages all new and modified files in the worktree.
func StageAll(repo *gogit.Repository) error {
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}
	if err := wt.AddWithOptions(&gogit.AddOptions{All: true}); err != nil {
		return fmt.Errorf("stage files: %w", err)
	}
	return nil
}

// Commit creates a commit with the given message. All previously staged
// files are included.
func Commit(repo *gogit.Repository, message string) error {
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}

	_, err = wt.Commit(message, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "bak",
			Email: "bak@local",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// AutoCommitMessage builds a standard commit message for bak operations.
func AutoCommitMessage(action string) string {
	return fmt.Sprintf("bak: %s — %s", action, time.Now().Format("2006-01-02 15:04:05"))
}
