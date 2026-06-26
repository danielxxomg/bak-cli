package screens

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// newTestSettings creates a SettingsModel with no save function for testing.
func newTestSettings() SettingsModel {
	return NewSettingsModel(nil)
}

// =============================================================================
// TestNewSettingsModel — RED (settings.go does not exist yet)
// =============================================================================

func TestNewSettingsModel(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := newTestSettings()

	if m.cursor != 0 {
		t.Errorf("NewSettingsModel().cursor = %d, want 0", m.cursor)
	}

	if len(m.options) == 0 {
		t.Fatal("NewSettingsModel().options is empty, want defaults")
	}

	// Verify default options exist.
	labels := make(map[string]bool)
	for _, o := range m.options {
		labels[o.Label] = true
	}
	wantLabels := []string{"Cloud Provider", "Theme", "Auto-sync", "Verbose"}
	for _, label := range wantLabels {
		if !labels[label] {
			t.Errorf("NewSettingsModel missing option %q", label)
		}
	}
}

// =============================================================================
// TestSettings_Update_Navigate — RED
// =============================================================================

func TestSettings_Update_Navigate(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := newTestSettings()
	maxIdx := len(m.options) - 1 // 3

	tests := []struct {
		name    string
		keys    []tea.KeyPressMsg
		wantCur int
	}{
		{
			name:    "j moves down",
			keys:    []tea.KeyPressMsg{{Code: 'j'}},
			wantCur: 1,
		},
		{
			name:    "k moves up",
			keys:    []tea.KeyPressMsg{{Code: 'j'}, {Code: 'j'}, {Code: 'k'}},
			wantCur: 1,
		},
		{
			name:    "arrow down moves cursor",
			keys:    []tea.KeyPressMsg{{Code: tea.KeyDown}},
			wantCur: 1,
		},
		{
			name:    "arrow up moves cursor",
			keys:    []tea.KeyPressMsg{{Code: 'j'}, {Code: 'j'}, {Code: tea.KeyUp}},
			wantCur: 1,
		},
		{
			name: "wraps from last to first (j)",
			keys: []tea.KeyPressMsg{
				{Code: 'j'}, {Code: 'j'}, {Code: 'j'}, // 0→1→2→3
				{Code: 'j'}, // 3→0 (wrap)
			},
			wantCur: 0,
		},
		{
			name: "wraps from last to first (arrow down)",
			keys: []tea.KeyPressMsg{
				{Code: 'j'}, {Code: 'j'}, {Code: 'j'}, // 0→1→2→3
				{Code: tea.KeyDown}, // 3→0 (wrap)
			},
			wantCur: 0,
		},
		{
			name: "wraps from first to last (k)",
			keys: []tea.KeyPressMsg{
				{Code: 'k'}, // 0→3 (wrap)
			},
			wantCur: maxIdx,
		},
		{
			name: "wraps from first to last (arrow up)",
			keys: []tea.KeyPressMsg{
				{Code: tea.KeyUp}, // 0→3 (wrap)
			},
			wantCur: maxIdx,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			cur := m
			for _, key := range tt.keys {
				newM, _ := cur.Update(key)
				cur = newM.(SettingsModel)
			}
			if cur.cursor != tt.wantCur {
				t.Errorf("after keys %v: cursor = %d, want %d (max %d)",
					tt.keys, cur.cursor, tt.wantCur, maxIdx)
			}
		})
	}
}

// =============================================================================
// TestSettings_Update_Toggle — RED
// =============================================================================

func TestSettings_Update_Toggle(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name string
		key  rune
	}{
		{"enter toggles", '\r'},
		{"space toggles", ' '},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			m := newTestSettings()

			// Find a toggle option.
			var toggleIdx int
			for i, o := range m.options {
				if o.Type == "toggle" {
					toggleIdx = i
					break
				}
			}

			// Navigate to the toggle option.
			cur := m
			for cur.cursor < toggleIdx {
				newM, _ := cur.Update(tea.KeyPressMsg{Code: 'j'})
				cur = newM.(SettingsModel)
			}

			// Record initial state.
			initial := cur.options[toggleIdx].Value

			// Toggle.
			m2, _ := cur.Update(tea.KeyPressMsg{Code: tt.key})
			after := m2.(SettingsModel)

			toggled := after.options[toggleIdx].Value
			if toggled == initial {
				t.Errorf("%s did not change toggle state: was %v, still %v",
					tt.name, initial, toggled)
			}
		})
	}
}

// =============================================================================
// TestSettings_Update_Back — RED
// =============================================================================

func TestSettings_Update_Back(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name string
		key  rune
	}{
		{"q goes back", 'q'},
		{"esc goes back", 27},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			m := newTestSettings()

			_, cmd := m.Update(tea.KeyPressMsg{Code: tt.key})

			if cmd == nil {
				t.Fatalf("%s returned nil cmd, want ScreenBackMsg", tt.name)
			}
			msg := cmd()
			if _, ok := msg.(ScreenBackMsg); !ok {
				t.Errorf("%s returned %T, want ScreenBackMsg", tt.name, msg)
			}
		})
	}
}

// =============================================================================
// TestSettings_View — RED
// =============================================================================

func TestSettings_View(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := newTestSettings()
	m.width = 80
	m.height = 24

	output := m.View().Content

	if len(output) == 0 {
		t.Fatal("View() returned empty content")
	}

	// Must show each option label.
	for _, o := range m.options {
		if !strings.Contains(output, o.Label) {
			t.Errorf("View() missing option label %q", o.Label)
		}
	}

	// Must include toggle indicators.
	if !strings.Contains(output, "\u2713") && !strings.Contains(output, "[x]") && !strings.Contains(output, "\u25CB") {
		t.Error("View() missing toggle indicators")
	}
}

