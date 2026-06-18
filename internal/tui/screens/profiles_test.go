// Package screens provides screen-specific render functions and sub-models
// for the bak-cli TUI. This file contains strict-TDD tests for ProfilesModel
// written BEFORE the production code.
package screens

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/danielxxomg/bak-cli/internal/tui/components"
)

// =============================================================================
// TestProfiles_NewProfilesModel — RED
// =============================================================================

func TestProfiles_NewProfilesModel(t *testing.T) {
	listFn := func() ([]ProfileInfo, error) { return nil, nil }
	m := NewProfilesModel(listFn, nil, nil, nil)

	if m.Cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", m.Cursor)
	}
	if len(m.Profiles) != 0 {
		t.Errorf("initial profiles length = %d, want 0", len(m.Profiles))
	}
}

// =============================================================================
// TestProfiles_Init_LoadsProfiles — RED
// =============================================================================

func TestProfiles_Init_LoadsProfiles(t *testing.T) {
	listFn := func() ([]ProfileInfo, error) {
		return []ProfileInfo{
			{Name: "work", Provider: "github", Preset: "full", Active: true},
			{Name: "personal", Provider: "github", Preset: "quick", Active: false},
		}, nil
	}

	m := NewProfilesModel(listFn, nil, nil, nil)

	cmd := m.Init()
	msg := cmd()
	newModel, _ := m.Update(msg)
	pm := newModel.(ProfilesModel)

	if len(pm.Profiles) != 2 {
		t.Fatalf("profiles length = %d, want 2", len(pm.Profiles))
	}
	if pm.Profiles[0].Name != "work" {
		t.Errorf("profiles[0].Name = %q, want %q", pm.Profiles[0].Name, "work")
	}
}

// =============================================================================
// TestProfiles_EmptyState — RED
// =============================================================================

func TestProfiles_EmptyState(t *testing.T) {
	listFn := func() ([]ProfileInfo, error) { return nil, nil }
	m := NewProfilesModel(listFn, nil, nil, nil)
	m.Width = 80
	m.Height = 24

	output := m.View().Content

	if !strings.Contains(output, "No profiles yet") {
		t.Errorf("empty state missing 'No profiles yet': %q", output)
	}
}

// =============================================================================
// TestProfiles_SwitchActiveProfile — RED
// =============================================================================

func TestProfiles_SwitchActiveProfile(t *testing.T) {
	var switchedTo string
	switchFn := func(name string) error {
		switchedTo = name
		return nil
	}

	listFn := func() ([]ProfileInfo, error) {
		return []ProfileInfo{
			{Name: "work", Provider: "github", Preset: "full", Active: false},
			{Name: "personal", Provider: "github", Preset: "quick", Active: true},
		}, nil
	}

	m := NewProfilesModel(listFn, switchFn, nil, nil)
	m.Width = 80
	m.Height = 24
	m.Profiles = []ProfileInfo{
		{Name: "work", Provider: "github", Preset: "full", Active: false},
		{Name: "personal", Provider: "github", Preset: "quick", Active: true},
	}

	// Enter on "work" (cursor 0).
	newModel, _ := m.Update(tea.KeyPressMsg{Code: '\r'})
	pm := newModel.(ProfilesModel)

	if switchedTo != "work" {
		t.Errorf("SetActiveProfile called with %q, want %q", switchedTo, "work")
	}
	// Work should now be active.
	if !pm.Profiles[0].Active {
		t.Error("work profile should be active after switch")
	}
}

// =============================================================================
// TestProfiles_DeleteNonActive — RED
// =============================================================================

func TestProfiles_DeleteNonActive(t *testing.T) {
	var deletedName string
	deleteFn := func(name string) error {
		deletedName = name
		return nil
	}

	m := NewProfilesModel(nil, nil, deleteFn, nil)
	m.Width = 80
	m.Height = 24
	m.Profiles = []ProfileInfo{
		{Name: "work", Provider: "github", Preset: "full", Active: true},
		{Name: "personal", Provider: "github", Preset: "quick", Active: false},
	}
	m.Cursor = 1 // select "personal"

	// Press 'd' to delete.
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'd'})
	pm := newModel.(ProfilesModel)

	// Should show a confirmation modal.
	if pm.Modal == nil {
		t.Error("Modal should be shown for delete confirmation")
	}

	// Confirm delete.
	newModel, _ = pm.Update(components.ModalResultMsg{Confirmed: true})
	pm = newModel.(ProfilesModel)

	if deletedName != "personal" {
		t.Errorf("DeleteProfile called with %q, want %q", deletedName, "personal")
	}
}

