# Design: Generic Adapter Base Struct

## Technical Approach

Extract a `GenericAdapter` struct into `internal/adapters/generic.go` that implements the full `adapters.Adapter` interface using configurable fields (name, config path, category map, error context). Each of the 7 identical adapters becomes a thin delegation wrapper (~25 lines) around a package-level `GenericAdapter` instance, preserving the existing `&codex.Adapter{}` zero-value construction used by `register.go`.

## Architecture Decisions

### Decision: Delegation over Embedding

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Embed `GenericAdapter` | Promoted methods = zero boilerplate, but zero-value `&Adapter{}` has uninitialized fields → breaks `register.go` | ❌ Rejected |
| Package-level var + delegation | 5 one-liner methods per wrapper, but `&Adapter{}` works unchanged | ✅ **Chosen** |
| Constructor function `New()` | Clean init, but violates AGENTS.md "no constructor functions for internal packages" | ❌ Rejected |

**Rationale**: `register.go` constructs adapters as `&claudecode.Adapter{}`. The spec requires this stays unchanged. A package-level `var base = adapters.GenericAdapter{...}` lets the zero-value `Adapter{}` delegate to a fully-initialized base.

### Decision: scanDir/scanRootFiles as Package-Level Functions

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Methods on GenericAdapter | Ties utility functions to a struct instance they don't need | ❌ Rejected |
| Package-level unexported functions in `adapters/` | Reusable, testable independently, no instance state needed | ✅ **Chosen** |

**Rationale**: `scanDir` and `scanRootFiles` take all their data via parameters (dir, category, configDir, homeDir, catSet). They don't read adapter fields. Making them methods adds coupling without benefit.

### Decision: Single Commit per Adapter Migration

| Option | Tradeoff | Decision |
|--------|----------|----------|
| All 7 adapters in one commit | ~1365 lines changed — exceeds 400-line review budget | ❌ Rejected |
| One commit per adapter | ~195 lines each, each independently verifiable with `go test ./...` | ✅ **Chosen** |

**Rationale**: Each migration is atomic: delete ~170 lines of duplicated code, add ~25 lines of delegation. `go test ./...` validates behavioral preservation per adapter. Review stays under budget.

### Decision: RelPath Conversion — Preserve filepath.ToSlash

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Replace with `path.Clean(strings.ReplaceAll(...))` | AGENTS.md-compliant but may alter edge-case behavior vs existing tests | ❌ Risky |
| Keep `filepath.ToSlash` | Matches existing behavior exactly; AGENTS.md rule targets canonical comparison, not RelPath formatting | ✅ **Chosen** |

**Rationale**: `filepath.ToSlash` is `strings.ReplaceAll(s, "\\", "/")` — identical output for well-formed paths. The AGENTS.md rule targets `path.Clean` for canonical *comparison* (e.g. `paths.ToCanonical`), not for `RelPath` storage. Zero behavioral change is the prime directive.

## Data Flow

```
register.go
    │
    ├─ &codex.Adapter{}──── delegates ──→ codex.base (GenericAdapter)
    ├─ &kiro.Adapter{}──── delegates ──→ kiro.base (GenericAdapter)
    ├─ &cursor.Adapter{}── delegates ──→ cursor.base (GenericAdapter)
    ├─ ...                                        │
    └─ &opencode.Adapter{}── (unchanged)          │
                                                  ▼
                                    adapters/generic.go
                                    ┌─────────────────────────┐
                                    │ GenericAdapter          │
                                    │  .Name()                │
                                    │  .Detect() → os.Stat    │
                                    │  .ListItems()           │
                                    │    ├─ scanDir()         │
                                    │    └─ scanRootFiles()   │
                                    │  .Backup() → CopyFile   │
                                    │  .Restore() → CopyFile  │
                                    └─────────────────────────┘
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/adapters/generic.go` | Create | GenericAdapter struct, CategoryDir type, all interface methods, scanDir, scanRootFiles (~150 lines) |
| `internal/adapters/generic_test.go` | Create | Table-driven tests for GenericAdapter (~120 lines) |
| `internal/adapters/codex/adapter.go` | Modify | Replace body with delegation to package-level GenericAdapter (~25 lines, was ~187) |
| `internal/adapters/kiro/adapter.go` | Modify | Same pattern (~25 lines, was ~187) |
| `internal/adapters/kilocode/adapter.go` | Modify | Same pattern (~25 lines, was ~187) |
| `internal/adapters/pidev/adapter.go` | Modify | Same pattern (~25 lines, was ~187) |
| `internal/adapters/windsurf/adapter.go` | Modify | Same pattern (~25 lines, was ~187) |
| `internal/adapters/cursor/adapter.go` | Modify | Same pattern (~25 lines, was ~196) |
| `internal/adapters/claudecode/adapter.go` | Modify | Same pattern (~28 lines, was ~239) |

