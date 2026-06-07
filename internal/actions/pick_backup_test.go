package actions

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/adapters"
)

// --- ResolveBackupID tests ---

func TestResolveBackupID_ExplicitArg(t *testing.T) {
	backupsDir := t.TempDir()

	id, err := ResolveBackupID(backupsDir, []string{"my-backup-id"})
	if err != nil {
		t.Fatalf("ResolveBackupID: %v", err)
	}
	if id != "my-backup-id" {
		t.Errorf("id = %q, want my-backup-id", id)
	}
}

func TestResolveBackupID_ExplicitArgEmptyString(t *testing.T) {
	// Empty string arg should fall back to finding most recent.
	backupsDir := t.TempDir()

	// Create two backup dirs.
	os.MkdirAll(filepath.Join(backupsDir, "20260101-120000"), 0755)
	os.MkdirAll(filepath.Join(backupsDir, "20260102-080000"), 0755)

	id, err := ResolveBackupID(backupsDir, []string{""})
	if err != nil {
		t.Fatalf("ResolveBackupID: %v", err)
	}
	// Most recent should be last alphabetically (timestamp string sort).
	if id != "20260102-080000" {
		t.Errorf("id = %q, want 20260102-080000 (most recent)", id)
	}
}

func TestResolveBackupID_FallbackToMostRecent(t *testing.T) {
	backupsDir := t.TempDir()

	// Create multiple backup dirs with different timestamps.
	os.MkdirAll(filepath.Join(backupsDir, "20260101-120000"), 0755)
	os.MkdirAll(filepath.Join(backupsDir, "20260102-080000"), 0755)
	os.MkdirAll(filepath.Join(backupsDir, "20260103-090000"), 0755)

	id, err := ResolveBackupID(backupsDir, nil)
	if err != nil {
		t.Fatalf("ResolveBackupID: %v", err)
	}
	if id != "20260103-090000" {
		t.Errorf("id = %q, want 20260103-090000 (most recent)", id)
	}
}

func TestResolveBackupID_NoBackups(t *testing.T) {
	backupsDir := t.TempDir()

	_, err := ResolveBackupID(backupsDir, nil)
	if err == nil {
		t.Fatal("expected error when no backups exist")
	}
	if !strings.Contains(err.Error(), "no backups found") {
		t.Errorf("error should mention no backups found: %v", err)
	}
}

func TestResolveBackupID_NoArgs_FindsMostRecent(t *testing.T) {
	backupsDir := t.TempDir()

	os.MkdirAll(filepath.Join(backupsDir, "20260101-120000"), 0755)

	id, err := ResolveBackupID(backupsDir, nil)
	if err != nil {
		t.Fatalf("ResolveBackupID: %v", err)
	}
	if id != "20260101-120000" {
		t.Errorf("id = %q, want 20260101-120000", id)
	}
}

func TestResolveBackupID_ReadDirError(t *testing.T) {
	// Use a temp dir that we then remove to guarantee ReadDir fails.
	backupsDir := filepath.Join(t.TempDir(), "removed")
	// The dir does not exist — ReadDir should fail.
	_, err := ResolveBackupID(backupsDir, nil)
	if err == nil {
		t.Fatal("expected error for nonexistent dir")
	}
	if !strings.Contains(err.Error(), "read backups dir") {
		t.Errorf("error should mention read backups dir: %v", err)
	}
}

func TestResolveBackupID_IgnoresFiles(t *testing.T) {
	// Only directories should be considered backup IDs.
	backupsDir := t.TempDir()

	// Create a regular file — should be ignored.
	os.WriteFile(filepath.Join(backupsDir, "not-a-backup.txt"), []byte("hello"), 0644)
	// Create an actual backup dir.
	os.MkdirAll(filepath.Join(backupsDir, "20260101-120000"), 0755)

	id, err := ResolveBackupID(backupsDir, nil)
	if err != nil {
		t.Fatalf("ResolveBackupID: %v", err)
	}
	if id != "20260101-120000" {
		t.Errorf("id = %q, want 20260101-120000", id)
	}
}

// --- PickBackupAction tests ---

func TestPickBackupAction_Cancel(t *testing.T) {
	var out strings.Builder

	action := &PickBackupAction{
		Stdout: &out,
		Picker: func(categories []CategoryItem) (PickResult, error) {
			return PickResult{Confirmed: false}, nil
		},
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "cancelled") {
		t.Errorf("output should mention cancelled: %q", output)
	}
}

func TestPickBackupAction_NoCategoriesSelected(t *testing.T) {
	var out strings.Builder

	action := &PickBackupAction{
		Stdout: &out,
		Picker: func(categories []CategoryItem) (PickResult, error) {
			return PickResult{
				Confirmed: true,
				Selected:  nil,
			}, nil
		},
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "No categories selected") {
		t.Errorf("output should mention no categories: %q", output)
	}
}

