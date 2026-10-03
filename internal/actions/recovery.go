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

const (
	statusPrepared   = "prepared"
	statusApplied    = "applied"
	statusRolledBack = "rolled_back"
	statusFailed     = "failed"
	statusUndone     = "undone"
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

type safeError struct {
	err error
	msg string
}

func (e *safeError) Error() string {
	return e.msg
}

func (e *safeError) Unwrap() error {
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
	if homeDir != "" {
		cleanHome := paths.CanonicalPath(homeDir)
		msg := err.Error()
		if strings.Contains(msg, cleanHome) || strings.Contains(msg, homeDir) {
			cleanMsg := strings.ReplaceAll(msg, cleanHome, "~")
			cleanMsg = strings.ReplaceAll(cleanMsg, homeDir, "~")
			return &safeError{err: err, msg: cleanMsg}
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
			Status:       statusPrepared,
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

	rm.Meta.Status = statusApplied
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
		rm.Meta.Status = statusFailed
	} else {
		rm.Meta.Status = statusRolledBack
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

// errNoAppliedPoint indicates that no applied recovery point was found.
var errNoAppliedPoint = errors.New("no applied recovery point found")

// ErrNoAppliedRecoveryPoint indicates that no applied recovery point was found.
var ErrNoAppliedRecoveryPoint = errNoAppliedPoint

func validateRecoveryBase(storageFS FileSystem, recoveryBase, homeDir string) (string, error) {
	cleanBase := paths.CanonicalPath(recoveryBase)
	cleanHome := paths.CanonicalPath(homeDir)

	if cleanBase == cleanHome || cleanBase == cleanHome+"/" || cleanBase == paths.CanonicalPath(filepath.Join(homeDir, ".bak")) {
		return "", fmt.Errorf("recovery directory cannot be home or .bak root: %s", sanitizePath(recoveryBase, homeDir))
	}
	if !strings.HasPrefix(cleanBase, cleanHome+"/") {
		return "", fmt.Errorf("recovery directory %s escapes home directory", sanitizePath(recoveryBase, homeDir))
	}

	if err := validateNoSymlinkAncestors(storageFS, homeDir, recoveryBase); err != nil {
		return "", fmt.Errorf("validate recovery storage %s: %w", sanitizePath(recoveryBase, homeDir), err)
	}
	return cleanBase, nil
}

func readCandidateAppliedPoint(storageFS FileSystem, cleanBase string, entry os.DirEntry) (*recoveryMetadata, bool) {
	if !entry.IsDir() {
		return nil, false
	}
	pointDir := filepath.Join(cleanBase, entry.Name())
	info, err := storageFS.Lstat(pointDir)
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		return nil, false
	}

	metaFilePath := filepath.Join(pointDir, "repo", "recovery-meta.json")
	metaBytes, err := storageFS.ReadFile(metaFilePath)
	if err != nil {
		metaFilePath = filepath.Join(pointDir, "recovery-meta.json")
		metaBytes, err = storageFS.ReadFile(metaFilePath)
		if err != nil {
			return nil, false
		}
	}

	var meta recoveryMetadata
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, false
	}
	if meta.Status != statusApplied {
		return nil, false
	}
	if meta.PointID == "" {
		meta.PointID = entry.Name()
	} else if meta.PointID != entry.Name() {
		return nil, false
	}
	return &meta, true
}

// findLatestAppliedPoint scans recoveryBase for recovery points with Status == "applied",
// skips corrupt entries, sorts by CreatedAt desc (with PointID desc tiebreak), and returns
// the newest point or a typed none-found error.
func findLatestAppliedPoint(storageFS FileSystem, recoveryBase, homeDir string) (*recoveryMetadata, error) {
	if homeDir == "" {
		return nil, fmt.Errorf("home directory cannot be empty")
	}
	if storageFS == nil {
		storageFS = &OSFileSystem{}
	}
	if recoveryBase == "" {
		recoveryBase = filepath.Join(homeDir, ".bak", "recovery")
	}

	cleanBase, err := validateRecoveryBase(storageFS, recoveryBase, homeDir)
	if err != nil {
		return nil, err
	}

	entries, err := storageFS.ReadDir(cleanBase)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errNoAppliedPoint
		}
		return nil, fmt.Errorf("read recovery dir %s: %w", sanitizePath(cleanBase, homeDir), sanitizeError(err, homeDir))
	}

	var candidates []recoveryMetadata
	for _, entry := range entries {
		if cand, ok := readCandidateAppliedPoint(storageFS, cleanBase, entry); ok {
			candidates = append(candidates, *cand)
		}
	}

	if len(candidates) == 0 {
		return nil, errNoAppliedPoint
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].CreatedAt.Equal(candidates[j].CreatedAt) {
			return candidates[i].PointID > candidates[j].PointID
		}
		return candidates[i].CreatedAt.After(candidates[j].CreatedAt)
	})

	return &candidates[0], nil
}

