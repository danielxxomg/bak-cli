package cmd

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// wizardStep represents the current step in the interactive wizard.
type wizardStep int

const (
	stepProvider   wizardStep = iota // choose cloud provider
	stepPreset                       // choose backup preset
	stepAdapters                     // toggle adapters
	stepCategories                   // toggle categories
	stepConfirm                      // review and confirm
)

// wizardModel is the bubbletea model for the interactive profile create
// and login wizards. It reuses the cursor/toggle patterns from pickModel.
type wizardModel struct {
	step      wizardStep
	mode      string // "profile-create" or "login"
	quitting  bool
	confirmed bool

	// Provider selection state.
	providers      []string
	providerCursor int

	// Preset selection state.
	presets        []string
	presetCursor   int
	selectedPreset string

	// Adapter toggle state (reuses pickModel pattern).
	adapterItems  []toggleItem
	adapterCursor int

	// Category toggle state.
	categoryItems  []toggleItem
	categoryCursor int

	// Results.
	selectedProvider string
}

// toggleItem represents a selectable item in a toggle list.
type toggleItem struct {
	name    string
	checked bool
}

// newWizardModel creates a wizardModel for the given mode.
func newWizardModel(mode string, providers []string) *wizardModel {
	presets := []string{"quick", "full", "skills"}

	// Default adapters.
	defaultAdapters := []string{"opencode", "claude-code", "codex", "cursor", "windsurf"}
	adapterItems := make([]toggleItem, len(defaultAdapters))
	for i, a := range defaultAdapters {
		adapterItems[i] = toggleItem{name: a, checked: a == "opencode"}
	}

	// Default categories.
	defaultCategories := []string{"skills", "commands", "config", "plugins", "agents"}
	categoryItems := make([]toggleItem, len(defaultCategories))
	for i, c := range defaultCategories {
		categoryItems[i] = toggleItem{name: c, checked: c == "skills" || c == "config"}
	}

	return &wizardModel{
		step:          stepProvider,
		mode:          mode,
		providers:     providers,
		presets:       presets,
		adapterItems:  adapterItems,
		categoryItems: categoryItems,
	}
}

// CurrentStep returns the current wizard step (used by tests).
func (m *wizardModel) CurrentStep() wizardStep {
	return m.step
}

// Init implements bubbletea.Model.
func (m *wizardModel) Init() tea.Cmd {
	return nil
}

// Update implements bubbletea.Model. It handles keyboard input for each step
// and manages step transitions.
func (m *wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit

		case tea.KeyEnter:
			return m.handleEnter()

		default:
			return m.handleNavigation(msg)
		}
	}
	return m, nil
}

// handleEnter processes the Enter key based on the current step.
func (m *wizardModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.step {
	case stepProvider:
		if m.providerCursor < len(m.providers) {
			m.selectedProvider = m.providers[m.providerCursor]
		}
		m.step = stepPreset

	case stepPreset:
		if m.presetCursor < len(m.presets) {
			m.selectedPreset = m.presets[m.presetCursor]
		}
		m.step = stepAdapters

	case stepAdapters:
		m.step = stepCategories

	case stepCategories:
		m.step = stepConfirm

	case stepConfirm:
		m.confirmed = true
		return m, tea.Quit
	}
	return m, nil
}

// handleNavigation processes arrow keys, j/k, and space for toggle items.
func (m *wizardModel) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.step {
	case stepProvider:
		switch msg.String() {
		case "up", "k":
			if m.providerCursor > 0 {
				m.providerCursor--
			}
		case "down", "j":
			if m.providerCursor < len(m.providers)-1 {
				m.providerCursor++
			}
		}

	case stepPreset:
		switch msg.String() {
		case "up", "k":
			if m.presetCursor > 0 {
				m.presetCursor--
			}
		case "down", "j":
			if m.presetCursor < len(m.presets)-1 {
				m.presetCursor++
			}
		}

	case stepAdapters:
		switch msg.String() {
		case "up", "k":
			if m.adapterCursor > 0 {
				m.adapterCursor--
			}
		case "down", "j":
			if m.adapterCursor < len(m.adapterItems)-1 {
				m.adapterCursor++
			}
		case " ":
			if len(m.adapterItems) > 0 {
				m.adapterItems[m.adapterCursor].checked = !m.adapterItems[m.adapterCursor].checked
			}
		}

	case stepCategories:
		switch msg.String() {
		case "up", "k":
			if m.categoryCursor > 0 {
				m.categoryCursor--
			}
		case "down", "j":
			if m.categoryCursor < len(m.categoryItems)-1 {
				m.categoryCursor++
			}
		case " ":
			if len(m.categoryItems) > 0 {
				m.categoryItems[m.categoryCursor].checked = !m.categoryItems[m.categoryCursor].checked
			}
		}
	}
	return m, nil
}

