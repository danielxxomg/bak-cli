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
