# Archive Report: ci-fix

**Date**: 2026-06-16
**Change**: ci-fix (CI Blocking Issues)
**Artifact store**: openspec
**Verification verdict**: PASS
**Archive location**: `openspec/changes/ci-fix/` (retained in active changes per orchestrator instruction)

## Change Summary

Fixed CI-blocking lint violations and cross-platform test flakiness in bak-cli. Five fixes delivered:

1. **errcheck in backup.go** — Extracted `warnf`/`infof` helpers to wrap `fmt.Fprintf` calls (6 violations)
2. **errcheck in util.go** — Wrapped 3 unchecked `Close()` calls with explicit discard
3. **goimports formatting** — Ran `goimports -w` across 24 files
4. **Cross-platform test DI** — Added `StatFn` injection to `GenericAdapter`; replaced `chmod 0000`/`runtime.GOOS` test hacks with deterministic filesystem approaches
5. **Commit** — Single atomic commit with all fixes

## Files Changed

| File | Action | Description |
|------|--------|-------------|
| `internal/actions/backup.go` | Modified | Added `warnf`/`infof` helpers; wrapped all `fmt.Fprintf` calls |
| `internal/actions/push.go` | Modified | Replaced `fmt.Fprintf` with `warnf`/`infof` (5 call sites) |
| `internal/adapters/util.go` | Modified | Fixed 3 deferred/error-path `Close()` calls |
| `internal/adapters/generic.go` | Modified | Added `StatFn` field; `Detect()` uses injection |
| `internal/adapters/generic_test.go` | Modified | Added 2 StatFn tests; restructured Backup/Restore error tests; removed `chmod`/`runtime` |
| 24 adapter/cloud/action files | Modified | `goimports -w` formatting |

## Verification Evidence

| Command | Result |
|---------|--------|
| `golangci-lint run ./...` | 0 issues |
| `go test ./... -count=1` | 1194 passed, 26 packages |
| `go vet ./...` | No issues |
| `go build ./...` | Success |
| `goimports -l .` | No output |

**Tasks**: 5/5 complete
**TDD compliance**: 6/6 checks passed
**New tests added**: 4 (2 StatFn injection, 2 restructured cross-platform)

## Specs Synced

No delta specs — this change had no `specs/` directory. CI fixes are structural/lint corrections, not feature-level spec changes. The existing `openspec/specs/ci-consistency/spec.md` covers CI conventions but was not affected by this change.

## Missing Artifacts

- proposal.md — not present
- spec (delta) — not present
- design.md — not present

These were not produced for this change. Verification scope was limited to tasks, apply-progress evidence, source inspection, and runtime commands. No CRITICAL issues resulted from this gap.

## Lessons Learned

1. **golangci-lint cascading reveals**: Fixing initial violations reveals additional ones previously hidden by per-file caps. Run lint in a loop until clean.
2. **`staticcheck` SA9003 blocks empty-branch errcheck fixes**: `if _, err := f(); err != nil {}` triggers staticcheck. Extract helpers with `//nolint:errcheck` instead.
3. **`chmod 0000` is unreliable for cross-platform error testing**: Fails on Windows (no permission model) and when running as root in CI. Use dependency injection (`StatFn`) or deterministic filesystem states (missing files, file-at-dir-path) instead.
4. **`warnf`/`infof` pattern scales**: Centralizing stderr/stdout write helpers with `//nolint:errcheck` is cleaner than scattering nolint directives across business logic.

## Recommendations

- Consider adding proposal/spec/design artifacts for future CI-related changes to enable requirement-level verification.
- The `StatFn` DI pattern in `GenericAdapter` is a good candidate for extending to other OS-dependent operations (`os.Open`, `os.ReadDir`) for full testability without filesystem side effects.

## Archive Note

Change directory retained at `openspec/changes/ci-fix/` per orchestrator instruction (not moved to `openspec/changes/archive/`).
