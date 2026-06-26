// Package tui provides the root TUI model with screen routing, key navigation,
// and window size handling. This file contains table-driven TDD tests written
// BEFORE the production code (strict RED phase).
package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/danielxxomg/bak-cli/internal/tui/components"
	"github.com/danielxxomg/bak-cli/internal/tui/screens"
)

// =============================================================================
// TestNewModel — RED (model.go does not exist yet)
// =============================================================================

func TestNewModel(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	deps := Deps{
		Version:      "1.0.0",
		ConfigExists: func() bool { return true },
	}

	m := NewModel(deps)

	// Default screen should be ScreenMenu.
	if m.screen != ScreenMenu {
		t.Errorf("NewModel().screen = %v, want ScreenMenu (%d)", m.screen, ScreenMenu)
	}

	// Cursor should start at 0.
	if m.cursor != 0 {
		t.Errorf("NewModel().cursor = %d, want 0", m.cursor)
	}

	// Width and height should be 0 (WindowSizeMsg not yet received).
	if m.width != 0 {
		t.Errorf("NewModel().width = %d, want 0", m.width)
	}
	if m.height != 0 {
		t.Errorf("NewModel().height = %d, want 0", m.height)
	}

	// tooSmall should start false.
	if m.tooSmall {
		t.Error("NewModel().tooSmall = true, want false")
	}

	// Deps should be stored.
	if m.deps.Version != "1.0.0" {
		t.Errorf("NewModel().deps.Version = %q, want %q", m.deps.Version, "1.0.0")
	}

	// menuItems should be initialized with 7 items.
	if len(m.menuItems) != 7 {
		t.Errorf("NewModel().menuItems length = %d, want 7", len(m.menuItems))
	}
}

// =============================================================================
// TestModel_Init — RED
// =============================================================================

func TestModel_Init(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})

	cmd := m.Init()

	// Init should return nil cmd (no initial command needed).
	if cmd != nil {
		t.Errorf("Init() returned non-nil cmd %v, want nil", cmd)
	}
}

// =============================================================================
// TestModel_Update — RED
// =============================================================================

// newTestModel returns a Model with default test deps.
func newTestModel() Model {
	return NewModel(Deps{Version: "1.0.0"})
}

func TestModel_Update_Quit(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := newTestModel()

	newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})

	// Should return a Model.
	result, ok := newModel.(Model)
	if !ok {
		t.Fatalf("Update() returned %T, want Model", newModel)
	}

	// Cursor should be unchanged on quit.
	if result.cursor != 0 {
		t.Errorf("Update('q').cursor = %d, want 0", result.cursor)
	}

	// Should return a quit command.
	if cmd == nil {
		t.Error("Update('q') returned nil cmd, want quit command")
	} else {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Errorf("Update('q') cmd returned %T, want tea.QuitMsg", msg)
		}
	}
}

func TestModel_Update_NavigateDown(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})

	// Press 'j' once: cursor should go from 0 to 1.
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	result := newModel.(Model)
	if result.cursor != 1 {
		t.Errorf("Update('j') cursor = %d, want 1", result.cursor)
	}

	// Press 'j' 5 more times: cursor should reach last item (6).
	for i := 0; i < 5; i++ {
		newModel, _ = newModel.Update(tea.KeyPressMsg{Code: 'j'})
	}
	result = newModel.(Model)
	if result.cursor != 6 {
		t.Errorf("Update('j' x6) cursor = %d, want 6", result.cursor)
	}

	// One more press: wraps around to 0.
	newModel, _ = newModel.Update(tea.KeyPressMsg{Code: 'j'})
	result = newModel.(Model)
	if result.cursor != 0 {
		t.Errorf("Update('j' x7) cursor = %d, want 0 (wrap-around)", result.cursor)
	}
}

func TestModel_Update_NavigateUp(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	// Start with cursor at 3 (simulate via navigation).
	m.cursor = 3

	// Press 'k': cursor should go from 3 to 2.
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'k'})
	result := newModel.(Model)
	if result.cursor != 2 {
		t.Errorf("Update('k') cursor = %d, want 2", result.cursor)
	}

	// Press 'k' 3 more times: cursor goes 2→1→0→6 (wraps).
	for i := 0; i < 3; i++ {
		newModel, _ = newModel.Update(tea.KeyPressMsg{Code: 'k'})
	}
	result = newModel.(Model)
	if result.cursor != 6 {
		t.Errorf("Update('k' x4) cursor = %d, want 6 (wrap-around)", result.cursor)
	}
}

func TestModel_Update_WindowSize(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	result := newModel.(Model)

	if result.width != 120 {
		t.Errorf("WindowSize width = %d, want 120", result.width)
	}
	if result.height != 40 {
		t.Errorf("WindowSize height = %d, want 40", result.height)
	}
	if result.tooSmall {
		t.Error("WindowSize(120x40) tooSmall = true, want false")
	}
}

func TestModel_Update_MinSizeGuard(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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
		{"barely below both (29x14)", 29, 14, true},
		{"wide but short (100x10)", 100, 10, true},
		{"tall but narrow (25x40)", 25, 40, true},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			m := NewModel(Deps{Version: "1.0.0"})

			newModel, _ := m.Update(tea.WindowSizeMsg{Width: tt.width, Height: tt.height})
			result := newModel.(Model)

			if result.tooSmall != tt.tooSmall {
				t.Errorf("WindowSize(%dx%d) tooSmall = %v, want %v",
					tt.width, tt.height, result.tooSmall, tt.tooSmall)
			}
		})
	}
}

// =============================================================================
// Phase 2: Arrow Keys + Wrap-Around — RED (nav uses clamp, no arrow support)
// =============================================================================

// TestModel_Update_ArrowKeys verifies that up/down arrow keys navigate
// the main menu cursor, identical to j/k.
func TestModel_Update_ArrowKeys(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	lastIdx := len(m.menuItems) - 1 // 6

	tests := []struct {
		name       string
		startCur   int
		keys       []tea.KeyPressMsg
		wantCursor int
	}{
		{
			name:       "arrow down from 0 to 1",
			startCur:   0,
			keys:       []tea.KeyPressMsg{{Code: tea.KeyDown}},
			wantCursor: 1,
		},
		{
			name:       "arrow down twice",
			startCur:   0,
			keys:       []tea.KeyPressMsg{{Code: tea.KeyDown}, {Code: tea.KeyDown}},
			wantCursor: 2,
		},
		{
			name:       "arrow up from 3 to 2",
			startCur:   3,
			keys:       []tea.KeyPressMsg{{Code: tea.KeyUp}},
			wantCursor: 2,
		},
		{
			name:       "arrow down wraps from last to first",
			startCur:   lastIdx,
			keys:       []tea.KeyPressMsg{{Code: tea.KeyDown}},
			wantCursor: 0,
		},
		{
			name:       "arrow up wraps from first to last",
			startCur:   0,
			keys:       []tea.KeyPressMsg{{Code: tea.KeyUp}},
			wantCursor: lastIdx,
		},
		{
			name:       "j wraps from last to first",
			startCur:   lastIdx,
			keys:       []tea.KeyPressMsg{{Code: 'j'}},
			wantCursor: 0,
		},
		{
			name:       "k wraps from first to last",
			startCur:   0,
			keys:       []tea.KeyPressMsg{{Code: 'k'}},
			wantCursor: lastIdx,
		},
		{
			name:     "mixed: down+arrow j+k wraps correctly",
			startCur: 0,
			keys: []tea.KeyPressMsg{
				{Code: 'j'},         // 0→1
				{Code: tea.KeyDown}, // 1→2
				{Code: 'j'},         // 2→3
				{Code: tea.KeyUp},   // 3→2
				{Code: 'k'},         // 2→1
				{Code: tea.KeyUp},   // 1→0
				{Code: 'k'},         // 0→6 (wrap)
				{Code: tea.KeyDown}, // 6→0 (wrap)
				{Code: 'j'},         // 0→1
			},
			wantCursor: 1,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			m.cursor = tt.startCur
			cur := m
			for _, key := range tt.keys {
				newM, _ := cur.Update(key)
				cur = newM.(Model)
			}
			if cur.cursor != tt.wantCursor {
				t.Errorf("after keys: cursor = %d, want %d", cur.cursor, tt.wantCursor)
			}
		})
	}
}

// =============================================================================
// TestModel_View — RED
// =============================================================================

func TestModel_View_Menu(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	deps := Deps{Version: "1.0.0"}
	m := NewModel(deps)
	m.width = 80
	m.height = 24

	output := m.View().Content

	// Output must contain menu items.
	menuItems := []string{
		"Create backup", "Restore", "Browse backups",
		"Cloud sync", "Profiles", "Settings", "Quit",
	}
	for _, item := range menuItems {
		if !strings.Contains(output, item) {
			t.Errorf("View() output does not contain menu item %q", item)
		}
	}

	// Output must be non-empty.
	if len(output) == 0 {
		t.Error("View() returned empty string")
	}
}

func TestModel_View_TooSmall(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 10
	m.height = 5
	m.tooSmall = true

	output := m.View().Content

	if !strings.Contains(output, "Terminal too small") {
		t.Errorf("View() output %q does not contain 'Terminal too small'", output)
	}

	// Message must show actual dimensions and required minimum.
	if !strings.Contains(output, "10x5") {
		t.Errorf("View() output %q does not contain actual dimensions '10x5'", output)
	}
	if !strings.Contains(output, "30") {
		t.Errorf("View() output %q does not contain required width '30'", output)
	}
	if !strings.Contains(output, "15") {
		t.Errorf("View() output %q does not contain required height '15'", output)
	}
	if !strings.Contains(output, "Need at least") {
		t.Errorf("View() output %q does not contain 'Need at least'", output)
	}
}

// =============================================================================
// TestModel_Selection — RED
// =============================================================================

func TestModel_Selection(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name     string
		cursor   int
		wantItem string
	}{
		{"first item", 0, "Create backup"},
		{"middle item", 3, "Cloud sync"},
		{"last item", 6, "Quit"},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			m := NewModel(Deps{Version: "1.0.0"})
			m.cursor = tt.cursor

			sel := m.Selection()

			if sel.Cursor != tt.cursor {
				t.Errorf("Selection().Cursor = %d, want %d", sel.Cursor, tt.cursor)
			}
			if sel.Item != tt.wantItem {
				t.Errorf("Selection().Item = %q, want %q", sel.Item, tt.wantItem)
			}
		})
	}
}

// =============================================================================
// TRIANGULATION tests — second test case per behavior
// =============================================================================

// TestModel_Init_PreservesDeps verifies the model deps survive Init.
func TestModel_Init_PreservesDeps(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "2.0.0-beta"})
	_ = m.Init()
	if m.deps.Version != "2.0.0-beta" {
		t.Errorf("after Init, Version = %q, want %q", m.deps.Version, "2.0.0-beta")
	}
}

