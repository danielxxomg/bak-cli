package git

import (
	"fmt"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Undo reverts the last commit by creating a new revert commit on top
// of HEAD. The revert commit has the same tree as HEAD~1, effectively
// undoing HEAD's changes while preserving history. This is equivalent
// to `git revert HEAD --no-edit`.
//
// Returns an error when HEAD has no parent (initial commit).
func Undo(repo *gogit.Repository) error {
	headRef, err := repo.Head()
	if err != nil {
		return fmt.Errorf("get HEAD: %w", err)
	}

	headCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return fmt.Errorf("get HEAD commit: %w", err)
	}

	// Get parent (HEAD~1). An initial commit has no parent.
	parentCommit, err := headCommit.Parent(0)
	if err != nil {
		return fmt.Errorf("cannot undo: HEAD has no parent to revert to: %w", err)
	}

	// Build the revert commit. Tree = parent's tree (undone state).
	// Parent = HEAD (keeps history linear — no force-push, no rewrite).
	sig := object.Signature{
		Name:  "bak",
		Email: "bak@local",
		When:  time.Now(),
	}

	message := fmt.Sprintf("bak: undo — reverted to previous state — %s",
		time.Now().Format("2006-01-02 15:04:05"))

	revertCommit := &object.Commit{
		Author:       sig,
		Committer:    sig,
		Message:      message,
		TreeHash:     parentCommit.TreeHash,
		ParentHashes: []plumbing.Hash{headCommit.Hash},
	}

	// Store the commit object.
	obj := repo.Storer.NewEncodedObject()
	if err := revertCommit.Encode(obj); err != nil {
		return fmt.Errorf("encode revert commit: %w", err)
	}
	newHash, err := repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return fmt.Errorf("store revert commit: %w", err)
	}

	// Update the branch reference to point to the new revert commit.
	newRef := plumbing.NewHashReference(headRef.Name(), newHash)
	if err := repo.Storer.SetReference(newRef); err != nil {
		return fmt.Errorf("update reference: %w", err)
	}

	// Update the working tree to match the reverted state.
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("get worktree: %w", err)
	}
	if err := wt.Checkout(&gogit.CheckoutOptions{
		Hash:  newHash,
		Force: true,
	}); err != nil {
		return fmt.Errorf("checkout revert: %w", err)
	}

	return nil
}
