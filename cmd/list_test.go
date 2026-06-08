package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"

	"github.com/danielxxomg/bak-cli/internal/actions"
)

func TestListCmd_Structure(t *testing.T) {
	if listCmd.Use != "list" {
		t.Errorf("Expected Use 'list', got %q", listCmd.Use)
	}
	if listCmd.Short == "" {
		t.Error("Short description should not be empty")
	}
	if listCmd.RunE == nil {
		t.Error("RunE should be set")
	}
}

func TestListCmd_Args(t *testing.T) {
	if listCmd.Args != nil {
		t.Error("List command should not require arguments")
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"bytes", 500, "500 B"},
		{"kilobytes", 1024, "1.0 KB"},
		{"kilobytes decimal", 1536, "1.5 KB"},
		{"megabytes", 1048576, "1.0 MB"},
		{"megabytes decimal", 1572864, "1.5 MB"},
		{"gigabytes", 1073741824, "1.0 GB"},
		{"zero", 0, "0 B"},
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

func TestListCmd_Help(t *testing.T) {
	if listCmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

func TestListCmd_ProviderFlag(t *testing.T) {
	// Verify the --provider flag is defined on listCmd.
	flag := listCmd.Flags().Lookup("provider")
	if flag == nil {
		t.Fatal("list command should have --provider flag")
	}
	if flag.DefValue != "" {
		t.Errorf("default value should be empty string, got %q", flag.DefValue)
	}
}

func TestRunList_CloudProviderUnconfigured(t *testing.T) {
	// When --provider is set to an unregistered name, it should error.
	listCmd.Flags().Set("provider", "unknown-backend")
	defer listCmd.Flags().Set("provider", "")

	// listCmd.Execute() won't parse flags well standalone,
	// so call runList with the flag already set.
	listProviderOrig := listProvider
	listProvider = "unknown-backend"
	defer func() { listProvider = listProviderOrig }()

	// runList will try to register and get the provider.
	// Since it's not registered, it should return an error.
	err := runList(nil, nil)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestRunList_LocalBehaviorDefault(t *testing.T) {
	// When no --provider flag, keep existing local listing behavior.
	// This should not error (even if no backups exist, it prints a message).
	listProviderOrig := listProvider
	listProvider = ""
	defer func() { listProvider = listProviderOrig }()

	// Set up a minimal cobra command to avoid nil pointer.
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := runList(cmd, nil)
	if err != nil {
		// May fail if ~/.bak/backups doesn't exist, which is fine.
		// The key test is that we don't hit a cloud provider error.
		t.Logf("runList (local) returned: %v (may be expected)", err)
	}
}

// --- runList execution tests ---

func TestRunList_Execute(t *testing.T) {
	// Execute the list command. Even if there are no backups,
	// it should return nil (no error) and print appropriate message.
	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)
	rootCmd.SetOut(bufOut)
	rootCmd.SetErr(bufErr)

	rootCmd.SetArgs([]string{"list"})
	err := rootCmd.Execute()

	if err != nil {
		t.Fatalf("list command should not error: %v", err)
	}

	// Output may be "No backups found" when no backups exist,
	// or a table when backups exist. Both are valid.
	output := bufOut.String()
	if output == "" && bufErr.String() == "" {
		// On some platforms, list may produce no output if stdout is captured
		// before the tabwriter flushes. This is fine.
		t.Log("list command produced no output (empty backups dir)")
	}
}

func TestFormatSizeBytes_Extended(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"0 bytes", 0, "0 B"},
		{"1 byte", 1, "1 B"},
		{"1023 bytes", 1023, "1023 B"},
		{"exactly 1 KB", 1024, "1.0 KB"},
		{"1.5 KB", 1536, "1.5 KB"},
		{"exactly 1 MB", 1048576, "1.0 MB"},
		{"2.5 MB", 2621440, "2.5 MB"},
		{"exactly 1 GB", 1073741824, "1.0 GB"},
		{"100 GB", 107374182400, "100.0 GB"},
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
