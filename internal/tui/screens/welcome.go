package screens

import (
	"strings"

	"github.com/danielxxomg/bak-cli/internal/tui/styles"
)

// ShouldShowWelcome returns true if the bak-cli configuration does not
// exist on this machine, indicating the user is running bak for the first
// time. A nil configExists function is treated as "config exists" (safe
// default — don't show welcome unnecessarily).
func ShouldShowWelcome(configExists func() bool) bool {
	if configExists == nil {
		return false
	}
	return !configExists()
}

// RenderWelcome renders the first-run welcome screen with a greeting,
// setup prompt, and continue instruction. On wide terminals (width >= 50),
// the content is wrapped in a DoubleBorder Frame for visual framing.
func RenderWelcome(width int) string {
	var b strings.Builder

	// Welcome heading.
	b.WriteString(styles.HeadingStyle.Render("Welcome to bak!"))
	b.WriteString("\n\n")

	// Setup prompt.
	b.WriteString("It looks like this is your first time running bak.\n")
	b.WriteString("Let's set up your backup configuration.\n")
	b.WriteString("\n")

	// Continue instruction.
	b.WriteString("Press ")
	b.WriteString(styles.SelectedStyle.Render("enter"))
	b.WriteString(" to continue, or ")
	b.WriteString(styles.SelectedStyle.Render("q"))
	b.WriteString(" to quit.")

	content := b.String()
	if width >= 50 {
		content = styles.Frame(content, width-4)
	}
	return content
}
