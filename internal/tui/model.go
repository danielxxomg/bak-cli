package tui

import (
	"fmt"

	"charm.land/lipgloss/v2"

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
	// ScreenRestore shows the restore picker and flow.
	ScreenRestore
	// ScreenProfiles shows the profile management screen.
	ScreenProfiles
	// ScreenWelcome shows the first-run onboarding screen (PR2).
	ScreenWelcome
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
	restore   *screens.RestoreModel
	profiles  *screens.ProfilesModel
	cloud     *screens.CloudModel

	// Reusable components owned by the root model.
	search components.Search
	toast  components.Toast

	// Backup channels for async progress reporting.
	backupCh   chan ProgressUpdate
	backupDone chan error

	// showHelp toggles the help overlay on any screen via '?'.
	showHelp bool

	// selected tracks whether the user pressed Enter on the main menu.
	// true after handleMenuEnter, false on q/Esc/Quit. Prevents
	// RouteSelection from firing on cursor-0 when the TUI exits via q.
	selected bool
}

// NewModel creates a root Model initialized to the main menu screen with
// the default 7 menu items and the provided dependencies.
// When Deps.ConfigExists is non-nil and returns false, the model starts
// at the Welcome screen instead (first-run detection).
func NewModel(deps Deps) Model {
	screen := ScreenMenu
	if deps.ConfigExists != nil && !deps.ConfigExists() {
		screen = ScreenWelcome
	}
	return Model{
		screen:    screen,
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
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { //nolint:maintidx // SEVERE: tracked for qa-refactor-analysis (needs extraction, not config)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.tooSmall = styles.IsTooSmall(msg.Width, msg.Height)
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
		case ScreenRestore:
			if m.restore != nil {
				newR, _ := m.restore.Update(msg)
				nr := newR.(screens.RestoreModel)
				m.restore = &nr
			}
		case ScreenProfiles:
			if m.profiles != nil {
				newP, _ := m.profiles.Update(msg)
				np := newP.(screens.ProfilesModel)
				m.profiles = &np
			}
		case ScreenCloud:
			if m.cloud != nil {
				newC, _ := m.cloud.Update(msg)
				nc := newC.(screens.CloudModel)
				m.cloud = &nc
			}
		}
		return m, nil

	case tea.KeyPressMsg:
		// Global help overlay toggle: '?' shows help on any screen;
		// Esc or second '?' dismisses it.
		if m.showHelp {
			switch msg.Code {
			case KeyEsc, '?':
				m.showHelp = false
				return m, nil
			}
			// Block all other keys while help is visible.
			return m, nil
		}
		if msg.Code == '?' {
			m.showHelp = true
			return m, nil
		}
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
				s := m.initSettings()
				m.settings = &s
			}
			return m, m.settings.Init()
		case ScreenHealth:
			if m.health == nil {
				h := screens.NewHealthModel()
				m.health = &h
			}
			return m, m.health.Init()
		case ScreenRestore:
			if m.restore == nil {
				r := m.initRestore()
				m.restore = &r
			}
			return m, m.restore.Init()
		case ScreenProfiles:
			if m.profiles == nil {
				p := m.initProfiles()
				m.profiles = &p
			}
			return m, m.profiles.Init()
		case ScreenCloud:
			if m.cloud == nil {
				c := m.initCloud()
				m.cloud = &c
			}
			return m, m.cloud.Init()
		case ScreenWelcome:
			return m, nil
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

	case screens.ProgressStepMsg:
		// Forward to progress sub-model and re-issue channel drain.
		if m.progress != nil && m.screen == ScreenProgress {
			newProg, cmd := m.progress.Update(msg)
			np := newProg.(screens.ProgressModel)
			m.progress = &np
			return m, tea.Batch(cmd, drainProgressCmd(m.backupCh))
		}
		// Keep draining even if progress model isn't initialized.
		return m, drainProgressCmd(m.backupCh)

	case screens.ProgressDoneMsg:
		// Collect result from backupDone channel (non-blocking select).
		var resultErr error
		if m.backupDone != nil {
			select {
			case err := <-m.backupDone:
				resultErr = err
			default:
			}
		}
		// Forward to progress sub-model.
		if m.progress != nil && m.screen == ScreenProgress {
			newProg, cmd := m.progress.Update(msg)
			np := newProg.(screens.ProgressModel)
			m.progress = &np
			return m, tea.Batch(cmd, func() tea.Msg { return actionResultMsg{err: resultErr} })
		}
		return m, func() tea.Msg { return actionResultMsg{err: resultErr} }
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
	case ScreenRestore:
		if m.restore != nil {
			newR, cmd := m.restore.Update(msg)
			nr := newR.(screens.RestoreModel)
			m.restore = &nr
			return m, cmd
		}
	case ScreenProfiles:
		if m.profiles != nil {
			newP, cmd := m.profiles.Update(msg)
			np := newP.(screens.ProfilesModel)
			m.profiles = &np
			return m, cmd
		}
	case ScreenCloud:
		if m.cloud != nil {
			newC, cmd := m.cloud.Update(msg)
			nc := newC.(screens.CloudModel)
			m.cloud = &nc
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
	case ScreenWelcome:
		switch msg.Code {
		case KeyQuit, KeyEsc:
			return m, tea.Quit
		case KeyEnter:
			m.screen = ScreenMenu
			return m, nil
		}
	case ScreenCloud:
		switch msg.Code {
		case KeyQuit, KeyEsc:
			m.screen = ScreenMenu
			return m, nil
		default:
			if m.cloud != nil {
				newC, cmd := m.cloud.Update(msg)
				nc := newC.(screens.CloudModel)
				m.cloud = &nc
				return m, cmd
			}
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
	case ScreenRestore:
		if m.restore != nil {
			newR, cmd := m.restore.Update(msg)
			nr := newR.(screens.RestoreModel)
			m.restore = &nr
			return m, cmd
		}
	case ScreenProfiles:
		if m.profiles != nil {
			newP, cmd := m.profiles.Update(msg)
			np := newP.(screens.ProfilesModel)
			m.profiles = &np
			return m, cmd
		}
	}
	return m, nil
}

// handleMenuEnter routes the enter key on the main menu to the appropriate
// screen based on the current cursor position.
func (m Model) handleMenuEnter() (tea.Model, tea.Cmd) {
	m.selected = true
	switch m.cursor {
	case 0: // "Create backup" → Progress
		if m.deps.RunBackup != nil {
			m.backupCh = make(chan ProgressUpdate, 32)
			m.backupDone = make(chan error, 1)
			go func() {
				err := m.deps.RunBackup(nil, m.backupCh)
				m.backupDone <- err
			}()
		}
		return m, tea.Batch(
			func() tea.Msg { return screenChangeMsg{screen: ScreenProgress} },
			drainProgressCmd(m.backupCh),
		)
	case 1: // "Restore" → Restore screen
		return m, func() tea.Msg { return screenChangeMsg{screen: ScreenRestore} }
	case 2: // "Browse backups" → Dashboard
		return m, func() tea.Msg { return screenChangeMsg{screen: ScreenDashboard} }
	case 3: // "Cloud sync" → Cloud
		return m, func() tea.Msg { return screenChangeMsg{screen: ScreenCloud} }
	case 4: // "Profiles" → Profiles screen
		return m, func() tea.Msg { return screenChangeMsg{screen: ScreenProfiles} }
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
			"Terminal too small (%dx%d). Need at least %d\u00d7%d.",
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
			if m.cloud != nil {
				content = m.cloud.View().Content
			} else {
				content = screens.RenderCloudStatus(screens.CloudInfo{}, m.width)
			}
		case ScreenSettings:
			if m.settings != nil {
				content = m.settings.View().Content
			}
		case ScreenHealth:
			if m.health != nil {
				content = m.health.View().Content
			}
		case ScreenRestore:
			if m.restore != nil {
				content = m.restore.View().Content
			} else {
				content = "Restore"
			}
		case ScreenProfiles:
			if m.profiles != nil {
				content = m.profiles.View().Content
			} else {
				content = "Profiles"
			}
		case ScreenWelcome:
			content = screens.RenderWelcome(m.width)
		case ScreenShortcuts:
			content = screens.RenderShortcuts(m.width)
		default:
			content = ""
		}

		// Overlay help when toggled via '?'.
		if m.showHelp {
			content = screens.RenderShortcuts(m.width)
		}

		// Render toast overlay. On wide terminals (>= 50 cols), position
		// the toast at bottom-right using lipgloss.Place. On narrow terminals,
		// fall back to inline append below the screen content.
		if toastContent := m.toast.View(); toastContent != "" {
			if m.width >= 50 {
				content = lipgloss.Place(
					m.width, m.height,
					lipgloss.Right, lipgloss.Bottom,
					toastContent,
				)
			} else {
				content += "\n" + toastContent
			}
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

// drainProgressCmd returns a tea.Cmd that reads one ProgressUpdate from
// the given channel and converts it to a tea.Msg. When ch is nil, it
// returns nil (no-op). When the channel produces Done=true, it returns
// ProgressDoneMsg. Otherwise it returns ProgressStepMsg.
func drainProgressCmd(ch <-chan ProgressUpdate) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		update, ok := <-ch
		if !ok {
			return screens.ProgressDoneMsg{}
		}
		if update.Done {
			return screens.ProgressDoneMsg{}
		}
		return screens.ProgressStepMsg{
			Step:    update.Step,
			Current: update.Current,
			Total:   update.Total,
		}
	}
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
		Selected: m.selected,
		Cursor:   cursor,
		Item:     m.menuItems[cursor],
	}
}

