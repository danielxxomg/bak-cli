// Package screens provides screen-specific render functions and sub-models
// for the bak-cli TUI. This file contains TDD tests written BEFORE the
// production code (strict RED phase) for the dashboard screen.
package screens

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
)

// =============================================================================
// TestNewDashboardModel — RED (dashboard.go does not exist yet)
// =============================================================================

func TestNewDashboardModel(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

func TestDashboard_Update_NavigateDown(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestDashboard_Update_NavigateUp(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestDashboard_Update_Back(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name string
		code rune
	}{
		{"quit with q", 'q'},
		{"quit with esc", 27},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
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

func TestDashboard_View_Populated(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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

func TestDashboard_View_EmptyState(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return []BackupInfo{}, nil
	})
	m.width = 80
	m.height = 24

	output := m.View().Content

	if !strings.Contains(output, "No backups yet") {
		t.Errorf("View() output %q does not contain 'No backups yet'", output)
	}

	if len(output) == 0 {
		t.Error("View() returned empty string for empty state")
	}
}

// TestDashboard_View_EmptyState_Styled verifies the empty dashboard renders the
// shared styled empty-state block (icon + message + hint) via
// components.RenderEmptyState, not a bare string (tui-personality REQ-TP-007).
// The hint presence is the behavioral proof: a bare message string has none.
func TestDashboard_View_EmptyState_Styled(t *testing.T) { //nolint:paralleltest // matches established codebase convention across all tui tests
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return []BackupInfo{}, nil
	})
	m.width = 80
	m.height = 24

	output := m.View().Content

	if !strings.Contains(output, "No backups yet") {
		t.Errorf("styled empty state missing message 'No backups yet': %q", output)
	}
	if !strings.Contains(output, "bak backup") {
		t.Errorf("styled empty state missing hint 'bak backup': %q", output)
	}
}

// =============================================================================
// TestDashboard_View_ErrorState — RED
// =============================================================================

func TestDashboard_View_ErrorState(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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
func TestDashboard_Init_ReturnsNil(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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
func TestDashboard_NavigateDown_EmptyList(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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
func TestDashboard_Update_WindowSize(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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
func TestDashboard_View_NarrowTerminal(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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
func TestDashboard_View_SingleRow(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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
// Phase 4: Mouse Navigation (PR 2 — Tier 2b) — RED
// =============================================================================

// TestDashboard_MouseWheel_ScrollsList verifies the mouse wheel advances and
// retreats the table cursor (REQ-TP-006: wheel scrolls the list).
func TestDashboard_MouseWheel_ScrollsList(t *testing.T) { //nolint:paralleltest // matches established codebase convention across all tui tests
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return []BackupInfo{{ID: "1"}, {ID: "2"}, {ID: "3"}}, nil
	})
	m.width = 80
	m.height = 24

	// Wheel down advances the cursor (0 -> 1).
	before := m.table.Cursor()
	nm, _ := m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	r := nm.(DashboardModel)
	if r.table.Cursor() <= before {
		t.Errorf("wheel down did not advance cursor: %d -> %d", before, r.table.Cursor())
	}

	// Wheel up retreats the cursor (1 -> 0).
	before = r.table.Cursor()
	nm, _ = r.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	r = nm.(DashboardModel)
	if r.table.Cursor() >= before {
		t.Errorf("wheel up did not retreat cursor: %d -> %d", before, r.table.Cursor())
	}
}

// TestDashboard_MouseClick_SelectsRow verifies a left click moves the cursor
// to the clicked row (REQ-TP-006: click selects the clicked row).
func TestDashboard_MouseClick_SelectsRow(t *testing.T) { //nolint:paralleltest // matches established codebase convention across all tui tests
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return []BackupInfo{{ID: "1"}, {ID: "2"}, {ID: "3"}}, nil
	})
	m.width = 80
	m.height = 24

	tests := []struct {
		name    string
		y       int
		wantCur int
	}{
		{"click first data row", 2, 0},
		{"click second data row", 3, 1},
		{"click third data row", 4, 2},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) {
			nm, _ := m.Update(tea.MouseClickMsg{Button: tea.MouseLeft, Y: tt.y})
			r := nm.(DashboardModel)
			if r.table.Cursor() != tt.wantCur {
				t.Errorf("click at Y=%d: cursor = %d, want %d", tt.y, r.table.Cursor(), tt.wantCur)
			}
		})
	}
}

// TestDashboard_MouseWheel_ClampsAtBounds verifies wheel scrolling clamps at
// the first/last row instead of going out of bounds.
func TestDashboard_MouseWheel_ClampsAtBounds(t *testing.T) { //nolint:paralleltest // matches established codebase convention across all tui tests
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return []BackupInfo{{ID: "1"}, {ID: "2"}, {ID: "3"}}, nil
	})
	m.width = 80
	m.height = 24

	// Wheel up from the top stays at 0.
	nm, _ := m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	r := nm.(DashboardModel)
	if r.table.Cursor() != 0 {
		t.Errorf("wheel up at top: cursor = %d, want 0", r.table.Cursor())
	}

	// Wheel down past the last row clamps at the last row (index 2).
	for i := 0; i < 10; i++ {
		nm, _ = r.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
		r = nm.(DashboardModel)
	}
	if r.table.Cursor() != 2 {
		t.Errorf("wheel down past end: cursor = %d, want 2 (clamped)", r.table.Cursor())
	}
}

