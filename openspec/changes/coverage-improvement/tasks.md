# Tasks: Coverage Improvement (cmd/ + actions/)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 500–650 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | 5 stacked PRs (per commit grouping below) |
| Delivery strategy | ask-always |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | cmd/ wrapper tests (WithDeps + isTTY refactor) | PR 1 | Standalone; includes `var isTTY` extraction in wizard.go |
| 2 | cmd/ bubbletea model tests (Update/View) | PR 2 | Depends on PR 1 (same package); pure model tests |
| 3 | actions/ error path tests | PR 3 | Independent of cmd/; new test files |
| 4 | actions/ FormatSizeBytes edge cases | PR 4 | Small; can merge with PR 3 if reviewer prefers |
| 5 | E2E tests (export, undo) | PR 5 | Independent; testscript files only |

## Phase 1: cmd/ Wrapper Tests (Commit 1)

- [x] 1.1 Extract `isTTY` to package-level `var isTTY = func() bool` in `cmd/wizard.go` (1-line refactor)
- [x] 1.2 Add `runBackupWithDeps` delegation test in `cmd/backup_test.go` — inject mock ConfigLoader error, verify wrapped error returned
- [x] 1.3 Add `runLoginWithDeps` tests in `cmd/login_test.go` — config error path + non-TTY guard (override `isTTY` → false)
- [x] 1.4 Add `runPickWithDeps` tests in `cmd/pick_test.go` — non-TTY guard + delegation with mock deps
- [x] 1.5 Add `runPushWithDeps` delegation test in `cmd/push_test.go` — verify action wired correctly via mock deps
- [x] 1.6 Add `runPullWithDeps` delegation test in `cmd/pull_test.go` — verify action wired correctly via mock deps
- [x] 1.7 Verify: `go test ./cmd/ -run TestRun -count=1` passes, `go vet ./cmd/` clean

## Phase 2: cmd/ Bubbletea Model Tests (Commit 2)

- [ ] 2.1 Add `pickModel.Update()` tests in `cmd/pick_test.go` — space key toggles selection, down arrow moves cursor, 'q' quits
- [ ] 2.2 Add `pickModel.View()` tests in `cmd/pick_test.go` — verify selected items render with marker, cursor highlights current
- [ ] 2.3 Add `wizardModel.Update()` tests in `cmd/wizard_test.go` — down arrow navigation between steps, enter confirms
- [ ] 2.4 Add `wizardModel.View()` tests in `cmd/wizard_test.go` — verify step highlighting and content rendering
- [ ] 2.5 Verify: `go test ./cmd/ -run TestModel -count=1` passes, no `Program.Run()` calls in any test

## Phase 3: actions/ Error Path Tests (Commit 3)

- [ ] 3.1 Create `internal/actions/export_test.go` — test `RunExport` with invalid ID, missing backup dir, and `writeFailingFS` create error
- [ ] 3.2 Add `CreateTarGz` gzip close error test in `internal/actions/export_test.go` — use pipe-based writer that fails on close
- [ ] 3.3 Add `saveManifest` write failure test in `internal/actions/backup_test.go` — inject `writeFailingFS`, assert wrapped error
- [ ] 3.4 Add `scanBackupForSecrets` fixture test in `internal/actions/backup_test.go` — create temp dir with secret pattern file, verify detection
- [ ] 3.5 Create `internal/actions/pick_backup_test.go` — test `PickBackupAction.Run` with Picker error, not-confirmed, and empty selection
- [ ] 3.6 Add `RestoreAction.Run` cancel path in `internal/actions/restore_test.go` — stdin="n\n", verify abort message
- [ ] 3.7 Verify: `go test ./internal/actions/ -count=1` passes, coverage ≥88%

## Phase 4: actions/ FormatSizeBytes Edge Cases (Commit 4)

- [ ] 4.1 Add table-driven `TestFormatSizeBytes` in `internal/actions/list_local_test.go` — cases: 0→"0 B", 1024→"1.0 KB", 1048576→"1.0 MB", 1073741824→"1.0 GB", 1099511627776→"1.0 TB"
- [ ] 4.2 Add boundary cases: negative input, max int64, values just below/above thresholds (1023, 1025)
- [ ] 4.3 Verify: `go test ./internal/actions/ -run TestFormatSizeBytes -v` passes

## Phase 5: E2E Tests (Commit 5)

- [ ] 5.1 Create `tests/e2e/testdata/export_roundtrip.txtar` — `bak backup --preset quick` → `bak export <id> --output out.tar.gz` → `exists out.tar.gz` → verify tar.gz contains manifest.json
- [ ] 5.2 Create `tests/e2e/testdata/undo_after_restore.txtar` — `bak backup` → `bak restore --force <id>` → `bak undo` → `exec git log` verifies revert commit
- [ ] 5.3 Verify: `go test ./tests/e2e/ -run TestE2E -count=1` passes on local OS
