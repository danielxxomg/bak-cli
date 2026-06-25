package cmd

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/danielxxomg/bak-cli/internal/actions"
	"github.com/danielxxomg/bak-cli/internal/tui/screens"
)

// TestIsTTY_NotTerminal verifies the isTTY function exists and returns
// a boolean without panicking. In test environments, os.Stdin is typically
// not a terminal so the exact return value is environment-dependent.
func TestIsTTY_NotTerminal(t *testing.T) {
	result := isTTY()
	// Just check it doesn't panic.
	_ = result
}

// equalStrings is a small helper for comparing string slices in tests.
func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestWizardSelections verifies the pure helper that collects checked
// adapters/categories from a completed wizard model into a
// ProfileCreateFromWizard value.
func TestWizardSelections(t *testing.T) {
	tests := []struct {
		name string
		wm   *screens.WizardModel
		want actions.ProfileCreateFromWizard
	}{
		{
			name: "checked adapters and categories collected in order",
			wm: &screens.WizardModel{
				Confirmed:        true,
				SelectedProvider: "github-gist",
				SelectedPreset:   "full",
				AdapterItems: []screens.ToggleItem{
					{Name: "opencode", Checked: true},
					{Name: "cursor", Checked: false},
					{Name: "codex", Checked: true},
				},
				CategoryItems: []screens.ToggleItem{
					{Name: "skills", Checked: true},
					{Name: "commands", Checked: false},
				},
			},
			want: actions.ProfileCreateFromWizard{
				Confirmed:        true,
				SelectedProvider: "github-gist",
				SelectedPreset:   "full",
				AdapterNames:     []string{"opencode", "codex"},
				CategoryNames:    []string{"skills"},
			},
		},
		{
			name: "none checked yields empty selections",
			wm: &screens.WizardModel{
				Confirmed:        false,
				SelectedProvider: "codeberg",
				SelectedPreset:   "quick",
				AdapterItems: []screens.ToggleItem{
					{Name: "opencode", Checked: false},
				},
				CategoryItems: []screens.ToggleItem{
					{Name: "skills", Checked: false},
				},
			},
			want: actions.ProfileCreateFromWizard{
				Confirmed:        false,
				SelectedProvider: "codeberg",
				SelectedPreset:   "quick",
				AdapterNames:     nil,
				CategoryNames:    nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wizardSelections(tt.wm)
			if got.Confirmed != tt.want.Confirmed {
				t.Errorf("Confirmed = %v, want %v", got.Confirmed, tt.want.Confirmed)
			}
			if got.SelectedProvider != tt.want.SelectedProvider {
				t.Errorf("SelectedProvider = %q, want %q", got.SelectedProvider, tt.want.SelectedProvider)
			}
			if got.SelectedPreset != tt.want.SelectedPreset {
				t.Errorf("SelectedPreset = %q, want %q", got.SelectedPreset, tt.want.SelectedPreset)
			}
			if !equalStrings(got.AdapterNames, tt.want.AdapterNames) {
				t.Errorf("AdapterNames = %v, want %v", got.AdapterNames, tt.want.AdapterNames)
			}
			if !equalStrings(got.CategoryNames, tt.want.CategoryNames) {
				t.Errorf("CategoryNames = %v, want %v", got.CategoryNames, tt.want.CategoryNames)
			}
		})
	}
}

// otherModel is a tea.Model that is NOT a *screens.WizardModel, used to
// exercise launchWizard's type-assertion error path.
type otherModel struct{}

func (otherModel) Init() tea.Cmd                       { return nil }
func (otherModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return otherModel{}, nil }
func (otherModel) View() tea.View                      { return tea.NewView("") }

// TestLaunchWizard verifies the extracted wizard-launch helper: TTY gate,
// program-run error wrapping, unexpected-model-type error, and the happy
// path returning selections plus the final wizard model.
func TestLaunchWizard(t *testing.T) {
	origTTY := isTTY
	origRun := runWizardProgram
	defer func() {
		isTTY = origTTY
		runWizardProgram = origRun
	}()

	t.Run("not a TTY returns terminal error", func(t *testing.T) {
		isTTY = func() bool { return false }
		runWizardProgram = func(m tea.Model) (tea.Model, error) {
			t.Fatal("program must not run without a TTY")
			return nil, nil
		}
		_, _, err := launchWizard([]string{"github-gist"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "interactive wizard requires a terminal") {
			t.Errorf("error = %q, want substring 'interactive wizard requires a terminal'", err.Error())
		}
	})

	t.Run("program run error is wrapped", func(t *testing.T) {
		isTTY = func() bool { return true }
		runWizardProgram = func(m tea.Model) (tea.Model, error) {
			return nil, errors.New("boom")
		}
		_, _, err := launchWizard(nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "wizard:") {
			t.Errorf("error = %q, want substring 'wizard:'", err.Error())
		}
	})

	t.Run("unexpected model type returns error", func(t *testing.T) {
		isTTY = func() bool { return true }
		runWizardProgram = func(m tea.Model) (tea.Model, error) {
			return otherModel{}, nil
		}
		_, _, err := launchWizard(nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "unexpected model type") {
			t.Errorf("error = %q, want substring 'unexpected model type'", err.Error())
		}
	})

	t.Run("returns selections and wizard model", func(t *testing.T) {
		isTTY = func() bool { return true }
		wm := &screens.WizardModel{
			Confirmed:        true,
			SelectedProvider: "github-gist",
			SelectedPreset:   "full",
			NameInput:        "my-profile",
			AdapterItems: []screens.ToggleItem{
				{Name: "opencode", Checked: true},
				{Name: "cursor", Checked: false},
			},
			CategoryItems: []screens.ToggleItem{
				{Name: "skills", Checked: true},
			},
		}
		runWizardProgram = func(m tea.Model) (tea.Model, error) {
			return wm, nil
		}
		got, gotWM, err := launchWizard([]string{"github-gist"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Confirmed != true {
			t.Errorf("Confirmed = %v, want true", got.Confirmed)
		}
		if got.SelectedProvider != "github-gist" {
			t.Errorf("SelectedProvider = %q, want %q", got.SelectedProvider, "github-gist")
		}
		if got.SelectedPreset != "full" {
			t.Errorf("SelectedPreset = %q, want %q", got.SelectedPreset, "full")
		}
		if !equalStrings(got.AdapterNames, []string{"opencode"}) {
			t.Errorf("AdapterNames = %v, want [opencode]", got.AdapterNames)
		}
		if !equalStrings(got.CategoryNames, []string{"skills"}) {
			t.Errorf("CategoryNames = %v, want [skills]", got.CategoryNames)
		}
		if gotWM != wm {
			t.Error("returned wizard model is not the program result")
		}
		if gotWM.ProfileName() != "my-profile" {
			t.Errorf("ProfileName = %q, want %q", gotWM.ProfileName(), "my-profile")
		}
	})
}
