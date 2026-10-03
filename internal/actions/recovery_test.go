package actions

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/git"
	"github.com/danielxxomg/bak-cli/internal/manifest"
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