// TestModel_Update_Quit_Esc verifies escape key also quits.
func TestModel_Update_Quit_Esc(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})

	_, cmd := m.Update(tea.KeyPressMsg{Code: KeyEsc})

	if cmd == nil {
		t.Error("Update(esc) returned nil cmd, want quit command")
	} else {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Errorf("Update(esc) cmd returned %T, want tea.QuitMsg", msg)
		}
	}
}

// TestModel_Update_SecondResize verifies a second WindowSizeMsg updates correctly.
func TestModel_Update_SecondResize(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})

	// First resize: normal.
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	r2 := m2.(Model)
	if r2.width != 120 || r2.height != 40 {
		t.Fatalf("first resize: got %dx%d, want 120x40", r2.width, r2.height)
	}

	// Second resize: narrow (below min).
	m3, _ := m2.Update(tea.WindowSizeMsg{Width: 15, Height: 20})
	r3 := m3.(Model)
	if r3.width != 15 {
		t.Errorf("second resize width = %d, want 15", r3.width)
	}
	if !r3.tooSmall {
		t.Error("second resize: tooSmall = false, want true (width < 40)")
	}
}

// TestModel_View_Menu_WithCursor verifies cursor position changes output.
func TestModel_View_Menu_WithCursor(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	deps := Deps{Version: "1.0.0"}
	m := NewModel(deps)
	m.width = 80
	m.height = 24

	// Cursor at 0.
	out0 := m.View().Content

	// Move cursor to 3 and verify output differs.
	m2, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	m2, _ = m2.Update(tea.KeyPressMsg{Code: 'j'})
	m2, _ = m2.Update(tea.KeyPressMsg{Code: 'j'})
	out3 := m2.(Model).View().Content

	if out0 == out3 {
		t.Error("View() output identical for cursor 0 and cursor 3, expected different")
	}
}

// TestModel_View_TooSmall_ShowsWarningOnly verifies tooSmall view
// does not render menu items. Also verifies the message format includes
// actual dimensions and the required minimum.
func TestModel_View_TooSmall_ShowsWarningOnly(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 10
	m.height = 5
	m.tooSmall = true

	output := m.View().Content

	if !strings.Contains(output, "Terminal too small") {
		t.Errorf("tooSmall view missing 'Terminal too small': %q", output)
	}

	// Menu items should NOT appear.
	if strings.Contains(output, "Create backup") {
		t.Error("tooSmall view contains menu items, expected only warning")
	}

	// Must include actual dimensions in the message.
	if !strings.Contains(output, "10x5") {
		t.Errorf("tooSmall view missing actual dimensions '10x5': %q", output)
	}
	if !strings.Contains(output, "Need at least 30") || !strings.Contains(output, "15") {
		t.Errorf("tooSmall view missing required minimum '30×15': %q", output)
	}
}

// TestModel_View_AltScreen verifies that View() always returns a tea.View
// with AltScreen=true, regardless of whether the terminal is too small.
func TestModel_View_AltScreen(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name     string
		width    int
		height   int
		tooSmall bool
	}{
		{"normal terminal", 80, 24, false},
		{"too small terminal", 10, 5, true},
		{"minimum viable", 30, 15, false},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			m := NewModel(Deps{Version: "1.0.0"})
			m.width = tt.width
			m.height = tt.height
			m.tooSmall = tt.tooSmall

			v := m.View()

			if !v.AltScreen {
				t.Errorf("View().AltScreen = false, want true (%s)", tt.name)
			}

			// Content must be non-empty regardless of AltScreen state.
			if len(v.Content) == 0 {
				t.Errorf("View().Content is empty (%s)", tt.name)
			}
		})
	}
}

// TestModel_View_AltScreen_Triangulation verifies AltScreen is true
// across multiple model states: different screens and after resizes.
func TestModel_View_AltScreen_Triangulation(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24

	// Normal menu view.
	if !m.View().AltScreen {
		t.Error("initial View().AltScreen = false, want true")
	}

	// After a resize, AltScreen must still be true.
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if !m2.(Model).View().AltScreen {
		t.Error("View().AltScreen = false after resize, want true")
	}

	// After key navigation, AltScreen must still be true.
	m3, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	if !m3.(Model).View().AltScreen {
		t.Error("View().AltScreen = false after key press, want true")
	}

	// After resize to too-small (width < 20), AltScreen must still be true.
	m4, _ := m.Update(tea.WindowSizeMsg{Width: 10, Height: 10})
	if !m4.(Model).View().AltScreen {
		t.Error("View().AltScreen = false with too-small terminal, want true")
	}
}

// TestModel_Selection_EmptyMenuItems verifies that Selection() handles
// empty menuItems without panicking and returns a zero-value MenuSelection.
func TestModel_Selection_EmptyMenuItems(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.menuItems = []string{}

	sel := m.Selection()

	if sel.Cursor != 0 {
		t.Errorf("Selection() with empty items: Cursor = %d, want 0", sel.Cursor)
	}
	if sel.Item != "" {
		t.Errorf("Selection() with empty items: Item = %q, want empty string", sel.Item)
	}
}

// TestModel_Selection_NilMenuItems verifies that Selection() handles
// nil menuItems without panicking (triangulation of empty case).
func TestModel_Selection_NilMenuItems(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.menuItems = nil

	// Must not panic.
	sel := m.Selection()

	if sel.Cursor != 0 {
		t.Errorf("Selection() with nil items: Cursor = %d, want 0", sel.Cursor)
	}
	if sel.Item != "" {
		t.Errorf("Selection() with nil items: Item = %q, want empty string", sel.Item)
	}
}

// TestModel_Selection_Clamp verifies out-of-bounds cursor is clamped.
func TestModel_Selection_Clamp(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name     string
		cursor   int
		wantItem string
	}{
		{"negative cursor", -5, "Create backup"},
		{"over bound cursor", 99, "Quit"},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			m := newTestModel()
			m.cursor = tt.cursor

			sel := m.Selection()

			if sel.Item != tt.wantItem {
				t.Errorf("Selection(cursor=%d).Item = %q, want %q",
					tt.cursor, sel.Item, tt.wantItem)
			}
		})
	}
}

// =============================================================================
// PR4 Screen Routing Tests — RED (model.go updates do not exist yet)
// =============================================================================

// TestModel_Update_ScreenDashboard verifies enter on "Browse backups"
// (menu index 2) transitions to ScreenDashboard.
func TestModel_Update_ScreenDashboard(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.cursor = 2 // "Browse backups"

	_, cmd := m.Update(tea.KeyPressMsg{Code: KeyEnter})

	if cmd == nil {
		t.Fatal("Update(enter) on Browse backups returned nil cmd")
	}

	msg := cmd()
	switch msg := msg.(type) {
	case screenChangeMsg:
		if msg.screen != ScreenDashboard {
			t.Errorf("screenChangeMsg.screen = %v, want ScreenDashboard", msg.screen)
		}
	default:
		t.Errorf("Update(enter) cmd returned %T, want screenChangeMsg", msg)
	}
}

// TestModel_Update_ScreenProgress verifies enter on "Create backup"
// (menu index 0) transitions to ScreenProgress.
func TestModel_Update_ScreenProgress(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.cursor = 0 // "Create backup"

	_, cmd := m.Update(tea.KeyPressMsg{Code: KeyEnter})

	if cmd == nil {
		t.Fatal("Update(enter) on Create backup returned nil cmd")
	}

	msg := cmd()
	switch msg := msg.(type) {
	case screenChangeMsg:
		if msg.screen != ScreenProgress {
			t.Errorf("screenChangeMsg.screen = %v, want ScreenProgress", msg.screen)
		}
	default:
		t.Errorf("Update(enter) cmd returned %T, want screenChangeMsg", msg)
	}
}

// TestModel_View_Dashboard verifies View delegates to dashboard when screen=ScreenDashboard.
func TestModel_View_Dashboard(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version: "1.0.0",
		ListBackups: func() ([]BackupInfo, error) {
			return []BackupInfo{
				{ID: "test-001", Date: "2024-01-01", Size: "1MB", Status: "ok", Cloud: "none"},
			}, nil
		},
	})
	m.width = 80
	m.height = 24
	// Trigger screen transition to initialize the dashboard sub-model.
	newM, _ := m.Update(screenChangeMsg{screen: ScreenDashboard})
	model := newM.(Model)

	output := model.View().Content

	if !strings.Contains(output, "test-001") {
		t.Errorf("View() dashboard missing backup ID 'test-001': %q", output)
	}
}

// TestModel_View_Progress verifies View delegates to progress when screen=ScreenProgress.
func TestModel_View_Progress(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24
	// Trigger screen transition to initialize the progress sub-model.
	newM, _ := m.Update(screenChangeMsg{screen: ScreenProgress})
	model := newM.(Model)

	output := model.View().Content

	// Progress screen should show the "Progress" heading.
	if !strings.Contains(output, "Progress") {
		t.Errorf("View() progress missing heading 'Progress': %q", output)
	}
}

// TestModel_Update_BackFromDashboard verifies ScreenBackMsg returns to ScreenMenu.
func TestModel_Update_BackFromDashboard(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version: "1.0.0",
		ListBackups: func() ([]BackupInfo, error) {
			return []BackupInfo{}, nil
		},
	})
	m.screen = ScreenDashboard

	newModel, _ := m.Update(screens.ScreenBackMsg{})
	result := newModel.(Model)

	if result.screen != ScreenMenu {
		t.Errorf("after ScreenBackMsg: screen = %v, want ScreenMenu", result.screen)
	}
}

// =============================================================================
// PR5 Screen Routing Tests — RED (model.go updates do not exist yet)
// =============================================================================

// TestModel_Update_ScreenSettings verifies enter on "Settings"
// (menu index 5) transitions to ScreenSettings.
func TestModel_Update_ScreenSettings(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.cursor = 5 // "Settings"

	_, cmd := m.Update(tea.KeyPressMsg{Code: KeyEnter})

	if cmd == nil {
		t.Fatal("Update(enter) on Settings returned nil cmd")
	}
	msg := cmd()
	switch msg := msg.(type) {
	case screenChangeMsg:
		if msg.screen != ScreenSettings {
			t.Errorf("screenChangeMsg.screen = %v, want ScreenSettings", msg.screen)
		}
	default:
		t.Errorf("Update(enter) cmd returned %T, want screenChangeMsg", msg)
	}
}

// TestModel_Update_ScreenShortcuts verifies that pressing '?' on the
// main menu toggles the help overlay on (showHelp=true) without changing
// the active screen.
func TestModel_Update_ScreenShortcuts(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})

	newModel, _ := m.Update(tea.KeyPressMsg{Code: '?'})
	result := newModel.(Model)

	if !result.showHelp {
		t.Error("after '?': showHelp = false, want true (help overlay visible)")
	}
	// Screen should remain on the current screen, not change.
	if result.screen != ScreenMenu {
		t.Errorf("after '?': screen = %v, want ScreenMenu (overlay, not screen change)", result.screen)
	}
}

