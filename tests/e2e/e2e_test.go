// Package e2e provides end-to-end testscript tests for the bak CLI binary.
// Tests run against a compiled bak binary and exercise real backup, verify,
// restore, diff, profile, and schedule workflows in isolated environments.
package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestE2E(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:           "testdata",
		Setup:         setupEnv,
		UpdateScripts: os.Getenv("UPDATE_SCRIPTS") != "",
	})
}

// setupEnv configures the test environment before each script runs.
func setupEnv(e *testscript.Env) error {
	// Set HOME to the testscript work directory so bak writes to a sandboxed
	// ~/.bak and reads from sandboxed ~/.config.
	e.Setenv("HOME", e.WorkDir)

	// On Windows, os.UserHomeDir() reads USERPROFILE (not HOME) and
	// os.UserConfigDir() reads APPDATA. Override both so the sandboxed
	// fixture directories are discovered.
	if runtime.GOOS == "windows" {
		e.Setenv("USERPROFILE", e.WorkDir)
		// os.UserConfigDir() returns %APPDATA%; point it to ~/.config
		// so paths.ConfigDir("bak") resolves to the fixture-written
		// $WORK/.config/bak/config.json.
		e.Setenv("APPDATA", filepath.Join(e.WorkDir, ".config"))
	}

	// Build the bak binary and place it in the test's PATH.
	bakBin := filepath.Join(e.WorkDir, "bak")
	if runtime.GOOS == "windows" {
		bakBin += ".exe"
	}
	// go build from the module root (testscript sets GOPATH correctly).
	//nolint:gosec // binary path controlled by test, not user input
	buildCmd := exec.Command("go", "build", "-o", bakBin, ".")
	buildCmd.Dir = e.WorkDir
	// We need the module source available. testscript copies testdata scripts
	// but not the full module. Use a relative path from the working dir.
	// Since testscript runs from the test file's directory, we reference
	// the module root relative to tests/e2e/.
	moduleRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		return fmt.Errorf("resolve module root: %w", err)
	}
	buildCmd.Dir = moduleRoot
	buildCmd.Args = []string{"go", "build", "-o", bakBin, "."}
	if out, err := buildCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("build bak: %w\n%s", err, string(out))
	}

	// Ensure bak is in PATH.
	envPath := e.Getenv("PATH")
	e.Setenv("PATH", e.WorkDir+string(os.PathListSeparator)+envPath)

	// Create minimal OpenCode fixture so bak backup can detect the adapter.
	configDir := filepath.Join(e.WorkDir, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(
		filepath.Join(configDir, "opencode.json"),
		[]byte(`{"version":"1.0"}`),
		0644,
	); err != nil {
		return err
	}
	if err := os.WriteFile(
		filepath.Join(configDir, "AGENTS.md"),
		[]byte("# Agents\n"),
		0644,
	); err != nil {
		return err
	}

	// Create skills directory with a sample skill.
	skillDir := filepath.Join(configDir, "skills", "my-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("# My Skill\n"),
		0644,
	); err != nil {
		return err
	}

	// Set up bak config with a minimal provider so profile create works.
	bakConfigDir := filepath.Join(e.WorkDir, ".config", "bak")
	if err := os.MkdirAll(bakConfigDir, 0755); err != nil {
		return err
	}
	configContent := `{"version":"0.3.0","providers":{"github-gist":{"token":"ghp_test00000000000000000000000000000000"}},"profiles":{}}
`
	if err := os.WriteFile(
		filepath.Join(bakConfigDir, "config.json"),
		[]byte(configContent),
		0644,
	); err != nil {
		return err
	}

	// On macOS, os.UserConfigDir() returns $HOME/Library/Application Support.
	// Write the config there too so bak finds it on macOS CI.
	if runtime.GOOS == "darwin" {
		macConfigDir := filepath.Join(e.WorkDir, "Library", "Application Support", "bak")
		if err := os.MkdirAll(macConfigDir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(
			filepath.Join(macConfigDir, "config.json"),
			[]byte(configContent),
			0644,
		); err != nil {
			return err
		}
	}

	return nil
}
