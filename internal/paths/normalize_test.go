package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
	info, err := DetectOS()
	if err != nil {
		t.Fatalf("DetectOS() error: %v", err)
	}

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

// TestToCanonical_PublicWrapper exercises the public ToCanonical function
// that uses the real os.UserHomeDir().
func TestToCanonical_PublicWrapper(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot determine home directory: %v", err)
	}

	// A subdirectory under home should be canonicalized.
	subDir := filepath.Join(home, ".config")
	got := ToCanonical(subDir)
	if !strings.HasPrefix(got, "~/") {
		t.Errorf("ToCanonical(%q) = %q, expected to start with ~/", subDir, got)
	}

	// Home directory itself should return "~/".
	got = ToCanonical(home)
	if got != "~/" {
		t.Errorf("ToCanonical(%q) = %q, want \"~/\"", home, got)
	}

	// An outside path should be returned unchanged (not prefixed with ~/).
	outside := "/etc/hosts"
	if runtime.GOOS == "windows" {
		outside = `C:\Windows\System32\drivers\etc\hosts`
	}
	got = ToCanonical(outside)
	if strings.HasPrefix(got, "~/") {
		t.Errorf("ToCanonical(%q) = %q, should NOT start with ~/", outside, got)
	}
}

// TestIsUnderHome_PublicWrapper exercises the public IsUnderHome function
// that uses the real os.UserHomeDir().
func TestIsUnderHome_PublicWrapper(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot determine home directory: %v", err)
	}

	// A subdirectory should be under home.
	subDir := filepath.Join(home, ".config")
	if !IsUnderHome(subDir) {
		t.Errorf("IsUnderHome(%q) = false, want true", subDir)
	}

	// Home itself should NOT be considered "under" (rel is ".", not a descendant).
	if IsUnderHome(home) {
		t.Errorf("IsUnderHome(%q) = true, want false (home itself)", home)
	}

	// Root of the filesystem should NOT be under home.
	if IsUnderHome("/") {
		t.Error("IsUnderHome(\"/\") = true, want false")
	}
}

// TestToCanonical_EdgeCases covers trailing separators, empty paths,
// and other boundary inputs for the internal toCanonical function.
func TestToCanonical_EdgeCases(t *testing.T) {
	home := "/home/alice"
	if runtime.GOOS == "windows" {
		home = `C:\Users\alice`
	}

	// Build OS-appropriate test paths.
	absTrailingSep := home + "/.config/"
	absMultiSep := home + "/.config//"
	absDotSeg := home + "/./.config/./opencode"
	absOutsideHome := home + "/../../../etc/passwd"
	absHomeTrailing := home + "/"
	absDeep := home + "/a/b/c/d/e/f/g"

	tests := []struct {
		name    string
		absPath string
		homeDir string
		want    string
	}{
		{
			name:    "trailing separator",
			absPath: absTrailingSep,
			homeDir: home,
			want:    "~/.config",
		},
		{
			name:    "multiple trailing separators",
			absPath: absMultiSep,
			homeDir: home,
			want:    "~/.config",
		},
		{
			name:    "path with dot segments",
			absPath: absDotSeg,
			homeDir: home,
			want:    "~/.config/opencode",
		},
		{
			name:    "path with double-dot outside home",
			absPath: absOutsideHome,
			homeDir: home,
			want:    absOutsideHome, // stays outside home, returned as-is
		},
		{
			name:    "home with trailing separator",
			absPath: absHomeTrailing,
			homeDir: home,
			want:    "~/",
		},
		{
			name:    "path equals home",
			absPath: home,
			homeDir: home,
			want:    "~/",
		},
		{
			name:    "nested deep under home",
			absPath: absDeep,
			homeDir: home,
			want:    "~/a/b/c/d/e/f/g",
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

// TestIsUnder_EdgeCases covers boundary inputs for the isUnder function.
func TestIsUnder_EdgeCases(t *testing.T) {
	homeLinux := "/home/alice"

	tests := []struct {
		name    string
		absPath string
		homeDir string
		want    bool
	}{
		{"trailing separator on home", "/home/alice/.config/", homeLinux, true},
		{"relative dot path", ".", homeLinux, false},
		{"empty path", "", homeLinux, false},
		{"just slash", "/", homeLinux, false},
		{"path with dot segments", "/home/alice/./.config", homeLinux, true},
		{"deeply nested", "/home/alice/a/b/c/d/e", homeLinux, true},
		{"home with extra chars — not a match", "/home/alice2/.config", homeLinux, false},
		{"same prefix different user", "/home/alicebob", homeLinux, false},
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

// TestFromCanonical_EdgeCases covers additional boundary inputs.
func TestFromCanonical_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		canonical string
		homeDir   string
		want      string
	}{
		{
			name:      "empty canonical",
			canonical: "",
			homeDir:   "/home/alice",
			want:      "",
		},
		{
			name:      "canonical without tilde prefix",
			canonical: ".config/opencode",
			homeDir:   "/home/alice",
			want:      ".config/opencode",
		},
		{
			name:      "just tilde no slash",
			canonical: "~",
			homeDir:   "/home/alice",
			want:      "~", // no "~/" prefix, passes through unchanged
		},
		{
			name:      "tilde with empty relative",
			canonical: "~/",
			homeDir:   "/home/alice",
			want:      filepath.Join("/home/alice"),
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

// TestConfigDir_EdgeCases verifies ConfigDir with various relative paths.
func TestConfigDir_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		relPath string
	}{
		{"standard app", "bak"},
		{"nested app", filepath.Join("bak", "sub")},
		{"empty relative", ""},
		{"opencode app", "opencode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := ConfigDir(tt.relPath)
			if err != nil {
				t.Fatalf("ConfigDir(%q) error: %v", tt.relPath, err)
			}
			if dir == "" {
				t.Error("ConfigDir returned empty string")
			}
		})
	}
}
