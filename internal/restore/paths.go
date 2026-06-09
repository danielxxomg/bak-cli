// Package restore implements the restore engine for bak: resolving
// cross-platform paths, computing dry-run diffs, and safely applying
// backed-up configurations to the target system.
package restore

import (
	"fmt"
	"strings"

	"github.com/danielxxomg/bak-cli/internal/manifest"
	pathsutil "github.com/danielxxomg/bak-cli/internal/paths"
)

// ResolvePath converts a canonical source_path from a manifest into an
// absolute path on the target OS. It verifies that the resolved path
// stays under homeDir (security gate). Returns an error if the path
// would escape the home directory.
func ResolvePath(canonical, homeDir string) (string, error) {
	resolved := pathsutil.FromCanonical(canonical, homeDir)

	// Preserve the separator style of the target home directory so
	// cross-platform tests produce consistent output.
	if isUnixHome(homeDir) {
		resolved = pathsutil.Slash(resolved)
	}

	// Security: refuse paths outside the home directory.
	if !isUnderHomeDir(resolved, homeDir) {
		return "", fmt.Errorf("path resolves outside home directory: %s", resolved)
	}

	return resolved, nil
}

// ResolveManifestPaths resolves every item's source_path in the manifest
// to an absolute target-OS path. Returns a map from source_path to
// resolved absolute path, or an error if any path escapes home.
func ResolveManifestPaths(m *manifest.Manifest, homeDir string) (map[string]string, error) {
	resolved := make(map[string]string)

	for _, am := range m.Adapters {
		for _, item := range am.Items {
			abs, err := ResolvePath(item.SourcePath, homeDir)
			if err != nil {
				return nil, err
			}
			resolved[item.SourcePath] = abs
		}
	}

	return resolved, nil
}

// isUnderHomeDir returns true when absPath is equal to or a descendant
// of homeDir. Normalizes separators for cross-platform comparison.
func isUnderHomeDir(absPath, homeDir string) bool {
	// Normalize both to forward slashes for cross-platform safety.
	// Use path.Clean (not filepath.Clean) to avoid OS separator re-conversion.
	absClean := pathsutil.CanonicalPath(absPath)
	homeClean := pathsutil.CanonicalPath(homeDir)

	// Equal to home directory is allowed.
	if absClean == homeClean {
		return true
	}

	// Must be a descendant: homeDir + separator + ...
	prefix := homeClean + "/"
	return strings.HasPrefix(absClean, prefix)
}

// isUnixHome returns true when the home directory path uses forward
// slashes and no backslashes, indicating a Unix-style target.
func isUnixHome(homeDir string) bool {
	return strings.Contains(homeDir, "/") && !strings.Contains(homeDir, "\\")
}
