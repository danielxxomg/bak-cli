// Package screens provides screen-specific render functions and sub-models
// for the bak-cli TUI. This file contains TDD tests written BEFORE the
// production code (strict RED phase) for the cloud status screen.
package screens

import (
	"strings"
	"testing"
)

// =============================================================================
// TestRenderCloudStatus — RED (cloud.go does not exist yet)
// =============================================================================

func TestRenderCloudStatus(t *testing.T) {
	info := CloudInfo{
		Provider:   "Google Drive",
		Connected:  true,
		LastSync:   "2024-06-09 14:30 UTC",
		LocalCount: 12,
		CloudCount: 10,
	}

	output := RenderCloudStatus(info, 80)

	if len(output) == 0 {
		t.Fatal("RenderCloudStatus returned empty string")
	}

	// Must contain provider name.
	if !strings.Contains(output, "Google Drive") {
		t.Error("RenderCloudStatus missing provider name 'Google Drive'")
	}

	// Must contain connection status.
	if !strings.Contains(output, "Connected") {
		t.Error("RenderCloudStatus missing connection status 'Connected'")
	}

	// Must contain last sync time.
	if !strings.Contains(output, "2024-06-09") {
		t.Error("RenderCloudStatus missing last sync time")
	}

	// Must contain local and cloud counts.
	if !strings.Contains(output, "12") {
		t.Error("RenderCloudStatus missing local count '12'")
	}
	if !strings.Contains(output, "10") {
		t.Error("RenderCloudStatus missing cloud count '10'")
	}
}

// =============================================================================
// TestRenderCloudStatus_NoProvider — RED
// =============================================================================

func TestRenderCloudStatus_NoProvider(t *testing.T) {
	info := CloudInfo{
		Provider: "",
	}

	output := RenderCloudStatus(info, 80)

	if !strings.Contains(output, "No cloud provider configured") {
		t.Errorf("RenderCloudStatus no-provider missing message: %q", output)
	}

	if len(output) == 0 {
		t.Fatal("RenderCloudStatus returned empty string for no provider")
	}
}

// =============================================================================
// TestRenderCloudStatus_Counts — RED
// =============================================================================

func TestRenderCloudStatus_Counts(t *testing.T) {
	tests := []struct {
		name      string
		info      CloudInfo
		wantLocal string
		wantCloud string
	}{
		{
			name: "both have backups",
			info: CloudInfo{
				Provider:   "S3",
				Connected:  true,
				LastSync:   "2024-01-01",
				LocalCount: 5,
				CloudCount: 5,
			},
			wantLocal: "5",
			wantCloud: "5",
		},
		{
			name: "only local backups",
			info: CloudInfo{
				Provider:   "Dropbox",
				Connected:  false,
				LastSync:   "never",
				LocalCount: 3,
				CloudCount: 0,
			},
			wantLocal: "3",
			wantCloud: "0",
		},
		{
			name: "disconnected provider",
			info: CloudInfo{
				Provider:   "OneDrive",
				Connected:  false,
				LastSync:   "—",
				LocalCount: 0,
				CloudCount: 0,
			},
			wantLocal: "0",
			wantCloud: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := RenderCloudStatus(tt.info, 80)

			if !strings.Contains(output, tt.wantLocal) {
				t.Errorf("RenderCloudStatus missing local count %q:\n%s", tt.wantLocal, output)
			}
			if !strings.Contains(output, tt.wantCloud) {
				t.Errorf("RenderCloudStatus missing cloud count %q:\n%s", tt.wantCloud, output)
			}
		})
	}
}

// =============================================================================
// TestRenderCloudStatus_NarrowTerminal — RED
// =============================================================================

func TestRenderCloudStatus_NarrowTerminal(t *testing.T) {
	info := CloudInfo{
		Provider:  "Google Drive",
		Connected: true,
		LastSync:  "2024-06-09",
	}

	output := RenderCloudStatus(info, 30)

	// Must not panic and must produce output even on narrow terminal.
	if len(output) == 0 {
		t.Fatal("RenderCloudStatus narrow terminal returned empty string")
	}

	// Core data should still be present.
	if !strings.Contains(output, "Google Drive") {
		t.Error("RenderCloudStatus narrow missing provider name")
	}
}

// =============================================================================
// TestRenderCloudStatus_DisconnectedProvider — RED
// =============================================================================

func TestRenderCloudStatus_DisconnectedProvider(t *testing.T) {
	info := CloudInfo{
		Provider:  "GitHub",
		Connected: false,
		LastSync:  "never",
	}

	output := RenderCloudStatus(info, 80)

	// Disconnected status should be visible.
	if !strings.Contains(output, "Disconnected") && !strings.Contains(output, "✗") {
		t.Errorf("RenderCloudStatus disconnected missing indicator: %q", output)
	}
}

// =============================================================================
// TestRenderCloudStatus_HelpBar — RED (Phase 3: help bar persistence)
// =============================================================================

func TestRenderCloudStatus_HelpBar(t *testing.T) {
	info := CloudInfo{
		Provider:   "Google Drive",
		Connected:  true,
		LastSync:   "2024-06-09 14:30 UTC",
		LocalCount: 12,
		CloudCount: 10,
	}

	output := RenderCloudStatus(info, 80)

	// Cloud help bar: q back
	if !strings.Contains(output, "back") {
		t.Errorf("cloud help bar missing 'back': %q", output)
	}
}

// TestRenderCloudStatus_HelpBar_NoProvider verifies the help bar appears
// even when no cloud provider is configured (triangulation).
func TestRenderCloudStatus_HelpBar_NoProvider(t *testing.T) {
	info := CloudInfo{
		Provider: "",
	}

	output := RenderCloudStatus(info, 80)

	// Must show the no-provider message...
	if !strings.Contains(output, "No cloud provider configured") {
		t.Errorf("no-provider cloud missing message: %q", output)
	}
	// ...AND the help bar.
	if !strings.Contains(output, "back") {
		t.Errorf("no-provider cloud help bar missing 'back': %q", output)
	}
}
