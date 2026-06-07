package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/danielxxomg/bak-cli/internal/actions"
	"github.com/spf13/cobra"
)

// pickCmd represents the interactive category picker command.
var pickCmd = &cobra.Command{
	Use:   "pick",
	Short: "Interactively select categories and run a backup",
	Long: `Opens a terminal UI where you can select which categories to back up
using the keyboard. Press space to toggle a category, enter to confirm,
and q or esc to quit.

The selected categories are passed to the backup engine as if you had
run 'bak backup' with a custom category set.

Examples:
  bak pick`,
	RunE: runPick,
}

func init() {
	rootCmd.AddCommand(pickCmd)
}

// categoryItem represents a selectable category in the TUI.
type categoryItem struct {
	name    string
	checked bool
}

// pickModel is the bubbletea model for the category picker.
type pickModel struct {
	items    []categoryItem
	cursor   int
	quitting bool
	confirmed bool
}

func (m pickModel) Init() tea.Cmd {
	return nil
}

func (m pickModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}

		case " ":
			if len(m.items) > 0 {
				m.items[m.cursor].checked = !m.items[m.cursor].checked
			}

		case "enter":
			m.confirmed = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m pickModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	b.WriteString(titleStyle.Render("Select categories to backup"))
	b.WriteString("\n\n")

	checkedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	uncheckedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))

	for i, item := range m.items {
		cursor := "  "
		if m.cursor == i {
			cursor = cursorStyle.Render("> ")
		}

		check := uncheckedStyle.Render("[ ]")
		if item.checked {
			check = checkedStyle.Render("[x]")
		}

		b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, check, item.name))
	}

	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("space: toggle • enter: confirm • q/esc: quit"))

	return b.String()
}

// Selected returns the names of checked categories.
func (m pickModel) Selected() []string {
	var selected []string
	for _, item := range m.items {
		if item.checked {
			selected = append(selected, item.name)
		}
	}
	return selected
}

func runPick(cmd *cobra.Command, args []string) error {
	return runPickWithDeps(cmd, args, depsFromCmd(cmd))
}

func runPickWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
	if !isTTY() {
		return fmt.Errorf("interactive picker requires a terminal (TTY)")
	}

	action := &actions.PickBackupAction{
		Stdout:  deps.Stdout,
		Verbose: verbose,
		Picker: func(categories []actions.CategoryItem) (actions.PickResult, error) {
			items := make([]categoryItem, len(categories))
			for i, c := range categories {
				items[i] = categoryItem{name: c.Name, checked: c.Checked}
			}

			m := pickModel{items: items}
			p := tea.NewProgram(m)
			result, err := p.Run()
			if err != nil {
				return actions.PickResult{}, err
			}

			model, ok := result.(pickModel)
			if !ok {
				return actions.PickResult{}, fmt.Errorf("unexpected model type: %T", result)
			}

			return actions.PickResult{
				Selected:  model.Selected(),
				Confirmed: model.confirmed && !model.quitting,
			}, nil
		},
	}

	return action.Run()
}
