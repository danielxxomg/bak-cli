package actions

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/danielxxomg/bak-cli/internal/git"
	"github.com/danielxxomg/bak-cli/internal/paths"
	restorepkg "github.com/danielxxomg/bak-cli/internal/restore"
	gogit "github.com/go-git/go-git/v5"
)

// targetPreState stores original state of a target file prior to any restore write.
type targetPreState struct {
	TargetPath string      `json:"target_path"`
	RelPath    string      `json:"rel_path"`
	Existed    bool        `json:"existed"`
	Mode       os.FileMode `json:"mode"`
	SHA256     string      `json:"sha256,omitempty"`
}

// targetPostState stores actual target state after restore mutation.
type targetPostState struct {
	TargetPath string      `json:"target_path"`
	RelPath    string      `json:"rel_path"`
	Existed    bool        `json:"existed"`
	Mode       os.FileMode `json:"mode"`
	SHA256     string      `json:"sha256,omitempty"`
}

// recoveryMetadata documents the recovery point.
type recoveryMetadata struct {
	PointID       string                     `json:"point_id"`
	BackupID      string                     `json:"backup_id"`
	CreatedAt     time.Time                  `json:"created_at"`
	TargetPreMap  map[string]targetPreState  `json:"target_pre_map"`
	TargetPostMap map[string]targetPostState `json:"target_post_map,omitempty"`
	Status        string                     `json:"status"` // prepared, applied, rolled_back, failed
}

// rollbackOutcome captures detailed results of a rollback attempt.
type rollbackOutcome struct {
	PointID    string
	Reverted   []string
	Unresolved []string
	Errors     []error
}

// recoveryManager manages private local recovery snapshots under ~/.bak/recovery/<point-id>.
type recoveryManager struct {
	FS          FileSystem // file system for targets
	StorageFS   FileSystem // file system for recovery repository (defaults to OSFileSystem)
	HomeDir     string
	BackupID    string
	PointID     string
	PointDir    string
	RepoDir     string
	RecoveryDir string // optional base recovery directory override
	Meta        recoveryMetadata
}

// sanitizePath replaces home directory prefix with ~ to prevent username leakage.
func sanitizePath(p, homeDir string) string {
	cleanP := paths.CanonicalPath(p)
	cleanHome := paths.CanonicalPath(homeDir)
	if cleanP == cleanHome {
		return "~"
	}
	if strings.HasPrefix(cleanP, cleanHome+"/") {
		return "~/" + strings.TrimPrefix(cleanP, cleanHome+"/")
	}
	return filepath.Base(cleanP)
}

type safePathError struct {
	op   string
	path string
	err  error
}

func (e *safePathError) Error() string {
	return fmt.Sprintf("%s %s: %v", e.op, e.path, e.err)
}

func (e *safePathError) Unwrap() error {
	return e.err
}

func sanitizeError(err error, homeDir string) error {
	if err == nil {
		return nil
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return &safePathError{
			op:   pathErr.Op,
			path: sanitizePath(pathErr.Path, homeDir),
			err:  sanitizeError(pathErr.Err, homeDir),
		}
	}
	return err
}

// generateRecoveryID generates an opaque, collision-resistant recovery ID.
func generateRecoveryID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate recovery id: %w", err)
	}
	return fmt.Sprintf("rec-%s-%s", time.Now().UTC().Format("20060102T150405Z"), hex.EncodeToString(b)), nil
}

// validateNoSymlinkAncestors verifies that path and all its ancestor components are not symlinks.
func validateNoSymlinkAncestors(fs FileSystem, homeDir, targetPath string) error {
	cleanTarget := paths.CanonicalPath(targetPath)
	cleanHome := paths.CanonicalPath(homeDir)

	if cleanTarget != cleanHome && !strings.HasPrefix(cleanTarget, cleanHome+"/") {
		return fmt.Errorf("target path %s escapes home directory", sanitizePath(targetPath, homeDir))
	}

	rel, err := filepath.Rel(cleanHome, cleanTarget)
	if err != nil {
		return fmt.Errorf("relative path: %w", err)
	}

	parts := strings.Split(rel, string(filepath.Separator))
	curr := cleanHome
	for _, p := range parts {
		if p == "" || p == "." {
			continue
		}
		curr = filepath.Join(curr, p)
		info, err := fs.Lstat(curr)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				break
			}
			return fmt.Errorf("lstat %s: %w", sanitizePath(curr, homeDir), sanitizeError(err, homeDir))
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink detected in path component: %s", sanitizePath(curr, homeDir))
		}
	}
	return nil
}

