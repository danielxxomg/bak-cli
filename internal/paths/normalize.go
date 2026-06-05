// Package paths provides cross-platform path normalization and OS
// metadata for the backup/restore workflow.
//
// All paths in the manifest are stored in canonical form using a "~/"
// prefix that represents the user's home directory. On restore, the
// canonical prefix is resolved to the target OS home directory.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// OSInfo holds platform metadata for the current host.
type OSInfo struct {
	OS       string // "windows", "darwin", "linux"
	Arch     string // "amd64", "arm64"
	HomeDir  string // absolute path to user home
	Sep      string // path separator as string
}

// DetectOS returns platform metadata for the current host.
func DetectOS() OSInfo {
	home, _ := os.UserHomeDir()
	return OSInfo{
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
		HomeDir: home,
		Sep:     string(filepath.Separator),
	}
}

// ConfigDir returns the OS-specific config directory for the given
// relative path (e.g., "opencode" → ~/.config/opencode on Linux,
// %APPDATA%/opencode on Windows, ~/Library/Application Support/opencode on macOS).
// Falls back to the user home when os.UserConfigDir returns an error.
func ConfigDir(relPath string) (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return "", fmt.Errorf("cannot determine config directory: %w", err)
		}
		base = home
	}
	return filepath.Join(base, relPath), nil
}

// ToCanonical converts an absolute path to its canonical "~/" form.
//
//	"C:\Users\alice\.config\opencode" → "~/.config/opencode"
//	"/home/alice/.config/opencode"    → "~/.config/opencode"
//
// Returns the input unchanged if it does not reside under the home directory.
func ToCanonical(absPath string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return absPath
	}
	return toCanonical(absPath, home)
}

// toCanonical is the testable core that accepts an explicit homeDir.
func toCanonical(absPath, homeDir string) string {
	// Clean both paths to remove trailing separators, dots, etc.
	cleanedAbs := filepath.Clean(absPath)
	cleanedHome := filepath.Clean(homeDir)

	// Case-insensitive prefix check for Windows.
	if strings.EqualFold(cleanedAbs, cleanedHome) {
		return "~/"
	}

	// Walk up from absPath to check if it starts with homeDir.
	rel, err := filepath.Rel(cleanedHome, cleanedAbs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return absPath
	}
	return "~/" + filepath.ToSlash(rel)
}

// FromCanonical resolves a canonical "~/" path back to an absolute
// path on the target system using the provided home directory.
//
//	"~/.config/opencode" + "/home/bob" → "/home/bob/.config/opencode"
func FromCanonical(canonical, homeDir string) string {
	if !strings.HasPrefix(canonical, "~/") {
		return canonical
	}
	rel := strings.TrimPrefix(canonical, "~/")
	return filepath.Join(homeDir, filepath.FromSlash(rel))
}

// IsUnderHome returns true when the resolved absolute path is a
// descendant of (or equal to) the home directory. Used as a security
// guard during restore to prevent writing outside the user's home.
func IsUnderHome(absPath string) bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	return isUnder(absPath, home)
}

// isUnder is the testable core with explicit homeDir.
func isUnder(absPath, homeDir string) bool {
	cleanedAbs := filepath.Clean(absPath)
	cleanedHome := filepath.Clean(homeDir)

	rel, err := filepath.Rel(cleanedHome, cleanedAbs)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..") && rel != "."
}