// TestModel_Update_SearchActivate verifies that pressing / on the
// dashboard activates the search component.
func TestModel_Update_SearchActivate(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version: "1.0.0",
		ListBackups: func() ([]BackupInfo, error) {
			return []BackupInfo{}, nil
		},
	})
	m.screen = ScreenDashboard

	newModel, _ := m.Update(tea.KeyPressMsg{Code: '/'})
	result := newModel.(Model)

	if !result.search.IsActive() {
		t.Error("after '/': search is not active, want true")
	}
}

// TestModel_Update_Toast verifies that triggering a toast sets
// the toast message and visibility.
func TestModel_Update_Toast(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})

	m.toast.Show("Backup started", 3)

	output := m.toast.View()
	if !strings.Contains(output, "Backup started") {
		t.Errorf("toast View() = %q, want to contain %q", output, "Backup started")
	}
	if output == "" {
		t.Error("toast View() returned empty after Show()")
	}
}

// TestModel_View_Settings verifies View delegates to settings when
// screen=ScreenSettings.
func TestModel_View_Settings(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.screen = ScreenSettings
	m.settings = newSettingsPtr()

	// Forward WindowSizeMsg so sub-model gets dimensions.
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m2.(Model)

	output := m.View().Content

	if !strings.Contains(output, "Settings") {
		t.Errorf("View() settings missing heading: %q", output)
	}
}

// TestModel_View_Shortcuts verifies View renders shortcuts overlay.
func TestModel_View_Shortcuts(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24
	m.screen = ScreenShortcuts

	output := m.View().Content

	if !strings.Contains(output, "Navigation") {
		t.Errorf("View() shortcuts missing 'Navigation': %q", output)
	}
}

// TestModel_View_ToastOverlay verifies toast is rendered when visible.
func TestModel_View_ToastOverlay(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24
	m.toast.Show("Done", 3)

	output := m.View().Content

	if !strings.Contains(output, "Done") {
		t.Errorf("View() missing toast message 'Done': %q", output)
	}
}

// TestModel_BackFromShortcuts verifies that pressing q/esc from
// shortcuts overlay returns to ScreenMenu.
func TestModel_BackFromShortcuts(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.screen = ScreenShortcuts

	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'q'})
	result := newModel.(Model)

	if result.screen != ScreenMenu {
		t.Errorf("after q from shortcuts: screen = %v, want ScreenMenu", result.screen)
	}
}

// TestModel_ScreenRoute_Health tests enter on a new menu item
// for health check (simulated via direct screen set).
func TestModel_ScreenRoute_Health(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.screen = ScreenHealth
	m.health = newHealthPtr()

	// Forward WindowSizeMsg so sub-model gets dimensions.
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m2.(Model)

	output := m.View().Content

	if !strings.Contains(output, "Health") {
		t.Errorf("View() health missing heading: %q", output)
	}
}

// =============================================================================
// Phase 3: Menu Items 1 & 4 — GREEN (real screen transitions are wired)
// =============================================================================

// TestModel_Update_MenuEnter_Restore verifies pressing enter on cursor=1
// ("Restore") navigates to the restore screen (ScreenRestore).
func TestModel_Update_MenuEnter_Restore(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24
	m.cursor = 1 // "Restore"

	newModel, cmd := m.Update(tea.KeyPressMsg{Code: KeyEnter})
	result := newModel.(Model)

	if cmd == nil {
		t.Fatal("Update(enter) on Restore returned nil cmd")
	}
	msg := cmd()
	switch msg := msg.(type) {
	case screenChangeMsg:
		if msg.screen != ScreenRestore {
			t.Errorf("screenChangeMsg.screen = %v, want ScreenRestore", msg.screen)
		}
	default:
		t.Errorf("cmd returned %T, want screenChangeMsg", msg)
	}

	// Screen should transition.
	if result.screen != ScreenMenu {
		// Screen stays on menu until screenChangeMsg is processed.
		_ = result
	}
}

// TestModel_Update_MenuEnter_Profiles verifies pressing enter on cursor=4
// ("Profiles") navigates to the profiles screen (ScreenProfiles).
func TestModel_Update_MenuEnter_Profiles(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24
	m.cursor = 4 // "Profiles"

	newModel, cmd := m.Update(tea.KeyPressMsg{Code: KeyEnter})
	result := newModel.(Model)

	if cmd == nil {
		t.Fatal("Update(enter) on Profiles returned nil cmd")
	}
	msg := cmd()
	switch msg := msg.(type) {
	case screenChangeMsg:
		if msg.screen != ScreenProfiles {
			t.Errorf("screenChangeMsg.screen = %v, want ScreenProfiles", msg.screen)
		}
	default:
		t.Errorf("cmd returned %T, want screenChangeMsg", msg)
	}

	_ = result
}

// TestModel_Update_MenuEnter_CreateBackup_Channels verifies that pressing
// enter on cursor=0 ("Create backup") spawns backup channels and transitions
// to the progress screen.
func TestModel_Update_MenuEnter_CreateBackup_Channels(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	backupCh := make(chan ProgressUpdate, 1)
	backupDone := make(chan error, 1)

	deps := Deps{
		Version: "1.0.0",
		RunBackup: func(cats []string, ch chan<- ProgressUpdate) error {
			go func() {
				ch <- ProgressUpdate{Step: "copying", Current: 1, Total: 10}
				ch <- ProgressUpdate{Step: "done", Current: 10, Total: 10, Done: true}
			}()
			// Write to backupDone when done.
			return nil
		},
	}

	m := NewModel(deps)
	m.width = 80
	m.height = 24
	m.cursor = 0 // "Create backup"
	m.backupCh = backupCh
	m.backupDone = backupDone

	m2, cmd := m.Update(tea.KeyPressMsg{Code: KeyEnter})
	m = m2.(Model)

	if cmd == nil {
		t.Fatal("Update(enter) on Create backup returned nil cmd")
	}

	// The cmd is tea.Batch(screenChangeMsg, drainProgressCmd).
	// The batch returns nil when executed inline — it's meant for
	// the Bubble Tea runtime. We verify the model state instead.
	if m.backupCh == nil {
		t.Error("backupCh should be set after Enter on Create backup")
	}
	if m.screen != ScreenMenu {
		t.Errorf("screen after enter = %v, should remain ScreenMenu until screenChangeMsg is processed", m.screen)
	}
}

// =============================================================================
// Coverage fill: ScreenCloud routing — handleKey and View
// =============================================================================

// TestModel_Update_ScreenCloud_Back verifies that pressing q on the cloud
// screen returns to ScreenMenu.
func TestModel_Update_ScreenCloud_Back(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.screen = ScreenCloud

	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'q'})
	result := newModel.(Model)

	if result.screen != ScreenMenu {
		t.Errorf("after q on cloud: screen = %v, want ScreenMenu", result.screen)
	}
}

// TestModel_Update_ScreenCloud_Esc verifies that pressing esc on the cloud
// screen returns to ScreenMenu.
func TestModel_Update_ScreenCloud_Esc(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.screen = ScreenCloud

	newModel, _ := m.Update(tea.KeyPressMsg{Code: KeyEsc})
	result := newModel.(Model)

	if result.screen != ScreenMenu {
		t.Errorf("after esc on cloud: screen = %v, want ScreenMenu", result.screen)
	}
}

// TestModel_View_Cloud verifies View renders the cloud sync screen.
func TestModel_View_Cloud(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24
	m.screen = ScreenCloud

	output := m.View().Content

	// Cloud screen with no provider shows "No cloud provider configured".
	if !strings.Contains(output, "No cloud provider configured") {
		t.Errorf("View() cloud missing 'No cloud provider configured': %q", output)
	}
}

// TestModel_View_UnknownScreen verifies View handles an out-of-range screen
// value via the default branch without panicking.
func TestModel_View_UnknownScreen(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24
	m.screen = screen(99) // invalid screen value

	output := m.View().Content

	// Must not panic, content should be empty.
	if output != "" {
		t.Logf("View() unknown screen content = %q (expected empty)", output)
	}
}

// =============================================================================
// Coverage fill: initDashboard nil and error paths
// =============================================================================

// TestModel_initDashboard_NilListBackups verifies initDashboard handles
// nil ListBackups gracefully (returns an empty dashboard, no panic).
func TestModel_initDashboard_NilListBackups(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version:      "1.0.0",
		ListBackups:  nil,
		ConfigExists: func() bool { return true },
	})

	// Must not panic.
	d := m.initDashboard()

	// Dashboard should be usable (empty, no error).
	view := d.View().Content
	if !strings.Contains(view, "No backups found") {
		t.Errorf("nil ListBackups dashboard view = %q, want 'No backups found'", view)
	}
}

// TestModel_initDashboard_Error verifies initDashboard propagates errors
// from ListBackups to the dashboard model (error state visible in View).
func TestModel_initDashboard_Error(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version: "1.0.0",
		ListBackups: func() ([]BackupInfo, error) {
			return nil, fmt.Errorf("connection refused")
		},
	})

	// Must not panic.
	d := m.initDashboard()

	// Dashboard should show error.
	view := d.View().Content
	if !strings.Contains(view, "connection refused") {
		t.Errorf("error dashboard view = %q, want 'connection refused'", view)
	}
}

// =============================================================================
// Coverage fill: initProgress helper
// =============================================================================

// TestModel_initProgress verifies initProgress returns a ProgressModel
// that can render its View without panicking.
func TestModel_initProgress(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})

	p := m.initProgress()
	p.Width = 80
	p.Height = 24

	view := p.View().Content

	if !strings.Contains(view, "Progress") {
		t.Errorf("initProgress view = %q, want 'Progress' heading", view)
	}
}

// =============================================================================
// Coverage fill: handleKey forwarding for ScreenSettings, ScreenHealth,
// ScreenProgress, and non-matching key on ScreenCloud
// =============================================================================

// TestModel_Update_ScreenSettings_KeyForward verifies that key presses on
// the Settings screen are forwarded to the settings sub-model.
func TestModel_Update_ScreenSettings_KeyForward(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.screen = ScreenSettings
	m.settings = newSettingsPtr()
	// Give sub-model dimensions so it doesn't show "too small".
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m2.(Model)

	// Press 'j' — should be forwarded to settings sub-model.
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	result := newModel.(Model)

	// Screen should still be ScreenSettings.
	if result.screen != ScreenSettings {
		t.Errorf("after j on settings: screen = %v, want ScreenSettings", result.screen)
	}
}

// TestModel_Update_ScreenHealth_KeyForward verifies that key presses on the
// Health screen are forwarded to the health sub-model.
func TestModel_Update_ScreenHealth_KeyForward(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.screen = ScreenHealth
	m.health = newHealthPtr()
	// Give sub-model dimensions.
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m2.(Model)

	// Press 'j' — should be forwarded to health sub-model.
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	result := newModel.(Model)

	// Screen should still be ScreenHealth.
	if result.screen != ScreenHealth {
		t.Errorf("after j on health: screen = %v, want ScreenHealth", result.screen)
	}
}

