package screens

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/danielxxomg/bak-cli/internal/tui/components"
	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// restoreState represents the current state of the restore flow.
type restoreState int

const (
	// restoreStateList shows the backup selection table.
	restoreStateList restoreState = iota
	// restoreStateDryRun shows the dry-run diff preview.
	restoreStateDryRun
	// restoreStateConfirm shows the confirmation modal.
	restoreStateConfirm
	// restoreStateRunning shows restore execution progress.
	restoreStateRunning
	// restoreStateDone shows the completion status.
	restoreStateDone
)

// restoreBackupsLoadedMsg is an internal message sent when backups finish loading.
type restoreBackupsLoadedMsg struct {
	backups []BackupInfo
	err     error
}

// restoreDryRunResultMsg is an internal message sent when a dry run completes.
type restoreDryRunResultMsg struct {
	output string
	err    error
}

// restoreExecResultMsg is an internal message sent when restore execution completes.
type restoreExecResultMsg struct {
	err error
}

// RestoreModel is the Bubble Tea sub-model for the restore flow:
// select a backup → preview diff → confirm → execute.
type RestoreModel struct {
	State  restoreState
	Cursor int
	Width  int
	Height int

	// Backups loaded from disk.
	Backups []BackupInfo
	// SelectedID is the backup chosen for restore.
	SelectedID string
	// DryRunOutput holds the diff preview text.
	DryRunOutput string
	// Err holds the last error.
	Err error

	// Deps.
	listBackups func() ([]BackupInfo, error)
	runRestore  func(backupID string, dryRun bool) (string, error)

	// Modal for confirm/cancel.
	Modal *components.ModalModel
}

// NewRestoreModel creates a RestoreModel with the provided dependencies.
func NewRestoreModel(
	listBackups func() ([]BackupInfo, error),
	runRestore func(backupID string, dryRun bool) (string, error),
) RestoreModel {
	return RestoreModel{
		State:       restoreStateList,
		Cursor:      0,
		listBackups: listBackups,
		runRestore:  runRestore,
	}
}

// Init triggers loading the backup list.
func (m RestoreModel) Init() tea.Cmd {
	return func() tea.Msg {
		if m.listBackups == nil {
			return restoreBackupsLoadedMsg{backups: nil, err: nil}
		}
		backups, err := m.listBackups()
		return restoreBackupsLoadedMsg{backups: backups, err: err}
	}
}

