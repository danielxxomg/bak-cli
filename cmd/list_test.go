package cmd

import (
	"testing"
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
