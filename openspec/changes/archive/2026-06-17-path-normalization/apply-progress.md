# Apply Progress: Path Normalization

## Status: Complete

## Summary

All 6 commits implemented following Strict TDD. Two functions added to `internal/paths/normalize.go` — `Slash()` and `CanonicalPath()` — centralizing the `strings.ReplaceAll(p, "\\", "/")` pattern. All 32 `filepath.ToSlash` violations and ~20 inline `strings.ReplaceAll` patterns replaced across the codebase.

## Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/paths/normalize.go` | Modified | Added `Slash()` and `CanonicalPath()`, replaced 5 internal `filepath.ToSlash` calls |
| `internal/paths/normalize_test.go` | Modified | Added 15 table-driven tests (7 for Slash, 8 for CanonicalPath) |
| `internal/adapters/yaml.go` | Modified | Replaced 11 `filepath.ToSlash` → `paths.Slash`/`paths.CanonicalPath`, added paths import, removed unused `path` import |
| `internal/adapters/opencode/adapter.go` | Modified | Replaced 1 `filepath.ToSlash` → `paths.Slash` |
| `internal/manifest/manifest.go` | Modified | Replaced 2 `filepath.ToSlash` → `paths.CanonicalPath`, added paths import, removed unused `path` import |
| `internal/backup/resolve.go` | Modified | Replaced 3 `filepath.ToSlash` → `paths.CanonicalPath`, added paths import, removed unused `path` import |
| `internal/backup/engine.go` | Modified | Replaced 5 inline patterns → `paths.Slash`/`paths.CanonicalPath`, removed unused `path` import |
| `internal/restore/engine.go` | Modified | Replaced 4 `filepath.ToSlash` → `paths.CanonicalPath`, added paths import, removed unused `path` import |
| `internal/restore/paths.go` | Modified | Replaced 3 `filepath.ToSlash` → `paths.Slash`/`paths.CanonicalPath`, removed unused `path`/`filepath` imports |
| `internal/restore/integration_test.go` | Modified | Replaced 1 `filepath.ToSlash` → `paths.Slash`, added paths import |
| `internal/actions/backup.go` | Modified | Replaced 4 inline patterns → `paths.Slash`/`paths.CanonicalPath`, removed unused `path` import |
| `internal/actions/restore.go` | Modified | Replaced 4 inline patterns → `paths.CanonicalPath`, added paths import, removed unused `path` import |
| `internal/actions/push.go` | Modified | Replaced 2 inline patterns → `paths.CanonicalPath`, added paths import, removed unused `path` import |
| `internal/actions/export.go` | Modified | Replaced 1 inline pattern → `paths.CanonicalPath`, added paths import, removed unused `path`/`strings` imports |
| `internal/cloud/pack.go` | Modified | Replaced 3 inline patterns → `paths.CanonicalPath`, added paths import, updated stale comment |
| `internal/diff/diff.go` | Modified | Removed local `canonicalPath()`, replaced with `paths.CanonicalPath`, removed unused `path`/`strings` imports |
| `internal/diff/diff_test.go` | Modified | Replaced `canonicalPath()` calls → `paths.CanonicalPath()`, removed obsoleted `TestCanonicalPath` test, added paths import |
| `internal/presets/loader.go` | Modified | Replaced 2 `filepath.ToSlash` → `paths.CanonicalPath`, added paths import, removed unused `path` import |

## Verification Results

- **Full test suite**: 1167 tests pass in 26 packages
- **`go vet ./...`**: No issues found
- **`filepath.ToSlash` grep**: Only 1 match — valid godoc comment in `normalize.go` explaining why `Slash` exists
- **`paths.Slash()` usage**: ≥95% of former violation sites now use centralized helpers

## TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1-1.2 | `paths/normalize_test.go` | Unit | ✅ 50/50 | ✅ Written | ✅ Passed | ✅ 7+8 cases | ✅ Clean |
| 1.3 | `paths/normalize.go` | Unit | N/A (refactor) | N/A (approval) | N/A | N/A | ✅ 5 replacements |
| 2.1-2.3 | `adapters/yaml.go`, `opencode/adapter.go` | Unit | ✅ 200/200 | N/A (approval) | ✅ Passed | N/A | ✅ 12 replacements |
| 3.1-3.2 | `manifest/manifest.go` | Unit | ✅ 17/17 | N/A (approval) | ✅ Passed | N/A | ✅ 2 replacements |
| 4.1-4.3 | `backup/resolve.go`, `backup/engine.go` | Unit | ✅ 35/35 | N/A (approval) | ✅ Passed | N/A | ✅ 8 replacements |
| 5.1-5.4 | `restore/engine.go`, `restore/paths.go`, `restore/integration_test.go` | Unit | ✅ 50/50 | N/A (approval) | ✅ Passed | N/A | ✅ 8 replacements |
| 6.1-6.8 | actions/, cloud/, diff/, presets/ | Unit | ✅ 907/907 | N/A (approval) | ✅ Passed | N/A | ✅ ~16 replacements |
| 7.1-7.4 | Full suite | Integration | N/A | N/A | ✅ 1167/1167 | N/A | N/A |

### Test Summary
- **Total tests written**: 15 new (7 Slash + 8 CanonicalPath)
- **Total tests passing**: 1167
- **Layers used**: Unit (15 new), Approval/Regression (existing 1152)
- **Approval tests (refactoring)**: All existing tests served as approval tests — zero behavioral changes detected
- **Pure functions created**: 2 (`Slash`, `CanonicalPath`)

## Deviations from Design

Two deviations from the strict per-commit design, driven by Go compiler requirements:

1. **`diff_test.go` modification**: The design said "all existing tests pass without modification." However, `diff/diff_test.go` referenced the removed local `canonicalPath()` function — it had to be updated to use `paths.CanonicalPath()`. The `TestCanonicalPath` test was removed from diff_test.go since the behavior is now tested in `paths/normalize_test.go`.

2. **Unused import cleanup**: Multiple files had `"path"`, `"path/filepath"`, and `"strings"` imports that became unused after replacing the inline patterns. These were removed to keep each commit compiling independently.

## Issues Found

None — pure mechanical refactor, zero behavioral changes.
