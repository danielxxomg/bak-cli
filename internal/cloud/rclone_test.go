package cloud

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// createMockRclone creates a fake rclone executable in a temp directory
// and returns the path to the mock binary. The script uses its first
// argument to determine behavior:
//
//	"copyto" — exit 0 (success)
//	"cat"    — write expected output to stdout via temp file
//	"lsf"    — write expected output to stdout via temp file
//
// If RCLONE_MOCK_FAIL is set to "1" in the environment, the mock
// always exits with code 1 and writes an error to stderr.
func createMockRclone(t *testing.T, dir string, output string) string {
	t.Helper()

	outputFile := filepath.Join(dir, "mock_output.txt")
	if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
		t.Fatalf("create mock output: %v", err)
	}

	var scriptPath, scriptContent string

	if runtime.GOOS == "windows" {
		scriptPath = filepath.Join(dir, "rclone.bat")
		scriptContent = fmt.Sprintf(`@echo off
if "%%RCLONE_MOCK_FAIL%%"=="1" (
    echo rclone error: something went wrong >&2
    exit /b 1
)
if "%%1"=="cat" (
    type "%s"
    exit /b 0
)
if "%%1"=="lsf" (
    type "%s"
    exit /b 0
)
exit /b 0
`, outputFile, outputFile)
	} else {
		scriptPath = filepath.Join(dir, "rclone")
		scriptContent = fmt.Sprintf(`#!/bin/sh
if [ "$RCLONE_MOCK_FAIL" = "1" ]; then
    echo "rclone error: something went wrong" >&2
    exit 1
fi
case "$1" in
  cat) cat "%s"; exit 0;;
  lsf) cat "%s"; exit 0;;
  *)   exit 0;;
esac
`, outputFile, outputFile)
	}

	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("create mock rclone: %v", err)
	}

	return scriptPath
}

func TestRcloneProvider_Name(t *testing.T) {
	p := NewRcloneProvider(nil, "myremote")
	if p.Name() != "rclone" {
		t.Errorf("Name() = %q, want rclone", p.Name())
	}
}

func TestRcloneProvider_Push(t *testing.T) {
	tmpDir := t.TempDir()
	rcloneBin := createMockRclone(t, tmpDir, "")

	p := &RcloneProvider{
		cfg:       nil,
		remote:    "myremote:backups",
		rcloneBin: rcloneBin,
	}

	id, err := p.Push([]byte("archive-data"), PushMeta{
		BackupID:  "20260605-120000",
		CreatedAt: time.Now(),
		Hostname:  "testbox",
	})
	if err != nil {
		t.Fatalf("Push: %v", err)
	}
	if id != "20260605-120000" {
		t.Errorf("id = %q, want 20260605-120000", id)
	}
}

func TestRcloneProvider_Push_NoRemote(t *testing.T) {
	p := &RcloneProvider{
		cfg:       nil,
		remote:    "",
		rcloneBin: "rclone",
	}
	_, err := p.Push([]byte("data"), PushMeta{BackupID: "test"})
	if err == nil {
		t.Fatal("expected error for missing remote")
	}
	if !strings.Contains(err.Error(), "remote is required") {
		t.Errorf("error = %v, want mention of remote", err)
	}
}

func TestRcloneProvider_Push_MissingBinary(t *testing.T) {
	p := &RcloneProvider{
		cfg:       nil,
		remote:    "myremote:path",
		rcloneBin: "rclone-nonexistent-xyz",
	}
	_, err := p.Push([]byte("data"), PushMeta{BackupID: "test"})
	if err == nil {
		t.Fatal("expected error for missing rclone binary")
	}
}

func TestRcloneProvider_Push_RcloneError(t *testing.T) {
	tmpDir := t.TempDir()
	rcloneBin := createMockRclone(t, tmpDir, "")

	// Set env var to trigger mock failure mode.
	t.Setenv("RCLONE_MOCK_FAIL", "1")

	p := &RcloneProvider{
		cfg:       nil,
		remote:    "myremote:backups",
		rcloneBin: rcloneBin,
	}

	_, err := p.Push([]byte("data"), PushMeta{BackupID: "test"})
	if err == nil {
		t.Fatal("expected error when rclone fails")
	}
	if !strings.Contains(err.Error(), "something went wrong") {
		t.Errorf("error = %v, want mention of stderr", err)
	}
}

