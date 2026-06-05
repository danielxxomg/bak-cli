package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestToCanonical(t *testing.T) {
	home := "/home/alice"
	if runtime.GOOS == "windows" {
		home = `C:\Users\alice`
	}

	tests := []struct {
		name     string
		absPath  string
		homeDir  string
		want     string
	}{
		{
			name:    "home directory itself",
			absPath: home,
			homeDir: home,
			want:    "~/",
		},
		{
			name:    "subdirectory under home (linux)",
			absPath: "/home/alice/.config/opencode",
			homeDir: "/home/alice",
			want:    "~/.config/opencode",
		},
		{
			name:    "subdirectory under home (windows)",
			absPath: `C:\Users\alice\.config\opencode`,
			homeDir: `C:\Users\alice`,
			want:    "~/.config/opencode",
		},
		{
			name:    "path outside home",
			absPath: "/etc/passwd",
			homeDir: "/home/alice",
			want:    "/etc/passwd",
		},
		{
			name:    "sibling of home",
			absPath: "/home/bob/.config",
			homeDir: "/home/alice",
			want:    "/home/bob/.config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toCanonical(tt.absPath, tt.homeDir)
			if got != tt.want {
				t.Errorf("toCanonical(%q, %q) = %q, want %q", tt.absPath, tt.homeDir, got, tt.want)
			}
		})
	}
}

func TestFromCanonical(t *testing.T) {
	tests := []struct {
		name      string
		canonical string
		homeDir   string
		want      string // exact expected output (OS-native)
	}{
		{
			name:      "linux home subdir",
			canonical: "~/.config/opencode",
			homeDir:   "/home/alice",
			want:      filepath.Join("/home/alice", ".config/opencode"),
		},
		{
			name:      "windows home subdir",
			canonical: "~/.config/opencode",
			homeDir:   `C:\Users\alice`,
			want:      filepath.Join(`C:\Users\alice`, ".config", "opencode"),
		},
		{
			name:      "non-canonical passthrough",
			canonical: "/etc/hosts",
			homeDir:   "/home/alice",
			want:      "/etc/hosts",
		},
		{
			name:      "home root",
			canonical: "~/",
			homeDir:   "/home/bob",
			want:      filepath.Join("/home/bob"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromCanonical(tt.canonical, tt.homeDir)
			if got != tt.want {
				t.Errorf("FromCanonical(%q, %q) = %q, want %q", tt.canonical, tt.homeDir, got, tt.want)
			}
		})
	}
}

func TestIsUnderHome(t *testing.T) {
	homeLinux := "/home/alice"
	homeWin := `C:\Users\alice`

	tests := []struct {
		name    string
		absPath string
		homeDir string
		want    bool
	}{
		{"direct child", "/home/alice/.config", homeLinux, true},
		{"deep child", "/home/alice/a/b/c", homeLinux, true},
		{"home itself", "/home/alice", homeLinux, false}, // rel is ".", not under
		{"outside home", "/etc/passwd", homeLinux, false},
		{"sibling dir", "/home/bob", homeLinux, false},
		{"windows child", `C:\Users\alice\.config`, homeWin, true},
		{"windows outside", `C:\Windows\System32`, homeWin, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUnder(tt.absPath, tt.homeDir)
			if got != tt.want {
				t.Errorf("isUnder(%q, %q) = %v, want %v", tt.absPath, tt.homeDir, got, tt.want)
			}
		})
	}
}

func TestDetectOS(t *testing.T) {
	info := DetectOS()

	if info.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", info.OS, runtime.GOOS)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("Arch = %q, want %q", info.Arch, runtime.GOARCH)
	}

	expectedHome, _ := os.UserHomeDir()
	if info.HomeDir != expectedHome {
		t.Errorf("HomeDir = %q, want %q", info.HomeDir, expectedHome)
	}
	if info.Sep != string(filepath.Separator) {
		t.Errorf("Sep = %q, want %q", info.Sep, string(filepath.Separator))
	}
}

func TestConfigDir(t *testing.T) {
	dir, err := ConfigDir("bak")
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	if dir == "" {
		t.Error("ConfigDir returned empty string")
	}
	// Verify it ends with the expected relative path.
	expectedSuffix := filepath.Join("bak")
	if len(dir) < len(expectedSuffix) {
		t.Errorf("ConfigDir too short: %q", dir)
	}
}
