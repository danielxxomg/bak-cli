package cmd

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/danielxxomg/bak-cli/internal/tui"
	"github.com/danielxxomg/bak-cli/internal/tui/components"
	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// restorePickerModel is the bubbletea model for the interactive restore picker.
// It lists available backups and lets the user select one with ↑/↓ + Enter.
// q/Esc cancels. Mirrors the pickModel pattern in cmd/pick.go.
type restorePickerModel struct {
	backups   []tui.BackupInfo
	cursor    int
	quitting  bool
	confirmed bool
	width     int
	height    int
}

func (m restorePickerModel) Init() tea.Cmd {
	return nil
}

func (m restorePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.backups)-1 {
				m.cursor++
			}

		case "enter":
			if len(m.backups) > 0 {
				m.confirmed = true
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m restorePickerModel) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	// Guard against terminals below the minimum usable size.
	if m.width > 0 && m.height > 0 && (m.width < 20 || m.height < 10) {
		return tea.NewView(styles.HelpStyle.Render("Terminal too small (min 20x10)"))
	}

	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Select backup to restore"))
	b.WriteString("\n\n")

	if len(m.backups) == 0 {
		b.WriteString(styles.HelpStyle.Render("No backups found."))
		b.WriteString("\n\n")
	} else {
		for i, bk := range m.backups {
			label := fmt.Sprintf("%-18s %-22s %s", bk.ID, bk.Date, bk.Size)
			b.WriteString(components.RenderRadio(label, i == m.cursor, i == m.cursor))
			b.WriteByte('\n')
		}
	}

	b.WriteString("\n")
	b.WriteString(components.RenderHelp([]components.HelpKey{
		{Key: "\u2191/\u2193", Desc: "navigate"},
		{Key: "enter", Desc: "select"},
		{Key: "q/esc", Desc: "cancel"},
	}))

	return tea.NewView(b.String())
}

// SelectedID returns the selected backup ID, or empty string if none.
func (m restorePickerModel) SelectedID() string {
	if !m.confirmed || len(m.backups) == 0 || m.cursor >= len(m.backups) {
		return ""
	}
	return m.backups[m.cursor].ID
}
