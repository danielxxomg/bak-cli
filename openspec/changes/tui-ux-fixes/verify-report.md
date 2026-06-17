# Verification Report — TUI UX Fixes

## Change

- **Change ID**: tui-ux-fixes
- **Project**: bak-cli
- **Mode**: openspec
- **TDD Mode**: Strict TDD (per apply-progress.md)
- **Verification Date**: 2026-06-17

## Completeness Table

| Artifact | Present | Status |
|---|---|---|
| Proposal | ✅ | Read |
| Spec | ✅ | Read |
| Design | ✅ | Read |
| Tasks | ✅ | All 22 checked [x] |
| Apply Progress | ✅ | Read |

## Build / Test / Coverage Evidence

| Command | Result | Output |
|---|---|---|
| `go test -race ./...` | ✅ PASS | 28/28 packages pass, zero failures, zero race conditions |
| `go vet ./...` | ✅ PASS | No warnings |
| `golangci-lint run ./...` | ✅ PASS | 0 issues |

### Coverage

| Package | Coverage | Threshold | Status |
|---|---|---|---|
| `internal/tui` | 91.7% | ≥80% | ✅ |
| `internal/tui/components` | 97.9% | ≥80% | ✅ |
| `internal/tui/screens` | 95.0% | ≥80% | ✅ |
| `internal/tui/styles` | 90.5% | ≥80% | ✅ |

## Spec Compliance Matrix

### tui-navigation

| Scenario | Implementation | Covering Test(s) | Status |
|---|---|---|---|
| Arrow down on main menu | `internal/tui/model.go` `handleKey` `ScreenMenu` matches `KeyDown`/`tea.KeyDown` | `TestModel_Update_ArrowKeys` "arrow down from 0 to 1" | ✅ PASS |
| Arrow up on main menu | `internal/tui/model.go` `handleKey` `ScreenMenu` matches `KeyUp`/`tea.KeyUp` | `TestModel_Update_ArrowKeys` "arrow up from 3 to 2" | ✅ PASS |
| Arrow keys on settings screen | `internal/tui/screens/settings.go` `Update` matches `'j'`, `tea.KeyDown` and `'k'`, `tea.KeyUp` | `TestSettings_Update_Navigate` "arrow down moves cursor", "arrow up moves cursor" | ✅ PASS |
| Arrow keys on dashboard | `internal/tui/model.go` forwards non-search keys to dashboard; `internal/tui/screens/dashboard.go` forwards non-`j`/`k` keys to table sub-model | No explicit arrow-key test for dashboard table | ⚠️ WARNING |
| Wrap down on menu | `internal/tui/model.go` `(m.cursor + 1) % len(menuItems)` | `TestModel_Update_NavigateDown`, `TestModel_Update_ArrowKeys` "arrow down wraps from last to first" | ✅ PASS |
| Wrap up on menu | `internal/tui/model.go` `(m.cursor - 1 + len(menuItems)) % len(menuItems)` | `TestModel_Update_NavigateUp`, `TestModel_Update_ArrowKeys` "arrow up wraps from first to last" | ✅ PASS |
| Wrap down on settings | `internal/tui/screens/settings.go` `(m.cursor + 1) % len(options)` | `TestSettings_Update_Navigate` "wraps from last to first (j)", "wraps from last to first (arrow down)" | ✅ PASS |
| Wrap up on settings | `internal/tui/screens/settings.go` `(m.cursor - 1 + len(options)) % len(options)` | `TestSettings_Update_Navigate` "wraps from first to last (k)", "wraps from first to last (arrow up)" | ✅ PASS |
| No wrap-around on dashboard table | `internal/tui/screens/dashboard.go` delegates to `bubbles/table` which clamps cursor | `TestDashboard_Update_NavigateDown`, `TestDashboard_Update_NavigateUp` | ✅ PASS |

### tui-help-bar

| Scenario | Implementation | Covering Test(s) | Status |
|---|---|---|---|
| Settings screen help bar | `internal/tui/screens/settings.go` `View` appends `components.RenderHelp` | `TestSettings_View_HelpBar`, `TestSettings_View_HelpBar_LiteralKeys` | ✅ PASS |
| Dashboard screen help bar | `internal/tui/screens/dashboard.go` `renderDashboardHelp` appended in populated/error/empty paths | `TestDashboard_View_HelpBar_Populated` | ✅ PASS |
| Health screen help bar | `internal/tui/screens/health.go` `View` uses `components.RenderHelp` for idle and completed states | `TestHealth_View_HelpBar_Idle`, `TestHealth_View_RerunFooter` | ✅ PASS |
| Cloud screen help bar | `internal/tui/screens/cloud.go` `RenderCloudStatus` appends `renderCloudHelp` | `TestRenderCloudStatus_HelpBar` | ✅ PASS |
| Dashboard empty state help bar | `internal/tui/screens/dashboard.go` `View` empty-state path appends help | `TestDashboard_View_HelpBar_Empty` | ✅ PASS |

### tui-dashboard-wiring

