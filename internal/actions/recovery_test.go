package actions

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/danielxxomg/bak-cli/internal/git"
	"github.com/danielxxomg/bak-cli/internal/manifest"
	"github.com/danielxxomg/bak-cli/internal/paths"
	restorepkg "github.com/danielxxomg/bak-cli/internal/restore"
)

// setupRecoveryTestEnv prepares home and backupDir with manifest and items.
func setupRecoveryTestEnv(t *testing.T, backupID string, files map[string]string) (string, string) {
	t.Helper()
	home := t.TempDir()
	bakDir := filepath.Join(home, ".bak")
	backupsDir := filepath.Join(bakDir, "backups")
	backupDir := filepath.Join(backupsDir, backupID)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	adapterDir := filepath.Join(backupDir, "test-adapter")
	if err := os.MkdirAll(adapterDir, 0755); err != nil {
		t.Fatal(err)
	}

	m := manifest.New(backupID, "linux", "testhost", "0.4.0", "quick", []string{"config"})
	m.Version = "0.4.0"
	items := make([]manifest.Item, 0, len(files))

	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, relName := range keys {
		content := files[relName]
		backedPath := filepath.Join(adapterDir, relName)
		if err := os.MkdirAll(filepath.Dir(backedPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(backedPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256([]byte(content))
		items = append(items, manifest.Item{
			Category:   "config",
			SourcePath: fmt.Sprintf("~/.config/bak/%s", relName),
			BackupPath: fmt.Sprintf("test-adapter/%s", relName),
			Hash:       fmt.Sprintf("sha256:%x", h),
			Size:       int64(len(content)),
			Mode:       0644,
		})
	}

	m.AddAdapter("test-adapter", "", "~/.config/bak", items)
	if err := m.Save(backupDir); err != nil {
		t.Fatal(err)
	}
	return home, backupDir
}

func TestRecoveryManager_DryRunNoSideEffects(t *testing.T) { //nolint:paralleltest // shared state
	home, backupDir := setupRecoveryTestEnv(t, "20260101-dryrun", map[string]string{
		"a.txt": "content-a\n",
	})

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		DryRun:    true,
		Stdout:    io.Discard,
		Stderr:    io.Discard,
	}

	if err := action.Run(); err != nil {
		t.Fatalf("dry run failed: %v", err)
	}

	// Assert no recovery directory was created.
	recDir := filepath.Join(home, ".bak", "recovery")
	if _, err := os.Stat(recDir); err == nil {
		t.Errorf("recovery directory should not exist on dry-run, found %s", recDir)
	}
}

func TestRecoveryManager_UserCancelNoSideEffects(t *testing.T) { //nolint:paralleltest // shared state
	home, backupDir := setupRecoveryTestEnv(t, "20260101-cancel", map[string]string{
		"a.txt": "content-a\n",
	})

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		Force:     false,
		Stdin:     strings.NewReader("n\n"),
		Stdout:    io.Discard,
		Stderr:    io.Discard,
	}

	if err := action.Run(); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	recDir := filepath.Join(home, ".bak", "recovery")
	if _, err := os.Stat(recDir); err == nil {
		t.Errorf("recovery directory should not exist on cancel, found %s", recDir)
	}
}

func TestRecoveryManager_UnchangedNoSideEffects(t *testing.T) { //nolint:paralleltest // shared state
	home, backupDir := setupRecoveryTestEnv(t, "20260101-unchanged", map[string]string{
		"a.txt": "same-content\n",
	})

	// Create target with exact same content.
	targetPath := filepath.Join(home, ".config", "bak", "a.txt")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, []byte("same-content\n"), 0644); err != nil {
		t.Fatal(err)
	}

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		Force:     true,
		Stdout:    io.Discard,
		Stderr:    io.Discard,
	}

	if err := action.Run(); err != nil {
		t.Fatalf("run failed: %v", err)
	}

	recDir := filepath.Join(home, ".bak", "recovery")
	if _, err := os.Stat(recDir); err == nil {
		t.Errorf("recovery directory should not exist when all diffs are unchanged, found %s", recDir)
	}
}

func TestRecoveryManager_RollbackRemovesNewlyCreatedTargets(t *testing.T) { //nolint:paralleltest // shared state
	home, backupDir := setupRecoveryTestEnv(t, "20260101-newtargets", map[string]string{
		"f1.txt": "new1\n",
		"f2.txt": "new2\n",
	})

	targetDir := filepath.Join(home, ".config", "bak")
	t1 := filepath.Join(targetDir, "f1.txt")
	t2 := filepath.Join(targetDir, "f2.txt")

	baseFS := newHomeFS(home)
	failingFS := &partialCopyFailingFS{
		FileSystem: baseFS,
		failOnDst:  t2,
		failErr:    fmt.Errorf("disk full writing t2"),
	}

	action := &RestoreAction{
		FS:        failingFS,
		BackupDir: backupDir,
		Force:     true,
		Stdout:    io.Discard,
		Stderr:    io.Discard,
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Since neither t1 nor t2 existed before restore, rollback must remove them.
	if _, err := os.Stat(t1); err == nil {
		t.Errorf("new target t1 should be removed by rollback, but still exists")
	}
	if _, err := os.Stat(t2); err == nil {
		t.Errorf("new target t2 should be removed by rollback, but still exists")
	}
}

func TestRecoveryManager_SymlinkTargetRefused(t *testing.T) { //nolint:paralleltest // shared state
	if isWindows() {
		t.Skip("skipping symlink test on Windows")
	}
	home, backupDir := setupRecoveryTestEnv(t, "20260101-symlink", map[string]string{
		"link.txt": "linkcontent\n",
	})

	targetDir := filepath.Join(home, ".config", "bak")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	outsideFile := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("outside"), 0644); err != nil {
		t.Fatal(err)
	}
	symlinkPath := filepath.Join(targetDir, "link.txt")
	if err := os.Symlink(outsideFile, symlinkPath); err != nil {
		t.Fatal(err)
	}

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		Force:     true,
		Stdout:    io.Discard,
		Stderr:    io.Discard,
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error when target is a symlink, got nil")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Errorf("error %q should mention symlink", err.Error())
	}
}

func TestRecoveryManager_SuccessfulRestoreGitEvidence(t *testing.T) { //nolint:paralleltest // shared state
	home, backupDir := setupRecoveryTestEnv(t, "20260101-success", map[string]string{
		"config.json": "updated-config\n",
	})

	targetPath := filepath.Join(home, ".config", "bak", "config.json")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, []byte("old-config\n"), 0644); err != nil {
		t.Fatal(err)
	}

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		Force:     true,
		Stdout:    io.Discard,
		Stderr:    io.Discard,
	}

	if err := action.Run(); err != nil {
		t.Fatalf("expected successful restore, got: %v", err)
	}

	recBase := filepath.Join(home, ".bak", "recovery")
	entries, err := os.ReadDir(recBase)
	if err != nil || len(entries) == 0 {
		t.Fatalf("expected recovery point directory, got err: %v, entries: %d", err, len(entries))
	}

	pointDir := filepath.Join(recBase, entries[0].Name())
	repoDir := filepath.Join(pointDir, "repo")
	if !git.IsRepo(repoDir) {
		t.Errorf("expected valid git repo at %s", repoDir)
	}

	// Verify post-restore snapshot commit exists.
	repo, err := git.OpenRepo(repoDir)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	head, err := repo.Head()
	if err != nil {
		t.Fatalf("get head: %v", err)
	}
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		t.Fatalf("get commit: %v", err)
	}
	if commit.Message != "post-restore snapshot" {
		t.Errorf("commit message = %q, want 'post-restore snapshot'", commit.Message)
	}
}

