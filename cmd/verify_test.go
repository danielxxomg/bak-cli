package cmd

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/danielxxomg/bak-cli/internal/manifest"
)

// stageVerifyBackup creates a valid backup directory structure in tmpDir
// and returns the backup ID and path to the backup directory.
func stageVerifyBackup(t *testing.T, tmpDir string, id string, fileCount int) string {
	t.Helper()

	backupsDir := filepath.Join(tmpDir, ".bak", "backups")
	backupDir := filepath.Join(backupsDir, id)
	os.MkdirAll(backupDir, 0755)

	items := make([]manifest.Item, fileCount)
	for i := 0; i < fileCount; i++ {
		fname := fmt.Sprintf("file-%d.txt", i)
		fpath := filepath.Join(backupDir, "opencode", fname)
		os.MkdirAll(filepath.Dir(fpath), 0755)
		content := []byte(fmt.Sprintf("content-%d", i))
		os.WriteFile(fpath, content, 0644)

		hash := sha256.Sum256(content)
		items[i] = manifest.Item{
			Category:   "config",
			SourcePath: fmt.Sprintf("~/.config/opencode/%s", fname),
			BackupPath: fmt.Sprintf("opencode/%s", fname),
			Hash:       fmt.Sprintf("sha256:%x", hash),
			Size:       int64(len(content)),
		}
	}

	m := &manifest.Manifest{
		Version:    "0.3.0",
		ID:         id,
		BakVersion: "0.3.0",
		Preset:     "quick",
		Adapters: map[string]manifest.AdapterManifest{
			"opencode": {Items: items},
		},
		FileCount: fileCount,
	}
	m.Save(backupDir)
	return backupDir
}

func TestVerifyCmd_Registered(t *testing.T) {
	found := false
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "verify" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("verify subcommand not registered on root")
	}
}

func TestVerifyCmd_Args(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "verify" {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("verify command not found")
	}

	// Verify requires exactly 1 arg.
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("expected error with 0 args")
	}
	if err := cmd.Args(cmd, []string{"a", "b"}); err == nil {
		t.Error("expected error with 2 args")
	}
	if err := cmd.Args(cmd, []string{"valid-id"}); err != nil {
		t.Errorf("expected no error with 1 arg, got %v", err)
	}
}

func TestRunVerify_Success(t *testing.T) {
	tmpDir := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("USERPROFILE", tmpDir)
	default:
		t.Setenv("HOME", tmpDir)
	}

	id := "20250101-120000"
	stageVerifyBackup(t, tmpDir, id, 2)

	deps, stdout, _ := setupTestDeps(t)
	cmd := &cobra.Command{}
	err := runVerifyWithDeps(cmd, []string{id}, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "verified") {
		t.Errorf("output should contain 'verified', got: %s", output)
	}
	if !strings.Contains(output, "2 file") {
		t.Errorf("output should mention file count 2, got: %s", output)
	}
}

func TestRunVerify_CorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("USERPROFILE", tmpDir)
	default:
		t.Setenv("HOME", tmpDir)
	}

	id := "20250101-120000"
	backupDir := stageVerifyBackup(t, tmpDir, id, 2)

	// Mutate a file to cause hash mismatch.
	fpath := filepath.Join(backupDir, "opencode", "file-0.txt")
	os.WriteFile(fpath, []byte("corrupted!"), 0644)

	deps, _, _ := setupTestDeps(t)
	cmd := &cobra.Command{}
	err := runVerifyWithDeps(cmd, []string{id}, deps)
	if err == nil {
		t.Fatal("expected error for corrupted file, got nil")
	}
	if !strings.Contains(err.Error(), "hash mismatch") {
		t.Errorf("error should mention 'hash mismatch', got: %v", err)
	}
}

func TestRunVerify_MissingBackup(t *testing.T) {
	tmpDir := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("USERPROFILE", tmpDir)
	default:
		t.Setenv("HOME", tmpDir)
	}

	deps, _, _ := setupTestDeps(t)
	cmd := &cobra.Command{}
	err := runVerifyWithDeps(cmd, []string{"nonexistent"}, deps)
	if err == nil {
		t.Fatal("expected error for missing backup")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestRunVerify_TraversalBlocked(t *testing.T) {
	tmpDir := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("USERPROFILE", tmpDir)
	default:
		t.Setenv("HOME", tmpDir)
	}

	deps, _, _ := setupTestDeps(t)
	cmd := &cobra.Command{}
	err := runVerifyWithDeps(cmd, []string{"../etc"}, deps)
	if err == nil {
		t.Fatal("expected error for traversal")
	}
	if !strings.Contains(err.Error(), "outside") {
		t.Errorf("error should mention 'outside', got: %v", err)
	}
}

func TestRunVerify_VerboseOutput(t *testing.T) {
	tmpDir := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("USERPROFILE", tmpDir)
	default:
		t.Setenv("HOME", tmpDir)
	}

	id := "20250101-120000"
	stageVerifyBackup(t, tmpDir, id, 1)

	deps, _, stderr := setupTestDeps(t)
	cmd := &cobra.Command{}

	// Enable verbose via package variable (flag binding happens in main).
	oldVerbose := verbose
	verbose = true
	defer func() { verbose = oldVerbose }()

	err := runVerifyWithDeps(cmd, []string{id}, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verbose output goes to stderr.
	stderrOutput := stderr.String()
	if !strings.Contains(stderrOutput, "Verifying") {
		t.Errorf("verbose stderr should contain 'Verifying', got: %s", stderrOutput)
	}
}

func TestVerifyCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"verify", "--help"})
	rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "verify") {
		t.Fatal("help output should mention 'verify'")
	}
}

func TestVerifyCmd_UseAndDescription(t *testing.T) {
	var cmd *cobra.Command
	for _, sub := range rootCmd.Commands() {
		if sub.Name() == "verify" {
			cmd = sub
			break
		}
	}
	if cmd == nil {
		t.Fatal("verify command not found")
	}
	if cmd.Use == "" {
		t.Fatal("verify Use should not be empty")
	}
	if cmd.Short == "" {
		t.Fatal("verify Short should not be empty")
	}
}