// mapBackupInfo converts a slice of tui.BackupInfo records into the
// screens.BackupInfo type used by the dashboard and restore screens. The
// local screens type mirrors tui.BackupInfo to avoid a circular import.
// Returns nil for nil/empty input.
func mapBackupInfo(backups []BackupInfo) []screens.BackupInfo {
	if len(backups) == 0 {
		return nil
	}
	result := make([]screens.BackupInfo, 0, len(backups))
	for _, b := range backups {
		result = append(result, screens.BackupInfo{
			ID:     b.ID,
			Date:   b.Date,
			Size:   b.Size,
			Status: b.Status,
			Cloud:  b.Cloud,
		})
	}
	return result
}

// listBackupsForScreens fetches backups via the injected ListBackups dep and
// maps them to screens.BackupInfo. When ListBackups is nil (e.g., during
// testing) it returns (nil, nil). Errors from ListBackups are propagated.
func (m Model) listBackupsForScreens() ([]screens.BackupInfo, error) {
	if m.deps.ListBackups == nil {
		return nil, nil
	}
	backups, err := m.deps.ListBackups()
	if err != nil {
		return nil, err
	}
	return mapBackupInfo(backups), nil
}

// initDashboard creates a new DashboardModel using the injected deps.
func (m Model) initDashboard() screens.DashboardModel {
	return screens.NewDashboardModel(m.listBackupsForScreens)
}

