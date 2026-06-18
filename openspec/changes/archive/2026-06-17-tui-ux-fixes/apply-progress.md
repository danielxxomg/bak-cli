# Apply Progress — TUI UX Fixes

**Phases**: 1 (Dashboard Wiring) + 2 (Arrow Keys + Wrap-Around) + 3 (Help Bar Persistence) + 4 (Terminal Minimums)
**Date**: 2026-06-17
**Mode**: Strict TDD

## TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|-------|------------|-----|-------|-------------|----------|
| 1.1 | `cmd/root_test.go` | Unit | ✅ all cmd passing | ✅ Written (compile fail) | ✅ 4/4 passed | ✅ 4 cases | ➖ Clean |
| 1.2 | `cmd/root.go` | Unit | N/A (new) | ✅ Written | ✅ 4/4 passed | ✅ 4 cases | ➖ Clean |
| 1.3 | Full suite | Unit | ✅ all cmd passing | N/A (verify) | ✅ all passing | N/A | N/A |
| 2.1 | `internal/tui/model_test.go` | Unit | ✅ 32 tests passing | ✅ Written (arrows not handled + clamp not wrap) | ✅ 10/10 passed | ✅ 8 cases (arrows + j/k + mixed + wrap both directions) | ➖ Clean |
| 2.2 | `internal/tui/screens/settings_test.go` | Unit | ✅ 32 tests passing | ✅ Written (arrows not handled + clamp not wrap) | ✅ 8/8 passed | ✅ 8 cases (arrows + j/k wrap both directions) | ➖ Clean |
| 2.3 | `internal/tui/keys.go`, `model.go` | Unit | N/A (incremental) | ✅ Applied (updated clamp→wrap + added arrow support) | ✅ 10/10 passed | ✅ 8 cases (same as 2.1) | ➖ Clean |
| 2.4 | `internal/tui/screens/settings.go` | Unit | N/A (incremental) | ✅ Applied (updated clamp→wrap + added arrow support) | ✅ 8/8 passed | ✅ 8 cases (same as 2.2) | ➖ Clean |
| 2.5 | Full suite | Unit | ✅ all passing | N/A (verify) | ✅ all passing, `go vet` clean | N/A | N/A |
| 3.1 | `screens/settings_test.go`, `dashboard_test.go`, `health_test.go`, `cloud_test.go` | Unit | ✅ all screens passing | ✅ Written (help bar not in any screen views) | ✅ 9/9 passed (3 settings + 3 dashboard×3 + 2 health + 1 cloud) | ✅ 2 triangulation (settings literal keys + cloud no-provider) | ➖ Clean |
| 3.2 | `internal/tui/screens/settings.go` | Unit | N/A (incremental) | ✅ Applied (added RenderHelp in View) | ✅ 3/3 passed | ✅ 2 cases (settings help + literal keys) | ➖ Clean |
| 3.3 | `internal/tui/screens/dashboard.go` | Unit | N/A (incremental) | ✅ Applied (added renderDashboardHelp + components import) | ✅ 3/3 passed (populated, empty, error) | ✅ 3 cases (all three return paths) | ➖ Clean |
| 3.4 | `internal/tui/screens/health.go` | Unit | N/A (incremental) | ✅ Applied (replaced inline q quit with RenderHelp; added idle help bar + components import) | ✅ 2/2 passed (idle + done) | ✅ 2 cases (idle and completed states) | ➖ Clean |
| 3.5 | `internal/tui/screens/cloud.go` | Unit | N/A (incremental) | ✅ Applied (restructured to use strings.Builder + renderCloudHelp helper + components import) | ✅ 2/2 passed (provider + no-provider) | ✅ 2 cases (both return paths) | ➖ Clean |
| 3.6 | Full suite | Unit | ✅ all passing | N/A (verify) | ✅ all 30 packages passing, `go vet` clean | N/A | N/A |
| 4.1 | `internal/tui/model_test.go`, `screens/settings_test.go`, `dashboard_test.go`, `health_test.go` | Unit | ✅ 64 tui + 66 screens passing | ✅ Written (16 failures: 4 model MinSizeGuard + 4 TooSmall dims + 2 TooSmallWarning + 6 sub-model guards) | ✅ 23/23 passed (8 model MinSizeGuard + 5 dashboard + 5 health + 5 settings) | ✅ 23 cases (8+5+5+5 across 4 test files; 5 threshold boundaries per sub-model) | ✅ Clean (removed orphan init(), updated comments) |
| 4.2 | `internal/tui/styles/styles.go`, `internal/tui/model.go` | Unit | N/A (incremental) | ✅ Applied (added MinWidth=40/MinHeight=12 in styles.go; updated model.go to use styles.MinWidth/MinHeight + dimensional message) | ✅ all passing | ✅ all threshold cases pass | ✅ Clean |
| 4.3 | `internal/tui/screens/settings.go`, `dashboard.go`, `health.go` | Unit | N/A (incremental) | ✅ Applied (replaced hardcoded 20/10 with styles.MinWidth/MinHeight in all 3 sub-screens) | ✅ all passing | ✅ 3 sub-models consistent | ✅ Clean |
| 4.4 | Full suite | Unit | ✅ all passing | N/A (verify) | ✅ all 28 packages passing, `go vet` clean, `-race` clean | N/A | N/A |

