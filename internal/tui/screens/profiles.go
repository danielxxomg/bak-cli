package screens

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/danielxxomg/bak-cli/internal/tui/components"
	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// ProfileInfo holds profile data for display in the profiles screen.
type ProfileInfo struct {
	Name     string
	Provider string
	Preset   string
	Active   bool
}

// profilesLoadMsg is an internal message sent when profiles finish loading.
type profilesLoadMsg struct {
	profiles []ProfileInfo
	err      error
}

// wizardResultMsg is an internal message sent when the wizard completes.
type wizardResultMsg struct {
	profile ProfileInfo
	err     error
}

// ProfilesModel is the Bubble Tea sub-model for the profiles management screen.
type ProfilesModel struct {
	Cursor   int
	Width    int
	Height   int
	Profiles []ProfileInfo
	Err      error
	Msg      string // toast-like message for errors/warnings

	listProfiles  func() ([]ProfileInfo, error)
	setActive     func(name string) error
	deleteProfile func(name string) error
	runWizard     func() (ProfileInfo, error)
	SaveProfile   func(name string, p ProfileInfo) error

	Modal *components.ModalModel
}

// NewProfilesModel creates a ProfilesModel with the provided dependencies.
func NewProfilesModel(
	listFn func() ([]ProfileInfo, error),
	switchFn func(name string) error,
	deleteFn func(name string) error,
	wizardFn func() (ProfileInfo, error),
) ProfilesModel {
	return ProfilesModel{
		Cursor:        0,
		listProfiles:  listFn,
		setActive:     switchFn,
		deleteProfile: deleteFn,
		runWizard:     wizardFn,
	}
}

// Init triggers loading the profile list.
func (m ProfilesModel) Init() tea.Cmd {
	return func() tea.Msg {
		if m.listProfiles == nil {
			return profilesLoadMsg{profiles: nil, err: nil}
		}
		profiles, err := m.listProfiles()
		return profilesLoadMsg{profiles: profiles, err: err}
	}
}

// Update handles navigation, profile operations, and modal interactions.
func (m ProfilesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case profilesLoadMsg:
		if msg.err != nil {
			m.Err = msg.err
			return m, nil
		}
		m.Profiles = msg.profiles
		return m, nil

	case wizardResultMsg:
		if msg.err != nil {
			m.Msg = msg.err.Error()
			return m, nil
		}
		m.Profiles = append(m.Profiles, msg.profile)
		if m.SaveProfile != nil {
			_ = m.SaveProfile(msg.profile.Name, msg.profile)
		}
		return m, nil

	case components.ModalResultMsg:
		if m.Modal != nil {
			modal := m.Modal
			m.Modal = nil
			if msg.Confirmed && m.Cursor < len(m.Profiles) && m.deleteProfile != nil {
				name := m.Profiles[m.Cursor].Name
				_ = m.deleteProfile(name)
				// Remove from local list.
				m.Profiles = append(m.Profiles[:m.Cursor], m.Profiles[m.Cursor+1:]...)
				if m.Cursor >= len(m.Profiles) && len(m.Profiles) > 0 {
					m.Cursor = len(m.Profiles) - 1
				}
			}
			_ = modal // suppress unused
			return m, nil
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	// Forward to modal if active.
	if m.Modal != nil {
		newModal, cmd := m.Modal.Update(msg)
		m2 := newModal.(components.ModalModel)
		m.Modal = &m2
		return m, cmd
	}

	return m, nil
}

func (m ProfilesModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case 'q', 27:
		return m, func() tea.Msg { return ScreenBackMsg{} }
	case 'j', tea.KeyDown:
		if len(m.Profiles) > 0 {
			m.Cursor = (m.Cursor + 1) % len(m.Profiles)
		}
	case 'k', tea.KeyUp:
		if len(m.Profiles) > 0 {
			m.Cursor = (m.Cursor - 1 + len(m.Profiles)) % len(m.Profiles)
		}
	case '\r':
		if len(m.Profiles) > 0 && m.Cursor < len(m.Profiles) {
			name := m.Profiles[m.Cursor].Name
			if m.setActive != nil {
				_ = m.setActive(name)
			}
			// Mark as active locally.
			for i := range m.Profiles {
				m.Profiles[i].Active = m.Profiles[i].Name == name
			}
		}
	case 'd':
		if len(m.Profiles) > 0 && m.Cursor < len(m.Profiles) {
			if m.Profiles[m.Cursor].Active {
				m.Msg = "Cannot delete the active profile"
				return m, nil
			}
			// Show confirmation modal.
			modal := components.NewModal("Delete Profile",
				fmt.Sprintf("Delete profile %q?", m.Profiles[m.Cursor].Name),
				[]string{"Delete", "Cancel"})
			m.Modal = &modal
		}
	case 'n':
		if m.runWizard != nil {
			return m, func() tea.Msg {
				profile, err := m.runWizard()
				return wizardResultMsg{profile: profile, err: err}
			}
		}
	}

	return m, nil
}

// View renders the profiles list or empty state.
func (m ProfilesModel) View() tea.View {
	if m.Width < styles.MinWidth || m.Height < styles.MinHeight {
		return tea.NewView("Terminal too small")
	}

	var content string

	if len(m.Profiles) == 0 && m.Err == nil {
		content = m.renderEmpty()
	} else if m.Err != nil {
		content = m.renderError()
	} else {
		content = m.renderList()
	}

	// Overlay modal if present.
	if m.Modal != nil {
		modal := *m.Modal
		modal.Width = m.Width
		modal.Height = m.Height
		return modal.View()
	}

	return tea.NewView(content)
}

func (m ProfilesModel) renderEmpty() string {
	var b strings.Builder
	b.WriteString(styles.ScreenTitleStyle.Render("Profiles"))
	b.WriteString("\n\n")
	b.WriteString("No profiles yet. Press 'n' to create one.")
	b.WriteString("\n\n")
	if m.Msg != "" {
		b.WriteString(styles.SelectedStyle.Render(m.Msg))
		b.WriteString("\n\n")
	}
	b.WriteString("[n] new  [q] back")
	return b.String()
}

func (m ProfilesModel) renderError() string {
	var b strings.Builder
	b.WriteString(styles.ScreenTitleStyle.Render("Profiles"))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "Error: %v", m.Err)
	b.WriteString("\n\n")
	b.WriteString("[q] back")
	return b.String()
}

func (m ProfilesModel) renderList() string {
	var b strings.Builder
	b.WriteString(styles.ScreenTitleStyle.Render("Profiles"))
	b.WriteString("\n\n")

	for i, p := range m.Profiles {
		cursor := "  "
		if i == m.Cursor {
			cursor = styles.CursorIndicator
		}
		activeMark := " "
		if p.Active {
			activeMark = "*"
		}
		line := fmt.Sprintf("%s%s %s  %s  %s", cursor, activeMark, p.Name, p.Provider, p.Preset)
		if i == m.Cursor {
			line = styles.SelectedStyle.Render(line)
		}
		b.WriteString(line)
		if i < len(m.Profiles)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n\n")
	if m.Msg != "" {
		b.WriteString(styles.SelectedStyle.Render(m.Msg))
		b.WriteString("\n\n")
	}
	b.WriteString("[↑/↓] navigate  [enter] switch  [n] new  [d] delete  [q] back")
	return b.String()
}
