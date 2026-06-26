// Package adapters_test validates that each adapter's compile-time constants
// (configRelPath, categoryMap) match the documented configuration structure
// of the corresponding AI coding tool.
package adapters_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/danielxxomg/bak-cli/internal/adapters/claudecode"
	"github.com/danielxxomg/bak-cli/internal/adapters/codex"
	"github.com/danielxxomg/bak-cli/internal/adapters/cursor"
	"github.com/danielxxomg/bak-cli/internal/adapters/kilocode"
	"github.com/danielxxomg/bak-cli/internal/adapters/kiro"
	"github.com/danielxxomg/bak-cli/internal/adapters/pidev"
	"github.com/danielxxomg/bak-cli/internal/adapters/windsurf"
)

// adapterKnowledge holds the expected documented values for a single adapter.
type adapterKnowledge struct {
	name       string   // adapter identifier
	configPath string   // expected ConfigRelPath
	categories []string // expected category keys in CategoryMap
}

// expectedKnowledge is the authoritative reference for what each adapter's
// constants should be, sourced from tool documentation research (see design.md).
var expectedKnowledge = []adapterKnowledge{
	{
		name:       "claude-code",
		configPath: ".claude",
		categories: []string{"config", "skills", "commands", "agents", "plugins"},
	},
	{
		name:       "cursor",
		configPath: ".cursor",
		categories: []string{"config", "extensions", "mcp"},
	},
	{
		name:       "codex",
		configPath: ".codex",
		categories: []string{"config", "agents"},
	},
	{
		name:       "windsurf",
		configPath: ".codeium/windsurf",
		categories: []string{"config", "rules", "skills"},
	},
	{
		name:       "kiro",
		configPath: ".kiro",
		categories: []string{"config", "agents", "steering", "specs"},
	},
	{
		name:       "kilocode",
		configPath: ".kilocode",
		categories: []string{"config", "rules", "workflows", "skills"},
	},
	{
		name:       "pidev",
		configPath: ".pi/agent",
		categories: []string{"config", "agents"},
	},
}

// adapterInfo couples an adapter's name with its exported constants.
type adapterInfo struct {
	name       string
	configPath string
	categories []string
}

// allAdapters returns the current runtime state of all GenericAdapter-based adapters.
func allAdapters() []adapterInfo {
	return []adapterInfo{
		{name: claudecode.AdapterName, configPath: claudecode.ConfigRelPath, categories: categoryKeys(claudecode.CategoryMap)},
		{name: cursor.AdapterName, configPath: cursor.ConfigRelPath, categories: categoryKeys(cursor.CategoryMap)},
		{name: codex.AdapterName, configPath: codex.ConfigRelPath, categories: categoryKeys(codex.CategoryMap)},
		{name: windsurf.AdapterName, configPath: windsurf.ConfigRelPath, categories: categoryKeys(windsurf.CategoryMap)},
		{name: kiro.AdapterName, configPath: kiro.ConfigRelPath, categories: categoryKeys(kiro.CategoryMap)},
		{name: kilocode.AdapterName, configPath: kilocode.ConfigRelPath, categories: categoryKeys(kilocode.CategoryMap)},
		{name: pidev.AdapterName, configPath: pidev.ConfigRelPath, categories: categoryKeys(pidev.CategoryMap)},
	}
}

// categoryKeys returns the sorted keys of a CategoryMap.
func categoryKeys(m interface{}) []string {
	v := reflect.ValueOf(m)
	if v.Kind() != reflect.Map {
		return nil
	}
	keys := make([]string, 0, v.Len())
	for _, k := range v.MapKeys() {
		keys = append(keys, k.String())
	}
	slices.Sort(keys)
	return keys
}

func TestAdapterKnowledge_ConfigPaths(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	actual := allAdapters()

	for _, exp := range expectedKnowledge { //nolint:paralleltest // subtests share table/struct state
		t.Run(exp.name+"/configPath", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			found := false
			for _, act := range actual {
				if act.name == exp.name {
					found = true
					if act.configPath != exp.configPath {
						t.Errorf("ConfigRelPath = %q, want %q", act.configPath, exp.configPath)
					}
					break
				}
			}
			if !found {
				t.Errorf("adapter %q not found in runtime adapters — check allAdapters()", exp.name)
			}
		})
	}
}

func TestAdapterKnowledge_Categories(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	actual := allAdapters()

	for _, exp := range expectedKnowledge { //nolint:paralleltest // subtests share table/struct state
		t.Run(exp.name+"/categories", func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			found := false
			for _, act := range actual {
				if act.name == exp.name {
					found = true
					for _, wantCat := range exp.categories {
						if !slices.Contains(act.categories, wantCat) {
							t.Errorf("CategoryMap missing category %q in %s adapter", wantCat, exp.name)
						}
					}
					break
				}
			}
			if !found {
				t.Errorf("adapter %q not found in runtime adapters — check allAdapters()", exp.name)
			}
		})
	}
}

func TestAdapterKnowledge_NoExtraAdapters(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	actual := allAdapters()
	for _, act := range actual {
		found := false
		for _, exp := range expectedKnowledge {
			if act.name == exp.name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("adapter %q present in runtime but missing from expectedKnowledge table", act.name)
		}
	}
}
