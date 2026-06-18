package cmd

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/danielxxomg/bak-cli/internal/tui"
)

// TestRestorePicker_SelectsBackup verifies Enter sets SelectedID.
func TestRestorePicker_SelectsBackup(t *testing.T) {
	backups := []tui.BackupInfo{
		{ID: "20260617-120000", Date: "2026-06-17", Size: "1.0 MB"},
		{ID: "20260618-090000", Date: "2026-06-18", Size: "2.5 MB"},
	}
	m := restorePickerModel{backups: backups, cursor: 1}

	newM, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newM.(restorePickerModel)

	if !m.confirmed {
		t.Error("Enter should set confirmed=true")
	}
	if m.SelectedID() != "20260618-090000" {
		t.Errorf("SelectedID = %q, want %q", m.SelectedID(), "20260618-090000")
	}
}

// TestRestorePicker_EmptyList verifies empty backups list handled.
func TestRestorePicker_EmptyList(t *testing.T) {
	m := restorePickerModel{backups: nil}

	view := m.View()
	if !strings.Contains(view.Content, "No backups") {
		t.Error("empty list should show 'No backups' message")
	}

	newM, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newM.(restorePickerModel)
	if m.confirmed {
		t.Error("Enter on empty list should not confirm")
	}
}

// TestRestorePicker_Cancel verifies q/Esc cancels.
func TestRestorePicker_Cancel(t *testing.T) {
	backups := []tui.BackupInfo{
		{ID: "20260617-120000", Date: "2026-06-17", Size: "1.0 MB"},
	}
	m := restorePickerModel{backups: backups, cursor: 0}
	newM, _ := m.Update(tea.KeyPressMsg{Code: 'q'})
	m = newM.(restorePickerModel)

	if m.confirmed {
		t.Error("q should not confirm selection")
	}
	if m.SelectedID() != "" {
		t.Errorf("SelectedID after q = %q, want empty", m.SelectedID())
	}
}

// TestRestorePicker_CursorBounds verifies cursor cannot go out of bounds.
func TestRestorePicker_CursorBounds(t *testing.T) {
	backups := []tui.BackupInfo{
		{ID: "a", Date: "d1", Size: "1"},
		{ID: "b", Date: "d2", Size: "2"},
	}

	m := restorePickerModel{backups: backups, cursor: 0}
	newM, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m = newM.(restorePickerModel)
	if m.cursor != 0 {
		t.Errorf("cursor should stay 0, got %d", m.cursor)
	}

	m2 := restorePickerModel{backups: backups, cursor: 1}
	newM2, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m2 = newM2.(restorePickerModel)
	if m2.cursor != 1 {
		t.Errorf("cursor should stay 1, got %d", m2.cursor)
	}
}

// TestRestorePicker_NarrowTerminal shows "too small" message.
func TestRestorePicker_NarrowTerminal(t *testing.T) {
	m := restorePickerModel{width: 19, height: 9}
	view := m.View()

	if !strings.Contains(view.Content, "too small") {
		t.Error("narrow terminal should show 'too small' message")
	}
}