func (rm *recoveryManager) validateTargetCorrespondence(k string, post targetPostState) (targetPreState, error) {
	cleanPost := paths.CanonicalPath(post.TargetPath)
	if cleanPost != k {
		return targetPreState{}, fmt.Errorf("target correspondence mismatch for %s: post target path %s does not match key", sanitizePath(k, rm.HomeDir), sanitizePath(post.TargetPath, rm.HomeDir))
	}
	pre, ok := rm.Meta.TargetPreMap[k]
	if !ok {
		return targetPreState{}, fmt.Errorf("target correspondence mismatch: missing pre-state for target %s", sanitizePath(post.TargetPath, rm.HomeDir))
	}
	cleanPre := paths.CanonicalPath(pre.TargetPath)
	if cleanPre != k || cleanPre != cleanPost {
		return targetPreState{}, fmt.Errorf("target correspondence mismatch for %s: pre target %s does not match post target %s", sanitizePath(k, rm.HomeDir), sanitizePath(pre.TargetPath, rm.HomeDir), sanitizePath(post.TargetPath, rm.HomeDir))
	}
	if pre.RelPath != post.RelPath {
		return targetPreState{}, fmt.Errorf("target correspondence mismatch for %s: pre rel_path %s does not match post rel_path %s", sanitizePath(k, rm.HomeDir), sanitizePath(pre.RelPath, rm.HomeDir), sanitizePath(post.RelPath, rm.HomeDir))
	}
	return pre, nil
}

func (rm *recoveryManager) verifyLiveTarget(post targetPostState) error {
	info, err := rm.FS.Lstat(post.TargetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if post.Existed {
				return fmt.Errorf("target drift: expected %s to exist, but file is missing", sanitizePath(post.TargetPath, rm.HomeDir))
			}
			return nil
		}
		return fmt.Errorf("inspect target %s: %w", sanitizePath(post.TargetPath, rm.HomeDir), sanitizeError(err, rm.HomeDir))
	}

	if !post.Existed {
		return fmt.Errorf("target drift: expected %s to be absent, but file exists", sanitizePath(post.TargetPath, rm.HomeDir))
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("target drift: refusing symlink target %s", sanitizePath(post.TargetPath, rm.HomeDir))
	}
	if info.IsDir() {
		return fmt.Errorf("target drift: expected regular file at %s, found directory", sanitizePath(post.TargetPath, rm.HomeDir))
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("target drift: expected regular file at %s", sanitizePath(post.TargetPath, rm.HomeDir))
	}
	if !isWindows() && info.Mode().Perm() != post.Mode.Perm() {
		return fmt.Errorf("target drift: mode mismatch for %s (expected %04o, got %04o)", sanitizePath(post.TargetPath, rm.HomeDir), post.Mode.Perm(), info.Mode().Perm())
	}

	data, err := rm.FS.ReadFile(post.TargetPath)
	if err != nil {
		return fmt.Errorf("read target %s: %w", sanitizePath(post.TargetPath, rm.HomeDir), sanitizeError(err, rm.HomeDir))
	}
	h := sha256.Sum256(data)
	actualHash := fmt.Sprintf("sha256:%x", h)
	if actualHash != post.SHA256 {
		return fmt.Errorf("target drift: content mismatch for %s (expected %s, got %s)", sanitizePath(post.TargetPath, rm.HomeDir), post.SHA256, actualHash)
	}
	return nil
}

