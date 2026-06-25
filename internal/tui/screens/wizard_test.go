// Package screens provides screen-specific TUI sub-models and render functions.
// This file contains white-box tests for the WizardModel moved from
// cmd/wizard_test.go so coverage is attributed to the screens package.
package screens

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// --- wizardModel step transitions ---

func TestWizardModel_Init(t *testing.T) {
	m := NewWizardModel("profile-create", nil)
	cmd := m.Init()
	if cmd != nil {
		t.Log("Init returned a command (expected for some modes)")
	}
	if m.CurrentStep() != StepName {
		t.Errorf("initial step = %d, want StepName (first step is name input)", m.CurrentStep())
	}
}

func TestWizardModel_StepTransitions(t *testing.T) {
	m := NewWizardModel("profile-create", []string{"github-gist", "codeberg"})

	steps := []WizardStep{StepName, StepProvider, StepPreset, StepAdapters, StepCategories, StepConfirm}

	for i, wantStep := range steps {
		got := m.CurrentStep()
		if got != wantStep {
			t.Fatalf("step %d: got %d, want %d", i, got, wantStep)
		}
		// Advance: simulate Enter key.
		if wantStep != StepConfirm {
			msg := tea.KeyPressMsg{Code: tea.KeyEnter}
			_, _ = m.Update(msg)
		}
	}
}

func TestWizardModel_ExitKeys(t *testing.T) {
	tests := []struct {
		name         string
		msg          tea.KeyPressMsg
		wantQuitting bool
		wantQuitCmd  bool // whether tea.Quit is returned
	}{
		{"ctrl+c", tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}, true, true},
		{"esc", tea.KeyPressMsg{Code: tea.KeyEsc}, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewWizardModel("profile-create", []string{"github-gist"})
			model, cmd := m.Update(tt.msg)

			if model.(*WizardModel).Quitting != tt.wantQuitting {
				t.Errorf("Quitting = %v, want %v", model.(*WizardModel).Quitting, tt.wantQuitting)
			}
			if tt.wantQuitCmd && cmd == nil {
				t.Error("expected tea.Quit command")
			}
		})
	}
}

// --- wizardModel name step ---

func TestWizardModel_NameStep_FirstStep(t *testing.T) {
	m := NewWizardModel("profile-create", []string{"github-gist", "codeberg"})

	// First step should be name input.
	if m.CurrentStep() != StepName {
		t.Errorf("first step = %d, want StepName", m.CurrentStep())
	}
}

func TestWizardModel_NameStep_EnterAdvances(t *testing.T) {
	m := NewWizardModel("profile-create", []string{"github-gist", "codeberg"})

	// Press Enter on name step → advances to provider.
	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	_, _ = m.Update(msg)

	if m.CurrentStep() != StepProvider {
		t.Errorf("after Enter on name step: step = %d, want StepProvider", m.CurrentStep())
	}
}

func TestWizardModel_NameStep_Typing(t *testing.T) {
	m := NewWizardModel("profile-create", []string{"github-gist"})

	// Type a name character by character.
	for _, r := range "my-profile" {
		msg := tea.KeyPressMsg{Code: r, Text: string(r)}
		_, _ = m.Update(msg)
	}

	if m.NameInput != "my-profile" {
		t.Errorf("NameInput = %q, want %q", m.NameInput, "my-profile")
	}
}

func TestWizardModel_NameStep_Backspace(t *testing.T) {
	m := NewWizardModel("profile-create", []string{"github-gist"})
	m.NameInput = "hello"

	// Press backspace.
	msg := tea.KeyPressMsg{Code: tea.KeyBackspace}
	_, _ = m.Update(msg)

	if m.NameInput != "hell" {
		t.Errorf("NameInput after backspace = %q, want %q", m.NameInput, "hell")
	}
}

func TestWizardModel_NameStep_BackspaceOnEmpty(t *testing.T) {
	m := NewWizardModel("profile-create", []string{"github-gist"})

	// Backspace on empty string should not panic.
	msg := tea.KeyPressMsg{Code: tea.KeyBackspace}
	_, _ = m.Update(msg)

	if m.NameInput != "" {
		t.Errorf("NameInput after backspace on empty = %q, want empty", m.NameInput)
	}
}

func TestWizardModel_NameStep_NamePersistsAcrossSteps(t *testing.T) {
	m := NewWizardModel("profile-create", []string{"github-gist"})

	// Type a name.
	for _, r := range "test-profile" {
		msg := tea.KeyPressMsg{Code: r, Text: string(r)}
		_, _ = m.Update(msg)
	}

	// Advance through all steps (name→provider→preset→adapters→categories→confirm).
	for i := 0; i < 5; i++ {
		msg := tea.KeyPressMsg{Code: tea.KeyEnter}
		_, _ = m.Update(msg)
	}

	if m.NameInput != "test-profile" {
		t.Errorf("NameInput after advancing = %q, want %q", m.NameInput, "test-profile")
	}
}

