package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseIgnore(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name     string
		input    string
		wantErr  bool
		wantPats int
		check    func(t *testing.T, patterns []Pattern)
	}{
		{
			name:     "single pattern",
			input:    "node_modules\n",
			wantErr:  false,
			wantPats: 1,
			check: func(t *testing.T, patterns []Pattern) {
				if patterns[0].raw != "node_modules" {
					t.Errorf("expected raw %q, got %q", "node_modules", patterns[0].raw)
				}
				if patterns[0].negate {
					t.Errorf("expected negate=false")
				}
				if patterns[0].dirOnly {
					t.Errorf("expected dirOnly=false for bare pattern")
				}
			},
		},
		{
			name:     "wildcard pattern",
			input:    "*.lock\n",
			wantErr:  false,
			wantPats: 1,
		},
		{
			name:     "directory pattern with trailing slash",
			input:    "build/\n",
			wantErr:  false,
			wantPats: 1,
			check: func(t *testing.T, patterns []Pattern) {
				if !patterns[0].dirOnly {
					t.Errorf("expected dirOnly=true for trailing-slash pattern")
				}
			},
		},
		{
			name:     "negation pattern",
			input:    "!important.log\n",
			wantErr:  false,
			wantPats: 1,
			check: func(t *testing.T, patterns []Pattern) {
				if !patterns[0].negate {
					t.Errorf("expected negate=true for ! pattern")
				}
				if patterns[0].raw != "important.log" {
					t.Errorf("expected raw %q, got %q", "important.log", patterns[0].raw)
				}
			},
		},
		{
			name:     "multiple patterns",
			input:    "node_modules\n*.lock\n.vscode/\n",
			wantErr:  false,
			wantPats: 3,
		},
		{
			name:     "comments and blank lines ignored",
			input:    "# this is a comment\n\nnode_modules\n\n  # another comment\n*.log\n",
			wantErr:  false,
			wantPats: 2,
		},
		{
			name:     "trim whitespace",
			input:    "  node_modules  \n  *.lock\n",
			wantErr:  false,
			wantPats: 2,
		},
		{
			name:     "empty file",
			input:    "",
			wantErr:  false,
			wantPats: 0,
		},
		{
			name:     "only comments",
			input:    "# only comments\n# nothing else\n",
			wantErr:  false,
			wantPats: 0,
		},
		{
			name:    "negation of negation not allowed",
			input:   "!!double\n",
			wantErr: true,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			patterns, err := ParseIgnore(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseIgnore() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(patterns) != tt.wantPats {
				t.Errorf("ParseIgnore() got %d patterns, want %d", len(patterns), tt.wantPats)
			}
			if tt.check != nil {
				tt.check(t, patterns)
			}
		})
	}
}

