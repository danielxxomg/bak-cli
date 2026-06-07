package backup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/adapters"
	opencodeadapter "github.com/danielxxomg/bak-cli/internal/adapters/opencode"
)

// BenchmarkEngine_Run measures the performance of a full backup cycle
// with the quick preset against an OpenCode fixture.
func BenchmarkEngine_Run(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		home := b.TempDir()
		createBenchFixture(b, home)

		reg := adapters.NewRegistry()
		if err := reg.Register(&opencodeadapter.Adapter{}); err != nil {
			b.Fatalf("register: %v", err)
		}

		bakDir := filepath.Join(home, ".bak")
		engine := &Engine{
			HomeDir:    home,
			BakDir:     bakDir,
			Registry:   reg,
			Preset:     "quick",
			BakVersion: "bench",
		}

		b.StartTimer()
		result, err := engine.Run()
		b.StopTimer()
		if err != nil {
			b.Fatalf("Run: %v", err)
		}
		if result.FileCount == 0 {
			b.Error("expected at least 1 file")
		}
	}
}

// createBenchFixture creates a minimal OpenCode fixture in the home directory.
func createBenchFixture(b *testing.B, home string) {
	b.Helper()

	configDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(configDir, "opencode.json"),
		[]byte(`{"version":"1.0"}`),
		0644,
	); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(configDir, "AGENTS.md"),
		[]byte("# Agents\n"),
		0644,
	); err != nil {
		b.Fatal(err)
	}

	skillDir := filepath.Join(configDir, "skills", "my-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("# My Skill\n"),
		0644,
	); err != nil {
		b.Fatal(err)
	}
}
