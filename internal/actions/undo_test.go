package actions

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/paths"
)

func TestUndoAction_Success(t *testing.T) { //nolint:paralleltest // shared state
	rm, home, t1, t2 := setupUndoTestScenario(t)
	var out strings.Builder
	undoCalled := false

	action := &UndoAction{
		FS:          rm.FS,
		StorageFS:   rm.StorageFS,
		RecoveryDir: rm.RecoveryDir,
		Stdout:      &out,
		HomeDir: func() (string, error) {
			return home, nil
		},
		BakDir: func(homeDir string) string {
			return filepath.Join(homeDir, ".bak")
		},
		IsRepo: func(path string) bool {
			return true
		},
		UndoFn: func(repoPath string) error {
			undoCalled = true
			return nil
		},
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if !undoCalled {
		t.Error("UndoFn was not called")
	}

	// Target 1 was pre="orig1\n", post="post1\n". It should now be "orig1\n".
	content1, err := os.ReadFile(t1)
	if err != nil {
		t.Fatalf("read t1: %v", err)
	}
	if string(content1) != "orig1\n" {
		t.Errorf("t1 not restored to pre-state, got %q", string(content1))
	}

	// Target 2 was pre-absent, post="post2\n". It should now be removed.
	if _, err := os.Stat(t2); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("t2 should be removed, got err: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Reverted to previous state") {
		t.Errorf("output should confirm revert: %q", output)
	}
	if !strings.Contains(output, "reverted: ~/file1.txt") {
		t.Errorf("output should mention reverted file1.txt: %q", output)
	}
	if !strings.Contains(output, "reverted: ~/file2.txt") {
		t.Errorf("output should mention reverted file2.txt: %q", output)
	}
}

func TestUndoAction_EachDriftClass_AbortsZeroWrite(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		mutate        func(t *testing.T, rm *recoveryManager, home, t1, t2 string)
		expectErrFrag string
		skipWindows   bool
	}{
		{
			name: "content drift on t1 rejected",
			mutate: func(t *testing.T, rm *recoveryManager, home, t1, t2 string) {
				t.Helper()
				if err := os.WriteFile(t1, []byte("user-drift-content\n"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			expectErrFrag: "content mismatch",
		},
		{
			name: "mode drift on t1 rejected",
			mutate: func(t *testing.T, rm *recoveryManager, home, t1, t2 string) {
				t.Helper()
				if err := os.Chmod(t1, 0755); err != nil {
					t.Fatal(err)
				}
			},
			expectErrFrag: "mode mismatch",
			skipWindows:   true,
		},
		{
			name: "deletion drift on t1 rejected",
			mutate: func(t *testing.T, rm *recoveryManager, home, t1, t2 string) {
				t.Helper()
				if err := os.Remove(t1); err != nil {
					t.Fatal(err)
				}
			},
			expectErrFrag: "missing",
		},
		{
			name: "unexpected existence drift when post was absent rejected",
			mutate: func(t *testing.T, rm *recoveryManager, home, t1, t2 string) {
				t.Helper()
				cleanT2 := paths.CanonicalPath(t2)
				post := rm.Meta.TargetPostMap[cleanT2]
				post.Existed = false
				rm.Meta.TargetPostMap[cleanT2] = post
				metaBytes, err := json.MarshalIndent(rm.Meta, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				metaPath := filepath.Join(rm.RepoDir, "recovery-meta.json")
				if err := os.WriteFile(metaPath, metaBytes, 0600); err != nil {
					t.Fatal(err)
				}
			},
			expectErrFrag: "expected ~/file2.txt to be absent",
		},
		{
			name: "symlink drift on t1 rejected",
			mutate: func(t *testing.T, rm *recoveryManager, home, t1, t2 string) {
				t.Helper()
				if err := os.Remove(t1); err != nil {
					t.Fatal(err)
				}
				other := filepath.Join(home, "other.txt")
				if err := os.WriteFile(other, []byte("x"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(other, t1); err != nil {
					t.Fatal(err)
				}
			},
			expectErrFrag: "symlink",
			skipWindows:   true,
		},
		{
			name: "directory drift on t1 rejected",
			mutate: func(t *testing.T, rm *recoveryManager, home, t1, t2 string) {
				t.Helper()
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
			name: "tampered pre snapshot rejected",
			mutate: func(t *testing.T, rm *recoveryManager, home, t1, t2 string) {
				t.Helper()
				preSnap := filepath.Join(rm.RepoDir, "snapshots", "pre", opaquePayloadName("file1.txt"))
				if err := os.WriteFile(preSnap, []byte("tampered\n"), 0600); err != nil {
					t.Fatal(err)
				}
			},
			expectErrFrag: "checksum mismatch",
		},
		{
			name: "missing pre snapshot rejected",
			mutate: func(t *testing.T, rm *recoveryManager, home, t1, t2 string) {
				t.Helper()
				preSnap := filepath.Join(rm.RepoDir, "snapshots", "pre", opaquePayloadName("file1.txt"))
				if err := os.Remove(preSnap); err != nil {
					t.Fatal(err)
				}
			},
			expectErrFrag: "pre-snapshot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.skipWindows && isWindows() {
				t.Skip("skipping on Windows")
			}
			rm, home, t1, t2 := setupUndoTestScenario(t)
			tt.mutate(t, rm, home, t1, t2)

			// Record file stats / existence before undo
			var t1ExistsBefore bool
			var t1ContentBefore []byte
			if info, err := os.Stat(t1); err == nil && !info.IsDir() {
				t1ExistsBefore = true
				t1ContentBefore, _ = os.ReadFile(t1)
			}
			t2ContentBefore, _ := os.ReadFile(t2)

			undoCalled := false
			var out strings.Builder
			action := &UndoAction{
				FS:          rm.FS,
				StorageFS:   rm.StorageFS,
				RecoveryDir: rm.RecoveryDir,
				Stdout:      &out,
				HomeDir:     func() (string, error) { return home, nil },
				BakDir:      func(string) string { return filepath.Join(home, ".bak") },
				IsRepo:      func(string) bool { return true },
				UndoFn: func(string) error {
					undoCalled = true
					return nil
				},
			}

			err := action.Run()
			if err == nil {
				t.Fatalf("expected drift error for %q, got nil", tt.name)
			}
			if !strings.Contains(err.Error(), tt.expectErrFrag) {
				t.Errorf("error %q should contain %q", err.Error(), tt.expectErrFrag)
			}
			if undoCalled {
				t.Errorf("UndoFn must not be called when drift is detected")
			}

			// Assert ZERO writes on targets after error:
			if t1ExistsBefore {
				t1ContentAfter, _ := os.ReadFile(t1)
				if string(t1ContentAfter) != string(t1ContentBefore) {
					t.Errorf("t1 was modified despite drift error: before %q, after %q", string(t1ContentBefore), string(t1ContentAfter))
				}
			}
			t2ContentAfter, _ := os.ReadFile(t2)
			if string(t2ContentAfter) != string(t2ContentBefore) {
				t.Errorf("t2 was modified despite drift error: before %q, after %q", string(t2ContentBefore), string(t2ContentAfter))
			}
		})
	}
}

func TestUndoAction_NoneFound_FailsClosed(t *testing.T) { //nolint:paralleltest // shared state
	var out strings.Builder
	home := t.TempDir()
	action := &UndoAction{
		Stdout: &out,
		HomeDir: func() (string, error) {
			return home, nil
		},
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error when no applied recovery point found, got nil")
	}
	if !errors.Is(err, ErrNoAppliedRecoveryPoint) && !strings.Contains(err.Error(), "no applied recovery point") {
		t.Fatalf("expected ErrNoAppliedRecoveryPoint or mention, got: %v", err)
	}
	if !strings.Contains(err.Error(), "nothing to undo") {
		t.Fatalf("expected error to mention 'nothing to undo', got: %v", err)
	}
}

func TestUndoAction_NotARepo_SkipsUndoFnSilently(t *testing.T) { //nolint:paralleltest // shared state
	rm, home, t1, t2 := setupUndoTestScenario(t)
	var out strings.Builder
	undoCalled := false

	action := &UndoAction{
		FS:          rm.FS,
		StorageFS:   rm.StorageFS,
		RecoveryDir: rm.RecoveryDir,
		Stdout:      &out,
		HomeDir: func() (string, error) {
			return home, nil
		},
		BakDir: func(homeDir string) string {
			return filepath.Join(homeDir, ".bak")
		},
		IsRepo: func(path string) bool {
			return false
		},
		UndoFn: func(repoPath string) error {
			undoCalled = true
			return nil
		},
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if undoCalled {
		t.Error("UndoFn must not be called when bakDir is not a repo")
	}

	// Target 1 was pre="orig1\n", post="post1\n". It should now be "orig1\n".
	content1, err := os.ReadFile(t1)
	if err != nil {
		t.Fatalf("read t1: %v", err)
	}
	if string(content1) != "orig1\n" {
		t.Errorf("t1 not restored to pre-state, got %q", string(content1))
	}

	// Target 2 was pre-absent, post="post2\n". It should now be removed.
	if _, err := os.Stat(t2); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("t2 should be removed, got err: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Reverted to previous state") {
		t.Errorf("output should confirm revert: %q", output)
	}
}

func TestUndoAction_NotARepo_NoRestore(t *testing.T) { //nolint:paralleltest // shared state
	var out strings.Builder

	action := &UndoAction{
		Stdout: &out,
		HomeDir: func() (string, error) {
			return t.TempDir(), nil
		},
		BakDir: func(homeDir string) string {
			return filepath.Join(homeDir, ".bak")
		},
		IsRepo: func(path string) bool {
			return false
		},
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error when no applied recovery point found, got nil")
	}
	if !strings.Contains(err.Error(), "nothing to undo") {
		t.Errorf("error should mention nothing to undo: %v", err)
	}
	if strings.Contains(err.Error(), "no bak repository") {
		t.Errorf("error should not mention no bak repository: %v", err)
	}
}

func TestUndoAction_UndoFails(t *testing.T) { //nolint:paralleltest // shared state
	rm, home, _, _ := setupUndoTestScenario(t)
	var out strings.Builder

	action := &UndoAction{
		FS:          rm.FS,
		StorageFS:   rm.StorageFS,
		RecoveryDir: rm.RecoveryDir,
		Stdout:      &out,
		HomeDir: func() (string, error) {
			return home, nil
		},
		BakDir: func(homeDir string) string {
			return filepath.Join(homeDir, ".bak")
		},
		IsRepo: func(path string) bool {
			return true
		},
		UndoFn: func(repoPath string) error {
			return errors.New("revert conflict in bak repo")
		},
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error when undo fails")
	}
	if !strings.Contains(err.Error(), "undo failed") {
		t.Errorf("error should mention undo failed: %v", err)
	}
}

func TestUndoAction_DefaultHomeDir(t *testing.T) { //nolint:paralleltest // shared state
	home := t.TempDir()
	action := &UndoAction{
		FS: &homeFS{home: home},
	}
	homeDir, bakDir, recBase, err := action.resolveDirectories()
	if err != nil {
		t.Fatalf("resolveDirectories: %v", err)
	}
	if homeDir != home {
		t.Errorf("expected homeDir %q, got %q", home, homeDir)
	}
	expectedBak := filepath.Join(home, ".bak")
	if bakDir != expectedBak {
		t.Errorf("expected bakDir %q, got %q", expectedBak, bakDir)
	}
	expectedRec := filepath.Join(home, ".bak", "recovery")
	if recBase != expectedRec {
		t.Errorf("expected recBase %q, got %q", expectedRec, recBase)
	}
}

func TestUndoAction_DefaultBakDir(t *testing.T) { //nolint:paralleltest // shared state
	home := "/home/test"
	action := &UndoAction{
		HomeDir: func() (string, error) {
			return home, nil
		},
	}
	homeDir, bakDir, recBase, err := action.resolveDirectories()
	if err != nil {
		t.Fatalf("resolveDirectories: %v", err)
	}
	if homeDir != home {
		t.Errorf("expected homeDir %q, got %q", home, homeDir)
	}
	expectedBak := filepath.Join(home, ".bak")
	if bakDir != expectedBak {
		t.Errorf("expected bakDir %q, got %q", expectedBak, bakDir)
	}
	expectedRec := filepath.Join(home, ".bak", "recovery")
	if recBase != expectedRec {
		t.Errorf("expected recBase %q, got %q", expectedRec, recBase)
	}
}

func TestUndoAction_NilIsRepo_NilUndoFn(t *testing.T) { //nolint:paralleltest // shared state
	rm, home, t1, t2 := setupUndoTestScenario(t)
	var out strings.Builder

	action := &UndoAction{
		FS:          rm.FS,
		StorageFS:   rm.StorageFS,
		RecoveryDir: rm.RecoveryDir,
		Stdout:      &out,
		HomeDir: func() (string, error) {
			return home, nil
		},
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run with nil IsRepo/UndoFn: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Reverted to previous state") {
		t.Errorf("output should confirm revert even without IsRepo/UndoFn: %q", output)
	}

	// Verify target 1 restored and target 2 removed
	c1, _ := os.ReadFile(t1)
	if string(c1) != "orig1\n" {
		t.Errorf("expected t1 to be restored to orig1, got %q", string(c1))
	}
	if _, err := os.Stat(t2); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected t2 to be removed, got err %v", err)
	}
}

func TestUndoAction_HomeDirError(t *testing.T) { //nolint:paralleltest // shared state
	var out strings.Builder

	action := &UndoAction{
		Stdout: &out,
		HomeDir: func() (string, error) {
			return "", errors.New("cannot determine home")
		},
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error when HomeDir fails")
	}
	if !strings.Contains(err.Error(), "cannot determine home") {
		t.Errorf("error should mention home directory: %v", err)
	}
}

func TestUndoAction_PartialAccounting(t *testing.T) { //nolint:paralleltest // shared state
	rm, home, _, t2 := setupUndoTestScenario(t)
	var out strings.Builder

	// Create an unexpected directory at t2. Since t2 was pre-absent,
	// rollbackTarget will refuse to remove an unexpected directory.
	// But first, to pass CheckUndoDrift, t2 must match post-state.
	// We can tamper with rm.Meta or make pre-state invalid for one target after drift check.
	// Better yet, remove the pre-snapshot file for file1.txt, but keep it for file2.txt?
	// Wait, CheckUndoDrift checks pre-snapshot presence for file1.txt.
	// To test partial failure during UndoTargets:
	// If pre-snapshot for file1.txt has invalid permissions or is corrupted after CheckUndoDrift?
	// Or we can corrupt file1.txt pre snapshot right between CheckUndoDrift and UndoTargets?
	// But in a black-box test on Action.Run(), CheckUndoDrift runs immediately before UndoTargets.
	// Wait! If t2 pre-state was absent (so rollbackTarget tries to remove t2),
	// what if t2 removal fails? E.g. t2 is read-only directory or file on a filesystem that errors?
	// Or we can inject a custom FileSystem for FS!
	// Let's create an fs wrapper where Remove(t2) fails!
	partialFS := &removeFailFS{
		FileSystem: rm.FS,
		failPath:   t2,
	}

	action := &UndoAction{
		FS:          partialFS,
		StorageFS:   rm.StorageFS,
		RecoveryDir: rm.RecoveryDir,
		Stdout:      &out,
		HomeDir: func() (string, error) {
			return home, nil
		},
		BakDir: func(string) string {
			return filepath.Join(home, ".bak")
		},
		IsRepo: func(string) bool {
			return true
		},
		UndoFn: func(string) error {
			return nil
		},
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error on partial undo target failure, got nil")
	}

	outStr := out.String()
	// Should report reverted file1.txt and unresolved file2.txt
	if !strings.Contains(outStr, "reverted: ~/file1.txt") {
		t.Errorf("stdout should report reverted file1.txt: %q", outStr)
	}
	if !strings.Contains(outStr, "unresolved: ~/file2.txt") {
		t.Errorf("stdout should report unresolved file2.txt: %q", outStr)
	}
}

type removeFailFS struct {
	FileSystem
	failPath string
}

var _ FileSystem = (*removeFailFS)(nil)

func (r *removeFailFS) Remove(name string) error {
	if paths.CanonicalPath(name) == paths.CanonicalPath(r.failPath) {
		return errors.New("simulated remove failure")
	}
	return r.FileSystem.Remove(name)
}

func TestUndoAction_Privacy(t *testing.T) { //nolint:paralleltest // shared state
	rm, home, t1, _ := setupUndoTestScenario(t)
	// Mutate t1 to trigger drift
	if err := os.WriteFile(t1, []byte("drifted\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	action := &UndoAction{
		FS:          rm.FS,
		StorageFS:   rm.StorageFS,
		RecoveryDir: rm.RecoveryDir,
		Stdout:      &out,
		HomeDir: func() (string, error) {
			return home, nil
		},
		BakDir: func(string) string {
			return filepath.Join(home, ".bak")
		},
		IsRepo: func(string) bool {
			return true
		},
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected drift error, got nil")
	}

	// Verify error does NOT leak raw home directory
	if strings.Contains(err.Error(), home) {
		t.Errorf("error leaked raw home directory %q: %v", home, err)
	}
	// Verify output does NOT leak raw home directory
	if strings.Contains(out.String(), home) {
		t.Errorf("stdout leaked raw home directory %q: %q", home, out.String())
	}
}