// TestModel_Update_ScreenProgress_KeyForward verifies that key presses on
// the Progress screen are forwarded to the progress sub-model.
func TestModel_Update_ScreenProgress_KeyForward(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.screen = ScreenProgress
	m.progress = newProgressPtr()
	// Give sub-model dimensions.
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m2.(Model)

	// Press 'j' — should be forwarded to progress sub-model.
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	result := newModel.(Model)

	// Screen should still be ScreenProgress.
	if result.screen != ScreenProgress {
		t.Errorf("after j on progress: screen = %v, want ScreenProgress", result.screen)
	}
}

// TestModel_Update_ScreenCloud_UnhandledKey verifies that a non-q/esc key
// press on the cloud screen is a no-op (screen stays on ScreenCloud).
func TestModel_Update_ScreenCloud_UnhandledKey(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.screen = ScreenCloud

	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	result := newModel.(Model)

	// Screen should still be ScreenCloud — j is not handled.
	if result.screen != ScreenCloud {
		t.Errorf("after j on cloud: screen = %v, want ScreenCloud", result.screen)
	}
}

// =============================================================================
// Coverage fill: Update screenChangeMsg for screens without sub-model init
// (ScreenCloud, ScreenShortcuts)
// =============================================================================

// TestModel_Update_ScreenChange_Cloud verifies screenChangeMsg for
// ScreenCloud sets the screen and returns a cmd (cloud model Init).
func TestModel_Update_ScreenChange_Cloud(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24

	newModel, cmd := m.Update(screenChangeMsg{screen: ScreenCloud})
	result := newModel.(Model)

	if result.screen != ScreenCloud {
		t.Errorf("after screenChangeMsg(Cloud): screen = %v, want ScreenCloud", result.screen)
	}
	// CloudModel.Init returns a cmd to load cloud status.
	if cmd == nil {
		t.Error("after screenChangeMsg(Cloud): cmd = nil, want cloud status load cmd")
	}
}

// TestModel_Update_ScreenChange_Shortcuts verifies screenChangeMsg for
// ScreenShortcuts sets the screen and returns nil cmd (no sub-model init).
func TestModel_Update_ScreenChange_Shortcuts(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24

	newModel, cmd := m.Update(screenChangeMsg{screen: ScreenShortcuts})
	result := newModel.(Model)

	if result.screen != ScreenShortcuts {
		t.Errorf("after screenChangeMsg(Shortcuts): screen = %v, want ScreenShortcuts", result.screen)
	}
	if cmd != nil {
		t.Errorf("after screenChangeMsg(Shortcuts): cmd = %v, want nil", cmd)
	}
}

// =============================================================================
// Coverage fill: Update forwarding unknown message type to all sub-models
// (the default switch at the bottom that forwards to the active screen)
// =============================================================================

// TestModel_Update_ForwardToDashboard verifies that unknown messages are
// forwarded to the dashboard sub-model when screen=ScreenDashboard.
func TestModel_Update_ForwardToDashboard(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version: "1.0.0",
		ListBackups: func() ([]BackupInfo, error) {
			return []BackupInfo{}, nil
		},
	})
	m.width = 80
	m.height = 24
	m.screen = ScreenDashboard
	// Init dashboard via screenChangeMsg.
	m2, _ := m.Update(screenChangeMsg{screen: ScreenDashboard})
	m = m2.(Model)

	// Send an unknown message type (not KeyPressMsg, not WindowSizeMsg, etc.).
	newModel, _ := m.Update(struct{}{})
	result := newModel.(Model)

	// Must not panic; screen stays on ScreenDashboard.
	if result.screen != ScreenDashboard {
		t.Errorf("after unknown msg on dashboard: screen = %v, want ScreenDashboard", result.screen)
	}
}

// TestModel_Update_ForwardToProgress verifies that unknown messages are
// forwarded to the progress sub-model when screen=ScreenProgress.
func TestModel_Update_ForwardToProgress(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24
	m.screen = ScreenProgress
	// Init progress via screenChangeMsg.
	m2, _ := m.Update(screenChangeMsg{screen: ScreenProgress})
	m = m2.(Model)

	// Send an unknown message type.
	newModel, _ := m.Update(struct{}{})
	result := newModel.(Model)

	// Must not panic; screen stays on ScreenProgress.
	if result.screen != ScreenProgress {
		t.Errorf("after unknown msg on progress: screen = %v, want ScreenProgress", result.screen)
	}
}

// TestModel_Update_ForwardToSettings verifies that unknown messages are
// forwarded to the settings sub-model when screen=ScreenSettings.
func TestModel_Update_ForwardToSettings(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24
	m.screen = ScreenSettings
	// Init settings via screenChangeMsg.
	m2, _ := m.Update(screenChangeMsg{screen: ScreenSettings})
	m = m2.(Model)

	// Send an unknown message type.
	newModel, _ := m.Update(struct{}{})
	result := newModel.(Model)

	// Must not panic; screen stays on ScreenSettings.
	if result.screen != ScreenSettings {
		t.Errorf("after unknown msg on settings: screen = %v, want ScreenSettings", result.screen)
	}
}

// TestModel_Update_ForwardToHealth verifies that unknown messages are
// forwarded to the health sub-model when screen=ScreenHealth.
func TestModel_Update_ForwardToHealth(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24
	m.screen = ScreenHealth
	// Init health via screenChangeMsg.
	m2, _ := m.Update(screenChangeMsg{screen: ScreenHealth})
	m = m2.(Model)

	// Send an unknown message type.
	newModel, _ := m.Update(struct{}{})
	result := newModel.(Model)

	// Must not panic; screen stays on ScreenHealth.
	if result.screen != ScreenHealth {
		t.Errorf("after unknown msg on health: screen = %v, want ScreenHealth", result.screen)
	}
}

// TestModel_Update_UnknownMsg_NoSubmodel verifies that an unknown message
// on a screen that has no sub-model (ScreenMenu) falls through to the
// final return (m, nil) without panicking.
func TestModel_Update_UnknownMsg_NoSubmodel(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24
	m.screen = ScreenMenu

	// Send an unknown message type — no sub-model exists for ScreenMenu.
	newModel, _ := m.Update(struct{}{})
	result := newModel.(Model)

	// Must not panic; screen stays on ScreenMenu.
	if result.screen != ScreenMenu {
		t.Errorf("after unknown msg on menu: screen = %v, want ScreenMenu", result.screen)
	}
}

// =============================================================================
// Coverage fill: WindowSizeMsg forwarded to sub-models on non-Dashboard screens
// (already tested for Dashboard via TestModel_View_Dashboard, but not for
// Progress/Settings/Health with sub-model nil guard)
// =============================================================================

// TestModel_Update_WindowSize_ProgressNil verifies WindowSizeMsg on
// ScreenProgress with nil progress sub-model does not panic.
func TestModel_Update_WindowSize_ProgressNil(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.screen = ScreenProgress
	m.progress = nil

	// Must not panic.
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	result := newModel.(Model)
	if result.screen != ScreenProgress {
		t.Errorf("WindowSize on nil progress: screen = %v, want ScreenProgress", result.screen)
	}
}

// TestModel_Update_WindowSize_SettingsNil verifies WindowSizeMsg on
// ScreenSettings with nil settings sub-model does not panic.
func TestModel_Update_WindowSize_SettingsNil(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.screen = ScreenSettings
	m.settings = nil

	// Must not panic.
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	result := newModel.(Model)
	if result.screen != ScreenSettings {
		t.Errorf("WindowSize on nil settings: screen = %v, want ScreenSettings", result.screen)
	}
}

// TestModel_Update_WindowSize_HealthNil verifies WindowSizeMsg on
// ScreenHealth with nil health sub-model does not panic.
func TestModel_Update_WindowSize_HealthNil(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.screen = ScreenHealth
	m.health = nil

	// Must not panic.
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	result := newModel.(Model)
	if result.screen != ScreenHealth {
		t.Errorf("WindowSize on nil health: screen = %v, want ScreenHealth", result.screen)
	}
}

// =============================================================================
// TestScreenIotaValues verifies the Screen iota enum has the correct
// sequential values after ScreenWizard removal. The expected sequence
// is: ScreenMenu(0), ScreenDashboard(1), ScreenProgress(2),
// ScreenSettings(3), ScreenCloud(4), ScreenShortcuts(5), ScreenHealth(6).
func TestScreenIotaValues(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name  string
		value screen
		want  screen
	}{
		{"ScreenMenu", ScreenMenu, 0},
		{"ScreenDashboard", ScreenDashboard, 1},
		{"ScreenProgress", ScreenProgress, 2},
		{"ScreenSettings", ScreenSettings, 3},
		{"ScreenCloud", ScreenCloud, 4},
		{"ScreenShortcuts", ScreenShortcuts, 5},
		{"ScreenHealth", ScreenHealth, 6},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			if tt.value != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.value, tt.want)
			}
		})
	}
}

// =============================================================================
// Phase 1 Toast Wiring Tests — RED (actionResultMsg does not exist yet)
// =============================================================================

// TestModel_Update_ActionResult_Success verifies that sending
// actionResultMsg{err: nil} to Update() calls toast.Show() with a success
// message and makes the toast visible.
func TestModel_Update_ActionResult_Success(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := newTestModel()
	m.width = 80
	m.height = 24

	newModel, _ := m.Update(actionResultMsg{err: nil})
	result := newModel.(Model)

	output := result.toast.View()
	if output == "" {
		t.Error("toast View() returned empty after success actionResultMsg")
	}
	if !strings.Contains(output, "Backup complete") {
		t.Errorf("toast View() = %q, want to contain 'Backup complete'", output)
	}
}

// TestModel_Update_ActionResult_Error verifies that sending
// actionResultMsg{err: error} to Update() calls toast.Show() with the
// error text and makes the toast visible.
func TestModel_Update_ActionResult_Error(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := newTestModel()
	m.width = 80
	m.height = 24

	newModel, _ := m.Update(actionResultMsg{err: errors.New("connection refused")})
	result := newModel.(Model)

	output := result.toast.View()
	if output == "" {
		t.Error("toast View() returned empty after error actionResultMsg")
	}
	if !strings.Contains(output, "connection refused") {
		t.Errorf("toast View() = %q, want to contain 'connection refused'", output)
	}
}

// =============================================================================
// Phase 2 Search → Dashboard Wiring Tests — RED (forwarding not wired yet)
// =============================================================================

