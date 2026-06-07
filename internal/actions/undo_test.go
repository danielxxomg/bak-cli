package actions

import (
	"errors"
	"strings"
	"testing"
)

func TestUndoAction_Success(t *testing.T) {
	var out strings.Builder
	undoCalled := false

	action := &UndoAction{
		Stdout: &out,
		HomeDir: func() (string, error) {
			return "/home/test", nil
		},
		BakDir: func(homeDir string) string {
			return homeDir + "/.bak"
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
	output := out.String()
	if !strings.Contains(output, "Reverted") {
		t.Errorf("output should confirm revert: %q", output)
	}
}

func TestUndoAction_NotARepo(t *testing.T) {
	var out strings.Builder

	action := &UndoAction{
		Stdout: &out,
		HomeDir: func() (string, error) {
			return "/home/test", nil
		},
		BakDir: func(homeDir string) string {
			return homeDir + "/.bak"
		},
		IsRepo: func(path string) bool {
			return false
		},
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error when not a repo")
	}
	if !strings.Contains(err.Error(), "no bak repository") {
		t.Errorf("error should mention no bak repository: %v", err)
	}
}

func TestUndoAction_UndoFails(t *testing.T) {
	var out strings.Builder

	action := &UndoAction{
		Stdout: &out,
		HomeDir: func() (string, error) {
			return "/home/test", nil
		},
		BakDir: func(homeDir string) string {
			return homeDir + "/.bak"
		},
		IsRepo: func(path string) bool {
			return true
		},
		UndoFn: func(repoPath string) error {
			return errors.New("revert conflict")
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

func TestUndoAction_DefaultHomeDir(t *testing.T) {
	// When HomeDir is nil, Run() defaults to os.UserHomeDir.
	// This exercises the default path without needing real git repo.
	var out strings.Builder

	action := &UndoAction{
		Stdout: &out,
		// HomeDir is nil — exercises default.
		IsRepo: func(path string) bool {
			return false // short-circuit: not a repo error before reaching undo
		},
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error when not a repo (default HomeDir)")
	}
	if !strings.Contains(err.Error(), "no bak repository") {
		t.Errorf("error should mention no bak repository: %v", err)
	}
}

func TestUndoAction_DefaultBakDir(t *testing.T) {
	// When BakDir is nil, Run() defaults to filepath.Join(homeDir, ".bak").
	var out strings.Builder

	action := &UndoAction{
		Stdout: &out,
		HomeDir: func() (string, error) {
			return "/home/test", nil
		},
		// BakDir is nil — exercises default.
		IsRepo: func(path string) bool {
			return false
		},
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error when not a repo (default BakDir)")
	}
	if !strings.Contains(err.Error(), "no bak repository") {
		t.Errorf("error should mention no bak repository: %v", err)
	}
}

func TestUndoAction_NilIsRepo_NilUndoFn(t *testing.T) {
	// When both IsRepo and UndoFn are nil, Run() skips the checks
	// and prints the success message.
	var out strings.Builder

	action := &UndoAction{
		Stdout: &out,
		HomeDir: func() (string, error) {
			return "/home/test", nil
		},
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run with nil IsRepo/UndoFn: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Reverted") {
		t.Errorf("output should confirm revert even without IsRepo/UndoFn: %q", output)
	}
}

func TestUndoAction_HomeDirError(t *testing.T) {
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
