# Archive Report — test-hardening

**Change**: Test Hardening — Fix Misleading Testscripts and Close Coverage Gaps
**Date**: 2026-06-16
**Verdict**: PASS WITH WARNINGS
**Archive location**: `openspec/changes/test-hardening/` (retained in active changes per user instruction)

---

## Change Summary

Four test gaps creating a false sense of security were identified and fixed:

1. **Misleading testscripts** (HIGH RISK) — Three testscripts (`backup_restore_roundtrip`, `diff_two_backups`, `backup_verify_roundtrip`) were named after scenarios they never executed. Rewritten in-place to exercise real `restore`, `diff`, and `verify` commands.
2. **Schedule happy path** (MEDIUM RISK) — Schedule cobra→action wiring had zero happy-path coverage at the cmd level. Added 3 cmd-level tests with mock scheduler injection via extended `cmdDeps.NewScheduler`.
3. **Cloud sync integration** (MEDIUM RISK) — No test exercised the full push→pull cycle at the action level. Added `cloud_sync_test.go` with table-driven round-trip tests using `MockProvider`.
4. **TUI launch smoke** (LOW RISK) — Binary had never been launched as a process. Added `smoke_test.go` with 3 tests proving the binary launches and handles `--help`, no-args, and unknown commands.

## Files Created/Modified

| File | Action | Description |
|------|--------|-------------|
| `tests/e2e/testdata/backup_restore_roundtrip.txtar` | Modified | Real restore: --dry-run diff, delete files, --force restore, exists checks, invalid-ID error |
| `tests/e2e/testdata/diff_two_backups.txtar` | Modified | Real diff: two backups with different content, Modified/Unchanged categories, identical comparison, invalid-ID error |
| `tests/e2e/testdata/backup_verify_roundtrip.txtar` | Modified | Real verify: success with checkmark, tampered-file hash mismatch detection, invalid-ID error |
| `cmd/deps.go` | Modified | Added `NewScheduler func() schedule.Scheduler` field to `cmdDeps` (nil → production default) |
| `cmd/schedule.go` | Modified | Passed `deps.NewScheduler` through to `ScheduleAction.NewScheduler` in create/list/remove |
| `cmd/schedule_test.go` | Modified | Added `cmdMockScheduler` + 3 happy-path tests (create/list/remove) |
| `internal/actions/cloud_sync_test.go` | Created | 3 integration tests: PushPullRoundTrip (table-driven, 3 sub-cases), PushInvalidToken, Pull_NotFound |
| `tests/e2e/smoke_test.go` | Created | 3 E2E smoke tests: BinaryHelp, BinaryNoArgs, BinaryUnknownCommand |

## Warnings Documented

### WARNING 1: Partial spec compliance — schedule config mutation not asserted at cmd level
`TestScheduleCreate_HappyPath` and `TestScheduleRemove_HappyPath` verify mock scheduler invocation and stdout confirmation but do NOT assert that the profile's `Schedule` config is updated in-memory (`Enabled: true, Interval: "daily"` on create; `nil` on remove). The mutation IS implemented in `internal/actions/schedule.go` and covered by `internal/actions/schedule_test.go`, but the cmd-level spec scenarios include these assertions.

**Severity**: WARNING — behavior is correct, assertion is missing at one layer.
**Recommendation**: Strengthen cmd-level tests to assert `cfg.Profiles["work"].Schedule` state after action runs.

### WARNING 2: Partial spec compliance — TUI --help asserts Long description, not Short
The spec says `bak --help` stdout MUST contain `"Backup and restore your AI coding setup"` (the Short description). Cobra's default help template renders the Long description, so the test asserts on `"packs, restores, and syncs your OpenCode configuration"` instead. The help banner IS verified and exit code is 0.

**Severity**: WARNING — spec-design mismatch, not a code bug.
**Recommendation**: Update spec text to reference the Long description that Cobra actually renders.

### WARNING 3: Design deviation — cloud sync uses MockProvider instead of HTTP server
The spec/proposal reference `setupMockGistAPI` (HTTP server). The implementation uses `MockProvider`/`MockProviderFactory` as chosen in `design.md`. The action-level push→pull behavior is fully exercised.

**Severity**: WARNING — design-consistent deviation, no behavior broken.

## Lessons Learned

### 1. Cobra Long vs Short description in help output
Cobra's default `--help` template renders the `Long` description field, NOT the `Short`. When writing specs that assert on help output, always check which field Cobra actually renders. The `Short` description appears in command listings (`bak --help` parent), while `Long` appears in the command's own help page.

### 2. Testscript regex capture limitations
The `rogpeppe/go-internal` testscript package does NOT support capturing stdout regex groups into environment variables. To extract values like backup IDs from command output, use shell-based workarounds: pipe to a temp file (`$WORK/bak_id.txt`) and read it back with `cat`.

### 3. Backup ID collision within the same second
Backup IDs are timestamp-based with second precision. Two backups created in the same second get identical IDs. In testscripts that create multiple backups, use `sleep 1` between backup commands to ensure distinct IDs. This is a known limitation of the timestamp-based ID scheme.

### 4. Backup path structure uses adapter names
Files in backups are stored under `<adapterName>/<relPath>` (e.g., `opencode/opencode.json`), not under a generic `config/` prefix. Tests asserting on backup content must use the correct adapter-prefixed paths.

## Task Completion

18/18 tasks complete across 5 phases:
- Phase 1 (Fix Misleading Testscripts): 5/5 ✅
- Phase 2 (Schedule Happy Path): 6/6 ✅
- Phase 3 (Cloud Sync Integration): 4/4 ✅
- Phase 4 (TUI Smoke Test): 4/4 ✅
- Phase 5 (Quality Gates): 3/3 ✅

## Verification Evidence

- **Build**: ✅ Clean compile
- **Tests**: ✅ 30 packages pass with race detector, 0 failures
- **go vet**: ✅ Clean
- **golangci-lint**: ✅ 0 issues (1 goimports fix applied during apply)
- **Coverage**: Changed files at 90.6% average (deps.go 100%, schedule.go 100%, push.go 82.5%, pull.go 80.0%)
- **TDD compliance**: 6/6 checks passed
- **Spec compliance**: 15/18 fully compliant, 3/18 partial (warnings above)

## Artifacts

| Artifact | File | Status |
|----------|------|--------|
| Proposal | `openspec/changes/test-hardening/proposal.md` | ✅ Complete |
| Spec | `openspec/changes/test-hardening/spec.md` | ✅ Complete |
| Design | `openspec/changes/test-hardening/design.md` | ✅ Complete |
| Tasks | `openspec/changes/test-hardening/tasks.md` | ✅ 18/18 complete |
| Apply Progress | `openspec/changes/test-hardening/apply-progress.md` | ✅ Complete |
| Verify Report | `openspec/changes/test-hardening/verify-report.md` | ✅ PASS WITH WARNINGS |
| Archive Report | `openspec/changes/test-hardening/archive-report.md` | ✅ This file |

## SDD Cycle Status

**COMPLETE** — The change has been fully planned, implemented, verified, and archived.
No delta specs to sync (test-only change, no `specs/` subdirectory).
Change directory retained in `openspec/changes/test-hardening/` per user instruction.
