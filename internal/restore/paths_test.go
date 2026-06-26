package restore

import (
	"runtime"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/manifest"
)

func TestResolvePath(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	tests := []struct {
		name        string
		canonical   string
		homeDir     string
		want        string
		wantErr     bool
		errContains string
	}{
		// Happy path: Windows home, Unix-style canonical path
		{
			name:      "unix canonical to windows home",
			canonical: "~/.config/opencode/skills/my-skill/SKILL.md",
			homeDir:   `C:\Users\alice`,
			want:      `C:\Users\alice\.config\opencode\skills\my-skill\SKILL.md`,
		},
		// Happy path: Unix home, Unix canonical
		{
			name:      "unix canonical to unix home",
			canonical: "~/.config/opencode/skills/test/SKILL.md",
			homeDir:   "/home/alice",
			want:      "/home/alice/.config/opencode/skills/test/SKILL.md",
		},
		// Home directory itself (edge case)
		{
			name:      "home dir itself",
			canonical: "~/",
			homeDir:   "/home/alice",
			want:      "/home/alice",
		},
		// Direct file under home
		{
			name:      "file directly under home",
			canonical: "~/.bashrc",
			homeDir:   "/home/alice",
			want:      "/home/alice/.bashrc",
		},
		// Non-canonical absolute path under home (return as-is)
		{
			name:      "non-canonical path under home",
			canonical: "/home/alice/projects/config.json",
			homeDir:   "/home/alice",
			want:      "/home/alice/projects/config.json",
		},

		// Path traversal attack: ../../
		{
			name:        "path traversal via dot dot",
			canonical:   "~/../../etc/passwd",
			homeDir:     "/home/alice",
			wantErr:     true,
			errContains: "outside home directory",
		},
		// Path traversal: absolute path outside home
		{
			name:        "absolute path outside home",
			canonical:   "/etc/shadow",
			homeDir:     "/home/alice",
			wantErr:     true,
			errContains: "outside home directory",
		},
		// Path traversal: windows absolute outside home
		{
			name:        "windows path outside home",
			canonical:   `C:\Program Files\secret\config.json`,
			homeDir:     `C:\Users\alice`,
			wantErr:     true,
			errContains: "outside home directory",
		},
		// Path traversal: parent of home
		{
			name:        "parent of home directory",
			canonical:   "~/../../../",
			homeDir:     "/home/alice",
			wantErr:     true,
			errContains: "outside home directory",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			// Skip Windows-specific test cases on non-Windows platforms.
			if strings.Contains(tt.name, "windows") && runtime.GOOS != "windows" {
				t.Skip("skipping Windows-specific test on non-Windows platform")
			}
			got, err := ResolvePath(tt.canonical, tt.homeDir)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errContains)
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Fatalf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("ResolvePath(%q, %q) = %q, want %q", tt.canonical, tt.homeDir, got, tt.want)
			}
		})
	}
}

func TestResolveManifestPaths(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	// Use a Unix-style homeDir so the resolved paths use forward slashes.
	// This keeps tests deterministic regardless of host OS.
	homeDir := "/home/alice"

	t.Run("all paths resolve within home", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		m := &manifest.Manifest{
			Adapters: map[string]manifest.AdapterManifest{
				"opencode": {
					ConfigDir: "~/.config/opencode",
					Items: []manifest.Item{
						{SourcePath: "~/.config/opencode/skills/go/SKILL.md", BackupPath: "opencode/skills/go/SKILL.md"},
						{SourcePath: "~/.config/opencode/opencode.json", BackupPath: "opencode/opencode.json"},
					},
				},
			},
		}

		resolved, err := ResolveManifestPaths(m, homeDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := map[string]string{
			"~/.config/opencode/skills/go/SKILL.md": "/home/alice/.config/opencode/skills/go/SKILL.md",
			"~/.config/opencode/opencode.json":      "/home/alice/.config/opencode/opencode.json",
		}
		for src, expected := range want {
			got, ok := resolved[src]
			if !ok {
				t.Fatalf("missing key %q in resolved map", src)
			}
			if got != expected {
				t.Fatalf("resolved[%q] = %q, want %q", src, got, expected)
			}
		}
	})

	t.Run("path outside home is rejected", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		m := &manifest.Manifest{
			Adapters: map[string]manifest.AdapterManifest{
				"opencode": {
					ConfigDir: "~/.config/opencode",
					Items: []manifest.Item{
						{SourcePath: "~/.config/opencode/skills/go/SKILL.md", BackupPath: "opencode/skills/go/SKILL.md"},
						{SourcePath: "~/../../../etc/passwd", BackupPath: "opencode/etc/passwd"},
					},
				},
			},
		}

		_, err := ResolveManifestPaths(m, homeDir)
		if err == nil {
			t.Fatal("expected error for path outside home, got nil")
		}
		if !contains(err.Error(), "outside home directory") {
			t.Fatalf("expected error containing 'outside home directory', got %q", err.Error())
		}
	})

	t.Run("empty manifest returns empty map", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
		m := &manifest.Manifest{
			Adapters: map[string]manifest.AdapterManifest{},
		}
		resolved, err := ResolveManifestPaths(m, homeDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resolved) != 0 {
			t.Fatalf("expected empty map, got %d entries", len(resolved))
		}
	})
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
