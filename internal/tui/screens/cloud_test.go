// Package screens provides screen-specific render functions and sub-models
// for the bak-cli TUI. This file contains TDD tests written BEFORE the
// production code (strict RED phase) for the cloud status screen.
package screens

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
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
	if !strings.Contains(output, "Disconnected") && !strings.Contains(output, "\u2717") {
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

// =============================================================================
// CloudModel Tests — RED (CloudModel sub-model does not exist yet)
// =============================================================================

func TestCloudModel_New(t *testing.T) {
	statusFn := func() (CloudInfo, error) {
		return CloudInfo{
			Provider:  "github",
			Connected: true,
			LastSync:  "2024-06-09",
		}, nil
	}
	m := NewCloudModel(statusFn)

	if m.Width != 0 {
		t.Errorf("initial width = %d, want 0", m.Width)
	}
}

func TestCloudModel_Init(t *testing.T) {
	statusFn := func() (CloudInfo, error) {
		return CloudInfo{
			Provider:  "github",
			Connected: true,
			LastSync:  "2024-06-09",
		}, nil
	}
	m := NewCloudModel(statusFn)

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil, want cloud status load cmd")
	}

	msg := cmd()
	newModel, _ := m.Update(msg)
	cm := newModel.(CloudModel)

	if cm.Info.Provider != "github" {
		t.Errorf("Info.Provider = %q, want %q", cm.Info.Provider, "github")
	}
	if !cm.Info.Connected {
		t.Error("Info.Connected = false, want true")
	}
}

func TestCloudModel_View(t *testing.T) {
	statusFn := func() (CloudInfo, error) {
		return CloudInfo{
			Provider:   "github",
			Connected:  true,
			LastSync:   "2024-06-09",
			LocalCount: 5,
			CloudCount: 3,
		}, nil
	}
	m := NewCloudModel(statusFn)
	m.Width = 80
	m.Height = 24
	m.Info = CloudInfo{
		Provider:   "github",
		Connected:  true,
		LastSync:   "2024-06-09",
		LocalCount: 5,
		CloudCount: 3,
	}

	output := m.View().Content

	if !strings.Contains(output, "github") {
		t.Errorf("cloud view missing provider: %q", output)
	}
	if !strings.Contains(output, "Connected") {
		t.Errorf("cloud view missing status: %q", output)
	}
}

func TestCloudModel_WindowSize(t *testing.T) {
	m := NewCloudModel(nil)
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	cm := newModel.(CloudModel)

	if cm.Width != 100 {
		t.Errorf("Width = %d, want 100", cm.Width)
	}
	if cm.Height != 30 {
		t.Errorf("Height = %d, want 30", cm.Height)
	}
}

// =============================================================================
// Phase 3: CloudModel error and disconnect coverage
// =============================================================================

func TestCloudModel_Update_StatusError(t *testing.T) {
	statusFn := func() (CloudInfo, error) {
		return CloudInfo{}, fmt.Errorf("network unreachable")
	}
	m := NewCloudModel(statusFn)

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil")
	}
	msg := cmd()
	newModel, _ := m.Update(msg)
	cm := newModel.(CloudModel)

	if cm.Err == nil {
		t.Error("error should be set when statusFn returns error")
	}
	if cm.Err.Error() != "network unreachable" {
		t.Errorf("Err = %q, want network unreachable", cm.Err.Error())
	}
}

func TestCloudModel_View_Disconnected(t *testing.T) {
	m := NewCloudModel(nil)
	m.Width = 80
	m.Height = 24
	m.Info = CloudInfo{
		Provider:  "github",
		Connected: false,
		LastSync:  "never",
	}

	output := m.View().Content

	if !strings.Contains(output, "github") {
		t.Errorf("disconnected view missing provider: %q", output)
	}
	if !strings.Contains(output, "Disconnected") {
		t.Errorf("disconnected view missing status: %q", output)
	}
}

func TestCloudModel_View_InitState(t *testing.T) {
	// Before Init completes, Info is empty and no error — View shows "No provider".
	m := NewCloudModel(nil)
	m.Width = 80
	m.Height = 24

	output := m.View().Content

	if !strings.Contains(output, "No cloud provider configured") {
		t.Errorf("init state view missing no-provider: %q", output)
	}
}