// targetStatFailingFS wraps FileSystem and fails Lstat on a target.
type targetStatFailingFS struct {
	FileSystem
	failPath string
}

func (f *targetStatFailingFS) Lstat(path string) (os.FileInfo, error) {
	if path == f.failPath {
		return nil, os.ErrPermission
	}
	return f.FileSystem.Lstat(path)
}

func TestRecoveryManager_PrepareFailureAbortsBeforeWrites(t *testing.T) { //nolint:paralleltest // shared state
	home, backupDir := setupRecoveryTestEnv(t, "20260101-prep-fail", map[string]string{
		"target.txt": "new-data\n",
	})

	targetDir := filepath.Join(home, ".config", "bak")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	targetFile := filepath.Join(targetDir, "target.txt")
	if err := os.WriteFile(targetFile, []byte("original-data\n"), 0644); err != nil {
		t.Fatal(err)
	}

	baseFS := newHomeFS(home)
	failingFS := &targetStatFailingFS{
		FileSystem: baseFS,
		failPath:   targetFile,
	}

	action := &RestoreAction{
		FS:        failingFS,
		BackupDir: backupDir,
		Force:     true,
		Stdout:    io.Discard,
		Stderr:    io.Discard,
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected run to fail when recovery preparation fails, got nil")
	}

	// Verify original file remained intact and was never mutated.
	got, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(got) != "original-data\n" {
		t.Errorf("target file was modified despite prepare failure: %q", string(got))
	}
}

// removeFailingFS fails RemoveAll on rollback to test unresolved reporting.
type removeFailingFS struct {
	FileSystem
	failRemovePath string
}

func (r *removeFailingFS) Remove(path string) error {
	if path == r.failRemovePath {
		return fmt.Errorf("injected remove error on %s", path)
	}
	return r.FileSystem.Remove(path)
}

func (r *removeFailingFS) RemoveAll(path string) error {
	if path == r.failRemovePath {
		return fmt.Errorf("injected remove error on %s", path)
	}
	return r.FileSystem.RemoveAll(path)
}

func TestRecoveryManager_RollbackFailureReported(t *testing.T) { //nolint:paralleltest // shared state
	home, backupDir := setupRecoveryTestEnv(t, "20260101-unresolved", map[string]string{
		"new1.txt": "data1\n",
		"new2.txt": "data2\n",
	})

	targetDir := filepath.Join(home, ".config", "bak")
	t2 := filepath.Join(targetDir, "new2.txt")

	baseFS := newHomeFS(home)
	// Fail copy on t2, and fail remove on t1 during rollback.
	partialFS := &partialCopyFailingFS{
		FileSystem: baseFS,
		failOnDst:  t2,
		failErr:    fmt.Errorf("injected copy failure on t2"),
	}
	remFS := &removeFailingFS{
		FileSystem:     partialFS,
		failRemovePath: t2,
	}

	action := &RestoreAction{
		FS:        remFS,
		BackupDir: backupDir,
		Force:     true,
		Stdout:    io.Discard,
		Stderr:    io.Discard,
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "rollback unresolved") {
		t.Errorf("error %q should mention unresolved rollback", err.Error())
	}
}

func TestRecoveryManager_NoContaminationOfBackupArchives(t *testing.T) { //nolint:paralleltest // shared state
	home, backupDir := setupRecoveryTestEnv(t, "20260101-archive-test", map[string]string{
		"item.txt": "content\n",
	})

	action := &RestoreAction{
		FS:        newHomeFS(home),
		BackupDir: backupDir,
		Force:     true,
		Stdout:    io.Discard,
		Stderr:    io.Discard,
	}

	if err := action.Run(); err != nil {
		t.Fatalf("restore run: %v", err)
	}

	// Verify backup directory contains ONLY its original files and manifest,
	// with NO recovery artifacts contaminated into ~/.bak/backups/<id>.
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("read backup dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "rec-") || e.Name() == "recovery" || e.Name() == "repo" {
			t.Errorf("backup archive was contaminated with recovery item: %s", e.Name())
		}
	}
}

func TestRecoveryManager_ManualRollbackAfterPostRecord_RestoresOriginalBytes(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	targetPath := filepath.Join(home, "file.txt")
	if err := os.WriteFile(targetPath, []byte("original-v1"), 0644); err != nil {
		t.Fatal(err)
	}

	rm := &recoveryManager{
		FS:       newHomeFS(home),
		HomeDir:  home,
		BackupID: "test-backup",
	}

	diffs := []restorepkg.FileDiff{
		{Status: restorepkg.DiffModified, TargetPath: targetPath},
	}
	if err := rm.Prepare(diffs); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	// Mutate target to restored bytes
	if err := os.WriteFile(targetPath, []byte("restored-v2"), 0644); err != nil {
		t.Fatal(err)
	}

	// Record post state (simulates post-recording having captured post-bytes)
	if err := rm.RecordPostState(); err != nil {
		t.Fatalf("record post state: %v", err)
	}

	// Simulate post-state failure triggering rollback
	outcome := rm.Rollback([]string{targetPath})
	if len(outcome.Errors) > 0 {
		t.Fatalf("rollback error: %v", outcome.Errors)
	}

	got, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "original-v1" {
		t.Fatalf("expected target to be rolled back to 'original-v1', got %q", string(got))
	}
}

func TestRecoveryManager_TargetNamedRecoveryMeta_NotCollided(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	targetPath := filepath.Join(home, "recovery-meta.json")
	if err := os.WriteFile(targetPath, []byte("user-meta-content"), 0644); err != nil {
		t.Fatal(err)
	}

	rm := &recoveryManager{
		FS:       newHomeFS(home),
		HomeDir:  home,
		BackupID: "test-backup",
	}

	diffs := []restorepkg.FileDiff{
		{Status: restorepkg.DiffModified, TargetPath: targetPath},
	}
	if err := rm.Prepare(diffs); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	// Mutate target
	if err := os.WriteFile(targetPath, []byte("tampered-content"), 0644); err != nil {
		t.Fatal(err)
	}

	outcome := rm.Rollback([]string{targetPath})
	if len(outcome.Errors) > 0 {
		t.Fatalf("rollback error: %v", outcome.Errors)
	}

	got, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "user-meta-content" {
		t.Fatalf("target was corrupted by metadata collision: got %q, want 'user-meta-content'", string(got))
	}
}

func TestRecoveryManager_StorageSymlinkAncestor_Rejected(t *testing.T) { //nolint:paralleltest // shared state
	if isWindows() {
		t.Skip("symlinks not supported on Windows")
	}
	home := t.TempDir()
	outsideDir := t.TempDir()
	bakSymlink := filepath.Join(home, ".bak")
	if err := os.Symlink(outsideDir, bakSymlink); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(home, "file.txt")
	if err := os.WriteFile(targetPath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	rm := &recoveryManager{
		FS:       newHomeFS(home),
		HomeDir:  home,
		BackupID: "test-backup",
	}

	diffs := []restorepkg.FileDiff{
		{Status: restorepkg.DiffModified, TargetPath: targetPath},
	}
	err := rm.Prepare(diffs)
	if err == nil {
		t.Fatal("expected prepare to fail when recovery storage has symlink ancestor, got nil")
	}

	// Ensure outside directory was not modified
	entries, err := os.ReadDir(outsideDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) > 0 {
		t.Fatalf("outside directory was modified despite symlink ancestor: %d entries", len(entries))
	}
}

func TestRecoveryManager_RollbackDeletion_RefusesDirectoryReplacement(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	targetPath := filepath.Join(home, "newfile.txt")

	rm := &recoveryManager{
		FS:       newHomeFS(home),
		HomeDir:  home,
		BackupID: "test-backup",
	}

	diffs := []restorepkg.FileDiff{
		{Status: restorepkg.DiffNew, TargetPath: targetPath},
	}
	if err := rm.Prepare(diffs); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	// Unexpected directory created at targetPath with user file
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		t.Fatal(err)
	}
	userFile := filepath.Join(targetPath, "user.txt")
	if err := os.WriteFile(userFile, []byte("precious user data"), 0644); err != nil {
		t.Fatal(err)
	}

	outcome := rm.Rollback([]string{targetPath})
	// Should refuse to remove directory and report unresolved
	if len(outcome.Unresolved) == 0 {
		t.Fatalf("expected targetPath to be unresolved, got reverted: %v", outcome.Reverted)
	}

	// Check user file was NOT deleted!
	if _, err := os.Stat(userFile); err != nil {
		t.Fatalf("user file inside directory was destroyed: %v", err)
	}
}