func TestPickBackupAction_PickerError(t *testing.T) {
	var out strings.Builder

	action := &PickBackupAction{
		Stdout: &out,
		Picker: func(categories []CategoryItem) (PickResult, error) {
			return PickResult{}, errors.New("tui crashed")
		},
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error from picker")
	}
	if !strings.Contains(err.Error(), "tui") {
		t.Errorf("error should mention tui: %v", err)
	}
}

func TestPickBackupAction_BakDirError(t *testing.T) {
	var out strings.Builder

	action := &PickBackupAction{
		Stdout: &out,
		Picker: func(categories []CategoryItem) (PickResult, error) {
			return PickResult{
				Confirmed: true,
				Selected:  []string{"config"},
			}, nil
		},
		BakDir: func() (string, error) {
			return "", errors.New("cannot determine bak dir")
		},
		HomeDir: func() (string, error) {
			return "/home/test", nil
		},
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error from BakDir")
	}
	if !strings.Contains(err.Error(), "bak dir") {
		t.Errorf("error should mention bak dir: %v", err)
	}
}

func TestPickBackupAction_HomeDirError(t *testing.T) {
	var out strings.Builder

	action := &PickBackupAction{
		Stdout: &out,
		Picker: func(categories []CategoryItem) (PickResult, error) {
			return PickResult{
				Confirmed: true,
				Selected:  []string{"config"},
			}, nil
		},
		BakDir: func() (string, error) {
			return "/home/test/.bak", nil
		},
		HomeDir: func() (string, error) {
			return "", errors.New("cannot determine home")
		},
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error from HomeDir")
	}
	if !strings.Contains(err.Error(), "home dir") {
		t.Errorf("error should mention home dir: %v", err)
	}
}

func TestPickBackupAction_NewRegistryError(t *testing.T) {
	var out strings.Builder

	action := &PickBackupAction{
		Stdout: &out,
		Picker: func(categories []CategoryItem) (PickResult, error) {
			return PickResult{
				Confirmed: true,
				Selected:  []string{"config"},
			}, nil
		},
		BakDir: func() (string, error) {
			return "/home/test/.bak", nil
		},
		HomeDir: func() (string, error) {
			return "/home/test", nil
		},
		NewRegistry: func() (*adapters.Registry, error) {
			return nil, errors.New("cannot create registry")
		},
	}

	err := action.Run()
	if err == nil {
		t.Fatal("expected error from NewRegistry")
	}
	if !strings.Contains(err.Error(), "create registry") {
		t.Errorf("error should mention create registry: %v", err)
	}
}

func TestPickBackupAction_DefaultBakDir(t *testing.T) {
	// When BakDir is nil, Run() defaults to backup.BakDir.
	// We cancel before engine to avoid real FS work.
	var out strings.Builder

	action := &PickBackupAction{
		Stdout: &out,
		Picker: func(categories []CategoryItem) (PickResult, error) {
			return PickResult{Confirmed: false}, nil
		},
		// BakDir is nil — exercises default.
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run with nil BakDir: %v", err)
	}

	// Backup was cancelled before engine ran.
	if !strings.Contains(out.String(), "cancelled") {
		t.Errorf("output should mention cancelled: %q", out.String())
	}
}

func TestPickBackupAction_DefaultHomeDir(t *testing.T) {
	// When HomeDir is nil, Run() defaults to os.UserHomeDir.
	var out strings.Builder

	action := &PickBackupAction{
		Stdout: &out,
		Picker: func(categories []CategoryItem) (PickResult, error) {
			return PickResult{Confirmed: false}, nil
		},
		// HomeDir is nil — exercises default.
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run with nil HomeDir: %v", err)
	}

	if !strings.Contains(out.String(), "cancelled") {
		t.Errorf("output should mention cancelled: %q", out.String())
	}
}

func TestPickBackupAction_DefaultNewRegistry(t *testing.T) {
	// When NewRegistry is nil, Run() defaults to real registry.
	// Cancel before engine to avoid real FS work.
	var out strings.Builder

	action := &PickBackupAction{
		Stdout: &out,
		Picker: func(categories []CategoryItem) (PickResult, error) {
			return PickResult{Confirmed: false}, nil
		},
		// NewRegistry is nil — exercises default.
	}

	err := action.Run()
	if err != nil {
		t.Fatalf("Run with nil NewRegistry: %v", err)
	}

	if !strings.Contains(out.String(), "cancelled") {
		t.Errorf("output should mention cancelled: %q", out.String())
	}
}
