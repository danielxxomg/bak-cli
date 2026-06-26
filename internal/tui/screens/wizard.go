package screens

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/danielxxomg/bak-cli/internal/tui/components"
	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// WizardStep represents the current step in the interactive wizard.
type WizardStep int

// wizardWindowTitle is the terminal window title shown on every wizard view
// branch (tui-personality REQ-TP-001).
const wizardWindowTitle = "bak — Wizard"

const (
	StepName       WizardStep = iota // enter profile name
	StepProvider                     // choose cloud provider
	StepPreset                       // choose backup preset
	StepAdapters                     // toggle adapters
	StepCategories                   // toggle categories
	StepConfirm                      // review and confirm
)

// WizardModel is the bubbletea model for the interactive profile create
// and login wizards. It reuses the cursor/toggle patterns from pickModel.
type WizardModel struct {
	Step     WizardStep
	Mode     string // "profile-create" or "login"
	Quitting bool
	// Confirmed is set to true when the user confirms on the final step.
	Confirmed bool
	Width     int
	Height    int

	// Name input state (StepName).
	NameInput string

	// Provider selection state.
	Providers      []string
	ProviderCursor int

	// Preset selection state.
	Presets        []string
	PresetCursor   int
	SelectedPreset string

	// Adapter toggle state (reuses pickModel pattern).
	AdapterItems  []ToggleItem
	AdapterCursor int

	// Category toggle state.
	CategoryItems  []ToggleItem
	CategoryCursor int

	// Results.
	SelectedProvider string
}

// ToggleItem represents a selectable item in a toggle list.
type ToggleItem struct {
	Name    string
	Checked bool
}

// NewWizardModel creates a WizardModel for the given mode.
func NewWizardModel(mode string, providers []string) *WizardModel {
	presets := []string{"quick", "full", categorySkills}

	// Start step depends on mode: profile-create in TUI path starts with
	// name input; CLI-driven wizards (login, profile create) start at provider.
	startStep := StepProvider
	if mode == "profile-create" {
		startStep = StepName
	}

	// Default adapters.
	defaultAdapters := []string{"opencode", "claude-code", "codex", "cursor", "windsurf"}
	adapterItems := make([]ToggleItem, len(defaultAdapters))
	for i, a := range defaultAdapters {
		adapterItems[i] = ToggleItem{Name: a, Checked: a == "opencode"}
	}

	// Default categories.
	defaultCategories := []string{categorySkills, "commands", "config", "plugins", "agents"}
	categoryItems := make([]ToggleItem, len(defaultCategories))
	for i, c := range defaultCategories {
		categoryItems[i] = ToggleItem{Name: c, Checked: c == categorySkills || c == "config"}
	}

	return &WizardModel{
		Step:          startStep,
		Mode:          mode,
		Providers:     providers,
		Presets:       presets,
		AdapterItems:  adapterItems,
		CategoryItems: categoryItems,
	}
}

// CurrentStep returns the current wizard step (used by tests).
func (m *WizardModel) CurrentStep() WizardStep {
	return m.Step
}

// ProfileName returns the effective profile name.
// Uses the user-entered name if non-empty; otherwise falls back to the selected provider.
// If both are empty, returns "untitled".
func (m *WizardModel) ProfileName() string {
	if m.NameInput != "" {
		return m.NameInput
	}
	if m.SelectedProvider != "" {
		return m.SelectedProvider
	}
	return "untitled"
}

// Init implements bubbletea.Model.
func (m *WizardModel) Init() tea.Cmd {
	return nil
}

// Update implements bubbletea.Model. It handles keyboard input for each step
// and manages step transitions.
func (m *WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.Quitting = true
			return m, tea.Quit

		case keyEnter:
			return m.handleEnter()

		default:
			return m.handleNavigation(msg)
		}
	}
	return m, nil
}

// handleEnter processes the Enter key based on the current step.
func (m *WizardModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.Step {
	case StepName:
		m.Step = StepProvider

	case StepProvider:
		if m.ProviderCursor < len(m.Providers) {
			m.SelectedProvider = m.Providers[m.ProviderCursor]
		}
		m.Step = StepPreset

	case StepPreset:
		if m.PresetCursor < len(m.Presets) {
			m.SelectedPreset = m.Presets[m.PresetCursor]
		}
		m.Step = StepAdapters

	case StepAdapters:
		m.Step = StepCategories

	case StepCategories:
		m.Step = StepConfirm

	case StepConfirm:
		m.Confirmed = true
		return m, tea.Quit
	}
	return m, nil
}

// MoveCursor adjusts cursor by ±1 based on up/down navigation keys.
// It clamps to [0, max]. When max < 0, the cursor is left unchanged
// (guard for empty lists).
func MoveCursor(cursor *int, max int, key string) {
	switch key {
	case "up", "k":
		if *cursor > 0 {
			*cursor--
		}
	case "down", "j":
		if *cursor < max {
			*cursor++
		}
	}
}

