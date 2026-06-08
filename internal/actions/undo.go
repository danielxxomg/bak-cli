package actions

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// UndoAction reverts the last bak operation via git revert.
// All dependencies are injectable for testability.
type UndoAction struct {
	// Stdout receives the success message.
	Stdout io.Writer

	// HomeDir returns the user home directory. Defaults to os.UserHomeDir.
	HomeDir func() (string, error)

	// BakDir builds the bak directory path from homeDir.
	// Defaults to filepath.Join(homeDir, ".bak").
	BakDir func(homeDir string) string

	// IsRepo checks if the given path is a git repository.
	IsRepo func(path string) bool

	// UndoFn performs a git revert on the repository at the given path.
	UndoFn func(repoPath string) error
}

// Run executes the undo workflow: open the bak git repository
// and create a revert commit.
func (a *UndoAction) Run() error {
	homeFn := a.HomeDir
	if homeFn == nil {
		homeFn = os.UserHomeDir
	}
	bakFn := a.BakDir
	if bakFn == nil {
		bakFn = func(homeDir string) string { return filepath.Join(homeDir, ".bak") }
	}

	homeDir, err := homeFn()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	bakDir := bakFn(homeDir)

	if a.IsRepo != nil && !a.IsRepo(bakDir) {
		return fmt.Errorf("no bak repository found — run 'bak backup' first")
	}

	if a.UndoFn != nil {
		if err := a.UndoFn(bakDir); err != nil {
			return fmt.Errorf("undo failed: %w", err)
		}
	}

	_, _ = fmt.Fprintln(a.Stdout, "✅ Reverted to previous state")
	return nil
}
