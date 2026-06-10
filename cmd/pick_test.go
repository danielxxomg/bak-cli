package cmd

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

// --- runPickWithDeps wrapper tests ---

func TestRunPickWithDeps_NonTTYGuard(t *testing.T) {
	// Override isTTY to simulate non-interactive terminal.
	origIsTTY := isTTY
	isTTY = func() bool { return false }
	defer func() { isTTY = origIsTTY }()

	deps, _, _ := setupTestDeps(t)
	cmd := &cobra.Command{}
	err := runPickWithDeps(cmd, nil, deps)

	if err == nil {
		t.Fatal("expected error from non-TTY pick")
	}
	if !strings.Contains(err.Error(), "TTY") {
		t.Errorf("error should contain 'TTY', got: %v", err)
	}
}

func TestPickModel_Init(t *testing.T) {
	m := pickModel{
		items: []categoryItem{
			{name: "skills", checked: true},
			{name: "config", checked: false},
		},
	}

	cmd := m.Init()
	if cmd != nil {
		t.Errorf("Init() should return nil, got %v", cmd)
	}
}

func TestPickModel_Update_Quit(t *testing.T) {
	m := pickModel{
		items: []categoryItem{
			{name: "skills", checked: true},
		},
	}

	msg := tea.KeyPressMsg{Code: 'q'}
	result, cmd := m.Update(msg)
	model := result.(pickModel)

	if !model.quitting {
		t.Error("Expected quitting to be true after 'q' key")
	}
	if cmd == nil {
		t.Error("Expected Quit command")
	}
}

func TestPickModel_Update_CursorDown(t *testing.T) {
	m := pickModel{
		items: []categoryItem{
			{name: "skills", checked: true},
			{name: "config", checked: false},
			{name: "plugins", checked: false},
		},
		cursor: 0,
	}

	msg := tea.KeyPressMsg{Code: 'j'}
	result, _ := m.Update(msg)
	model := result.(pickModel)

	if model.cursor != 1 {
		t.Errorf("Expected cursor at 1, got %d", model.cursor)
	}
}

func TestPickModel_Update_CursorUp(t *testing.T) {
	m := pickModel{
		items: []categoryItem{
			{name: "skills", checked: true},
			{name: "config", checked: false},
		},
		cursor: 1,
	}

	msg := tea.KeyPressMsg{Code: 'k'}
	result, _ := m.Update(msg)
	model := result.(pickModel)

	if model.cursor != 0 {
		t.Errorf("Expected cursor at 0, got %d", model.cursor)
	}
}

func TestPickModel_Update_Toggle(t *testing.T) {
	m := pickModel{
		items: []categoryItem{
			{name: "skills", checked: false},
			{name: "config", checked: true},
		},
		cursor: 0,
	}

	msg := tea.KeyPressMsg{Code: ' '}
	result, _ := m.Update(msg)
	model := result.(pickModel)

	if !model.items[0].checked {
		t.Error("Expected first item to be checked after space")
	}
	if !model.items[1].checked {
		t.Error("Expected second item to remain checked")
	}
}

func TestPickModel_Update_Confirm(t *testing.T) {
	m := pickModel{
		items: []categoryItem{
			{name: "skills", checked: true},
		},
	}

	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	result, cmd := m.Update(msg)
	model := result.(pickModel)

	if !model.confirmed {
		t.Error("Expected confirmed to be true after enter")
	}
	if cmd == nil {
		t.Error("Expected Quit command")
	}
}

func TestPickModel_View(t *testing.T) {
	m := pickModel{
		items: []categoryItem{
			{name: "skills", checked: true},
			{name: "config", checked: false},
		},
		cursor: 0,
	}

	view := m.View().Content
	if view == "" {
		t.Error("View should not be empty")
	}
	if !strings.Contains(view, "skills") {
		t.Error("View should contain 'skills'")
	}
	if !strings.Contains(view, "config") {
		t.Error("View should contain 'config'")
	}
}

func TestPickModel_Selected(t *testing.T) {
	m := pickModel{
		items: []categoryItem{
			{name: "skills", checked: true},
			{name: "config", checked: false},
			{name: "plugins", checked: true},
		},
	}

	selected := m.Selected()
	if len(selected) != 2 {
		t.Fatalf("Expected 2 selected items, got %d", len(selected))
	}
	if selected[0] != "skills" || selected[1] != "plugins" {
		t.Errorf("Expected [skills, plugins], got %v", selected)
	}
}

func TestPickCmd_Structure(t *testing.T) {
	if pickCmd.Use != "pick" {
		t.Errorf("Expected Use 'pick', got %q", pickCmd.Use)
	}
	if pickCmd.Short == "" {
		t.Error("Short description should not be empty")
	}
	if pickCmd.RunE == nil {
		t.Error("RunE should be set")
	}
}

func TestPickCmd_Args(t *testing.T) {
	if pickCmd.Args != nil {
		t.Error("Pick command should not require arguments")
	}
}