func (rm *recoveryManager) verifyPreSnapshot(pre targetPreState) error {
	if !pre.Existed {
		return nil
	}
	repoPrePath := filepath.Join(rm.RepoDir, "snapshots", "pre", opaquePayloadName(pre.RelPath))
	preData, err := rm.StorageFS.ReadFile(repoPrePath)
	if err != nil {
		return fmt.Errorf("read pre-snapshot for %s: %w", sanitizePath(pre.RelPath, rm.HomeDir), sanitizeError(err, rm.HomeDir))
	}
	preH := sha256.Sum256(preData)
	actualPreHash := fmt.Sprintf("sha256:%x", preH)
	if actualPreHash != pre.SHA256 {
		return fmt.Errorf("pre-snapshot checksum mismatch for %s (tampered or corrupted)", sanitizePath(pre.RelPath, rm.HomeDir))
	}
	return nil
}

func (rm *recoveryManager) validatePreMapCorrespondence() error {
	for preKey, pre := range rm.Meta.TargetPreMap {
		cleanPre := paths.CanonicalPath(pre.TargetPath)
		if cleanPre != preKey {
			return fmt.Errorf("target correspondence mismatch for %s: pre target path %s does not match key", sanitizePath(preKey, rm.HomeDir), sanitizePath(pre.TargetPath, rm.HomeDir))
		}
		if _, ok := rm.Meta.TargetPostMap[preKey]; !ok {
			return fmt.Errorf("target correspondence mismatch: missing post-state for target %s", sanitizePath(pre.TargetPath, rm.HomeDir))
		}
	}
	return nil
}

func (rm *recoveryManager) validateDriftPrerequisites() error {
	if rm.HomeDir == "" {
		return fmt.Errorf("home directory cannot be empty")
	}
	if rm.FS == nil {
		return fmt.Errorf("file system cannot be nil")
	}
	if rm.StorageFS == nil {
		rm.StorageFS = &OSFileSystem{}
	}
	if rm.Meta.Status != statusApplied {
		return fmt.Errorf("recovery point status is %s, expected applied", rm.Meta.Status)
	}
	if len(rm.Meta.TargetPostMap) == 0 {
		return fmt.Errorf("no post-restore targets recorded")
	}
	if len(rm.Meta.TargetPostMap) != len(rm.Meta.TargetPreMap) {
		return fmt.Errorf("target correspondence mismatch: post map has %d entries, pre map has %d", len(rm.Meta.TargetPostMap), len(rm.Meta.TargetPreMap))
	}
	return nil
}

func (rm *recoveryManager) initPointAndRepoDirs() {
	if rm.PointDir != "" || rm.PointID == "" {
		return
	}
	recoveryBase := rm.RecoveryDir
	if recoveryBase == "" {
		recoveryBase = filepath.Join(rm.HomeDir, ".bak", "recovery")
	}
	rm.PointDir = filepath.Join(recoveryBase, rm.PointID)
	rm.RepoDir = filepath.Join(rm.PointDir, "repo")
}

func (rm *recoveryManager) verifyTargetDrift(k string) error {
	post := rm.Meta.TargetPostMap[k]
	if err := validateNoSymlinkAncestors(rm.FS, rm.HomeDir, post.TargetPath); err != nil {
		return fmt.Errorf("validate target %s: %w", sanitizePath(post.TargetPath, rm.HomeDir), err)
	}
	pre, err := rm.validateTargetCorrespondence(k, post)
	if err != nil {
		return err
	}
	if err := rm.verifyLiveTarget(post); err != nil {
		return err
	}
	return rm.verifyPreSnapshot(pre)
}

// CheckUndoDrift verifies every TargetPostMap entry against the live filesystem BEFORE
// any write is performed: symlink-ancestor validation, absence/existence match, regular-file
// non-symlink, mode Perm equality (non-Windows), SHA-256 content equality, and pre-snapshot
// presence + hash verification. Any mismatch aborts with an error and zero writes.
func (rm *recoveryManager) CheckUndoDrift() error {
	if err := rm.validateDriftPrerequisites(); err != nil {
		return err
	}
	rm.initPointAndRepoDirs()

	targetKeys := make([]string, 0, len(rm.Meta.TargetPostMap))
	for k := range rm.Meta.TargetPostMap {
		targetKeys = append(targetKeys, k)
	}
	sort.Strings(targetKeys)

	for _, k := range targetKeys {
		if err := rm.verifyTargetDrift(k); err != nil {
			return err
		}
	}

	return rm.validatePreMapCorrespondence()
}