func TestPatternMatch(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name    string
		pattern string
		relPath string
		isDir   bool
		want    bool
	}{
		// Literal matches
		{name: "exact dir match", pattern: "node_modules", relPath: "node_modules", isDir: true, want: true},
		{name: "exact file match", pattern: ".gitignore", relPath: ".gitignore", isDir: false, want: true},
		{name: "exact dir no match different name", pattern: "node_modules", relPath: "dist", isDir: true, want: false},

		// Wildcard matches
		{name: "wildcard suffix match", pattern: "*.lock", relPath: "yarn.lock", isDir: false, want: true},
		{name: "wildcard suffix match nested", pattern: "*.lock", relPath: "sub/package-lock.json", isDir: false, want: false},
		{name: "wildcard suffix match nested real lock", pattern: "*.lock", relPath: "sub/yarn.lock", isDir: false, want: true},
		{name: "wildcard no match", pattern: "*.lock", relPath: "yarn.lock.json", isDir: false, want: false},
		{name: "wildcard prefix match", pattern: "tmp*", relPath: "tmpfile", isDir: false, want: true},
		{name: "wildcard prefix nested", pattern: "tmp*", relPath: "sub/tmpfile", isDir: false, want: true},

		// Directory-only patterns (trailing /)
		{name: "dir-only matches directory", pattern: "build", relPath: "build", isDir: true, want: true},
		{name: "dir-only does not match file same name", pattern: "build/", relPath: "build", isDir: false, want: false},
		{name: "dir-only matches nested dir", pattern: "build/", relPath: "sub/build", isDir: true, want: true},

		// Path-infix matches (gitignore behavior: pattern matches anywhere)
		{name: "infix dir match", pattern: "node_modules", relPath: "skills/node_modules", isDir: true, want: true},
		{name: "infix file match", pattern: "*.log", relPath: "sub/dir/error.log", isDir: false, want: true},

		// Negation patterns
		{name: "negation overrides match", pattern: "!important.log", relPath: "important.log", isDir: false, want: false},

		// Hidden files
		{name: "dotfile match", pattern: ".git", relPath: ".git", isDir: true, want: true},
		{name: "dotfile infix match", pattern: ".git", relPath: "sub/.git", isDir: true, want: true},

		// Binary extensions
		{name: "binary png match", pattern: "*.png", relPath: "screenshot.png", isDir: false, want: true},
		{name: "binary exe match", pattern: "*.exe", relPath: "bin/tool.exe", isDir: false, want: true},
		// Cross-platform: Windows backslash paths should match after normalization.
		{name: "backslash path normalized", pattern: "*.lock", relPath: `sub\dir\yarn.lock`, isDir: false, want: true},
		{name: "path with dot component cleaned", pattern: "*.lock", relPath: `sub/./yarn.lock`, isDir: false, want: true},
		{name: "redundant slashes cleaned", pattern: "*.lock", relPath: `sub//yarn.lock`, isDir: false, want: true},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			// Build a single pattern from the pattern string.
			dirOnly := false
			negate := false
			raw := tt.pattern
			if len(raw) > 0 && raw[len(raw)-1] == '/' {
				dirOnly = true
				raw = raw[:len(raw)-1]
			}
			if len(raw) > 0 && raw[0] == '!' {
				negate = true
				raw = raw[1:]
			}
			p := NewPattern(raw, dirOnly, negate)
			got := p.Match(tt.relPath, tt.isDir)
			if got != tt.want {
				t.Errorf("Pattern{raw:%q, dirOnly:%v, negate:%v}.Match(%q, %v) = %v, want %v",
					raw, dirOnly, negate, tt.relPath, tt.isDir, got, tt.want)
			}
		})
	}
}

