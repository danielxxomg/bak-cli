package cmd

import (
	"fmt"

	"github.com/danielxxomg/bak-cli/internal/backup"
	"github.com/danielxxomg/bak-cli/internal/diff"
	"github.com/danielxxomg/bak-cli/internal/manifest"
	"github.com/spf13/cobra"
)

// diffCmd represents the diff command.
var diffCmd = &cobra.Command{
	Use:   "diff <id1> <id2>",
	Short: "Show differences between two backups",
	Long: `Compares two backups and shows file-level differences grouped by
category: Added, Removed, Modified, and Unchanged.

Always exits 0 on success, even when differences are found.

Examples:
  bak diff 20260604-150405 20260605-080000`,
	Args: cobra.ExactArgs(2),
	RunE: runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	id1, id2 := args[0], args[1]

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

	if len(entries) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Backups %q and %q are identical — no differences.\n", id1, id2)
		return nil
	}

	// Group entries by category.
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

	// Print grouped output in a stable order.
	order := []diff.Category{
		diff.CategoryAdded,
		diff.CategoryRemoved,
		diff.CategoryModified,
		diff.CategoryUnchanged,
	}

	for _, cat := range order {
		items := groups[cat]
		if len(items) == 0 {
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s:\n", cat)
		for _, e := range items {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s (%s)\n", e.SourcePath, e.Adapter)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Summary: %d added, %d removed, %d modified, %d unchanged, %d total\n",
		counts[diff.CategoryAdded],
		counts[diff.CategoryRemoved],
		counts[diff.CategoryModified],
		counts[diff.CategoryUnchanged],
		len(entries),
	)

	return nil
}
