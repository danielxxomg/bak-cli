// Package components provides reusable TUI components for the bak-cli TUI.
package components

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// ModalResultMsg is emitted by ModalModel when the user confirms or cancels.
// The Confirmed field is true when the first button (index 0) is activated,
// false when any other button or Escape is pressed.
type ModalResultMsg struct {
	Confirmed bool
}

// Package-level modal frame style — defined once, Width() applied per render.
var modalFrameStyle = lipgloss.NewStyle().
	Border(lipgloss.DoubleBorder()).
	BorderForeground(styles.ColorOverlay).
	Padding(1)

// ModalModel is a reusable modal dialog sub-model for confirmations, alerts,
// and prompts. It renders a centered bordered box with title, message, and
// action buttons. The parent screen owns a *ModalModel, renders it on top
// when non-nil, and handles ModalResultMsg in its Update.
type ModalModel struct {
	Title   string
	Message string
	Buttons []string
	Width   int
	Height  int
	cursor  int
}

// NewModal creates a ModalModel with the given title, message, and buttons.
// The cursor starts at 0 (first button focused). An empty or nil Buttons
// slice is valid; the modal will still render but keyboard navigation is
// disabled.
func NewModal(title, message string, buttons []string) ModalModel {
	return ModalModel{
		Title:   title,
		Message: message,
		Buttons: buttons,
		cursor:  0,
	}
}

// Init returns nil — no initial side effects.
func (m ModalModel) Init() tea.Cmd {
	return nil
}

// Update handles keyboard navigation, action emission, and window resizing.
// Enter emits ModalResultMsg with Confirmed=true when cursor is on the first
// button (index 0), false otherwise. Escape emits Confirmed=false. Tab/Shift+Tab
// cycles buttons. WindowSizeMsg updates stored dimensions.
func (m ModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)
	}
	return m, nil
}

func (m ModalModel) handleKeyPress(keyMsg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch keyMsg.Code {
	case tea.KeyEnter:
		confirmed := m.cursor == 0
		return m, func() tea.Msg { return ModalResultMsg{Confirmed: confirmed} }

	case tea.KeyEscape:
		return m, func() tea.Msg { return ModalResultMsg{Confirmed: false} }

	case tea.KeyTab:
		if len(m.Buttons) == 0 {
			return m, nil
		}
		if keyMsg.Mod&tea.ModShift != 0 {
			m.cursor = (m.cursor - 1 + len(m.Buttons)) % len(m.Buttons)
		} else {
			m.cursor = (m.cursor + 1) % len(m.Buttons)
		}
		return m, nil
	}
	return m, nil
}

// View renders the modal as a centered bordered overlay. It adapts to
// terminal width and shows a fallback message if the terminal is below
// 20x10 columns/rows.
func (m ModalModel) View() tea.View {
	// Too-small guard: below 20 columns or 10 rows.
	if m.Width < 20 || m.Height < 10 {
		return tea.NewView("Terminal too small for modal")
	}

	if m.Width <= 0 {
		m.Width = 80
	}
	if m.Height <= 0 {
		m.Height = 24
	}

	// Determine modal width: fit within terminal with padding.
	modalWidth := m.Width - 4
	if modalWidth < 30 {
		modalWidth = m.Width - 2
	}
	if modalWidth > 60 {
		modalWidth = 60
	}

	var b strings.Builder

	// Title.
	b.WriteString(styles.TitleStyle.Render(m.Title))
	b.WriteString("\n\n")

	// Message.
	b.WriteString(m.Message)
	b.WriteString("\n\n")

	// Buttons.
	for i, btn := range m.Buttons {
		if i > 0 {
			b.WriteString("  ")
		}
		if i == m.cursor {
			b.WriteString(styles.SelectedStyle.Render(fmt.Sprintf("[ %s ]", btn)))
		} else {
			b.WriteString(styles.HelpStyle.Render(fmt.Sprintf("  %s  ", btn)))
		}
	}

	content := b.String()

	// Wrap in a bordered frame (package-level style, Width() applied per render).
	boxed := modalFrameStyle.Width(modalWidth).Render(content)

	// Center the modal on screen.
	centered := lipgloss.Place(
		m.Width, m.Height,
		lipgloss.Center, lipgloss.Center,
		boxed,
	)

	return tea.NewView(centered)
}
