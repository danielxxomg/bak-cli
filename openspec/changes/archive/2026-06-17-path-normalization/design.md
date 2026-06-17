# Design: Path Normalization

## Technical Approach

Add two helpers to `internal/paths/normalize.go` — `Slash()` for raw backslash conversion and `CanonicalPath()` for the repeated `path.Clean + slash` pattern — then replace all 32 `filepath.ToSlash` violations and consolidate ~20 inline `strings.ReplaceAll` call sites. Pure mechanical refactor, zero behavioral change.

## Architecture Decisions

### Decision: Slash helper location

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `internal/paths/normalize.go` | Natural home, leaf package (no internal imports), already has path utilities | ✅ **Chosen** |
| New `internal/paths/slash.go` | Separation of concerns, but adds a file for one function | Rejected — overkill |
| `internal/adapters/util.go` | Close to most callers | Rejected — creates import cycles, wrong layer |

### Decision: Add CanonicalPath helper

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `Slash()` only | Minimal change, callers still write `path.Clean(paths.Slash(p))` | Rejected — misses DRY opportunity |
| `Slash()` + `CanonicalPath()` | Eliminates ~20 inline `path.Clean(strings.ReplaceAll(...))` patterns; `diff/diff.go` already has a local copy of this exact function | ✅ **Chosen** |

`CanonicalPath(p string) string` = `path.Clean(Slash(p))`. Used everywhere a canonical cross-platform path form is needed.

### Decision: Commit strategy

| Option | Tradeoff | Decision |
|--------|----------|----------|
| One big commit | Simple, but 400+ line diff is hard to review | Rejected — exceeds review budget |
| Per-package commits | 6-7 small commits, each reviewable, each green | ✅ **Chosen** |
| Per-file commits | Too granular, noisy history | Rejected |

Commit order follows dependency depth: `paths/` first (adds helpers + fixes self), then leaf packages inward.

### Decision: Import cycle risk

`internal/paths` imports zero internal packages. Every package in `internal/` can safely import it. No inline `strings.ReplaceAll` exceptions needed — all sites use `paths.Slash` or `paths.CanonicalPath`.

## Data Flow

No data flow changes. This is a pure refactor — same inputs produce same outputs.

```
Before:  filepath.ToSlash(p)  ──→ OS-dependent (no-op on Linux for Windows paths)
After:   paths.Slash(p)       ──→ strings.ReplaceAll(p, "\\", "/")  ──→ always converts
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/paths/normalize.go` | Modify | Add `Slash()` + `CanonicalPath()`, fix 5 internal violations |
| `internal/paths/normalize_test.go` | Modify | Add table-driven tests for `Slash` and `CanonicalPath` |
| `internal/adapters/yaml.go` | Modify | Replace 11 `filepath.ToSlash` → `paths.Slash`/`paths.CanonicalPath` |
| `internal/adapters/opencode/adapter.go` | Modify | Replace 1 `filepath.ToSlash` → `paths.Slash` |
| `internal/manifest/manifest.go` | Modify | Replace 2 `filepath.ToSlash` → `paths.CanonicalPath` |
| `internal/backup/resolve.go` | Modify | Replace 3 `filepath.ToSlash` → `paths.CanonicalPath` |
| `internal/backup/engine.go` | Modify | Replace 5 inline `strings.ReplaceAll` → `paths.Slash`/`paths.CanonicalPath` |
| `internal/restore/engine.go` | Modify | Replace 4 `filepath.ToSlash` → `paths.CanonicalPath` |
| `internal/restore/paths.go` | Modify | Replace 3 `filepath.ToSlash` → `paths.Slash`/`paths.CanonicalPath` |
| `internal/restore/integration_test.go` | Modify | Replace 1 `filepath.ToSlash` → `paths.Slash` |
| `internal/presets/loader.go` | Modify | Replace 2 `filepath.ToSlash` → `paths.CanonicalPath` |
| `internal/actions/backup.go` | Modify | Replace 4 inline `strings.ReplaceAll` → `paths.Slash`/`paths.CanonicalPath` |
| `internal/actions/restore.go` | Modify | Replace 4 inline `strings.ReplaceAll` → `paths.CanonicalPath` |
| `internal/actions/push.go` | Modify | Replace 2 inline `strings.ReplaceAll` → `paths.CanonicalPath` |
| `internal/actions/export.go` | Modify | Replace 1 inline `strings.ReplaceAll` → `paths.CanonicalPath` |
| `internal/cloud/pack.go` | Modify | Replace 3 inline `strings.ReplaceAll` → `paths.CanonicalPath` |
| `internal/diff/diff.go` | Modify | Remove local `canonicalPath()`, use `paths.CanonicalPath` |

## Interfaces / Contracts

```go
// Slash replaces all backslashes with forward slashes.
// Unlike filepath.ToSlash, this is OS-independent: it always converts
// regardless of the host platform.
func Slash(path string) string {
    return strings.ReplaceAll(path, "\\", "/")
}

// CanonicalPath returns a cleaned, forward-slash path suitable for
// cross-platform comparison. Equivalent to path.Clean(Slash(p)).
func CanonicalPath(p string) string {
    return path.Clean(Slash(p))
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `Slash()` with Windows, Unix, empty, mixed inputs | Table-driven in `normalize_test.go`, runs on all OS |
| Unit | `CanonicalPath()` with redundant segments, mixed separators | Table-driven in `normalize_test.go` |
| Regression | All existing tests pass unchanged | `go test ./...` on 3-OS CI matrix |

## Migration / Rollout

No migration required. Pure code refactor, no schema or API changes.

Commit sequence (each commit compiles and passes tests):
1. `refactor(paths): add Slash and CanonicalPath helpers` — add functions + tests, fix `normalize.go` itself
2. `refactor(adapters): use paths.Slash instead of filepath.ToSlash` — yaml.go + opencode/adapter.go
3. `refactor(manifest): use paths.CanonicalPath` — manifest.go
4. `refactor(backup): use paths helpers` — resolve.go + engine.go
5. `refactor(restore): use paths helpers` — engine.go + paths.go + integration_test.go
6. `refactor(actions,cloud,diff,presets): consolidate path normalization` — remaining files

## Open Questions

- None
