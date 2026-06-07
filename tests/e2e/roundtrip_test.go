// Package e2e provides end-to-end tests for the bak CLI binary.
package e2e

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/manifest"
	"github.com/danielxxomg/bak-cli/internal/paths"
)

// TestBackupRestoreRoundtrip verifies that a backup→restore cycle preserves
// file content. It creates real files in an isolated temp home directory,
// runs `bak backup` and `bak restore --force`, then verifies every restored
// file's SHA-256 checksum matches its manifest entry. This test serves as a
// guardrail for the coverage DI refactor.
func TestBackupRestoreRoundtrip(t *testing.T) {
	t.Run("quick_preset_config_files", func(t *testing.T) {
		testRoundtrip(t, "quick", func(home string) {
			// Create config-level fixture files (backed up by "config" category).
			cfgDir := filepath.Join(home, ".config", "opencode")
			mustMkdirAll(t, cfgDir)
			mustWriteFile(t, filepath.Join(cfgDir, "opencode.json"), []byte(`{"version":"1.0","theme":"dark"}`+"\n"))
			mustWriteFile(t, filepath.Join(cfgDir, "AGENTS.md"), []byte("# OpenCode Agents\n\nTest agent configuration.\n"))
		})
	})

	t.Run("skills_preset_skill_dir", func(t *testing.T) {
		testRoundtrip(t, "skills", func(home string) {
			// Create skills directory with a SKILL.md file.
			cfgDir := filepath.Join(home, ".config", "opencode")
			mustMkdirAll(t, cfgDir)
			// The adapter needs the directory to exist for detection.
			mustWriteFile(t, filepath.Join(cfgDir, "opencode.json"), []byte(`{"version":"1.0"}`+"\n"))

			skillDir := filepath.Join(cfgDir, "skills", "roundtrip-skill")
			mustMkdirAll(t, skillDir)
			mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"), []byte("# Roundtrip Skill\n\nThis skill was created for E2E testing.\n"))
			mustWriteFile(t, filepath.Join(skillDir, "README.md"), []byte("# Roundtrip README\n\nAdditional file.\n"))
		})
	})
}

// testRoundtrip runs the full backup→restore cycle for the given preset
// and verifies all restored files match their manifest checksums.
func testRoundtrip(t *testing.T, preset string, setupFixtures func(home string)) {
	t.Helper()

	// --- 1. Build the bak binary from the module root -------------------
	tempHome := t.TempDir()

	moduleRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}

	bakBin := filepath.Join(tempHome, "bak")
	if runtime.GOOS == "windows" {
		bakBin += ".exe"
	}

	buildCmd := exec.Command("go", "build", "-o", bakBin, ".")
	buildCmd.Dir = moduleRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build bak: %v\n%s", err, string(out))
	}

	// --- 2. Create fixture files inside the sandboxed home --------------
	setupFixtures(tempHome)

	// --- 3. Build environment pointing to the sandbox -------------------
	env := sandboxEnv(tempHome)

	// --- 4. Run backup --------------------------------------------------
	backupID := runBakBackup(t, bakBin, env, preset)

	// --- 5. Run restore with --force ------------------------------------
	runBakRestore(t, bakBin, env, backupID)

	// --- 6. Load manifest and verify every file checksum ----------------
	backupDir := filepath.Join(tempHome, ".bak", "backups", backupID)
	m, err := manifest.Load(backupDir)
	if err != nil {
		t.Fatalf("load manifest from %s: %v", backupDir, err)
	}

	if len(m.Adapters) == 0 {
		t.Fatal("manifest contains no adapters — backup may have produced no files")
	}

	filesChecked := 0
	for adapterName, am := range m.Adapters {
		for _, item := range am.Items {
			// Resolve the canonical source path to the sandbox home.
			restoredAbs := paths.FromCanonical(item.SourcePath, tempHome)

			actualHash, err := computeFileSHA256(restoredAbs)
			if err != nil {
				t.Errorf("adapter %q, file %q (%s): compute hash: %v",
					adapterName, item.BackupPath, restoredAbs, err)
				continue
			}
			if actualHash != item.Hash {
				t.Errorf("adapter %q, file %q (%s): hash mismatch\n  expected: %s\n  actual:   %s",
					adapterName, item.BackupPath, restoredAbs, item.Hash, actualHash)
			}
			filesChecked++
		}
	}

	if filesChecked == 0 {
		t.Error("no files were checked — manifest may contain only directories")
	}

	// --- 7. Also run manifest.Validate as a secondary check -------------
	progressPaths := make([]string, 0)
	if err := m.Validate(backupDir, func(p string) {
		progressPaths = append(progressPaths, p)
	}); err != nil {
		t.Errorf("manifest.Validate: %v", err)
	}
	if len(progressPaths) != filesChecked {
		t.Errorf("manifest.Validate visited %d files, expected %d", len(progressPaths), filesChecked)
	}

	t.Logf("roundtrip %s: %d files verified OK", preset, filesChecked)
}

// runBakBackup executes `bak backup --preset <preset>` and returns the
// backup ID parsed from stdout.
func runBakBackup(t *testing.T, bakBin string, env []string, preset string) string {
	t.Helper()

	cmd := exec.Command(bakBin, "backup", "--preset", preset)
	cmd.Env = env
	cmd.Stderr = os.Stderr // show errors in test output for debugging

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start bak backup: %v", err)
	}

	scanner := bufio.NewScanner(stdout)
	var backupID string
	for scanner.Scan() {
		line := scanner.Text()
		t.Logf("[bak backup] %s", line)
		if strings.HasPrefix(line, "Backup created:") {
			backupID = strings.TrimSpace(strings.TrimPrefix(line, "Backup created:"))
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("read backup stdout: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("bak backup: %v", err)
	}

	if backupID == "" {
		t.Fatal("could not parse backup ID from output")
	}
	return backupID
}

// runBakRestore executes `bak restore <backupID> --force`.
func runBakRestore(t *testing.T, bakBin string, env []string, backupID string) {
	t.Helper()

	cmd := exec.Command(bakBin, "restore", backupID, "--force")
	cmd.Env = env
	cmd.Stderr = os.Stderr

	// Capture stdout for logging.
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("bak restore: %v\n%s", err, string(out))
	}
	t.Logf("[bak restore]\n%s", string(out))
}

// sandboxEnv builds a process environment that isolates the bak binary
// to the given tempHome directory.
func sandboxEnv(tempHome string) []string {
	env := os.Environ()
	// Replace HOME/USERPROFILE/APPDATA to sandbox the bak binary.
	env = setOrAppendEnv(env, "HOME", tempHome)
	if runtime.GOOS == "windows" {
		env = setOrAppendEnv(env, "USERPROFILE", tempHome)
		env = setOrAppendEnv(env, "APPDATA", filepath.Join(tempHome, ".config"))
	}
	return env
}

// setOrAppendEnv replaces an existing env var or appends a new one.
func setOrAppendEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

// computeFileSHA256 returns the "sha256:<hex>" digest of the file at path.
func computeFileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash: %w", err)
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}

// mustMkdirAll calls os.MkdirAll and fails the test on error.
func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

// mustWriteFile calls os.WriteFile and fails the test on error.
func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
