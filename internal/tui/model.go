package tui

import (
	"fmt"

	"charm.land/lipgloss/v2"

	"github.com/danielxxomg/bak-cli/internal/tui/components"
	"github.com/danielxxomg/bak-cli/internal/tui/screens"
	"github.com/danielxxomg/bak-cli/internal/tui/styles"

	tea "charm.land/bubbletea/v2"
)

// screen represents the active TUI screen. The root Model routes Update
// and View calls based on the current screen value. The type is unexported;
// the Screen* constants below remain exported so callers can use the values
// without naming the type.
type screen int

const (
	// ScreenMenu is the main navigation menu (default screen).
	ScreenMenu screen = iota
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
	screen screen
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
	screen   screen
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

	// subs is the lazy sub-model dispatch cache. It holds the same pointers
	// as the typed fields above and is populated by screenChangeMsg (and kept
	// in sync by forwardTo). The authoritative source for dispatch is the
	// subEntries closures, which read the typed fields live so direct field
	// assignment (e.g. in tests) stays consistent with map-based routing.
	subs map[screen]subModel

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

// subModel is the contract a screen sub-model satisfies for message
// dispatch and rendering. All root sub-models (dashboard, progress,
// settings, health, restore, profiles, cloud) implement Update with a
// value receiver, so their pointer types (*screens.XModel) satisfy this
// interface via Go's method-set promotion.
type subModel interface {
	Update(tea.Msg) (tea.Model, tea.Cmd)
	View() tea.View
}

// subEntry binds a screen to get/set closures that read and write its typed
// sub-model field on Model. The closures are stateless and always reflect
// the current typed field, so direct field assignment (used in tests and by
// screenChangeMsg) stays consistent with map-based dispatch without a
// separate synchronization step.
type subEntry struct {
	get func(m *Model) subModel     // current typed field as subModel, or nil
	set func(m *Model, u tea.Model) // persist an updated sub-model value
}

// subEntries is the dispatch map: one entry per screen that owns a
// sub-model. Each set closure performs the single type assertion +
// address-of + field assignment that was previously duplicated ~21 times
// across Update, handleKey, and the WindowSizeMsg forwarding.
var subEntries = map[screen]subEntry{
	ScreenDashboard: {
		get: func(m *Model) subModel {
			if m.dashboard == nil {
				return nil
			}
			return m.dashboard
		},
		set: func(m *Model, u tea.Model) {
			d := u.(screens.DashboardModel)
			m.dashboard = &d
			m.subs[ScreenDashboard] = m.dashboard
		},
	},
	ScreenProgress: {
		get: func(m *Model) subModel {
			if m.progress == nil {
				return nil
			}
			return m.progress
		},
		set: func(m *Model, u tea.Model) {
			p := u.(screens.ProgressModel)
			m.progress = &p
			m.subs[ScreenProgress] = m.progress
		},
	},
	ScreenSettings: {
		get: func(m *Model) subModel {
			if m.settings == nil {
				return nil
			}
			return m.settings
		},
		set: func(m *Model, u tea.Model) {
			s := u.(screens.SettingsModel)
			m.settings = &s
			m.subs[ScreenSettings] = m.settings
		},
	},
	ScreenHealth: {
		get: func(m *Model) subModel {
			if m.health == nil {
				return nil
			}
			return m.health
		},
		set: func(m *Model, u tea.Model) {
			h := u.(screens.HealthModel)
			m.health = &h
			m.subs[ScreenHealth] = m.health
		},
	},
	ScreenRestore: {
		get: func(m *Model) subModel {
			if m.restore == nil {
				return nil
			}
			return m.restore
		},
		set: func(m *Model, u tea.Model) {
			r := u.(screens.RestoreModel)
			m.restore = &r
			m.subs[ScreenRestore] = m.restore
		},
	},
	ScreenProfiles: {
		get: func(m *Model) subModel {
			if m.profiles == nil {
				return nil
			}
			return m.profiles
		},
		set: func(m *Model, u tea.Model) {
			p := u.(screens.ProfilesModel)
			m.profiles = &p
			m.subs[ScreenProfiles] = m.profiles
		},
	},
	ScreenCloud: {
		get: func(m *Model) subModel {
			if m.cloud == nil {
				return nil
			}
			return m.cloud
		},
		set: func(m *Model, u tea.Model) {
			c := u.(screens.CloudModel)
			m.cloud = &c
			m.subs[ScreenCloud] = m.cloud
		},
	},
}

// ensureSubs lazily initializes the sub-model dispatch cache. Idempotent.
func (m *Model) ensureSubs() {
	if m.subs == nil {
		m.subs = make(map[screen]subModel)
	}
}

// forwardTo dispatches msg to the sub-model registered for screen s. It
// reads the current sub-model from the typed field (via the get closure, so
// direct field assignment stays consistent), calls Update, and persists the
// result back to the typed field and the subs cache (via the set closure).
// It returns (cmd, true) when a sub-model handled the message and
// (nil, false) when no sub-model is registered or present, so callers can
// fall through to other handling (e.g. toast forwarding) without panicking.
func (m *Model) forwardTo(s screen, msg tea.Msg) (tea.Cmd, bool) {
	m.ensureSubs()
	entry, ok := subEntries[s]
	if !ok {
		return nil, false
	}
	sub := entry.get(m)
	if sub == nil {
		return nil, false
	}
	m.subs[s] = sub
	updated, cmd := sub.Update(msg)
	entry.set(m, updated)
	return cmd, true
}

// Update handles incoming messages and routes them based on the current
// screen and message type. It implements the tea.Model interface.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.tooSmall = styles.IsTooSmall(msg.Width, msg.Height)
		m.forwardTo(m.screen, msg)
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKeyPressMsg(msg)

	case screenChangeMsg:
		return m.handleScreenChange(msg)

	case screens.ScreenBackMsg:
		m.screen = ScreenMenu
		return m, nil

	case actionResultMsg:
		return m.handleActionResultMsg(msg)

	case screens.ProgressStepMsg:
		return m.handleProgressStepMsg(msg)

	case screens.ProgressDoneMsg:
		return m.handleProgressDoneMsg(msg)
	}

	// Forward remaining messages to the active sub-model.
	if cmd, ok := m.forwardTo(m.screen, msg); ok {
		return m, cmd
	}

	// Always forward tick messages to the toast component.
	newToast, cmd := m.toast.Update(msg)
	m.toast = newToast
	if cmd != nil {
		return m, cmd
	}
	return m, nil
}

