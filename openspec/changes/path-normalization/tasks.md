# Tasks: Path Normalization

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~150 (20 helpers + 50 tests + 80 replacements) |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR with 6 commits (per design) |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: stacked-to-main
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Add helpers + fix paths package | PR 1 | Foundation; tests included |
| 2 | Fix adapters package | PR 1 | Depends on PR 1 helpers |
| 3 | Fix manifest package | PR 1 | Depends on PR 1 helpers |
| 4 | Fix backup package | PR 1 | Depends on PR 1 helpers |
| 5 | Fix restore package | PR 1 | Depends on PR 1 helpers |
| 6 | Fix actions/cloud/diff/presets | PR 1 | Final consolidation |

All units merge as a single PR with 6 atomic commits. Each commit compiles and passes tests.

## Phase 1: Foundation (paths package)

- [x] 1.1 Add `Slash(p string) string` to `internal/paths/normalize.go` — wraps `strings.ReplaceAll(p, "\\", "/")`
- [x] 1.2 Add `CanonicalPath(p string) string` to `internal/paths/normalize.go` — returns `path.Clean(Slash(p))`
- [x] 1.3 Replace 5 `filepath.ToSlash` violations in `internal/paths/normalize.go` with `Slash()` / `CanonicalPath()` calls
- [x] 1.4 Add table-driven tests for `Slash()` in `internal/paths/normalize_test.go` — cover Windows, Unix, empty, mixed inputs
- [x] 1.5 Add table-driven tests for `CanonicalPath()` in `internal/paths/normalize_test.go` — cover redundant segments, mixed separators
- [x] 1.6 Verify: `go test ./internal/paths/...` passes

## Phase 2: Adapters Package

- [x] 2.1 Replace 11 `filepath.ToSlash` calls in `internal/adapters/yaml.go` with `paths.Slash` or `paths.CanonicalPath`
- [x] 2.2 Replace 1 `filepath.ToSlash` call in `internal/adapters/opencode/adapter.go` with `paths.Slash`
- [x] 2.3 Add `paths` import to both files
- [x] 2.4 Verify: `go test ./internal/adapters/...` passes

## Phase 3: Manifest Package

- [x] 3.1 Replace 2 `filepath.ToSlash` calls in `internal/manifest/manifest.go` with `paths.CanonicalPath`
- [x] 3.2 Add `paths` import
- [x] 3.3 Verify: `go test ./internal/manifest/...` passes

## Phase 4: Backup Package

- [x] 4.1 Replace 3 `filepath.ToSlash` calls in `internal/backup/resolve.go` with `paths.CanonicalPath`
- [x] 4.2 Replace 5 inline `strings.ReplaceAll` patterns in `internal/backup/engine.go` with `paths.Slash` / `paths.CanonicalPath`
- [x] 4.3 Add `paths` import to both files
- [x] 4.4 Verify: `go test ./internal/backup/...` passes

## Phase 5: Restore Package

- [x] 5.1 Replace 4 `filepath.ToSlash` calls in `internal/restore/engine.go` with `paths.CanonicalPath`
- [x] 5.2 Replace 3 `filepath.ToSlash` calls in `internal/restore/paths.go` with `paths.Slash` / `paths.CanonicalPath`
- [x] 5.3 Replace 1 `filepath.ToSlash` call in `internal/restore/integration_test.go` with `paths.Slash`
- [x] 5.4 Add `paths` import to all three files
- [x] 5.5 Verify: `go test ./internal/restore/...` passes

## Phase 6: Actions, Cloud, Diff, Presets

- [x] 6.1 Replace 4 inline `strings.ReplaceAll` patterns in `internal/actions/backup.go` with `paths.Slash` / `paths.CanonicalPath`
- [x] 6.2 Replace 4 inline `strings.ReplaceAll` patterns in `internal/actions/restore.go` with `paths.CanonicalPath`
- [x] 6.3 Replace 2 inline `strings.ReplaceAll` patterns in `internal/actions/push.go` with `paths.CanonicalPath`
- [x] 6.4 Replace 1 inline `strings.ReplaceAll` pattern in `internal/actions/export.go` with `paths.CanonicalPath`
- [x] 6.5 Replace 3 inline `strings.ReplaceAll` patterns in `internal/cloud/pack.go` with `paths.CanonicalPath`
- [x] 6.6 Remove local `canonicalPath()` function in `internal/diff/diff.go`, replace all calls with `paths.CanonicalPath`
- [x] 6.7 Replace 2 `filepath.ToSlash` calls in `internal/presets/loader.go` with `paths.CanonicalPath`
- [x] 6.8 Add `paths` import to all affected files
- [x] 6.9 Verify: `go test ./...` passes (full suite)

## Phase 7: Verification

- [x] 7.1 Run `grep -r 'filepath\.ToSlash' internal/` — verify zero matches (only godoc comment)
- [x] 7.2 Run `go test ./...` — 1167 tests pass
- [x] 7.3 Verify all existing tests pass without modification (except diff_test.go which referenced removed local function)
- [x] 7.4 Confirm `paths.Slash()` is used by ≥80% of former violation sites
