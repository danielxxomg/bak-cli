//go:build windows

package schedule

import (
	"testing"
)

// --- Schtasks CSV parsing ---

func TestParseSchtasksCSV_ValidEntries(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	output := `"bak-cli-work","2026-06-06 02:00:00","Ready"
"bak-cli-home","2026-06-06 03:00:00","Ready"`
	entries := parseSchtasksCSV(output)
	if len(entries) != 2 {
		t.Fatalf("parseSchtasksCSV returned %d entries, want 2", len(entries))
	}
	if entries[0].Profile != "work" {
		t.Errorf("entry[0].Profile = %q, want 'work'", entries[0].Profile)
	}
	if entries[1].Profile != "home" {
		t.Errorf("entry[1].Profile = %q, want 'home'", entries[1].Profile)
	}
}

func TestParseSchtasksCSV_EmptyOutput(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	entries := parseSchtasksCSV("")
	if len(entries) != 0 {
		t.Errorf("parseSchtasksCSV(empty) = %d entries, want 0", len(entries))
	}
}

func TestParseSchtasksCSV_NonBakCLI(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	output := `"SomeTask","2026-06-06 02:00:00","Ready"
"AnotherTask","2026-06-06 04:00:00","Ready"`
	entries := parseSchtasksCSV(output)
	if len(entries) != 0 {
		t.Errorf("parseSchtasksCSV on non-bak-cli CSV = %d entries, want 0", len(entries))
	}
}

func TestParseSchtasksCSV_MixedEntries(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	output := `"SomeTask","2026-06-06 02:00:00","Ready"
"bak-cli-dev","2026-06-06 08:00:00","Ready"
"AnotherTask","2026-06-06 04:00:00","Ready"`
	entries := parseSchtasksCSV(output)
	if len(entries) != 1 {
		t.Fatalf("parseSchtasksCSV with mixed = %d entries, want 1", len(entries))
	}
	if entries[0].Profile != "dev" {
		t.Errorf("Profile = %q, want 'dev'", entries[0].Profile)
	}
}

func TestParseSchtasksCSV_SingleColumn(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// Malformed CSV with only task name, no extra columns.
	output := `"bak-cli-test"`
	entries := parseSchtasksCSV(output)
	if len(entries) != 1 {
		t.Fatalf("parseSchtasksCSV with single column = %d entries, want 1", len(entries))
	}
	if entries[0].Profile != "test" {
		t.Errorf("Profile = %q, want 'test'", entries[0].Profile)
	}
}

func TestParseSchtasksCSV_WhitespaceTrim(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	output := "\n\"bak-cli-ws\"\n\n"
	entries := parseSchtasksCSV(output)
	if len(entries) != 1 {
		t.Fatalf("parseSchtasksCSV with whitespace = %d entries, want 1", len(entries))
	}
	if entries[0].Profile != "ws" {
		t.Errorf("Profile = %q, want 'ws'", entries[0].Profile)
	}
}
