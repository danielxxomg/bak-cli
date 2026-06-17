package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/actions"
	"github.com/danielxxomg/bak-cli/internal/restore"
)

// --- Root command structure tests ---

func TestRootCmd_Structure(t *testing.T) {
	if rootCmd.Use != "bak" {
		t.Errorf("Use = %q, want \"bak\"", rootCmd.Use)
	}
	if rootCmd.Short == "" {
		t.Error("root command should have a short description")
	}
	if rootCmd.Long == "" {
		t.Error("root command should have a long description")
	}
	if !rootCmd.SilenceUsage {
		t.Error("root command should SilenceUsage")
	}
	if !rootCmd.SilenceErrors {
		t.Error("root command should SilenceErrors")
	}
}

func TestRootCmd_HasSubcommands(t *testing.T) {
	expected := []string{
		"backup", "list", "restore", "export",
		"push", "pull", "login", "undo", "pick", "version", "profile",
	}

	registered := rootCmd.Commands()
	names := make(map[string]bool)
	for _, cmd := range registered {
		names[cmd.Name()] = true
	}

	for _, want := range expected {
		if !names[want] {
			t.Errorf("expected subcommand %q not registered on root", want)
		}
	}
}

func TestRootCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"--help"})
	rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "bak") {
		t.Fatal("help output should mention 'bak'")
	}
	if !strings.Contains(output, "backup") {
		t.Fatal("help output should mention 'backup'")
	}
}

// --- Backup command structure tests ---

func TestBackupCmd_Structure(t *testing.T) {
	cmd := findSubcommand(t, "backup")
	if cmd == nil {
		t.Fatal("backup subcommand not registered on root")
	}
	if cmd.Use != "backup" {
		t.Errorf("Use = %q, want \"backup\"", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("backup should have a short description")
	}
	if cmd.Long == "" {
		t.Error("backup should have a long description")
	}
	if cmd.RunE == nil {
		t.Error("backup RunE should be set")
	}
}

func TestBackupCmd_Flags(t *testing.T) {
	cmd := findSubcommand(t, "backup")
	if cmd == nil {
		t.Fatal("backup command not found")
	}

	presetFlag := cmd.Flags().Lookup("preset")
	if presetFlag == nil {
		t.Fatal("--preset flag not defined")
	}
	if presetFlag.DefValue != "quick" {
		t.Errorf("--preset default = %q, want \"quick\"", presetFlag.DefValue)
	}

	adapterFlag := cmd.Flags().Lookup("adapter")
	if adapterFlag == nil {
		t.Fatal("--adapter flag not defined")
	}
	if adapterFlag.DefValue != "" {
		t.Errorf("--adapter default = %q, want \"\"", adapterFlag.DefValue)
	}
}

func TestBackupCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"backup", "--help"})
	rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "backup") {
		t.Fatal("help output should mention 'backup'")
	}
	if !strings.Contains(output, "preset") {
		t.Fatal("help output should mention --preset")
	}
	if !strings.Contains(output, "adapter") {
		t.Fatal("help output should mention --adapter")
	}
	if !strings.Contains(output, "profile") {
		t.Fatal("help output should mention --profile")
	}
}

// --- Version command structure tests ---

func TestVersionCmd_Structure(t *testing.T) {
	cmd := findSubcommand(t, "version")
	if cmd == nil {
		t.Fatal("version subcommand not registered on root")
	}
	if cmd.Use != "version" {
		t.Errorf("Use = %q, want \"version\"", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("version should have a short description")
	}
	if cmd.Long == "" {
		t.Error("version should have a long description")
	}
	if cmd.Run == nil {
		t.Error("version Run should be set")
	}
}

func TestVersionCmd_Help(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	rootCmd.SetArgs([]string{"version", "--help"})
	rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "version") {
		t.Fatal("help output should mention 'version'")
	}
}

func TestVersionVariables(t *testing.T) {
	// Version, Commit, Date are build-time variables.
	// Verify they are initialized (even if to defaults).
	if Version == "" {
		t.Error("Version should not be empty (default is 'dev')")
	}
	if Commit == "" {
		t.Error("Commit should not be empty (default is 'unknown')")
	}
	if Date == "" {
		t.Error("Date should not be empty (default is 'unknown')")
	}
}