// handleKeyPressMsg applies the global help-overlay toggle before delegating
// to screen-specific key handling. Extracted from Update to keep the
// message-type router within the cyclomatic complexity budget.
func (m Model) handleKeyPressMsg(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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
}

// handleActionResultMsg shows a toast for the backup result and forwards the
// message to the toast component so its tick countdown starts immediately.
func (m Model) handleActionResultMsg(msg actionResultMsg) (tea.Model, tea.Cmd) {
	if msg.err == nil {
		m.toast.Show("Backup complete", 3)
	} else {
		m.toast.Show(msg.err.Error(), 3)
	}
	newToast, cmd := m.toast.Update(msg)
	m.toast = newToast
	return m, cmd
}

// handleProgressStepMsg forwards a progress step to the progress sub-model
// (when on the progress screen) and re-issues the channel drain command.
func (m Model) handleProgressStepMsg(msg screens.ProgressStepMsg) (tea.Model, tea.Cmd) {
	if m.screen == ScreenProgress {
		if cmd, ok := m.forwardTo(ScreenProgress, msg); ok {
			return m, tea.Batch(cmd, drainProgressCmd(m.backupCh))
		}
	}
	// Keep draining even if progress model isn't initialized.
	return m, drainProgressCmd(m.backupCh)
}

// handleProgressDoneMsg collects the backup result, forwards the done
// message to the progress sub-model when visible, and emits an
// actionResultMsg so the toast reflects the outcome.
func (m Model) handleProgressDoneMsg(msg screens.ProgressDoneMsg) (tea.Model, tea.Cmd) {
	resultErr := m.collectBackupResult()
	if m.screen == ScreenProgress {
		if cmd, ok := m.forwardTo(ScreenProgress, msg); ok {
			return m, tea.Batch(cmd, func() tea.Msg { return actionResultMsg{err: resultErr} })
		}
	}
	return m, func() tea.Msg { return actionResultMsg{err: resultErr} }
}

