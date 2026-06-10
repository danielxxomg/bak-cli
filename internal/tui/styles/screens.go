package styles

import (
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
)

// Screen-specific dashboard styles.
var (
	// DashboardTableStyle is applied to the bubbles table in the dashboard screen.
	DashboardTableStyle = func() table.Styles {
		s := table.DefaultStyles()
		s.Header = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorGold).
			Padding(0, 1)
		s.Cell = lipgloss.NewStyle().
			Foreground(ColorText).
			Padding(0, 1)
		s.Selected = lipgloss.NewStyle().
			Foreground(ColorRose).
			Bold(true).
			Padding(0, 1)
		return s
	}()

	// DashboardEmptyStyle is the style for the empty-state message on the dashboard.
	DashboardEmptyStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Padding(1, 2)

	// DashboardErrorStyle is the style for error messages on the dashboard.
	DashboardErrorStyle = lipgloss.NewStyle().
				Foreground(ColorLove).
				Padding(1, 2)

	// DashboardTitleStyle is the style for the dashboard heading.
	DashboardTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorLavender).
				Padding(0, 1)
)

// Screen-specific progress styles.
var (
	// ProgressTitleStyle is the style for the progress screen heading.
	ProgressTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorLavender).
				Padding(0, 1)

	// ProgressDoneStyle is used for completed steps.
	ProgressDoneStyle = lipgloss.NewStyle().
				Foreground(ColorPine).
				Padding(0, 1)

	// ProgressRunningStyle is used for the currently running step.
	ProgressRunningStyle = lipgloss.NewStyle().
				Foreground(ColorGold).
				Padding(0, 1)

	// ProgressPendingStyle is used for pending steps.
	ProgressPendingStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Padding(0, 1)

	// ProgressCompleteStyle is shown when all steps are done.
	ProgressCompleteStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorPine).
				Padding(0, 1)
)

// Screen-specific cloud styles.
var (
	// CloudTitleStyle is the style for the cloud screen heading.
	CloudTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorLavender).
			Padding(0, 1)

	// CloudLabelStyle is the style for field labels.
	CloudLabelStyle = lipgloss.NewStyle().
			Foreground(ColorSubtle).
			Padding(0, 1)

	// CloudConnectedStyle indicates connected status.
	CloudConnectedStyle = lipgloss.NewStyle().
				Foreground(ColorPine).
				Padding(0, 1)

	// CloudDisconnectedStyle indicates disconnected status.
	CloudDisconnectedStyle = lipgloss.NewStyle().
				Foreground(ColorLove).
				Padding(0, 1)

	// CloudValueStyle is the style for field values.
	CloudValueStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Padding(0, 1)

	// CloudEmptyStyle is used for the no-provider message.
	CloudEmptyStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(1, 2)
)
