package actions

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// UndoAction reverts the last restore operation to its pre-restore state
// with fail-closed drift protection, and creates a revert commit in the bak repo
// when present.
// All dependencies are injectable for testability.
type UndoAction struct {
	// FS is the file system for live target files. Defaults to &OSFileSystem{}.
	FS FileSystem

	// StorageFS is the file system for the recovery repository. Defaults to &OSFileSystem{}.
	StorageFS FileSystem

	// RecoveryDir optionally overrides the base recovery directory.
	// When empty, defaults to filepath.Join(homeDir, ".bak", "recovery").
	RecoveryDir string

	// Stdout receives informational output.
	Stdout io.Writer

	// HomeDir returns the user home directory. Defaults to FS.UserHomeDir.
	HomeDir func() (string, error)

	// BakDir builds the bak directory path from homeDir.
	// Defaults to filepath.Join(homeDir, ".bak").
	BakDir func(homeDir string) string

	// IsRepo checks if the given path is a git repository.
	IsRepo func(path string) bool

	// UndoFn performs a git revert on the repository at the given path.
	UndoFn func(repoPath string) error
}

func (a *UndoAction) resolveDirectories() (homeDir, bakDir, recBase string, err error) {
	homeFn := a.HomeDir
	if homeFn == nil {
		homeFn = a.FS.UserHomeDir
	}
	bakFn := a.BakDir
	if bakFn == nil {
		bakFn = func(homeDir string) string { return filepath.Join(homeDir, ".bak") }
	}

	homeDir, err = homeFn()
	if err != nil {
		return "", "", "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	bakDir = bakFn(homeDir)
	recBase = a.RecoveryDir
	if recBase == "" {
		recBase = filepath.Join(homeDir, ".bak", "recovery")
	}
	return homeDir, bakDir, recBase, nil
}

func reportOutcome(out io.Writer, outcome rollbackOutcome, homeDir string) error {
	var b strings.Builder
	for _, p := range outcome.Reverted {
		fmt.Fprintf(&b, "reverted: %s\n", sanitizePath(p, homeDir))
	}
	for _, p := range outcome.Unresolved {
		fmt.Fprintf(&b, "unresolved: %s\n", sanitizePath(p, homeDir))
	}
	if _, err := fmt.Fprint(out, b.String()); err != nil {
		return fmt.Errorf("write outcome: %w", err)
	}
	return nil
}

func (a *UndoAction) initAction() (io.Writer, string, string, string, error) {
	if a.FS == nil {
		a.FS = &OSFileSystem{}
	}
	if a.StorageFS == nil {
		a.StorageFS = &OSFileSystem{}
	}
	out := a.Stdout
	if out == nil {
		out = io.Discard
	}

	homeDir, bakDir, recBase, err := a.resolveDirectories()
	if err != nil {
		return nil, "", "", "", err
	}
	return out, homeDir, bakDir, recBase, nil
}

func (a *UndoAction) discoverRecoveryManager(recBase, homeDir string) (*recoveryManager, error) {
	meta, err := findLatestAppliedPoint(a.StorageFS, recBase, homeDir)
	if err != nil {
		if errors.Is(err, ErrNoAppliedRecoveryPoint) {
			return nil, fmt.Errorf("nothing to undo — no applied restore found: %w", err)
		}
		return nil, fmt.Errorf("discover recovery point: %w", err)
	}

	if meta.PointID == "" || meta.PointID != filepath.Base(meta.PointID) || meta.PointID == "." || meta.PointID == ".." {
		return nil, fmt.Errorf("invalid recovery point id: %s", meta.PointID)
	}

	pointDir := filepath.Join(recBase, meta.PointID)
	return &recoveryManager{
		FS:          a.FS,
		StorageFS:   a.StorageFS,
		HomeDir:     homeDir,
		BackupID:    meta.BackupID,
		PointID:     meta.PointID,
		PointDir:    pointDir,
		RepoDir:     filepath.Join(pointDir, "repo"),
		RecoveryDir: recBase,
		Meta:        *meta,
	}, nil
}

func (a *UndoAction) applyUndo(recMgr *recoveryManager, out io.Writer, homeDir string) error {
	outcome, undoErr := recMgr.UndoTargets()
	repErr := reportOutcome(out, outcome, homeDir)
	if undoErr != nil || len(outcome.Unresolved) > 0 {
		if undoErr == nil {
			undoErr = fmt.Errorf("unresolved targets remain")
		}
		if repErr != nil {
			return errors.Join(fmt.Errorf("undo targets: %w", undoErr), repErr)
		}
		return fmt.Errorf("undo targets: %w", undoErr)
	}
	if repErr != nil {
		return repErr
	}
	return nil
}

// Run executes the undo workflow: resolves the home directory, discovers the
// latest applied recovery point, verifies targets against post-restore state
// for drift BEFORE any mutation or commit, projects pre-state back to target
// files, records reverted and unresolved targets with sanitized paths, and
// creates a revert commit in the bak repository when present.
func (a *UndoAction) Run() error {
	out, homeDir, bakDir, recBase, err := a.initAction()
	if err != nil {
		return err
	}

	recMgr, err := a.discoverRecoveryManager(recBase, homeDir)
	if err != nil {
		return err
	}

	if err := recMgr.CheckUndoDrift(); err != nil {
		return fmt.Errorf("check undo drift: %w", err)
	}

	if err := a.applyUndo(recMgr, out, homeDir); err != nil {
		return err
	}

	if a.UndoFn != nil && a.IsRepo != nil && a.IsRepo(bakDir) {
		if err := a.UndoFn(bakDir); err != nil {
			return fmt.Errorf("undo failed: %w", err)
		}
	}

	if _, err := fmt.Fprintln(out, "✅ Reverted to previous state"); err != nil {
		return fmt.Errorf("write confirmation: %w", err)
	}
	return nil
}