// handleScreenChange processes a screenChangeMsg: it records the new screen,
// lazily initializes its sub-model on first entry (populating the subs
// dispatch cache), and returns the sub-model's Init command.
//
//nolint:gocyclo // screen dispatch: each case is a uniform lazy-init + register step; per-screen construction varies (Progress sets dimensions, Health uses NewHealthModel directly), so a map-driven init closure would duplicate the get/set boilerplate (dupl) without reducing real complexity.
func (m *Model) handleScreenChange(msg screenChangeMsg) (tea.Model, tea.Cmd) {
	m.screen = msg.screen
	m.ensureSubs()
	switch msg.screen {
	case ScreenDashboard:
		if m.dashboard == nil {
			d := m.initDashboard()
			m.dashboard = &d
		}
		m.subs[ScreenDashboard] = m.dashboard
		return *m, m.dashboard.Init()
	case ScreenProgress:
		if m.progress == nil {
			p := m.initProgress()
			p.Width = m.width
			p.Height = m.height
			m.progress = &p
		}
		m.subs[ScreenProgress] = m.progress
		return *m, m.progress.Init()
	case ScreenSettings:
		if m.settings == nil {
			s := m.initSettings()
			m.settings = &s
		}
		m.subs[ScreenSettings] = m.settings
		return *m, m.settings.Init()
	case ScreenHealth:
		if m.health == nil {
			h := screens.NewHealthModel()
			m.health = &h
		}
		m.subs[ScreenHealth] = m.health
		return *m, m.health.Init()
	case ScreenRestore:
		if m.restore == nil {
			r := m.initRestore()
			m.restore = &r
		}
		m.subs[ScreenRestore] = m.restore
		return *m, m.restore.Init()
	case ScreenProfiles:
		if m.profiles == nil {
			p := m.initProfiles()
			m.profiles = &p
		}
		m.subs[ScreenProfiles] = m.profiles
		return *m, m.profiles.Init()
	case ScreenCloud:
		if m.cloud == nil {
			c := m.initCloud()
			m.cloud = &c
		}
		m.subs[ScreenCloud] = m.cloud
		return *m, m.cloud.Init()
	case ScreenWelcome, ScreenMenu, ScreenShortcuts:
		return *m, nil
	}
	return *m, nil
}

// collectBackupResult drains the backupDone channel non-blockingly,
// returning the backup result error (or nil when no result is available).
func (m *Model) collectBackupResult() error {
	if m.backupDone == nil {
		return nil
	}
	select {
	case err := <-m.backupDone:
		return err
	default:
		return nil
	}
}

// handleKey processes key presses based on the active screen.
func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case ScreenMenu:
		return m.handleMenuKey(msg)
	case ScreenWelcome:
		return m.handleWelcomeKey(msg)
	case ScreenCloud:
		if msg.Code == KeyQuit || msg.Code == KeyEsc {
			m.screen = ScreenMenu
			return m, nil
		}
		if cmd, ok := m.forwardTo(ScreenCloud, msg); ok {
			return m, cmd
		}
	case ScreenDashboard:
		if newM, cmd, handled := m.handleDashboardKey(msg); handled {
			return newM, cmd
		}
	case ScreenShortcuts:
		if msg.Code == KeyQuit || msg.Code == KeyEsc || msg.Code == '?' {
			m.screen = ScreenMenu
			return m, nil
		}
	case ScreenSettings, ScreenHealth, ScreenProgress, ScreenRestore, ScreenProfiles:
		if cmd, ok := m.forwardTo(m.screen, msg); ok {
			return m, cmd
		}
	}
	return m, nil
}

// handleMenuKey routes keystrokes on the main menu: quit, cursor movement,
// enter (screen selection), and the shortcuts overlay. Extracted from
// handleKey to keep the screen router within the complexity budget.
func (m Model) handleMenuKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case KeyQuit, KeyEsc:
		return m, tea.Quit
	case KeyDown, tea.KeyDown:
		m.cursor = (m.cursor + 1) % len(m.menuItems)
		return m, nil
	case KeyUp, tea.KeyUp:
		m.cursor = (m.cursor - 1 + len(m.menuItems)) % len(m.menuItems)
		return m, nil
	case KeyEnter:
		return m.handleMenuEnter()
	case '?':
		m.screen = ScreenShortcuts
		return m, nil
	}
	return m, nil
}

// handleWelcomeKey routes keystrokes on the first-run welcome screen:
// quit or press enter to proceed to the main menu.
func (m Model) handleWelcomeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case KeyQuit, KeyEsc:
		return m, tea.Quit
	case KeyEnter:
		m.screen = ScreenMenu
		return m, nil
	}
	return m, nil
}

// handleDashboardKey routes keystrokes on the dashboard screen. When the
// search field is active, keystrokes feed the search query and keep the table
// filter in sync; otherwise normal dashboard navigation and forwarding apply.
// It returns (model, cmd, true) when the key was consumed and (model, nil,
// false) when handleKey should fall through to its default no-op return.
// Extracted from handleKey to keep handleKey within the funlen statement budget.
func (m Model) handleDashboardKey(msg tea.KeyPressMsg) (Model, tea.Cmd, bool) {
	// When search is active, forward keystrokes to the search component first,
	// then filter the dashboard table with the current query.
	if m.search.IsActive() {
		if msg.Code == KeyEsc {
			// Esc deactivates search and restores all rows.
			m.search.Deactivate()
			if m.dashboard != nil {
				m.dashboard.SetFilter("")
			}
			return m, nil, true
		}
		newSearch, cmd := m.search.Update(msg)
		m.search = newSearch
		if m.dashboard != nil {
			m.dashboard.SetFilter(m.search.Query())
		}
		return m, cmd, true
	}
	// When search is inactive, handle normal dashboard navigation.
	switch msg.Code {
	case KeyQuit, KeyEsc:
		m.screen = ScreenMenu
		return m, nil, true
	case '/':
		m.search.Activate()
		return m, nil, true
	default:
		if cmd, ok := m.forwardTo(ScreenDashboard, msg); ok {
			return m, cmd, true
		}
	}
	return m, nil, false
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
		content = styles.RenderTooSmall(m.width, m.height)
	} else {
		content = m.renderContent()
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.WindowTitle = titleForScreen(m)
	return v
}