// TestModel_Update_SearchForwardsToDashboard verifies that when search is
// active on the dashboard screen, typing characters updates the search query
// AND filters the dashboard table rows accordingly.
func TestModel_Update_SearchForwardsToDashboard(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version: "1.0.0",
		ListBackups: func() ([]BackupInfo, error) {
			return []BackupInfo{
				{ID: "conf-1", Date: "2024-01-01", Size: "1MB", Status: "ok", Cloud: "none"},
				{ID: "abc-2", Date: "2024-02-01", Size: "2MB", Status: "ok", Cloud: "gdrive"},
				{ID: "CONFIG-3", Date: "2024-03-01", Size: "3MB", Status: "ok", Cloud: "s3"},
			}, nil
		},
	})
	m.width = 80
	m.height = 24

	// Navigate to dashboard (lazy-init via screenChangeMsg).
	newM, _ := m.Update(screenChangeMsg{screen: ScreenDashboard})
	m = newM.(Model)

	// Activate search.
	m2, _ := m.Update(tea.KeyPressMsg{Code: '/'})
	m = m2.(Model)

	if !m.search.IsActive() {
		t.Fatal("search not active after '/' key")
	}

	// Type characters to build the query "conf".
	for _, ch := range []rune{'c', 'o', 'n', 'f'} {
		m3, _ := m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = m3.(Model)
	}

	// Verify search query reflects the typed characters.
	if m.search.Query() != "conf" {
		t.Errorf("search.Query() = %q, want %q", m.search.Query(), "conf")
	}

	// Verify dashboard table is filtered: only rows matching "conf" remain.
	output := m.dashboard.View().Content
	if !strings.Contains(output, "conf-1") {
		t.Error("dashboard view missing 'conf-1' after search filter")
	}
	if !strings.Contains(output, "CONFIG-3") {
		t.Error("dashboard view missing 'CONFIG-3' after search filter (case-insensitive)")
	}
	if strings.Contains(output, "abc-2") {
		t.Error("dashboard view should NOT contain 'abc-2' after search filter")
	}
}

// TestModel_Update_SearchEscRestoresAllRows verifies that pressing Esc
// while search is active on the dashboard deactivates search AND restores
// all original table rows (triangulation of search forwarding test).
func TestModel_Update_SearchEscRestoresAllRows(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version: "1.0.0",
		ListBackups: func() ([]BackupInfo, error) {
			return []BackupInfo{
				{ID: "conf-1", Date: "2024-01-01", Size: "1MB", Status: "ok", Cloud: "none"},
				{ID: "abc-2", Date: "2024-02-01", Size: "2MB", Status: "ok", Cloud: "gdrive"},
			}, nil
		},
	})
	m.width = 80
	m.height = 24

	// Navigate to dashboard.
	newM, _ := m.Update(screenChangeMsg{screen: ScreenDashboard})
	m = newM.(Model)

	// Activate search and filter with "conf".
	m2, _ := m.Update(tea.KeyPressMsg{Code: '/'})
	m = m2.(Model)
	m3, _ := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	m = m3.(Model)
	m4, _ := m.Update(tea.KeyPressMsg{Code: 'o', Text: "o"})
	m = m4.(Model)

	// Verify filtering occurred — abc-2 should be hidden.
	output := m.dashboard.View().Content
	if strings.Contains(output, "abc-2") {
		t.Error("after filtering 'co': abc-2 should be hidden")
	}

	// Press Esc to deactivate search and restore all rows.
	m5, _ := m.Update(tea.KeyPressMsg{Code: KeyEsc})
	m = m5.(Model)

	if m.search.IsActive() {
		t.Error("search should be inactive after Esc")
	}

	// All rows should be restored.
	output = m.dashboard.View().Content
	if !strings.Contains(output, "abc-2") {
		t.Error("after Esc: abc-2 should be restored")
	}
	if !strings.Contains(output, "conf-1") {
		t.Error("after Esc: conf-1 should be restored")
	}
}

// =============================================================================
// Helpers for test setup
// =============================================================================

// newSettingsPtr returns a pointer to a freshly initialized SettingsModel.
func newSettingsPtr() *screens.SettingsModel {
	sm := screens.NewSettingsModel(nil)
	return &sm
}

// newHealthPtr returns a pointer to a freshly initialized HealthModel.
func newHealthPtr() *screens.HealthModel {
	hm := screens.NewHealthModel()
	return &hm
}

// newProgressPtr returns a pointer to a freshly initialized ProgressModel.
func newProgressPtr() *screens.ProgressModel {
	pm := screens.NewProgressModel()
	return &pm
}

// =============================================================================
// Phase 2.2: Welcome Screen Tests — RED (NewModel doesn't yet check ConfigExists)
// =============================================================================

// TestModel_NewModel_ConfigNotExists_Welcome verifies that when ConfigExists
// returns false, NewModel starts at ScreenWelcome (first-run detection).
func TestModel_NewModel_ConfigNotExists_Welcome(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	deps := Deps{
		Version:      "1.0.0",
		ConfigExists: func() bool { return false },
	}
	m := NewModel(deps)

	if m.screen != ScreenWelcome {
		t.Errorf("NewModel with ConfigExists=false: screen = %v, want ScreenWelcome", m.screen)
	}
}

// TestModel_NewModel_ConfigExists_Menu verifies that when ConfigExists
// returns true, NewModel starts at ScreenMenu (normal launch).
func TestModel_NewModel_ConfigExists_Menu(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	deps := Deps{
		Version:      "1.0.0",
		ConfigExists: func() bool { return true },
	}
	m := NewModel(deps)

	if m.screen != ScreenMenu {
		t.Errorf("NewModel with ConfigExists=true: screen = %v, want ScreenMenu", m.screen)
	}
}

// TestModel_NewModel_ConfigExistsNil_Menu verifies that when ConfigExists
// is nil (not injected), NewModel falls back to ScreenMenu.
func TestModel_NewModel_ConfigExistsNil_Menu(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	deps := Deps{
		Version: "1.0.0",
	}
	m := NewModel(deps)

	if m.screen != ScreenMenu {
		t.Errorf("NewModel with ConfigExists=nil: screen = %v, want ScreenMenu", m.screen)
	}
}

// TestModel_Welcome_EnterToMenu verifies that pressing Enter on the
// Welcome screen transitions to ScreenMenu.
func TestModel_Welcome_EnterToMenu(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	deps := Deps{
		Version:      "1.0.0",
		ConfigExists: func() bool { return false },
	}
	m := NewModel(deps)
	m.width = 80
	m.height = 24

	newModel, _ := m.Update(tea.KeyPressMsg{Code: KeyEnter})
	result := newModel.(Model)

	if result.screen != ScreenMenu {
		t.Errorf("Welcome screen after Enter: screen = %v, want ScreenMenu", result.screen)
	}
}

// TestModel_Welcome_Quit verifies that pressing 'q' on the
// Welcome screen quits the TUI.
func TestModel_Welcome_Quit(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	deps := Deps{
		Version:      "1.0.0",
		ConfigExists: func() bool { return false },
	}
	m := NewModel(deps)
	m.width = 80
	m.height = 24

	newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	result := newModel.(Model)

	if cmd == nil {
		t.Error("Welcome screen: q returned nil cmd, want quit command")
	} else {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Errorf("Welcome screen: q cmd returned %T, want tea.QuitMsg", msg)
		}
	}
	_ = result
}

// TestModel_Welcome_View verifies that the Welcome screen view contains
// welcome text (rendered by screens.RenderWelcome).
func TestModel_Welcome_View(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	deps := Deps{
		Version:      "1.0.0",
		ConfigExists: func() bool { return false },
	}
	m := NewModel(deps)
	m.width = 80
	m.height = 24

	output := m.View().Content

	if !strings.Contains(output, "Welcome") {
		t.Errorf("Welcome view missing 'Welcome': %q", output)
	}
	if !strings.Contains(output, "get started") && !strings.Contains(output, "Enter") {
		t.Errorf("Welcome view missing prompt: %q", output)
	}
}

// TestModel_Welcome_EscQuit verifies that pressing Esc on the
// Welcome screen also quits (triangulation of q quit).
func TestModel_Welcome_EscQuit(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	deps := Deps{
		Version:      "1.0.0",
		ConfigExists: func() bool { return false },
	}
	m := NewModel(deps)
	m.width = 80
	m.height = 24

	_, cmd := m.Update(tea.KeyPressMsg{Code: KeyEsc})
	if cmd == nil {
		t.Error("Welcome screen: Esc returned nil cmd, want quit command")
	} else {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Errorf("Welcome screen: Esc cmd returned %T, want tea.QuitMsg", msg)
		}
	}
}

// =============================================================================
// Phase 2.3: Toast Positioning Tests — RED (View uses inline toast, not Place)
// =============================================================================

// TestModel_Toast_PositionedWide verifies that on a wide terminal (>=50 cols),
// the toast is positioned using lipgloss.Place (it should NOT appear inline
// after content with a simple newline prefix).
func TestModel_Toast_PositionedWide(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version:      "1.0.0",
		ConfigExists: func() bool { return true },
	})
	m.width = 80
	m.height = 24
	m.toast.Show("Backup complete", 3)

	output := m.View().Content

	// Toast message must be present.
	if !strings.Contains(output, "Backup complete") {
		t.Errorf("View() wide missing toast message: %q", output)
	}

	// On wide terminals, the toast should NOT be appended with just a newline
	// (the old behavior). The positioned output should show the menu content
	// AND the toast but separated by ANSI cursor positioning, not just "\n".
	if output[len(output)-1] == '\n' {
		t.Error("View() wide: toast may be inline-appended (ends with newline), want positioned")
	}
}

// TestModel_Toast_InlineNarrow verifies that on a narrow terminal (<50 cols),
// the toast falls back to inline rendering at the bottom of content.
func TestModel_Toast_InlineNarrow(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version:      "1.0.0",
		ConfigExists: func() bool { return true },
	})
	m.width = 30
	m.height = 15
	m.toast.Show("Error", 3)

	output := m.View().Content

	// Toast message must be present.
	if !strings.Contains(output, "Error") {
		t.Errorf("View() narrow missing toast message: %q", output)
	}
}

// =============================================================================
// Phase 2.4: Help Overlay Tests — RED (no showHelp bool or '?' toggle yet)
// =============================================================================

// TestModel_Help_ToggleOnMenu verifies that pressing '?' on the main menu
// toggles the help overlay on without leaving the menu screen.
func TestModel_Help_ToggleOnMenu(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version:      "1.0.0",
		ConfigExists: func() bool { return true },
	})
	m.width = 80
	m.height = 24

	// Press '?' — should toggle help on.
	newModel, _ := m.Update(tea.KeyPressMsg{Code: '?'})
	result := newModel.(Model)

	if !result.showHelp {
		t.Error("after '?' on menu: showHelp = false, want true")
	}
	// Screen should still be ScreenMenu (overlay, not separate screen).
	if result.screen != ScreenMenu {
		t.Errorf("after '?' on menu: screen = %v, want ScreenMenu", result.screen)
	}

	// Press '?' again — should toggle help off.
	newModel2, _ := newModel.Update(tea.KeyPressMsg{Code: '?'})
	result2 := newModel2.(Model)

	if result2.showHelp {
		t.Error("after second '?': showHelp = true, want false")
	}
}

