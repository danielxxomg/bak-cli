package screens

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/danielxxomg/bak-cli/internal/tui/components"
	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

const keyEsc = "esc"

// RestorePickerModel is the bubbletea model for the interactive restore picker.
// It lists available backups and lets the user select one with ↑/↓ + Enter.
// q/Esc cancels.
type RestorePickerModel struct {
	// Backups holds the list of available backups to display.
	Backups []BackupInfo

	// Cursor is the index of the currently highlighted backup.
	Cursor int

	// Quitting indicates whether the picker was dismissed without selection.
	Quitting bool

	// Confirmed indicates whether the user confirmed a selection with Enter.
	Confirmed bool

	// Width is the current terminal width in columns.
	Width int

	// Height is the current terminal height in rows.
	Height int
}

// Init initializes the RestorePickerModel. Returns nil command.
func (m RestorePickerModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the picker: resize events and navigation keystrokes.
func (m RestorePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", keyEsc, "ctrl+c":
			m.Quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}

		case "down", "j":
			if m.Cursor < len(m.Backups)-1 {
				m.Cursor++
			}

		case keyEnter:
			if len(m.Backups) > 0 {
				m.Confirmed = true
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

// View renders the restore picker interface.
func (m RestorePickerModel) View() tea.View {
	if m.Quitting {
		return tea.NewView("")
	}

	// Guard against terminals below the minimum usable size.
	if m.Width > 0 && m.Height > 0 && (m.Width < 20 || m.Height < 10) {
		return tea.NewView(styles.HelpStyle.Render("Terminal too small (min 20x10)"))
	}

	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Select backup to restore"))
	b.WriteString("\n\n")

	if len(m.Backups) == 0 {
		b.WriteString(styles.HelpStyle.Render("No backups found."))
		b.WriteString("\n\n")
	} else {
		for i, bk := range m.Backups {
			label := fmt.Sprintf("%-18s %-22s %s", bk.ID, bk.Date, bk.Size)
			b.WriteString(components.RenderRadio(label, i == m.Cursor, i == m.Cursor))
			b.WriteByte('\n')
		}
	}

	b.WriteString("\n")
	b.WriteString(components.RenderHelp([]components.HelpKey{
		{Key: keyArrowUpDown, Desc: keyNavigate},
		{Key: keyEnter, Desc: keySelect},
		{Key: "q/" + keyEsc, Desc: "cancel"},
	}))

	return tea.NewView(b.String())
}

// SelectedID returns the selected backup ID, or empty string if none.
func (m RestorePickerModel) SelectedID() string {
	if !m.Confirmed || len(m.Backups) == 0 || m.Cursor >= len(m.Backups) || m.Cursor < 0 {
		return ""
	}
	return m.Backups[m.Cursor].ID
}
