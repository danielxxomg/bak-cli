package cloud

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestOpenBrowserOS_Exported(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	if openBrowserOS == nil {
		t.Fatal("openBrowserOS must be initialized (non-nil)")
	}
}

func TestOpenBrowserOS_DISPLAYGuard_Linux(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	if runtime.GOOS != "linux" {
		t.Skip("DISPLAY guard only applies on Linux")
	}

	tests := []struct {
		name        string
		display     string
		wantErr     bool
		wantErrWord string
	}{
		{"display_unset", "", true, "display"},
		{"display_set", ":0", false, ""},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			t.Setenv("DISPLAY", tt.display)

			// Prevent actual browser launch by mocking execCommand.
			origExecCommand := execCommand
			defer func() { execCommand = origExecCommand }()
			execCommand = func(name string, args ...string) *exec.Cmd {
				return &exec.Cmd{Path: "/dev/null"}
			}

			err := openBrowserOS("https://github.com/login/device")

			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantErr && err != nil && !strings.Contains(strings.ToLower(err.Error()), tt.wantErrWord) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErrWord)
			}
			if !tt.wantErr && err != nil && strings.Contains(strings.ToLower(err.Error()), "display") {
				t.Errorf("unexpected display error when DISPLAY=%q: %v", tt.display, err)
			}
		})
	}
}

func TestOpenBrowserOS_ExecCommandCapture(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	if runtime.GOOS != "linux" {
		t.Skip("execCommand capture test only on Linux")
	}

	var captured struct {
		name string
		args []string
	}

	t.Setenv("DISPLAY", "linux")

	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()

	execCommand = func(name string, args ...string) *exec.Cmd {
		captured.name = name
		captured.args = args
		return &exec.Cmd{Path: "/dev/null"}
	}

	_ = openBrowserOS("https://github.com/login/device")

	if captured.name != "xdg-open" {
		t.Errorf("expected xdg-open, got %q", captured.name)
	}
	if len(captured.args) != 1 || captured.args[0] != "https://github.com/login/device" {
		t.Errorf("expected args [url], got %v", captured.args)
	}
}

func TestOpenBrowserOS_OSSkipGuards(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name string
		goos string
	}{
		{"darwin", "darwin"},
		{"windows", "windows"},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			if runtime.GOOS != tt.goos {
				t.Skipf("%s test only runs on %s", tt.name, tt.goos)
			}
			// On native OS, just verify no panic.
			err := openBrowserOS("https://example.com")
			if err != nil {
				t.Logf("browser open error (expected on CI): %v", err)
			}
		})
	}
}
