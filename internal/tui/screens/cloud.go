package screens

import (
	"fmt"
	"strings"

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

// RenderCloudStatus renders the cloud sync status screen. It displays the
// provider name, connection status, last sync time, and local vs cloud
// backup counts. If no provider is configured, a message is shown.
//
// On wide terminals (width >= 50), the content is wrapped in a Frame.
func RenderCloudStatus(info CloudInfo, width int) string {
	var b strings.Builder

	b.WriteString(styles.CloudTitleStyle.Render("Cloud Sync"))
	b.WriteString("\n\n")

	// No provider configured.
	if info.Provider == "" {
		content := styles.CloudEmptyStyle.Render("No cloud provider configured")
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

	content := b.String()
	if width >= 50 {
		content = styles.Frame(content, width-4)
	}
	return content
}
