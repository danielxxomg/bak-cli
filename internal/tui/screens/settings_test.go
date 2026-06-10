package screens

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// =============================================================================
// TestNewSettingsModel — RED (settings.go does not exist yet)
// =============================================================================

func TestNewSettingsModel(t *testing.T) {
	m := NewSettingsModel()

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

func TestSettings_Update_Navigate(t *testing.T) {
	m := NewSettingsModel()
	maxIdx := len(m.options) - 1

	tests := []struct {
		name    string
		keys    []rune
		wantCur int
	}{
		{
			name:    "j moves down",
			keys:    []rune{'j'},
			wantCur: 1,
		},
		{
			name:    "k moves up",
			keys:    []rune{'j', 'j', 'k'},
			wantCur: 1,
		},
		{
			name:    "clamped at bottom",
			keys:    []rune{'j', 'j', 'j', 'j', 'j'},
			wantCur: maxIdx,
		},
		{
			name:    "clamped at top",
			keys:    []rune{'k', 'k'},
			wantCur: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cur := m
			for _, key := range tt.keys {
				newM, _ := cur.Update(tea.KeyPressMsg{Code: key})
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

func TestSettings_Update_Toggle(t *testing.T) {
	tests := []struct {
		name string
		key  rune
	}{
		{"enter toggles", '\r'},
		{"space toggles", ' '},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewSettingsModel()

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

func TestSettings_Update_Back(t *testing.T) {
	tests := []struct {
		name string
		key  rune
	}{
		{"q goes back", 'q'},
		{"esc goes back", 27},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewSettingsModel()

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

func TestSettings_View(t *testing.T) {
	m := NewSettingsModel()
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
// TestSettings_View_TooSmall — RED (triangulation)
// =============================================================================

func TestSettings_View_TooSmall(t *testing.T) {
	m := NewSettingsModel()
	m.width = 10
	m.height = 5

	output := m.View().Content

	if !strings.Contains(output, "Terminal too small") {
		t.Errorf("View() too-small missing warning: %q", output)
	}
}