## Test Summary

- **Total tests written**: 4 (Phase 1) + 18 (Phase 2) + 9 (Phase 3) + 23 (Phase 4) = 54 test cases
- **Total tests passing**: 54 (all phases) + existing = 133 total (64 tui + 69 screens)
- **Layers used**: Unit (54)
- **Approval tests** (refactoring): Phase 2: existing j/k clamp tests updated to assert wrap-around. Phase 3: existing `TestHealth_View_RerunFooter` extended to also assert "back" in help bar. Phase 4: existing `TestModel_Update_MinSizeGuard` table extended from 5→8 cases with 40×12 thresholds; `TestModel_View_TooSmall` extended to assert dimensional message.
- **Pure functions created**: 1 (`listBackupsFrom`)

## Completed Tasks

### Phase 1
- [x] 1.1 TEST: Added 4 tests in `cmd/root_test.go`
- [x] 1.2 IMPL: Added `listBackups()` + `listBackupsFrom()` in `cmd/root.go`
- [x] 1.3 VERIFY: `go test -race ./...` all pass

### Phase 2
- [x] 2.1 TEST: Added 8 table-driven test cases in `internal/tui/model_test.go` — arrow down/up moves cursor (3 cases), j/k wrap-around (2 cases), arrow wrap-around (2 cases), mixed j/k+arrows (1 case); updated existing `TestModel_Update_NavigateDown` and `TestModel_Update_NavigateUp` to assert wrap-around instead of clamp
- [x] 2.2 TEST: Added 8 table-driven test cases in `internal/tui/screens/settings_test.go` — arrow down/up (2 cases), j/k+arrow wrap-around (4 cases), basic j/k (2 cases); replaced old clamp assertions with wrap-around assertions
- [x] 2.3 IMPL: Added `KeyDownArrow`/`KeyUpArrow` string constants in `internal/tui/keys.go`; updated `handleKey` ScreenMenu case to use `(cursor+1)%len` / `(cursor-1+len)%len` modular arithmetic; added `tea.KeyDown`/`tea.KeyUp` alongside `KeyDown`/`KeyUp` for arrow key handling via `msg.Code`
- [x] 2.4 IMPL: Updated `SettingsModel.Update` in `screens/settings.go` — replaced bounded cursor with modular arithmetic; added `tea.KeyDown`/`tea.KeyUp` alongside `'j'`/`'k'`
- [x] 2.5 VERIFY: `go test -race ./internal/tui/... ./internal/tui/screens/...` all green; `go vet ./...` clean

### Phase 3
- [x] 3.1 TEST: Added 9 new test assertions across 4 test files: `TestSettings_View_HelpBar` + `TestSettings_View_HelpBar_LiteralKeys` (settings), `TestDashboard_View_HelpBar_Populated` + `TestDashboard_View_HelpBar_Empty` + `TestDashboard_View_HelpBar_Error` (dashboard), `TestHealth_View_HelpBar_Idle` + extended `TestHealth_View_RerunFooter` (health), `TestRenderCloudStatus_HelpBar` + `TestRenderCloudStatus_HelpBar_NoProvider` (cloud)
- [x] 3.2 IMPL: Appended `components.RenderHelp` with `↑/↓ navigate • enter toggle • q back` in `settings.go` View()
- [x] 3.3 IMPL: Added `renderDashboardHelp()` helper in `dashboard.go` using `components.HelpKey`; appended after all three return paths (error, empty, populated) with `↑/↓ navigate • / search • q back`
- [x] 3.4 IMPL: Replaced inline `"q quit • enter rerun"` in `health.go` with `components.RenderHelp` for both idle state (`enter run • q back`) and completed state (`q back • enter rerun`)
- [x] 3.5 IMPL: Restructured `cloud.go` `RenderCloudStatus` to use `strings.Builder`; added `renderCloudHelp()` helper; appended `q back` help bar in both provider and no-provider paths
- [x] 3.6 VERIFY: `go test -race ./...` all 30 packages pass; `go vet ./...` clean