func (rm *recoveryManager) initDefaults() error {
	if rm.HomeDir == "" {
		return fmt.Errorf("home directory cannot be empty")
	}
	if rm.StorageFS == nil {
		rm.StorageFS = &OSFileSystem{}
	}
	if rm.PointID == "" {
		id, err := generateRecoveryID()
		if err != nil {
			return err
		}
		rm.PointID = id
	}

	recoveryBase := rm.RecoveryDir
	if recoveryBase == "" {
		recoveryBase = filepath.Join(rm.HomeDir, ".bak", "recovery")
	}

	cleanBase := paths.CanonicalPath(recoveryBase)
	cleanHome := paths.CanonicalPath(rm.HomeDir)

	if cleanBase == cleanHome || cleanBase == cleanHome+"/" || cleanBase == paths.CanonicalPath(filepath.Join(rm.HomeDir, ".bak")) {
		return fmt.Errorf("recovery directory cannot be home or .bak root: %s", sanitizePath(recoveryBase, rm.HomeDir))
	}
	if !strings.HasPrefix(cleanBase, cleanHome+"/") {
		return fmt.Errorf("recovery directory %s escapes home directory", sanitizePath(recoveryBase, rm.HomeDir))
	}

	if err := validateNoSymlinkAncestors(rm.StorageFS, rm.HomeDir, recoveryBase); err != nil {
		return fmt.Errorf("validate recovery storage %s: %w", sanitizePath(recoveryBase, rm.HomeDir), err)
	}

	rm.PointDir = filepath.Join(cleanBase, rm.PointID)
	rm.RepoDir = filepath.Join(rm.PointDir, "repo")

	if rm.Meta.TargetPreMap == nil {
		rm.Meta = recoveryMetadata{
			PointID:      rm.PointID,
			BackupID:     rm.BackupID,
			CreatedAt:    time.Now().UTC(),
			TargetPreMap: make(map[string]targetPreState),
			Status:       "prepared",
		}
	}
	return nil
}

// opaquePayloadName returns a deterministic flat filename derived from the SHA-256
// hash of the canonical relative path, preventing control name conflicts (such as
// nested .git directories or captured .gitignore files).
func opaquePayloadName(relSlash string) string {
	cleanRel := path.Clean(strings.ReplaceAll(relSlash, "\\", "/"))
	h := sha256.Sum256([]byte(cleanRel))
	return fmt.Sprintf("%x", h)
}

// capturePreTarget captures an individual target's pre-state into the recovery repository.
func (rm *recoveryManager) capturePreTarget(d restorepkg.FileDiff, cleanHome string) error {
	cleanTarget := paths.CanonicalPath(d.TargetPath)
	rel, err := filepath.Rel(cleanHome, cleanTarget)
	if err != nil {
		return fmt.Errorf("rel target path: %w", err)
	}
	relSlash := path.Clean(strings.ReplaceAll(rel, "\\", "/"))
	if strings.HasPrefix(relSlash, "../") || relSlash == ".." {
		return fmt.Errorf("target path escapes home directory: %s", sanitizePath(d.TargetPath, rm.HomeDir))
	}
	repoDestPath := filepath.Join(rm.RepoDir, "snapshots", "pre", opaquePayloadName(relSlash))

	preState := targetPreState{
		TargetPath: d.TargetPath,
		RelPath:    relSlash,
	}

	info, err := rm.FS.Lstat(d.TargetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			preState.Existed = false
			rm.Meta.TargetPreMap[cleanTarget] = preState
			return nil
		}
		return fmt.Errorf("lstat target %s: %w", sanitizePath(d.TargetPath, rm.HomeDir), sanitizeError(err, rm.HomeDir))
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing symlink target: %s", sanitizePath(d.TargetPath, rm.HomeDir))
	}
	preState.Existed = true
	preState.Mode = info.Mode().Perm()

	data, err := rm.FS.ReadFile(d.TargetPath)
	if err != nil {
		return fmt.Errorf("read target %s: %w", sanitizePath(d.TargetPath, rm.HomeDir), sanitizeError(err, rm.HomeDir))
	}
	h := sha256.Sum256(data)
	preState.SHA256 = fmt.Sprintf("sha256:%x", h)

	if err := rm.StorageFS.MkdirAll(filepath.Dir(repoDestPath), 0700); err != nil {
		return fmt.Errorf("mkdir recovery repo dest: %w", sanitizeError(err, rm.HomeDir))
	}
	if err := rm.StorageFS.WriteFile(repoDestPath, data, 0600); err != nil {
		return fmt.Errorf("write recovery repo file: %w", sanitizeError(err, rm.HomeDir))
	}

	rm.Meta.TargetPreMap[cleanTarget] = preState
	return nil
}