// Update handles keyboard navigation, state transitions, and message processing.
func (m RestoreModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case restoreBackupsLoadedMsg:
		if msg.err != nil {
			m.Err = msg.err
			return m, nil
		}
		m.Backups = msg.backups
		return m, nil

	case restoreDryRunResultMsg:
		if msg.err != nil {
			m.Err = msg.err
			return m, nil
		}
		m.DryRunOutput = msg.output
		m.State = restoreStateDryRun
		return m, nil

	case restoreExecResultMsg:
		if msg.err != nil {
			m.Err = msg.err
			m.State = restoreStateDone
			return m, nil
		}
		m.State = restoreStateDone
		return m, nil

	case components.ModalResultMsg:
		if m.State == restoreStateConfirm && m.Modal != nil {
			m.Modal = nil
			if msg.Confirmed {
				m.State = restoreStateRunning
				return m, m.runRestoreCmd(m.SelectedID)
			}
			m.State = restoreStateList
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

func (m RestoreModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.State {
	case restoreStateList:
		switch msg.Code {
		case 'q', 27:
			return m, func() tea.Msg { return ScreenBackMsg{} }
		case 'j', tea.KeyDown:
			if len(m.Backups) > 0 {
				m.Cursor = (m.Cursor + 1) % len(m.Backups)
			}
		case 'k', tea.KeyUp:
			if len(m.Backups) > 0 {
				m.Cursor = (m.Cursor - 1 + len(m.Backups)) % len(m.Backups)
			}
		case '\r':
			if len(m.Backups) > 0 && m.Cursor < len(m.Backups) {
				m.SelectedID = m.Backups[m.Cursor].ID
				m.State = restoreStateDryRun
				return m, m.dryRunCmd(m.SelectedID)
			}
		}

	case restoreStateDryRun:
		switch msg.Code {
		case 'q', 27:
			m.State = restoreStateList
			return m, nil
		case '\r':
			m.State = restoreStateConfirm
			modal := components.NewModal("Confirm Restore",
				fmt.Sprintf("Restore backup %s? This will overwrite current config.", m.SelectedID),
				[]string{"Confirm", "Cancel"})
			m.Modal = &modal
			return m, nil
		}

	case restoreStateDone:
		switch msg.Code {
		case 'q', 27, '\r', ' ':
			return m, func() tea.Msg { return ScreenBackMsg{} }
		}
	}

	return m, nil
}

func (m RestoreModel) dryRunCmd(backupID string) tea.Cmd {
	return func() tea.Msg {
		if m.runRestore == nil {
			return restoreDryRunResultMsg{output: "(no dry-run available)", err: nil}
		}
		output, err := m.runRestore(backupID, true)
		return restoreDryRunResultMsg{output: output, err: err}
	}
}

func (m RestoreModel) runRestoreCmd(backupID string) tea.Cmd {
	return func() tea.Msg {
		if m.runRestore == nil {
			return restoreExecResultMsg{err: fmt.Errorf("restore not available")}
		}
		_, err := m.runRestore(backupID, false)
		return restoreExecResultMsg{err: err}
	}
}

// View renders the current restore state.
func (m RestoreModel) View() tea.View {
	var content string

	if m.State == restoreStateList && len(m.Backups) == 0 && m.Err == nil {
		content = m.renderEmptyState()
	} else if m.Err != nil {
		content = m.renderErrorState()
	} else {
		switch m.State {
		case restoreStateList:
			content = m.renderBackupList()
		case restoreStateDryRun:
			content = m.renderDryRun()
		case restoreStateConfirm:
			content = m.renderConfirm()
		case restoreStateRunning:
			content = m.renderRunning()
		case restoreStateDone:
			content = m.renderDone()
		}
	}

	// Overlay modal if present.
	if m.Modal != nil {
		modal := *m.Modal
		modal.Width = m.Width
		modal.Height = m.Height
		overlay := modal.View().Content
		content = overlay
	}

	return tea.NewView(content)
}

func (m RestoreModel) renderEmptyState() string {
	var b strings.Builder
	b.WriteString(styles.ScreenTitleStyle.Render("Restore"))
	b.WriteString("\n\n")
	b.WriteString("No backups found. Create one first.")
	b.WriteString("\n\n")
	b.WriteString("[q] back")
	return b.String()
}

func (m RestoreModel) renderErrorState() string {
	var b strings.Builder
	b.WriteString(styles.ScreenTitleStyle.Render("Restore"))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "Error: %v", m.Err)
	b.WriteString("\n\n")
	b.WriteString("[q] back")
	return b.String()
}

func (m RestoreModel) renderBackupList() string {
	var b strings.Builder
	b.WriteString(styles.ScreenTitleStyle.Render("Restore — Select a Backup"))
	b.WriteString("\n\n")

	for i, backup := range m.Backups {
		cursor := "  "
		if i == m.Cursor {
			cursor = styles.CursorIndicator
		}
		line := fmt.Sprintf("%s%s  %s  %s  %s", cursor, backup.ID, backup.Date, backup.Size, backup.Cloud)
		if i == m.Cursor {
			line = styles.SelectedStyle.Render(line)
		}
		b.WriteString(line)
		if i < len(m.Backups)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n\n")
	b.WriteString("[↑/↓] navigate  [enter] select  [q] back")
	return b.String()
}

func (m RestoreModel) renderDryRun() string {
	var b strings.Builder
	b.WriteString(styles.ScreenTitleStyle.Render("Restore — Dry Run Preview"))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "Backup: %s\n\n", m.SelectedID)
	b.WriteString(m.DryRunOutput)
	b.WriteString("\n\n")
	b.WriteString("[enter] Confirm restore  [q] Cancel")
	return b.String()
}

func (m RestoreModel) renderConfirm() string {
	// When modal is present, the View will render it above.
	return styles.ScreenTitleStyle.Render("Restore — Confirm") + "\n\nConfirm restore of " + m.SelectedID
}

func (m RestoreModel) renderRunning() string {
	return styles.ScreenTitleStyle.Render("Restore") + "\n\nRestoring backup " + m.SelectedID + "..."
}

func (m RestoreModel) renderDone() string {
	var b strings.Builder
	b.WriteString(styles.ScreenTitleStyle.Render("Restore"))
	b.WriteString("\n\n")
	if m.Err != nil {
		fmt.Fprintf(&b, "Error: %v", m.Err)
	} else {
		b.WriteString("Restore completed successfully.")
	}
	b.WriteString("\n\n")
	b.WriteString("[enter/q] back to menu")
	return b.String()
}
