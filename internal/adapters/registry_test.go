package adapters

import (
	"errors"
	"testing"
)

// mockAdapter implements Adapter for testing.
type mockAdapter struct {
	name      string
	installed bool
	configDir string
}

func (m *mockAdapter) Name() string { return m.name }
func (m *mockAdapter) Detect(homeDir string) (bool, string, error) {
	return m.installed, m.configDir, nil
}
func (m *mockAdapter) ListItems(homeDir string, categories []string) ([]Item, error) {
	return nil, nil
}
func (m *mockAdapter) Backup(homeDir, backupDir string, items []Item) error  { return nil }
func (m *mockAdapter) Restore(backupDir, homeDir string, items []Item) error { return nil }

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}

	err := r.Register(&mockAdapter{name: "opencode"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Duplicate registration should fail.
	err = r.Register(&mockAdapter{name: "opencode"})
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockAdapter{name: "opencode"})

	a, ok := r.Get("opencode")
	if !ok {
		t.Fatal("Get: adapter not found")
	}
	if a.Name() != "opencode" {
		t.Errorf("Name() = %q, want %q", a.Name(), "opencode")
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Error("Get: found adapter that was never registered")
	}
}

func TestRegistry_GetByName(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockAdapter{name: "opencode"})

	a, ok := r.GetByName("opencode")
	if !ok {
		t.Error("GetByName: adapter not found")
	}
	if a.Name() != "opencode" {
		t.Errorf("Name() = %q, want %q", a.Name(), "opencode")
	}
}

func TestRegistry_All(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockAdapter{name: "opencode"})
	r.Register(&mockAdapter{name: "claude-code"})

	all := r.All()
	if len(all) != 2 {
		t.Fatalf("All() len = %d, want 2", len(all))
	}

	names := make(map[string]bool)
	for _, a := range all {
		names[a.Name()] = true
	}
	if !names["opencode"] || !names["claude-code"] {
		t.Errorf("All() missing expected adapters: %v", names)
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockAdapter{name: "opencode"})
	r.Register(&mockAdapter{name: "claude-code"})

	names := r.List()
	if len(names) != 2 {
		t.Fatalf("List() len = %d, want 2", len(names))
	}
}

func TestRegistry_DetectAll(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockAdapter{name: "opencode", installed: true, configDir: "/home/user/.config/opencode"})
	r.Register(&mockAdapter{name: "claude-code", installed: false})
	r.Register(&mockAdapter{name: "codex", installed: true, configDir: "/home/user/.codex"})

	detected := r.DetectAll("/home/user")
	if len(detected) != 2 {
		t.Fatalf("DetectAll() len = %d, want 2", len(detected))
	}

	found := make(map[string]string)
	for _, d := range detected {
		found[d.Adapter.Name()] = d.ConfigDir
	}

	if found["opencode"] != "/home/user/.config/opencode" {
		t.Errorf("opencode configDir = %q", found["opencode"])
	}
	if found["codex"] != "/home/user/.codex" {
		t.Errorf("codex configDir = %q", found["codex"])
	}
	if _, exists := found["claude-code"]; exists {
		t.Error("claude-code should not appear in DetectAll (not installed)")
	}
}

// errorAdapter always returns an error from Detect.
type errorAdapter struct{ mockAdapter }

func (e *errorAdapter) Detect(homeDir string) (bool, string, error) {
	return false, "", errors.New("detection failed")
}

func TestRegistry_RegisterOrReplace(t *testing.T) {
	tests := []struct {
		name     string
		first    string
		second   string
		override bool
		wantErr  bool
		wantName string // name of adapter after operation
	}{
		{
			name:     "first registration succeeds",
			first:    "alpha",
			second:   "beta",
			override: false,
			wantErr:  false,
			wantName: "beta",
		},
		{
			name:     "duplicate without override fails",
			first:    "alpha",
			second:   "alpha",
			override: false,
			wantErr:  true,
			wantName: "alpha",
		},
		{
			name:     "duplicate with override succeeds",
			first:    "alpha-v1",
			second:   "alpha-v2",
			override: true,
			wantErr:  false,
			wantName: "alpha-v2",
		},
		{
			name:     "unique names, override true",
			first:    "gamma",
			second:   "delta",
			override: true,
			wantErr:  false,
			wantName: "delta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			if err := r.Register(&mockAdapter{name: tt.first}); err != nil {
				t.Fatalf("first register: %v", err)
			}

			err := r.RegisterOrReplace(&mockAdapter{name: tt.second}, tt.override)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				// Verify original still exists
				a, ok := r.Get(tt.first)
				if !ok || a.Name() != tt.first {
					t.Error("original adapter lost on failed replace")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			a, ok := r.Get(tt.second)
			if !ok {
				t.Fatalf("adapter %q not found after RegisterOrReplace", tt.second)
			}
			if a.Name() != tt.wantName {
				t.Errorf("Name() = %q, want %q", a.Name(), tt.wantName)
			}
		})
	}
}

func TestRegistry_DetectAll_GracefulError(t *testing.T) {
	r := NewRegistry()
	r.Register(&errorAdapter{mockAdapter{name: "broken"}})
	r.Register(&mockAdapter{name: "opencode", installed: true, configDir: "/home/user/.config/opencode"})

	detected := r.DetectAll("/home/user")
	if len(detected) != 1 {
		t.Fatalf("DetectAll() len = %d, want 1 (error adapter skipped)", len(detected))
	}
	if detected[0].Adapter.Name() != "opencode" {
		t.Errorf("expected opencode, got %q", detected[0].Adapter.Name())
	}
}