func TestWizardModel_ProfileName(t *testing.T) {
	tests := []struct {
		name             string
		providers        []string
		nameInput        string
		selectedProvider string
		want             string
	}{
		{"uses entered name", []string{"github-gist"}, "my-custom-profile", "codeberg", "my-custom-profile"},
		{"falls back to provider", []string{"github-gist"}, "", "github-gist", "github-gist"},
		{"falls back to untitled", nil, "", "", "untitled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewWizardModel("profile-create", tt.providers)
			m.NameInput = tt.nameInput
			m.SelectedProvider = tt.selectedProvider

			got := m.ProfileName()
			if got != tt.want {
				t.Errorf("ProfileName() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- wizardModel view ---

func TestWizardModel_View_ContainsTitle(t *testing.T) {
	m := NewWizardModel("profile-create", []string{"github-gist"})
	view := m.View().Content

	if !strings.Contains(view, "Create Profile") && !strings.Contains(view, "Profile") {
		t.Errorf("View should contain title, got: %s", view)
	}
}

func TestWizardModel_View_QuittingEmpty(t *testing.T) {
	m := NewWizardModel("profile-create", nil)
	m.Quitting = true

	view := m.View().Content
	if view != "" {
		t.Errorf("View should be empty when quitting, got: %q", view)
	}
}

// --- wizardModel selected values ---

func TestWizardModel_ProviderSelection(t *testing.T) {
	providers := []string{"github-gist", "codeberg", "gitea"}
	m := NewWizardModel("profile-create", providers)

	// Advance past the name step to the provider step.
	msg := tea.KeyPressMsg{Code: tea.KeyEnter}
	_, _ = m.Update(msg)

	// Initially cursor at 0.
	if m.ProviderCursor != 0 {
		t.Errorf("initial cursor = %d, want 0", m.ProviderCursor)
	}

	// Move down.
	msg = tea.KeyPressMsg{Code: 'j'}
	_, _ = m.Update(msg)
	if m.ProviderCursor != 1 {
		t.Errorf("cursor after down = %d, want 1", m.ProviderCursor)
	}

	// Move up.
	msg = tea.KeyPressMsg{Code: 'k'}
	_, _ = m.Update(msg)
	if m.ProviderCursor != 0 {
		t.Errorf("cursor after up = %d, want 0", m.ProviderCursor)
	}
}

// --- WindowSizeMsg handling ---

func TestWizardModel_Update_WindowSize(t *testing.T) {
	m := NewWizardModel("profile-create", []string{"github-gist"})

	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	result, _ := m.Update(msg)
	model := result.(*WizardModel)

	if model.Width != 100 {
		t.Errorf("Width = %d, want 100", model.Width)
	}
	if model.Height != 30 {
		t.Errorf("Height = %d, want 30", model.Height)
	}
}

func TestWizardModel_Update_WindowSize_SecondResize(t *testing.T) {
	m := NewWizardModel("profile-create", []string{"github-gist"})

	// First resize.
	msg1 := tea.WindowSizeMsg{Width: 100, Height: 30}
	m1, _ := m.Update(msg1)

	// Second resize — values should update.
	msg2 := tea.WindowSizeMsg{Width: 60, Height: 15}
	m2, _ := m1.Update(msg2)
	model := m2.(*WizardModel)

	if model.Width != 60 {
		t.Errorf("Width = %d, want 60", model.Width)
	}
	if model.Height != 15 {
		t.Errorf("Height = %d, want 15", model.Height)
	}
}

func TestMoveCursor(t *testing.T) {
	tests := []struct {
		name  string
		start int
		max   int
		key   string
		want  int
	}{
		// Down movements.
		{"down from 0", 0, 4, "down", 1},
		{"j from 0", 0, 4, "j", 1},
		{"down at max", 4, 4, "down", 4},
		{"j at max", 4, 4, "j", 4},
		{"down negative max", 0, -1, "down", 0},
		// Up movements.
		{"up from 3", 3, 4, "up", 2},
		{"k from 3", 3, 4, "k", 2},
		{"up at 0", 0, 4, "up", 0},
		{"k at 0", 0, 4, "k", 0},
		{"up negative max", 0, -1, "up", 0},
		// Unknown keys — no change.
		{"enter key ignored", 2, 4, "enter", 2},
		{"space key ignored", 1, 4, "space", 1},
		{"empty key ignored", 2, 4, "", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor := tt.start
			MoveCursor(&cursor, tt.max, tt.key)
			if cursor != tt.want {
				t.Errorf("MoveCursor(%d, %d, %q) = %d, want %d",
					tt.start, tt.max, tt.key, cursor, tt.want)
			}
		})
	}
}

// --- renderCheckboxList ---

// TestWizardModel_renderCheckboxList verifies the checkbox list renderer
// produces one rendered line per item, includes each item's label, and
// reflects the checked state via the shared checkbox component ([x] vs [ ]).
func TestWizardModel_renderCheckboxList(t *testing.T) {
	tests := []struct {
		name        string
		items       []ToggleItem
		cursor      int
		wantLabels  []string
		wantNewline int // number of '\n' separators between items
	}{
		{
			name:        "empty list renders nothing",
			items:       nil,
			cursor:      0,
			wantLabels:  nil,
			wantNewline: 0,
		},
		{
			name:        "single item has no separator",
			items:       []ToggleItem{{Name: "opencode", Checked: true}},
			cursor:      0,
			wantLabels:  []string{"opencode"},
			wantNewline: 0,
		},
		{
			name: "multiple items separated by newlines",
			items: []ToggleItem{
				{Name: "opencode", Checked: true},
				{Name: "claude-code", Checked: false},
				{Name: "codex", Checked: false},
			},
			cursor:      1,
			wantLabels:  []string{"opencode", "claude-code", "codex"},
			wantNewline: 2, // 3 items → 2 separators
		},
		{
			name: "out-of-bounds cursor still renders all items",
			items: []ToggleItem{
				{Name: "a", Checked: false},
				{Name: "b", Checked: true},
			},
			cursor:      99,
			wantLabels:  []string{"a", "b"},
			wantNewline: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewWizardModel("profile-create", nil)
			got := m.renderCheckboxList(tt.items, tt.cursor)

			for _, label := range tt.wantLabels {
				if !strings.Contains(got, label) {
					t.Errorf("renderCheckboxList output missing label %q\ngot: %q", label, got)
				}
			}

			if n := strings.Count(got, "\n"); n != tt.wantNewline {
				t.Errorf("renderCheckboxList newline count = %d, want %d\ngot: %q", n, tt.wantNewline, got)
			}

			if len(tt.items) == 0 && got != "" {
				t.Errorf("renderCheckboxList for empty list = %q, want empty string", got)
			}
		})
	}
}

// TestWizardModel_renderCheckboxList_CheckedState verifies that checked and
// unchecked items render their respective indicators from the checkbox
// component, proving the Checked field flows through to the output.
func TestWizardModel_renderCheckboxList_CheckedState(t *testing.T) {
	m := NewWizardModel("profile-create", nil)
	items := []ToggleItem{
		{Name: "checked-item", Checked: true},
		{Name: "unchecked-item", Checked: false},
	}

	got := m.renderCheckboxList(items, 0)

	if !strings.Contains(got, "[x]") {
		t.Errorf("renderCheckboxList missing checked indicator [x]\ngot: %q", got)
	}
	if !strings.Contains(got, "[ ]") {
		t.Errorf("renderCheckboxList missing unchecked indicator [ ]\ngot: %q", got)
	}
}

// --- renderConfirmSummary ---

// TestWizardModel_renderConfirmSummary verifies the confirm summary includes
// the selected provider, preset, only the checked adapters/categories, and
// the closing call-to-action line.
func TestWizardModel_renderConfirmSummary(t *testing.T) {
	m := NewWizardModel("profile-create", []string{"github"})
	m.SelectedProvider = "github"
	m.SelectedPreset = "full"
	m.AdapterItems = []ToggleItem{
		{Name: "opencode", Checked: true},
		{Name: "claude-code", Checked: false},
		{Name: "codex", Checked: true},
	}
	m.CategoryItems = []ToggleItem{
		{Name: "skills", Checked: true},
		{Name: "config", Checked: false},
	}

	got := m.renderConfirmSummary()

	wantContains := []string{
		"Provider:   github",
		"Preset:     full",
		"Adapters:   opencode, codex",
		"Categories: skills",
		"Press Enter to create the profile.",
	}
	for _, want := range wantContains {
		if !strings.Contains(got, want) {
			t.Errorf("renderConfirmSummary missing %q\ngot:\n%s", want, got)
		}
	}

	// Unchecked items must NOT appear in the joined lists.
	if strings.Contains(got, "Adapters:   opencode, claude-code") {
		t.Errorf("renderConfirmSummary included unchecked adapter claude-code\ngot:\n%s", got)
	}
	if strings.Contains(got, "config") {
		t.Errorf("renderConfirmSummary included unchecked category config\ngot:\n%s", got)
	}
}

// TestWizardModel_renderConfirmSummary_NoSelections verifies the summary
// renders cleanly when nothing is selected: empty provider/preset and empty
// adapter/category lists produce trailing empty values, not a panic.
func TestWizardModel_renderConfirmSummary_NoSelections(t *testing.T) {
	m := NewWizardModel("profile-create", nil)
	m.AdapterItems = nil
	m.CategoryItems = nil

	got := m.renderConfirmSummary()

	if !strings.Contains(got, "Provider:") {
		t.Errorf("renderConfirmSummary missing Provider label\ngot:\n%s", got)
	}
	if !strings.Contains(got, "Adapters:   \n") {
		t.Errorf("renderConfirmSummary should render empty adapters line\ngot:\n%q", got)
	}
	if !strings.Contains(got, "Press Enter to create the profile.") {
		t.Errorf("renderConfirmSummary missing call-to-action\ngot:\n%s", got)
	}
}
