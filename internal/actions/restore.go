package actions

import (
	"bufio"
	"errors"
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
		if d.Status == restorepkg.DiffNew || d.Status == restorepkg.DiffModified || d.Status == restorepkg.DiffRedacted {
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

// showDryRun formats and displays dry-run diffs, and if DryRun is enabled,
// writes the summary and reports completion.
func (a *RestoreAction) showDryRun(out io.Writer, diffs []restorepkg.FileDiff) error {
	if err := a.printDryRunDiff(out, diffs); err != nil {
		return err
	}
	if !a.DryRun {
		return nil
	}
	secretExcluded := countByStatus(diffs, restorepkg.DiffSecretExcluded)
	secretRedacted := countByStatus(diffs, restorepkg.DiffRedacted)
	var suffixes []string
	if secretRedacted > 0 {
		suffixes = append(suffixes, fmt.Sprintf("%d redacted", secretRedacted))
	}
	if secretExcluded > 0 {
		suffixes = append(suffixes, fmt.Sprintf("%d excluded (secrets)", secretExcluded))
	}
	var extraSuffix string
	if len(suffixes) > 0 {
		extraSuffix = ", " + strings.Join(suffixes, ", ")
	}
	if _, err := fmt.Fprintf(out, "Dry-run complete. %d file(s) would be restored, %d unchanged, %d missing%s.\n",
		countByStatus(diffs, restorepkg.DiffNew)+countByStatus(diffs, restorepkg.DiffModified)+secretRedacted,
		countByStatus(diffs, restorepkg.DiffUnchanged),
		countByStatus(diffs, restorepkg.DiffMissing),
		extraSuffix,
	); err != nil {
		return fmt.Errorf("write dry-run summary: %w", err)
	}
	if secretRedacted > 0 {
		if _, err := fmt.Fprintf(out, "  Warning: Restoring a redacted file overwrites the live file's real secrets with placeholders. The user must re-enter them.\n"); err != nil {
			return fmt.Errorf("write dry-run warning: %w", err)
		}
	}
	if secretExcluded > 0 {
		if _, err := fmt.Fprintf(out, "  ⚠ %d secret-bearing file(s) cannot be restored and must be re-entered by hand (see .env.example).\n", secretExcluded); err != nil {
			return fmt.Errorf("write secret-excluded note: %w", err)
		}
	}
	return nil
}

// finishRestore handles post-apply reporting, post-state recording, and error propagation.
func (a *RestoreAction) finishRestore(out io.Writer, m *manifest.Manifest, diffs []restorepkg.FileDiff, recMgr *recoveryManager, restored, skipped, failed, degraded int, applyErr error) error {
	redactedFiles := collectRedactedTargets(diffs)
	if failed > 0 || applyErr != nil {
		if repErr := reportRestore(out, m, restored, skipped, failed, degraded, redactedFiles); repErr != nil && applyErr != nil {
			return errors.Join(applyErr, repErr)
		} else if repErr != nil {
			return repErr
		}
		return applyErr
	}

	if err := recMgr.RecordPostState(); err != nil {
		if repErr := reportRestore(out, m, 0, skipped, countByStatus(diffs, restorepkg.DiffNew)+countByStatus(diffs, restorepkg.DiffModified)+countByStatus(diffs, restorepkg.DiffRedacted), degraded, nil); repErr != nil {
			return errors.Join(handlePostStateFailure(recMgr, diffs, err), repErr)
		}
		return handlePostStateFailure(recMgr, diffs, err)
	}

	return reportRestore(out, m, restored, skipped, failed, degraded, redactedFiles)
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

	if err := a.warnBakVersionMismatch(m.ID, m.BakVersion, errOut); err != nil {
		return err
	}

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

	// 4. Show diffs / handle dry-run.
	if err := a.showDryRun(out, diffs); err != nil {
		return err
	}
	if a.DryRun {
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

	// 9. Post-apply reporting and recovery completion.
	return a.finishRestore(out, m, diffs, recMgr, restored, skipped, failed, degraded, applyErr)
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
func (a *RestoreAction) warnBakVersionMismatch(backupID, manifestBakVersion string, errOut io.Writer) error {
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
		if _, err := fmt.Fprintf(errOut, "warning: backup %s was created with bak %s, running version is %s\n", id, backupVer, runningVer); err != nil {
			return fmt.Errorf("write version warning: %w", err)
		}
	}
	return nil
}

// printDryRunDiff writes the per-file dry-run diff to out. When verbose and
// a file is modified with a non-empty diff, the unified diff is appended.
func (a *RestoreAction) printDryRunDiff(out io.Writer, diffs []restorepkg.FileDiff) error {
	if len(diffs) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString("Dry-run diff:\n")
	for _, d := range diffs {
		fmt.Fprintf(&b, "  [%s] %s\n", d.Status, d.SourcePath)
		if (d.Status == restorepkg.DiffModified || d.Status == restorepkg.DiffRedacted) && d.Diff != "" && a.Verbose {
			b.WriteString(d.Diff)
		}
	}
	b.WriteByte('\n')
	if _, err := fmt.Fprint(out, b.String()); err != nil {
		return fmt.Errorf("write dry-run diff: %w", err)
	}
	return nil
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
	if _, err := fmt.Fprint(out, "Apply restore? [y/N]: "); err != nil {
		return false, fmt.Errorf("write confirmation prompt: %w", err)
	}
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
		if _, err := fmt.Fprintln(errOut, "Restore cancelled."); err != nil {
			return false, fmt.Errorf("write cancellation: %w", err)
		}
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
		if d.Status == restorepkg.DiffNew || d.Status == restorepkg.DiffModified || d.Status == restorepkg.DiffRedacted {
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
		case restorepkg.DiffNew, restorepkg.DiffModified, restorepkg.DiffRedacted:
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
					if _, wErr := fmt.Fprintf(errOut, "restore %s: %v\n", d.SourcePath, err); wErr != nil {
						err = errors.Join(err, fmt.Errorf("write verbose log: %w", wErr))
					}
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
		case restorepkg.DiffMissing, restorepkg.DiffSecretExcluded:
			skipped++
			if a.Verbose {
				if _, wErr := fmt.Fprintf(errOut, "warning: missing backup file %s\n", d.BackupPath); wErr != nil {
					return restored, skipped, failed, degraded, fmt.Errorf("write verbose warning: %w", wErr)
				}
			}
		}
	}
	return restored, skipped, failed, degraded, nil
}

func collectRedactedTargets(diffs []restorepkg.FileDiff) []string {
	var targets []string
	for _, d := range diffs {
		if d.Status == restorepkg.DiffRedacted || d.Redacted {
			targets = append(targets, d.SourcePath)
		}
	}
	return targets
}

// reportRestore writes the final restore summary to out, including the failed
// count only when at least one file failed, restored placeholders, and degraded permissions warning.
func reportRestore(out io.Writer, m *manifest.Manifest, restored, skipped, failed, degraded int, redactedFiles []string) error {
	var b strings.Builder
	if failed > 0 {
		fmt.Fprintf(&b, "Restore failed: %s\n", m.ID)
	} else {
		fmt.Fprintf(&b, "Restore complete: %s\n", m.ID)
	}
	fmt.Fprintf(&b, "  Restored: %d\n", restored)
	fmt.Fprintf(&b, "  Skipped:  %d\n", skipped)
	if failed > 0 {
		fmt.Fprintf(&b, "  Failed:   %d\n", failed)
	}
	if len(redactedFiles) > 0 && restored > 0 {
		fmt.Fprintf(&b, "  ⚠ Restored with placeholders (%d file(s) — secrets must be re-entered by hand):\n", len(redactedFiles))
		for _, rf := range redactedFiles {
			fmt.Fprintf(&b, "    - %s\n", rf)
		}
		fmt.Fprintf(&b, "  Warning: Restoring a redacted file overwrites the live file's real secrets with placeholders. The user must re-enter them.\n")
	}
	if degraded > 0 {
		fmt.Fprintf(&b, "  Warning: degraded permissions (%d file(s) lack mode metadata)\n", degraded)
	}
	if _, err := fmt.Fprint(out, b.String()); err != nil {
		return fmt.Errorf("write restore report: %w", err)
	}
	return nil
}

// restoreFile copies a single file from the backup directory to the
// target path, creating parent directories as needed. Validates path
// traversal safety and applies stored permission bits when mode != 0.
func (a *RestoreAction) restoreFile(d restorepkg.FileDiff, mode uint32) error {
	src := filepath.Join(a.BackupDir, d.BackupPath)

	// Security: validate source path stays under backup directory.
	cleanSrc := paths.CanonicalPath(src)
	cleanBackupDir := paths.CanonicalPath(a.BackupDir) + "/"
	if !strings.HasPrefix(strings.ToLower(cleanSrc), strings.ToLower(cleanBackupDir)) {
		return fmt.Errorf("source path escapes backup directory")
	}

	// Security: validate target path stays under home directory.
	homeDir, err := a.FS.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	cleanTarget := paths.CanonicalPath(d.TargetPath)
	cleanHome := paths.CanonicalPath(homeDir) + "/"
	if !strings.HasPrefix(strings.ToLower(cleanTarget), strings.ToLower(cleanHome)) {
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
