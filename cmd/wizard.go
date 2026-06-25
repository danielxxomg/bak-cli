package cmd

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/mattn/go-isatty"

	"github.com/danielxxomg/bak-cli/internal/actions"
	"github.com/danielxxomg/bak-cli/internal/tui/screens"
)

// isTTY reports whether stdin is a terminal.
// Exposed as a package-level variable so tests can override it
// (follows the var execCommand pattern from AGENTS.md).
var isTTY = func() bool {
	return isatty.IsTerminal(os.Stdin.Fd())
}

// runWizardProgram is the injection point for running the interactive
// wizard Bubble Tea program. Tests override this to return a pre-built
// model without launching a real TTY (mirrors the runTUI pattern in tty.go).
var runWizardProgram func(model tea.Model) (tea.Model, error) = defaultRunWizardProgram

// defaultRunWizardProgram runs the wizard program against the real terminal.
func defaultRunWizardProgram(model tea.Model) (tea.Model, error) {
	p := tea.NewProgram(model)
	return p.Run()
}

// wizardSelections collects the checked adapters and categories from a
// completed wizard model into a ProfileCreateFromWizard value. It is a
// pure function over the model's selection state.
func wizardSelections(wm *screens.WizardModel) actions.ProfileCreateFromWizard {
	var adapterNames []string
	for _, item := range wm.AdapterItems {
		if item.Checked {
			adapterNames = append(adapterNames, item.Name)
		}
	}
	var categoryNames []string
	for _, item := range wm.CategoryItems {
		if item.Checked {
			categoryNames = append(categoryNames, item.Name)
		}
	}
	return actions.ProfileCreateFromWizard{
		Confirmed:        wm.Confirmed,
		SelectedProvider: wm.SelectedProvider,
		SelectedPreset:   wm.SelectedPreset,
		AdapterNames:     adapterNames,
		CategoryNames:    categoryNames,
	}
}

// launchWizard runs the interactive profile-creation wizard and returns the
// collected selections plus the final wizard model. The caller reads
// wm.ProfileName() when no name was supplied on the command line.
//
// It enforces the TTY gate, runs the (injectable) wizard program, asserts the
// result is a *screens.WizardModel, and gathers the checked selections.
// Returns an error when there is no TTY, the program run fails, or the
// resulting model has an unexpected type.
func launchWizard(providers []string) (actions.ProfileCreateFromWizard, *screens.WizardModel, error) {
	if !isTTY() {
		return actions.ProfileCreateFromWizard{}, nil, fmt.Errorf("interactive wizard requires a terminal (TTY)")
	}
	m := screens.NewWizardModel("profile-create", providers)
	finalModel, runErr := runWizardProgram(m)
	if runErr != nil {
		return actions.ProfileCreateFromWizard{}, nil, fmt.Errorf("wizard: %w", runErr)
	}
	wm, ok := finalModel.(*screens.WizardModel)
	if !ok {
		return actions.ProfileCreateFromWizard{}, nil, fmt.Errorf("wizard: unexpected model type %T", finalModel)
	}
	return wizardSelections(wm), wm, nil
}