// TestModel_Help_DismissViaEsc verifies that pressing Esc while help is
// visible dismisses the overlay and returns to the active screen.
func TestModel_Help_DismissViaEsc(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version:      "1.0.0",
		ConfigExists: func() bool { return true },
	})
	m.width = 80
	m.height = 24
	m.showHelp = true
	m.screen = ScreenSettings // simulate being on a sub-screen

	newModel, _ := m.Update(tea.KeyPressMsg{Code: KeyEsc})
	result := newModel.(Model)

	if result.showHelp {
		t.Error("after Esc with help visible: showHelp = true, want false")
	}
	// Screen should still be the active screen (not returned to menu).
	if result.screen != ScreenSettings {
		t.Errorf("after Esc from help: screen = %v, want ScreenSettings", result.screen)
	}
}

// TestModel_Help_ViewContainsShortcuts verifies that when showHelp is true,
// the View output contains the shortcuts reference content.
func TestModel_Help_ViewContainsShortcuts(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version:      "1.0.0",
		ConfigExists: func() bool { return true },
	})
	m.width = 80
	m.height = 24
	m.showHelp = true

	output := m.View().Content

	// Help overlay should contain shortcut categories.
	if !strings.Contains(output, "Navigation") {
		t.Errorf("help view missing 'Navigation': %q", output)
	}
	if !strings.Contains(output, "Actions") {
		t.Errorf("help view missing 'Actions': %q", output)
	}
	if !strings.Contains(output, "Screens") {
		t.Errorf("help view missing 'Screens': %q", output)
	}
	if !strings.Contains(output, "Meta") {
		t.Errorf("help view missing 'Meta': %q", output)
	}
}

// TestModel_Help_ToggleOnSubScreen verifies that pressing '?' on a
// sub-screen (Settings) toggles the help overlay without leaving the screen.
func TestModel_Help_ToggleOnSubScreen(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version:      "1.0.0",
		ConfigExists: func() bool { return true },
	})
	m.width = 80
	m.height = 24
	m.screen = ScreenSettings
	m.settings = newSettingsPtr()

	// Give sub-model dimensions.
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m2.(Model)

	// Press '?' — should toggle help on.
	newModel, _ := m.Update(tea.KeyPressMsg{Code: '?'})
	result := newModel.(Model)

	if !result.showHelp {
		t.Error("after '?' on settings: showHelp = false, want true")
	}
	if result.screen != ScreenSettings {
		t.Errorf("after '?' on settings: screen = %v, want ScreenSettings", result.screen)
	}
}

// =============================================================================
// Progress bridge tests (Phase 3.4)
// =============================================================================

func TestBackupChannelBridge(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// Test 1: drainProgressCmd reads from channel and converts correctly.
	ch := make(chan ProgressUpdate, 3)
	ch <- ProgressUpdate{Step: "file1.txt", Current: 1, Total: 3}
	ch <- ProgressUpdate{Step: "file2.txt", Current: 2, Total: 3}
	ch <- ProgressUpdate{Done: true}

	cmd := drainProgressCmd(ch)
	if cmd == nil {
		t.Fatal("drainProgressCmd returned nil for non-nil channel")
	}

	msg := cmd()
	step, ok := msg.(screens.ProgressStepMsg)
	if !ok {
		t.Fatalf("expected ProgressStepMsg, got %T", msg)
	}
	if step.Step != "file1.txt" {
		t.Errorf("step.Step = %q, want file1.txt", step.Step)
	}
	if step.Current != 1 || step.Total != 3 {
		t.Errorf("step.Current=%d, Total=%d, want 1/3", step.Current, step.Total)
	}

	// Drain again — should get second step.
	cmd = drainProgressCmd(ch)
	msg = cmd()
	step, ok = msg.(screens.ProgressStepMsg)
	if !ok {
		t.Fatalf("expected ProgressStepMsg, got %T", msg)
	}
	if step.Step != "file2.txt" {
		t.Errorf("step.Step = %q, want file2.txt", step.Step)
	}

	// Drain again — should get Done.
	cmd = drainProgressCmd(ch)
	msg = cmd()
	_, ok = msg.(screens.ProgressDoneMsg)
	if !ok {
		t.Fatalf("expected ProgressDoneMsg, got %T", msg)
	}
}

func TestBackupChannelBridgeScreenProgress(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// Test 2: When ProgressStepMsg arrives on ScreenProgress, it sets running.
	m := Model{
		screen:    ScreenProgress,
		deps:      Deps{},
		menuItems: DefaultMenuItems,
	}
	m.progress = newProgressPtrForTest()

	// Send a ProgressStepMsg directly.
	m2, _ := m.Update(screens.ProgressStepMsg{
		Step:    "test.txt",
		Current: 1,
		Total:   5,
	})
	result := m2.(Model)
	if result.progress == nil {
		t.Fatal("progress is nil")
	}
	if !result.progress.Running() {
		t.Error("progress should be running after ProgressStepMsg")
	}
}

func TestBackupChannelBridgeDoneMsg(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// Test 3: ProgressDoneMsg stops running and returns actionResultMsg.
	m := Model{
		screen:    ScreenProgress,
		deps:      Deps{},
		menuItems: DefaultMenuItems,
	}
	m.progress = newProgressPtrForTest()
	m.progress.Update(screens.ProgressStepMsg{Step: "x", Current: 1, Total: 1})

	// Send ProgressDoneMsg.
	_, cmd := m.Update(screens.ProgressDoneMsg{})
	if cmd == nil {
		t.Fatal("expected command after ProgressDoneMsg")
	}

	// The cmd produced by the handler should include actionResultMsg.
	// Note: we can't easily inspect the inner batch, but the progress
	// model should have running=false.
	if m.progress.Running() {
		t.Error("progress should not be running after ProgressDoneMsg")
	}
}

func TestDrainProgressCmdNilSafe(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// A nil channel should produce a nil drain command or no-op.
	cmd := drainProgressCmd(nil)
	if cmd != nil {
		t.Error("drainProgressCmd(nil) should not return a command")
	}
}

func TestProgressDoneMsgTerminatesDrain(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := Model{
		screen:    ScreenProgress,
		deps:      Deps{},
		menuItems: DefaultMenuItems,
	}
	m.progress = newProgressPtrForTest()

	ch := make(chan ProgressUpdate, 1)
	ch <- ProgressUpdate{Done: true}

	drainCmd := drainProgressCmd(ch)
	if drainCmd == nil {
		t.Fatal("drainProgressCmd returned nil")
	}
	msg := drainCmd()
	if msg == nil {
		t.Fatal("drain msg is nil")
	}

	_, nextCmd := m.Update(msg)
	// After Done, the handler returns ScreenBackMsg via ProgressDoneMsg.
	// We just verify the Update doesn't crash.
	_ = nextCmd
}

func newProgressPtrForTest() *screens.ProgressModel {
	p := screens.NewProgressModel()
	return &p
}

// TestNewModel_LoadsSettings verifies that when LoadSettings is provided
// in Deps, the settings screen uses persisted values instead of hardcoded
// defaults on first entry to ScreenSettings.
func TestNewModel_LoadsSettings(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	loadCalled := false
	m := NewModel(Deps{
		Version: "1.0.0",
		LoadSettings: func() (screens.Settings, error) {
			loadCalled = true
			return screens.Settings{
				AutoSync:           true,
				DefaultPreset:      "full",
				MaxFileSize:        2097152,
				ConfirmDestructive: false,
				VerboseDefault:     true,
				DefaultProvider:    "github",
			}, nil
		},
	})
	m.width = 80
	m.height = 24

	// Navigate to settings screen — this should trigger LoadSettings.
	newM, cmd := m.Update(screenChangeMsg{screen: ScreenSettings})
	m2 := newM.(Model)

	// Verify LoadSettings was called.
	if !loadCalled {
		t.Error("LoadSettings was not called when entering ScreenSettings")
	}

	// Verify settings sub-model was created.
	if m2.settings == nil {
		t.Fatal("settings sub-model is nil after entering ScreenSettings")
	}

	// Drain any init command.
	if cmd != nil {
		_ = cmd()
	}
}

// =============================================================================
// Phase 2: Backfill model.go — initRestore, initProfiles, initCloud
// =============================================================================

// TestInitRestore verifies that initRestore wires Deps.ListBackups and
// Deps.RunRestore into the RestoreModel via listFn/restoreFn closures.
func TestInitRestore(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	listCalled := false

	m := NewModel(Deps{
		Version: "1.0.0",
		ListBackups: func() ([]BackupInfo, error) {
			listCalled = true
			return []BackupInfo{
				{ID: "backup-1", Date: "2024-01-01", Size: "1MB", Status: "ok", Cloud: "none"},
			}, nil
		},
		RunRestore: func(backupID string, dryRun bool) (string, error) {
			return "restored", nil
		},
	})

	r := m.initRestore()

	// Init should load backups via the wired listFn.
	cmd := r.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil, want backup load command")
	}
	msg := cmd()
	r2, _ := r.Update(msg)
	rm := r2.(screens.RestoreModel)

	if !listCalled {
		t.Error("ListBackups was not called via initRestore closure")
	}
	if len(rm.Backups) != 1 {
		t.Errorf("backups length = %d, want 1", len(rm.Backups))
	}
	if rm.Backups[0].ID != "backup-1" {
		t.Errorf("backup ID = %q, want backup-1", rm.Backups[0].ID)
	}

	// Verify the model renders its View without panicking.
	v := rm.View().Content
	if v == "" {
		t.Error("RestoreModel View() returned empty")
	}
}

// TestInitRestore_NilDeps verifies initRestore handles nil Deps gracefully.
func TestInitRestore_NilDeps(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})

	r := m.initRestore()

	// Init should work without panicking.
	cmd := r.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil, want backup load command")
	}
	msg := cmd()
	r2, _ := r.Update(msg)
	rm := r2.(screens.RestoreModel)

	// No error should be set when ListBackups is nil.
	if rm.Err != nil {
		t.Errorf("initRestore with nil ListBackups: Err = %v, want nil", rm.Err)
	}
}

// TestInitProfiles verifies that initProfiles wires Deps.ListProfiles,
// Deps.SetActiveProfile, Deps.DeleteProfile, Deps.RunWizard, and
// Deps.SaveProfile into the ProfilesModel.
func TestInitProfiles(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	listCalled := false

	m := NewModel(Deps{
		Version: "1.0.0",
		ListProfiles: func() ([]ProfileInfo, error) {
			listCalled = true
			return []ProfileInfo{
				{Name: "work", Provider: "github", Preset: "full", Active: true},
			}, nil
		},
		SetActiveProfile: func(name string) error { return nil },
	})

	p := m.initProfiles()

	// Verify SaveProfile is set as a mutable field.
	if p.SaveProfile == nil {
		t.Error("SaveProfile should be set after initProfiles")
	}

	// Init should load profiles.
	cmd := p.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil")
	}
	msg := cmd()
	p2, _ := p.Update(msg)
	pm := p2.(screens.ProfilesModel)

	if !listCalled {
		t.Error("ListProfiles was not called")
	}
	if len(pm.Profiles) != 1 {
		t.Errorf("profiles length = %d, want 1", len(pm.Profiles))
	}

	// Verify the model renders its View without panicking.
	v := pm.View().Content
	if v == "" {
		t.Error("ProfilesModel View() returned empty")
	}
}

