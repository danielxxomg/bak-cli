package cmd

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// --- wizardModel step transitions ---

func TestWizardModel_Init(t *testing.T) {
	m := newWizardModel("profile-create", nil)
	cmd := m.Init()
	if cmd != nil {
		t.Log("Init returned a command (expected for some modes)")
	}
	if m.CurrentStep() != stepProvider {
		t.Errorf("initial step = %d, want stepProvider", m.CurrentStep())
	}
}

func TestWizardModel_StepTransitions(t *testing.T) {
	m := newWizardModel("profile-create", []string{"github-gist", "codeberg"})

	steps := []wizardStep{stepProvider, stepPreset, stepAdapters, stepCategories, stepConfirm}

	for i, wantStep := range steps {
		got := m.CurrentStep()
		if got != wantStep {
			t.Fatalf("step %d: got %d, want %d", i, got, wantStep)
		}
		// Advance: simulate Enter key.
		if wantStep != stepConfirm {
			msg := tea.KeyPressMsg{Code: tea.KeyEnter}
			_, _ = m.Update(msg)
		}
	}
}

func TestWizardModel_CtrlC_Exits(t *testing.T) {
	m := newWizardModel("profile-create", []string{"github-gist"})

	msg := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	model, cmd := m.Update(msg)

	if model.(*wizardModel).quitting != true {
		t.Error("wizardModel.quitting should be true after Ctrl+C")
	}
	if cmd == nil {
		t.Error("Ctrl+C should return tea.Quit")
	}
}

func TestWizardModel_Esc_Exits(t *testing.T) {
	m := newWizardModel("profile-create", []string{"github-gist"})

	msg := tea.KeyPressMsg{Code: tea.KeyEsc}
	model, _ := m.Update(msg)

	if model.(*wizardModel).quitting != true {
		t.Error("wizardModel.quitting should be true after Esc")
	}
}

// --- wizardModel view ---

func TestWizardModel_View_ContainsTitle(t *testing.T) {
	m := newWizardModel("profile-create", []string{"github-gist"})
	view := m.View().Content

	if !strings.Contains(view, "Create Profile") && !strings.Contains(view, "Profile") {
		t.Errorf("View should contain title, got: %s", view)
	}
}

func TestWizardModel_View_QuittingEmpty(t *testing.T) {
	m := newWizardModel("profile-create", nil)
	m.quitting = true

	view := m.View().Content
	if view != "" {
		t.Errorf("View should be empty when quitting, got: %q", view)
	}
}

// --- wizardModel selected values ---

func TestWizardModel_ProviderSelection(t *testing.T) {
	providers := []string{"github-gist", "codeberg", "gitea"}
	m := newWizardModel("profile-create", providers)

	// Initially cursor at 0.
	if m.providerCursor != 0 {
		t.Errorf("initial cursor = %d, want 0", m.providerCursor)
	}

	// Move down.
	msg := tea.KeyPressMsg{Code: 'j'}
	_, _ = m.Update(msg)
	if m.providerCursor != 1 {
		t.Errorf("cursor after down = %d, want 1", m.providerCursor)
	}

	// Move up.
	msg = tea.KeyPressMsg{Code: 'k'}
	_, _ = m.Update(msg)
	if m.providerCursor != 0 {
		t.Errorf("cursor after up = %d, want 0", m.providerCursor)
	}
}

// --- TTY detection ---

func TestIsTTY_NotTerminal(t *testing.T) {
	// In test environment, os.Stdin is typically not a terminal.
	// We test the function exists and returns a boolean.
	result := isTTY()
	// Just check it doesn't panic.
	_ = result
}

// --- WindowSizeMsg handling ---

func TestWizardModel_Update_WindowSize(t *testing.T) {
	m := newWizardModel("profile-create", []string{"github-gist"})

	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	result, _ := m.Update(msg)
	model := result.(*wizardModel)

	if model.width != 100 {
		t.Errorf("width = %d, want 100", model.width)
	}
	if model.height != 30 {
		t.Errorf("height = %d, want 30", model.height)
	}
}

func TestWizardModel_Update_WindowSize_SecondResize(t *testing.T) {
	m := newWizardModel("profile-create", []string{"github-gist"})

	// First resize.
	msg1 := tea.WindowSizeMsg{Width: 100, Height: 30}
	m1, _ := m.Update(msg1)

	// Second resize — values should update.
	msg2 := tea.WindowSizeMsg{Width: 60, Height: 15}
	m2, _ := m1.Update(msg2)
	model := m2.(*wizardModel)

	if model.width != 60 {
		t.Errorf("width = %d, want 60", model.width)
	}
	if model.height != 15 {
		t.Errorf("height = %d, want 15", model.height)
	}
}
