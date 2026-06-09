# Apply Progress: ci-fix

**Date**: 2026-06-08  
**Mode**: Strict TDD (test runner: `go test`)  
**Change**: Fix CI blocking issues — errcheck violations, goimports formatting, cross-platform test flakiness

## TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| Fix 1: errcheck backup.go | `internal/actions/backup.go` | Lint | ✅ 1190/1190 | ✅ 6 errcheck violations | ✅ 0 violations | ➖ Structural | ✅ `warnf`/`infof` helpers extracted |
| Fix 2: errcheck util.go | `internal/adapters/util.go` | Lint | ✅ 1190/1190 | ✅ 3 errcheck violations | ✅ 0 violations | ➖ Structural | ➖ None needed |
| Fix 3: goimports formatting | `internal/actions/{backup,export,push}.go` + cloud/adapter files | Lint | ✅ 1190/1190 | ✅ 9+ goimports violations | ✅ 0 violations | ➖ Structural | ➖ None needed |
| Fix 4a: StatFn DI — Detect | `internal/adapters/generic_test.go` | Unit | ✅ 92/92 | ✅ Compile error: `StatFn undefined` | ✅ 7 Detect tests pass | ✅ 2 StatFn cases | ➖ Clean |
| Fix 4b: Backup/Restore cross-platform | `internal/adapters/generic_test.go` | Unit | ✅ 95/95 | N/A (existing tests restructured) | ✅ 95 adapter tests | ✅ 2 error paths | ✅ Removed `chmod`/`runtime` |

## Test Summary
- **Total tests passing**: 1194 (was 1190)
- **New tests added**: 4 (2 StatFn injection, 2 restructured cross-platform)
- **Layers used**: Unit (1194)
- **Lint**: 0 violations (was 9)
- **Vet**: clean
- **Build**: success

## Fix 1: errcheck violations in backup.go

**Before**: 6 unchecked `fmt.Fprintf` calls to stderr/writer in `backup.go`.
- Lines 119, 124: hostname warnings
- Line 137: cleanup warning
- Line 173: secret file removal warning
- Line 277: secret scan warning
- Lines 226–233: backup report info messages

**After**: 
- Extracted `warnf(w io.Writer, format string, args ...any)` helper for stderr warnings
- Extracted `infof(w io.Writer, format string, args ...any)` helper for stdout info messages
- Both helpers use `//nolint:errcheck` internally — write failures to stderr/stdout are non-actionable
- Also applied same helpers to `push.go` for consistency (5 call sites)

## Fix 2: errcheck violations in adapters/util.go

**Before**: 3 unchecked `Close()` calls.
- Line 22: `defer sf.Close()` → now `defer func() { _ = sf.Close() }()`
- Line 35: `df.Close()` → now `_ = df.Close()`
- Line 52: `defer f.Close()` → now `defer func() { _ = f.Close() }()`

**Rationale**: Close() errors are non-actionable in these contexts. The source file is already fully read before closing, and the destination file's close error is already handled at line 39. The error-path close at line 35 is a best-effort cleanup.

## Fix 3: goimports formatting

**Files affected**:
- `internal/actions/backup.go`
- `internal/actions/export.go`
- `internal/actions/push.go`
- `internal/cloud/gitea.go`
- `internal/cloud/gist.go`
- `internal/cloud/content_types.go`
- `internal/cloud/github_gist.go`
- `internal/cloud/github_repo.go`
- `internal/cloud/pack.go`
- `internal/adapters/claudecode/adapter.go`
- `internal/adapters/codex/adapter.go`
- `internal/adapters/codex/adapter_test.go`
- `internal/adapters/cursor/adapter.go`
- `internal/adapters/generic.go`
- `internal/adapters/generic_test.go`
- `internal/adapters/kilocode/adapter.go`
- `internal/adapters/kiro/adapter.go`
- `internal/adapters/kiro/adapter_test.go`
- `internal/adapters/knowledge_test.go`
- `internal/adapters/opencode/adapter.go`
- `internal/adapters/pidev/adapter.go`
- `internal/adapters/pidev/adapter_test.go`
- `internal/adapters/windsurf/adapter.go`
- `internal/adapters/windsurf/adapter_test.go`
- `internal/adapters/yaml.go`
- `internal/backup/engine.go`
- `internal/backup/resolve.go`
- `internal/diff/diff.go`

All fixed with `goimports -w`.

## Fix 4: Cross-platform test — DI approach

### StatFn Injection (GenericAdapter.Detect)

**Added** `StatFn func(string) (os.FileInfo, error)` field to `GenericAdapter`.

**Modified** `Detect()` to use `ga.StatFn` when non-nil, falling back to `os.Stat`.

**Tests added**:
1. `stat_error` — injects StatFn returning `os.ErrPermission`, asserts error propagated
2. `stat not exist via injection` — injects StatFn returning `os.ErrNotExist`, asserts `installed=false`

**Tests removed**: Original `stat_error` test that used `chmod 0000` (skipped on Windows, bypassed by root in CI)

### Backup/Restore Error Tests

**Backup**: Replaced `chmod 0000` with missing source file approach. The `Item` references a path that doesn't exist in `configDir`, so `CopyFile` → `os.Open` fails predictably on all platforms.

**Restore**: Replaced `chmod 0500` with file-at-dir-path approach. Creates a file at `home/.test`, so when `copyItems` tries to create files under it, `os.MkdirAll` fails because the parent is not a directory.

**Removed**: `runtime` import (no longer needed after removing `runtime.GOOS` checks)

## Verification Results

| Check | Result |
|-------|--------|
| `golangci-lint run ./...` | 0 issues |
| `go vet ./...` | No issues found |
| `go build ./...` | Success |
| `go test ./... -count=1` | 1194 passed, 26 packages |

## Files Changed

| File | Action | Description |
|------|--------|-------------|
| `internal/actions/backup.go` | Modified | Added `warnf`/`infof` helpers; wrapped all `fmt.Fprintf` calls |
| `internal/actions/push.go` | Modified | Replaced `fmt.Fprintf` with `warnf`/`infof` (5 call sites) |
| `internal/adapters/util.go` | Modified | Fixed 3 deferred/error-path `Close()` calls |
| `internal/adapters/generic.go` | Modified | Added `StatFn` field; modified `Detect()` to use injection |
| `internal/adapters/generic_test.go` | Modified | Added 2 StatFn tests; restructured Backup/Restore error tests; removed `chmod`/`runtime` dependencies |
| Multiple adapter/cloud files | Modified | `goimports -w` formatting fixes |

## Deviations from Design

None — implementation matches design.

## Issues Found

1. **golangci-lint cascading reveals**: As violations are fixed, golangci-lint reveals additional violations that were previously capped by the linter's per-file limit. Fixed all revealed issues.
2. **`staticcheck` empty-branch**: The user's suggested `if _, err := fmt.Fprintf(...); err != nil { // comment }` pattern triggers `staticcheck` SA9003 (empty branch). Resolved by extracting `warnf`/`infof` helpers with `//nolint:errcheck`.

## Status

5/5 tasks complete. Ready for verify.
