# Tasks: TUI Personality

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~380 (3 PRs: ~150 / ~200 / ~30) |
| 400-line budget risk | Low |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (Tier 1) → PR 2 (Tier 2) → PR 3 (Tier 3) |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Window title + spinner indicators + status bar + gradient logo | PR 1 | base: main; ~150 lines; tests included |
| 2 | Viewport dry-run + mouse navigation + styled empty states | PR 2 | base: main (after PR 1 merge); ~200 lines; depends on PR 1 status bar |
| 3 | Paste support in wizard | PR 3 | base: main (after PR 2 merge); ~30 lines; independent |

## Phase 1: Window Title & Spinner Indicators (PR 1 — Tier 1a)

- [x] 1.1 [RED] `internal/tui/model_test.go`: table-driven test asserting `View().WindowTitle` per screen (Menu→`bak — Main Menu`, Wizard→`bak — Wizard`, Progress w/ step 3/7→`bak — Backup 3/7`, Restore w/ id→contains id)
- [x] 1.2 [GREEN] `internal/tui/model.go`: add `titleForScreen(m Model) string` pure helper; set `v.WindowTitle = titleForScreen(m)` in `View()`
- [x] 1.3 [RED] `internal/tui/screens/progress_test.go`: advance spinner N ticks, assert running-step row contains `m.spinner.View()` output
- [x] 1.4 [GREEN] `internal/tui/screens/progress.go`: replace static `"⠹"` in `StepRunning` case with `m.spinner.View()`; keep `✓` for `StepDone`, `○` for `StepPending`
- [x] 1.5 [RED] `internal/tui/screens/health_test.go`: same spinner-frame assertion for health screen running step
- [x] 1.6 [GREEN] `internal/tui/screens/health.go`: add `spinner spinner.Model` field; `Init()` returns `m.spinner.Tick`; propagate `spinner.TickMsg`; use `m.spinner.View()` for running row

## Phase 2: Status Bar & Gradient Logo (PR 1 — Tier 1b)

- [x] 2.1 [RED] `internal/tui/components/statusbar_test.go`: table-driven — wide terminal shows version/preset/path, narrow (<40) returns empty, long path truncated with ellipsis
- [x] 2.2 [GREEN] `internal/tui/components/statusbar.go`: stateless `RenderStatusBar(width int, version, preset, path string) string`; hidden when width < 40
- [x] 2.3 [GREEN] `internal/tui/styles/screens.go`: add package-level `StatusBarStyle` var (Rose Pine semantic colors)
- [x] 2.4 [REFACTOR] `internal/tui/model.go`: `renderContent()` appends `components.RenderStatusBar(m.width, m.deps.Version, preset, backupPath)` at bottom; add `Deps` accessors for preset/path if missing
- [x] 2.5 [RED] `internal/tui/styles/logo_test.go`: assert `RenderLogo` returns `len(lines)` gradient-colored lines on color profile; assert empty on width < 40; assert uncolored on Ascii profile
- [x] 2.6 [GREEN] `internal/tui/styles/logo.go`: replace 5 fixed `Foreground()` styles with `lipgloss.Blend1D` gradient (Love→Gold→Rose→Pine→Lavender); Ascii profile fallback to plain text. **Verify `Blend1D` signature before implementing** (open question from design)

## Phase 3: Viewport Dry-Run (PR 2 — Tier 2a)

- [ ] 3.1 [RED] `internal/tui/screens/restore_test.go`: send `restoreDryRunResultMsg{output: "diff..."}`, assert `viewport.SetContent` called and `View()` renders viewport output
- [ ] 3.2 [GREEN] `internal/tui/screens/restore.go`: add `viewport viewport.Model` + `vpReady bool` fields; `WindowSizeMsg` sets viewport dimensions; `restoreDryRunResultMsg` calls `SetContent`; `renderDryRun` writes `m.viewport.View()`
- [ ] 3.3 [RED] `restore_test.go`: press `PgDn`/`PgUp`/`j`/`k`/`g`/`G` in `restoreStateDryRun`, assert viewport scroll position changes; press `q`, assert transition to `restoreStateList`
- [ ] 3.4 [GREEN] `restore.go` Update: forward scroll keys (`j/k/↑/↓/PgUp/PgDn/g/G`) to `m.viewport.Update`; `q` transitions to list state

## Phase 4: Mouse Navigation & Empty States (PR 2 — Tier 2b)

- [ ] 4.1 [RED] `internal/tui/screens/dashboard_test.go`: `MouseWheelMsg{Button: MouseWheelDown}` advances table cursor; `MouseClickMsg{Y: 2}` sets cursor; mouse suppressed when `search.IsActive()`
- [ ] 4.2 [GREEN] `internal/tui/screens/dashboard.go`: set `v.MouseMode = tea.MouseModeCellMotion` in `View()`; add `MouseWheelMsg`/`MouseClickMsg` cases in `Update`; guard `m.search.IsActive()` return early
- [ ] 4.3 [RED] `internal/tui/components/empty_state_test.go`: table-driven — output contains icon, italic message, hint text
- [ ] 4.4 [GREEN] `internal/tui/components/empty_state.go`: stateless `RenderEmptyState(icon, message, hint string) string`
- [ ] 4.5 [GREEN] `internal/tui/styles/screens.go`: add `EmptyStateIconStyle`, `EmptyStateMsgStyle`, `EmptyStateHintStyle` package-level vars
- [ ] 4.6 [REFACTOR] `dashboard.go`, `restore.go`, `cloud.go`: replace bare empty strings with `components.RenderEmptyState(...)` calls

## Phase 5: Paste Support (PR 3 — Tier 3)

- [ ] 5.1 [RED] `internal/tui/screens/wizard_test.go`: send `tea.PasteMsg{Content: "work-laptop"}`, assert input buffer equals `"work-laptop"`; send paste to pre-filled input, assert append
- [ ] 5.2 [GREEN] `internal/tui/screens/wizard.go`: add `case tea.PasteMsg:` in active textinput Update paths; append `msg.Content` to input buffer. **Note: field is `Content` not `Text`** (v2 API)

## Phase 6: Verification & Coverage

- [ ] 6.1 Run `go test ./internal/tui/...` — all tests green
- [ ] 6.2 Run `go test -cover ./internal/tui/...` — verify ≥80% per package
- [ ] 6.3 Run `go vet ./...` and `golangci-lint run` — clean
