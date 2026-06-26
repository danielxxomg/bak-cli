//go:build !windows

//nolint:gosec // Test file — exec.Command and file paths are test-controlled
package schedule

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCronScheduler_TypeCheck verifies CronScheduler satisfies Scheduler.
func TestCronScheduler_TypeCheck(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	var s Scheduler = &CronScheduler{}
	if _, ok := s.(*CronScheduler); !ok {
		t.Fatal("CronScheduler does not satisfy Scheduler")
	}
}

// --- readCrontab ---

func TestReadCrontab_HasContent(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	orig := execCommand
	defer func() { execCommand = orig }()

	stateFile := filepath.Join(t.TempDir(), "crontab.txt")
	os.WriteFile(stateFile, []byte("0 2 * * * bak backup --profile work # bak-cli:work\n"), 0644)

	execCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("cat", stateFile)
	}

	result, err := readCrontab()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "bak-cli:work") {
		t.Errorf("missing expected tag, got: %q", result)
	}
}

func TestReadCrontab_Empty(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	orig := execCommand
	defer func() { execCommand = orig }()

	execCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("cat", "/dev/null")
	}

	result, err := readCrontab()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(result) != "" {
		t.Errorf("expected empty, got: %q", result)
	}
}

func TestReadCrontab_Error(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	orig := execCommand
	defer func() { execCommand = orig }()

	execCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("false")
	}

	_, err := readCrontab()
	if err == nil {
		t.Fatal("expected error from readCrontab when command fails")
	}
}

// --- writeCrontab ---

func TestWriteCrontab_Success(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	orig := execCommand
	defer func() { execCommand = orig }()

	stateFile := filepath.Join(t.TempDir(), "crontab.txt")

	execCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("tee", stateFile)
	}

	err := writeCrontab("0 2 * * * bak backup # bak-cli:work\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(stateFile)
	if !strings.Contains(string(content), "bak-cli:work") {
		t.Errorf("expected bak-cli:work in written crontab, got: %s", string(content))
	}
}

// --- CronScheduler integration tests (mocked execCommand) ---

func setupMockExec(t *testing.T) (stateFile string, restore func()) {
	t.Helper()

	orig := execCommand
	stateFile = filepath.Join(t.TempDir(), "crontab.txt")

	execCommand = func(name string, arg ...string) *exec.Cmd {
		if len(arg) > 0 && arg[0] == "-l" {
			return exec.Command("cat", stateFile)
		}
		return exec.Command("tee", stateFile)
	}

	return stateFile, func() { execCommand = orig }
}

func TestCronScheduler_CreateAndList(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	stateFile, restore := setupMockExec(t)
	defer restore()

	s := &CronScheduler{}

	if err := s.Create("work", "daily"); err != nil {
		t.Fatalf("Create work: %v", err)
	}
	if err := s.Create("home", "weekly"); err != nil {
		t.Fatalf("Create home: %v", err)
	}

	entries, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	content, _ := os.ReadFile(stateFile)
	sc := string(content)
	if !strings.Contains(sc, "bak-cli:work") {
		t.Error("missing work entry in crontab state")
	}
	if !strings.Contains(sc, "bak-cli:home") {
		t.Error("missing home entry in crontab state")
	}
}

func TestCronScheduler_Create_Duplicate(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	stateFile, restore := setupMockExec(t)
	defer restore()

	s := &CronScheduler{}

	s.Create("work", "daily")
	s.Create("work", "daily")

	entries, _ := s.List()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after duplicate create, got %d", len(entries))
	}

	content, _ := os.ReadFile(stateFile)
	sc := string(content)
	// Verify "bak-cli:work" appears exactly once.
	count := strings.Count(sc, "bak-cli:work")
	if count != 1 {
		t.Errorf("expected 1 occurrence of bak-cli:work, got %d\ncontent: %s", count, sc)
	}
}

func TestCronScheduler_Remove(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	stateFile, restore := setupMockExec(t)
	defer restore()

	// Pre-populate the crontab with one entry.
	initial := "0 2 * * * bak backup --profile work && bak push --profile work # bak-cli:work\n"
	os.WriteFile(stateFile, []byte(initial), 0644)

	s := &CronScheduler{}

	if err := s.Remove("work"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	entries, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries after remove, got %d", len(entries))
	}
}

func TestCronScheduler_Remove_NotFound(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	stateFile, restore := setupMockExec(t)
	defer restore()

	os.WriteFile(stateFile, []byte(""), 0644)

	s := &CronScheduler{}
	err := s.Remove("nonexistent")
	if err == nil {
		t.Fatal("expected error when removing non-existent profile")
	}
}

func TestCronScheduler_List_EmptyCrontab(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	_, restore := setupMockExec(t)
	defer restore()

	s := &CronScheduler{}
	entries, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries from empty crontab, got %d", len(entries))
	}
}

func TestCronScheduler_Create_NoPreExistingCrontab(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// When crontab file doesn't exist, readCrontab fails → Create starts fresh.
	orig := execCommand
	defer func() { execCommand = orig }()

	stateFile := filepath.Join(t.TempDir(), "crontab.txt")
	// Do NOT pre-create the state file — simulate "no crontab" scenario.

	execCommand = func(name string, arg ...string) *exec.Cmd {
		if len(arg) > 0 && arg[0] == "-l" {
			// File doesn't exist → cat fails → readCrontab returns error.
			return exec.Command("cat", stateFile)
		}
		return exec.Command("tee", stateFile)
	}

	s := &CronScheduler{}
	err := s.Create("work", "daily")
	if err != nil {
		t.Fatalf("Create with no pre-existing crontab: %v", err)
	}

	content, _ := os.ReadFile(stateFile)
	if !strings.Contains(string(content), "bak-cli:work") {
		t.Errorf("expected bak-cli:work after fresh create, got: %s", string(content))
	}
}
