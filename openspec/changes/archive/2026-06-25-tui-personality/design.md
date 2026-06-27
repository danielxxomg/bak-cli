# Design: tui-personality

## Technical Approach

Additive, render-only personality layer on the existing bak-cli TUI — no architectural refactor, no new dependencies (bubbles/lipgloss v2 already in go.mod). Bubbletea v2 moved program options to declarative `tea.View` fields, so every gap is a field assignment in an existing `View()` returning `tea.View`, consistent with the existing `v.AltScreen = true` in `model.go:619`. All new logic is testable as pure `Update`/`View` functions per AGENTS.md TUI rules; no `Program.Run` in unit tests.

Maps to specs: `tui-personality` (F1–F7), `tui-interactive-preview` (F5 surface), `restore-flow` delta (F5 routing), `wizard-flow` delta (F8 paste).

## Architecture Decisions

### Decision: Window title via View field, not command

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `tea.SetWindowTitle(...)` cmd in `handleScreenChange` | **Rejected: v1 API, removed in v2.0.7** — Context7 v2 upgrade guide: "replaced by setting the `WindowTitle` field on the View." Won't compile. | ✗ |
| `v.WindowTitle = titleForScreen(m.screen)` in `View()` | Pure read at render time; same frame as content; no command plumbing; matches `v.AltScreen` pattern. | ✓ |

**Rationale**: Field assignment is the v2-correct primitive, requires zero cmd state, and stays consistent with existing code. `screenChangeMsg` already mutates `m.screen`; `View()` recomputes the title each render from current state.

### Decision: Single shared spinner for step indicators

**Choice**: Reuse `ProgressModel.spinner` (already ticking) as the source for the running-step row.
**Alternatives**: per-step spinner.Model (TickMsg storm risk), separate health spinner.
**Rationale**: Progress already batches `m.spinner.Tick` + `pgCmd`. Health has no spinner today — add one `spinner.Model` field there with `Init()` returning `m.spinner.Tick`. Avoids double-tick contention.

### Decision: Viewport embedded in RestoreModel, not a shared component

**Choice**: `RestoreModel.viewport viewport.Model` field.
**Alternatives**: `internal/tui/components/viewport.go` shared wrapper.
**Rationale**: AGENTS.md DRY — no other screen needs a scrollable viewport yet. Premature shared component adds coupling without payoff. If a second consumer appears, extract then.

### Decision: Status bar / empty state as stateless render fns

**Choice**: `components/statusbar.go`, `components/empty_state.go` — pure functions matching existing `RenderMenu`/`RenderHelp`/`RenderCheckbox` pattern.
**Alternatives**: sub-models with their own `Init/Update/View`.
**Rationale**: Both are pure renders (no message handling). Matches established component category in the codebase and AGENTS.md grouping.

### Decision: Mouse scroll guard via `search.IsActive()`

**Choice**: In `DashboardModel.Update`, `case tea.MouseMsg:` returns `(m, nil)` unmodified when `m.search.IsActive()`.
**Rationale**: Persona rule — familiar feel bug (wheel steals focus while typing in search). Cheap guard, easy to forget, requires an explicit test.

## Data Flow

