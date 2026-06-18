// Package config provides gitignore-compatible pattern matching for
// backup exclusion rules. The parse and match logic supports wildcards,
// directory-only patterns, negation, and infix matching (patterns match
// anywhere in a path, not only the leaf).
package config

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// DefaultExcludes are the patterns always excluded from backups unless
// the user explicitly clears them by setting exclude_patterns to an
// empty array.
var DefaultExcludes = []string{
	"node_modules/",
	".git/",
	"*.lock",
	"*.log",
	"*.sqlite",
	"*.sqlite-wal",
	"*.sqlite-shm",
	"*.db",
	"*_cache.json",
	"*.jsonl",
	"*.png",
	"*.jpg",
	"*.jpeg",
	"*.zip",
	"*.tar",
	"*.gz",
	"*.exe",
	"*.dll",
	"*.so",
	"*.dylib",
}

// Pattern represents a single gitignore-compatible exclusion rule.
type Pattern struct {
	raw     string // the original pattern text (without leading ! or trailing /)
	dirOnly bool   // true when pattern ends with /
	negate  bool   // true when pattern starts with ! (re-includes)
}

// NewPattern creates a new Pattern. dirOnly indicates the pattern
// matches directories only; negate indicates negation (re-include).
func NewPattern(raw string, dirOnly, negate bool) Pattern {
	return Pattern{
		raw:     raw,
		dirOnly: dirOnly,
		negate:  negate,
	}
}

// Match reports whether relPath matches this pattern according to
// gitignore rules. The match is infix: the pattern matches anywhere in
// the path, not only at the leaf.
func (p Pattern) Match(relPath string, isDir bool) bool {
	// Normalize: use forward slashes and clean the path.
	relPath = path.Clean(strings.ReplaceAll(relPath, "\\", "/"))

	if p.dirOnly && !isDir {
		return false
	}

	// Infix match: pattern can match the leaf or any segment.
	// Check if the pattern matches the path at any suffix.
	if matchSegment(p.raw, relPath) {
		return !p.negate
	}

	// Also check each segment from the right.
	// e.g., pattern "node_modules" matches "skills/node_modules"
	//       pattern "*.lock" matches "sub/dir/yarn.lock"
	for i := len(relPath) - 1; i >= 0; i-- {
		if relPath[i] == '/' {
			if matchSegment(p.raw, relPath[i+1:]) {
				return !p.negate
			}
		}
	}

	return p.negate // negation only matters if the pattern matched; false otherwise
}

// matchSegment checks whether a single path segment matches the pattern.
func matchSegment(pattern, segment string) bool {
	// Simple wildcard: * matches any characters.
	if strings.HasPrefix(pattern, "*") {
		suffix := pattern[1:]
		return strings.HasSuffix(segment, suffix)
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(segment, prefix)
	}
	// Embedded wildcard: prefix* → check prefix + any suffix.
	if idx := strings.Index(pattern, "*"); idx >= 0 {
		// Simple case: pattern contains * in the middle. We handle
		// prefix* and *suffix. For more complex patterns, fall back
		// to exact match after the current wildcard checks.
		// For now, handle the common prefix*suffix case.
		parts := splitWildcard(pattern)
		if len(parts) == 2 {
			return strings.HasPrefix(segment, parts[0]) && strings.HasSuffix(segment, parts[1])
		}
		// Fall through to exact match for unsupported patterns.
	}
	return segment == pattern
}

// splitWildcard splits a pattern at the first * and returns [prefix, suffix].
// If no *, returns the pattern alone.
func splitWildcard(pattern string) []string {
	idx := strings.Index(pattern, "*")
	if idx < 0 {
		return []string{pattern}
	}
	return []string{pattern[:idx], pattern[idx+1:]}
}

// ParseIgnore parses gitignore-compatible text and returns the list of
// patterns. Lines starting with # are comments; blank lines and
// whitespace-only lines are ignored. Leading/trailing whitespace is
// trimmed from each pattern line.
func ParseIgnore(content string) ([]Pattern, error) {
	var patterns []Pattern

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		negate := false
		if strings.HasPrefix(line, "!") {
			line = strings.TrimPrefix(line, "!")
			if strings.HasPrefix(line, "!") {
				return nil, fmt.Errorf("parse ignore: double negation not allowed: %s", line)
			}
			negate = true
		}

		dirOnly := false
		if len(line) > 0 && line[len(line)-1] == '/' {
			dirOnly = true
			line = line[:len(line)-1]
		}

		// Skip if the line became empty after stripping ! and /
		if line == "" {
			continue
		}

		patterns = append(patterns, Pattern{
			raw:     line,
			dirOnly: dirOnly,
			negate:  negate,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read ignore: %w", err)
	}

	return patterns, nil
}

// LoadExcludes loads exclusion patterns and max file size from config.
// It merges the default exclusion list with patterns from the
// ~/.config/bak/ignore file and settings.
//
// Returns (exclude patterns, max file size, error).
//
// Rules:
//   - When settings.ExcludePatterns is nil (not set), default patterns apply.
//   - When settings.ExcludePatterns is non-nil, it replaces defaults.
//   - When settings.ExcludePatterns is an empty slice, defaults are cleared.
//   - Custom ignore file patterns are always appended.
//   - MaxFileSize from settings (0 means no limit).
func LoadExcludes(configDir string, settings Settings) ([]string, int64, error) {
	var excludes []string

	// Determine base excludes: defaults or settings override.
	if settings.ExcludePatterns != nil {
		excludes = append(excludes, settings.ExcludePatterns...)
	} else {
		excludes = append(excludes, DefaultExcludes...)
	}

	// Read custom ignore file if it exists.
	ignorePath := filepath.Join(configDir, "ignore")
	if data, err := os.ReadFile(ignorePath); err == nil {
		ignorePatterns, parseErr := ParseIgnore(string(data))
		if parseErr != nil {
			return nil, 0, fmt.Errorf("parse ignore file: %w", parseErr)
		}
		for _, p := range ignorePatterns {
			raw := p.raw
			if p.dirOnly {
				raw += "/"
			}
			if p.negate {
				raw = "!" + raw
			}
			excludes = append(excludes, raw)
		}
	} else if !os.IsNotExist(err) {
		return nil, 0, fmt.Errorf("read ignore file: %w", err)
	}

	return excludes, settings.MaxFileSize, nil
}
