// Package tui provides the root TUI model with screen routing, key navigation,
// and window size handling. This file contains table-driven TDD tests written
// BEFORE the production code (strict RED phase).
package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/danielxxomg/bak-cli/internal/tui/screens"
)

// =============================================================================
// TestNewModel — RED (model.go does not exist yet)
// =============================================================================

func TestNewModel(t *testing.T) {
	deps := Deps{
		Version:      "1.0.0",
		ConfigExists: func() bool { return false },
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

func TestModel_Init(t *testing.T) {
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

func TestModel_Update_Quit(t *testing.T) {
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

func TestModel_Update_NavigateDown(t *testing.T) {
	m := NewModel(Deps{Version: "1.0.0"})

	// Press 'j' once: cursor should go from 0 to 1.
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
	result := newModel.(Model)
	if result.cursor != 1 {
		t.Errorf("Update('j') cursor = %d, want 1", result.cursor)
	}

	// Press 'j' 5 more times: cursor should clamp at len-1 (6).
	for i := 0; i < 5; i++ {
		newModel, _ = newModel.Update(tea.KeyPressMsg{Code: 'j'})
	}
	result = newModel.(Model)
	if result.cursor != 6 {
		t.Errorf("Update('j' x6) cursor = %d, want 6 (clamped)", result.cursor)
	}

	// One more press: still 6 (clamped).
	newModel, _ = newModel.Update(tea.KeyPressMsg{Code: 'j'})
	result = newModel.(Model)
	if result.cursor != 6 {
		t.Errorf("Update('j' x7) cursor = %d, want 6 (clamped at max)", result.cursor)
	}
}

func TestModel_Update_NavigateUp(t *testing.T) {
	m := NewModel(Deps{Version: "1.0.0"})
	// Start with cursor at 3 (simulate via navigation).
	m.cursor = 3

	// Press 'k': cursor should go from 3 to 2.
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'k'})
	result := newModel.(Model)
	if result.cursor != 2 {
		t.Errorf("Update('k') cursor = %d, want 2", result.cursor)
	}

	// Press 'k' 3 more times: should clamp at 0.
	for i := 0; i < 3; i++ {
		newModel, _ = newModel.Update(tea.KeyPressMsg{Code: 'k'})
	}
	result = newModel.(Model)
	if result.cursor != 0 {
		t.Errorf("Update('k' x4) cursor = %d, want 0 (clamped)", result.cursor)
	}
}

func TestModel_Update_WindowSize(t *testing.T) {
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

func TestModel_Update_MinSizeGuard(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		height   int
		tooSmall bool
	}{
		{"below width", 19, 20, true},
		{"below height", 20, 9, true},
		{"both below", 10, 5, true},
		{"exactly min", 20, 10, false},
		{"above min", 80, 24, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
// TestModel_View — RED
// =============================================================================

func TestModel_View_Menu(t *testing.T) {
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

func TestModel_View_TooSmall(t *testing.T) {
	m := NewModel(Deps{Version: "1.0.0"})
	m.width = 10
	m.height = 5
	m.tooSmall = true

	output := m.View().Content

	if !strings.Contains(output, "Terminal too small") {
		t.Errorf("View() output %q does not contain 'Terminal too small'", output)
	}
}

// =============================================================================
// TestModel_Selection — RED
// =============================================================================

func TestModel_Selection(t *testing.T) {
	tests := []struct {
		name     string
		cursor   int
		wantItem string
	}{
		{"first item", 0, "Create backup"},
		{"middle item", 3, "Cloud sync"},
		{"last item", 6, "Quit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
func TestModel_Init_PreservesDeps(t *testing.T) {
	m := NewModel(Deps{Version: "2.0.0-beta"})
	_ = m.Init()
	if m.deps.Version != "2.0.0-beta" {
		t.Errorf("after Init, Version = %q, want %q", m.deps.Version, "2.0.0-beta")
	}
}

// TestModel_Update_Quit_Esc verifies escape key also quits.
func TestModel_Update_Quit_Esc(t *testing.T) {
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
func TestModel_Update_SecondResize(t *testing.T) {
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
		t.Error("second resize: tooSmall = false, want true (width < 20)")
	}
}

// TestModel_View_Menu_WithCursor verifies cursor position changes output.
func TestModel_View_Menu_WithCursor(t *testing.T) {
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
// does not render menu items.
func TestModel_View_TooSmall_ShowsWarningOnly(t *testing.T) {
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
}

// TestModel_View_AltScreen verifies that View() always returns a tea.View
// with AltScreen=true, regardless of whether the terminal is too small.
func TestModel_View_AltScreen(t *testing.T) {
	tests := []struct {
		name     string
		width    int
		height   int
		tooSmall bool
	}{
		{"normal terminal", 80, 24, false},
		{"too small terminal", 10, 5, true},
		{"minimum viable", 20, 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
func TestModel_View_AltScreen_Triangulation(t *testing.T) {
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
func TestModel_Selection_EmptyMenuItems(t *testing.T) {
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
func TestModel_Selection_NilMenuItems(t *testing.T) {
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
func TestModel_Selection_Clamp(t *testing.T) {
	tests := []struct {
		name     string
		cursor   int
		wantItem string
	}{
		{"negative cursor", -5, "Create backup"},
		{"over bound cursor", 99, "Quit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
func TestModel_Update_ScreenDashboard(t *testing.T) {
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
func TestModel_Update_ScreenProgress(t *testing.T) {
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
func TestModel_View_Dashboard(t *testing.T) {
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
func TestModel_View_Progress(t *testing.T) {
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
func TestModel_Update_BackFromDashboard(t *testing.T) {
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
func TestModel_Update_ScreenSettings(t *testing.T) {
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

// TestModel_Update_ScreenShortcuts verifies that pressing ? on the
// main menu activates the shortcuts overlay.
func TestModel_Update_ScreenShortcuts(t *testing.T) {
	m := NewModel(Deps{Version: "1.0.0"})

	newModel, _ := m.Update(tea.KeyPressMsg{Code: '?'})
	result := newModel.(Model)

	if result.screen != ScreenShortcuts {
		t.Errorf("after '?': screen = %v, want ScreenShortcuts", result.screen)
	}
}

// TestModel_Update_SearchActivate verifies that pressing / on the
// dashboard activates the search component.
func TestModel_Update_SearchActivate(t *testing.T) {
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
func TestModel_Update_Toast(t *testing.T) {
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
func TestModel_View_Settings(t *testing.T) {
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
func TestModel_View_Shortcuts(t *testing.T) {
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
func TestModel_View_ToastOverlay(t *testing.T) {
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
func TestModel_BackFromShortcuts(t *testing.T) {
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
func TestModel_ScreenRoute_Health(t *testing.T) {
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

// newSettingsPtr returns a pointer to a freshly initialized SettingsModel.
func newSettingsPtr() *screens.SettingsModel {
	sm := screens.NewSettingsModel()
	return &sm
}

// newHealthPtr returns a pointer to a freshly initialized HealthModel.
func newHealthPtr() *screens.HealthModel {
	hm := screens.NewHealthModel()
	return &hm
}