// TestDashboard_SetFilter_MatchingRows verifies that SetFilter with a
// matching query returns only rows containing that substring (case-insensitive).
func TestDashboard_SetFilter_MatchingRows(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return []BackupInfo{
			{ID: "conf-1", Date: "2024-01-01", Size: "1MB", Status: "ok", Cloud: "none"},
			{ID: "abc-2", Date: "2024-02-01", Size: "2MB", Status: "ok", Cloud: "gdrive"},
			{ID: "CONFIG-3", Date: "2024-03-01", Size: "3MB", Status: "fail", Cloud: "s3"},
		}, nil
	})

	m.SetFilter("conf")

	rows := m.table.Rows()
	if len(rows) != 2 {
		t.Fatalf("SetFilter('conf') returned %d rows, want 2", len(rows))
	}

	// Verify matching rows (case-insensitive) and non-matching row excluded.
	ids := rowIDs(rows)
	if !contains(ids, "conf-1") {
		t.Error("SetFilter('conf') missing 'conf-1'")
	}
	if !contains(ids, "CONFIG-3") {
		t.Error("SetFilter('conf') missing 'CONFIG-3' (case-insensitive)")
	}
	if contains(ids, "abc-2") {
		t.Error("SetFilter('conf') should NOT contain 'abc-2'")
	}
}

// TestDashboard_SetFilter_EmptyRestoresAll verifies that SetFilter("")
// restores all original rows after a previous filter was applied.
func TestDashboard_SetFilter_EmptyRestoresAll(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return []BackupInfo{
			{ID: "a-1", Date: "2024-01-01", Size: "1MB", Status: "ok", Cloud: "none"},
			{ID: "b-2", Date: "2024-02-01", Size: "2MB", Status: "ok", Cloud: "gdrive"},
			{ID: "c-3", Date: "2024-03-01", Size: "3MB", Status: "ok", Cloud: "s3"},
		}, nil
	})

	// Apply a filter first.
	m.SetFilter("b-2")
	if len(m.table.Rows()) != 1 {
		t.Fatalf("after SetFilter('b-2'): rows = %d, want 1", len(m.table.Rows()))
	}

	// Clearing filter must restore all 3 original rows.
	m.SetFilter("")
	rows := m.table.Rows()
	if len(rows) != 3 {
		t.Errorf("SetFilter('') returned %d rows, want 3 (all restored)", len(rows))
	}
	ids := rowIDs(rows)
	for _, want := range []string{"a-1", "b-2", "c-3"} {
		if !contains(ids, want) {
			t.Errorf("SetFilter('') missing %q", want)
		}
	}
}

// TestDashboard_SetFilter_NoMatch verifies that SetFilter with a query
// that matches nothing produces an empty row set.
func TestDashboard_SetFilter_NoMatch(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return []BackupInfo{
			{ID: "alpha", Date: "2024-01-01", Size: "1MB", Status: "ok", Cloud: "none"},
			{ID: "beta", Date: "2024-02-01", Size: "2MB", Status: "ok", Cloud: "gdrive"},
		}, nil
	})

	m.SetFilter("xyz")

	rows := m.table.Rows()
	if len(rows) != 0 {
		t.Errorf("SetFilter('xyz') returned %d rows, want 0 (no matches)", len(rows))
	}
}

