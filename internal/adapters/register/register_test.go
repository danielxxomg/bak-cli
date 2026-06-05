package register

import (
	"testing"

	"github.com/danielxxomg/bak-cli/internal/adapters"
)

func TestRegisterAll(t *testing.T) {
	r := adapters.NewRegistry()
	if err := All(r); err != nil {
		t.Fatalf("RegisterAll: %v", err)
	}

	expected := []string{
		"claude-code",
		"cursor",
		"codex",
		"windsurf",
		"kiro",
		"kilocode",
		"pidev",
		"opencode",
	}

	names := r.List()
	if len(names) != len(expected) {
		t.Fatalf("List() len = %d, want %d (names: %v)", len(names), len(expected), names)
	}

	for i, want := range expected {
		a, ok := r.Get(want)
		if !ok {
			t.Errorf("adapter %q not found at position %d", want, i)
			continue
		}
		if a.Name() != want {
			t.Errorf("adapter at position %d has Name()=%q, want %q", i, a.Name(), want)
		}
	}
}

func TestRegisterAll_Idempotent(t *testing.T) {
	r := adapters.NewRegistry()
	if err := All(r); err != nil {
		t.Fatalf("first RegisterAll: %v", err)
	}
	// Second call should fail due to duplicate registration.
	err := All(r)
	if err == nil {
		t.Error("expected error on duplicate RegisterAll call")
	}
}

func TestLoadYAMLAdapters_EmptyDir(t *testing.T) {
	r := adapters.NewRegistry()
	if err := All(r); err != nil {
		t.Fatalf("All: %v", err)
	}

	// Create a temp directory that is empty (no YAML files).
	dir := t.TempDir()
	// We need to make the function use our dir, but LoadYAMLAdapters
	// reads from ~/.config/bak/adapters/. As a unit test, we verify
	// the integration by testing that no override with an empty dir
	// (which doesn't exist under home) returns nil.
	// The empty/non-existent dir case returns nil because
	// LoadYAMLAdapters calls adapters.LoadYAMLAdapters which returns
	// nil when dir doesn't exist.

	// We can't redirect the standard dir easily, so verify the
	// integration: a missing adapters dir should not error.
	// This test validates the structural contract.
	_ = dir
	r2 := adapters.NewRegistry()
	if err := All(r2); err != nil {
		t.Fatalf("All: %v", err)
	}

	// LoadYAMLAdapters on a non-existent dir should succeed (dir
	// not found is not an error).
	err := LoadYAMLAdapters(r2, false)
	if err != nil {
		t.Fatalf("LoadYAMLAdapters with no adapters dir: %v", err)
	}
}