// initProgress creates a new ProgressModel.
func (m Model) initProgress() screens.ProgressModel {
	return screens.NewProgressModel()
}

// initSettings creates a SettingsModel pre-populated with persisted settings
// from LoadSettings. If LoadSettings is nil or returns an error, defaults
// are used (NewSettingsModel behavior).
func (m Model) initSettings() screens.SettingsModel {
	if m.deps.LoadSettings == nil {
		return screens.NewSettingsModel(m.deps.SaveSetting)
	}
	s, err := m.deps.LoadSettings()
	if err != nil {
		return screens.NewSettingsModel(m.deps.SaveSetting)
	}
	return screens.NewSettingsModelWithSettings(s, m.deps.SaveSetting)
}

// initRestore creates a new RestoreModel using injected deps.
func (m Model) initRestore() screens.RestoreModel {
	listFn := m.listBackupsForScreens
	restoreFn := func(backupID string, dryRun bool) (string, error) {
		if m.deps.RunRestore == nil {
			return "", nil
		}
		return m.deps.RunRestore(backupID, dryRun)
	}
	return screens.NewRestoreModel(listFn, restoreFn)
}

// initProfiles creates a new ProfilesModel using injected deps.
func (m Model) initProfiles() screens.ProfilesModel {
	listFn := func() ([]screens.ProfileInfo, error) {
		if m.deps.ListProfiles == nil {
			return nil, nil
		}
		profiles, err := m.deps.ListProfiles()
		if err != nil {
			return nil, err
		}
		var result []screens.ProfileInfo
		for _, p := range profiles {
			result = append(result, screens.ProfileInfo{
				Name:     p.Name,
				Provider: p.Provider,
				Preset:   p.Preset,
				Active:   p.Active,
			})
		}
		return result, nil
	}
	switchFn := func(name string) error {
		if m.deps.SetActiveProfile == nil {
			return nil
		}
		return m.deps.SetActiveProfile(name)
	}
	deleteFn := func(name string) error {
		if m.deps.DeleteProfile == nil {
			return nil
		}
		return m.deps.DeleteProfile(name)
	}
	wizardFn := func() (screens.ProfileInfo, error) {
		if m.deps.RunWizard == nil {
			return screens.ProfileInfo{}, nil
		}
		p, err := m.deps.RunWizard()
		if err != nil {
			return screens.ProfileInfo{}, err
		}
		return screens.ProfileInfo{
			Name:     p.Name,
			Provider: p.Provider,
			Preset:   p.Preset,
			Active:   p.Active,
		}, nil
	}
	pm := screens.NewProfilesModel(listFn, switchFn, deleteFn, wizardFn)
	// Set SaveProfile as a mutable field.
	pm.SaveProfile = func(name string, profile screens.ProfileInfo) error {
		if m.deps.SaveProfile == nil {
			return nil
		}
		return m.deps.SaveProfile(name, profile)
	}
	return pm
}

// initCloud creates a new CloudModel using injected deps.
func (m Model) initCloud() screens.CloudModel {
	statusFn := func() (screens.CloudInfo, error) {
		if m.deps.GetCloudStatus == nil {
			return screens.CloudInfo{}, nil
		}
		s, err := m.deps.GetCloudStatus()
		if err != nil {
			return screens.CloudInfo{}, err
		}
		return screens.CloudInfo{
			Provider:   s.Provider,
			Connected:  s.Connected,
			LastSync:   s.LastSync,
			LocalCount: s.LocalCount,
			CloudCount: s.CloudCount,
		}, nil
	}
	return screens.NewCloudModel(statusFn)
}
