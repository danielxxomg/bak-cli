package actions

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/danielxxomg/bak-cli/internal/adapters"
	opencodeadapter "github.com/danielxxomg/bak-cli/internal/adapters/opencode"
	"github.com/danielxxomg/bak-cli/internal/backup"
)

// CategoryItem represents a selectable backup category.
type CategoryItem struct {
	Name    string
	Checked bool
}

// PickResult carries the outcome of an interactive picker session.
type PickResult struct {
	Selected  []string
	Confirmed bool
}

// Picker launches an interactive TUI and returns the user's selection.
// The implementation is provided by the cmd package.
type Picker func(categories []CategoryItem) (PickResult, error)

// ResolveBackupID returns the backup ID from args or finds the most
// recent backup when no argument is given.
func ResolveBackupID(backupsDir string, args []string) (string, error) {
	if len(args) > 0 && args[0] != "" {
		return args[0], nil
	}

	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		return "", fmt.Errorf("read backups dir: %w", err)
	}

	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}

	if len(ids) == 0 {
		return "", fmt.Errorf("no backups found — run 'bak backup' first")
	}

	sort.Strings(ids)
	// Most recent backup has the largest timestamp string.
	return ids[len(ids)-1], nil
}

// PickBackupAction orchestrates the interactive picker and backup workflow.
type PickBackupAction struct {
	// Stdout receives user-facing output.
	Stdout io.Writer

	// Picker runs the TUI for category selection.
	Picker Picker

	// Verbose enables detailed output.
	Verbose bool

	// NewRegistry creates an adapter registry. Defaults to real registry
	// with opencode adapter.
	NewRegistry func() (*adapters.Registry, error)

	// BakDir returns the bak storage directory. Defaults to backup.BakDir.
	BakDir func() (string, error)

	// HomeDir returns the user home directory. Defaults to os.UserHomeDir.
	HomeDir func() (string, error)
}

// Run launches the picker and, if confirmed, runs a backup with the
// selected categories.
func (a *PickBackupAction) Run() error {
	categories := []CategoryItem{
		{Name: "skills", Checked: true},
		{Name: "commands", Checked: true},
		{Name: "config", Checked: true},
		{Name: "plugins", Checked: false},
		{Name: "agents", Checked: false},
	}

	result, err := a.Picker(categories)
	if err != nil {
		return fmt.Errorf("tui: %w", err)
	}

	if !result.Confirmed {
		fmt.Fprintln(a.Stdout, "Backup cancelled.")
		return nil
	}

	selected := result.Selected
	if len(selected) == 0 {
		fmt.Fprintln(a.Stdout, "No categories selected. Backup cancelled.")
		return nil
	}

	fmt.Fprintf(a.Stdout, "Selected categories: %s\n", strings.Join(selected, ", "))

	// Run backup with custom categories.
	bakFn := a.BakDir
	if bakFn == nil {
		bakFn = backup.BakDir
	}
	bakDir, err := bakFn()
	if err != nil {
		return fmt.Errorf("bak dir: %w", err)
	}

	homeFn := a.HomeDir
	if homeFn == nil {
		homeFn = os.UserHomeDir
	}
	homeDir, err := homeFn()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}

	regFn := a.NewRegistry
	if regFn == nil {
		regFn = func() (*adapters.Registry, error) {
			reg := adapters.NewRegistry()
			if err := reg.Register(&opencodeadapter.Adapter{}); err != nil {
				return nil, err
			}
			return reg, nil
		}
	}
	reg, err := regFn()
	if err != nil {
		return fmt.Errorf("create registry: %w", err)
	}

	engine := &backup.Engine{
		HomeDir:          homeDir,
		BakDir:           bakDir,
		Registry:         reg,
		CustomCategories: selected,
		Verbose:          a.Verbose,
	}

	bakResult, err := engine.Run()
	if err != nil {
		return fmt.Errorf("backup: %w", err)
	}

	fmt.Fprintf(a.Stdout, "✅ Backup created: %s\n", bakResult.ID)
	fmt.Fprintf(a.Stdout, "   Files: %d, Size: %d bytes\n", bakResult.FileCount, bakResult.TotalSize)
	if bakResult.Secrets > 0 {
		fmt.Fprintf(a.Stdout, "   Secrets excluded: %d files\n", bakResult.Secrets)
	}

	return nil
}
