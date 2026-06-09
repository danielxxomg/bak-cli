# Tasks: actions-di (Dependency Injection for Actions Package)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~150–180 (additions + deletions) |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | ask-always |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: stacked-to-main
400-line budget risk: Low

## Phase 1: BackupAction DI

- [x] 1.1 **RED**: Add `TestBackupAction_StdoutInjection` in `internal/actions/backup_test.go` — inject `bytes.Buffer` for Stdout/Stderr, call `Run()`, assert output captured in buffers
- [x] 1.2 **GREEN**: In `internal/actions/backup.go` — remove `cobra` import, add `io` import, add `Stdout io.Writer` + `Stderr io.Writer` fields to `BackupAction`, change signature to `Run() error`, add nil-check fallback (`out`/`errOut` locals), replace all `fmt.Printf` → `fmt.Fprintf(out, ...)`, replace all `fmt.Fprintf(os.Stderr, ...)` → `fmt.Fprintf(errOut, ...)`
- [x] 1.3 **MIGRATE**: Update all 15 `Run(nil, nil)` calls in `backup_test.go` → `Run()`, inject `Stdout: io.Discard, Stderr: io.Discard` in every `BackupAction` struct literal
- [x] 1.4 **WIRE**: In `cmd/backup.go` — add `Stdout: deps.Stdout, Stderr: deps.Stderr` to `BackupAction` literal, change `action.Run(cmd, args)` → `action.Run()`
- [x] 1.5 **VERIFY**: `go build ./...` and `go test ./internal/actions/ -run Backup` and `go test ./cmd/ -run Backup` pass; `grep cobra internal/actions/backup.go` returns nothing

## Phase 2: PushAction DI

- [x] 2.1 **RED**: Add `TestPushAction_StdoutInjection` in `internal/actions/push_test.go` — inject `bytes.Buffer` for Stdout/Stderr, call `Run(args)`, assert output captured
- [x] 2.2 **GREEN**: In `internal/actions/push.go` — remove `cobra` import, add `io` import, add `Stdout io.Writer` + `Stderr io.Writer` fields, change signature to `Run(args []string) error`, add nil-check fallback, replace all `fmt.Printf` → `fmt.Fprintf(out, ...)`, replace all `fmt.Fprintf(os.Stderr, ...)` → `fmt.Fprintf(errOut, ...)`
- [x] 2.3 **MIGRATE**: Update all 21 `Run(nil, ...)` calls in `push_test.go` → `Run(...)` (drop first nil arg), inject `Stdout: io.Discard, Stderr: io.Discard` in `PushAction` struct literals
- [x] 2.4 **WIRE**: In `cmd/push.go` — change `_ cmdDeps` → `deps cmdDeps`, add `Stdout: deps.Stdout, Stderr: deps.Stderr` to `PushAction` literal, change `action.Run(cmd, args)` → `action.Run(args)`
- [x] 2.5 **VERIFY**: `go build ./...` and `go test ./internal/actions/ -run Push` and `go test ./cmd/ -run Push` pass; `grep cobra internal/actions/push.go` returns nothing

## Phase 3: RestoreAction DI

- [x] 3.1 **RED**: Add `TestRestoreAction_StdoutInjection` in `internal/actions/restore_test.go` — inject `bytes.Buffer` for Stdout/Stderr, call `Run()`, assert output captured in buffers
- [x] 3.2 **GREEN**: In `internal/actions/restore.go` — remove `cobra` import, add `Stdout io.Writer` + `Stderr io.Writer` fields, change signature to `Run() error`, replace cmd-based writer resolution (lines 47–53) with nil-check fallback on `a.Stdout`/`a.Stderr`, replace all `out`/`errOut` usages to use the injected fields
- [x] 3.3 **MIGRATE**: Update all 12 `Run(nil, []string{...})` calls in `restore_test.go` → `Run()`, inject `Stdout: io.Discard, Stderr: io.Discard` in `RestoreAction` struct literals
- [x] 3.4 **WIRE**: In `cmd/restore.go` — add `Stdout: deps.Stdout, Stderr: deps.Stderr` to `RestoreAction` literal (alongside existing `Stdin: deps.Stdin`), change `action.Run(cmd, args)` → `action.Run()`
- [x] 3.5 **VERIFY**: `go build ./...` and `go test ./internal/actions/ -run Restore` and `go test ./cmd/ -run Restore` pass; `grep cobra internal/actions/restore.go` returns nothing

## Phase 4: Final Verification

- [x] 4.1 Run `go build ./...` — must succeed with zero errors
- [x] 4.2 Run `go test ./...` — all tests pass across all packages
- [x] 4.3 Run `grep -r "spf13/cobra" internal/actions/` — must return nothing (architecture boundary enforced)
- [x] 4.4 Run `gga` (Guardian Angel) if available — must pass AGENTS.md boundary checks
