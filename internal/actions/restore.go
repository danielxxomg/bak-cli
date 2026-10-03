package actions

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/danielxxomg/bak-cli/internal/manifest"
	"github.com/danielxxomg/bak-cli/internal/paths"
	restorepkg "github.com/danielxxomg/bak-cli/internal/restore"
)

// RestoreAction encapsulates the restore workflow with injectable
// dependencies. All OS operations that are the action's responsibility
// go through a.FS.
type RestoreAction struct {
	FS         FileSystem
	BackupDir  string // resolved backup directory
	BakVersion string // version of running bak tool; zero value means unknown
	DryRun     bool
	Force      bool
	Verbose    bool
	GitDir     string // optional git repo for safety commits

	// RecoveryDir optionally overrides the base recovery directory.
	// When empty, defaults to <home>/.bak/recovery.
	RecoveryDir string

	storageFS FileSystem // unexported injection for recovery storage in tests

	// ProgressFn is an optional callback invoked once per file during restore.
	// When nil (default), no progress is reported.
	ProgressFn func(currentFile string, filesDone int, filesTotal int)

	// Stdin is the reader for confirmation prompts. Nil falls back to os.Stdin.
	Stdin io.Reader
	// Stdout receives informational output. Nil falls back to os.Stdout.
	Stdout io.Writer
	// Stderr receives warnings and error diagnostics. Nil falls back to os.Stderr.
	Stderr io.Writer
}

// ResolveBackup resolves the backup ID to a directory path and sets a.BackupDir.
func (a *RestoreAction) ResolveBackup(backupID string) error {
	dir, err := backup.ResolveBackupID(backupID)
	if err != nil {
		return fmt.Errorf("resolve backup %q: %w", backupID, err)
	}
	a.BackupDir = dir
	return nil
}

// handlePostStateFailure attempts rollback when post-state recording fails.
func handlePostStateFailure(recMgr *recoveryManager, diffs []restorepkg.FileDiff, err error) error {
	var attempted []string
	for _, d := range diffs {
		if d.Status == restorepkg.DiffNew || d.Status == restorepkg.DiffModified {
			attempted = append(attempted, d.TargetPath)
		}
	}
	outcome := recMgr.Rollback(attempted)
	var errSuffix string
	if len(outcome.Errors) > 0 {
		errMsgs := make([]string, 0, len(outcome.Errors))
		for _, e := range outcome.Errors {
			errMsgs = append(errMsgs, e.Error())
		}
		errSuffix = "; " + strings.Join(errMsgs, "; ")
	}
	if len(outcome.Unresolved) > 0 {
		return fmt.Errorf("record post-restore state: %w (rollback unresolved for %d file(s); recovery point: %s%s)", err, len(outcome.Unresolved), recMgr.PointID, errSuffix)
	}
	return fmt.Errorf("record post-restore state: %w (all targets rolled back to pre-restore state; recovery point: %s%s)", err, recMgr.PointID, errSuffix)
}

// Run executes the restore workflow: load manifest, compute diffs, and
// optionally apply changes. Each phase is delegated to a helper to keep the
// orchestration readable and below the cognitive-complexity threshold.
func (a *RestoreAction) Run() error {
	out, errOut := a.resolveWriters()

	// 1. Load manifest.
	m, err := manifest.Load(a.BackupDir)
	if err != nil {
		return fmt.Errorf("load manifest: %w", err)
	}

	a.warnBakVersionMismatch(m.ID, m.BakVersion, errOut)

	// 2. Get home directory.
	homeDir, err := a.FS.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	// 3. Compute dry-run diffs.
	diffs, err := restorepkg.ComputeDryRun(m, a.BackupDir, homeDir)
	if err != nil {
		return fmt.Errorf("compute dry-run: %w", err)
	}

	// 4. Show diffs.
	a.printDryRunDiff(out, diffs)

	if a.DryRun {
		_, _ = fmt.Fprintf(out, "Dry-run complete. %d file(s) would be restored, %d unchanged, %d missing.\n",
			countByStatus(diffs, restorepkg.DiffNew)+countByStatus(diffs, restorepkg.DiffModified),
			countByStatus(diffs, restorepkg.DiffUnchanged),
			countByStatus(diffs, restorepkg.DiffMissing),
		)
		return nil
	}

	// 5. Validate manifest checksums before applying.
	if err := a.validateManifest(m); err != nil {
		return err
	}

	// 6. Confirmation prompt (unless --force).
	proceed, err := a.confirmRestore(out, errOut)
	if err != nil {
		return err
	}
	if !proceed {
		return nil
	}

	// 7. Setup recovery manager and prepare pre-state before any target mutation.
	recMgr := &recoveryManager{
		FS:          a.FS,
		StorageFS:   a.storageFS,
		HomeDir:     homeDir,
		BackupID:    m.ID,
		RecoveryDir: a.RecoveryDir,
	}

	if err := recMgr.Prepare(diffs); err != nil {
		return fmt.Errorf("prepare recovery snapshot: %w", err)
	}

	// 8. Apply restore with recovery and automatic rollback on first failure.
	restored, skipped, failed, degraded, applyErr := a.applyRestore(m, diffs, recMgr, out, errOut)

	if failed > 0 || applyErr != nil {
		reportRestore(out, m, restored, skipped, failed, degraded)
		return applyErr
	}

	// Record post-state commit on successful apply.
	if err := recMgr.RecordPostState(); err != nil {
		reportRestore(out, m, 0, skipped, countByStatus(diffs, restorepkg.DiffNew)+countByStatus(diffs, restorepkg.DiffModified), degraded)
		return handlePostStateFailure(recMgr, diffs, err)
	}

	// 9. Report results.
	reportRestore(out, m, restored, skipped, failed, degraded)

	return nil
}