// rowIDs extracts the first column (ID) from each row.
func rowIDs(rows []table.Row) []string {
	ids := make([]string, len(rows))
	for i, r := range rows {
		if len(r) > 0 {
			ids[i] = r[0]
		}
	}
	return ids
}

// contains reports whether s is in ss.
func contains(ss []string, s string) bool {
	for _, item := range ss {
		if item == s {
			return true
		}
	}
	return false
}

// =============================================================================
// TestDashboard_View_HelpBar_Populated — RED (Phase 3: help bar persistence)
// =============================================================================

func TestDashboard_View_HelpBar_Populated(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return []BackupInfo{
			{ID: "abc123", Date: "2024-01-15", Size: "1.2MB", Status: "ok", Cloud: "none"},
		}, nil
	})
	m.width = 80
	m.height = 24

	output := m.View().Content

	// Help bar: ↑/↓ navigate • / search • q back
	if !strings.Contains(output, "navigate") {
		t.Errorf("populated dashboard help bar missing 'navigate': %q", output)
	}
	if !strings.Contains(output, "search") {
		t.Errorf("populated dashboard help bar missing 'search': %q", output)
	}
	if !strings.Contains(output, "back") {
		t.Errorf("populated dashboard help bar missing 'back': %q", output)
	}
}

// =============================================================================
// TestDashboard_View_HelpBar_Empty — RED (Phase 3: help bar persistence)
// =============================================================================

func TestDashboard_View_HelpBar_Empty(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return []BackupInfo{}, nil
	})
	m.width = 80
	m.height = 24

	output := m.View().Content

	// Must show empty state AND help bar.
	if !strings.Contains(output, "No backups yet") {
		t.Errorf("empty dashboard missing 'No backups yet': %q", output)
	}
	if !strings.Contains(output, "navigate") {
		t.Errorf("empty dashboard help bar missing 'navigate': %q", output)
	}
	if !strings.Contains(output, "back") {
		t.Errorf("empty dashboard help bar missing 'back': %q", output)
	}
}

// =============================================================================
// TestDashboard_View_HelpBar_Error — RED (Phase 3: help bar persistence)
// =============================================================================

func TestDashboard_View_HelpBar_Error(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewDashboardModel(func() ([]BackupInfo, error) {
		return nil, assertAnError("connection refused")
	})
	m.width = 80
	m.height = 24

	output := m.View().Content

	// Must show error AND help bar.
	if !strings.Contains(output, "Error") {
		t.Errorf("error dashboard missing 'Error': %q", output)
	}
	if !strings.Contains(output, "navigate") {
		t.Errorf("error dashboard help bar missing 'navigate': %q", output)
	}
	if !strings.Contains(output, "back") {
		t.Errorf("error dashboard help bar missing 'back': %q", output)
	}
}

// =============================================================================
// Error type used in tests — matches NewDashboardModel tests.
// =============================================================================

// assertAnError returns an error with the given message, used throughout
// this file to simulate ListBackups errors.
type assertAnError string

func (e assertAnError) Error() string { return string(e) }

// =============================================================================
// TestDashboard_View_MinSizeGuard — threshold guard at 40×12 (Phase 4)
// =============================================================================

func TestDashboard_View_MinSizeGuard(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name     string
		width    int
		height   int
		tooSmall bool
	}{
		{"below width (29x20)", 29, 20, true},
		{"below height (60x14)", 60, 14, true},
		{"both below (20x10)", 20, 10, true},
		{"exactly min (30x15)", 30, 15, false},
		{"above min (80x24)", 80, 24, false},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			m := NewDashboardModel(func() ([]BackupInfo, error) {
				return []BackupInfo{}, nil
			})
			m.width = tt.width
			m.height = tt.height

			output := m.View().Content

			if tt.tooSmall {
				if !strings.Contains(output, "Terminal too small") {
					t.Errorf("View() %dx%d: expected 'Terminal too small', got %q",
						tt.width, tt.height, output)
				}
			} else {
				if strings.Contains(output, "Terminal too small") {
					t.Errorf("View() %dx%d: got 'Terminal too small', expected normal content",
						tt.width, tt.height)
				}
			}
		})
	}
}