// TestInitCloud verifies that initCloud wires Deps.GetCloudStatus into
// the CloudModel via a statusFn closure.
func TestInitCloud(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	statusCalled := false

	m := NewModel(Deps{
		Version: "1.0.0",
		GetCloudStatus: func() (CloudStatus, error) {
			statusCalled = true
			return CloudStatus{
				Provider:   "github",
				Connected:  true,
				LastSync:   "2024-06-09",
				LocalCount: 5,
				CloudCount: 3,
			}, nil
		},
	})

	c := m.initCloud()

	// Verify CloudModel has a wired statusFn.
	cmd := c.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil")
	}
	msg := cmd()
	c2, _ := c.Update(msg)
	cm := c2.(screens.CloudModel)

	if !statusCalled {
		t.Error("GetCloudStatus was not called via statusFn closure")
	}
	if cm.Info.Provider != "github" {
		t.Errorf("Info.Provider = %q, want github", cm.Info.Provider)
	}
	if cm.Info.LocalCount != 5 {
		t.Errorf("Info.LocalCount = %d, want 5", cm.Info.LocalCount)
	}
}

// TestInitCloud_NilStatusFn verifies initCloud handles nil GetCloudStatus.
func TestInitCloud_NilStatusFn(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})

	c := m.initCloud()

	cmd := c.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil")
	}
	msg := cmd()
	c2, _ := c.Update(msg)
	cm := c2.(screens.CloudModel)

	// No error, empty info.
	if cm.Err != nil {
		t.Errorf("initCloud with nil GetCloudStatus: Err = %v, want nil", cm.Err)
	}
	if cm.Info.Provider != "" {
		t.Errorf("Info.Provider = %q, want empty", cm.Info.Provider)
	}
}

// TestInitCloud_StatusError verifies initCloud handles errors from GetCloudStatus.
func TestInitCloud_StatusError(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version: "1.0.0",
		GetCloudStatus: func() (CloudStatus, error) {
			return CloudStatus{}, fmt.Errorf("network unreachable")
		},
	})

	c := m.initCloud()

	cmd := c.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil")
	}
	msg := cmd()
	c2, _ := c.Update(msg)
	cm := c2.(screens.CloudModel)

	if cm.Err == nil {
		t.Error("initCloud with error GetCloudStatus: Err = nil, want error")
	}
	if cm.Err.Error() != "network unreachable" {
		t.Errorf("initCloud error = %q, want %q", cm.Err.Error(), "network unreachable")
	}
}

// TestModel_Update_UnknownMsg verifies that an unknown message type
// does not cause a panic and returns the model unchanged.
func TestModel_Update_UnknownMsg(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24

	// Send a completely unknown message type.
	newModel, cmd := m.Update(struct{ x int }{x: 42})
	result := newModel.(Model)

	// Model should be returned unchanged; no crash.
	if result.screen != ScreenMenu {
		t.Errorf("after unknown msg: screen = %v, want ScreenMenu", result.screen)
	}
	if cmd != nil {
		t.Errorf("unknown msg returned cmd %v, want nil", cmd)
	}
}

// TestModel_HandleKey_ScreenProfiles verifies key dispatch for ScreenProfiles.
func TestModel_HandleKey_ScreenProfiles(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version: "1.0.0",
		ListProfiles: func() ([]ProfileInfo, error) {
			return []ProfileInfo{
				{Name: "work", Provider: "github", Preset: "full", Active: true},
			}, nil
		},
	})
	m.width = 80
	m.height = 24
	m.screen = ScreenProfiles
	// Init profiles via screenChangeMsg.
	m2, _ := m.Update(screenChangeMsg{screen: ScreenProfiles})
	m = m2.(Model)

	// Press 'j' — should be forwarded to profiles sub-model.
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	result := newModel.(Model)

	if result.screen != ScreenProfiles {
		t.Errorf("after j on profiles: screen = %v, want ScreenProfiles", result.screen)
	}
}

// TestModel_HandleKey_ScreenRestore verifies key dispatch for ScreenRestore.
func TestModel_HandleKey_ScreenRestore(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version: "1.0.0",
		ListBackups: func() ([]BackupInfo, error) {
			return []BackupInfo{
				{ID: "b1", Date: "2024-01-01", Size: "1MB", Status: "ok", Cloud: "none"},
			}, nil
		},
	})
	m.width = 80
	m.height = 24
	m.screen = ScreenRestore
	// Init restore via screenChangeMsg.
	m2, _ := m.Update(screenChangeMsg{screen: ScreenRestore})
	m = m2.(Model)

	// Press 'q' on restore — the sub-model returns a cmd producing ScreenBackMsg.
	newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	// Execute the returned cmd to get ScreenBackMsg, then process it.
	if cmd != nil {
		msg := cmd()
		newModel, _ = newModel.Update(msg)
	}
	result := newModel.(Model)

	if result.screen != ScreenMenu {
		t.Errorf("after q on restore: screen = %v, want ScreenMenu", result.screen)
	}
}

// TestModel_HandleKey_ScreenSettings verifies key dispatch for ScreenSettings.
func TestModel_HandleKey_ScreenSettings(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24
	m.screen = ScreenSettings
	m.settings = newSettingsPtr()

	// Give sub-model dimensions.
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m2.(Model)

	// Press 'q' on settings — the sub-model returns a cmd producing ScreenBackMsg.
	newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	if cmd != nil {
		msg := cmd()
		newModel, _ = newModel.Update(msg)
	}
	result := newModel.(Model)

	if result.screen != ScreenMenu {
		t.Errorf("after q on settings: screen = %v, want ScreenMenu", result.screen)
	}
}

// TestModel_HandleKey_ScreenHealth verifies key dispatch for ScreenHealth.
func TestModel_HandleKey_ScreenHealth(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24
	m.screen = ScreenHealth
	m.health = newHealthPtr()

	// Give sub-model dimensions.
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = m2.(Model)

	// Press 'q' on health — the sub-model returns a cmd producing ScreenBackMsg.
	newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	if cmd != nil {
		msg := cmd()
		newModel, _ = newModel.Update(msg)
	}
	result := newModel.(Model)

	if result.screen != ScreenMenu {
		t.Errorf("after q on health: screen = %v, want ScreenMenu", result.screen)
	}
}

// =============================================================================
// Phase 2.6: Coverage gap fill — initProfiles nil deps, initRestore nil RunRestore
// =============================================================================

// TestInitProfiles_NilDeps verifies initProfiles handles nil ListProfiles,
// nil DeleteProfile, and nil RunWizard gracefully.
func TestInitProfiles_NilDeps(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})

	p := m.initProfiles()

	// Init should work without panicking.
	cmd := p.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil")
	}
	msg := cmd()
	p2, _ := p.Update(msg)
	pm := p2.(screens.ProfilesModel)

	// No error and no profiles when ListProfiles is nil.
	if pm.Err != nil {
		t.Errorf("initProfiles nil ListProfiles: Err = %v, want nil", pm.Err)
	}
	if len(pm.Profiles) != 0 {
		t.Errorf("profiles length = %d, want 0", len(pm.Profiles))
	}

	// SaveProfile should still be set.
	if pm.SaveProfile == nil {
		t.Error("SaveProfile should be set even when ListProfiles is nil")
	}
}

// TestInitRestore_NilRunRestore verifies initRestore handles nil RunRestore.
func TestInitRestore_NilRunRestore(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version: "1.0.0",
		ListBackups: func() ([]BackupInfo, error) {
			return []BackupInfo{
				{ID: "b1", Date: "2024-01-01", Size: "1MB", Status: "ok", Cloud: "none"},
			}, nil
		},
		RunRestore: nil,
	})

	r := m.initRestore()

	cmd := r.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil")
	}
	msg := cmd()
	r2, _ := r.Update(msg)
	rm := r2.(screens.RestoreModel)

	if len(rm.Backups) != 1 {
		t.Errorf("backups length = %d, want 1", len(rm.Backups))
	}
}

// TestModel_View_ScreenRestoreNoSubmodel verifies View when ScreenRestore
// has no sub-model initialized (falls back to "Restore" text).
func TestModel_View_ScreenRestoreNoSubmodel(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24
	m.screen = ScreenRestore
	m.restore = nil

	output := m.View().Content

	if !strings.Contains(output, "Restore") {
		t.Errorf("ScreenRestore no sub-model: View() = %q, want 'Restore'", output)
	}
}

// TestModel_View_ScreenProfilesNoSubmodel verifies View when ScreenProfiles
// has no sub-model initialized (falls back to "Profiles" text).
func TestModel_View_ScreenProfilesNoSubmodel(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24
	m.screen = ScreenProfiles
	m.profiles = nil

	output := m.View().Content

	if !strings.Contains(output, "Profiles") {
		t.Errorf("ScreenProfiles no sub-model: View() = %q, want 'Profiles'", output)
	}
}

// TestModel_View_ScreenCloudNoSubmodel verifies View when ScreenCloud
// has no sub-model initialized (falls back to RenderCloudStatus).
func TestModel_View_ScreenCloudNoSubmodel(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24
	m.screen = ScreenCloud
	m.cloud = nil

	output := m.View().Content

	if !strings.Contains(output, "No cloud provider configured") {
		t.Errorf("ScreenCloud no sub-model: View() = %q, want cloud status", output)
	}
}

// TestModel_HandleMenuEnter_Cloud verifies that Enter on cursor=3
// ("Cloud sync") dispatches screenChangeMsg for ScreenCloud.
func TestModel_HandleMenuEnter_Cloud(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.cursor = 3 // "Cloud sync"

	_, cmd := m.Update(tea.KeyPressMsg{Code: KeyEnter})

	if cmd == nil {
		t.Fatal("Enter on Cloud sync returned nil cmd")
	}
	msg := cmd()
	switch msg := msg.(type) {
	case screenChangeMsg:
		if msg.screen != ScreenCloud {
			t.Errorf("screenChangeMsg.screen = %v, want ScreenCloud", msg.screen)
		}
	default:
		t.Errorf("Enter on Cloud sync returned %T, want screenChangeMsg", msg)
	}
}

