package screens

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/danielxxomg/bak-cli/internal/tui/components"
	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// SettingsOption represents a single configurable option in the settings screen.
// Type "toggle" uses a boolean Value; Type "select" uses a string Value.
type SettingsOption struct {
	Label string
	Type  string // "toggle" or "select"
	Value bool   // true = toggled on (for "toggle" type)
}

// SettingsModel is the Bubble Tea sub-model for the interactive settings screen.
// It supports j/k navigation and enter/space to toggle options.
type SettingsModel struct {
	cursor  int
	options []SettingsOption
	width   int
	height  int
}

// NewSettingsModel creates a SettingsModel pre-populated with default options:
// Cloud Provider, Theme (locked to Rose Pine), Auto-sync, and Verbose.
func NewSettingsModel() SettingsModel {
	return SettingsModel{
		options: []SettingsOption{
			{Label: "Cloud Provider", Type: "toggle", Value: false},
			{Label: "Theme", Type: "select", Value: true},
			{Label: "Auto-sync", Type: "toggle", Value: false},
			{Label: "Verbose", Type: "toggle", Value: false},
		},
	}
}

// Init returns nil — no initial side effects.
func (m SettingsModel) Init() tea.Cmd {
	return nil
}

// Update handles keyboard navigation (j/k), toggling (enter/space), and
// back navigation (q/esc). WindowSizeMsg updates stored dimensions.
func (m SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		switch msg.Code {
		case 'j':
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case 'k':
			if m.cursor > 0 {
				m.cursor--
			}
		case '\r', ' ':
			// Toggle the focused option if it is a toggle type.
			if m.cursor >= 0 && m.cursor < len(m.options) {
				opt := &m.options[m.cursor]
				if opt.Type == "toggle" {
					opt.Value = !opt.Value
				}
			}
		case 'q', 27:
			return m, func() tea.Msg { return ScreenBackMsg{} }
		}
	}

	return m, nil
}

// View renders the settings screen as a list of checkbox-style options.
// The focused item is styled with SelectedStyle; checked items show ✓.
func (m SettingsModel) View() tea.View {
	if m.width < 20 || m.height < 10 {
		return tea.NewView("Terminal too small")
	}

	var b strings.Builder

	b.WriteString(styles.SettingsTitleStyle.Render("Settings"))
	b.WriteString("\n\n")

	for i, opt := range m.options {
		focused := i == m.cursor
		b.WriteString(components.RenderCheckbox(opt.Label, opt.Value, focused))
		if i < len(m.options)-1 {
			b.WriteString("\n")
		}
	}

	return tea.NewView(b.String())
}
