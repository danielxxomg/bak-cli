// Package e2e provides end-to-end tests for the bak CLI binary.
package e2e

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/git"
)

var (
	bakBinaryOnce sync.Once
	cachedBakBin  string
	bakBinaryErr  error
)

// getOrBuildBakBinary compiles the bak binary once from the repository root
// into a temporary directory and caches the executable path for the test run.
func getOrBuildBakBinary(t *testing.T) string {
	t.Helper()
	bakBinaryOnce.Do(func() {
		moduleRoot, err := filepath.Abs(filepath.Join("..", ".."))
		if err != nil {
			bakBinaryErr = fmt.Errorf("resolve module root: %w", err)
			return
		}

		binDir, err := os.MkdirTemp("", "bak-journey-bin-*")
		if err != nil {
			bakBinaryErr = fmt.Errorf("create temp bin dir: %w", err)
			return
		}

		binPath := filepath.Join(binDir, "bak")
		if runtime.GOOS == "windows" {
			binPath += ".exe"
		}

		buildCmd := exec.Command("go", "build", "-o", binPath, ".")
		buildCmd.Dir = moduleRoot
		if out, err := buildCmd.CombinedOutput(); err != nil {
			bakBinaryErr = fmt.Errorf("build bak: %w\n%s", err, string(out))
			return
		}
		cachedBakBin = binPath
	})

	if bakBinaryErr != nil {
		t.Fatalf("getOrBuildBakBinary: %v", bakBinaryErr)
	}
	return cachedBakBin
}

// cmdResult captures execution outcomes from a real binary invocation.
type cmdResult struct {
	exitCode int
	stdout   string
	stderr   string
	err      error
}

// runBakCmd executes the bak binary with the supplied arguments and sandboxed environment.
func runBakCmd(bakBin string, env []string, args ...string) cmdResult {
	cmd := exec.Command(bakBin, args...)
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	res := cmdResult{
		stdout: stdout.String(),
		stderr: stderr.String(),
		err:    err,
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			res.exitCode = exitErr.ExitCode()
		} else {
			res.exitCode = -1
		}
	}
	return res
}

// setupJourneySandbox creates an isolated temporary home directory with valid
// OpenCode configuration fixtures and returns the directory and its sandbox environment.
func setupJourneySandbox(t *testing.T) (string, []string) {
	t.Helper()
	tempHome := t.TempDir()
	env := sandboxEnv(tempHome)

	cfgDir := filepath.Join(tempHome, ".config", "opencode")
	mustMkdirAll(t, cfgDir)

	jsonPath := filepath.Join(cfgDir, "opencode.json")
	mustWriteFile(t, jsonPath, []byte("{\"version\":\"1.0\",\"theme\":\"dark\"}\n"))

	agentsPath := filepath.Join(cfgDir, "AGENTS.md")
	mustWriteFile(t, agentsPath, []byte("# OpenCode Agents\nInitial agent configuration.\n"))

	if runtime.GOOS != "windows" {
		if err := os.Chmod(jsonPath, 0600); err != nil {
			t.Fatalf("chmod json: %v", err)
		}
		if err := os.Chmod(agentsPath, 0755); err != nil {
			t.Fatalf("chmod agents: %v", err)
		}
	}

	return tempHome, env
}

// parseBackupID parses the created backup ID from bak backup stdout.
func parseBackupID(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Backup created:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Backup created:"))
		}
	}
	return ""
}

// setupJourneyWithBackup prepares a sandboxed home and executes bak backup --preset quick,
// returning the tempHome, environment, and created backup ID.
func setupJourneyWithBackup(t *testing.T, bakBin string) (string, []string, string) {
	t.Helper()
	tempHome, env := setupJourneySandbox(t)

	res := runBakCmd(bakBin, env, "backup", "--preset", "quick")
	if res.exitCode != 0 {
		t.Fatalf("setup backup failed: exit %d, stderr: %s", res.exitCode, res.stderr)
	}

	backupID := parseBackupID(res.stdout)
	if backupID == "" {
		t.Fatalf("could not parse backup ID from stdout:\n%s", res.stdout)
	}
	return tempHome, env, backupID
}

