## Verification Report

**Change**: coverage-improvement
**Version**: N/A
**Mode**: Standard

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 19 |
| Tasks complete | 17 (claimed) |
| Tasks incomplete | 2 (actual) |

> **Note**: Tasks 3.1 and 3.2 are marked complete in `tasks.md` but the corresponding tests are **not present** in the codebase.

### Build & Tests Execution

**Build**: ✅ Passed
```text
$ go build ./...
Success
```

**Tests**: ✅ 1235 passed / 0 failed / 0 skipped
```text
$ go test ./... -count=1
ok  	github.com/danielxxomg/bak-cli/cmd				7.971s
ok  	github.com/danielxxomg/bak-cli/internal/actions		2.851s
ok  	github.com/danielxxomg/bak-cli/tests/e2e			11.000s
...
Go test: 1235 passed in 26 packages
```

**Go vet**: ✅ Passed
```text
$ go vet ./...
Go vet: No issues found
```

**E2E tests**: ✅ 11 passed
```text
$ go test ./tests/e2e/ -count=1
Go test: 11 passed in 1 packages
```

### Coverage Evidence

**cmd/ coverage**: 58.6% (target: 70–75%) ❌ **Below target**
```text
$ go test ./cmd/... -cover
coverage: 58.6% of statements
```

**actions/ coverage**: 82.9% (target: 88–90%) ❌ **Below target**
```text
$ go test ./internal/actions/... -cover
coverage: 82.9% of statements
```

### Spec Compliance Matrix

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| cmd/ wrapper tests | Delegation and error propagation | `cmd/backup_test.go > TestRunBackup_InvalidInputs` | ✅ COMPLIANT |
| cmd/ wrapper tests | Push with mock factory | `cmd/push_test.go > TestRunPushWithDeps_Delegation` | ⚠️ PARTIAL (tests error only; mock factory not injected at cmd/ level per design) |
| cmd/ TUI guards | Non-TTY error | `cmd/login_test.go > TestRunLoginWithDeps_NonTTYGuard` | ✅ COMPLIANT |
| cmd/ TUI guards | Non-TTY error | `cmd/pick_test.go > TestRunPickWithDeps_NonTTYGuard` | ✅ COMPLIANT |
| cmd/ bubbletea model tests | Picker toggle | `cmd/pick_test.go > TestPickModel_Update_Toggle` | ✅ COMPLIANT |
| cmd/ bubbletea model tests | Picker View | `cmd/pick_test.go > TestPickModel_View` | ✅ COMPLIANT |
| cmd/ bubbletea model tests | Wizard navigation | `cmd/wizard_test.go > TestWizardModel_StepTransitions` | ✅ COMPLIANT |
| cmd/ bubbletea model tests | Wizard View | `cmd/wizard_test.go > TestWizardModel_View_ContainsTitle` | ✅ COMPLIANT |
| actions/ error path tests | saveManifest write failure | `internal/actions/backup_test.go > TestBackupAction_SaveManifestError` | ✅ COMPLIANT |
| actions/ error path tests | scanBackupForSecrets fixture error | `internal/actions/backup_test.go > TestBackupAction_ScanBackupForSecrets_WalkError` | ✅ COMPLIANT |
| actions/ error path tests | RunExport create error | (none found) | ❌ UNTESTED |
| actions/ error path tests | CreateTarGz gzip close error | (none found) | ❌ UNTESTED |
| actions/ FormatSizeBytes edge cases | Boundary values | `internal/actions/list_local_test.go > TestFormatSizeBytes` | ✅ COMPLIANT |
| actions/ FormatSizeBytes edge cases | Large value | `internal/actions/list_local_test.go > TestFormatSizeBytes` (1 TB case) | ✅ COMPLIANT |
| E2E export and undo coverage | Export roundtrip | `tests/e2e/testdata/export_roundtrip.txtar` | ✅ COMPLIANT |
| E2E export and undo coverage | Undo after restore | `tests/e2e/testdata/undo_after_restore.txtar` | ✅ COMPLIANT |
| AGENTS.md testing compliance | No `Program.Run()` in tests | grep across `*_test.go` | ✅ COMPLIANT |
| AGENTS.md testing compliance | No `os.Exit` in tests | grep across `*_test.go` | ✅ COMPLIANT |

**Compliance summary**: 14/18 scenarios fully compliant, 2 partial, 2 untested

### Correctness (Static Evidence)

| Requirement | Status | Notes |
|------------|--------|-------|
| `isTTY` extraction | ✅ Implemented | `cmd/wizard.go:330` — package-level `var isTTY` |
| `runBackupWithDeps` test | ✅ Implemented | `cmd/backup_test.go` — mock ConfigLoader error |
| `runLoginWithDeps` test | ✅ Implemented | `cmd/login_test.go` — config error + non-TTY guard |
| `runPickWithDeps` test | ⚠️ Partial | `cmd/pick_test.go` — only non-TTY guard (12.5% coverage) |
| `runPushWithDeps` test | ✅ Implemented | `cmd/push_test.go` — delegation verified |
| `runPullWithDeps` test | ✅ Implemented | `cmd/pull_test.go` — delegation verified |
| `pickModel` Update/View tests | ✅ Implemented | `cmd/pick_test.go` — space toggle, cursor, quit, confirm |
| `wizardModel` Update/View tests | ✅ Implemented | `cmd/wizard_test.go` — step transitions, navigation, Ctrl+C |
| `RunExport` happy path | ✅ Implemented | `internal/actions/export_test.go` |
| `RunExport` invalid ID | ✅ Implemented | `internal/actions/export_test.go` |
| `RunExport` create error | ❌ Missing | `writeFailingFS` test for `os.Create` failure not found |
| `CreateTarGz` gzip close error | ❌ Missing | Pipe-based writer test not found |
| `saveManifest` write failure | ✅ Implemented | `internal/actions/backup_test.go > TestBackupAction_SaveManifestError` |
| `scanBackupForSecrets` walk error | ✅ Implemented | `internal/actions/backup_test.go > TestBackupAction_ScanBackupForSecrets_WalkError` |
| `FormatSizeBytes` edge cases | ✅ Implemented | `internal/actions/list_local_test.go` — 0, 1024, 1 MB, 1 GB, 1 TB, etc. |
| E2E export roundtrip | ✅ Implemented | `tests/e2e/testdata/export_roundtrip.txtar` |
| E2E undo after restore | ✅ Implemented | `tests/e2e/testdata/undo_after_restore.txtar` |