// =============================================================================
// TestSettings_View_TooSmall — threshold guard at 40×12
// =============================================================================

func TestSettings_View_MinSizeGuard(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
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
			m := newTestSettings()
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

// TestSettings_View_TooSmall verifies the too-small view shows
// the dimensional message (legacy test, kept for coverage).
func TestSettings_View_TooSmall(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := newTestSettings()
	m.width = 10
	m.height = 5

	output := m.View().Content

	if !strings.Contains(output, "Terminal too small") {
		t.Errorf("View() too-small missing warning: %q", output)
	}
}

// =============================================================================
// TestSettings_View_HelpBar — RED (Phase 3: help bar persistence)
// =============================================================================

func TestSettings_View_HelpBar(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := newTestSettings()
	m.width = 80
	m.height = 24

	output := m.View().Content

	if len(output) == 0 {
		t.Fatal("View() returned empty content")
	}

	// Help bar must contain contextual keys: ↑/↓ navigate • enter toggle • q back
	if !strings.Contains(output, "navigate") {
		t.Errorf("View() help bar missing 'navigate': %q", output)
	}
	if !strings.Contains(output, "toggle") {
		t.Errorf("View() help bar missing 'toggle': %q", output)
	}
	if !strings.Contains(output, "back") {
		t.Errorf("View() help bar missing 'back': %q", output)
	}
}

// TestSettings_View_HelpBar_LiteralKeys verifies the literal key symbols
// appear in the help bar output (triangulation for TestSettings_View_HelpBar).
func TestSettings_View_HelpBar_LiteralKeys(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := newTestSettings()
	m.width = 80
	m.height = 24

	output := m.View().Content

	// The arrow symbol must be rendered literally.
	if !strings.Contains(output, "↑") || !strings.Contains(output, "↓") {
		t.Errorf("View() help bar missing arrow symbols '↑/↓': %q", output)
	}
	// "enter" key label must appear.
	if !strings.Contains(output, "enter") {
		t.Errorf("View() help bar missing 'enter' key: %q", output)
	}
}

// =============================================================================
// Settings Persistence Tests — RED (SaveSetting not wired yet)
// =============================================================================

// TestSettings_SaveSetting_CalledOnToggle verifies that toggling an option
// calls the injected SaveSetting function with the correct key and value.
func TestSettings_SaveSetting_CalledOnToggle(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	var savedKey string
	var savedValue any

	saveFn := func(key string, value any) error {
		savedKey = key
		savedValue = value
		return nil
	}

	m := NewSettingsModel(saveFn)

	// Navigate to "Auto-sync" (index 2 in default options).
	for i := 0; i < 2; i++ {
		newM, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
		m = newM.(SettingsModel)
	}

	initial := m.options[2].Value // should be false (default)

	// Toggle with enter.
	newM, _ := m.Update(tea.KeyPressMsg{Code: '\r'})
	m = newM.(SettingsModel)

	// Verify toggle changed.
	if m.options[2].Value == initial {
		t.Error("toggle did not change value")
	}

	// Verify SaveSetting was called.
	if savedKey != "auto_sync" {
		t.Errorf("SaveSetting key = %q, want %q", savedKey, "auto_sync")
	}
	if savedValue == nil {
		t.Error("SaveSetting value is nil, want non-nil")
	}
	boolVal, ok := savedValue.(bool)
	if !ok || !boolVal {
		t.Errorf("SaveSetting value = %v (%T), want true", savedValue, savedValue)
	}
}

// TestSettings_LoadInitialSettings verifies that NewSettingsModel uses
// the provided initial Settings values to set the option toggles.
func TestSettings_LoadInitialSettings(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	saveFn := func(key string, value any) error { return nil }

	s := Settings{
		AutoSync:           true,
		DefaultPreset:      "full",
		MaxFileSize:        2097152,
		ConfirmDestructive: false,
		VerboseDefault:     true,
		DefaultProvider:    "github",
	}

	m := NewSettingsModelWithSettings(s, saveFn)

	// Auto-sync should be true (index 2).
	if !m.options[2].Value {
		t.Error("Auto-sync toggle = false, want true (from Settings)")
	}

	// Verbose should be true (index 3).
	if !m.options[3].Value {
		t.Error("Verbose toggle = false, want true (from Settings)")
	}
}

// TestSettings_SaveSetting_ErrorsAreIgnored verifies that even when
// SaveSetting returns an error, the toggle still updates locally.
func TestSettings_SaveSetting_ErrorsAreIgnored(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	saveFn := func(key string, value any) error {
		return fmt.Errorf("write failed")
	}

	m := NewSettingsModel(saveFn)

	// Navigate to Auto-sync (index 2) and toggle.
	for i := 0; i < 2; i++ {
		newM, _ := m.Update(tea.KeyPressMsg{Code: 'j'})
		m = newM.(SettingsModel)
	}

	newM, _ := m.Update(tea.KeyPressMsg{Code: '\r'})
	m = newM.(SettingsModel)

	// Toggle should still have changed locally even though save failed.
	if !m.options[2].Value {
		t.Error("Auto-sync toggle = false after toggle, want true (local state updates regardless of save error)")
	}
}

// =============================================================================
// Phase 3: Init nil-return coverage
// =============================================================================

func TestSettingsModel_Init_ReturnsNil(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	m := newTestSettings()
	cmd := m.Init()
	if cmd != nil {
		t.Errorf("Init() = %v, want nil", cmd)
	}
}