// resolveWriters returns the output and error writers, falling back to
// os.Stdout / os.Stderr when the action's fields are nil.
func (a *RestoreAction) resolveWriters() (io.Writer, io.Writer) {
	out := a.Stdout
	if out == nil {
		out = os.Stdout
	}
	errOut := a.Stderr
	if errOut == nil {
		errOut = os.Stderr
	}
	return out, errOut
}

const unknownVersion = "unknown"

// warnBakVersionMismatch checks for version compatibility between the running bak
// tool and the version that created the backup. Any mismatch or unknown/dev version
// on either side produces a warning on errOut naming the backup id and both versions,
// but does not block restore.
func (a *RestoreAction) warnBakVersionMismatch(backupID, manifestBakVersion string, errOut io.Writer) {
	runningVer := a.BakVersion
	if runningVer == "" {
		runningVer = unknownVersion
	}
	backupVer := manifestBakVersion
	if backupVer == "" {
		backupVer = unknownVersion
	}

	id := backupID
	if id == "" {
		id = unknownVersion
	}

	isUnknownOrDev := func(v string) bool {
		return v == unknownVersion || v == "dev"
	}

	if isUnknownOrDev(runningVer) || isUnknownOrDev(backupVer) || runningVer != backupVer {
		_, _ = fmt.Fprintf(errOut, "warning: backup %s was created with bak %s, running version is %s\n", id, backupVer, runningVer)
	}
}

// printDryRunDiff writes the per-file dry-run diff to out. When verbose and
// a file is modified with a non-empty diff, the unified diff is appended.
func (a *RestoreAction) printDryRunDiff(out io.Writer, diffs []restorepkg.FileDiff) {
	if len(diffs) == 0 {
		return
	}
	_, _ = fmt.Fprintln(out, "Dry-run diff:")
	for _, d := range diffs {
		_, _ = fmt.Fprintf(out, "  [%s] %s\n", d.Status, d.SourcePath)
		if d.Status == restorepkg.DiffModified && d.Diff != "" && a.Verbose {
			_, _ = fmt.Fprint(out, d.Diff)
		}
	}
	_, _ = fmt.Fprintln(out)
}

// validateManifest validates checksums of all backed-up files against the
// manifest. Integrity validation is mandatory and always fails hard on mismatch,
// with or without --force, before any file write.
func (a *RestoreAction) validateManifest(m *manifest.Manifest) error {
	if err := m.Validate(a.BackupDir, nil); err != nil {
		return fmt.Errorf("manifest validation failed: %w", err)
	}
	return nil
}

// confirmRestore prompts the user (unless --force) and reports whether the
// restore should proceed. A "y"/"yes" answer (or --force) returns true; any
// other answer prints "Restore cancelled." and returns false. A read error
// is propagated.
func (a *RestoreAction) confirmRestore(out, errOut io.Writer) (bool, error) {
	if a.Force {
		return true, nil
	}
	_, _ = fmt.Fprint(out, "Apply restore? [y/N]: ")
	stdin := a.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	reader := bufio.NewReader(stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("read input: %w", err)
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		_, _ = fmt.Fprintln(errOut, "Restore cancelled.")
		return false, nil
	}
	return true, nil
}

func formatRollbackOutcome(outcome rollbackOutcome) string {
	var base string
	switch {
	case len(outcome.Unresolved) > 0:
		base = fmt.Sprintf("rollback unresolved for %d file(s); recovery point: %s", len(outcome.Unresolved), outcome.PointID)
	case len(outcome.Reverted) > 0:
		base = fmt.Sprintf("reverted %d attempted target(s); recovery point: %s", len(outcome.Reverted), outcome.PointID)
	default:
		base = fmt.Sprintf("recovery point: %s", outcome.PointID)
	}
	if len(outcome.Errors) > 0 {
		errMsgs := make([]string, 0, len(outcome.Errors))
		for _, e := range outcome.Errors {
			errMsgs = append(errMsgs, e.Error())
		}
		return fmt.Sprintf("%s; %s", base, strings.Join(errMsgs, "; "))
	}
	return base
}

func buildModeMap(m *manifest.Manifest) map[string]uint32 {
	modeMap := make(map[string]uint32)
	for _, am := range m.Adapters {
		for _, item := range am.Items {
			modeMap[item.BackupPath] = item.Mode
		}
	}
	return modeMap
}

