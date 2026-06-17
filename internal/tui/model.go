package tui

import (
	"fmt"

	"github.com/danielxxomg/bak-cli/internal/tui/components"
	"github.com/danielxxomg/bak-cli/internal/tui/screens"
	"github.com/danielxxomg/bak-cli/internal/tui/styles"

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
	// ScreenSettings allows toggling cloud providers and themes.
	ScreenSettings
	// ScreenCloud shows the cloud sync status screen.
	ScreenCloud
	// ScreenShortcuts displays the keyboard shortcut reference overlay.
	ScreenShortcuts
	// ScreenHealth runs the backup health diagnostic check.
	ScreenHealth
)

// screenChangeMsg is an internal message that triggers a screen transition.
// It is returned as a tea.Cmd from handleKey when the user presses enter
// on a menu item that navigates to a sub-screen.
type screenChangeMsg struct {
	screen Screen
}

// actionResultMsg is a tea.Msg that signals an action (backup/restore)
// has completed. The err field is nil on success, non-nil on failure.
// The root Model's Update handler displays the result via Toast.Show().
type actionResultMsg struct {
	err error
}

// Minimum terminal dimensions (styles.MinWidth / styles.MinHeight) are
// defined in internal/tui/styles/styles.go so sub-screens can import them
// without circular dependencies.
//
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
	settings  *screens.SettingsModel
	health    *screens.HealthModel

	// Reusable components owned by the root model.
	search components.Search
	toast  components.Toast
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
		m.tooSmall = msg.Width < styles.MinWidth || msg.Height < styles.MinHeight
		// Forward to active sub-model.
		switch m.screen {
		case ScreenDashboard:
			if m.dashboard != nil {
				newDash, _ := m.dashboard.Update(msg)
				nd := newDash.(screens.DashboardModel)
				m.dashboard = &nd
			}
		case ScreenProgress:
			if m.progress != nil {
				newProg, _ := m.progress.Update(msg)
				np := newProg.(screens.ProgressModel)
				m.progress = &np
			}
		case ScreenSettings:
			if m.settings != nil {
				newSet, _ := m.settings.Update(msg)
				ns := newSet.(screens.SettingsModel)
				m.settings = &ns
			}
		case ScreenHealth:
			if m.health != nil {
				newH, _ := m.health.Update(msg)
				nh := newH.(screens.HealthModel)
				m.health = &nh
			}
		}
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
				p.Width = m.width
				p.Height = m.height
				m.progress = &p
			}
			return m, m.progress.Init()
		case ScreenSettings:
			if m.settings == nil {
				s := screens.NewSettingsModel()
				m.settings = &s
			}
			return m, m.settings.Init()
		case ScreenHealth:
			if m.health == nil {
				h := screens.NewHealthModel()
				m.health = &h
			}
			return m, m.health.Init()
		}
		return m, nil

	case screens.ScreenBackMsg:
		m.screen = ScreenMenu
		return m, nil

	case actionResultMsg:
		if msg.err == nil {
			m.toast.Show("Backup complete", 3)
		} else {
			m.toast.Show(msg.err.Error(), 3)
		}
		// Forward to toast so it starts the tick countdown immediately.
		newToast, cmd := m.toast.Update(msg)
		m.toast = newToast
		return m, cmd
	}

	// Forward remaining messages to the active sub-model.
	switch m.screen {
	case ScreenDashboard:
		if m.dashboard != nil {
			newDash, cmd := m.dashboard.Update(msg)
			nd := newDash.(screens.DashboardModel)
			m.dashboard = &nd
			return m, cmd
		}
	case ScreenProgress:
		if m.progress != nil {
			newProg, cmd := m.progress.Update(msg)
			np := newProg.(screens.ProgressModel)
			m.progress = &np
			return m, cmd
		}
	case ScreenSettings:
		if m.settings != nil {
			newSet, cmd := m.settings.Update(msg)
			ns := newSet.(screens.SettingsModel)
			m.settings = &ns
			return m, cmd
		}
	case ScreenHealth:
		if m.health != nil {
			newH, cmd := m.health.Update(msg)
			nh := newH.(screens.HealthModel)
			m.health = &nh
			return m, cmd
		}
	}

	// Always forward tick messages to the toast component.
	newToast, cmd := m.toast.Update(msg)
	m.toast = newToast
	if cmd != nil {
		return m, cmd
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
		case KeyDown, tea.KeyDown:
			m.cursor = (m.cursor + 1) % len(m.menuItems)
		case KeyUp, tea.KeyUp:
			m.cursor = (m.cursor - 1 + len(m.menuItems)) % len(m.menuItems)
		case KeyEnter:
			return m.handleMenuEnter()
		case '?':
			m.screen = ScreenShortcuts
			return m, nil
		}
	case ScreenCloud:
		switch msg.Code {
		case KeyQuit, KeyEsc:
			m.screen = ScreenMenu
			return m, nil
		}
	case ScreenDashboard:
		// When search is active, forward keystrokes to the search component
		// first, then filter the dashboard table with the current query.
		if m.search.IsActive() {
			switch msg.Code {
			case KeyEsc:
				// Esc deactivates search and restores all rows.
				m.search.Deactivate()
				if m.dashboard != nil {
					m.dashboard.SetFilter("")
				}
				return m, nil
			default:
				newSearch, cmd := m.search.Update(msg)
				m.search = newSearch
				if m.dashboard != nil {
					m.dashboard.SetFilter(m.search.Query())
				}
				return m, cmd
			}
		}

		// When search is inactive, handle normal dashboard navigation.
		switch msg.Code {
		case KeyQuit, KeyEsc:
			m.screen = ScreenMenu
			return m, nil
		case '/':
			m.search.Activate()
			return m, nil
		default:
			// Forward to dashboard sub-model.
			if m.dashboard != nil {
				newDash, cmd := m.dashboard.Update(msg)
				nd := newDash.(screens.DashboardModel)
				m.dashboard = &nd
				return m, cmd
			}
		}
	case ScreenShortcuts:
		switch msg.Code {
		case KeyQuit, KeyEsc, '?':
			m.screen = ScreenMenu
			return m, nil
		}
	case ScreenSettings:
		if m.settings != nil {
			newSet, cmd := m.settings.Update(msg)
			ns := newSet.(screens.SettingsModel)
			m.settings = &ns
			return m, cmd
		}
	case ScreenHealth:
		if m.health != nil {
			newH, cmd := m.health.Update(msg)
			nh := newH.(screens.HealthModel)
			m.health = &nh
			return m, cmd
		}
	case ScreenProgress:
		if m.progress != nil {
			newProg, cmd := m.progress.Update(msg)
			np := newProg.(screens.ProgressModel)
			m.progress = &np
			return m, cmd
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
	case 1: // "Restore" → not yet implemented
		m.toast.Show("Restore: coming soon", 3)
		return m, nil
	case 2: // "Browse backups" → Dashboard
		return m, func() tea.Msg { return screenChangeMsg{screen: ScreenDashboard} }
	case 3: // "Cloud sync" → Cloud
		return m, func() tea.Msg { return screenChangeMsg{screen: ScreenCloud} }
	case 4: // "Profiles" → not yet implemented
		m.toast.Show("Profiles: coming soon", 3)
		return m, nil
	case 5: // "Settings"
		return m, func() tea.Msg { return screenChangeMsg{screen: ScreenSettings} }
	case 6: // "Quit"
		return m, tea.Quit
	}
	return m, nil
}

// View renders the current screen. If the terminal is too small
// (below styles.MinWidth × styles.MinHeight), a warning message
// showing actual and required dimensions is displayed instead.
// The alternate screen buffer is enabled so the TUI takes the full
// terminal and restores the previous content on exit.
func (m Model) View() tea.View {
	var content string
	if m.tooSmall {
		content = fmt.Sprintf(
			"Terminal too small (%dx%d). Need at least %dx%d.",
			m.width, m.height, styles.MinWidth, styles.MinHeight,
		)
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
		case ScreenSettings:
			if m.settings != nil {
				content = m.settings.View().Content
			}
		case ScreenHealth:
			if m.health != nil {
				content = m.health.View().Content
			}
		case ScreenShortcuts:
			content = screens.RenderShortcuts(m.width)
		default:
			content = ""
		}

		// Render toast overlay on top of screen content when visible.
		if toastContent := m.toast.View(); toastContent != "" {
			content += "\n" + toastContent
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
