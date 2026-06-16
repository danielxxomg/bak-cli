# Archive Report: coverage-improvement

**Change**: coverage-improvement
**Archived**: 2026-06-16
**Verdict**: PASS WITH WARNINGS (intentional-with-warnings)

## Change Summary

Testing-only change to improve code coverage in `cmd/` (46.6% → 58.6%) and `internal/actions/` (82.9% → 83.6%). Added quality tests exercising real logic: delegation wrappers, bubbletea model Update/View, error paths, pure functions, and E2E testscripts. No production code behavior changes (except 1-line `isTTY` extraction for test injection and `FormatSizeBytes` consolidation).

## Story

1. **Phases 1-5 implemented** across 5 stacked PRs (PR1: cmd/ wrappers, PR2: bubbletea models, PR3: actions/ error paths, PR4: FormatSizeBytes, PR5: E2E tests)
2. **Initial verify found 2 missing tests**: `RunExport` create error and `CreateTarGz` gzip close error were marked complete in tasks.md but not present in the codebase. Verify verdict: FAIL.
3. **PR #20 added the missing tests**: `TestRunExport_CreateError` and `TestCreateTarGz_GzipCloseError` were added and merged to main.
4. **Verify updated**: Verdict changed from FAIL to PASS WITH WARNINGS. All 18 spec scenarios now compliant (16 fully, 2 partial by design).
5. **Archived**: Coverage targets not met (cmd/ 58.6% vs 70-75%, actions/ 83.6% vs 88-90%) but all tests exercise real logic and provide regression safety.

## Files Created/Modified

### cmd/ (Phase 1-2)
| File | Action | Description |
|------|--------|-------------|
| `cmd/wizard.go` | Modified | Extracted `isTTY` to package-level var for test injection |
| `cmd/backup_test.go` | Modified | Added `runBackupWithDeps` delegation test |
| `cmd/login_test.go` | Created | `runLoginWithDeps` config error + non-TTY guard tests |
| `cmd/pick_test.go` | Modified | Added `runPickWithDeps` non-TTY guard + `pickModel` Update/View tests |
| `cmd/push_test.go` | Modified | Added `runPushWithDeps` delegation test |
| `cmd/pull_test.go` | Modified | Added `runPullWithDeps` delegation test |
| `cmd/wizard_test.go` | Modified | Added `wizardModel` Update/View tests |

### internal/actions/ (Phase 3-4)
| File | Action | Description |
|------|--------|-------------|
| `internal/actions/export_test.go` | Created | 20 tests: RunExport (happy, invalid ID, create error), CreateTarGz (gzip close error, empty dir), IsValidBackupID, FormatBackupIDError |
| `internal/actions/backup_test.go` | Modified | +4 scanBackupForSecrets tests with real file fixtures |
| `internal/actions/pick_backup_test.go` | Created | 7 tests for PickBackupAction.Run error paths + happy path |
| `internal/actions/restore_test.go` | Modified | +5 cancel/confirm/force tests; fixed real SHA-256 in fixture |
| `internal/actions/list_local_test.go` | Modified | Table-driven TestFormatSizeBytes with 19 sub-cases |
| `internal/actions/list_local.go` | Modified | Extended FormatSizeBytes from GB to EB; DRY consolidation |
| `internal/actions/backup.go` | Modified | Consolidated formatSize to delegate to FormatSizeBytes |

### tests/e2e/ (Phase 5)
| File | Action | Description |
|------|--------|-------------|
| `tests/e2e/testdata/export_roundtrip.txtar` | Created | Export E2E: fixture-based backup, export to tar.gz, error paths |
| `tests/e2e/testdata/undo_after_restore.txtar` | Created | Undo E2E: git repo, commits, undo, content verification |

## Coverage Results vs Targets

| Package | Before | After | Target | Status |
|---------|--------|-------|--------|--------|
| cmd/ | 46.6% | 58.6% | 70-75% | ⚠️ Below target |
| internal/actions/ | 82.9% | 83.6% | 88-90% | ⚠️ Below target |

### Why targets weren't met
- **cmd/**: Several `*WithDeps` functions have interactive paths (TUI) that can't be unit-tested per AGENTS.md (`runLoginInteractiveWithDeps`, `runPickWithDeps`). `runRestoreWithDeps`, `runUndoWithDeps`, `runVerifyWithDeps` remain at 0-33% coverage.
- **actions/**: Improved from 82.9% to 83.6% but remaining gaps in `RunExport` (57.7%), `CreateTarGz` (66.7%), `scanBackupForSecrets` (64.7%) require more complex test setup.

## Archive Contents

- proposal.md ✅
- spec.md ✅ (delta spec — testing-only, no behavioral changes to sync)
- design.md ✅
- tasks.md ✅ (19/19 tasks complete)
- apply-progress.md ✅ (5 phases + remediation)
- verify-report.md ✅ (updated: PASS WITH WARNINGS)
- explore.md ✅

## Specs Synced

No main specs updated. This was a testing-only change with no user-facing behavioral changes. The delta spec describes testing requirements (internal quality), not capability changes.

## Lessons Learned

1. **Verify must check actual codebase, not just task checkboxes**: Tasks 3.1 and 3.2 were marked [x] in tasks.md but the tests were not in the codebase. The verify phase caught this, but it highlights that task completion claims need codebase verification.
2. **Git index corruption can cause test loss**: Phase 4's GGA hook corrupted the git index, and the recovery (`git checkout HEAD --`) silently restored files to HEAD versions, losing Phase 3 modifications. New files survived; modified files were lost.
3. **Coverage targets are aspirational**: The 70-75% cmd/ target was unrealistic given AGENTS.md constraints (no `Program.Run()` testing). The actual coverage improvement (+12% cmd/, +0.7% actions/) is meaningful even if below target.
4. **Stacked PRs work for testing changes**: 5 stacked PRs allowed incremental review and early issue detection.

## SDD Cycle Status

**COMPLETE**. The change has been planned, implemented, verified, and archived. Coverage improved meaningfully. All spec scenarios are compliant. Coverage target gaps are documented for future follow-up.
