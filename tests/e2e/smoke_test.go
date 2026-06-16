// Package e2e provides end-to-end smoke tests that verify the bak binary
// launches correctly in common scenarios: --help, no args (non-TTY), and
// unknown subcommand. These guard against startup panics and ensure the
// CLI entry point behaves as expected in CI/non-interactive environments.
package e2e

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestBinaryHelp verifies that `bak --help` exits cleanly and prints the
// expected help banner.
func TestBinaryHelp(t *testing.T) {
	bakBin := buildSmokeBinary(t)

	cmd := exec.Command(bakBin, "--help")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("bak --help: %v\n%s", err, string(out))
	}

	output := string(out)
	// The root command Short description ("Backup and restore your AI coding
	// setup") is not rendered in cobra's default --help output; cobra displays
	// the Long description instead. Assert on the Long text that actually
	// appears in help output.
	wantBanner := "packs, restores, and syncs your OpenCode configuration"
	if !strings.Contains(output, wantBanner) {
		t.Errorf("bak --help output missing expected banner %q\ngot:\n%s", wantBanner, output)
	}
}

// TestBinaryNoArgs verifies that running `bak` with no arguments in a
// non-TTY environment falls through to cobra's help output and exits 0.
// In CI/test environments, stdin is piped so isTTY() returns false,
// triggering the help fallback in root.go instead of the interactive TUI.
func TestBinaryNoArgs(t *testing.T) {
	bakBin := buildSmokeBinary(t)

	// exec.Command with nil stdin → child process has no TTY → isTTY() false
	cmd := exec.Command(bakBin)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("bak (no args): %v\n%s", err, string(out))
	}

	output := string(out)
	if !strings.Contains(output, "Usage:") {
		t.Errorf("bak (no args) output missing help text\ngot:\n%s", output)
	}
}

// TestBinaryUnknownCommand verifies that `bak <nonexistent>` fails with a
// non-zero exit code and an error message indicating the command is unknown.
func TestBinaryUnknownCommand(t *testing.T) {
	bakBin := buildSmokeBinary(t)

	cmd := exec.Command(bakBin, "nonexistent-command")
	out, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("bak nonexistent-command: expected non-zero exit, got 0")
	}

	output := string(out)
	if !strings.Contains(output, "unknown command") {
		t.Errorf("bak nonexistent-command stderr missing 'unknown command'\ngot:\n%s", output)
	}
}

// buildSmokeBinary compiles the bak binary from the module root into a
// temporary directory and returns the path to the resulting executable.
func buildSmokeBinary(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	moduleRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}

	bakBin := filepath.Join(tempDir, "bak")
	if runtime.GOOS == "windows" {
		bakBin += ".exe"
	}

	//nolint:gosec // binary path from go build output, controlled by test
	buildCmd := exec.Command("go", "build", "-o", bakBin, ".")
	buildCmd.Dir = moduleRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build bak: %v\n%s", err, string(out))
	}

	return bakBin
}