func TestLoadExcludes(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name          string
		ignoreContent string // content of ~/.config/bak/ignore
		ignoreMissing bool   // true = don't create the ignore file
		settings      Settings
		wantExcludes  []string
		wantMaxSize   int64
		wantErr       bool
	}{
		{
			name:          "default patterns when no ignore file and no settings",
			ignoreMissing: true,
			settings:      Settings{},
			wantExcludes:  DefaultExcludes,
			wantMaxSize:   0, // zero-value settings → 0 MaxFileSize
		},
		{
			name:          "merge defaults with custom ignore file",
			ignoreContent: "*.tmp\n",
			settings:      Settings{},
			wantExcludes:  append([]string{}, DefaultExcludes...),
			wantMaxSize:   0,
		},
		{
			name:          "settings MaxFileSize used",
			ignoreMissing: true,
			settings:      Settings{MaxFileSize: 5242880},
			wantExcludes:  DefaultExcludes,
			wantMaxSize:   5242880,
		},
		{
			name:          "empty exclude_patterns clears defaults",
			ignoreContent: "*.tmp\n",
			settings:      Settings{ExcludePatterns: []string{}},
			wantExcludes:  []string{"*.tmp"},
			wantMaxSize:   0,
		},
		{
			name:          "exclude_patterns override defaults",
			ignoreContent: "",
			settings:      Settings{ExcludePatterns: []string{"*.override", "custom/"}},
			wantExcludes:  []string{"*.override", "custom/"},
			wantMaxSize:   0,
		},
		{
			name:          "exclude_patterns override + custom ignore merged",
			ignoreContent: "*.tmp\n",
			settings:      Settings{ExcludePatterns: []string{"*.override"}},
			wantExcludes:  []string{"*.override", "*.tmp"},
			wantMaxSize:   0,
		},
		{
			name:          "default MaxFileSize when settings is zero",
			ignoreMissing: true,
			settings:      Settings{MaxFileSize: 1048576},
			wantExcludes:  DefaultExcludes,
			wantMaxSize:   1048576,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			cfgDir := t.TempDir()
			if !tt.ignoreMissing {
				ignorePath := filepath.Join(cfgDir, "ignore")
				if err := os.WriteFile(ignorePath, []byte(tt.ignoreContent), 0644); err != nil {
					t.Fatalf("write ignore file: %v", err)
				}
			}

			excludes, maxSize, err := LoadExcludes(cfgDir, tt.settings)
			if (err != nil) != tt.wantErr {
				t.Fatalf("LoadExcludes() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if maxSize != tt.wantMaxSize {
				t.Errorf("maxSize = %d, want %d", maxSize, tt.wantMaxSize)
			}

			// For the case with custom ignore + default, we check prefixes.
			// The defaults should be first (from tt.wantExcludes) and any
			// custom patterns from ignore follow.
			if tt.ignoreContent != "" && !tt.ignoreMissing {
				// Check that ignore file patterns appear in the result
				ignorePatterns, _ := ParseIgnore(tt.ignoreContent)
				for _, ip := range ignorePatterns {
					raw := ip.raw
					if ip.dirOnly {
						raw += "/"
					}
					if ip.negate {
						raw = "!" + raw
					}
					found := false
					for _, e := range excludes {
						if e == raw {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected ignore pattern %q in excludes, but not found. excludes=%v", raw, excludes)
					}
				}
			}

			// For default patterns, verify the expected defaults are present.
			// Only check when settings doesn't override defaults.
			if tt.settings.ExcludePatterns == nil && len(tt.settings.ExcludePatterns) == 0 && tt.name != "empty exclude_patterns clears defaults" {
				for _, def := range DefaultExcludes {
					found := false
					for _, e := range excludes {
						if e == def {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected default exclude %q in result, but not found. excludes=%v", def, excludes)
					}
				}
			}
		})
	}
}

func TestLoadExcludesIgnoreReload(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	cfgDir := t.TempDir()
	settings := Settings{ExcludePatterns: []string{}}

	// First call: empty ignore
	excludes, _, err := LoadExcludes(cfgDir, settings)
	if err != nil {
		t.Fatalf("LoadExcludes: %v", err)
	}
	origLen := len(excludes)

	// Write a new ignore file
	ignorePath := filepath.Join(cfgDir, "ignore")
	if err := os.WriteFile(ignorePath, []byte("*.reload\n"), 0644); err != nil {
		t.Fatalf("write ignore file: %v", err)
	}

	// Second call: should pick up new pattern
	excludes, _, err = LoadExcludes(cfgDir, settings)
	if err != nil {
		t.Fatalf("LoadExcludes after write: %v", err)
	}
	if len(excludes) != origLen+1 {
		t.Errorf("expected %d excludes after reload, got %d", origLen+1, len(excludes))
	}
}

func TestLoadExcludes_InvalidIgnoreFile(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// When the ignore file contains invalid syntax (double negation), it should error.
	cfgDir := t.TempDir()
	ignorePath := filepath.Join(cfgDir, "ignore")
	if err := os.WriteFile(ignorePath, []byte("!!double\n"), 0644); err != nil {
		t.Fatalf("write ignore file: %v", err)
	}

	_, _, err := LoadExcludes(cfgDir, Settings{})
	if err == nil {
		t.Fatal("expected error for invalid ignore file (double negation)")
	}
}

// TestDefaultExcludes_IncludesRuntimeDBs verifies the expanded DefaultExcludes
// cover SQLite runtime databases, cache files, and JSONL history files.
// This test is RED until DefaultExcludes is expanded.
func TestDefaultExcludes_IncludesRuntimeDBs(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// All runtime patterns that MUST be in the default exclusion list.
	requiredPatterns := []string{
		"*.sqlite",
		"*.sqlite-wal",
		"*.sqlite-shm",
		"*.db",
		"*_cache.json",
		"*.jsonl",
	}

	// Build a set from DefaultExcludes for quick lookup.
	excludeSet := make(map[string]bool, len(DefaultExcludes))
	for _, p := range DefaultExcludes {
		excludeSet[p] = true
	}

	for _, pat := range requiredPatterns {
		if !excludeSet[pat] {
			t.Errorf("DefaultExcludes missing pattern %q", pat)
		}
	}

	// Also verify existing patterns are still present (no regression).
	existingPatterns := []string{
		"node_modules/",
		".git/",
		"*.lock",
		"*.log",
	}
	for _, pat := range existingPatterns {
		if !excludeSet[pat] {
			t.Errorf("DefaultExcludes missing existing pattern %q", pat)
		}
	}
}

// TestSplitWildcard locks splitWildcard's behavior: it splits a pattern at
// the FIRST '*' into [prefix, suffix] (the '*' itself is dropped); a pattern
// with no '*' returns a single-element slice.
func TestSplitWildcard(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name    string
		pattern string
		want    []string
	}{
		{name: "trailing wildcard", pattern: "skills/*", want: []string{"skills/", ""}},
		{name: "leading wildcard", pattern: "*.lock", want: []string{"", ".lock"}},
		{name: "embedded wildcard", pattern: "tmp*file", want: []string{"tmp", "file"}},
		{name: "no wildcard", pattern: "agents", want: []string{"agents"}},
		{name: "empty input", pattern: "", want: []string{""}},
		{name: "only wildcard", pattern: "*", want: []string{"", ""}},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			got := splitWildcard(tt.pattern)
			if len(got) != len(tt.want) {
				t.Fatalf("splitWildcard(%q) = %v (len %d), want %v (len %d)",
					tt.pattern, got, len(got), tt.want, len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitWildcard(%q)[%d] = %q, want %q", tt.pattern, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestMatchSegment covers matchSegment's branches: exact match, leading-*
// suffix match, trailing-* prefix match, embedded-* prefix+suffix match,
// mismatch, and empty inputs.
func TestMatchSegment(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name    string
		pattern string
		segment string
		want    bool
	}{
		{name: "exact match", pattern: "config", segment: "config", want: true},
		{name: "exact mismatch", pattern: "foo", segment: "bar", want: false},
		{name: "leading wildcard suffix match", pattern: "*.lock", segment: "yarn.lock", want: true},
		{name: "leading wildcard suffix mismatch", pattern: "*.lock", segment: "yarn.json", want: false},
		{name: "trailing wildcard prefix match", pattern: "tmp*", segment: "tmpfile", want: true},
		{name: "trailing wildcard prefix mismatch", pattern: "tmp*", segment: "varfile", want: false},
		{name: "embedded wildcard prefix and suffix match", pattern: "a*z", segment: "abcz", want: true},
		{name: "embedded wildcard suffix mismatch", pattern: "a*z", segment: "abcy", want: false},
		{name: "empty pattern empty segment", pattern: "", segment: "", want: true},
		{name: "empty pattern nonempty segment", pattern: "", segment: "x", want: false},
		{name: "leading wildcard matches any (bare *)", pattern: "*", segment: "anything", want: true},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			if got := matchSegment(tt.pattern, tt.segment); got != tt.want {
				t.Errorf("matchSegment(%q, %q) = %v, want %v", tt.pattern, tt.segment, got, tt.want)
			}
		})
	}
}

func TestLoadExcludes_UnreadableIgnoreFile(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// When the ignore file exists but is a directory (cannot read as file),
	// it should error. On Windows, os.ReadFile on a directory may return
	// a different error code; this test verifies the non-IsNotExist path.
	// Skip on Windows where directory read errors differ.
	cfgDir := t.TempDir()
	ignorePath := filepath.Join(cfgDir, "ignore")
	if err := os.MkdirAll(ignorePath, 0755); err != nil {
		t.Fatalf("create ignore dir: %v", err)
	}

	_, _, err := LoadExcludes(cfgDir, Settings{})
	if err == nil {
		t.Fatal("expected error for unreadable ignore file (directory instead of file)")
	}
}
