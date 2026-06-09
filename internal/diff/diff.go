// Package diff compares two backup manifests and reports file-level
// differences categorized as Added, Removed, Modified, or Unchanged.
package diff

import (
	"sort"

	"github.com/danielxxomg/bak-cli/internal/manifest"
	"github.com/danielxxomg/bak-cli/internal/paths"
)

// Category labels the type of difference for a single entry.
type Category string

const (
	CategoryAdded     Category = "Added"
	CategoryRemoved   Category = "Removed"
	CategoryModified  Category = "Modified"
	CategoryUnchanged Category = "Unchanged"
)

// DiffEntry represents one file-level difference between two backups.
type DiffEntry struct {
	SourcePath string // canonical path (path.Clean + strings.ReplaceAll)
	Category   Category
	Adapter    string // adapter name from the manifest where the item was found
}

// indexedItem pairs a manifest item with its adapter name for diff computation.
type indexedItem struct {
	Item    manifest.Item
	Adapter string
}

// Compare returns the set of differences between manifests a and b.
// Entries are sorted by SourcePath.
func Compare(a, b *manifest.Manifest) []DiffEntry {
	flatA := flatten(a)
	flatB := flatten(b)

	// Collect all unique canonical paths.
	keys := make(map[string]struct{}, len(flatA)+len(flatB))
	for k := range flatA {
		keys[k] = struct{}{}
	}
	for k := range flatB {
		keys[k] = struct{}{}
	}

	var result []DiffEntry
	for key := range keys {
		itemA, inA := flatA[key]
		itemB, inB := flatB[key]

		switch {
		case inA && !inB:
			result = append(result, DiffEntry{
				SourcePath: key,
				Category:   CategoryRemoved,
				Adapter:    itemA.Adapter,
			})
		case !inA && inB:
			result = append(result, DiffEntry{
				SourcePath: key,
				Category:   CategoryAdded,
				Adapter:    itemB.Adapter,
			})
		case itemA.Item.Hash == itemB.Item.Hash:
			result = append(result, DiffEntry{
				SourcePath: key,
				Category:   CategoryUnchanged,
				Adapter:    itemA.Adapter,
			})
		default:
			result = append(result, DiffEntry{
				SourcePath: key,
				Category:   CategoryModified,
				Adapter:    itemA.Adapter,
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].SourcePath < result[j].SourcePath
	})

	return result
}

// flatten builds a map of canonical source paths to (item, adapter) pairs
// from all adapters in the manifest.
func flatten(m *manifest.Manifest) map[string]indexedItem {
	if m == nil {
		return nil
	}
	out := make(map[string]indexedItem)
	for adapterName, am := range m.Adapters {
		for _, item := range am.Items {
			key := paths.CanonicalPath(item.SourcePath)
			out[key] = indexedItem{Item: item, Adapter: adapterName}
		}
	}
	return out
}