func (rm *recoveryManager) persistFailedMetadata(outcome *rollbackOutcome) {
	rm.Meta.Status = statusFailed
	metaBytes, err := json.MarshalIndent(rm.Meta, "", "  ")
	if err != nil {
		outcome.Errors = append(outcome.Errors, fmt.Errorf("marshal recovery meta: %w", err))
		return
	}
	metaFilePath := filepath.Join(rm.RepoDir, "recovery-meta.json")
	if writeErr := rm.StorageFS.WriteFile(metaFilePath, metaBytes, 0600); writeErr != nil {
		outcome.Errors = append(outcome.Errors, fmt.Errorf("write failed recovery metadata: %w", sanitizeError(writeErr, rm.HomeDir)))
	}
}

func (rm *recoveryManager) commitOrPersistUndoOutcome(outcome *rollbackOutcome) {
	if len(outcome.Unresolved) > 0 {
		rm.Meta.Status = statusFailed
	} else {
		rm.Meta.Status = statusUndone
	}

	gitRepo, err := git.OpenRepo(rm.RepoDir)
	if err != nil {
		outcome.Errors = append(outcome.Errors, fmt.Errorf("open recovery git repo: %w", sanitizeError(err, rm.HomeDir)))
		rm.persistFailedMetadata(outcome)
		return
	}

	commitMsg := "undo target snapshot"
	if rm.Meta.Status == statusFailed {
		commitMsg = "partial undo target failure"
	}
	if commitErr := rm.commitMetadataAndSnapshot(gitRepo, commitMsg); commitErr != nil {
		outcome.Errors = append(outcome.Errors, fmt.Errorf("commit undo metadata: %w", commitErr))
		rm.persistFailedMetadata(outcome)
	}
}

// UndoTargets projects TargetPreMap back to live files (bytes, mode including zero, absence
// via single-file remove refusing directories), hash-verifies pre snapshots, updates metadata
// Status to "undone" and commits the metadata update in the recovery repo. On partial failure,
// Status is set to "failed", Reverted/Unresolved accounting is preserved, and an error is returned.
func (rm *recoveryManager) UndoTargets() (rollbackOutcome, error) {
	outcome := rollbackOutcome{
		PointID: rm.PointID,
	}

	rm.initPointAndRepoDirs()
	if rm.StorageFS == nil {
		rm.StorageFS = &OSFileSystem{}
	}

	targets := make([]string, 0, len(rm.Meta.TargetPreMap))
	for target := range rm.Meta.TargetPreMap {
		targets = append(targets, target)
	}
	sort.Strings(targets)

	for _, target := range targets {
		pre, ok := rm.Meta.TargetPreMap[target]
		if !ok {
			outcome.Unresolved = append(outcome.Unresolved, target)
			outcome.Errors = append(outcome.Errors, fmt.Errorf("target correspondence mismatch: no pre-state for %s", sanitizePath(target, rm.HomeDir)))
			continue
		}
		cleanPreTarget := paths.CanonicalPath(pre.TargetPath)
		if cleanPreTarget != target {
			outcome.Unresolved = append(outcome.Unresolved, pre.TargetPath)
			outcome.Errors = append(outcome.Errors, fmt.Errorf("target correspondence mismatch for %s: pre path %s", sanitizePath(target, rm.HomeDir), sanitizePath(pre.TargetPath, rm.HomeDir)))
			continue
		}
		post, ok := rm.Meta.TargetPostMap[target]
		if !ok || paths.CanonicalPath(post.TargetPath) != target {
			outcome.Unresolved = append(outcome.Unresolved, pre.TargetPath)
			outcome.Errors = append(outcome.Errors, fmt.Errorf("target correspondence mismatch: no matching post-state for %s", sanitizePath(target, rm.HomeDir)))
			continue
		}

		if err := rm.rollbackTarget(pre); err != nil {
			outcome.Unresolved = append(outcome.Unresolved, pre.TargetPath)
			outcome.Errors = append(outcome.Errors, err)
		} else {
			outcome.Reverted = append(outcome.Reverted, pre.TargetPath)
		}
	}
	sort.Strings(outcome.Reverted)
	sort.Strings(outcome.Unresolved)

	rm.commitOrPersistUndoOutcome(&outcome)

	if len(outcome.Unresolved) > 0 || len(outcome.Errors) > 0 {
		errMsgs := make([]string, 0, len(outcome.Errors))
		for _, e := range outcome.Errors {
			errMsgs = append(errMsgs, e.Error())
		}
		return outcome, fmt.Errorf("undo targets failed: %s", strings.Join(errMsgs, "; "))
	}

	return outcome, nil
}