// --- Login command structure tests (supplement cloud_test.go) ---

func TestLoginCmd_Use(t *testing.T) {
	cmd := findSubcommand(t, "login")
	if cmd == nil {
		t.Fatal("login command not found")
	}
	if cmd.Use != "login" {
		t.Errorf("Use = %q, want \"login\"", cmd.Use)
	}
	if cmd.Long == "" {
		t.Error("login should have a long description")
	}
	if cmd.RunE == nil {
		t.Error("login RunE should be set")
	}
}

// --- formatSizeBytes tests ---

func TestFormatSizeBytes_FullRange(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"zero bytes", 0, "0 B"},
		{"1 byte", 1, "1 B"},
		{"500 bytes", 500, "500 B"},
		{"exactly 1 KB", 1024, "1.0 KB"},
		{"1536 bytes", 1536, "1.5 KB"},
		{"exactly 1 MB", 1048576, "1.0 MB"},
		{"1.5 MB", 1572864, "1.5 MB"},
		{"exactly 1 GB", 1073741824, "1.0 GB"},
		{"2.5 GB", 2684354560, "2.5 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := actions.FormatSizeBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("actions.FormatSizeBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

// --- formatSize tests (from backup.go) ---

func TestFormatSize_FullRange(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"zero bytes", 0, "0 B"},
		{"1 byte", 1, "1 B"},
		{"500 bytes", 500, "500 B"},
		{"1023 bytes", 1023, "1023 B"},
		{"exactly 1 KB", 1024, "1.0 KB"},
		{"1536 bytes", 1536, "1.5 KB"},
		{"exactly 1 MB", 1048576, "1.0 MB"},
		{"1.5 MB", 1572864, "1.5 MB"},
		{"exactly 1 GB", 1073741824, "1.0 GB"},
		{"exactly 1 TB", 1099511627776, "1.0 TB"},
		{"exactly 1 PB", 1125899906842624, "1.0 PB"},
		{"2 KB", 2048, "2.0 KB"},
		{"boundary below KB", 1023, "1023 B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSize(tt.bytes)
			if got != tt.want {
				t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

// --- countByStatus tests ---

func TestCountByStatus(t *testing.T) {
	tests := []struct {
		name   string
		diffs  []restore.FileDiff
		status restore.DiffStatus
		want   int
	}{
		{
			name:   "empty slice",
			diffs:  []restore.FileDiff{},
			status: restore.DiffNew,
			want:   0,
		},
		{
			name: "one match",
			diffs: []restore.FileDiff{
				{Status: restore.DiffNew, SourcePath: "/a"},
				{Status: restore.DiffModified, SourcePath: "/b"},
			},
			status: restore.DiffNew,
			want:   1,
		},
		{
			name: "multiple matches",
			diffs: []restore.FileDiff{
				{Status: restore.DiffNew, SourcePath: "/a"},
				{Status: restore.DiffNew, SourcePath: "/b"},
				{Status: restore.DiffModified, SourcePath: "/c"},
			},
			status: restore.DiffNew,
			want:   2,
		},
		{
			name: "no match",
			diffs: []restore.FileDiff{
				{Status: restore.DiffNew, SourcePath: "/a"},
				{Status: restore.DiffModified, SourcePath: "/b"},
			},
			status: restore.DiffMissing,
			want:   0,
		},
		{
			name: "all match unchanged",
			diffs: []restore.FileDiff{
				{Status: restore.DiffUnchanged, SourcePath: "/a"},
				{Status: restore.DiffUnchanged, SourcePath: "/b"},
				{Status: restore.DiffUnchanged, SourcePath: "/c"},
			},
			status: restore.DiffUnchanged,
			want:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := restore.CountByStatus(tt.diffs, tt.status)
			if got != tt.want {
				t.Errorf("restore.CountByStatus(diffs, %v) = %d, want %d", tt.status, got, tt.want)
			}
		})
	}
}

// --- List command supplementary tests ---

func TestListCmd_Use(t *testing.T) {
	if listCmd.Use != "list" {
		t.Errorf("Use = %q, want \"list\"", listCmd.Use)
	}
	if listCmd.RunE == nil {
		t.Error("list RunE should be set")
	}
}

// --- Export command supplementary tests ---

func TestExportCmd_Use(t *testing.T) {
	if exportCmd.Use != "export <backup-id>" {
		t.Errorf("Use = %q, want \"export <backup-id>\"", exportCmd.Use)
	}
	if exportCmd.RunE == nil {
		t.Error("export RunE should be set")
	}
}

// --- isValidBackupID supplementary edge cases ---

func TestIsValidBackupID_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{"dash in first position", "-20260604-15040", false},
		{"dash in middle", "20260604-150405", true},
		{"spaces", "20260604 150405", false},
		{"unicode digits", "20260604-１50405", false}, // fullwidth digits
		{"valid start of month", "20260101-000000", true},
		{"valid end of year", "20261231-235959", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := actions.IsValidBackupID(tt.id)
			if got != tt.want {
				t.Errorf("actions.IsValidBackupID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

// --- formatBackupIDError supplementary ---

func TestFormatBackupIDError_ContainsFormatHint(t *testing.T) {
	msg := actions.FormatBackupIDError("bad-id")
	if !strings.Contains(msg, "YYYYMMDD-HHMMSS") {
		t.Error("Error message should contain the expected format hint")
	}
	if !strings.Contains(msg, "bad-id") {
		t.Error("Error message should contain the invalid ID")
	}
	if !strings.Contains(msg, "20260604-150405") {
		t.Error("Error message should contain an example")
	}
}

// TestVersionIsNonEmpty verifies the Version variable is set.
func TestVersionIsNonEmpty(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

// --- Execute tests ---

func TestExecute_Help(t *testing.T) {
	// Test root command --help execution.
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"--help"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("--help should not error: %v", err)
	}
	output := bufOut.String()
	if !strings.Contains(output, "bak") {
		t.Error("help should contain 'bak'")
	}
}

func TestExecute_NoSubcommand(t *testing.T) {
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	// Running root without a subcommand or help flag should show help.
	rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("root execution without subcommand should not error: %v", err)
	}
}

func TestExecute_VerboseFlag(t *testing.T) {
	// The --verbose flag is added in the Execute() function (root.go).
	// Test that it works after execution.
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	// Add the persistent verbose flag like Execute() does.
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	rootCmd.SetArgs([]string{"--help"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("--help with verbose flag added should not error: %v", err)
	}
	output := bufOut.String()
	if !strings.Contains(output, "verbose") && !strings.Contains(output, "-v") {
		t.Log("verbose flag may not appear in help output for root")
	}
}

func TestExecute_UnknownCommand(t *testing.T) {
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"nonexistent_cmd_xyz"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("unknown command should produce error")
	}
}

// --- listBackups tests ---

func TestListBackups_WithManifests(t *testing.T) {
	bakDir := t.TempDir()
	backupsDir := bakDir + "/backups"

	// Create two backup directories with valid manifests.
	mustMkdirAll(t, backupsDir+"/20260617-120000")
	mustWriteFile(t, backupsDir+"/20260617-120000/manifest.json",
		`{
  "version": "0.3.0",
  "id": "20260617-120000",
  "created_at": "2026-06-17T12:00:00Z",
  "os_source": "linux",
  "hostname": "testbox",
  "bak_version": "0.1.0",
  "preset": "quick",
  "categories": ["config", "skills"],
  "adapters": {
    "opencode": {
      "config_dir": "~/.config/opencode",
      "items": []
    }
  },
  "secrets_excluded": false,
  "file_count": 10,
  "total_size": 1048576
}`)

	mustMkdirAll(t, backupsDir+"/20260617-150000")
	mustWriteFile(t, backupsDir+"/20260617-150000/manifest.json",
		`{
  "version": "0.3.0",
  "id": "20260617-150000",
  "created_at": "2026-06-17T15:00:00Z",
  "os_source": "linux",
  "hostname": "testbox",
  "bak_version": "0.1.0",
  "preset": "full",
  "categories": ["config", "skills", "vim"],
  "adapters": {
    "neovim": {
      "config_dir": "~/.config/nvim",
      "items": []
    },
    "ghostty": {
      "config_dir": "~/.config/ghostty",
      "items": []
    }
  },
  "secrets_excluded": false,
  "file_count": 25,
  "total_size": 3145728
}`)

	backups, err := listBackupsFrom(bakDir)
	if err != nil {
		t.Fatalf("listBackupsFrom: %v", err)
	}

	if len(backups) != 2 {
		t.Fatalf("len(backups) = %d, want 2", len(backups))
	}

	// First backup: 20260617-120000, 1 adapter, 1.0 MB
	b1 := backups[0]
	if b1.ID != "20260617-120000" {
		t.Errorf("backups[0].ID = %q, want %q", b1.ID, "20260617-120000")
	}
	if b1.Size != "1.0 MB" {
		t.Errorf("backups[0].Size = %q, want %q", b1.Size, "1.0 MB")
	}
	if b1.Status != "ok" {
		t.Errorf("backups[0].Status = %q, want %q", b1.Status, "ok")
	}
	if b1.Cloud != "opencode" {
		t.Errorf("backups[0].Cloud = %q, want %q", b1.Cloud, "opencode")
	}
	if b1.Date == "" {
		t.Error("backups[0].Date should not be empty")
	}

	// Second backup: 20260617-150000, 2 adapters, 3.0 MB
	b2 := backups[1]
	if b2.ID != "20260617-150000" {
		t.Errorf("backups[1].ID = %q, want %q", b2.ID, "20260617-150000")
	}
	if b2.Size != "3.0 MB" {
		t.Errorf("backups[1].Size = %q, want %q", b2.Size, "3.0 MB")
	}
	if b2.Status != "ok" {
		t.Errorf("backups[1].Status = %q, want %q", b2.Status, "ok")
	}

	// Check that date is formatted from ID (YYYYMMDD-HHMMSS → YYYY-MM-DD HH:MM:SS).
	expectedDate := "2026-06-17 15:00:00"
	if b2.Date != expectedDate {
		t.Errorf("backups[1].Date = %q, want %q", b2.Date, expectedDate)
	}
}

func TestListBackups_EmptyDir(t *testing.T) {
	bakDir := t.TempDir()

	backups, err := listBackupsFrom(bakDir)
	if err != nil {
		t.Fatalf("listBackupsFrom: %v", err)
	}

	// No backups dir → no backups (no error).
	if len(backups) != 0 {
		t.Errorf("len(backups) = %d, want 0", len(backups))
	}
}

func TestListBackups_SkipsNonDirs(t *testing.T) {
	bakDir := t.TempDir()
	backupsDir := bakDir + "/backups"

	mustMkdirAll(t, backupsDir)
	mustWriteFile(t, backupsDir+"/not-a-backup.txt", "some random file")

	backups, err := listBackupsFrom(bakDir)
	if err != nil {
		t.Fatalf("listBackupsFrom: %v", err)
	}

	if len(backups) != 0 {
		t.Errorf("len(backups) = %d, want 0 (non-dir entries skipped)", len(backups))
	}
}

func TestListBackups_SkipsCorruptManifest(t *testing.T) {
	bakDir := t.TempDir()
	backupsDir := bakDir + "/backups"

	mustMkdirAll(t, backupsDir+"/valid-backup")
	mustWriteFile(t, backupsDir+"/valid-backup/manifest.json",
		`{
  "version": "0.3.0",
  "id": "valid-backup",
  "created_at": "2026-06-17T12:00:00Z",
  "os_source": "linux",
  "bak_version": "0.1.0",
  "preset": "quick",
  "categories": [],
  "adapters": {},
  "secrets_excluded": false,
  "file_count": 0,
  "total_size": 0
}`)

	mustMkdirAll(t, backupsDir+"/corrupt-backup")
	mustWriteFile(t, backupsDir+"/corrupt-backup/manifest.json", "not valid json {{{")

	backups, err := listBackupsFrom(bakDir)
	if err != nil {
		t.Fatalf("listBackupsFrom: %v", err)
	}

	if len(backups) != 1 {
		t.Fatalf("len(backups) = %d, want 1 (corrupt manifest skipped)", len(backups))
	}
	if backups[0].ID != "valid-backup" {
		t.Errorf("backups[0].ID = %q, want %q", backups[0].ID, "valid-backup")
	}
}

// mustMkdirAll creates a directory or fails the test.
func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
}

// mustWriteFile writes data to a file or fails the test.
func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
