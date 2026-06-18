package screens

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/danielxxomg/bak-cli/internal/tui/components"
	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// SettingsOption represents a single configurable option in the settings screen.
// The Value field is a boolean: true means toggled on (checked).
type SettingsOption struct {
	Label string
	Type  string // "toggle" or "select"
	Value bool   // true = toggled on
	Key   string // config key used for persistence (e.g. "auto_sync")
}

// Settings holds the user preferences that the settings screen can display
// and modify. This mirrors config.Settings without importing the config package.
type Settings struct {
	DefaultPreset      string
	AutoSync           bool
	ExcludePatterns    []string
	MaxFileSize        int64
	ConfirmDestructive bool
	VerboseDefault     bool
	DefaultProvider    string
}

// SettingsModel is the Bubble Tea sub-model for the interactive settings screen.
// It supports j/k navigation and enter/space to toggle options. When a toggle
// changes, the SaveSetting function is called immediately to persist the change.
type SettingsModel struct {
	cursor   int
	options  []SettingsOption
	width    int
	height   int
	saveFunc func(key string, value any) error
}

// NewSettingsModel creates a SettingsModel with default options and the given
// save function. The save function is called immediately when a toggle changes.
func NewSettingsModel(saveFunc func(key string, value any) error) SettingsModel {
	m := SettingsModel{
		options: []SettingsOption{
			{Label: "Cloud Provider", Type: "toggle", Value: false, Key: "default_provider"},
			{Label: "Theme", Type: "select", Value: true, Key: ""},
			{Label: "Auto-sync", Type: "toggle", Value: false, Key: "auto_sync"},
			{Label: "Verbose", Type: "toggle", Value: false, Key: "verbose_default"},
		},
		saveFunc: saveFunc,
	}
	return m
}

// NewSettingsModelWithSettings creates a SettingsModel pre-populated with the
// given settings values and a save function. Toggle options are set from the
// provided settings struct.
func NewSettingsModelWithSettings(s Settings, saveFunc func(key string, value any) error) SettingsModel {
	m := NewSettingsModel(saveFunc)
	// Apply settings to toggle options.
	for i := range m.options {
		switch m.options[i].Key {
		case "auto_sync":
			m.options[i].Value = s.AutoSync
		case "verbose_default":
			m.options[i].Value = s.VerboseDefault
		case "default_provider":
			m.options[i].Value = s.DefaultProvider != ""
		case "confirm_destructive":
			m.options[i].Value = s.ConfirmDestructive
		}
	}
	return m
}

// Init returns nil — no initial side effects.
func (m SettingsModel) Init() tea.Cmd {
	return nil
}

// Update handles keyboard navigation (j/k), toggling (enter/space), and
// back navigation (q/esc). WindowSizeMsg updates stored dimensions.
// When a toggle changes, the injected SaveSetting function is called.
func (m SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		switch msg.Code {
		case 'j', tea.KeyDown:
			m.cursor = (m.cursor + 1) % len(m.options)
		case 'k', tea.KeyUp:
			m.cursor = (m.cursor - 1 + len(m.options)) % len(m.options)
		case '\r', ' ':
			// Toggle the focused option if it is a toggle type.
			if m.cursor >= 0 && m.cursor < len(m.options) {
				opt := &m.options[m.cursor]
				if opt.Type == "toggle" {
					opt.Value = !opt.Value
					// Persist immediately.
					if m.saveFunc != nil && opt.Key != "" {
						_ = m.saveFunc(opt.Key, opt.Value)
					}
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
	if m.width < styles.MinWidth || m.height < styles.MinHeight {
		return tea.NewView("Terminal too small")
	}

	var b strings.Builder

	b.WriteString(styles.ScreenTitleStyle.Render("Settings"))
	b.WriteString("\n\n")

	for i, opt := range m.options {
		focused := i == m.cursor
		b.WriteString(components.RenderCheckbox(opt.Label, opt.Value, focused))
		if i < len(m.options)-1 {
			b.WriteString("\n")
		}
	}

	// Help bar with context-appropriate keybindings.
	b.WriteString("\n\n")
	helpKeys := []components.HelpKey{
		{Key: "↑/↓", Desc: "navigate"},
		{Key: "enter", Desc: "toggle"},
		{Key: "q", Desc: "back"},
	}
	b.WriteString(components.RenderHelp(helpKeys))

	return tea.NewView(b.String())
}
