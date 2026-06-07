package presets

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name      string
		preset    string
		wantCats  []string
		wantErr   bool
	}{
		{
			name:     "quick preset",
			preset:   Quick,
			wantCats: []string{CatConfig},
		},
		{
			name:     "full preset",
			preset:   Full,
			wantCats: AllCategories,
		},
		{
			name:     "skills preset",
			preset:   Skills,
			wantCats: []string{CatSkills},
		},
		{
			name:    "unknown preset",
			preset:  "bananas",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.preset)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, tt.wantCats) {
				t.Errorf("categories = %v, want %v", got, tt.wantCats)
			}
		})
	}
}

func TestResolve_ReturnsCopy(t *testing.T) {
	cats1, _ := Resolve(Quick)
	cats2, _ := Resolve(Quick)

	// Mutate one, verify the other is unchanged.
	cats1[0] = "corrupted"

	if cats2[0] != CatConfig {
		t.Errorf("Resolve did not return a copy: cats2[0] = %q after mutating cats1", cats2[0])
	}
}

func TestNames(t *testing.T) {
	names := Names()
	if len(names) != 3 {
		t.Errorf("Names() len = %d, want 3", len(names))
	}
	expected := []string{"quick", "full", "skills"}
	if !slices.Equal(names, expected) {
		t.Errorf("Names() = %v, want %v", names, expected)
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{Quick, true},
		{Full, true},
		{Skills, true},
		{"all", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValid(tt.name)
			if got != tt.want {
				t.Errorf("IsValid(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestResolveAll_BuiltinFallback(t *testing.T) {
	tests := []struct {
		name     string
		preset   string
		wantCats []string
		wantErr  bool
	}{
		{
			name:     "quick preset (no yaml override)",
			preset:   Quick,
			wantCats: []string{CatConfig},
		},
		{
			name:     "full preset (no yaml override)",
			preset:   Full,
			wantCats: AllCategories,
		},
		{
			name:     "skills preset (no yaml override)",
			preset:   Skills,
			wantCats: []string{CatSkills},
		},
		{
			name:    "unknown preset",
			preset:  "nonexistent_xyz",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveAll(tt.preset, false)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, tt.wantCats) {
				t.Errorf("categories = %v, want %v", got, tt.wantCats)
			}
		})
	}
}

func TestResolveAll_ReturnsCopy(t *testing.T) {
	cats1, _ := ResolveAll(Quick, false)
	cats2, _ := ResolveAll(Quick, false)

	cats1[0] = "corrupted"
	if cats2[0] != CatConfig {
		t.Errorf("ResolveAll did not return a copy: cats2[0] = %q after mutating cats1", cats2[0])
	}
}

func TestResolveAll_ConflictWarning(t *testing.T) {
	// When a YAML preset name conflicts with a built-in and override=false,
	// ResolveAll should print a warning to stderr and fall back to the built-in.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	presetsDir := filepath.Join(home, ".config", "bak", "presets")
	if err := os.MkdirAll(presetsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write a YAML preset that conflicts with built-in "quick".
	yamlContent := []byte(`name: quick
categories:
  - skills
  - commands
`)
	presetFile := filepath.Join(presetsDir, "my-preset.yaml")
	if err := os.WriteFile(presetFile, yamlContent, 0644); err != nil {
		t.Fatal(err)
	}

	// Capture stderr.
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Without override, should fall back to built-in "quick" (CatConfig).
	cats, err := ResolveAll(Quick, false)

	// Restore stderr.
	w.Close()
	os.Stderr = oldStderr

	var stderrBuf bytes.Buffer
	io.Copy(&stderrBuf, r)

	if err != nil {
		t.Fatalf("ResolveAll should not error on conflict (should warn), got: %v", err)
	}
	if !slices.Equal(cats, []string{CatConfig}) {
		t.Errorf("categories = %v, want %v (built-in quick)", cats, []string{CatConfig})
	}
	stderrStr := stderrBuf.String()
	if !strings.Contains(stderrStr, "warning") {
		t.Errorf("stderr should contain 'warning', got: %q", stderrStr)
	}
	if !strings.Contains(stderrStr, "quick") {
		t.Errorf("stderr should mention 'quick', got: %q", stderrStr)
	}
}

func TestResolveAll_ConflictOverride(t *testing.T) {
	// When override=true, the YAML version wins.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	presetsDir := filepath.Join(home, ".config", "bak", "presets")
	if err := os.MkdirAll(presetsDir, 0755); err != nil {
		t.Fatal(err)
	}

	yamlContent := []byte(`name: quick
categories:
  - skills
  - commands
`)
	presetFile := filepath.Join(presetsDir, "my-preset.yaml")
	if err := os.WriteFile(presetFile, yamlContent, 0644); err != nil {
		t.Fatal(err)
	}

	cats, err := ResolveAll(Quick, true)
	if err != nil {
		t.Fatalf("ResolveAll with override: %v", err)
	}
	if !slices.Equal(cats, []string{CatSkills, CatCommands}) {
		t.Errorf("categories = %v, want [CatSkills, CatCommands] (yaml override)", cats)
	}
}