func TestRcloneProvider_Pull(t *testing.T) {
	tmpDir := t.TempDir()
	rcloneBin := createMockRclone(t, tmpDir, "pulled-backup-data")

	p := &RcloneProvider{
		cfg:       nil,
		remote:    "myremote:backups",
		rcloneBin: rcloneBin,
	}

	data, err := p.Pull("20260605-120000")
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if string(data) != "pulled-backup-data" {
		t.Errorf("Pull data = %q, want pulled-backup-data", string(data))
	}
}

func TestRcloneProvider_Pull_NoRemote(t *testing.T) {
	p := &RcloneProvider{
		cfg:       nil,
		remote:    "",
		rcloneBin: "rclone",
	}
	_, err := p.Pull("some-id")
	if err == nil {
		t.Fatal("expected error for missing remote")
	}
}

func TestRcloneProvider_Pull_EmptyID(t *testing.T) {
	p := &RcloneProvider{
		cfg:       nil,
		remote:    "myremote:path",
		rcloneBin: "rclone",
	}
	_, err := p.Pull("")
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestRcloneProvider_Pull_MissingBinary(t *testing.T) {
	p := &RcloneProvider{
		cfg:       nil,
		remote:    "myremote:path",
		rcloneBin: "rclone-nonexistent-xyz",
	}
	_, err := p.Pull("some-id")
	if err == nil {
		t.Fatal("expected error for missing rclone binary")
	}
}

func TestRcloneProvider_List(t *testing.T) {
	tmpDir := t.TempDir()
	rcloneBin := createMockRclone(t, tmpDir, "20260605-120000.tar.gz\n20260604-100000.tar.gz")

	p := &RcloneProvider{
		cfg:       nil,
		remote:    "myremote:backups",
		rcloneBin: rcloneBin,
	}

	metas, err := p.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(metas) != 2 {
		t.Fatalf("List length = %d, want 2", len(metas))
	}
	if metas[0].ID != "20260605-120000" {
		t.Errorf("metas[0].ID = %q, want 20260605-120000", metas[0].ID)
	}
	if metas[1].ID != "20260604-100000" {
		t.Errorf("metas[1].ID = %q, want 20260604-100000", metas[1].ID)
	}
}

func TestRcloneProvider_List_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	rcloneBin := createMockRclone(t, tmpDir, "")

	p := &RcloneProvider{
		cfg:       nil,
		remote:    "myremote:backups",
		rcloneBin: rcloneBin,
	}

	metas, err := p.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(metas) != 0 {
		t.Errorf("List length = %d, want 0", len(metas))
	}
}

func TestRcloneProvider_List_NoRemote(t *testing.T) {
	p := &RcloneProvider{
		cfg:       nil,
		remote:    "",
		rcloneBin: "rclone",
	}
	_, err := p.List()
	if err == nil {
		t.Fatal("expected error for missing remote")
	}
}

func TestRcloneProvider_List_MissingBinary(t *testing.T) {
	p := &RcloneProvider{
		cfg:       nil,
		remote:    "myremote:path",
		rcloneBin: "rclone-nonexistent-xyz",
	}
	_, err := p.List()
	if err == nil {
		t.Fatal("expected error for missing rclone binary")
	}
}

func TestRcloneProvider_TokenResolution(t *testing.T) {
	p := NewRcloneProvider(nil, "myremote")
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.remote != "myremote" {
		t.Errorf("remote = %q, want myremote", p.remote)
	}
}

func TestRcloneProvider_ConfigRemote(t *testing.T) {
	p := NewRcloneProvider(nil, "gdrive:bak")
	if p.remote != "gdrive:bak" {
		t.Errorf("remote = %q, want gdrive:bak", p.remote)
	}
}

func TestNewRcloneProvider_DefaultBinary(t *testing.T) {
	p := NewRcloneProvider(nil, "myremote")
	if p.rcloneBin != "rclone" {
		t.Errorf("rcloneBin = %q, want rclone", p.rcloneBin)
	}
}