func TestRecoveryManager_Mode0000_Restored(t *testing.T) { //nolint:paralleltest // shared state
	mockFS := &MockFileSystem{
		HomeDir:    "/home/test",
		Files:      make(map[string][]byte),
		ChmodCalls: make(map[string]os.FileMode),
	}
	rm := &recoveryManager{
		FS:        mockFS,
		StorageFS: mockFS,
		HomeDir:   "/home/test",
		PointID:   "rec-test",
		RepoDir:   "/home/test/.bak/recovery/rec-test/repo",
		PointDir:  "/home/test/.bak/recovery/rec-test",
	}

	data := []byte("data")
	h := sha256.Sum256(data)
	pre := targetPreState{
		TargetPath: "/home/test/mode0000.txt",
		RelPath:    "mode0000.txt",
		Existed:    true,
		Mode:       0000,
		SHA256:     fmt.Sprintf("sha256:%x", h),
	}
	// Place mock data in repo pre snapshot
	mockFS.Files[filepath.Join(rm.RepoDir, "snapshots", "pre", opaquePayloadName(pre.RelPath))] = data

	err := rm.rollbackTarget(pre)
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}

	mode, called := mockFS.ChmodCalls["/home/test/mode0000.txt"]
	if !called {
		t.Fatal("expected Chmod to be called for mode 0000, but it was skipped")
	}
	if mode != 0000 {
		t.Fatalf("expected mode 0000, got %v", mode)
	}
}