// View implements bubbletea.Model.
func (m *wizardModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	stepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	switch m.step {
	case stepProvider:
		b.WriteString(titleStyle.Render("Create Profile — Provider"))
		b.WriteString("\n\n")
		b.WriteString(stepStyle.Render("Step 1/5: Choose a cloud provider"))
		b.WriteString("\n\n")
		b.WriteString(m.renderProviderList())

	case stepPreset:
		b.WriteString(titleStyle.Render("Create Profile — Preset"))
		b.WriteString("\n\n")
		b.WriteString(stepStyle.Render("Step 2/5: Choose a backup preset"))
		b.WriteString("\n\n")
		b.WriteString(m.renderPresetList())

	case stepAdapters:
		b.WriteString(titleStyle.Render("Create Profile — Adapters"))
		b.WriteString("\n\n")
		b.WriteString(stepStyle.Render("Step 3/5: Select adapters to back up (space to toggle)"))
		b.WriteString("\n\n")
		b.WriteString(m.renderToggleList(m.adapterItems, m.adapterCursor))

	case stepCategories:
		b.WriteString(titleStyle.Render("Create Profile — Categories"))
		b.WriteString("\n\n")
		b.WriteString(stepStyle.Render("Step 4/5: Select categories to back up (space to toggle)"))
		b.WriteString("\n\n")
		b.WriteString(m.renderToggleList(m.categoryItems, m.categoryCursor))

	case stepConfirm:
		b.WriteString(titleStyle.Render("Create Profile — Confirm"))
		b.WriteString("\n\n")
		b.WriteString(m.renderConfirmSummary())
	}

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter: next • q/esc: quit"))

	return b.String()
}

func (m *wizardModel) renderProviderList() string {
	return renderCursorList(m.providers, m.providerCursor)
}

func (m *wizardModel) renderPresetList() string {
	return renderCursorList(m.presets, m.presetCursor)
}

func (m *wizardModel) renderToggleList(items []toggleItem, cursor int) string {
	var b strings.Builder
	checkedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	uncheckedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))

	for i, item := range items {
		cursorMark := "  "
		if cursor == i {
			cursorMark = cursorStyle.Render("> ")
		}
		check := uncheckedStyle.Render("[ ]")
		if item.checked {
			check = checkedStyle.Render("[x]")
		}
		fmt.Fprintf(&b, "%s%s %s\n", cursorMark, check, item.name)
	}
	return b.String()
}

func (m *wizardModel) renderConfirmSummary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Provider:   %s\n", m.selectedProvider)
	fmt.Fprintf(&b, "Preset:     %s\n", m.selectedPreset)

	var adapters []string
	for _, item := range m.adapterItems {
		if item.checked {
			adapters = append(adapters, item.name)
		}
	}
	fmt.Fprintf(&b, "Adapters:   %s\n", strings.Join(adapters, ", "))

	var categories []string
	for _, item := range m.categoryItems {
		if item.checked {
			categories = append(categories, item.name)
		}
	}
	fmt.Fprintf(&b, "Categories: %s\n", strings.Join(categories, ", "))

	b.WriteString("\nPress Enter to create the profile.")
	return b.String()
}

// renderCursorList renders a simple cursor-based list.
func renderCursorList(items []string, cursor int) string {
	var b strings.Builder
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))

	for i, item := range items {
		cursorMark := "  "
		if cursor == i {
			cursorMark = cursorStyle.Render("> ")
		}
		fmt.Fprintf(&b, "%s%s\n", cursorMark, item)
	}
	return b.String()
}

// isTTY reports whether stdin is a terminal.
func isTTY() bool {
	return isatty.IsTerminal(os.Stdin.Fd())
}