### Phase 4
- [x] 4.1 TEST: Updated `TestModel_Update_MinSizeGuard` from 5 cases (20×10 thresholds) to 8 cases (40×12 thresholds) with 4 new boundary cases (barely below both, wide but short, tall but narrow). Updated `TestModel_View_TooSmall` and `TestModel_View_TooSmall_ShowsWarningOnly` to assert dimensional message format ("Terminal too small (NxM). Need at least 40×12."). Added `TestDashboard_View_MinSizeGuard`, `TestHealth_View_MinSizeGuard`, `TestSettings_View_MinSizeGuard` table-driven tests (5 threshold cases each) in 3 sub-model test files.
- [x] 4.2 IMPL: Added `MinWidth = 40` and `MinHeight = 12` exported constants in `internal/tui/styles/styles.go` (single source of truth, importable by both `tui` and `screens` packages). Updated `model.go` to use `styles.MinWidth`/`styles.MinHeight` in WindowSizeMsg handler and dimensional `fmt.Sprintf("Terminal too small (%dx%d). Need at least %dx%d.", ...)` message in View().
- [x] 4.3 IMPL: Updated sub-screen guards in `settings.go` (line 81), `dashboard.go` (line 148), and `health.go` (line 125) — all changed from hardcoded `m.width < 20 || m.height < 10` to `m.width < styles.MinWidth || m.height < styles.MinHeight`.
- [x] 4.4 VERIFY: `go test -race -count=1 ./...` — all 28 packages pass. `go vet ./...` — clean, zero warnings.

## Files Changed

| File | Action | What Was Done |
|------|--------|---------------|
| `cmd/root.go` | Modified | Added `listBackups()` and `listBackupsFrom()` functions (~55 new lines); wired `ListBackups: listBackups` into `tui.Deps` |
| `cmd/root_test.go` | Modified | Added 4 tests covering manifest scanning, empty/no-backup dirs, non-dir entries, corrupt manifest (~110 new lines) |
| `internal/tui/keys.go` | Modified | Added `KeyDownArrow = "down"` and `KeyUpArrow = "up"` string constants (documentation + future `msg.String()` matching) |
| `internal/tui/model.go` | Modified | ScreenMenu: clamp→modular arithmetic + arrow keys. Terminal minimums: replaced `minWidth`/`minHeight` constants with `styles.MinWidth`/`styles.MinHeight`; dimensional too-small message. Added `fmt` and `styles` imports. |
| `internal/tui/model_test.go` | Modified | Arrow keys + wrap-around (8 new cases + 2 updated). Phase 4: `TestModel_Update_MinSizeGuard` expanded 5→8 cases with 40×12 thresholds; `TestModel_View_TooSmall` extended with dimensional assertions; `TestModel_View_TooSmall_ShowsWarningOnly` extended similarly; `TestModel_View_AltScreen` "minimum viable" updated to 40×12. |
| `internal/tui/screens/settings.go` | Modified | Update: clamp→modular arithmetic + arrow keys. View: help bar added. Phase 4: guard changed from `20/10` to `styles.MinWidth`/`styles.MinHeight`. |
| `internal/tui/screens/settings_test.go` | Modified | Replaced 4 old tests with 8 arrow+wrap tests. Added 2 help bar tests. Phase 4: added `TestSettings_View_MinSizeGuard` (5 threshold cases). |
| `internal/tui/screens/dashboard.go` | Modified | Added `components` import + `renderDashboardHelp()` + help bar on all 3 return paths. Phase 4: guard changed from `20/10` to `styles.MinWidth`/`styles.MinHeight`. |
| `internal/tui/screens/dashboard_test.go` | Modified | Added 3 help bar tests. Phase 4: added `TestDashboard_View_MinSizeGuard` (5 threshold cases). |
| `internal/tui/screens/health.go` | Modified | Added `components` import; replaced inline with `RenderHelp` for both idle and completed states. Phase 4: guard changed from `20/10` to `styles.MinWidth`/`styles.MinHeight`. |
| `internal/tui/screens/health_test.go` | Modified | Extended `TestHealth_View_RerunFooter`; added `TestHealth_View_HelpBar_Idle`. Phase 4: added `TestHealth_View_MinSizeGuard` (5 threshold cases). |
| `internal/tui/screens/cloud.go` | Modified | Restructured to `strings.Builder`; added `renderCloudHelp()` helper; appended help bar in both paths. |
| `internal/tui/screens/cloud_test.go` | Modified | Added 2 help bar tests (provider + no-provider). |
| `internal/tui/styles/styles.go` | Modified | Phase 4: added exported `MinWidth = 40` and `MinHeight = 12` constants (single source of truth for terminal size guards). |
| `openspec/changes/tui-ux-fixes/tasks.md` | Modified | Marked Phase 3 and Phase 4 tasks [x]. |
| `cmd/root_test.go` | Modified | goimports formatting fix (Phase 5). |
| `internal/tui/model_test.go` | Modified | goimports formatting fix (Phase 5). |
| `internal/tui/screens/settings.go` | Modified | goimports formatting fix (Phase 5). |