func TestRecoveryManager_SnapshotTampered_RefusesRollback(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	targetPath := filepath.Join(home, "file.txt")
	if err := os.WriteFile(targetPath, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	rm := &recoveryManager{
		FS:       newHomeFS(home),
		HomeDir:  home,
		BackupID: "test-backup",
	}

	diffs := []restorepkg.FileDiff{
		{Status: restorepkg.DiffModified, TargetPath: targetPath},
	}
	if err := rm.Prepare(diffs); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	// Tamper snapshot file directly
	snapFile := filepath.Join(rm.RepoDir, "snapshots", "pre", opaquePayloadName("file.txt"))
	if err := os.WriteFile(snapFile, []byte("tampered-snapshot"), 0600); err != nil {
		t.Fatal(err)
	}

	outcome := rm.Rollback([]string{targetPath})
	if len(outcome.Unresolved) == 0 {
		t.Fatalf("expected rollback to refuse tampered snapshot and mark unresolved, got reverted: %v", outcome.Reverted)
	}
}

func TestRecoveryManager_TargetOverlapCaseInsensitive_Rejected(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	rm := &recoveryManager{
		FS:       newHomeFS(home),
		HomeDir:  home,
		BackupID: "test-backup",
	}

	// Target overlaps with case alias of recovery directory
	overlapTarget := filepath.Join(home, ".BAK", "RECOVERY", "rogue.txt")
	diffs := []restorepkg.FileDiff{
		{Status: restorepkg.DiffNew, TargetPath: overlapTarget},
	}
	err := rm.Prepare(diffs)
	if err == nil {
		t.Fatal("expected prepare to reject case-insensitive recovery storage overlap, got nil")
	}
	if !strings.Contains(err.Error(), "overlaps recovery storage") {
		t.Errorf("error %q should mention overlaps recovery storage", err.Error())
	}
}

func TestRecoveryManager_RecoveryDirOutsideHome_Rejected(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	outsideDir := t.TempDir()
	rm := &recoveryManager{
		FS:          newHomeFS(home),
		HomeDir:     home,
		BackupID:    "test-backup",
		RecoveryDir: outsideDir,
	}

	diffs := []restorepkg.FileDiff{
		{Status: restorepkg.DiffNew, TargetPath: filepath.Join(home, "file.txt")},
	}
	err := rm.Prepare(diffs)
	if err == nil {
		t.Fatal("expected prepare to reject recovery directory outside home, got nil")
	}
	if !strings.Contains(err.Error(), "escapes home directory") {
		t.Errorf("error %q should mention escapes home directory", err.Error())
	}
}

func TestRecoveryManager_ErrorPrivacy_NoUsernames(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	outsideFile := filepath.Join(t.TempDir(), "secret.txt")

	rm := &recoveryManager{
		FS:       newHomeFS(home),
		HomeDir:  home,
		BackupID: "test-backup",
	}

	diffs := []restorepkg.FileDiff{
		{Status: restorepkg.DiffNew, TargetPath: outsideFile},
	}
	err := rm.Prepare(diffs)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Error message should NOT leak the raw home directory path
	errMsg := err.Error()
	if strings.Contains(errMsg, home) {
		t.Errorf("error message %q leaked home path %q", errMsg, home)
	}
}

func TestRecoveryManager_ThirdTargetUntouchedOnFailure(t *testing.T) { //nolint:paralleltest // shared state
	home, backupDir := setupRecoveryTestEnv(t, "20260101-three-targets", map[string]string{
		"t1.txt": "t1-new\n",
		"t2.txt": "t2-new\n",
		"t3.txt": "t3-new\n",
	})

	targetDir := filepath.Join(home, ".config", "bak")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	t1 := filepath.Join(targetDir, "t1.txt")
	t2 := filepath.Join(targetDir, "t2.txt")
	t3 := filepath.Join(targetDir, "t3.txt")

	if err := os.WriteFile(t1, []byte("t1-orig\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(t2, []byte("t2-orig\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(t3, []byte("t3-orig\n"), 0644); err != nil {
		t.Fatal(err)
	}

	baseFS := newHomeFS(home)
	failingFS := &partialCopyFailingFS{
		FileSystem: baseFS,
		failOnDst:  t2,
		failErr:    fmt.Errorf("injected failure on t2"),
	}

	action := &RestoreAction{
		FS:        failingFS,
		BackupDir: backupDir,
		Force:     true,
		Stdout:    io.Discard,
		Stderr:    io.Discard,
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// t3 was never attempted; its content must be untouched original content
	got3, err := os.ReadFile(t3)
	if err != nil {
		t.Fatal(err)
	}
	if string(got3) != "t3-orig\n" {
		t.Fatalf("t3 was modified: got %q, want 't3-orig\\n'", string(got3))
	}
}

func TestRecoveryManager_GitPayloadDurability_NestedGitAndGitignoreHandled(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	fs := newHomeFS(home)

	gitConfigPath := filepath.Join(home, ".git", "config")
	gitignorePath := filepath.Join(home, ".gitignore")
	ignoredFilePath := filepath.Join(home, "ignored.txt")

	if err := os.MkdirAll(filepath.Dir(gitConfigPath), 0755); err != nil {
		t.Fatal(err)
	}
	gitConfigBytes := []byte("[core]\n\trepositoryformatversion = 0\n")
	gitignoreBytes := []byte("ignored.txt\n*.txt\n")
	ignoredFileBytes := []byte("payload-must-be-durable\n")

	if err := os.WriteFile(gitConfigPath, gitConfigBytes, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(gitignorePath, gitignoreBytes, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ignoredFilePath, ignoredFileBytes, 0644); err != nil {
		t.Fatal(err)
	}

	rm := &recoveryManager{
		FS:       fs,
		HomeDir:  home,
		BackupID: "test-backup",
	}

	diffs := []restorepkg.FileDiff{
		{Status: restorepkg.DiffModified, TargetPath: gitConfigPath},
		{Status: restorepkg.DiffModified, TargetPath: gitignorePath},
		{Status: restorepkg.DiffModified, TargetPath: ignoredFilePath},
	}

	if err := rm.Prepare(diffs); err != nil {
		t.Fatalf("prepare failed: %v", err)
	}

	// Open the recovery repository and inspect the pre-commit tree.
	repo, err := git.OpenRepo(rm.RepoDir)
	if err != nil {
		t.Fatalf("open recovery git repo: %v", err)
	}
	head, err := repo.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	tree, err := commit.Tree()
	if err != nil {
		t.Fatalf("tree: %v", err)
	}

	// Calculate opaque payload names derived from relative path hash
	relGitConfig, _ := filepath.Rel(home, gitConfigPath)
	relGitignore, _ := filepath.Rel(home, gitignorePath)
	relIgnoredFile, _ := filepath.Rel(home, ignoredFilePath)

	hGitConfig := sha256.Sum256([]byte(path.Clean(strings.ReplaceAll(relGitConfig, "\\", "/"))))
	hGitignore := sha256.Sum256([]byte(path.Clean(strings.ReplaceAll(relGitignore, "\\", "/"))))
	hIgnoredFile := sha256.Sum256([]byte(path.Clean(strings.ReplaceAll(relIgnoredFile, "\\", "/"))))

	payloadGitConfig := fmt.Sprintf("%x", hGitConfig)
	payloadGitignore := fmt.Sprintf("%x", hGitignore)
	payloadIgnoredFile := fmt.Sprintf("%x", hIgnoredFile)

	// Assert the files are present in the git commit tree under opaque flat filenames
	entryGitConfig, err := tree.FindEntry("snapshots/pre/" + payloadGitConfig)
	if err != nil {
		t.Fatalf("expected .git/config payload %s to be present in pre-commit tree: %v", payloadGitConfig, err)
	}
	blobGitConfig, err := repo.BlobObject(entryGitConfig.Hash)
	if err != nil {
		t.Fatalf("read blob for git config: %v", err)
	}
	rGitConfig, err := blobGitConfig.Reader()
	if err != nil {
		t.Fatalf("blob reader: %v", err)
	}
	readGitConfigBytes, _ := io.ReadAll(rGitConfig)
	_ = rGitConfig.Close()
	if string(readGitConfigBytes) != string(gitConfigBytes) {
		t.Errorf("git config bytes mismatch in tree: got %q, want %q", string(readGitConfigBytes), string(gitConfigBytes))
	}

	entryIgnored, err := tree.FindEntry("snapshots/pre/" + payloadIgnoredFile)
	if err != nil {
		t.Fatalf("expected ignored file payload %s to be present in pre-commit tree (despite captured .gitignore): %v", payloadIgnoredFile, err)
	}
	blobIgnored, err := repo.BlobObject(entryIgnored.Hash)
	if err != nil {
		t.Fatalf("read blob for ignored file: %v", err)
	}
	rIgnored, err := blobIgnored.Reader()
	if err != nil {
		t.Fatalf("blob reader: %v", err)
	}
	readIgnoredBytes, _ := io.ReadAll(rIgnored)
	_ = rIgnored.Close()
	if string(readIgnoredBytes) != string(ignoredFileBytes) {
		t.Errorf("ignored file bytes mismatch in tree: got %q, want %q", string(readIgnoredBytes), string(ignoredFileBytes))
	}

	entryGitignore, err := tree.FindEntry("snapshots/pre/" + payloadGitignore)
	if err != nil {
		t.Fatalf("expected .gitignore payload %s to be present in pre-commit tree: %v", payloadGitignore, err)
	}
	_ = entryGitignore

	// Mutate all 3 files to prove rollback restores them accurately
	if err := os.WriteFile(gitConfigPath, []byte("mutated-config"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(gitignorePath, []byte("mutated-ignore"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ignoredFilePath, []byte("mutated-ignored-file"), 0644); err != nil {
		t.Fatal(err)
	}

	outcome := rm.Rollback([]string{gitConfigPath, gitignorePath, ignoredFilePath})
	if len(outcome.Errors) > 0 {
		t.Fatalf("rollback returned errors: %v", outcome.Errors)
	}
	if len(outcome.Unresolved) > 0 {
		t.Fatalf("rollback left unresolved targets: %v", outcome.Unresolved)
	}

	gotConfig, err := os.ReadFile(gitConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotConfig) != string(gitConfigBytes) {
		t.Errorf("rollback failed to restore .git/config: got %q, want %q", string(gotConfig), string(gitConfigBytes))
	}

	gotIgnore, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotIgnore) != string(gitignoreBytes) {
		t.Errorf("rollback failed to restore .gitignore: got %q, want %q", string(gotIgnore), string(gitignoreBytes))
	}

	gotIgnored, err := os.ReadFile(ignoredFilePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotIgnored) != string(ignoredFileBytes) {
		t.Errorf("rollback failed to restore ignored file: got %q, want %q", string(gotIgnored), string(ignoredFileBytes))
	}
}

type pathErrorStorageFS struct {
	FileSystem
	failOnWrite string
}

func (p *pathErrorStorageFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	if strings.Contains(path, p.failOnWrite) {
		return &os.PathError{Op: "open", Path: path, Err: os.ErrPermission}
	}
	return p.FileSystem.WriteFile(path, data, perm)
}

func TestRecoveryManager_StorageErrorPrivacy_WrappedPathError(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	fs := newHomeFS(home)

	targetFile := filepath.Join(home, "target.txt")
	if err := os.WriteFile(targetFile, []byte("target-content"), 0644); err != nil {
		t.Fatal(err)
	}

	storageFS := &pathErrorStorageFS{
		FileSystem:  &OSFileSystem{},
		failOnWrite: "snapshots",
	}

	rm := &recoveryManager{
		FS:        fs,
		StorageFS: storageFS,
		HomeDir:   home,
		BackupID:  "test-backup",
		PointID:   "rec-storage-privacy",
	}

	diffs := []restorepkg.FileDiff{
		{Status: restorepkg.DiffModified, TargetPath: targetFile},
	}

	err := rm.Prepare(diffs)
	if err == nil {
		t.Fatal("expected prepare to fail on snapshot storage write error, got nil")
	}

	// Must NOT leak raw home path in error message
	if strings.Contains(err.Error(), home) {
		t.Errorf("error %q leaks absolute home path %q", err.Error(), home)
	}

	// Must preserve errors.Is through wrapped errors
	if !errors.Is(err, os.ErrPermission) {
		t.Errorf("expected errors.Is(err, os.ErrPermission) to be true, got error: %v", err)
	}
}

type chmodFailStorageFS struct {
	FileSystem
}

func (c *chmodFailStorageFS) Chmod(path string, perm os.FileMode) error {
	return fmt.Errorf("injected private chmod failure on %s", path)
}

func TestRecoveryManager_PrivateChmodFailure_NoTargetWrites(t *testing.T) { //nolint:paralleltest // shared state
	if isWindows() {
		t.Skip("skipping chmod failure test on Windows")
	}
	home := t.TempDir()
	fs := newHomeFS(home)

	targetFile := filepath.Join(home, "target.txt")
	origContent := []byte("original-unmodified-content")
	if err := os.WriteFile(targetFile, origContent, 0644); err != nil {
		t.Fatal(err)
	}

	rm := &recoveryManager{
		FS:        fs,
		StorageFS: &chmodFailStorageFS{FileSystem: &OSFileSystem{}},
		HomeDir:   home,
		BackupID:  "test-backup",
	}

	diffs := []restorepkg.FileDiff{
		{Status: restorepkg.DiffModified, TargetPath: targetFile},
	}

	err := rm.Prepare(diffs)
	if err == nil {
		t.Fatal("expected prepare to fail on private chmod failure, got nil")
	}

	// Target file must remain untouched
	got, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(origContent) {
		t.Errorf("target file was modified: got %q, want %q", string(got), string(origContent))
	}
}

type lstatErrFS struct {
	FileSystem
	failPath string
}

func (l *lstatErrFS) Lstat(path string) (os.FileInfo, error) {
	if path == l.failPath {
		return nil, fmt.Errorf("injected IO error on lstat")
	}
	return l.FileSystem.Lstat(path)
}

func TestRecoveryManager_RollbackLstatIO_MarkedUnresolved(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	targetPath := filepath.Join(home, "newfile.txt")

	rm := &recoveryManager{
		FS:       newHomeFS(home),
		HomeDir:  home,
		BackupID: "test-backup",
	}

	diffs := []restorepkg.FileDiff{
		{Status: restorepkg.DiffNew, TargetPath: targetPath},
	}
	if err := rm.Prepare(diffs); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	// Set FS to return non-ErrNotExist I/O error on targetPath
	rm.FS = &lstatErrFS{
		FileSystem: rm.FS,
		failPath:   targetPath,
	}

	outcome := rm.Rollback([]string{targetPath})
	if len(outcome.Unresolved) == 0 {
		t.Fatalf("expected targetPath to be unresolved on lstat IO error, got reverted: %v", outcome.Reverted)
	}
	if len(outcome.Errors) == 0 {
		t.Fatal("expected rollback errors, got none")
	}
}

func TestRecoveryManager_DuplicateCaseAlias_Refused(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	rm := &recoveryManager{
		FS:       newHomeFS(home),
		HomeDir:  home,
		BackupID: "test-backup",
	}

	p1 := filepath.Join(home, "config.txt")
	p2 := filepath.Join(home, "CONFIG.TXT")

	diffs := []restorepkg.FileDiff{
		{Status: restorepkg.DiffNew, TargetPath: p1},
		{Status: restorepkg.DiffNew, TargetPath: p2},
	}

	err := rm.Prepare(diffs)
	if err == nil {
		t.Fatal("expected prepare to refuse duplicate case-alias targets, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate target path") {
		t.Errorf("error %q should mention duplicate target path", err.Error())
	}
}

func TestRecoveryManager_UnsafeRecoveryDirOverride_LeavesExternalDirUntouched(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	outsideDir := t.TempDir()
	canaryFile := filepath.Join(outsideDir, "canary.txt")
	if err := os.WriteFile(canaryFile, []byte("canary"), 0644); err != nil {
		t.Fatal(err)
	}

	rm := &recoveryManager{
		FS:          newHomeFS(home),
		HomeDir:     home,
		BackupID:    "test-backup",
		RecoveryDir: outsideDir,
	}

	diffs := []restorepkg.FileDiff{
		{Status: restorepkg.DiffNew, TargetPath: filepath.Join(home, "f.txt")},
	}

	err := rm.Prepare(diffs)
	if err == nil {
		t.Fatal("expected unsafe recovery dir override to fail, got nil")
	}

	// Canary file must still exist and outsideDir must not have repo created in it
	canaryBytes, err := os.ReadFile(canaryFile)
	if err != nil {
		t.Fatalf("canary file missing: %v", err)
	}
	if string(canaryBytes) != "canary" {
		t.Errorf("canary file was corrupted: %q", string(canaryBytes))
	}
	entries, _ := os.ReadDir(outsideDir)
	if len(entries) != 1 {
		t.Errorf("outside directory was modified: %d entries", len(entries))
	}
}

func TestRecoveryManager_RealChangedModeRestoration(t *testing.T) { //nolint:paralleltest // shared state
	if isWindows() {
		t.Skip("skipping mode restoration test on Windows")
	}
	home := t.TempDir()
	targetPath := filepath.Join(home, "script.sh")
	if err := os.WriteFile(targetPath, []byte("#!/bin/sh\necho hi\n"), 0755); err != nil {
		t.Fatal(err)
	}

	rm := &recoveryManager{
		FS:       newHomeFS(home),
		HomeDir:  home,
		BackupID: "test-backup",
	}

	diffs := []restorepkg.FileDiff{
		{Status: restorepkg.DiffModified, TargetPath: targetPath},
	}
	if err := rm.Prepare(diffs); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	// Mutate both content and mode (e.g. 0644)
	if err := os.WriteFile(targetPath, []byte("#!/bin/sh\necho mutated\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(targetPath, 0644); err != nil {
		t.Fatal(err)
	}

	outcome := rm.Rollback([]string{targetPath})
	if len(outcome.Errors) > 0 {
		t.Fatalf("rollback error: %v", outcome.Errors)
	}

	fi, err := os.Stat(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0755 {
		t.Errorf("target mode was not restored to 0755: got %v", fi.Mode().Perm())
	}
}

func TestRecovery_FindLatestAppliedPoint(t *testing.T) {
	t.Parallel()

	writeMeta := func(recBase, pointID string, meta recoveryMetadata) {
		pDir := filepath.Join(recBase, pointID, "repo")
		if err := os.MkdirAll(pDir, 0755); err != nil {
			t.Fatal(err)
		}
		data, err := json.MarshalIndent(meta, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pDir, "recovery-meta.json"), data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name          string
		setup         func(recBase string)
		overrideBase  func(home string) string
		wantErrIs     error
		wantPointID   string
		expectErrFrag string
	}{
		{
			name:  "none when directory does not exist",
			setup: func(string) {},
			overrideBase: func(home string) string {
				return filepath.Join(home, ".bak", "nonexistent")
			},
			wantErrIs: errNoAppliedPoint,
		},
		{
			name: "none when directory is empty",
			setup: func(recBase string) {
				_ = os.MkdirAll(recBase, 0755)
			},
			wantErrIs: errNoAppliedPoint,
		},
		{
			name: "skip non-applied points",
			setup: func(recBase string) {
				t0 := time.Now().UTC()
				writeMeta(recBase, "p-prepared", recoveryMetadata{PointID: "p-prepared", Status: "prepared", CreatedAt: t0})
				writeMeta(recBase, "p-failed", recoveryMetadata{PointID: "p-failed", Status: "failed", CreatedAt: t0.Add(time.Minute)})
				writeMeta(recBase, "p-rolledback", recoveryMetadata{PointID: "p-rolledback", Status: "rolled_back", CreatedAt: t0.Add(2 * time.Minute)})
				writeMeta(recBase, "p-undone", recoveryMetadata{PointID: "p-undone", Status: "undone", CreatedAt: t0.Add(3 * time.Minute)})
			},
			wantErrIs: errNoAppliedPoint,
		},
		{
			name: "skip corrupt points and find valid applied point",
			setup: func(recBase string) {
				t0 := time.Now().UTC()
				// Corrupt: invalid JSON
				corruptDir := filepath.Join(recBase, "p-corrupt", "repo")
				_ = os.MkdirAll(corruptDir, 0755)
				_ = os.WriteFile(filepath.Join(corruptDir, "recovery-meta.json"), []byte("{bad json"), 0644)

				// Corrupt: empty directory without meta
				_ = os.MkdirAll(filepath.Join(recBase, "p-nometa"), 0755)

				// Regular file instead of directory
				_ = os.WriteFile(filepath.Join(recBase, "p-file"), []byte("not a dir"), 0644)

				// Valid applied point
				writeMeta(recBase, "p-valid", recoveryMetadata{PointID: "p-valid", Status: "applied", CreatedAt: t0})
			},
			wantPointID: "p-valid",
		},
		{
			name: "sort multiple applied points by created_at desc",
			setup: func(recBase string) {
				t0 := time.Now().UTC()
				writeMeta(recBase, "p-old", recoveryMetadata{PointID: "p-old", Status: "applied", CreatedAt: t0.Add(-10 * time.Minute)})
				writeMeta(recBase, "p-newest", recoveryMetadata{PointID: "p-newest", Status: "applied", CreatedAt: t0})
				writeMeta(recBase, "p-mid", recoveryMetadata{PointID: "p-mid", Status: "applied", CreatedAt: t0.Add(-5 * time.Minute)})
			},
			wantPointID: "p-newest",
		},
		{
			name: "tiebreak on identical created_at by point_id desc",
			setup: func(recBase string) {
				t0 := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
				writeMeta(recBase, "p-alpha", recoveryMetadata{PointID: "p-alpha", Status: "applied", CreatedAt: t0})
				writeMeta(recBase, "p-omega", recoveryMetadata{PointID: "p-omega", Status: "applied", CreatedAt: t0})
				writeMeta(recBase, "p-beta", recoveryMetadata{PointID: "p-beta", Status: "applied", CreatedAt: t0})
			},
			wantPointID: "p-omega",
		},
		{
			name:  "recovery base escaping home directory rejected",
			setup: func(string) {},
			overrideBase: func(home string) string {
				return t.TempDir()
			},
			expectErrFrag: "escapes home directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			home := t.TempDir()
			storageFS := newHomeFS(home)
			recBase := filepath.Join(home, ".bak", "recovery")
			if tt.overrideBase != nil {
				recBase = tt.overrideBase(home)
			}
			tt.setup(recBase)

			meta, err := findLatestAppliedPoint(storageFS, recBase, home)
			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("expected error %v, got %v", tt.wantErrIs, err)
				}
				return
			}
			if tt.expectErrFrag != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErrFrag) {
					t.Fatalf("expected error containing %q, got %v", tt.expectErrFrag, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if meta == nil || meta.PointID != tt.wantPointID {
				t.Fatalf("expected PointID %q, got %+v", tt.wantPointID, meta)
			}
		})
	}
}

func setupUndoTestScenario(t *testing.T) (*recoveryManager, string, string, string) {
	t.Helper()
	home := t.TempDir()
	fs := newHomeFS(home)
	recDir := filepath.Join(home, ".bak", "recovery")

	// Target 1: existed prior to restore with "orig1"
	t1 := filepath.Join(home, "file1.txt")
	if err := os.WriteFile(t1, []byte("orig1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Target 2: will be a new file created by restore
	t2 := filepath.Join(home, "file2.txt")

	rm := &recoveryManager{
		FS:          fs,
		StorageFS:   fs,
		HomeDir:     home,
		BackupID:    "b-undo-test",
		RecoveryDir: recDir,
	}

	diffs := []restorepkg.FileDiff{
		{Status: restorepkg.DiffModified, TargetPath: t1},
		{Status: restorepkg.DiffNew, TargetPath: t2},
	}
	if err := rm.Prepare(diffs); err != nil {
		t.Fatalf("prepare: %v", err)
	}

	// Simulate restore apply: t1 modified to "post1", t2 created with "post2"
	if err := os.WriteFile(t1, []byte("post1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(t2, []byte("post2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := rm.RecordPostState(); err != nil {
		t.Fatalf("record post state: %v", err)
	}

	return rm, home, t1, t2
}

func TestRecovery_CheckUndoDrift(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		mutate        func(rm *recoveryManager, home, t1, t2 string)
		expectErrFrag string
		skipWindows   bool
	}{
		{
			name:   "clean state with zero drift passes",
			mutate: func(rm *recoveryManager, home, t1, t2 string) {},
		},
		{
			name: "content drift on t1 rejected",
			mutate: func(rm *recoveryManager, home, t1, t2 string) {
				if err := os.WriteFile(t1, []byte("user-drift-content\n"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			expectErrFrag: "content mismatch",
		},
		{
			name: "mode drift on t1 rejected",
			mutate: func(rm *recoveryManager, home, t1, t2 string) {
				if err := os.Chmod(t1, 0755); err != nil {
					t.Fatal(err)
				}
			},
			expectErrFrag: "mode mismatch",
			skipWindows:   true,
		},
		{
			name: "deletion drift on t1 rejected",
			mutate: func(rm *recoveryManager, home, t1, t2 string) {
				if err := os.Remove(t1); err != nil {
					t.Fatal(err)
				}
			},
			expectErrFrag: "missing",
		},
		{
			name: "unexpected existence drift when post was absent rejected",
			mutate: func(rm *recoveryManager, home, t1, t2 string) {
				// Mark t2 post-state as not existed, but leave file on disk
				cleanT2 := paths.CanonicalPath(t2)
				post := rm.Meta.TargetPostMap[cleanT2]
				post.Existed = false
				rm.Meta.TargetPostMap[cleanT2] = post
			},
			expectErrFrag: "expected ~/file2.txt to be absent",
		},
		{
			name: "symlink drift on t1 rejected",
			mutate: func(rm *recoveryManager, home, t1, t2 string) {
				if err := os.Remove(t1); err != nil {
					t.Fatal(err)
				}
				target := filepath.Join(home, "other.txt")
				_ = os.WriteFile(target, []byte("x"), 0644)
				if err := os.Symlink(target, t1); err != nil {
					t.Fatal(err)
				}
			},
			expectErrFrag: "symlink",
			skipWindows:   true,
		},
		{
			name: "directory replacement drift on t1 rejected",
			mutate: func(rm *recoveryManager, home, t1, t2 string) {
				if err := os.Remove(t1); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(t1, 0755); err != nil {
					t.Fatal(err)
				}
			},
			expectErrFrag: "directory",
		},
		{
			name: "tampered pre snapshot in recovery repo rejected",
			mutate: func(rm *recoveryManager, home, t1, t2 string) {
				preSnap := filepath.Join(rm.RepoDir, "snapshots", "pre", opaquePayloadName("file1.txt"))
				if err := os.WriteFile(preSnap, []byte("tampered-pre"), 0600); err != nil {
					t.Fatal(err)
				}
			},
			expectErrFrag: "tampered or corrupted",
		},
		{
			name: "missing pre snapshot in recovery repo rejected",
			mutate: func(rm *recoveryManager, home, t1, t2 string) {
				preSnap := filepath.Join(rm.RepoDir, "snapshots", "pre", opaquePayloadName("file1.txt"))
				if err := os.Remove(preSnap); err != nil {
					t.Fatal(err)
				}
			},
			expectErrFrag: "pre-snapshot",
		},
		{
			name: "symlink ancestor drift on nested target rejected",
			mutate: func(rm *recoveryManager, home, t1, t2 string) {
				nestedDir := filepath.Join(home, "sub")
				if err := os.Mkdir(nestedDir, 0755); err != nil {
					t.Fatal(err)
				}
				nestedTarget := filepath.Join(nestedDir, "nested.txt")
				if err := os.WriteFile(nestedTarget, []byte("nest"), 0644); err != nil {
					t.Fatal(err)
				}
				cleanNT := paths.CanonicalPath(nestedTarget)
				h := sha256.Sum256([]byte("nest"))
				rm.Meta.TargetPostMap[cleanNT] = targetPostState{
					TargetPath: nestedTarget,
					RelPath:    "sub/nested.txt",
					Existed:    true,
					Mode:       0644,
					SHA256:     fmt.Sprintf("sha256:%x", h),
				}
				rm.Meta.TargetPreMap[cleanNT] = targetPreState{
					TargetPath: nestedTarget,
					RelPath:    "sub/nested.txt",
					Existed:    true,
					Mode:       0644,
					SHA256:     fmt.Sprintf("sha256:%x", h),
				}
				preSnap := filepath.Join(rm.RepoDir, "snapshots", "pre", opaquePayloadName("sub/nested.txt"))
				_ = os.MkdirAll(filepath.Dir(preSnap), 0755)
				_ = os.WriteFile(preSnap, []byte("nest"), 0600)

				otherDir := t.TempDir()
				_ = os.RemoveAll(nestedDir)
				if err := os.Symlink(otherDir, nestedDir); err != nil {
					t.Fatal(err)
				}
			},
			expectErrFrag: "symlink",
			skipWindows:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.skipWindows && isWindows() {
				t.Skip("skipping on Windows")
			}
			rm, home, t1, t2 := setupUndoTestScenario(t)
			tt.mutate(rm, home, t1, t2)

			// Read file stats before check to assert ZERO writes
			stat1Before, _ := os.Stat(t1)
			stat2Before, _ := os.Stat(t2)

			err := rm.CheckUndoDrift()
			if tt.expectErrFrag != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErrFrag) {
					t.Fatalf("expected error containing %q, got %v", tt.expectErrFrag, err)
				}
				// Assert ZERO writes occurred
				stat1After, _ := os.Stat(t1)
				stat2After, _ := os.Stat(t2)
				if (stat1Before == nil) != (stat1After == nil) {
					t.Fatal("file1 existence changed during drift check")
				}
				if (stat2Before == nil) != (stat2After == nil) {
					t.Fatal("file2 existence changed during drift check")
				}
			} else if err != nil {
				t.Fatalf("unexpected drift error: %v", err)
			}
		})
	}
}

func TestRecovery_UndoTargets_Success(t *testing.T) { //nolint:paralleltest // shared state
	rm, home, t1, t2 := setupUndoTestScenario(t)

	// Additional target: mode 0000 restored
	t3 := filepath.Join(home, "mode0000.txt")
	if err := os.WriteFile(t3, []byte("mode0000-orig"), 0644); err != nil {
		t.Fatal(err)
	}
	cleanT3 := paths.CanonicalPath(t3)
	h3 := sha256.Sum256([]byte("mode0000-orig"))
	rm.Meta.TargetPreMap[cleanT3] = targetPreState{
		TargetPath: t3,
		RelPath:    "mode0000.txt",
		Existed:    true,
		Mode:       0000,
		SHA256:     fmt.Sprintf("sha256:%x", h3),
	}
	t3Snap := filepath.Join(rm.RepoDir, "snapshots", "pre", opaquePayloadName("mode0000.txt"))
	if err := os.WriteFile(t3Snap, []byte("mode0000-orig"), 0600); err != nil {
		t.Fatal(err)
	}
	rm.Meta.TargetPostMap[cleanT3] = targetPostState{
		TargetPath: t3,
		RelPath:    "mode0000.txt",
		Existed:    true,
		Mode:       0644,
		SHA256:     fmt.Sprintf("sha256:%x", h3),
	}

	outcome, err := rm.UndoTargets()
	if err != nil {
		t.Fatalf("unexpected undo error: %v (errors: %v)", err, outcome.Errors)
	}

	// 1. Target 1 reverted to original bytes
	data1, err := os.ReadFile(t1)
	if err != nil {
		t.Fatalf("read t1: %v", err)
	}
	if string(data1) != "orig1\n" {
		t.Errorf("expected t1 to have 'orig1\\n', got %q", string(data1))
	}

	// 2. Target 2 was new, so it must be absent after undo
	if _, err := os.Stat(t2); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected t2 to be removed/absent, found: %v", err)
	}

	// 3. Target 3 mode checked (non-Windows)
	if !isWindows() {
		fi3, err := os.Stat(t3)
		if err != nil {
			t.Fatalf("stat t3: %v", err)
		}
		if fi3.Mode().Perm() != 0000 {
			t.Errorf("expected t3 mode 0000, got %04o", fi3.Mode().Perm())
		}
	}

	// 4. Status is undone
	if rm.Meta.Status != "undone" {
		t.Errorf("expected status undone, got %q", rm.Meta.Status)
	}

	// 5. recovery-meta.json committed with undone status
	metaBytes, err := os.ReadFile(filepath.Join(rm.RepoDir, "recovery-meta.json"))
	if err != nil {
		t.Fatalf("read meta: %v", err)
	}
	var savedMeta recoveryMetadata
	if err := json.Unmarshal(metaBytes, &savedMeta); err != nil {
		t.Fatalf("unmarshal meta: %v", err)
	}
	if savedMeta.Status != "undone" {
		t.Errorf("expected meta file status undone, got %q", savedMeta.Status)
	}

	// 6. Outcome accounting
	if len(outcome.Unresolved) != 0 {
		t.Errorf("expected 0 unresolved, got %v", outcome.Unresolved)
	}
	if len(outcome.Reverted) != 3 {
		t.Errorf("expected 3 reverted targets, got %v", outcome.Reverted)
	}
}

func TestRecovery_UndoTargets_PartialFailure(t *testing.T) { //nolint:paralleltest // shared state
	rm, _, _, t2 := setupUndoTestScenario(t)

	// Mutate t2 to be a directory: rollbackNewTarget will refuse to remove it!
	if err := os.Remove(t2); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(t2, 0755); err != nil {
		t.Fatal(err)
	}
	userFile := filepath.Join(t2, "preserve-me.txt")
	if err := os.WriteFile(userFile, []byte("precious"), 0644); err != nil {
		t.Fatal(err)
	}

	// Update t2 post-state so drift check passes for existence
	cleanT2 := paths.CanonicalPath(t2)
	post := rm.Meta.TargetPostMap[cleanT2]
	post.SHA256 = "" // skip hash check on dir if not checked
	rm.Meta.TargetPostMap[cleanT2] = post

	// Call rollbackTarget directly or let UndoTargets execute
	outcome, err := rm.UndoTargets()
	if err == nil {
		t.Fatal("expected error on partial failure, got nil")
	}

	// Assert Status is failed
	if rm.Meta.Status != "failed" {
		t.Errorf("expected status failed, got %q", rm.Meta.Status)
	}

	// Reverted/Unresolved accounting
	if len(outcome.Unresolved) == 0 {
		t.Errorf("expected unresolved target for directory refusal, got %v", outcome.Unresolved)
	}
	if len(outcome.Errors) == 0 {
		t.Error("expected non-empty outcome.Errors")
	}

	// User directory and file must be preserved
	if _, err := os.Stat(userFile); err != nil {
		t.Errorf("user file was damaged during partial failure: %v", err)
	}
}

func TestRecovery_UndoTargets_TamperRefused(t *testing.T) { //nolint:paralleltest // shared state
	rm, _, t1, _ := setupUndoTestScenario(t)

	// Tamper pre-snapshot for t1
	preSnap := filepath.Join(rm.RepoDir, "snapshots", "pre", opaquePayloadName("file1.txt"))
	if err := os.WriteFile(preSnap, []byte("tampered-bytes"), 0600); err != nil {
		t.Fatal(err)
	}

	outcome, err := rm.UndoTargets()
	if err == nil {
		t.Fatal("expected error when pre-snapshot is tampered, got nil")
	}
	if rm.Meta.Status != "failed" {
		t.Errorf("expected status failed, got %q", rm.Meta.Status)
	}
	if len(outcome.Unresolved) == 0 {
		t.Fatalf("expected unresolved targets on tampered snapshot, got %v", outcome.Reverted)
	}
	// Assert live file t1 was NOT modified to tampered bytes
	data, _ := os.ReadFile(t1)
	if string(data) == "tampered-bytes" {
		t.Fatal("target file was overwritten with tampered bytes")
	}
}

func TestRecovery_ErrorPrivacy(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	fs := newHomeFS(home)
	outsideDir := t.TempDir()

	// 1. findLatestAppliedPoint with escaping path
	_, err1 := findLatestAppliedPoint(fs, outsideDir, home)
	if err1 != nil && strings.Contains(err1.Error(), home) {
		t.Errorf("findLatestAppliedPoint leaked home directory in error: %q", err1.Error())
	}

	// 2. CheckUndoDrift with invalid target outside home
	rm := &recoveryManager{
		FS:        fs,
		StorageFS: fs,
		HomeDir:   home,
		Meta: recoveryMetadata{
			Status: "applied",
			TargetPostMap: map[string]targetPostState{
				filepath.Join(outsideDir, "out.txt"): {
					TargetPath: filepath.Join(outsideDir, "out.txt"),
					Existed:    true,
				},
			},
		},
	}
	err2 := rm.CheckUndoDrift()
	if err2 != nil && strings.Contains(err2.Error(), home) {
		t.Errorf("CheckUndoDrift leaked home directory in error: %q", err2.Error())
	}

	// 3. UndoTargets with escaping target in pre-map
	rm3 := &recoveryManager{
		FS:        fs,
		StorageFS: fs,
		HomeDir:   home,
		PointID:   "rec-privacy",
		PointDir:  filepath.Join(home, ".bak", "recovery", "rec-privacy"),
		RepoDir:   filepath.Join(home, ".bak", "recovery", "rec-privacy", "repo"),
		Meta: recoveryMetadata{
			PointID: "rec-privacy",
			Status:  "applied",
			TargetPreMap: map[string]targetPreState{
				filepath.Join(outsideDir, "out.txt"): {
					TargetPath: filepath.Join(outsideDir, "out.txt"),
					Existed:    true,
				},
			},
		},
	}
	_, err3 := rm3.UndoTargets()
	if err3 != nil && strings.Contains(err3.Error(), home) {
		t.Errorf("UndoTargets leaked home directory in error: %q", err3.Error())
	}
}

func TestRecovery_UndoTargets_PersistFailedStateMetadata_OnGitFailure(t *testing.T) {
	t.Parallel()
	rm, _, _, _ := setupUndoTestScenario(t)
	// Corrupt git repo so git.OpenRepo fails
	_ = os.RemoveAll(filepath.Join(rm.RepoDir, ".git"))

	outcome, err := rm.UndoTargets()
	if err == nil {
		t.Fatal("expected error on git open failure, got nil")
	}
	if rm.Meta.Status != "failed" {
		t.Errorf("expected in-memory status failed, got %q", rm.Meta.Status)
	}

	// Verify that recovery-meta.json on disk was updated with Status = "failed"
	metaBytes, readErr := os.ReadFile(filepath.Join(rm.RepoDir, "recovery-meta.json"))
	if readErr != nil {
		t.Fatalf("read recovery-meta.json: %v", readErr)
	}
	var diskMeta recoveryMetadata
	if err := json.Unmarshal(metaBytes, &diskMeta); err != nil {
		t.Fatalf("unmarshal recovery-meta.json: %v", err)
	}
	if diskMeta.Status != "failed" {
		t.Errorf("expected on-disk status failed, got %q", diskMeta.Status)
	}
	if len(outcome.Errors) == 0 {
		t.Error("expected outcome.Errors to contain git failure")
	}
}

func TestRecovery_UndoTargets_PersistFailedStateMetadata_WriteErrorSurfaced(t *testing.T) {
	t.Parallel()
	rm, _, _, _ := setupUndoTestScenario(t)
	// Corrupt git repo so git.OpenRepo fails
	_ = os.RemoveAll(filepath.Join(rm.RepoDir, ".git"))

	// Inject storageFS that fails when writing recovery-meta.json
	rm.StorageFS = &pathErrorStorageFS{
		FileSystem:  &OSFileSystem{},
		failOnWrite: "recovery-meta.json",
	}

	outcome, err := rm.UndoTargets()
	if err == nil {
		t.Fatal("expected error on git failure + metadata write failure, got nil")
	}
	foundWriteErr := false
	for _, e := range outcome.Errors {
		if strings.Contains(e.Error(), "write") || strings.Contains(e.Error(), "recovery-meta.json") {
			foundWriteErr = true
			break
		}
	}
	if !foundWriteErr {
		t.Errorf("expected metadata write error in outcome.Errors, got %v", outcome.Errors)
	}
}

func TestRecovery_FindLatestAppliedPoint_MismatchedPointID_Skipped(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	fs := newHomeFS(home)
	recBase := filepath.Join(home, ".bak", "recovery")

	pointDir := filepath.Join(recBase, "real-entry-id", "repo")
	if err := os.MkdirAll(pointDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Metadata inside real-entry-id claims PointID is spoofed-other-id
	meta := recoveryMetadata{
		PointID:   "spoofed-other-id",
		Status:    "applied",
		CreatedAt: time.Now(),
	}
	data, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(filepath.Join(pointDir, "recovery-meta.json"), data, 0644)

	res, err := findLatestAppliedPoint(fs, recBase, home)
	if !errors.Is(err, errNoAppliedPoint) {
		t.Fatalf("expected errNoAppliedPoint when PointID does not match directory name, got res=%v, err=%v", res, err)
	}
}

func TestRecovery_CheckUndoDrift_PrePostCorrespondenceMismatch(t *testing.T) {
	t.Parallel()
	rm, home, t1, _ := setupUndoTestScenario(t)
	cleanT1 := paths.CanonicalPath(t1)

	// Tamper pre-state so pre.TargetPath points to a different file
	spoofedTarget := filepath.Join(home, "other-file.txt")
	pre := rm.Meta.TargetPreMap[cleanT1]
	pre.TargetPath = spoofedTarget
	rm.Meta.TargetPreMap[cleanT1] = pre

	err := rm.CheckUndoDrift()
	if err == nil || !strings.Contains(err.Error(), "correspondence") {
		t.Fatalf("expected error containing 'correspondence', got %v", err)
	}
}

func TestRecovery_UndoTargets_PrePostCorrespondenceMismatch(t *testing.T) {
	t.Parallel()
	rm, home, t1, _ := setupUndoTestScenario(t)
	cleanT1 := paths.CanonicalPath(t1)

	// Tamper pre-state so pre.TargetPath points to a different file
	spoofedTarget := filepath.Join(home, "other-file.txt")
	pre := rm.Meta.TargetPreMap[cleanT1]
	pre.TargetPath = spoofedTarget
	rm.Meta.TargetPreMap[cleanT1] = pre

	outcome, err := rm.UndoTargets()
	if err == nil || !strings.Contains(err.Error(), "correspondence") {
		t.Fatalf("expected error containing 'correspondence', got %v (outcome errors: %v)", err, outcome.Errors)
	}
	// Assert live target was not touched
	if _, statErr := os.Stat(spoofedTarget); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected spoofed target not to exist, got stat err: %v", statErr)
	}
}