// filterAffectedTargets verifies targets and filters down to DiffNew / DiffModified diffs.
func (rm *recoveryManager) filterAffectedTargets(diffs []restorepkg.FileDiff) ([]restorepkg.FileDiff, error) {
	var affected []restorepkg.FileDiff
	seenTargets := make(map[string]bool)
	cleanRecoveryBase := paths.CanonicalPath(filepath.Dir(rm.PointDir))
	cleanPointDir := paths.CanonicalPath(rm.PointDir)

	for _, d := range diffs {
		if d.Status != restorepkg.DiffNew && d.Status != restorepkg.DiffModified {
			continue
		}
		cleanTarget := paths.CanonicalPath(d.TargetPath)
		lowerTarget := strings.ToLower(cleanTarget)
		if seenTargets[lowerTarget] {
			return nil, fmt.Errorf("duplicate target path in restore diffs: %s", sanitizePath(d.TargetPath, rm.HomeDir))
		}
		seenTargets[lowerTarget] = true

		lowerBase := strings.ToLower(cleanRecoveryBase)
		lowerPoint := strings.ToLower(cleanPointDir)
		if lowerTarget == lowerBase || strings.HasPrefix(lowerTarget, lowerBase+"/") ||
			lowerTarget == lowerPoint || strings.HasPrefix(lowerTarget, lowerPoint+"/") {
			return nil, fmt.Errorf("target path overlaps recovery storage: %s", sanitizePath(d.TargetPath, rm.HomeDir))
		}

		if err := validateNoSymlinkAncestors(rm.FS, rm.HomeDir, d.TargetPath); err != nil {
			return nil, fmt.Errorf("validate target path %s: %w", sanitizePath(d.TargetPath, rm.HomeDir), err)
		}

		affected = append(affected, d)
	}
	return affected, nil
}

func (rm *recoveryManager) initRepoDirectories() (*gogit.Repository, error) {
	if err := rm.StorageFS.MkdirAll(rm.RepoDir, 0700); err != nil {
		return nil, fmt.Errorf("create recovery repo dir: %w", sanitizeError(err, rm.HomeDir))
	}
	if err := rm.StorageFS.Chmod(rm.PointDir, 0700); err != nil && !isWindows() {
		return nil, fmt.Errorf("chmod recovery point dir: %w", sanitizeError(err, rm.HomeDir))
	}
	if err := rm.StorageFS.Chmod(rm.RepoDir, 0700); err != nil && !isWindows() {
		return nil, fmt.Errorf("chmod recovery repo dir: %w", sanitizeError(err, rm.HomeDir))
	}
	gitRepo, err := git.InitRepo(rm.RepoDir)
	if err != nil {
		return nil, fmt.Errorf("init recovery git repo: %w", sanitizeError(err, rm.HomeDir))
	}
	return gitRepo, nil
}

func (rm *recoveryManager) commitMetadataAndSnapshot(gitRepo *gogit.Repository, message string) error {
	manifestBytes, err := json.MarshalIndent(rm.Meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal recovery meta: %w", err)
	}
	metaFilePath := filepath.Join(rm.RepoDir, "recovery-meta.json")
	if err := rm.StorageFS.WriteFile(metaFilePath, manifestBytes, 0600); err != nil {
		return fmt.Errorf("write recovery meta file: %w", sanitizeError(err, rm.HomeDir))
	}
	if err := git.StageAll(gitRepo); err != nil {
		return fmt.Errorf("stage recovery files: %w", sanitizeError(err, rm.HomeDir))
	}
	if err := git.Commit(gitRepo, message); err != nil {
		return fmt.Errorf("commit %s: %w", message, sanitizeError(err, rm.HomeDir))
	}
	return nil
}

