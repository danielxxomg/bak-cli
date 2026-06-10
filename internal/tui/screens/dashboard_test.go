// Package screens provides screen-specific render functions and sub-models
// for the bak-cli TUI. This file contains TDD tests written BEFORE the
// production code (strict RED phase) for the dashboard screen.
package screens

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// =============================================================================
// TestNewDashboardModel — RED (dashboard.go does not exist yet)
// =============================================================================

func TestNewDashboardModel(t *testing.T) {
	tests := []struct {
		name       string
		listFn     func() ([]BackupInfo, error)
		wantRows   int
		wantCursor int
		wantErrNil bool
	}{
		{
			name: "three backups loaded",
			listFn: func() ([]BackupInfo, error) {
				return []BackupInfo{
					{ID: "abc123", Date: "2024-01-15", Size: "1.2MB", Status: "ok", Cloud: "none"},
					{ID: "def456", Date: "2024-02-20", Size: "3.4MB", Status: "ok", Cloud: "gdrive"},
					{ID: "ghi789", Date: "2024-03-25", Size: "5.6MB", Status: "fail", Cloud: "none"},
				}, nil
			},
			wantRows:   3,
			wantCursor: 0,
			wantErrNil: true,
		},
		{
			name: "empty backup list",
			listFn: func() ([]BackupInfo, error) {
				return []BackupInfo{}, nil
			},
			wantRows:   0,
			wantCursor: 0,
			wantErrNil: true,
		},
		{
			name: "error from list function",
			listFn: func() ([]BackupInfo, error) {
				return nil, assertAnError("db connection refused")
			},
			wantRows:   0,
			wantCursor: 0,
			wantErrNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewDashboardModel(tt.listFn)

			if tt.wantErrNil && m.err != nil {
				t.Errorf("NewDashboardModel().err = %v, want nil", m.err)
			}
			if !tt.wantErrNil && m.err == nil {
				t.Error("NewDashboardModel().err = nil, want non-nil error")
			}

			rows := m.table.Rows()
			if len(rows) != tt.wantRows {
				t.Errorf("NewDashboardModel() table rows = %d, want %d", len(rows), tt.wantRows)
			}

			if m.table.Cursor() != tt.wantCursor {
				t.Errorf("NewDashboardModel() cursor = %d, want %d", m.table.Cursor(), tt.wantCursor)
			}
		})
	}
}

// =============================================================================
// TestDashboard_Update_NavigateDown — RED
// =============================================================================

func TestDashboard_Update_NavigateDown(t *testing.T) {
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return []BackupInfo{
			{ID: "1"}, {ID: "2"}, {ID: "3"},
		}, nil
	})
	m.width = 80
	m.height = 24

	// Press 'j' once: cursor should go from 0 to 1.
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	result := newModel.(DashboardModel)
	if result.table.Cursor() != 1 {
		t.Errorf("after first 'j': cursor = %d, want 1", result.table.Cursor())
	}

	// Press 'j' again: cursor should go to 2.
	newModel, _ = result.Update(tea.KeyPressMsg{Code: 'j'})
	result = newModel.(DashboardModel)
	if result.table.Cursor() != 2 {
		t.Errorf("after second 'j': cursor = %d, want 2", result.table.Cursor())
	}

	// Press 'j' once more: cursor should clamp at len-1 (2).
	newModel, _ = result.Update(tea.KeyPressMsg{Code: 'j'})
	result = newModel.(DashboardModel)
	if result.table.Cursor() != 2 {
		t.Errorf("after third 'j': cursor = %d, want 2 (clamped at max)", result.table.Cursor())
	}
}

// =============================================================================
// TestDashboard_Update_NavigateUp — RED
// =============================================================================

func TestDashboard_Update_NavigateUp(t *testing.T) {
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return []BackupInfo{
			{ID: "1"}, {ID: "2"}, {ID: "3"},
		}, nil
	})
	m.width = 80
	m.height = 24
	// Start at cursor 2.
	m.table.SetCursor(2)

	// Press 'k': cursor should go from 2 to 1.
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'k'})
	result := newModel.(DashboardModel)
	if result.table.Cursor() != 1 {
		t.Errorf("after first 'k': cursor = %d, want 1", result.table.Cursor())
	}

	// Press 'k' again: cursor should go to 0.
	newModel, _ = result.Update(tea.KeyPressMsg{Code: 'k'})
	result = newModel.(DashboardModel)
	if result.table.Cursor() != 0 {
		t.Errorf("after second 'k': cursor = %d, want 0", result.table.Cursor())
	}

	// Press 'k' once more: cursor should clamp at 0.
	newModel, _ = result.Update(tea.KeyPressMsg{Code: 'k'})
	result = newModel.(DashboardModel)
	if result.table.Cursor() != 0 {
		t.Errorf("after third 'k': cursor = %d, want 0 (clamped at min)", result.table.Cursor())
	}
}

// =============================================================================
// TestDashboard_Update_Back — RED
// =============================================================================

func TestDashboard_Update_Back(t *testing.T) {
	tests := []struct {
		name string
		code rune
	}{
		{"quit with q", 'q'},
		{"quit with esc", 27},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewDashboardModel(func() ([]BackupInfo, error) {
				return []BackupInfo{{ID: "1"}}, nil
			})

			_, cmd := m.Update(tea.KeyPressMsg{Code: tt.code})

			if cmd == nil {
				t.Fatal("Update() returned nil cmd, want ScreenBackMsg")
			}

			msg := cmd()
			if _, ok := msg.(ScreenBackMsg); !ok {
				t.Errorf("Update() returned %T, want ScreenBackMsg", msg)
			}
		})
	}
}

