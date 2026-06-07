package register

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/adapters"
)

func TestRegisterAll(t *testing.T) {
	r := adapters.NewRegistry()
	if err := All(r); err != nil {
		t.Fatalf("All: %v", err)
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

	for _, want := range expected {
		a, ok := r.Get(want)
		if !ok {
			t.Errorf("adapter %q not found", want)
			continue
		}
		if a.Name() != want {
			t.Errorf("adapter %q has Name()=%q, want %q", want, a.Name(), want)
		}
	}
}

func TestRegisterAll_Idempotent(t *testing.T) {
	r := adapters.NewRegistry()
	if err := All(r); err != nil {
		t.Fatalf("first All: %v", err)
	}
	// Second call should fail due to duplicate registration.
	err := All(r)
	if err == nil {
		t.Error("expected error on duplicate All call")
	}
}

func TestLoadYAMLAdapters_EmptyDir(t *testing.T) {
	homeDir := t.TempDir()

	r := adapters.NewRegistry()
	if err := All(r); err != nil {
		t.Fatalf("All: %v", err)
	}

	// LoadYAMLAdapters on a non-existent dir should succeed (dir
	// not found is not an error).
	err := LoadYAMLAdapters(r, false, homeDir)
	if err != nil {
		t.Fatalf("LoadYAMLAdapters with no adapters dir: %v", err)
	}
}

func TestLoadYAMLAdapters_ValidAdapter(t *testing.T) {
	homeDir := t.TempDir()
	adaptersDir := filepath.Join(homeDir, ".config", "bak", "adapters")
	if err := os.MkdirAll(adaptersDir, 0755); err != nil {
		t.Fatal(err)
	}

	yamlContent := `name: custom-app
config_path: .config/customapp
categories:
  - name: config
    root_files:
      - settings.json
`
	if err := os.WriteFile(filepath.Join(adaptersDir, "custom.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	r := adapters.NewRegistry()
	if err := All(r); err != nil {
		t.Fatalf("All: %v", err)
	}
	if err := LoadYAMLAdapters(r, false, homeDir); err != nil {
		t.Fatalf("LoadYAMLAdapters: %v", err)
	}

	a, ok := r.Get("custom-app")
	if !ok {
		t.Fatal("expected custom-app adapter to be registered")
	}
	if a.Name() != "custom-app" {
		t.Errorf("Name() = %q, want custom-app", a.Name())
	}
}

func TestLoadYAMLAdapters_OverrideWarning(t *testing.T) {
	homeDir := t.TempDir()
	adaptersDir := filepath.Join(homeDir, ".config", "bak", "adapters")
	if err := os.MkdirAll(adaptersDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a YAML adapter with the same name as a built-in.
	yamlContent := `name: opencode
config_path: .config/opencode-custom
categories: []
`
	if err := os.WriteFile(filepath.Join(adaptersDir, "opencode.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	r := adapters.NewRegistry()
	if err := All(r); err != nil {
		t.Fatalf("All: %v", err)
	}

	// Capture stderr.
	origStderr := os.Stderr
	rpipe, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w

	loadErr := LoadYAMLAdapters(r, true, homeDir)
	w.Close()
	os.Stderr = origStderr

	if loadErr != nil {
		t.Fatalf("LoadYAMLAdapters: %v", loadErr)
	}

	// Read captured stderr output.
	captured, err := io.ReadAll(rpipe)
	if err != nil {
		t.Fatal(err)
	}
	output := string(captured)

	if !strings.Contains(output, "warning: overriding built-in adapter") {
		t.Errorf("expected override warning on stderr, got: %q", output)
	}
}

func TestLoadYAMLAdapters_InvalidYAML(t *testing.T) {
	homeDir := t.TempDir()
	adaptersDir := filepath.Join(homeDir, ".config", "bak", "adapters")
	if err := os.MkdirAll(adaptersDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(adaptersDir, "bad.yaml"), []byte(`{{{invalid`), 0644); err != nil {
		t.Fatal(err)
	}

	r := adapters.NewRegistry()
	if err := All(r); err != nil {
		t.Fatalf("All: %v", err)
	}

	err := LoadYAMLAdapters(r, false, homeDir)
	if err == nil {
		t.Error("expected error for invalid YAML adapter")
	}
}

func TestLoadYAMLAdapters_NoOverrideKeepsBuiltin(t *testing.T) {
	homeDir := t.TempDir()
	adaptersDir := filepath.Join(homeDir, ".config", "bak", "adapters")
	if err := os.MkdirAll(adaptersDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a YAML adapter with same name as built-in, but override=false.
	yamlContent := `name: opencode
config_path: .config/opencode-custom
categories: []
`
	if err := os.WriteFile(filepath.Join(adaptersDir, "opencode.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	r := adapters.NewRegistry()
	if err := All(r); err != nil {
		t.Fatalf("All: %v", err)
	}

	// With override=false, registering a duplicate should fail.
	err := LoadYAMLAdapters(r, false, homeDir)
	if err == nil {
		t.Error("expected error when registering duplicate adapter without override")
	}
}

// Compile-time check: ensure All registers the expected number of adapters.
func TestAll_AdapterCount(t *testing.T) {
	r := adapters.NewRegistry()
	if err := All(r); err != nil {
		t.Fatalf("All: %v", err)
	}
	if got := len(r.All()); got != 8 {
		t.Errorf("All() adapter count = %d, want 8", got)
	}
}