func countRestoreTargets(diffs []restorepkg.FileDiff) int {
	filesTotal := 0
	for _, d := range diffs {
		if d.Status == restorepkg.DiffNew || d.Status == restorepkg.DiffModified {
			filesTotal++
		}
	}
	return filesTotal
}

// applyRestore copies each new/modified file, stopping at the first failure
// and attempting rollback of all attempted targets. Returns the counts of
// restored, skipped, failed, and degraded files along with an aggregated error if any
// copy or chmod failed.
func (a *RestoreAction) applyRestore(m *manifest.Manifest, diffs []restorepkg.FileDiff, recMgr *recoveryManager, _, errOut io.Writer) (restored, skipped, failed, degraded int, err error) {
	modeMap := buildModeMap(m)
	filesTotal := countRestoreTargets(diffs)
	filesDone := 0
	var attemptedTargets []string

	for _, d := range diffs {
		switch d.Status {
		case restorepkg.DiffNew, restorepkg.DiffModified:
			filesDone++
			if a.ProgressFn != nil {
				a.ProgressFn(d.SourcePath, filesDone, filesTotal)
			}
			mode := modeMap[d.BackupPath]
			if mode == 0 {
				degraded++
			}
			attemptedTargets = append(attemptedTargets, d.TargetPath)
			if err := a.restoreFile(d, mode); err != nil {
				failed++
				if a.Verbose {
					_, _ = fmt.Fprintf(errOut, "restore %s: %v\n", d.SourcePath, err)
				}
				var outcome rollbackOutcome
				if recMgr != nil {
					outcome = recMgr.Rollback(attemptedTargets)
				}
				rollbackMsg := formatRollbackOutcome(outcome)
				return restored, skipped, failed, degraded, fmt.Errorf("restore %s: %w (%s)", d.SourcePath, err, rollbackMsg)
			}
			restored++
		case restorepkg.DiffUnchanged:
			skipped++
		case restorepkg.DiffMissing:
			skipped++
			if a.Verbose {
				_, _ = fmt.Fprintf(errOut, "warning: missing backup file %s\n", d.BackupPath)
			}
		}
	}
	return restored, skipped, failed, degraded, nil
}

// reportRestore writes the final restore summary to out, including the failed
// count only when at least one file failed and degraded permissions warning.
func reportRestore(out io.Writer, m *manifest.Manifest, restored, skipped, failed, degraded int) {
	if failed > 0 {
		_, _ = fmt.Fprintf(out, "Restore failed: %s\n", m.ID)
	} else {
		_, _ = fmt.Fprintf(out, "Restore complete: %s\n", m.ID)
	}
	_, _ = fmt.Fprintf(out, "  Restored: %d\n", restored)
	_, _ = fmt.Fprintf(out, "  Skipped:  %d\n", skipped)
	if failed > 0 {
		_, _ = fmt.Fprintf(out, "  Failed:   %d\n", failed)
	}
	if degraded > 0 {
		_, _ = fmt.Fprintf(out, "  Warning: degraded permissions (%d file(s) lack mode metadata)\n", degraded)
	}
}

// restoreFile copies a single file from the backup directory to the
// target path, creating parent directories as needed. Validates path
// traversal safety and applies stored permission bits when mode != 0.
func (a *RestoreAction) restoreFile(d restorepkg.FileDiff, mode uint32) error {
	src := filepath.Join(a.BackupDir, d.BackupPath)

	// Security: validate source path stays under backup directory.
	cleanSrc := paths.CanonicalPath(src)
	cleanBackupDir := paths.CanonicalPath(a.BackupDir) + "/"
	if !strings.HasPrefix(cleanSrc, cleanBackupDir) {
		return fmt.Errorf("source path escapes backup directory")
	}

	// Security: validate target path stays under home directory.
	homeDir, err := a.FS.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	cleanTarget := paths.CanonicalPath(d.TargetPath)
	cleanHome := paths.CanonicalPath(homeDir) + "/"
	if !strings.HasPrefix(cleanTarget, cleanHome) {
		return fmt.Errorf("target path escapes home directory")
	}

	// Ensure target parent directory exists.
	if err := a.FS.MkdirAll(filepath.Dir(d.TargetPath), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	if err := a.FS.CopyFile(src, d.TargetPath); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	if mode != 0 {
		if err := a.FS.Chmod(d.TargetPath, os.FileMode(mode)); err != nil {
			// Windows restore applies best-effort (Chmod may be a no-op for exec bits —
			// acceptable, but must not error the restore on Windows for mode-only reasons).
			if !isWindows() {
				return fmt.Errorf("chmod: %w", err)
			}
		}
	}

	return nil
}

// countByStatus returns the number of diffs with the given status.
func countByStatus(diffs []restorepkg.FileDiff, status restorepkg.DiffStatus) int {
	count := 0
	for _, d := range diffs {
		if d.Status == status {
			count++
		}
	}
	return count
}
