# Archive: cmd-di-refactor

- **Archived**: 2026-06-07
- **Status**: Completed
- **Archive type**: Standard

## Summary

Dependency injection refactor for the `cmd/` package to achieve full test isolation across all platforms. Created `cmdDeps` struct in `cmd/deps.go` and refactored all 13 cmd files to use the `runXWithDeps` wrapper pattern. Fixed `canonicalPath()` in `internal/diff/diff.go` to use `strings.ReplaceAll` instead of OS-dependent `filepath.ToSlash`. Updated all tests to use injected dependencies instead of real filesystem calls.

## Final Metrics

| Metric | Value |
|--------|-------|
| Files created | 2 (`cmd/deps.go`, `cmd/testhelper_test.go`) |
| Files modified | 15 (13 cmd files + `internal/diff/diff.go` + tests) |
| Tests passing | 984 (0 failures, 0 regressions) |
| `go vet` | Clean |
| Docker test (Linux) | All tests pass |
| User-facing changes | Zero (pure refactor) |

## Key Changes

1. **cmdDeps struct** (`cmd/deps.go`): Unexported struct with `ConfigLoader`, `Stdout`, `Stderr`, `Stdin` fields + `defaultDeps` with production defaults
2. **Wrapper pattern**: Every `runX` function now delegates to `runXWithDeps(cmd, args, deps)`
3. **canonicalPath fix**: Replaced `filepath.ToSlash(p)` → `strings.ReplaceAll(p, "\\", "/")` per AGENTS.md
4. **Test isolation**: `setupTestDeps(t)` provides mock loader + buffer I/O for all tests

## Commits

- `refactor(cmd): inject dependencies for test isolation`

## Open Items

None. All tasks marked complete. All tests pass on Windows and Linux.

## Artifacts

- `proposal.md` — Initial proposal with scope and approach
- `specs/spec.md` — Delta spec with ADDED/MODIFIED requirements
- `design.md` — Technical design with architecture decisions
- `tasks.md` — Implementation tasks (all 19 tasks completed)
