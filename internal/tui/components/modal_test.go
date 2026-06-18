// Package components provides reusable TUI components for the bak-cli TUI.
// This file contains strict-TDD tests for ModalModel written BEFORE the
// production code.
package components

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// =============================================================================
// TestModal_New — table-driven creation tests
// =============================================================================

func TestModal_NewModal(t *testing.T) {
	tests := []struct {
		name          string
		title         string
		message       string
		buttons       []string
		wantCursor    int
		wantBtnLen    int
		wantBtnFirst  string
		wantBtnSecond string
	}{
		{
			name:          "two buttons",
			title:         "Confirm Restore",
			message:       "This will overwrite current config.",
			buttons:       []string{"Confirm", "Cancel"},
			wantCursor:    0,
			wantBtnLen:    2,
			wantBtnFirst:  "Confirm",
			wantBtnSecond: "Cancel",
		},
		{
			name:       "single button",
			title:      "Done",
			message:    "Backup created",
			buttons:    []string{"OK"},
			wantCursor: 0,
			wantBtnLen: 1,
		},
		{
			name:       "nil buttons",
			title:      "Alert",
			message:    "No actions",
			buttons:    nil,
			wantCursor: 0,
			wantBtnLen: 0,
		},
		{
			name:       "empty buttons",
			title:      "Alert",
			message:    "No actions",
			buttons:    []string{},
			wantCursor: 0,
			wantBtnLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModal(tt.title, tt.message, tt.buttons)

			if m.Title != tt.title {
				t.Errorf("Title = %q, want %q", m.Title, tt.title)
			}
			if m.Message != tt.message {
				t.Errorf("Message = %q, want %q", m.Message, tt.message)
			}
			if len(m.Buttons) != tt.wantBtnLen {
				t.Errorf("Buttons length = %d, want %d", len(m.Buttons), tt.wantBtnLen)
			}
			if m.cursor != tt.wantCursor {
				t.Errorf("cursor = %d, want %d", m.cursor, tt.wantCursor)
			}
		})
	}
}

// =============================================================================
// TestModal_Init
// =============================================================================

func TestModal_Init(t *testing.T) {
	m := NewModal("Test", "message", []string{"OK"})
	cmd := m.Init()
	if cmd != nil {
		t.Errorf("Init() returned non-nil cmd, want nil")
	}
}

// =============================================================================
// TestModal_Update_Enter — table-driven
// =============================================================================

func TestModal_Update_Enter(t *testing.T) {
	tests := []struct {
		name        string
		cursor      int
		wantConfirm bool
	}{
		{"cursor on Confirm (0)", 0, true},
		{"cursor on Cancel (1)", 1, false},
		{"single OK button (0)", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModal("Title", "msg", []string{"Confirm", "Cancel"})
			m.cursor = tt.cursor

			_, cmd := m.Update(tea.KeyPressMsg{Code: '\r'})
			if cmd == nil {
				t.Fatal("Update(Enter) returned nil cmd")
			}
			msg := cmd()
			result, ok := msg.(ModalResultMsg)
			if !ok {
				t.Fatalf("returned %T, want ModalResultMsg", msg)
			}
			if result.Confirmed != tt.wantConfirm {
				t.Errorf("Confirmed = %v, want %v", result.Confirmed, tt.wantConfirm)
			}
		})
	}
}

// =============================================================================
// TestModal_Update_Escape — table-driven
// =============================================================================

func TestModal_Update_Escape(t *testing.T) {
	tests := []struct {
		name string
		code rune
	}{
		{"KeyEscape", tea.KeyEscape},
		{"KeyEsc", tea.KeyEsc},
		{"ASCII 27", 27},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModal("Title", "msg", []string{"Confirm", "Cancel"})

			_, cmd := m.Update(tea.KeyPressMsg{Code: tt.code})
			if cmd == nil {
				t.Fatal("Update(Esc) returned nil cmd")
			}
			msg := cmd()
			result, ok := msg.(ModalResultMsg)
			if !ok {
				t.Fatalf("returned %T, want ModalResultMsg", msg)
			}
			if result.Confirmed {
				t.Error("Confirmed = true, want false (Esc cancels)")
			}
		})
	}
}

// =============================================================================
// TestModal_Update_TabCycling — table-driven
// =============================================================================

func TestModal_Update_TabCycling(t *testing.T) {
	tests := []struct {
		name     string
		buttons  []string
		startCur int
		nPresses int
		shift    bool
		wantCur  int
	}{
		{"tab 0→1 3 buttons", []string{"A", "B", "C"}, 0, 1, false, 1},
		{"tab 0→1→2 3 buttons", []string{"A", "B", "C"}, 0, 2, false, 2},
		{"tab wrap 3 buttons", []string{"A", "B", "C"}, 2, 1, false, 0},
		{"shift+tab 2→1", []string{"A", "B", "C"}, 2, 1, true, 1},
		{"shift+tab wrap to last", []string{"A", "B", "C"}, 0, 1, true, 2},
		{"tab 2 buttons", []string{"Yes", "No"}, 0, 1, false, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModal("Title", "msg", tt.buttons)
			m.cursor = tt.startCur

			for i := 0; i < tt.nPresses; i++ {
				mod := tea.ModShift
				if !tt.shift {
					mod = 0
				}
				newM, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: mod})
				m = newM.(ModalModel)
			}

			if m.cursor != tt.wantCur {
				t.Errorf("cursor = %d, want %d", m.cursor, tt.wantCur)
			}
		})
	}
}

