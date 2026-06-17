# Tasks: TUI UX Fixes

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~380–420 |
| 400-line budget risk | Medium |
| Chained PRs recommended | No |
| Suggested split | Single PR (borderline; each phase is an independent commit) |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Medium

## Phase 1: Dashboard Wiring (fix empty dashboard)

- [x] 1.1 **TEST**: Add test in `cmd/root_test.go` verifying `listBackups()` returns `[]tui.BackupInfo` from a temp dir with a manifest fixture
- [x] 1.2 **IMPL**: Add `listBackups()` function in `cmd/root.go` using `backup.ListManifests` + `formatSize`; wire it into `tui.Deps{ListBackups: listBackups}`
- [x] 1.3 **VERIFY**: `go test ./cmd/...` passes; dashboard test with non-nil `ListBackups` returns rows

## Phase 2: Arrow Keys + Wrap-Around (navigation UX)

- [x] 2.1 **TEST**: Add table-driven tests in `internal/tui/model_test.go` — arrow down/up on ScreenMenu moves cursor; wrap down from last→0; wrap up from 0→last
- [x] 2.2 **TEST**: Add table-driven tests in `internal/tui/screens/settings_test.go` — same wrap-around + arrow key scenarios for SettingsModel
- [x] 2.3 **IMPL**: Add `KeyDownArrow`/`KeyUpArrow` string constants in `internal/tui/keys.go`; update `handleKey` ScreenMenu case to use `(cursor+1)%len` / `(cursor-1+len)%len` and match `"down"`/`"up"` via `msg.String()`
- [x] 2.4 **IMPL**: Update `SettingsModel.Update` in `screens/settings.go` — replace bounded cursor with modular arithmetic; add `"down"`/`"up"` cases via `msg.String()`
- [x] 2.5 **VERIFY**: `go test ./internal/tui/... ./internal/tui/screens/...` — all navigation tests green

## Phase 3: Help Bar Persistence (footer on all screens)

- [x] 3.1 **TEST**: Add tests in `screens/settings_test.go`, `dashboard_test.go`, `health_test.go`, `cloud_test.go` asserting `View()`/`RenderCloudStatus()` output contains help text from `components.RenderHelp`
- [x] 3.2 **IMPL**: Append `components.RenderHelp([]HelpKey{{"↑/↓","navigate"},{"enter","toggle"},{"q","back"}})` in `settings.go` `View()`
- [x] 3.3 **IMPL**: Add `renderHelp()` helper in `dashboard.go`; append after all three return paths (error, empty, populated)
- [x] 3.4 **IMPL**: Replace inline `"q quit • enter rerun"` in `health.go` with `components.RenderHelp` calls for idle and completed states
- [x] 3.5 **IMPL**: Append `components.RenderHelp([]HelpKey{{"q","back"}})` in `cloud.go` `RenderCloudStatus()` before return
- [x] 3.6 **VERIFY**: `go test ./internal/tui/screens/...` — help bar present in all screen outputs

## Phase 4: Terminal Minimums (40×12 threshold)

- [x] 4.1 **TEST**: Add tests in `model_test.go` — terminal at 40×12 renders normally; 39×12 and 40×11 show "Terminal too small"
- [x] 4.2 **IMPL**: Add `MinWidth = 40` and `MinHeight = 12` constants in `internal/tui/styles/styles.go`; update `model.go` to use `styles.MinWidth`/`styles.MinHeight`
- [x] 4.3 **IMPL**: Update sub-screen guards in `settings.go`, `dashboard.go`, `health.go` — replace hardcoded `20`/`10` with `styles.MinWidth`/`styles.MinHeight`
- [x] 4.4 **VERIFY**: `go test ./internal/tui/...` — threshold tests pass; no regressions

## Phase 5: Quality Gates

- [x] 5.1 Run `go test -race ./...` — all tests pass (28/28 packages, zero failures)
- [x] 5.2 Run `go vet ./...` — no warnings
- [x] 5.3 Run `golangci-lint run` — 0 issues (fixed 3 goimports formatting issues)
- [x] 5.4 Verify coverage ≥80% for `internal/tui/` and `internal/tui/screens/` (91.7% / 95.0%)