// Prepare captures pre-state for all affected targets (DiffNew and DiffModified)
// into private recovery repository and commits it.
func (rm *recoveryManager) Prepare(diffs []restorepkg.FileDiff) error {
	if err := rm.initDefaults(); err != nil {
		return err
	}

	affected, err := rm.filterAffectedTargets(diffs)
	if err != nil {
		return err
	}
	if len(affected) == 0 {
		return nil
	}

	gitRepo, err := rm.initRepoDirectories()
	if err != nil {
		return err
	}

	cleanHome := paths.CanonicalPath(rm.HomeDir)
	for _, d := range affected {
		if err := rm.capturePreTarget(d, cleanHome); err != nil {
			return err
		}
	}

	return rm.commitMetadataAndSnapshot(gitRepo, "pre-restore snapshot")
}

// capturePostTarget captures an individual target's post-state into the recovery repository.
func (rm *recoveryManager) capturePostTarget(pre targetPreState) (targetPostState, error) {
	post := targetPostState{
		TargetPath: pre.TargetPath,
		RelPath:    pre.RelPath,
	}
	info, err := rm.FS.Lstat(pre.TargetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			post.Existed = false
			return post, nil
		}
		return post, fmt.Errorf("lstat post-restore %s: %w", sanitizePath(pre.TargetPath, rm.HomeDir), sanitizeError(err, rm.HomeDir))
	}

	post.Existed = true
	post.Mode = info.Mode().Perm()
	repoDestPath := filepath.Join(rm.RepoDir, "snapshots", "post", opaquePayloadName(pre.RelPath))
	data, err := rm.FS.ReadFile(pre.TargetPath)
	if err != nil {
		return post, fmt.Errorf("read post-restore %s: %w", sanitizePath(pre.TargetPath, rm.HomeDir), sanitizeError(err, rm.HomeDir))
	}
	h := sha256.Sum256(data)
	post.SHA256 = fmt.Sprintf("sha256:%x", h)

	if err := rm.StorageFS.MkdirAll(filepath.Dir(repoDestPath), 0700); err != nil {
		return post, fmt.Errorf("mkdir post recovery dest: %w", sanitizeError(err, rm.HomeDir))
	}
	if err := rm.StorageFS.WriteFile(repoDestPath, data, 0600); err != nil {
		return post, fmt.Errorf("write post recovery file: %w", sanitizeError(err, rm.HomeDir))
	}

	return post, nil
}

// RecordPostState captures the final state of target files after successful apply and commits.
func (rm *recoveryManager) RecordPostState() error {
	if len(rm.Meta.TargetPreMap) == 0 {
		return nil
	}

	gitRepo, err := git.OpenRepo(rm.RepoDir)
	if err != nil {
		return fmt.Errorf("open recovery git repo: %w", err)
	}

	rm.Meta.TargetPostMap = make(map[string]targetPostState)
	for targetPath, pre := range rm.Meta.TargetPreMap {
		post, err := rm.capturePostTarget(pre)
		if err != nil {
			return err
		}
		rm.Meta.TargetPostMap[targetPath] = post
	}

	rm.Meta.Status = "applied"
	return rm.commitMetadataAndSnapshot(gitRepo, "post-restore snapshot")
}

func (rm *recoveryManager) rollbackNewTarget(pre targetPreState) error {
	info, err := rm.FS.Lstat(pre.TargetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("inspect new target %s: %w", sanitizePath(pre.TargetPath, rm.HomeDir), sanitizeError(err, rm.HomeDir))
	}
	if info.IsDir() {
		return fmt.Errorf("refusing to remove unexpected directory at target %s", sanitizePath(pre.TargetPath, rm.HomeDir))
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("refusing to remove non-regular file at target %s", sanitizePath(pre.TargetPath, rm.HomeDir))
	}
	if remErr := rm.FS.Remove(pre.TargetPath); remErr != nil {
		return fmt.Errorf("remove new target %s: %w", sanitizePath(pre.TargetPath, rm.HomeDir), sanitizeError(remErr, rm.HomeDir))
	}
	return nil
}