// =============================================================================
// TestModal_Update_EmptyButtons — edge case
// =============================================================================

func TestModal_Update_EmptyButtons(t *testing.T) {
	tests := []struct {
		name    string
		buttons []string
	}{
		{"nil buttons", nil},
		{"empty buttons", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModal("Title", "msg", tt.buttons)

			// Tab should not panic on empty buttons.
			newM, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
			result := newM.(ModalModel)
			if result.cursor != 0 {
				t.Errorf("cursor = %d, want 0", result.cursor)
			}

			// Enter should not panic.
			_, cmd := m.Update(tea.KeyPressMsg{Code: '\r'})
			if cmd == nil {
				t.Error("Enter on empty buttons should still emit ModalResultMsg")
			}
		})
	}
}

// =============================================================================
// TestModal_View — table-driven
// =============================================================================

func TestModal_View_ContainsElements(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		message string
		buttons []string
		wants   []string
	}{
		{
			name:    "two buttons",
			title:   "Confirm Delete",
			message: "Are you sure?",
			buttons: []string{"Yes", "No"},
			wants:   []string{"Confirm Delete", "Are you sure?", "Yes", "No"},
		},
		{
			name:    "single button",
			title:   "Done",
			message: "Backup created successfully",
			buttons: []string{"OK"},
			wants:   []string{"Done", "Backup created successfully", "OK"},
		},
		{
			name:    "nil buttons still render title+msg",
			title:   "Alert",
			message: "Something happened",
			buttons: nil,
			wants:   []string{"Alert", "Something happened"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModal(tt.title, tt.message, tt.buttons)
			m.Width = 80
			m.Height = 24

			output := m.View().Content
			if len(output) == 0 {
				t.Fatal("View() returned empty string")
			}
			for _, want := range tt.wants {
				if !strings.Contains(output, want) {
					t.Errorf("View() missing %q: %q", want, output)
				}
			}
		})
	}
}

// =============================================================================
// TestModal_View_NarrowTerminal — table-driven
// =============================================================================

func TestModal_View_NarrowTerminal(t *testing.T) {
	tests := []struct {
		name             string
		width            int
		height           int
		wantTooSmall     bool
		wantStillPresent string
	}{
		{"normal 80x24", 80, 24, false, "Long Title Here"},
		{"narrow 35x15 (below 40 but above min)", 35, 15, false, "Long Title Here"},
		{"too small 15x8", 15, 8, true, "Terminal too small for modal"},
		{"too narrow 19x30", 19, 30, true, "Terminal too small for modal"},
		{"too short 40x9", 40, 9, true, "Terminal too small for modal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModal("Long Title Here", "Some message", []string{"OK", "Cancel"})
			m.Width = tt.width
			m.Height = tt.height

			output := m.View().Content
			if len(output) == 0 {
				t.Fatal("View() returned empty string")
			}

			if tt.wantTooSmall {
				if !strings.Contains(output, "Terminal too small for modal") {
					t.Errorf("View() missing too-small message: %q", output)
				}
			} else {
				if !strings.Contains(output, tt.wantStillPresent) {
					t.Errorf("View() missing %q: %q", tt.wantStillPresent, output)
				}
			}
		})
	}
}

// =============================================================================
// TestModal_View_CursorHighlights — table-driven
// =============================================================================

func TestModal_View_CursorHighlightsFocused(t *testing.T) {
	m := NewModal("Title", "msg", []string{"First", "Second"})
	m.Width = 80
	m.Height = 24

	out0 := m.View().Content
	if out0 == "" {
		t.Fatal("View() returned empty at cursor 0")
	}

	// Move cursor to 1.
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	out1 := m2.(ModalModel).View().Content
	if out1 == "" {
		t.Fatal("View() returned empty at cursor 1")
	}

	if out0 == out1 {
		t.Error("View() output identical for cursor 0 and cursor 1, expected different focus")
	}
}

// =============================================================================
// TestModal_WindowSize — handles window resizing
// =============================================================================

func TestModal_WindowSize(t *testing.T) {
	m := NewModal("Title", "msg", []string{"OK"})

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	result := newModel.(ModalModel)

	if result.Width != 100 {
		t.Errorf("Width = %d, want 100", result.Width)
	}
	if result.Height != 30 {
		t.Errorf("Height = %d, want 30", result.Height)
	}
}
