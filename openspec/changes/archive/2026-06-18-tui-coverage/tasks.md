# Tasks: TUI Coverage Backfill

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~250 (test code only) |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | single-pr |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

## Phase 1: Move Wizard Tests (Mechanical)

- [x] 1.1 Create `internal/tui/screens/wizard_test.go` with `package screens` and required imports (`testing`, `tea`, `strings`)
- [x] 1.2 Move 16 tests from `cmd/wizard_test.go` → new file: `TestWizardModel_Init`, `TestWizardModel_StepTransitions`, `TestWizardModel_ProviderSelection`, `TestWizardModel_PresetSelection`, `TestWizardModel_AdaptersToggle`, `TestWizardModel_CategoriesToggle`, `TestWizardModel_ConfirmFlow`, `TestWizardModel_CancelFlow`, `TestWizardModel_ExitKeys`, `TestWizardModel_NameStep_*` (5 tests), `TestWizardModel_MoveCursor_*` (0 — covered by `TestMoveCursor`), `TestWizardModel_ProfileName`, `TestWizardModel_View_*` (2), `TestWizardModel_Update_WindowSize*` (2), `TestMoveCursor`
- [x] 1.3 Remove `screens.` package prefix from all moved test bodies (now same-package — direct field access)
- [x] 1.4 Reduce `cmd/wizard_test.go` to only `TestIsTTY_NotTerminal`; remove unused imports
- [x] 1.5 Run `go test ./internal/tui/screens/... ./cmd/...` — all 16+1 tests pass, `wizard.go` coverage ≥80%

## Phase 2: Backfill model.go Coverage

- [x] 2.1 Add `TestInitRestore` in `internal/tui/model_test.go` — mock `Deps.ListBackups`/`RunRestore`, verify returned model has wired closures
- [x] 2.2 Add `TestInitProfiles` — mock `Deps.ListProfiles`/`SaveProfile`, verify model state and closure injection
- [x] 2.3 Add `TestInitCloud` — cover 3 branches: `statusFn` set, `statusFn` nil, `Status` error path
- [x] 2.4 Add `TestUpdate_ScreenBackMsg` and `TestHandleKey_UnhitArms` — cover unknown msg type and per-screen key dispatch gaps
- [x] 2.5 Run `go test -coverprofile=out ./internal/tui/` — verify aggregate ≥80%

## Phase 3: Backfill screens/ Coverage

- [x] 3.1 Add `TestRestoreModel_RenderHelpers` in `internal/tui/screens/restore_test.go` — cover `renderErrorState`, `renderBackupList` (empty + populated), `renderRunning`, `renderDone` via substring assertions on `View().Content`
- [x] 3.2 Add `TestProfilesModel_RenderError` in `internal/tui/screens/profiles_test.go` — set `m.err`, call `View()`, assert error substring
- [x] 3.3 Add `TestCloudModel_Update_Disconnect` — cover error-from-`statusFn` path and `Disconnected` status branch
- [x] 3.4 Add `TestSettingsModel_Init_ReturnsNil` and `TestHealthModel_Init_ReturnsNil` — explicit nil-return coverage
- [x] 3.5 Run `go test -coverprofile=out ./internal/tui/screens/` — verify aggregate ≥80%

## Phase 4: Backfill components/ Coverage (If Needed)

- [x] 4.1 Add `TestSearch_IsActive` in `internal/tui/components/search_test.go` — call after `Activate()` and `Deactivate()`
- [x] 4.2 Add `TestToast_Update_TickExpired` in `internal/tui/components/toast_test.go` — verify dismiss on expired tick
- [x] 4.3 Run `go test -cover ./internal/tui/components/` — verify ≥80% (currently 95.1%, likely pass)

## Phase 5: Quality Gates

- [x] 5.1 `go test -race ./...` — all pass
- [x] 5.2 `go vet ./...` — clean
- [x] 5.3 `golangci-lint run` — exit 0
- [x] 5.4 Verify per-package coverage: `internal/tui/` ≥80%, `internal/tui/screens/` ≥80%, `internal/tui/components/` ≥80%, `internal/tui/styles/` ≥80%