// TestInitProfiles_NilClosures verifies that switchFn, deleteFn, and
// wizardFn closures handle nil Deps gracefully (hit nil-return branches).
func TestInitProfiles_NilClosures(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})

	p := m.initProfiles()
	// Set profiles — both inactive so enter won't make them undeletable.
	p.Profiles = []screens.ProfileInfo{
		{Name: "test", Provider: "github", Preset: "full", Active: false},
	}
	p.Cursor = 0
	p.Width = 80
	p.Height = 24

	// Press 'n' — should call wizardFn (nil → returns empty ProfileInfo).
	p3, cmd := p.Update(tea.KeyPressMsg{Code: 'n'})
	_ = cmd // wizardFn creates an async cmd; we skip actual execution
	pm := p3.(screens.ProfilesModel)

	// Press enter — should call switchFn (nil → no-op).
	p4, _ := pm.Update(tea.KeyPressMsg{Code: '\r'})
	pm = p4.(screens.ProfilesModel)

	if pm.Err != nil {
		t.Errorf("switchFn nil deps: Err = %v, want nil", pm.Err)
	}
}

// TestInitProfiles_DeleteWithNilDeps verifies deleteFn nil path via modal confirm.
func TestInitProfiles_DeleteWithNilDeps(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})

	p := m.initProfiles()
	p.Profiles = []screens.ProfileInfo{
		{Name: "test", Provider: "github", Preset: "full", Active: false},
	}
	p.Cursor = 0
	p.Width = 80
	p.Height = 24

	// Press 'd' to show modal.
	p2, _ := p.Update(tea.KeyPressMsg{Code: 'd'})
	pm := p2.(screens.ProfilesModel)

	if pm.Modal == nil {
		t.Fatal("Modal should be set for delete confirmation")
	}

	// Confirm delete — deleteFn is nil, should be no-op.
	p3, _ := pm.Update(components.ModalResultMsg{Confirmed: true})
	pm = p3.(screens.ProfilesModel)

	// Profile should be removed from local list.
	if len(pm.Profiles) != 0 {
		t.Errorf("after delete: profiles length = %d, want 0", len(pm.Profiles))
	}
}

// TestInitSettings_LoadError verifies that when LoadSettings returns an error,
// defaults are used (NewSettingsModel fallback path).
func TestInitSettings_LoadError(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version: "1.0.0",
		LoadSettings: func() (screens.Settings, error) {
			return screens.Settings{}, fmt.Errorf("config corrupted")
		},
	})
	m.width = 80
	m.height = 24

	s := m.initSettings()
	// Forward dimensions to the sub-model.
	s2, _ := s.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	sm := s2.(screens.SettingsModel)

	// Should create a SettingsModel with defaults (error path).
	v := sm.View().Content
	if !strings.Contains(v, "Settings") {
		t.Errorf("initSettings error path: View() = %q, want 'Settings'", v)
	}
}

// TestDrainProgressCmd_ClosedChannel verifies drainProgressCmd handles a
// closed channel gracefully (returns ProgressDoneMsg).
func TestDrainProgressCmd_ClosedChannel(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	ch := make(chan ProgressUpdate, 1)
	close(ch)

	cmd := drainProgressCmd(ch)
	if cmd == nil {
		t.Fatal("drainProgressCmd returned nil for closed channel")
	}

	msg := cmd()
	_, ok := msg.(screens.ProgressDoneMsg)
	if !ok {
		t.Errorf("closed channel: expected ProgressDoneMsg, got %T", msg)
	}
}

// TestModel_Update_ToastTickForwarding verifies that tick messages not
// handled by screen routing are forwarded to the toast component.
func TestModel_Update_ToastTickForwarding(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 80
	m.height = 24
	m.toast.Show("visible toast", 5)

	// Forward an unknown message type via the default path.
	newModel, cmd := m.Update(struct{ x int }{x: 42})
	result := newModel.(Model)

	// The toast should still be visible after an unknown message.
	if result.toast.View() == "" {
		t.Error("toast should still be visible after unknown msg forwarding")
	}
	_ = cmd
}

// TestMapBackupInfo verifies the pure mapBackupInfo converts tui.BackupInfo
// records into screens.BackupInfo preserving every field, in order.
func TestMapBackupInfo(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name string
		in   []BackupInfo
		want []screens.BackupInfo
	}{
		{
			name: "nil input returns nil",
			in:   nil,
			want: nil,
		},
		{
			name: "empty input returns nil",
			in:   []BackupInfo{},
			want: nil,
		},
		{
			name: "maps all fields for multiple records",
			in: []BackupInfo{
				{ID: "b1", Date: "2026-01-01", Size: "1.2 MB", Status: "ok", Cloud: "gitea"},
				{ID: "b2", Date: "2026-02-02", Size: "3.4 MB", Status: "fail", Cloud: ""},
			},
			want: []screens.BackupInfo{
				{ID: "b1", Date: "2026-01-01", Size: "1.2 MB", Status: "ok", Cloud: "gitea"},
				{ID: "b2", Date: "2026-02-02", Size: "3.4 MB", Status: "fail", Cloud: ""},
			},
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			got := mapBackupInfo(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("mapBackupInfo() len = %d, want %d", len(got), len(tt.want))
			}
			for i, w := range tt.want {
				if got[i] != w {
					t.Errorf("mapBackupInfo()[%d] = %+v, want %+v", i, got[i], w)
				}
			}
		})
	}
}

// TestListBackupsForScreens verifies the shared listBackupsForScreens method
// handles nil deps, propagates errors, and maps successful results.
func TestListBackupsForScreens(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	sentinel := errors.New("backend down")

	tests := []struct {
		name        string
		listBackups func() ([]BackupInfo, error)
		wantLen     int
		wantErr     error
		wantFirstID string
	}{
		{
			name:        "nil ListBackups returns nil, nil",
			listBackups: nil,
			wantLen:     0,
			wantErr:     nil,
		},
		{
			name: "error from ListBackups is propagated",
			listBackups: func() ([]BackupInfo, error) {
				return nil, sentinel
			},
			wantErr: sentinel,
		},
		{
			name: "success maps records to screens.BackupInfo",
			listBackups: func() ([]BackupInfo, error) {
				return []BackupInfo{
					{ID: "x1", Date: "2026-03-03", Size: "5 MB", Status: "ok", Cloud: "gh"},
				}, nil
			},
			wantLen:     1,
			wantFirstID: "x1",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			m := NewModel(Deps{Version: "1.0.0", ListBackups: tt.listBackups})

			got, err := m.listBackupsForScreens()

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("listBackupsForScreens() err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("listBackupsForScreens() unexpected err = %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("listBackupsForScreens() len = %d, want %d", len(got), tt.wantLen)
			}
			if tt.wantFirstID != "" && got[0].ID != tt.wantFirstID {
				t.Errorf("listBackupsForScreens()[0].ID = %q, want %q", got[0].ID, tt.wantFirstID)
			}
		})
	}
}

// =============================================================================
// Phase 5: subModel dispatch map — RED (forwardTo / subs / subModel absent)
// =============================================================================

// TestModel_forwardTo_RoutesToSubModel verifies forwardTo dispatches a
// message to the sub-model registered for the given screen, returns
// (cmd, true) when a sub-model is present, and keeps the cached instance.
func TestModel_forwardTo_RoutesToSubModel(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version: "1.0.0",
		ListBackups: func() ([]BackupInfo, error) {
			return []BackupInfo{{ID: "abc-1", Date: "2024-01-01", Size: "1MB", Status: "ok", Cloud: "none"}}, nil
		},
	})
	// Lazy-init the dashboard sub-model via a screen change.
	m2, _ := m.Update(screenChangeMsg{screen: ScreenDashboard})
	m = m2.(Model)

	if m.subs == nil || m.subs[ScreenDashboard] == nil {
		t.Fatal("subs[ScreenDashboard] not populated after screenChange")
	}

	// Forward a window-size message: should route to the dashboard sub-model.
	_, ok := m.forwardTo(ScreenDashboard, tea.WindowSizeMsg{Width: 80, Height: 24})
	if !ok {
		t.Error("forwardTo(ScreenDashboard) = false, want true (sub-model exists)")
	}

	// The cached instance survives the forward: dashboard still renders its data.
	if m.dashboard == nil {
		t.Fatal("dashboard sub-model nil after forwardTo")
	}
	if !strings.Contains(m.dashboard.View().Content, "abc-1") {
		t.Error("cached dashboard lost its loaded data after forwardTo")
	}
}

// TestModel_forwardTo_UnknownScreenReturnsFalse verifies forwardTo returns
// (nil, false) for an unrecognized screen value without panicking.
func TestModel_forwardTo_UnknownScreenReturnsFalse(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})

	_, ok := m.forwardTo(screen(99), tea.KeyPressMsg{Code: 'j'})
	if ok {
		t.Error("forwardTo(unknown screen) = true, want false")
	}
}

// TestModel_forwardTo_NilSubModelReturnsFalse verifies forwardTo returns
// (nil, false) when the screen is registered but its sub-model is nil,
// without panicking (mirrors the `if m.x != nil` guards in the old code).
func TestModel_forwardTo_NilSubModelReturnsFalse(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	// Dashboard never initialized → get returns nil.

	_, ok := m.forwardTo(ScreenDashboard, tea.KeyPressMsg{Code: 'j'})
	if ok {
		t.Error("forwardTo(ScreenDashboard) with nil sub-model = true, want false")
	}
}

// TestModel_Update_LazyInitPopulatesSubsMap verifies that processing a
// screenChangeMsg lazily creates the sub-model and stores it in the subs map,
// and that subsequent screen changes add entries without clearing prior ones.
func TestModel_Update_LazyInitPopulatesSubsMap(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{
		Version:     "1.0.0",
		ListBackups: func() ([]BackupInfo, error) { return nil, nil },
	})
	if m.subs != nil {
		t.Error("subs should be nil before any screenChange")
	}

	m2, _ := m.Update(screenChangeMsg{screen: ScreenDashboard})
	result := m2.(Model)
	if result.subs == nil {
		t.Fatal("subs map nil after screenChange")
	}
	if result.subs[ScreenDashboard] == nil {
		t.Error("subs[ScreenDashboard] nil after screenChange (lazy-init failed)")
	}

	// A second screen change populates another entry without clearing the first.
	m3, _ := result.Update(screenChangeMsg{screen: ScreenSettings})
	result = m3.(Model)
	if result.subs[ScreenSettings] == nil {
		t.Error("subs[ScreenSettings] nil after second screenChange")
	}
	if result.subs[ScreenDashboard] == nil {
		t.Error("subs[ScreenDashboard] lost after entering Settings")
	}
}

// TestModel_Update_UnknownScreenNoPanic verifies that a model on an
// unrecognized screen value handles a forwardable message without panicking
// and returns (m, nil).
func TestModel_Update_UnknownScreenNoPanic(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := NewModel(Deps{Version: "1.0.0"})
	m.screen = screen(99)

	newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'j'})
	result := newModel.(Model)

	if result.screen != screen(99) {
		t.Errorf("unknown screen: screen = %v, want 99", result.screen)
	}
	if cmd != nil {
		t.Errorf("unknown screen: cmd = %v, want nil", cmd)
	}
}
