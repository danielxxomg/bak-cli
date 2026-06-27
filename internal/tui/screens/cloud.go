package screens

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/danielxxomg/bak-cli/internal/tui/components"
	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// CloudInfo holds the cloud provider status data for the cloud screen.
type CloudInfo struct {
	Provider   string
	Connected  bool
	LastSync   string
	LocalCount int
	CloudCount int
}

// cloudStatusMsg is an internal message sent when the cloud status loads.
type cloudStatusMsg struct {
	info CloudInfo
	err  error
}

// CloudModel is the Bubble Tea sub-model for the cloud sync screen.
// It fetches real cloud provider data via GetCloudStatus instead of
// rendering an empty CloudInfo{}.
type CloudModel struct {
	Info      CloudInfo
	Err       error
	Width     int
	Height    int
	getStatus func() (CloudInfo, error)
}

// NewCloudModel creates a CloudModel with the given status function.
func NewCloudModel(getStatus func() (CloudInfo, error)) CloudModel {
	return CloudModel{
		getStatus: getStatus,
	}
}

// Init triggers loading the cloud status.
func (m CloudModel) Init() tea.Cmd {
	return func() tea.Msg {
		if m.getStatus == nil {
			return cloudStatusMsg{info: CloudInfo{}, err: nil}
		}
		info, err := m.getStatus()
		return cloudStatusMsg{info: info, err: err}
	}
}

// Update handles WindowSizeMsg and cloud status loading.
func (m CloudModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case cloudStatusMsg:
		if msg.err != nil {
			m.Err = msg.err
			return m, nil
		}
		m.Info = msg.info
		return m, nil

	case tea.KeyPressMsg:
		if msg.Code == 'q' || msg.Code == 27 {
			return m, func() tea.Msg { return ScreenBackMsg{} }
		}
	}

	return m, nil
}

// View renders the cloud status using the loaded data or the render function.
func (m CloudModel) View() tea.View {
	if m.Info.Provider == "" && m.Err == nil {
		content := RenderCloudStatus(CloudInfo{}, m.Width)
		return tea.NewView(content)
	}
	if m.Err != nil {
		content := RenderCloudStatus(CloudInfo{}, m.Width)
		return tea.NewView(content)
	}
	content := RenderCloudStatus(m.Info, m.Width)
	return tea.NewView(content)
}

// RenderCloudStatus renders the cloud sync status screen. It displays the
// provider name, connection status, last sync time, and local vs cloud
// backup counts. If no provider is configured, a message is shown.
//
// On wide terminals (width >= 50), the content is wrapped in a Frame.
// A contextual help bar is appended below the content.
func RenderCloudStatus(info CloudInfo, width int) string {
	var b strings.Builder

	b.WriteString(styles.CloudTitleStyle.Render("Cloud Sync"))
	b.WriteString("\n\n")

	// No provider configured (styled icon + message + hint, REQ-TP-007).
	if info.Provider == "" {
		b.WriteString(components.RenderEmptyState("\u2601", "No cloud provider configured", "Run 'bak cloud login' to connect"))
		b.WriteString("\n\n")
		b.WriteString(renderCloudHelp())
		content := b.String()
		if width >= 50 {
			content = styles.Frame(content, width-4)
		}
		return content
	}

	// Provider name.
	b.WriteString(styles.CloudLabelStyle.Render("Provider:"))
	b.WriteString(" ")
	b.WriteString(styles.CloudValueStyle.Render(info.Provider))
	b.WriteString("\n\n")

	// Connection status.
	b.WriteString(styles.CloudLabelStyle.Render("Status:"))
	b.WriteString(" ")
	if info.Connected {
		b.WriteString(styles.CloudConnectedStyle.Render("✓ Connected"))
	} else {
		b.WriteString(styles.CloudDisconnectedStyle.Render("✗ Disconnected"))
	}
	b.WriteString("\n\n")

	// Last sync.
	b.WriteString(styles.CloudLabelStyle.Render("Last Sync:"))
	b.WriteString(" ")
	b.WriteString(styles.CloudValueStyle.Render(info.LastSync))
	b.WriteString("\n\n")

	// Backup counts.
	b.WriteString(styles.CloudLabelStyle.Render("Local Backups:"))
	b.WriteString(" ")
	b.WriteString(styles.CloudValueStyle.Render(fmt.Sprintf("%d", info.LocalCount)))
	b.WriteString("\n")

	b.WriteString(styles.CloudLabelStyle.Render("Cloud Backups:"))
	b.WriteString(" ")
	b.WriteString(styles.CloudValueStyle.Render(fmt.Sprintf("%d", info.CloudCount)))
	b.WriteString("\n")

	// Contextual help bar.
	b.WriteString("\n")
	b.WriteString(renderCloudHelp())

	content := b.String()
	if width >= 50 {
		content = styles.Frame(content, width-4)
	}
	return content
}

// renderCloudHelp returns the cloud screen help bar.
func renderCloudHelp() string {
	helpKeys := []components.HelpKey{
		{Key: "q", Desc: keyBack},
	}
	return components.RenderHelp(helpKeys)
}
