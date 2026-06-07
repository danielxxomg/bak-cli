package diff

import (
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/manifest"
)

// makeManifest builds a minimal manifest with items keyed by adapter name.
func makeManifest(adapterName string, items []manifest.Item) *manifest.Manifest {
	return &manifest.Manifest{
		Version:    "0.3.0",
		ID:         "test",
		Adapters: map[string]manifest.AdapterManifest{
			adapterName: {Items: items},
		},
	}
}

// item is a helper to build a manifest.Item concisely.
func item(sourcePath, hash, adapter string) manifest.Item {
	return manifest.Item{
		SourcePath: sourcePath,
		Hash:       hash,
		BackupPath: strings.ToLower(adapter) + "/" + sourcePath,
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		name     string
		a        *manifest.Manifest
		b        *manifest.Manifest
		want     []DiffEntry
	}{
		{
			name: "added file",
			a:    makeManifest("opencode", nil),
			b: makeManifest("opencode", []manifest.Item{
				item("skills/new-skill.md", "sha256:abc123", "opencode"),
			}),
			want: []DiffEntry{
				{SourcePath: canonicalPath("skills/new-skill.md"), Category: CategoryAdded, Adapter: "opencode"},
			},
		},
		{
			name: "removed file",
			a: makeManifest("opencode", []manifest.Item{
				item("skills/old-skill.md", "sha256:abc123", "opencode"),
			}),
			b: makeManifest("opencode", nil),
			want: []DiffEntry{
				{SourcePath: canonicalPath("skills/old-skill.md"), Category: CategoryRemoved, Adapter: "opencode"},
			},
		},
		{
			name: "modified file",
			a: makeManifest("opencode", []manifest.Item{
				item("skills/config.json", "sha256:aaa111", "opencode"),
			}),
			b: makeManifest("opencode", []manifest.Item{
				item("skills/config.json", "sha256:bbb222", "opencode"),
			}),
			want: []DiffEntry{
				{SourcePath: canonicalPath("skills/config.json"), Category: CategoryModified, Adapter: "opencode"},
			},
		},
		{
			name: "unchanged file",
			a: makeManifest("opencode", []manifest.Item{
				item("skills/stable.md", "sha256:same", "opencode"),
			}),
			b: makeManifest("opencode", []manifest.Item{
				item("skills/stable.md", "sha256:same", "opencode"),
			}),
			want: []DiffEntry{
				{SourcePath: canonicalPath("skills/stable.md"), Category: CategoryUnchanged, Adapter: "opencode"},
			},
		},
		{
			name: "mixed categories",
			a: makeManifest("opencode", []manifest.Item{
				item("a/removed.md", "sha256:r1", "opencode"),
				item("b/modified.md", "sha256:old", "opencode"),
				item("c/unchanged.md", "sha256:same", "opencode"),
			}),
			b: makeManifest("opencode", []manifest.Item{
				item("b/modified.md", "sha256:new", "opencode"),
				item("c/unchanged.md", "sha256:same", "opencode"),
				item("d/added.md", "sha256:a1", "opencode"),
			}),
			want: []DiffEntry{
				{SourcePath: canonicalPath("a/removed.md"), Category: CategoryRemoved, Adapter: "opencode"},
				{SourcePath: canonicalPath("b/modified.md"), Category: CategoryModified, Adapter: "opencode"},
				{SourcePath: canonicalPath("c/unchanged.md"), Category: CategoryUnchanged, Adapter: "opencode"},
				{SourcePath: canonicalPath("d/added.md"), Category: CategoryAdded, Adapter: "opencode"},
			},
		},
		{
			name: "identical manifests",
			a: makeManifest("opencode", []manifest.Item{
				item("a.md", "sha256:h1", "opencode"),
				item("b.md", "sha256:h2", "opencode"),
			}),
			b: makeManifest("opencode", []manifest.Item{
				item("a.md", "sha256:h1", "opencode"),
				item("b.md", "sha256:h2", "opencode"),
			}),
			want: []DiffEntry{
				{SourcePath: canonicalPath("a.md"), Category: CategoryUnchanged, Adapter: "opencode"},
				{SourcePath: canonicalPath("b.md"), Category: CategoryUnchanged, Adapter: "opencode"},
			},
		},
		{
			name: "empty manifests",
			a:    makeManifest("opencode", nil),
			b:    makeManifest("opencode", nil),
			want: nil,
		},
		{
			name: "windows backslash paths normalized",
			a: makeManifest("opencode", []manifest.Item{
				{SourcePath: `skills\win-path.md`, Hash: "sha256:h1", BackupPath: "opencode/skills/win-path.md"},
			}),
			b: makeManifest("opencode", []manifest.Item{
				{SourcePath: "skills/win-path.md", Hash: "sha256:h1", BackupPath: "opencode/skills/win-path.md"},
			}),
			want: []DiffEntry{
				{SourcePath: canonicalPath("skills/win-path.md"), Category: CategoryUnchanged, Adapter: "opencode"},
			},
		},
		{
			name: "multiple adapters",
			a: makeManifest("opencode", []manifest.Item{
				item("skills/a.md", "sha256:h1", "opencode"),
			}),
			b: makeManifest("cursor", []manifest.Item{
				item("cursor/b.md", "sha256:h2", "cursor"),
			}),
			want: []DiffEntry{
				{SourcePath: canonicalPath("cursor/b.md"), Category: CategoryAdded, Adapter: "cursor"},
				{SourcePath: canonicalPath("skills/a.md"), Category: CategoryRemoved, Adapter: "opencode"},
			},
		},
		{
			name: "result sorted by source path",
			a: makeManifest("opencode", []manifest.Item{
				item("z.md", "sha256:z", "opencode"),
				item("a.md", "sha256:a", "opencode"),
			}),
			b: makeManifest("opencode", []manifest.Item{
				item("z.md", "sha256:z_new", "opencode"),
				item("a.md", "sha256:a", "opencode"),
			}),
			want: []DiffEntry{
				{SourcePath: canonicalPath("a.md"), Category: CategoryUnchanged, Adapter: "opencode"},
				{SourcePath: canonicalPath("z.md"), Category: CategoryModified, Adapter: "opencode"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip Windows-specific test cases on non-Windows platforms.
			if strings.Contains(tt.name, "windows") && runtime.GOOS != "windows" {
				t.Skip("skipping Windows-specific test on non-Windows platform")
			}
			got := Compare(tt.a, tt.b)

			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d\ngot:  %+v\nwant: %+v", len(got), len(tt.want), got, tt.want)
			}

			for i := range got {
				if got[i].SourcePath != tt.want[i].SourcePath {
					t.Errorf("[%d] SourcePath = %q, want %q", i, got[i].SourcePath, tt.want[i].SourcePath)
				}
				if got[i].Category != tt.want[i].Category {
					t.Errorf("[%d] Category = %q, want %q", i, got[i].Category, tt.want[i].Category)
				}
				if got[i].Adapter != tt.want[i].Adapter {
					t.Errorf("[%d] Adapter = %q, want %q", i, got[i].Adapter, tt.want[i].Adapter)
				}
			}
		})
	}
}

func TestCanonicalPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "already clean unix path",
			input: "skills/foo.md",
			want:  "skills/foo.md",
		},
		{
			name:  "windows backslash normalized",
			input: `skills\foo.md`,
			want:  "skills/foo.md",
		},
		{
			name:  "dot segments cleaned",
			input: "skills/./foo.md",
			want:  "skills/foo.md",
		},
		{
			name:  "double dot segments cleaned",
			input: "skills/a/../foo.md",
			want:  "skills/foo.md",
		},
		{
			name:  "trailing slash removed",
			input: "skills/",
			want:  "skills",
		},
		{
			name:  "windows absolute path",
			input: `C:\Users\alice\.config\opencode`,
			want:  "C:/Users/alice/.config/opencode",
		},
		{
			name:  "unix absolute path",
			input: "/home/alice/.config/opencode",
			want:  "/home/alice/.config/opencode",
		},
		{
			name:  "mixed slashes",
			input: `C:/Users\alice/.config\opencode`,
			want:  "C:/Users/alice/.config/opencode",
		},
		{
			name:  "relative dot-dot path",
			input: "../config",
			want:  "../config",
		},
		{
			name:  "empty string",
			input: "",
			want:  ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip Windows-specific test cases on non-Windows platforms.
			if strings.Contains(tt.name, "windows") && runtime.GOOS != "windows" {
				t.Skip("skipping Windows-specific test on non-Windows platform")
			}
			got := canonicalPath(tt.input)
			if got != tt.want {
				t.Errorf("canonicalPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCategoryConstants(t *testing.T) {
	// Verify all 4 categories are distinct non-empty strings.
	cats := []Category{CategoryAdded, CategoryRemoved, CategoryModified, CategoryUnchanged}
	seen := make(map[Category]bool)
	for _, c := range cats {
		if c == "" {
			t.Error("category must not be empty")
		}
		if seen[c] {
			t.Errorf("duplicate category: %q", c)
		}
		seen[c] = true
	}
	if len(seen) != 4 {
		t.Errorf("expected 4 distinct categories, got %d", len(seen))
	}
}

func TestCompareSortsIndependently(t *testing.T) {
	// Verify that Compare sorts its result, not the input manifests.
	a := makeManifest("adapter", []manifest.Item{
		item("z.md", "sha256:z", "adapter"),
		item("a.md", "sha256:a", "adapter"),
	})
	b := makeManifest("adapter", []manifest.Item{
		item("a.md", "sha256:a", "adapter"),
		item("z.md", "sha256:z_new", "adapter"),
	})

	got := Compare(a, b)

	if !sort.SliceIsSorted(got, func(i, j int) bool {
		return got[i].SourcePath < got[j].SourcePath
	}) {
		t.Errorf("result is not sorted by SourcePath: %+v", got)
	}
}