// handleNavigation processes arrow keys, j/k, and space for toggle items.
func (m *WizardModel) handleNavigation(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.Step {
	case StepName:
		switch msg.String() {
		case "backspace":
			if len(m.NameInput) > 0 {
				m.NameInput = m.NameInput[:len(m.NameInput)-1]
			}
		default:
			// Accept printable characters (single rune).
			if len(msg.String()) == 1 {
				m.NameInput += msg.String()
			}
		}
		return m, nil

	case StepProvider:
		MoveCursor(&m.ProviderCursor, len(m.Providers)-1, msg.String())

	case StepPreset:
		MoveCursor(&m.PresetCursor, len(m.Presets)-1, msg.String())

	case StepAdapters:
		MoveCursor(&m.AdapterCursor, len(m.AdapterItems)-1, msg.String())
		if msg.String() == keySpace && len(m.AdapterItems) > 0 {
			m.AdapterItems[m.AdapterCursor].Checked = !m.AdapterItems[m.AdapterCursor].Checked
		}

	case StepCategories:
		MoveCursor(&m.CategoryCursor, len(m.CategoryItems)-1, msg.String())
		if msg.String() == keySpace && len(m.CategoryItems) > 0 {
			m.CategoryItems[m.CategoryCursor].Checked = !m.CategoryItems[m.CategoryCursor].Checked
		}
	default:
		// StepConfirm displays a summary; no navigation keys
	}
	return m, nil
}

// View implements bubbletea.Model.
func (m *WizardModel) View() tea.View {
	if m.Quitting {
		v := tea.NewView("")
		v.WindowTitle = wizardWindowTitle
		return v
	}

	// Guard against terminals below the minimum usable size.
	if m.Width > 0 && m.Height > 0 && styles.IsTooSmall(m.Width, m.Height) {
		v := tea.NewView(styles.HelpStyle.Render(styles.RenderTooSmall(m.Width, m.Height)))
		v.WindowTitle = wizardWindowTitle
		return v
	}

	var b strings.Builder

	switch m.Step {
	case StepName:
		b.WriteString(styles.TitleStyle.Render("Create Profile \u2014 Name"))
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Step 1/6: Enter profile name (e.g., opencode-default)"))
		b.WriteString("\n\n")
		displayName := m.NameInput
		if displayName == "" {
			displayName = "_"
		}
		b.WriteString(displayName)

	case StepProvider:
		b.WriteString(styles.TitleStyle.Render("Create Profile \u2014 Provider"))
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Step 2/6: Choose a cloud provider"))
		b.WriteString("\n\n")
		b.WriteString(components.RenderMenu(m.Providers, m.ProviderCursor))

	case StepPreset:
		b.WriteString(styles.TitleStyle.Render("Create Profile \u2014 Preset"))
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Step 3/6: Choose a backup preset"))
		b.WriteString("\n\n")
		b.WriteString(components.RenderMenu(m.Presets, m.PresetCursor))

	case StepAdapters:
		b.WriteString(styles.TitleStyle.Render("Create Profile \u2014 Adapters"))
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Step 4/6: Select adapters to back up (space to toggle)"))
		b.WriteString("\n\n")
		b.WriteString(m.renderCheckboxList(m.AdapterItems, m.AdapterCursor))

	case StepCategories:
		b.WriteString(styles.TitleStyle.Render("Create Profile \u2014 Categories"))
		b.WriteString("\n\n")
		b.WriteString(styles.HelpStyle.Render("Step 5/6: Select categories to back up (space to toggle)"))
		b.WriteString("\n\n")
		b.WriteString(m.renderCheckboxList(m.CategoryItems, m.CategoryCursor))

	case StepConfirm:
		b.WriteString(styles.TitleStyle.Render("Create Profile \u2014 Confirm"))
		b.WriteString("\n\n")
		b.WriteString(m.renderConfirmSummary())
	}

	b.WriteString("\n")
	b.WriteString(components.RenderHelp([]components.HelpKey{
		{Key: keyEnter, Desc: "next"},
		{Key: "q/esc", Desc: keyQuit},
	}))

	v := tea.NewView(b.String())
	v.WindowTitle = wizardWindowTitle
	return v
}

// renderCheckboxList renders a list of toggleable items using the shared
// checkbox component. Each item is rendered with its checked state and
// the focused item is highlighted via SelectedStyle.
func (m *WizardModel) renderCheckboxList(items []ToggleItem, cursor int) string {
	var b strings.Builder
	for i, item := range items {
		b.WriteString(components.RenderCheckbox(item.Name, item.Checked, i == cursor))
		if i < len(items)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m *WizardModel) renderConfirmSummary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Provider:   %s\n", m.SelectedProvider)
	fmt.Fprintf(&b, "Preset:     %s\n", m.SelectedPreset)

	var adapters []string
	for _, item := range m.AdapterItems {
		if item.Checked {
			adapters = append(adapters, item.Name)
		}
	}
	fmt.Fprintf(&b, "Adapters:   %s\n", strings.Join(adapters, ", "))

	var categories []string
	for _, item := range m.CategoryItems {
		if item.Checked {
			categories = append(categories, item.Name)
		}
	}
	fmt.Fprintf(&b, "Categories: %s\n", strings.Join(categories, ", "))

	b.WriteString("\nPress Enter to create the profile.")
	return b.String()
}