**Net reduction**: ~1020 lines → ~310 lines = **~710 lines removed** (meets ≥700 target).

## Interfaces / Contracts

```go
// CategoryDir maps a category to its subdirectory pattern under the config root.
type CategoryDir struct {
    SubPath string // relative path under configDir; empty = root
    IsDir   bool   // true when SubPath is a directory to scan
}

// GenericAdapter implements Adapter for tools that follow the standard
// scan-dir + scan-root-files pattern.
type GenericAdapter struct {
    AdapterName      string
    ConfigRelPath    string
    Categories       map[string]CategoryDir
    DetectErrContext string // e.g. "stat codex config dir"
}

// Compile-time check.
var _ Adapter = (*GenericAdapter)(nil)
```

**Wrapper pattern** (each of the 7 adapter packages):

```go
package codex

import "github.com/danielxxomg/bak-cli/internal/adapters"

const adapterName = "codex"
const configRelPath = ".codex"

var categoryMap = map[string]adapters.CategoryDir{
    "config":       {SubPath: "", IsDir: false},
    "instructions": {SubPath: "instructions", IsDir: true},
}

var base = adapters.GenericAdapter{
    AdapterName:      adapterName,
    ConfigRelPath:    configRelPath,
    Categories:       categoryMap,
    DetectErrContext: "stat codex config dir",
}

type Adapter struct{}

var _ adapters.Adapter = (*Adapter)(nil)

func (a *Adapter) Name() string                                                { return base.Name() }
func (a *Adapter) Detect(homeDir string) (bool, string, error)                 { return base.Detect(homeDir) }
func (a *Adapter) ListItems(homeDir string, cats []string) ([]adapters.Item, error) { return base.ListItems(homeDir, cats) }
func (a *Adapter) Backup(homeDir, backupDir string, items []adapters.Item) error    { return base.Backup(homeDir, backupDir, items) }
func (a *Adapter) Restore(backupDir, homeDir string, items []adapters.Item) error   { return base.Restore(backupDir, homeDir, items) }
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | GenericAdapter methods (Detect, ListItems, Backup, Restore) | Table-driven tests in `generic_test.go` using `t.TempDir()`. Cover: dir exists/missing, file-not-dir, all categories, empty categories, copy errors |
| Regression | All 7 existing adapter test suites | Run `go test ./...` after each migration — zero modifications to existing test files |
| Compile | Interface compliance | `var _ adapters.Adapter = (*GenericAdapter)(nil)` in generic.go; each wrapper keeps its own compile-time check |

## Migration / Rollout

**Order** (simplest first, build confidence):

1. Create `generic.go` + `generic_test.go` → `go test ./...` passes
2. Migrate `codex` (2 categories, simplest) → `go test ./...`
3. Migrate `kiro` (2 categories) → `go test ./...`
4. Migrate `kilocode` (2 categories) → `go test ./...`
5. Migrate `pidev` (2 categories) → `go test ./...`
6. Migrate `windsurf` (2 categories, nested configRelPath) → `go test ./...`
7. Migrate `cursor` (2 categories) → `go test ./...`
8. Migrate `claudecode` (3 categories — most complex) → `go test ./...`

Each step is a separate commit. Each step is independently rollbackable.

## Commit Strategy

| # | Commit | Lines Changed | Content |
|---|--------|--------------|---------|
| 1 | `refactor: add GenericAdapter base struct` | ~270 add | `generic.go` + `generic_test.go` |
| 2 | `refactor: migrate codex adapter to GenericAdapter` | ~170 del, ~25 add | codex/adapter.go |
| 3 | `refactor: migrate kiro adapter to GenericAdapter` | ~170 del, ~25 add | kiro/adapter.go |
| 4 | `refactor: migrate kilocode adapter to GenericAdapter` | ~170 del, ~25 add | kilocode/adapter.go |
| 5 | `refactor: migrate pidev adapter to GenericAdapter` | ~170 del, ~25 add | pidev/adapter.go |
| 6 | `refactor: migrate windsurf adapter to GenericAdapter` | ~170 del, ~25 add | windsurf/adapter.go |
| 7 | `refactor: migrate cursor adapter to GenericAdapter` | ~170 del, ~25 add | cursor/adapter.go |
| 8 | `refactor: migrate claudecode adapter to GenericAdapter` | ~210 del, ~28 add | claudecode/adapter.go |

Each commit is under the 400-line review budget.

## Open Questions

- None
