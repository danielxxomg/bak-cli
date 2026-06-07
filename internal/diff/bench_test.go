package diff

import (
	"fmt"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/manifest"
)

// BenchmarkCompare measures the performance of diffing two manifests
// with varying item counts.
func BenchmarkCompare(b *testing.B) {
	sizes := []int{10, 100, 1000}

	for _, n := range sizes {
		b.Run(fmt.Sprintf("items=%d", n), func(b *testing.B) {
			a := makeManifestWithItems("opencode", n, "sha256:old")
			bManifest := makeManifestWithItems("opencode", n, "sha256:new")

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = Compare(a, bManifest)
			}
		})
	}
}

// BenchmarkCompare_Unchanged measures the fast path where all items match.
func BenchmarkCompare_Unchanged(b *testing.B) {
	a := makeManifestWithItems("opencode", 500, "sha256:same")
	bManifest := makeManifestWithItems("opencode", 500, "sha256:same")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Compare(a, bManifest)
	}
}

// makeManifestWithItems creates a manifest with n synthetic items.
func makeManifestWithItems(adapterName string, n int, hash string) *manifest.Manifest {
	items := make([]manifest.Item, n)
	for i := 0; i < n; i++ {
		items[i] = manifest.Item{
			Category:   "config",
			SourcePath: fmt.Sprintf("path/to/file_%d.json", i),
			BackupPath: fmt.Sprintf("opencode/path/to/file_%d.json", i),
			Hash:       hash,
			Size:       1024,
		}
	}
	return &manifest.Manifest{
		Version: "0.3.0",
		ID:      "bench",
		Adapters: map[string]manifest.AdapterManifest{
			adapterName: {Items: items},
		},
	}
}