| Scenario | Implementation | Covering Test(s) | Status |
|---|---|---|---|
| Dashboard with backups | `cmd/root.go` wires `ListBackups: listBackups`; `listBackupsFrom` loads manifests | `TestListBackups_WithManifests`, `TestModel_View_Dashboard` | ✅ PASS |
| Dashboard with no backups | Nil guard in `internal/tui/model.go` `initDashboard`; empty-dir handling in `listBackupsFrom` | `TestListBackups_EmptyDir`, `TestModel_initDashboard_NilListBackups`, `TestDashboard_View_EmptyState` | ✅ PASS |
| Dashboard with ListBackups error | Error propagated from `listBackups` → `initDashboard` → `DashboardModel.err` | `TestModel_initDashboard_Error`, `TestDashboard_View_ErrorState` | ✅ PASS |
| ListBackups nil guard | `internal/tui/model.go` `initDashboard` returns empty result when `m.deps.ListBackups == nil` | `TestModel_initDashboard_NilListBackups` | ✅ PASS |

### tui-terminal-minimums

| Scenario | Implementation | Covering Test(s) | Status |
|---|---|---|---|
| Terminal at 40×12 | `internal/tui/styles/styles.go` `MinWidth=40`, `MinHeight=12`; root and sub-screens use `<` comparison | `TestModel_Update_MinSizeGuard` "exactly min", `TestSettings_View_MinSizeGuard`, `TestDashboard_View_MinSizeGuard`, `TestHealth_View_MinSizeGuard` | ✅ PASS |
| Terminal below minimum width | Root and sub-screens check `width < styles.MinWidth` | `TestModel_Update_MinSizeGuard` "below width", sub-model guard tests | ✅ PASS |
| Terminal below minimum height | Root and sub-screens check `height < styles.MinHeight` | `TestModel_Update_MinSizeGuard` "below height", sub-model guard tests | ✅ PASS |
| Sub-model terminal guard consistency | `settings.go`, `dashboard.go`, `health.go` all import and use `styles.MinWidth`/`styles.MinHeight` | `TestSettings_View_MinSizeGuard`, `TestDashboard_View_MinSizeGuard`, `TestHealth_View_MinSizeGuard` | ✅ PASS |

## Correctness Table

| Requirement | Implementation Location | Test Evidence | Status |
|---|---|---|---|
| Arrow key navigation (menu/settings) | `internal/tui/model.go`, `internal/tui/screens/settings.go` | `TestModel_Update_ArrowKeys`, `TestSettings_Update_Navigate` | ✅ |
| Wrap-around navigation (menu/settings) | `internal/tui/model.go`, `internal/tui/screens/settings.go` | Same as above + `TestModel_Update_NavigateDown/Up` | ✅ |
| Persistent help bar on all screens | `settings.go`, `dashboard.go`, `health.go`, `cloud.go` | Settings/dashboard/health/cloud help-bar tests | ✅ |
| `ListBackups` dependency wired | `cmd/root.go` `listBackups`/`listBackupsFrom` wired into `tui.Deps` | `TestListBackups_*`, `TestModel_View_Dashboard`, `TestModel_initDashboard_*` | ✅ |
| Less aggressive terminal size guard (40×12) | `internal/tui/styles/styles.go` constants; root + sub-screen guards | `TestModel_Update_MinSizeGuard`, sub-model min-size guard tests | ✅ |

## Design Coherence Table

| Design Decision | Implementation | Deviation | Impact | Status |
|---|---|---|---|---|
| Add `KeyDownArrow`/`KeyUpArrow` string constants | `internal/tui/keys.go` | None | — | ✅ |
| Use `msg.String()` to match `"down"`/`"up"` | `internal/tui/model.go`/`settings.go` use `msg.Code` with `tea.KeyDown`/`tea.KeyUp` instead | Implementation uses type-safe `msg.Code`; constants still defined for documentation | No spec impact; tests pass | ⚠️ Accepted deviation |
| Wrap-around via modular arithmetic | `internal/tui/model.go`, `internal/tui/screens/settings.go` | None | — | ✅ |
| Help bar via `components.RenderHelp` | All four sub-screens | None | — | ✅ |
| `ListBackups` wiring via `backup.BakDir()` + `manifest.Load()` | `cmd/root.go` | Design referenced non-existent `config.BackupDir()` and `backup.ListManifests()` | Implementation uses real helpers | ⚠️ Accepted deviation |
| Terminal minimums centralized in `styles` package | `internal/tui/styles/styles.go` `MinWidth`/`MinHeight` | None | — | ✅ |

## Issues

### CRITICAL

None.

### WARNING

1. **Dashboard arrow-key forwarding lacks explicit test coverage**
   - Spec scenario: "Arrow keys on dashboard" requires the down arrow to be forwarded to the table sub-model for row navigation.
   - Implementation correctly forwards arrow keys via the default path in `model.go` and the fall-through path in `dashboard.go`.
   - However, no test explicitly sends `tea.KeyDown`/`tea.KeyUp` to the dashboard and asserts table cursor movement. Existing `TestDashboard_Update_NavigateDown/Up` only cover `j`/`k`.
   - Recommendation: Add a narrow test case sending arrow keys to `DashboardModel.Update` and asserting cursor change to close this gap.

### SUGGESTION

1. **Design deviations are documented and harmless**: apply-progress.md already records the `msg.Code` vs `msg.String()` choice and the `backup.BakDir()`/`manifest.Load()` substitution. Both deviations are technically sound and do not break specs.

## Final Verdict

**PASS WITH WARNINGS**

All 22 tasks are implemented and checked. `go test -race ./...`, `go vet ./...`, and `golangci-lint run ./...` all pass with no failures or warnings. Coverage exceeds the 80% threshold for all TUI packages. The only remaining gap is a missing explicit test for dashboard arrow-key forwarding; the implementation is correct, but a covering test would make the spec compliance matrix fully green.
