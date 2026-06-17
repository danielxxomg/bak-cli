package screens

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"github.com/danielxxomg/bak-cli/internal/tui/components"
	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// BackupInfo represents a single backup record displayed in the dashboard.
// This mirrors tui.BackupInfo to avoid circular import (screens cannot
// import the parent tui package).
type BackupInfo struct {
	ID     string
	Date   string
	Size   string
	Status string
	Cloud  string
}

// ScreenBackMsg is sent by sub-models when the user requests to navigate
// back to the previous screen (e.g., pressing q or esc).
type ScreenBackMsg struct{}

// dashboard table columns.
var dashboardColumns = []table.Column{
	{Title: "ID", Width: 8},
	{Title: "Date", Width: 12},
	{Title: "Size", Width: 10},
	{Title: "Status", Width: 8},
	{Title: "Cloud", Width: 10},
}

// DashboardModel is the Bubble Tea sub-model for the dashboard screen.
// It wraps a bubbles/table sub-model and handles keyboard navigation.
//
// allRows stores the original unfiltered rows so that SetFilter can
// rebuild the table from scratch on each filter change.
type DashboardModel struct {
	table   table.Model
	allRows []table.Row
	err     error
	width   int
	height  int
}

// NewDashboardModel creates a DashboardModel by calling listBackups to
// populate the table. If listBackups returns an error, the model stores it
// and displays an error message instead of the table.
//
// The listBackups function matches the signature of tui.Deps.ListBackups
// but uses a local BackupInfo type to avoid circular imports.
func NewDashboardModel(listBackups func() ([]BackupInfo, error)) DashboardModel {
	backups, err := listBackups()

	var rows []table.Row
	for _, b := range backups {
		rows = append(rows, table.Row{b.ID, b.Date, b.Size, b.Status, b.Cloud})
	}

	tbl := table.New(
		table.WithColumns(dashboardColumns),
		table.WithRows(rows),
		table.WithStyles(styles.DashboardTableStyle),
		table.WithFocused(true),
		table.WithWidth(76),
		table.WithHeight(20),
	)

	return DashboardModel{
		table:   tbl,
		allRows: rows,
		err:     err,
		width:   80,
		height:  24,
	}
}

// Init returns nil — the dashboard has no initial side effects.
func (m DashboardModel) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages for the dashboard screen. Key presses
// for navigation (j/k) are forwarded to the table sub-model. q and esc
// return a ScreenBackMsg. WindowSizeMsg updates stored dimensions.
func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetWidth(msg.Width - 4)
		m.table.SetHeight(msg.Height - 4)
		return m, nil

	case tea.KeyPressMsg:
		switch msg.Code {
		case 'q', 27: // esc
			return m, func() tea.Msg { return ScreenBackMsg{} }

		case 'j', 'k':
			newTable, cmd := m.table.Update(msg)
			m.table = newTable
			return m, cmd
		}
	}

	// Forward all other messages to the table sub-model.
	newTable, cmd := m.table.Update(msg)
	m.table = newTable
	return m, cmd
}

// SetFilter rebuilds the table rows from allRows, keeping only rows that
// contain query as a case-insensitive substring in any column. An empty
// query restores all original rows. The table cursor is reset to 0.
func (m *DashboardModel) SetFilter(query string) {
	q := strings.ToLower(query)
	if q == "" {
		m.table.SetRows(m.allRows)
		m.table.SetCursor(0)
		return
	}

	var filtered []table.Row
	for _, row := range m.allRows {
		for _, col := range row {
			if strings.Contains(strings.ToLower(col), q) {
				filtered = append(filtered, row)
				break
			}
		}
	}
	m.table.SetRows(filtered)
	m.table.SetCursor(0)
}

// View renders the dashboard screen. If an error occurred during data
// loading, an error message is shown. If no backups exist, an empty state
// message is shown. Otherwise, the styled table is rendered.
//
// If the terminal is below 20×10, a "Terminal too small" message is shown.
func (m DashboardModel) View() tea.View {
	if m.width < styles.MinWidth || m.height < styles.MinHeight {
		return tea.NewView("Terminal too small")
	}

	var b strings.Builder

	// Heading.
	b.WriteString(styles.DashboardTitleStyle.Render("Backups"))
	b.WriteString("\n\n")

	// Error state.
	if m.err != nil {
		b.WriteString(styles.DashboardErrorStyle.Render(
			fmt.Sprintf("Error: %v", m.err),
		))
		b.WriteString("\n\n")
		b.WriteString(renderDashboardHelp())
		return tea.NewView(b.String())
	}

	// Empty state.
	if len(m.table.Rows()) == 0 {
		b.WriteString(styles.DashboardEmptyStyle.Render("No backups found"))
		b.WriteString("\n\n")
		b.WriteString(renderDashboardHelp())
		return tea.NewView(b.String())
	}

	// Populated table.
	b.WriteString(m.table.View())
	b.WriteString("\n\n")
	b.WriteString(renderDashboardHelp())
	return tea.NewView(b.String())
}

// renderDashboardHelp returns the dashboard help bar with context-appropriate keys.
func renderDashboardHelp() string {
	helpKeys := []components.HelpKey{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "/", Desc: "search"},
		{Key: "q", Desc: "back"},
	}
	return components.RenderHelp(helpKeys)
}
