# Proposal: Path Normalization

## Intent

`filepath.ToSlash` is OS-dependent: on Linux, `\` is a valid filename character (not a separator), so `filepath.ToSlash("C:\\Users\\foo")` is a **no-op**. This breaks cross-platform manifest comparison — a Windows backup restored on Linux produces corrupted relative paths. AGENTS.md already mandates `path.Clean` + `strings.ReplaceAll(path, "\\", "/")` but 32 call sites still use the wrong function.

## Scope

### In Scope
- Replace all `filepath.ToSlash` calls with `strings.ReplaceAll(p, "\\", "/")` across 9 files
- Extract a shared `paths.Slash(p string) string` helper to centralize the pattern (DRY)
- Update existing tests to cover Windows-style input on all platforms

### Out of Scope
- Changing `filepath.Join` / `filepath.Rel` usage (these are correct for OS-native paths)
- Adding new cross-platform features
- Modifying manifest schema or backup format

## Capabilities

### New Capabilities
- `path-normalization`: Centralized, OS-independent path slash conversion using `strings.ReplaceAll` instead of `filepath.ToSlash`. All cross-platform canonical path comparisons MUST use this helper.

### Modified Capabilities
None — no existing spec-level behavior changes. This is a bugfix at the implementation level.

## Approach

1. Add `paths.Slash(p string) string` in `internal/paths/normalize.go` — wraps `strings.ReplaceAll(p, "\\", "/")`
2. Fix `paths/normalize.go` itself (5 violations in `toCanonical`, `isUnder`)
3. Replace violations in dependency order: `paths/` → `adapters/` → `manifest/` → `backup/` → `restore/` → `presets/`
4. Each call site: `filepath.ToSlash(x)` → `paths.Slash(x)` (or inline `strings.ReplaceAll` where importing `paths` creates a cycle)
5. Add table-driven tests with Windows, macOS, and Linux path inputs

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/paths/normalize.go` | Modified | Fix 5 violations + add `Slash()` helper |
| `internal/adapters/yaml.go` | Modified | Fix 11 violations |
| `internal/adapters/opencode/adapter.go` | Modified | Fix 1 violation |
| `internal/manifest/manifest.go` | Modified | Fix 2 violations |
| `internal/backup/resolve.go` | Modified | Fix 3 violations |
| `internal/restore/engine.go` | Modified | Fix 4 violations |
| `internal/restore/paths.go` | Modified | Fix 3 violations |
| `internal/restore/integration_test.go` | Modified | Fix 1 violation |
| `internal/presets/loader.go` | Modified | Fix 2 violations |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Import cycle if `paths` is imported by adapters that `paths` depends on | Low | Check dependency graph; inline `strings.ReplaceAll` where cycles would occur |
| Behavioral change on Linux paths containing literal `\` | Very Low | Such paths are vanishingly rare in config dirs; document the edge case |
| Missed call sites | Low | Add GGA/linter rule to ban `filepath.ToSlash` after migration |

## Rollback Plan

Revert the commit(s). This is a pure mechanical replacement with no schema or API changes — `git revert` restores the previous state cleanly.

## Dependencies

None — uses only Go stdlib (`strings`, `path`).

## Success Criteria

- [ ] Zero `filepath.ToSlash` calls remain in `internal/` (verified via `grep -r`)
- [ ] All existing tests pass on Windows, macOS, and Linux CI matrix
- [ ] New table-driven tests cover Windows-style paths (`C:\Users\...`) on all platforms
- [ ] `paths.Slash()` exported and used by ≥80% of former violation sites