// titleForScreen maps the active screen to a contextual terminal window title
// of the form "bak — {Screen}" (REQ-TP-001). On ScreenProgress with a running
// operation it appends the live step counter ("bak — Backup 3/7"); on
// ScreenRestore it appends the selected backup id when present. The title is a
// pure read of current model state, recomputed every render — no command
// plumbing, matching the existing v.AltScreen field-assignment pattern.
func titleForScreen(m Model) string {
	switch m.screen {
	case ScreenMenu:
		return "bak — Main Menu"
	case ScreenWelcome:
		return "bak — Welcome"
	case ScreenDashboard:
		return "bak — Backups"
	case ScreenSettings:
		return "bak — Settings"
	case ScreenCloud:
		return "bak — Cloud"
	case ScreenShortcuts:
		return "bak — Shortcuts"
	case ScreenHealth:
		return "bak — Health"
	case ScreenProfiles:
		return "bak — Profiles"
	case ScreenProgress:
		return progressTitle(m)
	case ScreenRestore:
		return restoreTitle(m)
	default:
		return "bak"
	}
}

// progressTitle renders the progress screen title, appending the live step
// counter ("bak — Backup 3/7") when an operation is running with a known total.
func progressTitle(m Model) string {
	if m.progress != nil && m.progress.Running() && m.progress.Total > 0 {
		return fmt.Sprintf("bak — Backup %d/%d", m.progress.Current, m.progress.Total)
	}
	return "bak — Backup"
}

// restoreTitle renders the restore screen title, appending the selected backup
// id ("bak — Restore:abc1234") when one has been chosen.
func restoreTitle(m Model) string {
	if m.restore != nil && m.restore.SelectedID != "" {
		return "bak — Restore:" + m.restore.SelectedID
	}
	return "bak — Restore"
}

// renderContent renders the active screen with optional help and toast
// overlays. It is the non-tooSmall branch of View, extracted to keep View's
// nesting shallow.
func (m Model) renderContent() string {
	content := m.renderScreen()
	// Overlay help when toggled via '?'.
	if m.showHelp {
		content = screens.RenderShortcuts(m.width)
	}
	// Persistent status bar at the bottom of every screen (REQ-TP-003).
	// Hidden on narrow terminals (<40 cols) by RenderStatusBar itself.
	if bar := components.RenderStatusBar(m.width, m.deps.Version, m.deps.Preset, m.deps.BackupPath); bar != "" {
		content += "\n" + bar
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
	return content
}

// renderScreen returns the content for the active screen, delegating to the
// sub-model when one is initialized and falling back to placeholder text or
// the stateless screen renderers otherwise.
func (m Model) renderScreen() string {
	if m.screen == ScreenMenu {
		return m.renderMenu()
	}
	if content, ok := m.subView(m.screen); ok {
		return content
	}
	// Fall back to stateless renderers / placeholders for screens without
	// an initialized sub-model.
	switch m.screen {
	case ScreenCloud:
		return screens.RenderCloudStatus(screens.CloudInfo{}, m.width)
	case ScreenRestore:
		return "Restore"
	case ScreenProfiles:
		return "Profiles"
	case ScreenWelcome:
		return screens.RenderWelcome(m.width)
	case ScreenShortcuts:
		return screens.RenderShortcuts(m.width)
	default:
		// Screens with an initialized sub-model are handled by subView
		// above; remaining screens render an empty placeholder.
		return ""
	}
}

// subView returns the rendered content of the sub-model registered for
// screen s, or ("", false) when no sub-model is registered or initialized.
// It reuses the subEntries dispatch map so render routing stays in sync with
// message dispatch.
func (m Model) subView(s screen) (string, bool) {
	entry, ok := subEntries[s]
	if !ok {
		return "", false
	}
	sub := entry.get(&m)
	if sub == nil {
		return "", false
	}
	return sub.View().Content, true
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
