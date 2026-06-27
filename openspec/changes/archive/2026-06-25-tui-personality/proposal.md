# Proposal: tui-personality

## Intent

TUI personality is ~4/10. Charm primitives are already wired — the gap is **application**. Apply 9 affordances to reach 8/10: persistent context, living feedback, scrollable previews, mouse nav, branded polish. Zero new deps.

## Scope

### In Scope

**Tier 1** (high value, low effort)
- F1: `tea.View.WindowTitle` — contextual tab title
- F2: Spinning step indicators — `spinner.View()` replaces static `"⠹"`
- F3: Status bar — version + screen + running op
- F4: Gradient logo — `lipgloss.Blend1D` on Rose Pine stops

**Tier 2** (high value, medium effort)
- F5: Viewport dry-run — scrollable diff in restore
- F6: Mouse scroll/click — `MouseModeCellMotion` on dashboard/restore
- F7: Styled empty states — bordered panel + CTA

**Tier 3** (medium value, low effort)
- F8: Paste — `tea.PasteMsg` in wizard + search

### Out of Scope
- Startup reveal animation (defer)
- harmonica springs (overkill)
- bubbles/list (churn, picker works)
- glamour (new dep, plain text suffices)

## Capabilities

### New
- `tui-personality`: window title, status bar, gradient logo, spinning indicators, styled empty states
- `tui-interactive-preview`: viewport dry-run scroll, mouse navigation

### Modified
- `restore-flow`: dry-run becomes scrollable viewport
- `wizard-flow`: paste support in text inputs

## Approach

Additive `tea.View` field assignments on existing `View()`/`Update()`. No refactor. New stateless `components/statusbar.go`. Viewport screen-owned in `restore.go`. Pure-function testable.

**PR split**: Tier 1 (~150 lines) → Tier 2 (~200 lines) → Tier 3 (~30 lines).

## Affected Areas

- `internal/tui/model.go` — `currentWindowTitle()`, status bar wiring
- `internal/tui/screens/progress.go` — spinner-driven step indicator
- `internal/tui/screens/health.go` — add `spinner.Model`, drive indicator
- `internal/tui/screens/restore.go` — embed `viewport.Model`, scroll routing
- `internal/tui/screens/dashboard.go` — mouse mode + wheel routing
- `internal/tui/screens/cloud.go` — styled empty state
- `internal/tui/screens/wizard.go` — `tea.PasteMsg` handling
- `internal/tui/components/statusbar.go` — **new** stateless render fn
- `internal/tui/styles/logo.go` — `Blend1D` gradient
- `internal/tui/styles/screens.go` — empty-state styles

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Viewport test surface | Med | TDD pure-fn tests for scroll + wheel |
| Wheel during search focus | Low | Guard: skip when `search.IsActive()` |
| No-color gradient | Low | `Blend1D` auto-detects; monochrome fallback |

## Rollback Plan

Each tier is an independent PR. All additive render-only — revert by PR, no migration.

## Dependencies

None. Uses existing `bubbles/v2`, `bubbletea/v2`, `lipgloss/v2`.

## Success Criteria

- [ ] Contextual terminal title during operations
- [ ] Rotating spinner frames on running steps
- [ ] Status bar on every screen
- [ ] Gradient logo with no-color fallback
- [ ] Scrollable dry-run (keys + mouse)
- [ ] Mouse wheel on dashboard/restore
- [ ] Styled empty states with CTA
- [ ] Paste in wizard name field
- [ ] ≥80% coverage, TDD