```
handleKey ──► screenChangeMsg ──► m.screen = X
                                       │
View() ──► renderContent()             │
  ├─ renderScreen (sub-model View)     │  (sub-model owns its View fields:
  │      └ sets MouseMode, Cursor…     │   MouseModeCellMotion, viewport content)
  ├─ toast overlay                      │
  └─ components.RenderStatusBar(w,v,p,path)  ← reads m.screen/m.deps
View() ──► v.WindowTitle = titleForScreen(m.screen)   (per REQ-TP-001)
View() ──► v.AltScreen = true  (existing)

restoreDryRunResultMsg ──► viewport.SetContent(diff) ──► restoreStateDryRun
  └ tea.KeyPressMsg (j/k/PgUp/PgDn/g/G/↑↓) ──► viewport.Update ──► new viewport
  └ q ──► restoreStateList
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/tui/model.go` | Modify | Add `titleForScreen(m) string` pure helper; set `v.WindowTitle` in `View()`; call `components.RenderStatusBar(...)` in `renderContent()`; expose `m.deps.Version/Preset/BackupPath` for the bar |
| `internal/tui/components/statusbar.go` | Create | Stateless `RenderStatusBar(width, version, preset, path) string` |
| `internal/tui/components/statusbar_test.go` | Create | Table-driven: wide/narrow width, truncation, hidden <40 |
| `internal/tui/components/empty_state.go` | Create | Stateless `RenderEmptyState(icon, msg, hint) string` |
| `internal/tui/components/empty_state_test.go` | Create | Table-driven render assertions |
| `internal/tui/styles/screens.go` | Modify | Add pkg-level `StatusBarStyle`, `EmptyStateIconStyle`, `EmptyStateMsgStyle`, `EmptyStateHintStyle` (Rose Pine semantic colors) |
| `internal/tui/styles/logo.go` | Modify | Replace 5 fixed `Foreground()` styles with `lipgloss.Blend1D(len(lines), ColorLove,ColorGold,ColorRose,ColorPine,ColorLavender)`; no-color profile fallback to plain text |
| `internal/tui/styles/logo_test.go` | Modify/Add | Assert `len(grad) == len(lines)`; no-color branch yields uncolored logo |
| `internal/tui/screens/progress.go` | Modify | In `stepIndicator(StepRunning)`, return `m.spinner.View()` instead of literal `"⠹"` (delete `progressStepRunningIndicator` const usage for running rows) |
| `internal/tui/screens/progress_test.go` | Modify | Advance spinner N ticks, assert running-row output contains current frame |
| `internal/tui/screens/health.go` | Modify | Add `spinner spinner.Model` field; `Init()` returns `m.spinner.Tick`; `Update` propagates `spinner.TickMsg`; running step uses `m.spinner.View()` |
| `internal/tui/screens/dashboard.go` | Modify | Set `v.MouseMode = tea.MouseModeCellMotion` in `View()`; add `case tea.MouseWheelMsg` (forward to `table.Update`) and `case tea.MouseClickMsg` (set cursor by Y); guard `m.search.IsActive()`; empty branch calls `RenderEmptyState` |
| `internal/tui/screens/restore.go` | Modify | Embed `viewport viewport.Model`; size on `WindowSizeMsg`; `restoreDryRunResultMsg` → `m.viewport.SetContent(output)`; in `restoreStateDryRun` forward scroll keys to `viewport.Update`; `renderDryRun` writes `m.viewport.View()`; set `v.MouseMode`; empty branch + error branch use `RenderEmptyState` |
| `internal/tui/screens/restore_test.go` | Modify | viewport sizing, `PgDn`/`PgUp`/`g`/`G` scroll assertions, `q` returns to list, `MouseWheelMsg{Y:-1}` scrolls; `MouseMode` set |
| `internal/tui/screens/dashboard_test.go` | Modify | `MouseWheelMsg`/`MouseClickMsg` cases; mouse suppressed when `search.IsActive()` |
| `internal/tui/screens/cloud.go` | Modify | No-provider branch uses `RenderEmptyState` instead of bare string; surface error branch (pre-existing bug flagged in explore.md) |
| `internal/tui/screens/wizard.go` | Modify | Add `case tea.PasteMsg:` to active textinput Update paths; append `msg.Content`; keep `DisableBracketedPasteMode = false` |
| `internal/tui/screens/wizard_test.go` | Modify | Send `tea.PasteMsg{Content: "work-laptop"}`, assert input updated |
| `internal/tui/model_test.go` | Modify | Table-driven `View().WindowTitle` assertions per `m.screen` (Menu, Wizard, Restore w/ id, Progress w/ step counter) |

## Interfaces / Contracts

```go
// Pure, exported for table-driven tests, lives in components package.
func RenderStatusBar(width int, version, preset, path string) string
func RenderEmptyState(icon, message, hint string) string

// Pure title mapper in package tui (root).
func titleForScreen(s screen) string // "bak — Main Menu", "bak — Wizard", ...

// Spinner on health (mirrors ProgressModel).
type HealthModel struct {
    // …existing…
    spinner spinner.Model
}
```

No new interfaces for the dispatch layer: the existing `subModel` interface and `subEntries` map handle the new fields transparently — `View()` returning a `tea.View` with extra fields doesn't change routing.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `titleForScreen`, `RenderStatusBar`, `RenderEmptyState`, `RenderLogo` gradient/no-color, step indicator frame, viewport scroll keys, mouse wheel/click suppress-on-search, `PasteMsg` append — all pure `Update`/`View` | Table-driven `go test`; construct msgs literally (`tea.MouseWheelMsg{Y:-1}`, `tea.PasteMsg{Content:"…"}`, `restoreDryRunResultMsg{output:DryRunOutput}`); assert on `View().Content` / `View().WindowTitle` / `View().MouseMode`. No `Program.Run`. |
| Unit | `View().WindowTitle` per active screen + step counter on `ScreenProgress` | Construct `Model{screen: X}`, call `View()`, assert field |
| Integration | End-to-end restore dry-run shows viewport content; dashboard mouse wheel moves table cursor | Existing `internal/tui/` integration test pattern (real `RestoreAction` stub via deps) |
| Coverage | ≥80% per `internal/tui/` package per AGENTS.md | `go test -cover ./internal/tui/...`; gap-fill before apply exit |

## Migration / Rollout

No migration. All changes are additive render-only. Per proposal PR split Tier 1→2→3 — each tier is an independent PR, revert by PR.

## Open Questions

- [ ] Exact `lipgloss.Blend1D` signature in `charm.land/lipgloss/v2 v2.0.3` (explore.md says Context7-verified; design assumes `Blend1D(n int, stops ...lipgloss.Color)`). **Verify in apply before writing the logo.go change.** If signature differs, fall back to per-line `Foreground()` with profile-gated color (degraded but spec-compliant gradient).
- [ ] Does `tea.View` expose `MouseMode` as a settable `MouseMode` enum field? Context7 confirms yes (`v.MouseMode = tea.MouseModeCellMotion`). Confirm in apply against the actual struct in v2.0.7.
- [ ] Where to source status-bar `preset` and `backup path` for the version/preset/path segments — `m.deps` needs accessors (Version exists; preset/path may need new `Deps` fields or a config loader call). Decide in tasks phase.