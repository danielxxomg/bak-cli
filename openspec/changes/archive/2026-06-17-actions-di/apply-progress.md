# Apply Progress: actions-di

## Implementation Summary

Removed the `github.com/spf13/cobra` dependency from `internal/actions/` by injecting `io.Writer` fields for stdout/stderr output. All three actions (BackupAction, PushAction, RestoreAction) now use injected writers with nil-fallback to `os.Stdout`/`os.Stderr`.

## TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1-1.3 | `backup_test.go` | Unit | ✅ 17/17 | ✅ Written | ✅ 21 passed | ✅ 4 cases (Stdout, Stderr, NilFallback, Discard) | ✅ Clean |
| 2.1-2.3 | `push_test.go` | Unit | ✅ 20/20 | ✅ Written | ✅ 22 passed | ✅ 2 cases (Stdout, Discard) | ✅ Clean |
| 3.1-3.3 | `restore_test.go` | Unit | ✅ 17/17 | ✅ Written | ✅ 20 passed | ✅ 3 cases (Stdout, Discard, NilFallback) | ✅ Clean |

### Test Summary
- **Total tests written**: 9 (4 backup + 2 push + 3 restore)
- **Total tests passing**: 1161 (full suite)
- **Layers used**: Unit (9)
- **Approval tests**: None — pure refactor of existing code
- **Pure functions created**: 0

## Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `internal/actions/backup.go` | Modified | Removed cobra import, added `Stdout`/`Stderr io.Writer` fields, changed `Run()` signature, nil-fallback, replaced `fmt.Printf`/`fmt.Fprintf(os.Stderr)` with injected writers |
| `internal/actions/backup_test.go` | Modified | Added 4 new injection tests, migrated all 17 call sites to `Run()` with `io.Discard` |
| `internal/actions/push.go` | Modified | Removed cobra import, added `Stdout`/`Stderr` fields, changed `Run(args []string)` signature, nil-fallback, replaced fmt calls |
| `internal/actions/push_test.go` | Modified | Added 2 new injection tests, migrated all 21 call sites to `Run(...)` with `io.Discard` |
| `internal/actions/restore.go` | Modified | Removed cobra import, added `Stdout`/`Stderr` fields, changed `Run()` signature, replaced `cmd.OutOrStdout()`/`cmd.ErrOrStderr()` with struct field fallback |
| `internal/actions/restore_test.go` | Modified | Added 3 new injection tests, migrated all 12 call sites to `Run()` with `io.Discard` |
| `cmd/backup.go` | Modified | Added `Stdout: deps.Stdout, Stderr: deps.Stderr` to BackupAction, changed to `action.Run()` |
| `cmd/push.go` | Modified | Added `Stdout: deps.Stdout, Stderr: deps.Stderr` to PushAction, changed to `action.Run(args)` |
| `cmd/restore.go` | Modified | Added `Stdout: deps.Stdout, Stderr: deps.Stderr` to RestoreAction, changed to `action.Run()` |

## Deviations from Design

None — implementation matches design.

## Issues Found

None.

## Verification Results

- ✅ `go build ./...` — Success
- ✅ `go test ./...` — 1161 passed in 26 packages
- ✅ `go vet ./...` — No issues found
- ✅ `grep cobra internal/actions/*.go` — No matches (architecture boundary enforced)

## Status

19/19 tasks complete. Ready for `sdd-verify`.
