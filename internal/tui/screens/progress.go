package screens

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// StepStatus represents the current state of a backup/restore step.
type StepStatus int

const (
	// StepPending means the step has not started yet.
	StepPending StepStatus = iota
	// StepRunning means the step is currently executing.
	StepRunning
	// StepDone means the step has completed successfully.
	StepDone
)

// Step represents a single step in the backup/restore pipeline.
type Step struct {
	Name   string
	Status StepStatus
}

// ProgressStepMsg is sent by the backup goroutine to report step progress.
type ProgressStepMsg struct {
	Step    string
	Current int
	Total   int
}

// ProgressDoneMsg is sent when the backup/restore operation completes.
type ProgressDoneMsg struct{}

// progressStepDoneIndicator is displayed for completed steps.
const progressStepDoneIndicator = "✓"

// progressStepRunningIndicator is displayed for the currently running step.
const progressStepRunningIndicator = "⠹"

// progressStepPendingIndicator is displayed for steps not yet started.
const progressStepPendingIndicator = "○"

// ProgressModel is the Bubble Tea sub-model for the progress screen.
// It displays a spinner, progress bar, and step list during backup/restore.
type ProgressModel struct {
	spinner  spinner.Model
	progress progress.Model
	steps    []Step
	running  bool
	width    int
	height   int
}

// NewProgressModel creates a ProgressModel with initialized spinner and
// progress bar sub-models. The spinner starts with default styling and
// the progress bar uses Rose Pine colors.
func NewProgressModel() ProgressModel {
	sp := spinner.New(
		spinner.WithStyle(lipgloss.NewStyle().Foreground(styles.ColorGold)),
	)
	pg := progress.New(
		progress.WithDefaultBlend(),
		progress.WithWidth(40),
	)

	return ProgressModel{
		spinner:  sp,
		progress: pg,
	}
}

// Init starts the spinner animation by returning a spinner.Tick command.
func (m ProgressModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles incoming messages for the progress screen.
func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		// Block back navigation while running.
		if m.running {
			return m, nil
		}
		switch msg.Code {
		case 'q', 27: // esc
			return m, func() tea.Msg { return ScreenBackMsg{} }
		}

	case spinner.TickMsg:
		if !m.running {
			return m, nil
		}
		newSp, cmd := m.spinner.Update(msg)
		m.spinner = newSp
		return m, cmd

	case ProgressStepMsg:
		m.running = true
		// Mark previous steps as done, add new step as running.
		for i := range m.steps {
			m.steps[i].Status = StepDone
		}
		m.steps = append(m.steps, Step{
			Name:   msg.Step,
			Status: StepRunning,
		})
		// Update progress bar.
		ratio := float64(msg.Current) / float64(msg.Total)
		pgCmd := m.progress.SetPercent(ratio)
		return m, tea.Batch(m.spinner.Tick, pgCmd)

	case ProgressDoneMsg:
		m.running = false
		// Mark all steps as done.
		for i := range m.steps {
			m.steps[i].Status = StepDone
		}
		m.progress.SetPercent(1.0)
		return m, func() tea.Msg { return ScreenBackMsg{} }
	}

	return m, nil
}

// View renders the progress screen with spinner, progress bar, and step list.
func (m ProgressModel) View() tea.View {
	var b strings.Builder

	b.WriteString(styles.ProgressTitleStyle.Render("Progress"))
	b.WriteString("\n\n")

	// Spinner row.
	spin := m.spinner.View()
	b.WriteString(fmt.Sprintf("  %s", spin))
	b.WriteString("\n\n")

	// Progress bar.
	bar := m.progress.View()
	b.WriteString(bar)
	b.WriteString("\n\n")

	// Step list.
	if len(m.steps) > 0 {
		for _, step := range m.steps {
			indicator, style := stepIndicator(step.Status)
			b.WriteString("  ")
			b.WriteString(style.Render(indicator + " " + step.Name))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Completion message.
	if !m.running && len(m.steps) > 0 {
		b.WriteString(styles.ProgressCompleteStyle.Render("Complete!"))
		b.WriteString("\n\n")
		helpText := styles.HelpStyle.Render("q quit • enter continue")
		b.WriteString("  ")
		b.WriteString(helpText)
	}

	return tea.NewView(b.String())
}

// stepIndicator returns the visual indicator and style for a step status.
func stepIndicator(status StepStatus) (string, lipgloss.Style) {
	switch status {
	case StepDone:
		return progressStepDoneIndicator, styles.ProgressDoneStyle
	case StepRunning:
		return progressStepRunningIndicator, styles.ProgressRunningStyle
	default:
		return progressStepPendingIndicator, styles.ProgressPendingStyle
	}
}