// =============================================================================
// TestProfiles_CannotDeleteActive — RED
// =============================================================================

func TestProfiles_CannotDeleteActive(t *testing.T) {
	var deleteCalled string
	deleteFn := func(name string) error {
		deleteCalled = name
		return nil
	}

	m := NewProfilesModel(nil, nil, deleteFn, nil)
	m.Width = 80
	m.Height = 24
	m.Profiles = []ProfileInfo{
		{Name: "work", Provider: "github", Preset: "full", Active: true},
	}
	m.Cursor = 0 // active profile

	// Press 'd' on active profile.
	newModel, _ := m.Update(tea.KeyPressMsg{Code: 'd'})
	pm := newModel.(ProfilesModel)

	// No modal should appear, no delete called, toast message should be set.
	if pm.Modal != nil {
		t.Error("Modal should NOT be shown for active profile delete")
	}
	if deleteCalled != "" {
		t.Errorf("DeleteProfile should NOT be called for active profile, got %q", deleteCalled)
	}
}

// =============================================================================
// TestProfiles_CreateViaWizard — RED
// =============================================================================

func TestProfiles_CreateViaWizard(t *testing.T) {
	var wizardCalled bool
	wizardFn := func() (ProfileInfo, error) {
		wizardCalled = true
		return ProfileInfo{Name: "new-profile", Provider: "github", Preset: "full", Active: false}, nil
	}
	saveFn := func(name string, p ProfileInfo) error { return nil }

	m := NewProfilesModel(nil, nil, nil, wizardFn)
	m.SaveProfile = saveFn
	m.Width = 80
	m.Height = 24
	m.Profiles = []ProfileInfo{}

	// Press 'n' to create via wizard.
	newModel, cmd := m.Update(tea.KeyPressMsg{Code: 'n'})
	pm := newModel.(ProfilesModel)

	if cmd != nil {
		msg := cmd()
		newModel2, _ := pm.Update(msg)
		pm = newModel2.(ProfilesModel)
	}

	if !wizardCalled {
		t.Error("RunWizard should have been called")
	}
	if len(pm.Profiles) != 1 {
		t.Errorf("profiles length = %d, want 1 (new profile added)", len(pm.Profiles))
	}
}

// =============================================================================
// TestProfiles_ListError — RED
// =============================================================================

func TestProfiles_ListError(t *testing.T) {
	listFn := func() ([]ProfileInfo, error) {
		return nil, errors.New("connection refused")
	}
	m := NewProfilesModel(listFn, nil, nil, nil)
	m.Width = 80
	m.Height = 24

	cmd := m.Init()
	msg := cmd()
	newModel, _ := m.Update(msg)
	pm := newModel.(ProfilesModel)

	if pm.Err == nil {
		t.Error("error should be set when listProfiles fails")
	}
}

// =============================================================================
// TestProfiles_View_List — RED
// =============================================================================

func TestProfiles_View_List(t *testing.T) {
	m := NewProfilesModel(nil, nil, nil, nil)
	m.Width = 80
	m.Height = 24
	m.Profiles = []ProfileInfo{
		{Name: "work", Provider: "github", Preset: "full", Active: true},
		{Name: "personal", Provider: "github", Preset: "quick", Active: false},
	}

	output := m.View().Content

	if !strings.Contains(output, "work") {
		t.Errorf("view missing profile 'work': %q", output)
	}
	if !strings.Contains(output, "personal") {
		t.Errorf("view missing profile 'personal': %q", output)
	}
	// Active profile should have a marker.
	if !strings.Contains(output, "Active") && !strings.Contains(output, "*") {
		t.Errorf("view missing active marker: %q", output)
	}
}

// =============================================================================
// Phase 3: Render error state coverage
// =============================================================================

func TestProfiles_RenderError(t *testing.T) {
	m := NewProfilesModel(nil, nil, nil, nil)
	m.Width = 80
	m.Height = 24
	m.Err = errors.New("connection refused")

	output := m.View().Content

	// Error state shows the error message.
	if !strings.Contains(output, "connection refused") {
		t.Errorf("renderError missing error message: %q", output)
	}
	if !strings.Contains(output, "Error") {
		t.Errorf("renderError missing 'Error': %q", output)
	}
	if !strings.Contains(output, "Profiles") {
		t.Errorf("renderError missing title: %q", output)
	}
}