// =============================================================================
// TestDashboard_View_Populated — RED
// =============================================================================

func TestDashboard_View_Populated(t *testing.T) {
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return []BackupInfo{
			{ID: "abc123", Date: "2024-01-15", Size: "1.2MB", Status: "ok", Cloud: "none"},
			{ID: "def456", Date: "2024-02-20", Size: "3.4MB", Status: "ok", Cloud: "gdrive"},
			{ID: "ghi789", Date: "2024-03-25", Size: "5.6MB", Status: "fail", Cloud: "s3"},
		}, nil
	})
	m.width = 80
	m.height = 24

	output := m.View().Content

	// Output must contain backup data.
	wantContents := []string{
		"abc123", "2024-01-15", "1.2MB",
		"def456", "2024-02-20", "3.4MB",
		"ghi789", "2024-03-25", "5.6MB",
	}
	for _, want := range wantContents {
		if !strings.Contains(output, want) {
			t.Errorf("View() output missing %q", want)
		}
	}

	// Table column headers must be present.
	wantHeaders := []string{"ID", "Date", "Size", "Status", "Cloud"}
	for _, h := range wantHeaders {
		if !strings.Contains(output, h) {
			t.Errorf("View() output missing header %q", h)
		}
	}

	// Output must be non-empty.
	if len(output) == 0 {
		t.Error("View() returned empty string")
	}
}

// =============================================================================
// TestDashboard_View_EmptyState — RED
// =============================================================================

func TestDashboard_View_EmptyState(t *testing.T) {
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return []BackupInfo{}, nil
	})
	m.width = 80
	m.height = 24

	output := m.View().Content

	if !strings.Contains(output, "No backups found") {
		t.Errorf("View() output %q does not contain 'No backups found'", output)
	}

	if len(output) == 0 {
		t.Error("View() returned empty string for empty state")
	}
}

// =============================================================================
// TestDashboard_View_ErrorState — RED
// =============================================================================

func TestDashboard_View_ErrorState(t *testing.T) {
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return nil, assertAnError("connection refused")
	})
	m.width = 80
	m.height = 24

	output := m.View().Content

	if !strings.Contains(output, "Error") {
		t.Errorf("View() output %q does not contain 'Error'", output)
	}

	if !strings.Contains(output, "connection refused") {
		t.Errorf("View() output %q does not contain 'connection refused'", output)
	}

	if len(output) == 0 {
		t.Error("View() returned empty string for error state")
	}
}

// =============================================================================
// TRIANGULATION tests — second test case per behavior
// =============================================================================

// TestDashboard_Init_ReturnsNil verifies Init() has no initial side effects.
func TestDashboard_Init_ReturnsNil(t *testing.T) {
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return []BackupInfo{{ID: "1"}}, nil
	})

	cmd := m.Init()
	if cmd != nil {
		t.Errorf("Init() returned non-nil cmd %v, want nil", cmd)
	}
}

// TestDashboard_NavigateDown_EmptyList verifies that navigation on empty
// list is a no-op (cursor stays at -1, the bubbles table default for no rows).
func TestDashboard_NavigateDown_EmptyList(t *testing.T) {
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return []BackupInfo{}, nil
	})

	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	result := newModel.(DashboardModel)

	// With 0 rows, bubbles/table starts cursor at -1 and it stays there.
	if result.table.Cursor() != -1 {
		t.Errorf("navigate down on empty list: cursor = %d, want -1", result.table.Cursor())
	}
}

// TestDashboard_Update_WindowSize verifies WindowSizeMsg updates dimensions.
func TestDashboard_Update_WindowSize(t *testing.T) {
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return []BackupInfo{{ID: "1"}}, nil
	})

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	result := newModel.(DashboardModel)

	if result.width != 120 {
		t.Errorf("WindowSize width = %d, want 120", result.width)
	}
	if result.height != 40 {
		t.Errorf("WindowSize height = %d, want 40", result.height)
	}
}

// TestDashboard_View_NarrowTerminal verifies output is non-empty on narrow terminals.
func TestDashboard_View_NarrowTerminal(t *testing.T) {
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return []BackupInfo{{ID: "abc123", Date: "2024-01-15", Size: "1.2MB", Status: "ok", Cloud: "none"}}, nil
	})
	m.width = 30
	m.height = 10

	output := m.View().Content

	if len(output) == 0 {
		t.Error("View() returned empty string on narrow terminal")
	}
}

// TestDashboard_View_SingleRow verifies a single backup row renders correctly.
func TestDashboard_View_SingleRow(t *testing.T) {
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return []BackupInfo{
			{ID: "single01", Date: "2024-06-01", Size: "0.5MB", Status: "ok", Cloud: "dropbox"},
		}, nil
	})
	m.width = 80
	m.height = 24

	output := m.View().Content

	// Must contain the single row data and headers.
	if !strings.Contains(output, "single01") {
		t.Error("View() missing backup ID 'single01'")
	}
	if !strings.Contains(output, "dropbox") {
		t.Error("View() missing cloud provider 'dropbox'")
	}
	if !strings.Contains(output, "0.5MB") {
		t.Error("View() missing size '0.5MB'")
	}
}

// =============================================================================
// Error type used in tests — matches NewDashboardModel tests.
// =============================================================================

// assertAnError returns an error with the given message, used throughout
// this file to simulate ListBackups errors.
type assertAnError string

func (e assertAnError) Error() string { return string(e) }
