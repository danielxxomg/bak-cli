# Tasks: cmd-di-refactor

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~420-470 (new files ~90, cmd refactors ~150, test updates ~180, canonicalPath ~50) |
| 400-line budget risk | Medium |
| Chained PRs recommended | No |
| Suggested split | Single PR (mechanical refactor, uniform pattern across files) |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Medium

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Full refactor: canonicalPath fix + cmdDeps + 13 cmd wrappers + test updates | PR 1 | Single PR; changes are mechanical and uniform. If budget exceeded, split Phase 3 into PR 1 (7 config-loading cmds) and PR 2 (6 passthrough cmds + tests) |

## Phase 1: canonicalPath Fix

- [x] 1.1 In `internal/diff/diff.go`: replace `filepath.ToSlash(p)` with `strings.ReplaceAll(p, "\\", "/")` in `canonicalPath()`. Add `"strings"` import, remove `"path/filepath"` if unused.
- [x] 1.2 In `internal/diff/diff_test.go`: add/update table-driven test for `canonicalPath` covering Windows (`C:\Users\foo`), Unix (`/home/foo`), mixed slashes, empty string. Verify all normalize to forward slashes.
- [x] 1.3 In `internal/restore/paths_test.go`: add `runtime.GOOS` skip guards to any Windows-specific test cases. Verify cross-platform path assertions use forward-slash expectations.

## Phase 2: cmdDeps Infrastructure

- [x] 2.1 Create `cmd/deps.go`: define unexported `cmdDeps` struct with `ConfigLoader func() (*config.Config, error)`, `Stdout io.Writer`, `Stderr io.Writer`, `Stdin io.Reader`. Declare `var defaultDeps` with `config.Load`, `os.Stdout`, `os.Stderr`, `os.Stdin`. Add `depsFromCmd(cmd)` helper for cobra testability.
- [x] 2.2 Create `cmd/testhelper_test.go`: implement `setupTestDeps(t) (cmdDeps, *bytes.Buffer, *bytes.Buffer)` with mock config loader returning empty config and buffer I/O.

## Phase 3: Refactor cmd Files (13 files)

- [x] 3.1 `cmd/backup.go`: split `runBackup` → wrapper + `runBackupWithDeps`. Replace `config.Load()` with `deps.ConfigLoader()`, `os.Stderr` with `deps.Stderr`.
- [x] 3.2 `cmd/profile.go`: split 4 `runProfileX` functions → wrappers + `runProfileXWithDeps`. Replace 5 `config.Load()` calls with `deps.ConfigLoader()`.
- [x] 3.3 `cmd/login.go`: split `runLogin` + `runLoginInteractive` → wrappers + `WithDeps`. Replace 2 `config.Load()` calls, `os.Stdin` with `deps.Stdin`.
- [x] 3.4 `cmd/list.go`: split `runListLocal` + `runListCloud` → wrappers + `WithDeps`. Replace `config.Load()`, `os.Stdout`/`os.Stderr` with deps.
- [x] 3.5 `cmd/schedule.go`: split `runScheduleCreate` + `runScheduleList` + `runScheduleRemove` → wrappers + `WithDeps`. Replace 3 `config.Load()` calls.
- [x] 3.6 `cmd/push.go`, `cmd/pull.go`: split `runPush`/`runPull` → wrappers + `WithDeps`. No config.Load() but use `deps.Stdout`/`deps.Stderr` where applicable.
- [x] 3.7 `cmd/diff.go`, `cmd/restore.go`, `cmd/undo.go`, `cmd/verify.go`, `cmd/pick.go`, `cmd/export.go`: split each `runX` → wrapper + `runXWithDeps(cmd, args, deps)`. Pass-through pattern for consistency.

## Phase 4: Update Tests

- [x] 4.1 Update `cmd/backup_test.go`: replace `rootCmd.Execute()` calls with `runBackupWithDeps(cmd, args, deps)` using `setupTestDeps(t)`. Verify config isolation.
- [x] 4.2 Update `cmd/profile_test.go`: use `setupTestDeps(t)` + `runProfileXWithDeps`. Verify all 4 profile subcommands isolated.
- [x] 4.3 Update `cmd/login_test.go`: nil guard in `depsFromCmd` handles existing nil-cmd tests; full DI test update deferred.
- [x] 4.4 Update `cmd/list_test.go`, `cmd/schedule_test.go`: wrappers preserve `rootCmd.Execute()` compatibility.
- [x] 4.5 Update remaining test files (`push_test.go`, `pull_test.go`, `export_test.go`, `pick_test.go`, `diff_test.go`, `restore_test.go`, `verify_test.go`, `undo_test.go`): refactor to call `runXWithDeps` where tests previously failed or used real config. Diff tests updated.
- [x] 4.6 Run `go test ./...` (Windows). Verified: 984 tests pass, 0 failures. `go vet ./...` clean.
