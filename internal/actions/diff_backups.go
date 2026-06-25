package actions

import (
	"fmt"
	"io"

	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/danielxxomg/bak-cli/internal/diff"
	"github.com/danielxxomg/bak-cli/internal/manifest"
)

// DiffBackupsAction compares two backups and writes a human-readable
// diff to the given output writer. All I/O is injected for testability.
type DiffBackupsAction struct {
	// Stdout is the writer for formatted diff output.
	Stdout io.Writer
}

// Run compares two backups by ID and writes grouped diff output to Stdout.
// Returns nil even when differences are found (diff is informational).
func (a *DiffBackupsAction) Run(id1, id2 string) error {
	dir1, err := backup.ResolveBackupID(id1)
	if err != nil {
		return fmt.Errorf("backup %q: %w", id1, err)
	}

	dir2, err := backup.ResolveBackupID(id2)
	if err != nil {
		return fmt.Errorf("backup %q: %w", id2, err)
	}

	m1, err := manifest.Load(dir1)
	if err != nil {
		return fmt.Errorf("load manifest %q: %w", id1, err)
	}

	m2, err := manifest.Load(dir2)
	if err != nil {
		return fmt.Errorf("load manifest %q: %w", id2, err)
	}

	entries := diff.Compare(m1, m2)

	out := a.Stdout
	if len(entries) == 0 {
		_, _ = fmt.Fprintf(out, "Backups %q and %q are identical — no differences.\n", id1, id2)
		return nil
	}

	groups, counts := groupDiffEntries(entries)
	printDiffGroups(out, groups)
	printDiffSummary(out, counts, len(entries))

	return nil
}

// diffCategoryOrder is the stable display order for diff categories.
var diffCategoryOrder = []diff.Category{
	diff.CategoryAdded,
	diff.CategoryRemoved,
	diff.CategoryModified,
	diff.CategoryUnchanged,
}

// groupDiffEntries groups diff entries by category and counts per category.
// It is a pure function over the input slice; all four category keys are
// always present in the returned maps (empty groups are initialized).
func groupDiffEntries(entries []diff.DiffEntry) (map[diff.Category][]diff.DiffEntry, map[diff.Category]int) {
	groups := map[diff.Category][]diff.DiffEntry{
		diff.CategoryAdded:     {},
		diff.CategoryRemoved:   {},
		diff.CategoryModified:  {},
		diff.CategoryUnchanged: {},
	}
	counts := map[diff.Category]int{}
	for _, e := range entries {
		groups[e.Category] = append(groups[e.Category], e)
		counts[e.Category]++
	}
	return groups, counts
}

// printDiffGroups writes diff entries grouped by category to out in a
// stable order (Added, Removed, Modified, Unchanged). Empty groups are
// skipped; each non-empty group is followed by a blank line.
func printDiffGroups(out io.Writer, groups map[diff.Category][]diff.DiffEntry) {
	for _, cat := range diffCategoryOrder {
		items := groups[cat]
		if len(items) == 0 {
			continue
		}
		_, _ = fmt.Fprintf(out, "%s:\n", cat)
		for _, e := range items {
			_, _ = fmt.Fprintf(out, "  %s (%s)\n", e.SourcePath, e.Adapter)
		}
		_, _ = fmt.Fprintln(out)
	}
}

// printDiffSummary writes the one-line diff summary to out.
func printDiffSummary(out io.Writer, counts map[diff.Category]int, total int) {
	_, _ = fmt.Fprintf(out, "Summary: %d added, %d removed, %d modified, %d unchanged, %d total\n",
		counts[diff.CategoryAdded],
		counts[diff.CategoryRemoved],
		counts[diff.CategoryModified],
		counts[diff.CategoryUnchanged],
		total,
	)
}
