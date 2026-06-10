package tui

import (
	"github.com/danielxxomg/bak-cli/internal/tui/screens"

	tea "charm.land/bubbletea/v2"
)

// Screen represents the active TUI screen. The root Model routes Update
// and View calls based on the current screen value.
type Screen int

const (
	// ScreenMenu is the main navigation menu (default screen).
	ScreenMenu Screen = iota
	// ScreenDashboard displays backup records in a table.
	ScreenDashboard
	// ScreenProgress shows a live backup/restore progress bar.
	ScreenProgress
	// ScreenWizard guides first-run configuration.
	ScreenWizard
	// ScreenSettings allows toggling cloud providers and themes.
	ScreenSettings
	// ScreenCloud shows the cloud sync status screen.
	ScreenCloud
)

// screenChangeMsg is an internal message that triggers a screen transition.
// It is returned as a tea.Cmd from handleKey when the user presses enter
// on a menu item that navigates to a sub-screen.
type screenChangeMsg struct {
	screen Screen
}

// Minimum terminal dimensions required to render the TUI layout.
// If the terminal is smaller than these values, a "Terminal too small"
// warning is displayed instead of the normal screen.
const (
	minWidth  = 20
	minHeight = 10
)

// Model is the root Bubble Tea model for the bak-cli TUI. It owns window
// dimensions, cursor state, and routes Update/View to the active screen.
//
// All fields are unexported; tests in the tui package access them directly
// for white-box assertions.
type Model struct {
	screen   Screen
	width    int
	height   int
	cursor   int
	tooSmall bool
	deps     Deps

	// menuItems are the labels for the main menu (7 items, PR2).
	menuItems []string

	// Sub-models for complex screens (lazily initialized on first visit).
	dashboard *screens.DashboardModel
	progress  *screens.ProgressModel
}

// NewModel creates a root Model initialized to the main menu screen with
// the default 7 menu items and the provided dependencies.
func NewModel(deps Deps) Model {
	return Model{
		screen:    ScreenMenu,
		cursor:    0,
		deps:      deps,
		menuItems: DefaultMenuItems,
	}
}

// Init is the Bubble Tea initialisation command. The root model has no
// initial side effects, so it returns a nil command.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages and routes them based on the current
// screen and message type. It implements the tea.Model interface.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.tooSmall = msg.Width < minWidth || msg.Height < minHeight
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case screenChangeMsg:
		m.screen = msg.screen
		// Lazy-init sub-models on first screen entry.
		switch msg.screen {
		case ScreenDashboard:
			if m.dashboard == nil {
				d := m.initDashboard()
				m.dashboard = &d
			}
			return m, m.dashboard.Init()
		case ScreenProgress:
			if m.progress == nil {
				p := m.initProgress()
				m.progress = &p
			}
			return m, m.progress.Init()
		}
		return m, nil

	case screens.ScreenBackMsg:
		m.screen = ScreenMenu
		return m, nil
	}

	return m, nil
}

// handleKey processes key presses based on the active screen.
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case ScreenMenu:
		switch msg.Code {
		case KeyQuit, KeyEsc:
			return m, tea.Quit
		case KeyDown:
			if m.cursor < len(m.menuItems)-1 {
				m.cursor++
			}
		case KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case KeyEnter:
			return m.handleMenuEnter()
		}
	case ScreenCloud:
		switch msg.Code {
		case KeyQuit, KeyEsc:
			m.screen = ScreenMenu
			return m, nil
		}
	}
	return m, nil
}

// handleMenuEnter routes the enter key on the main menu to the appropriate
// screen based on the current cursor position.
func (m Model) handleMenuEnter() (tea.Model, tea.Cmd) {
	switch m.cursor {
	case 0: // "Create backup" → Progress
		return m, func() tea.Msg { return screenChangeMsg{screen: ScreenProgress} }
	case 2: // "Browse backups" → Dashboard
		return m, func() tea.Msg { return screenChangeMsg{screen: ScreenDashboard} }
	case 3: // "Cloud sync" → Cloud
		return m, func() tea.Msg { return screenChangeMsg{screen: ScreenCloud} }
	case 6: // "Quit"
		return m, tea.Quit
	}
	return m, nil
}

// View renders the current screen. If the terminal is too small
// (width < 20 or height < 10), a warning message is shown instead.
// The alternate screen buffer is enabled so the TUI takes the full
// terminal and restores the previous content on exit.
func (m Model) View() tea.View {
	var content string
	if m.tooSmall {
		content = "Terminal too small"
	} else {
		switch m.screen {
		case ScreenMenu:
			content = m.renderMenu()
		case ScreenDashboard:
			if m.dashboard != nil {
				content = m.dashboard.View().Content
			}
		case ScreenProgress:
			if m.progress != nil {
				content = m.progress.View().Content
			}
		case ScreenCloud:
			content = screens.RenderCloudStatus(screens.CloudInfo{}, m.width)
		default:
			content = ""
		}
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// renderMenu composes the main menu view using the full screen renderer
// which includes the logo, version, menu items, and help bar.
func (m Model) renderMenu() string {
	return screens.RenderMainMenu(m.deps.Version, "", m.menuItems, m.cursor, m.width)
}

// Selection returns the current menu selection. If the cursor is out of
// bounds or menuItems is empty, a zero-value MenuSelection is returned.
// This is the primary mechanism for cmd/root.go to determine which action
// to run after the TUI exits.
func (m Model) Selection() MenuSelection {
	if len(m.menuItems) == 0 {
		return MenuSelection{}
	}
	cursor := m.cursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(m.menuItems) {
		cursor = len(m.menuItems) - 1
	}
	return MenuSelection{
		Cursor: cursor,
		Item:   m.menuItems[cursor],
	}
}

// initDashboard creates a new DashboardModel using the injected deps.
func (m Model) initDashboard() screens.DashboardModel {
	return screens.NewDashboardModel(func() ([]screens.BackupInfo, error) {
		if m.deps.ListBackups == nil {
			return nil, nil
		}
		backups, err := m.deps.ListBackups()
		if err != nil {
			return nil, err
		}
		var result []screens.BackupInfo
		for _, b := range backups {
			result = append(result, screens.BackupInfo{
				ID:     b.ID,
				Date:   b.Date,
				Size:   b.Size,
				Status: b.Status,
				Cloud:  b.Cloud,
			})
		}
		return result, nil
	})
}

// initProgress creates a new ProgressModel.
func (m Model) initProgress() screens.ProgressModel {
	return screens.NewProgressModel()
}
