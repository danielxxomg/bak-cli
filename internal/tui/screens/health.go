package screens

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/danielxxomg/bak-cli/internal/tui/components"
	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// HealthCheck represents a single diagnostic check in the health screen.
type HealthCheck struct {
	Name   string
	Status StepStatus
	Detail string
}

// healthCheckResultMsg is sent after a simulated health check completes.
type healthCheckResultMsg struct {
	index  int
	detail string
}

// HealthModel is the Bubble Tea sub-model for the backup health check screen.
// Pressing enter starts a sequence of simulated health checks (config exists,
// backup dir valid, git configured, cloud reachable). Each check runs as a
// tea.Cmd and reports its result asynchronously.
type HealthModel struct {
	checks  []HealthCheck
	running bool
	width   int
	height  int
}

// healthCheckNames are the names of the health checks in execution order.
var healthCheckNames = []string{
	"Config exists",
	"Backup directory valid",
	"Git configured",
	"Cloud reachable",
}

// NewHealthModel creates a new HealthModel in idle state with no checks.
func NewHealthModel() HealthModel {
	return HealthModel{}
}

// Init returns nil — no initial side effects.
func (m HealthModel) Init() tea.Cmd {
	return nil
}

// Update handles keyboard input: enter starts the health checks, q/esc
// navigates back when idle. WindowSizeMsg updates stored dimensions.
func (m HealthModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		switch msg.Code {
		case '\r':
			if !m.running {
				return m.startChecks()
			}
		case 'q', 27:
			if !m.running {
				return m, func() tea.Msg { return ScreenBackMsg{} }
			}
		}

	case healthCheckResultMsg:
		if msg.index >= 0 && msg.index < len(m.checks) {
			m.checks[msg.index].Detail = msg.detail
			m.checks[msg.index].Status = StepDone
		}
		// Check if all checks are done.
		allDone := true
		for _, c := range m.checks {
			if c.Status != StepDone {
				allDone = false
				break
			}
		}
		if allDone {
			m.running = false
		}
		return m, nil
	}

	return m, nil
}

// startChecks initializes the health check list and fires off simulated
// checks as tea.Cmd functions.
func (m HealthModel) startChecks() (tea.Model, tea.Cmd) {
	m.running = true
	m.checks = make([]HealthCheck, len(healthCheckNames))
	for i, name := range healthCheckNames {
		m.checks[i] = HealthCheck{
			Name:   name,
			Status: StepRunning,
		}
	}

	// Fire all checks concurrently with staggered delays.
	var cmds []tea.Cmd
	for i := range healthCheckNames {
		idx := i
		cmds = append(cmds, tea.Tick(time.Duration(i+1)*150*time.Millisecond, func(_ time.Time) tea.Msg {
			return healthCheckResultMsg{index: idx, detail: "OK"}
		}))
	}
	return m, tea.Batch(cmds...)
}

// View renders the health check screen with a list of checks and their
// status indicators (✓/✗/⠹/○).
func (m HealthModel) View() tea.View {
	if styles.IsTooSmall(m.width, m.height) {
		return tea.NewView(styles.RenderTooSmall(m.width, m.height))
	}

	var b strings.Builder

	b.WriteString(styles.ScreenTitleStyle.Render("Health Check"))
	b.WriteString("\n\n")

	if len(m.checks) == 0 && !m.running {
		b.WriteString(styles.HelpStyle.Render("Press enter to run health check"))
		b.WriteString("\n\n")
		helpKeys := []components.HelpKey{
			{Key: "enter", Desc: "run"},
			{Key: "q", Desc: "back"},
		}
		b.WriteString(components.RenderHelp(helpKeys))
		return tea.NewView(b.String())
	}

	for _, check := range m.checks {
		indicator, style := healthIndicator(check.Status)
		b.WriteString("  ")
		b.WriteString(style.Render(indicator + " " + check.Name))
		if check.Detail != "" {
			b.WriteString(" — ")
			b.WriteString(styles.HelpStyle.Render(check.Detail))
		}
		b.WriteString("\n")
	}

	if !m.running && len(m.checks) > 0 {
		b.WriteString("\n\n")
		helpKeys := []components.HelpKey{
			{Key: "q", Desc: "back"},
			{Key: "enter", Desc: "rerun"},
		}
		b.WriteString(components.RenderHelp(helpKeys))
	}

	return tea.NewView(b.String())
}

// healthIndicator returns the visual indicator and style for a health check status.
func healthIndicator(status StepStatus) (string, lipgloss.Style) {
	switch status {
	case StepDone:
		return "\u2713", styles.ProgressDoneStyle
	case StepRunning:
		return "\u28f9", styles.ProgressRunningStyle
	default:
		return "○", styles.ProgressPendingStyle
	}
}