## Deviations from Design

### Phase 1
- **`config.BackupDir()` vs `backup.BakDir()`**: Design referenced non-existent `config.BackupDir()`. Used `backup.BakDir()`.
- **`backup.ListManifests` vs `manifest.Load`**: Design referenced non-existent `backup.ListManifests()`. Used `manifest.Load()`.

### Phase 2
- **`msg.String()` vs `msg.Code` for arrow keys**: Design specified using `msg.String()` and matching on string `"down"`/`"up"`. Used `msg.Code` with `tea.KeyDown`/`tea.KeyUp` rune constants instead. Rationale: `msg.Code` is type-safe, matches existing code patterns (no switch restructuring needed), and avoids edge cases with `msg.String()` (e.g., space→"space", enter→"enter", requiring all existing cases to be rewritten). Both approaches are valid bubbletea v2 patterns; `msg.Code` is actually the recommended "more foolproof" approach per bubbletea docs.

### Phase 3
- **Cloud `RenderCloudStatus` restructuring**: Design showed appending help bar to the returned `content` variable via `content += "\n\n" + components.RenderHelp(...)`. Refactored the function to use `strings.Builder` consistently (matching the existing pattern used in other screens) and added a `renderCloudHelp()` helper. The help bar is appended to both the no-provider and provider paths before the final Frame wrapping. This improves consistency with the rest of the codebase.
- **Health idle view now appends help bar**: Design for health showed separate help bar blocks for idle and completed states. The idle state previously returned early with just the prompt. Now the prompt is followed by the help bar before returning, matching the design's intent.

### Phase 4
None — implementation matches design exactly.

### Phase 5
None — quality gates only; no implementation changes.

## Issues Found

### Phase 5
- **goimports formatting**: 3 files had improper import ordering (`cmd/root_test.go:605`, `internal/tui/model_test.go:271`, `internal/tui/screens/settings.go:90`). Fixed by running `goimports -w`.

## Remaining Tasks

None — all 5 phases complete.

### Phase 5
- [x] 5.1 `go test -race ./...`: ✅ 28/28 packages pass, zero failures, zero race conditions
- [x] 5.2 `go vet ./...`: ✅ zero warnings
- [x] 5.3 `golangci-lint run`: ✅ 0 issues (fixed 3 goimports formatting violations in `cmd/root_test.go`, `internal/tui/model_test.go`, `internal/tui/screens/settings.go`)
- [x] 5.4 Coverage: `internal/tui/` 91.7%, `internal/tui/components/` 97.9%, `internal/tui/screens/` 95.0%, `internal/tui/styles/` 90.5% — all ≥80%

## Quality Gates

- `go test -race ./...`: ✅ all 28 packages pass
- `go vet ./...`: ✅ zero warnings
- `golangci-lint run`: ✅ 0 issues
- Coverage (internal/tui/): 91.7% (≥80% ✅)
- Coverage (internal/tui/screens/): 95.0% (≥80% ✅)

## Workload / PR Boundary

- Mode: ask-on-risk (single PR, Medium risk)
- Current work unit: Phase 5 — Quality Gates
- Boundary: 3 files fixed (goimports formatting), zero production logic changes
- Cumulative: 20 files changed, ~388 lines total (all 5 phases)
- Estimated review budget impact: zero new logic (formatting-only fixes in Phase 5)