### Coherence (Design)

| Decision | Followed? | Notes |
|----------|-----------|-------|
| `isTTY` extraction to package-level var | ✅ Yes | `cmd/wizard.go:330` — matches `var execCommand` pattern |
| Provider Factory testing deferred to E2E | ✅ Yes | `cmd/` does not import `ProviderFactory`; E2E covers full wiring |
| Bubbletea models tested as pure functions | ✅ Yes | `Update()`/`View()` only; no `tea.Program.Run()` in tests |
| Test doubles hand-rolled | ✅ Yes | `MockFileSystem`, `writeFailingFS`, `mkdirFailingFS` used |
| Error wrapping with `fmt.Errorf` | ✅ Yes | All errors use `%w` formatting |
| Table-driven tests | ✅ Yes | `TestFormatSizeBytes`, `TestRunListLocal_*`, etc. |

### Issues Found

**CRITICAL**:
1. **`cmd/` coverage below target**: `cmd/` coverage is 58.6% (design target: 70–75%). `runPickWithDeps` (12.5%), `runLoginInteractiveWithDeps` (10.0%), `runRestoreWithDeps` (0.0%), `runUndoWithDeps` (33.3%), `runVerifyWithDeps` (0.0%) remain significantly uncovered.
2. **`actions/` coverage below target**: `actions/` coverage is 82.9% (design target: 88–90%). `FormatSizeBytes` (25.0% — only the new table-driven test covers it, but coverage report shows low), `RunExport` (57.7%), `CreateTarGz` (66.7%), `scanBackupForSecrets` (64.7%) are under-covered.
3. **`RunExport` create error test MISSING**: Task 3.1 claims a `writeFailingFS` test for `RunExport` create error was added, but `internal/actions/export_test.go` contains no such test. The file only tests happy path, backup not found, invalid ID, and edge cases.
4. **`CreateTarGz` gzip close error test MISSING**: Task 3.2 claims a pipe-based writer test that fails on `Close` was added, but `internal/actions/export_test.go` contains no `CreateTarGz` error test at all.

**WARNING**:
1. **`runPickWithDeps` TTY path untested**: The non-TTY guard is tested, but the interactive path (which calls `tea.NewProgram(m).Run()`) is not covered by unit tests. This is by design per AGENTS.md, but it leaves 87.5% of the function uncovered.
2. **`runPushWithDeps` mock factory scenario**: The spec scenario "Push with mock factory" is not testable at the `cmd/` level because `runPushWithDeps` creates `RealProviderFactory` internally; the test only verifies error delegation. The design explicitly chose this boundary.
3. **`runPickWithDeps` calls `tea.NewProgram().Run()` in production**: While AGENTS.md forbids testing `Program.Run()`, the production code in `cmd/pick.go:151` still calls it. This is acceptable for production but means the interactive path cannot be unit-tested.

**SUGGESTION**:
1. Add the missing `RunExport` create error test by injecting a `FileSystem` mock that fails `Create` (or use a read-only directory path) to cover the `os.Create` error path.
2. Add the missing `CreateTarGz` gzip close error test by passing a custom `io.Writer` that returns an error on `Close` (e.g., `type failOnCloseWriter struct{ io.Writer }`) to exercise the `gw.Close()` error path.
3. Consider adding a `cmd/` test for `runRestoreWithDeps` and `runUndoWithDeps` using the same mock-`cmdDeps` pattern to boost `cmd/` coverage toward the 70% target.
4. Verify why `FormatSizeBytes` shows only 25% coverage despite the comprehensive table-driven test — the function may be defined in multiple files or the coverage profile may be outdated.

### Verdict

**FAIL**

Two spec scenarios (`RunExport` create error, `CreateTarGz` gzip close error) are untested despite being marked complete in tasks. `cmd/` and `actions/` coverage improved but did not reach the design targets (58.6% vs 70–75% and 82.9% vs 88–90%). All build, vet, and test commands pass cleanly. No AGENTS.md violations (no `Program.Run()` or `os.Exit` in tests).

---

**Status**: partial
**Summary**: Verification completed for `coverage-improvement`. Build, tests, and E2E pass. Coverage improved for `cmd/` (46.6% → 58.6%) but missed the 70–75% target. `actions/` coverage did not reach the 88–90% target. Two claimed tests (`RunExport` create error, `CreateTarGz` gzip close error) are missing from the codebase.
**Artifacts**: `openspec/changes/coverage-improvement/verify-report.md`
**Next**: sdd-apply (to add missing tests) or manual fix
**Risks**: Missing error-path tests may leave regression gaps in `export` and `CreateTarGz`.
**Skill Resolution**: paths-injected — 2 skills requested (sdd-verify, golang-pro); golang-pro not found at path.