// initBakGitRepo initializes a Git repository in ~/.bak with two linear commits
// so that bak undo has a valid repository with a parent commit to revert.
func initBakGitRepo(t *testing.T, tempHome string) {
	t.Helper()
	bakDir := filepath.Join(tempHome, ".bak")
	mustMkdirAll(t, bakDir)
	repo, err := git.InitRepo(bakDir)
	if err != nil {
		t.Fatalf("init bak git repo: %v", err)
	}
	markerFile := filepath.Join(bakDir, ".bak-git-marker")
	mustWriteFile(t, markerFile, []byte("v1\n"))
	if err := git.StageAll(repo); err != nil {
		t.Fatalf("stage marker: %v", err)
	}
	if err := git.Commit(repo, "bak: initial"); err != nil {
		t.Fatalf("commit marker: %v", err)
	}
	mustWriteFile(t, markerFile, []byte("v2\n"))
	if err := git.StageAll(repo); err != nil {
		t.Fatalf("stage marker v2: %v", err)
	}
	if err := git.Commit(repo, "bak: restore applied"); err != nil {
		t.Fatalf("commit marker v2: %v", err)
	}
}

// TestJourneyMatrix executes the T3 eight-stage real-binary journey matrix
// verifying end-to-end correctness, permissions, zero-write guarantees, and recovery.
func TestJourneyMatrix(t *testing.T) { //nolint:paralleltest // integration tests with process execution and file sandboxing
	bakBin := getOrBuildBakBinary(t)

	t.Run("Stage1_Discovery", func(t *testing.T) { //nolint:paralleltest // subtest runs sequentially
		_, env, backupID := setupJourneyWithBackup(t, bakBin)

		// Table-driven discovery cases
		tests := []struct {
			name         string
			args         []string
			wantExit     int
			wantStdoutIn string
			wantStderrIn string
		}{
			{
				name:         "list reports created backup",
				args:         []string{"list"},
				wantExit:     0,
				wantStdoutIn: backupID,
			},
			{
				name:         "negative invalid backup id format",
				args:         []string{"restore", "--dry-run", "invalid-id-format"},
				wantExit:     1,
				wantStderrIn: "invalid backup ID",
			},
			{
				name:         "negative unknown backup id",
				args:         []string{"restore", "--dry-run", "20000101-000000"},
				wantExit:     1,
				wantStderrIn: "not found",
			},
			{
				name:         "negative verify unknown backup id",
				args:         []string{"verify", "20000101-000000"},
				wantExit:     1,
				wantStderrIn: "not found",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share state
				res := runBakCmd(bakBin, env, tt.args...)
				if res.exitCode != tt.wantExit {
					t.Errorf("%s: expected exit code %d, got %d (stderr: %s)", tt.name, tt.wantExit, res.exitCode, res.stderr)
				}
				if tt.wantStdoutIn != "" && !strings.Contains(res.stdout, tt.wantStdoutIn) {
					t.Errorf("%s: expected stdout to contain %q, got:\n%s", tt.name, tt.wantStdoutIn, res.stdout)
				}
				if tt.wantStderrIn != "" && !strings.Contains(res.stderr, tt.wantStderrIn) {
					t.Errorf("%s: expected stderr to contain %q, got:\n%s", tt.name, tt.wantStderrIn, res.stderr)
				}
			})
		}
	})

	t.Run("Stage2_MutationDeletion", func(t *testing.T) { //nolint:paralleltest // subtest runs sequentially
		tempHome, env, backupID := setupJourneyWithBackup(t, bakBin)

		jsonPath := filepath.Join(tempHome, ".config", "opencode", "opencode.json")
		agentsPath := filepath.Join(tempHome, ".config", "opencode", "AGENTS.md")

		origJSONBytes := []byte("{\"version\":\"1.0\",\"theme\":\"dark\"}\n")
		origAgentsBytes := []byte("# OpenCode Agents\nInitial agent configuration.\n")

		// Modify opencode.json and delete AGENTS.md
		mutatedJSON := []byte("{\"version\":\"2.0\",\"mutated\":true}\n")
		mustWriteFile(t, jsonPath, mutatedJSON)

		if err := os.Remove(agentsPath); err != nil {
			t.Fatalf("remove agents path: %v", err)
		}
		if _, err := os.Stat(agentsPath); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("agents path should not exist: %v", err)
		}

		// Diff / --dry-run must reflect modified content and deletion (as new target)
		dryRes := runBakCmd(bakBin, env, "restore", "--dry-run", backupID)
		if dryRes.exitCode != 0 {
			t.Fatalf("restore --dry-run: exit %d, stderr: %s", dryRes.exitCode, dryRes.stderr)
		}
		if !strings.Contains(dryRes.stdout, "opencode.json") || !strings.Contains(dryRes.stdout, "[modified]") {
			t.Errorf("dry-run should show [modified] opencode.json, got:\n%s", dryRes.stdout)
		}
		if !strings.Contains(dryRes.stdout, "AGENTS.md") || !strings.Contains(dryRes.stdout, "[new]") {
			t.Errorf("dry-run should show [new] AGENTS.md, got:\n%s", dryRes.stdout)
		}

		// Real restore recovers original bytes for both files
		restoreRes := runBakCmd(bakBin, env, "restore", "--force", backupID)
		if restoreRes.exitCode != 0 {
			t.Fatalf("restore --force: exit %d, stderr: %s", restoreRes.exitCode, restoreRes.stderr)
		}

		gotJSON, err := os.ReadFile(jsonPath)
		if err != nil {
			t.Fatalf("read restored json: %v", err)
		}
		if !bytes.Equal(gotJSON, origJSONBytes) {
			t.Errorf("restored json mismatch: got %q, want %q", gotJSON, origJSONBytes)
		}

		gotAgents, err := os.ReadFile(agentsPath)
		if err != nil {
			t.Fatalf("read restored agents: %v", err)
		}
		if !bytes.Equal(gotAgents, origAgentsBytes) {
			t.Errorf("restored agents mismatch: got %q, want %q", gotAgents, origAgentsBytes)
		}
	})

	t.Run("Stage3_DryRunNoWrite", func(t *testing.T) { //nolint:paralleltest // subtest runs sequentially
		tempHome, env, backupID := setupJourneyWithBackup(t, bakBin)

		jsonPath := filepath.Join(tempHome, ".config", "opencode", "opencode.json")
		agentsPath := filepath.Join(tempHome, ".config", "opencode", "AGENTS.md")

		// Modify opencode.json and leave AGENTS.md untouched
		mutatedJSON := []byte("{\"version\":\"3.0\",\"dry_run_check\":true}\n")
		mustWriteFile(t, jsonPath, mutatedJSON)

		jsonInfoBefore, err := os.Stat(jsonPath)
		if err != nil {
			t.Fatal(err)
		}
		jsonBytesBefore, err := os.ReadFile(jsonPath)
		if err != nil {
			t.Fatal(err)
		}

		agentsInfoBefore, err := os.Stat(agentsPath)
		if err != nil {
			t.Fatal(err)
		}
		agentsBytesBefore, err := os.ReadFile(agentsPath)
		if err != nil {
			t.Fatal(err)
		}

		// Execute dry run
		dryRes := runBakCmd(bakBin, env, "restore", "--dry-run", backupID)
		if dryRes.exitCode != 0 {
			t.Fatalf("restore --dry-run: exit %d, stderr: %s", dryRes.exitCode, dryRes.stderr)
		}
		if !strings.Contains(dryRes.stdout, "opencode.json") {
			t.Errorf("dry-run output should name opencode.json, got:\n%s", dryRes.stdout)
		}

		// Assert zero writes: byte equality, mode equality, and modtime equality
		jsonBytesAfter, err := os.ReadFile(jsonPath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(jsonBytesBefore, jsonBytesAfter) {
			t.Errorf("json bytes altered by dry-run: before %q, after %q", jsonBytesBefore, jsonBytesAfter)
		}

		jsonInfoAfter, err := os.Stat(jsonPath)
		if err != nil {
			t.Fatal(err)
		}
		if jsonInfoBefore.Mode() != jsonInfoAfter.Mode() {
			t.Errorf("json mode altered by dry-run: before %v, after %v", jsonInfoBefore.Mode(), jsonInfoAfter.Mode())
		}
		if !jsonInfoBefore.ModTime().Equal(jsonInfoAfter.ModTime()) {
			t.Errorf("json modtime altered by dry-run: before %v, after %v", jsonInfoBefore.ModTime(), jsonInfoAfter.ModTime())
		}

		agentsBytesAfter, err := os.ReadFile(agentsPath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(agentsBytesBefore, agentsBytesAfter) {
			t.Errorf("agents bytes altered by dry-run: before %q, after %q", agentsBytesBefore, agentsBytesAfter)
		}

		agentsInfoAfter, err := os.Stat(agentsPath)
		if err != nil {
			t.Fatal(err)
		}
		if agentsInfoBefore.Mode() != agentsInfoAfter.Mode() {
			t.Errorf("agents mode altered by dry-run: before %v, after %v", agentsInfoBefore.Mode(), agentsInfoAfter.Mode())
		}
		if !agentsInfoBefore.ModTime().Equal(agentsInfoAfter.ModTime()) {
			t.Errorf("agents modtime altered by dry-run: before %v, after %v", agentsInfoBefore.ModTime(), agentsInfoAfter.ModTime())
		}
	})

	t.Run("Stage4_ApplyCorrectness", func(t *testing.T) { //nolint:paralleltest // subtest runs sequentially
		tempHome, env, backupID := setupJourneyWithBackup(t, bakBin)

		jsonPath := filepath.Join(tempHome, ".config", "opencode", "opencode.json")
		agentsPath := filepath.Join(tempHome, ".config", "opencode", "AGENTS.md")

		origJSONBytes := []byte("{\"version\":\"1.0\",\"theme\":\"dark\"}\n")
		origAgentsBytes := []byte("# OpenCode Agents\nInitial agent configuration.\n")

		// Overwrite targets with drifted content and modes
		mustWriteFile(t, jsonPath, []byte("{\"corrupted\":true}\n"))
		mustWriteFile(t, agentsPath, []byte("# Corrupted\n"))
		if runtime.GOOS != "windows" {
			if err := os.Chmod(jsonPath, 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(agentsPath, 0644); err != nil {
				t.Fatal(err)
			}
		}

		// Apply restore
		restoreRes := runBakCmd(bakBin, env, "restore", "--force", backupID)
		if restoreRes.exitCode != 0 {
			t.Fatalf("restore --force: exit %d, stderr: %s", restoreRes.exitCode, restoreRes.stderr)
		}

		// Assert exact byte equality
		gotJSON, err := os.ReadFile(jsonPath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(gotJSON, origJSONBytes) {
			t.Errorf("json content mismatch: got %q, want %q", gotJSON, origJSONBytes)
		}

		gotAgents, err := os.ReadFile(agentsPath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(gotAgents, origAgentsBytes) {
			t.Errorf("agents content mismatch: got %q, want %q", gotAgents, origAgentsBytes)
		}

		// Assert permission bits equal manifest mode on non-Windows
		if runtime.GOOS != "windows" {
			fiJSON, err := os.Stat(jsonPath)
			if err != nil {
				t.Fatal(err)
			}
			if fiJSON.Mode().Perm() != 0600 {
				t.Errorf("json permission mismatch: expected 0600, got %o", fiJSON.Mode().Perm())
			}

			fiAgents, err := os.Stat(agentsPath)
			if err != nil {
				t.Fatal(err)
			}
			if fiAgents.Mode().Perm() != 0755 {
				t.Errorf("agents permission mismatch: expected 0755, got %o", fiAgents.Mode().Perm())
			}
		}
	})

	t.Run("Stage5_Verify", func(t *testing.T) { //nolint:paralleltest // subtest runs sequentially
		_, env, backupID := setupJourneyWithBackup(t, bakBin)

		verRes := runBakCmd(bakBin, env, "verify", backupID)
		if verRes.exitCode != 0 {
			t.Errorf("bak verify on untouched backup: expected exit 0, got %d, stderr: %s", verRes.exitCode, verRes.stderr)
		}
		if !strings.Contains(verRes.stdout, "verified") || !strings.Contains(verRes.stdout, backupID) {
			t.Errorf("bak verify output missing confirmation: %s", verRes.stdout)
		}
	})

	t.Run("Stage6_TamperNegative", func(t *testing.T) { //nolint:paralleltest // subtest runs sequentially
		tempHome, env, backupID := setupJourneyWithBackup(t, bakBin)

		// Corrupt the backed-up payload file in ~/.bak/backups/<backupID>/opencode/opencode.json
		payloadPath := filepath.Join(tempHome, ".bak", "backups", backupID, "opencode", "opencode.json")
		tamperedData := []byte("TAMPERED_PAYLOAD_CORRUPTION\n")
		mustWriteFile(t, payloadPath, tamperedData)

		// Target on disk has canary content
		jsonPath := filepath.Join(tempHome, ".config", "opencode", "opencode.json")
		canaryBytes := []byte("{\"canary\":\"untouched\"}\n")
		mustWriteFile(t, jsonPath, canaryBytes)

		// bak verify must fail closed with exit 1
		verRes := runBakCmd(bakBin, env, "verify", backupID)
		if verRes.exitCode != 1 {
			t.Errorf("bak verify on tampered payload: expected exit 1, got %d", verRes.exitCode)
		}

		// bak restore must fail closed with exit 1
		restoreRes := runBakCmd(bakBin, env, "restore", "--force", backupID)
		if restoreRes.exitCode != 1 {
			t.Errorf("bak restore on tampered payload: expected exit 1, got %d", restoreRes.exitCode)
		}
		if !strings.Contains(restoreRes.stderr, "checksum") && !strings.Contains(restoreRes.stderr, "mismatch") {
			t.Errorf("bak restore: expected checksum mismatch on stderr, got: %s", restoreRes.stderr)
		}

		// Confirm target files are unchanged and tampered content was not applied
		gotJSON, err := os.ReadFile(jsonPath)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(gotJSON, tamperedData) {
			t.Fatal("tampered payload content was applied to target file!")
		}
		if !bytes.Equal(gotJSON, canaryBytes) {
			t.Errorf("target file changed: got %q, want %q", gotJSON, canaryBytes)
		}
	})

	t.Run("Stage7_PartialFailure", func(t *testing.T) { //nolint:paralleltest // subtest runs sequentially
		tempHome, env, backupID := setupJourneyWithBackup(t, bakBin)

		jsonPath := filepath.Join(tempHome, ".config", "opencode", "opencode.json")
		agentsPath := filepath.Join(tempHome, ".config", "opencode", "AGENTS.md")

		preAgentsBytes := []byte("# Pre-restore Agents\n")
		preJSONBytes := []byte("{\"pre_restore\":true}\n")
		mustWriteFile(t, agentsPath, preAgentsBytes)
		mustWriteFile(t, jsonPath, preJSONBytes)

		// Make jsonPath unwritable (mode 0400).
		// AGENTS.md is processed first in manifest order, so AGENTS.md copy succeeds
		// and opencode.json copy fails with permission denied.
		if err := os.Chmod(jsonPath, 0400); err != nil {
			t.Fatal(err)
		}
		defer func() {
			_ = os.Chmod(jsonPath, 0644)
		}()

		restoreRes := runBakCmd(bakBin, env, "restore", "--force", backupID)
		if restoreRes.exitCode != 1 {
			t.Errorf("restore with unwritable target: expected exit 1, got %d (stdout: %s, stderr: %s)",
				restoreRes.exitCode, restoreRes.stdout, restoreRes.stderr)
		}

		// Unaffected target (AGENTS.md) must NOT be left half-written; rollback must revert it to pre-state
		gotAgents, err := os.ReadFile(agentsPath)
		if err != nil {
			t.Fatalf("read agentsPath: %v", err)
		}
		if !bytes.Equal(gotAgents, preAgentsBytes) {
			t.Errorf("unaffected target was left modified/half-written: got %q, want %q", gotAgents, preAgentsBytes)
		}

		// Reported outcome must name applied/reverted/unresolved counts consistent with actual disk state
		combinedOut := restoreRes.stdout + "\n" + restoreRes.stderr
		if !strings.Contains(combinedOut, "reverted") && !strings.Contains(combinedOut, "unresolved") {
			t.Errorf("expected outcome to mention reverted/unresolved counts, got:\n%s", combinedOut)
		}
	})

	t.Run("Stage8_RealRecovery", func(t *testing.T) { //nolint:paralleltest // subtest runs sequentially
		tempHome, env, backupID := setupJourneyWithBackup(t, bakBin)

		// Initialize Git repo in ~/.bak for bak undo to operate on
		initBakGitRepo(t, tempHome)

		jsonPath := filepath.Join(tempHome, ".config", "opencode", "opencode.json")
		agentsPath := filepath.Join(tempHome, ".config", "opencode", "AGENTS.md")

		// Phase A: Establish an applied recovery point via a successful restore
		preAgents := []byte("# Pre-restore Agents for Undo\n")
		preJSON := []byte("{\"pre_restore_undo\":true}\n")
		mustWriteFile(t, agentsPath, preAgents)
		mustWriteFile(t, jsonPath, preJSON)

		resApply := runBakCmd(bakBin, env, "restore", "--force", backupID)
		if resApply.exitCode != 0 {
			t.Fatalf("setup successful restore: exit %d, stderr: %s", resApply.exitCode, resApply.stderr)
		}

		backedUpAgents := []byte("# OpenCode Agents\nInitial agent configuration.\n")
		backedUpJSON := []byte("{\"version\":\"1.0\",\"theme\":\"dark\"}\n")
		gotAgents, err := os.ReadFile(agentsPath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(gotAgents, backedUpAgents) {
			t.Fatalf("setup: agents not restored: got %q, want %q", gotAgents, backedUpAgents)
		}

		// Phase B: Drift protection refusal
		// Mutate target to introduce drift
		driftedAgents := []byte("# Drifted agents content\n")
		mustWriteFile(t, agentsPath, driftedAgents)

		// bak undo must refuse drift with exit 1
		undoDriftRes := runBakCmd(bakBin, env, "undo")
		if undoDriftRes.exitCode != 1 {
			t.Errorf("bak undo with drift: expected exit 1, got %d, stdout: %s, stderr: %s",
				undoDriftRes.exitCode, undoDriftRes.stdout, undoDriftRes.stderr)
		}
		if !strings.Contains(undoDriftRes.stderr, "drift") {
			t.Errorf("bak undo with drift: expected 'drift' on stderr, got: %s", undoDriftRes.stderr)
		}

		// Assert zero writes on drift refusal
		gotAfterDrift, err := os.ReadFile(agentsPath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(gotAfterDrift, driftedAgents) {
			t.Errorf("target modified despite drift rejection: got %q, want %q", gotAfterDrift, driftedAgents)
		}

		// Phase C: Clean recovery when no drift exists
		// Re-align target back to post-restore state
		mustWriteFile(t, agentsPath, backedUpAgents)
		mustWriteFile(t, jsonPath, backedUpJSON)

		// bak undo must succeed with exit 0 and revert targets to pre-state
		undoCleanRes := runBakCmd(bakBin, env, "undo")
		if undoCleanRes.exitCode != 0 {
			t.Fatalf("bak undo without drift: expected exit 0, got %d, stderr: %s, stdout: %s",
				undoCleanRes.exitCode, undoCleanRes.stderr, undoCleanRes.stdout)
		}
		if !strings.Contains(undoCleanRes.stdout, "Reverted to previous state") {
			t.Errorf("bak undo: expected 'Reverted to previous state' in stdout, got:\n%s", undoCleanRes.stdout)
		}

		// Assert real on-disk bytes match the pre-restore state
		revertedAgents, err := os.ReadFile(agentsPath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(revertedAgents, preAgents) {
			t.Errorf("target agents not reverted to pre-state: got %q, want %q", revertedAgents, preAgents)
		}

		revertedJSON, err := os.ReadFile(jsonPath)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(revertedJSON, preJSON) {
			t.Errorf("target json not reverted to pre-state: got %q, want %q", revertedJSON, preJSON)
		}
	})
}