func (rm *recoveryManager) rollbackExistingTarget(pre targetPreState) error {
	repoSrcPath := filepath.Join(rm.RepoDir, "snapshots", "pre", opaquePayloadName(pre.RelPath))
	data, err := rm.StorageFS.ReadFile(repoSrcPath)
	if err != nil {
		return fmt.Errorf("read pre-state file %s: %w", sanitizePath(pre.RelPath, rm.HomeDir), sanitizeError(err, rm.HomeDir))
	}

	h := sha256.Sum256(data)
	if fmt.Sprintf("sha256:%x", h) != pre.SHA256 {
		return fmt.Errorf("pre-state file %s checksum mismatch (tampered or corrupted)", sanitizePath(pre.RelPath, rm.HomeDir))
	}

	if err := rm.FS.MkdirAll(filepath.Dir(pre.TargetPath), 0755); err != nil {
		return fmt.Errorf("mkdir target dir %s: %w", sanitizePath(pre.TargetPath, rm.HomeDir), sanitizeError(err, rm.HomeDir))
	}

	if err := rm.FS.WriteFile(pre.TargetPath, data, pre.Mode); err != nil {
		return fmt.Errorf("restore target %s: %w", sanitizePath(pre.TargetPath, rm.HomeDir), sanitizeError(err, rm.HomeDir))
	}

	if err := rm.FS.Chmod(pre.TargetPath, pre.Mode); err != nil && !isWindows() {
		return fmt.Errorf("chmod target %s: %w", sanitizePath(pre.TargetPath, rm.HomeDir), sanitizeError(err, rm.HomeDir))
	}
	return nil
}

// rollbackTarget restores a single target file to its pre-state.
func (rm *recoveryManager) rollbackTarget(pre targetPreState) error {
	if err := validateNoSymlinkAncestors(rm.FS, rm.HomeDir, pre.TargetPath); err != nil {
		return fmt.Errorf("validate target path %s: %w", sanitizePath(pre.TargetPath, rm.HomeDir), err)
	}

	if !pre.Existed {
		return rm.rollbackNewTarget(pre)
	}
	return rm.rollbackExistingTarget(pre)
}

// Rollback restores every attempted target to its pre-restore state.
func (rm *recoveryManager) Rollback(attemptedTargets []string) rollbackOutcome {
	outcome := rollbackOutcome{
		PointID: rm.PointID,
	}

	seen := make(map[string]bool)
	var targets []string
	for _, t := range attemptedTargets {
		cleanT := paths.CanonicalPath(t)
		if !seen[cleanT] {
			seen[cleanT] = true
			targets = append(targets, cleanT)
		}
	}

	for _, target := range targets {
		pre, ok := rm.Meta.TargetPreMap[target]
		if !ok {
			outcome.Unresolved = append(outcome.Unresolved, target)
			outcome.Errors = append(outcome.Errors, fmt.Errorf("no pre-state recorded for %s", sanitizePath(target, rm.HomeDir)))
			continue
		}

		if err := rm.rollbackTarget(pre); err != nil {
			outcome.Unresolved = append(outcome.Unresolved, pre.TargetPath)
			outcome.Errors = append(outcome.Errors, err)
			continue
		}
		outcome.Reverted = append(outcome.Reverted, pre.TargetPath)
	}

	if len(outcome.Unresolved) > 0 {
		rm.Meta.Status = "failed"
	} else {
		rm.Meta.Status = "rolled_back"
	}

	metaBytes, err := json.MarshalIndent(rm.Meta, "", "  ")
	if err == nil {
		metaFilePath := filepath.Join(rm.RepoDir, "recovery-meta.json")
		if writeErr := rm.StorageFS.WriteFile(metaFilePath, metaBytes, 0600); writeErr != nil {
			outcome.Errors = append(outcome.Errors, fmt.Errorf("write rollback metadata: %w", sanitizeError(writeErr, rm.HomeDir)))
		}
	} else {
		outcome.Errors = append(outcome.Errors, fmt.Errorf("marshal rollback metadata: %w", err))
	}

	sort.Strings(outcome.Reverted)
	sort.Strings(outcome.Unresolved)
	return outcome
}
