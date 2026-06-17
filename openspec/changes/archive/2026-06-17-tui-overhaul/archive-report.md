# Archive Report: tui-overhaul

## Change Summary

**Change**: tui-overhaul
**Project**: bak-cli
**Status**: ARCHIVED (complete)
**Date**: 2026-06-16
**Artifact Store**: openspec

Transformed bak-cli from plain-text checklist TUIs into a beautiful interactive Rose Pine-themed experience. Running `bak` with no args now launches an interactive main menu with ASCII art, box-drawing frames, and vim-style navigation. Added 8 screens (menu, dashboard, settings, cloud, health, progress, shortcuts, welcome) and 6 reusable components (menu, checkbox, radio, help, toast, search).

## PR History

| PR | Focus | Status | Key Deliverables |
|----|-------|--------|-----------------|
| PR1 | Foundation | ✅ Complete | Rose Pine theme (11 colors), 13 package-level lipgloss styles, ASCII logo with 5-band gradient, Frame helper, 4 shared components (menu, checkbox, radio, help), AGENTS.md TUI rules (6 sections) |
| PR2 | Main Menu | ✅ Complete | Root model with 8-screen router, key navigation (q/j/k/enter/esc), Deps struct with DI, main menu screen, first-run welcome detection, TTY-aware cmd wiring (`bak` no-args launches TUI) |
| PR3 | Screens I | ✅ Complete | Dashboard (bubbles/table), Settings (4 toggles), Cloud status display |
| PR4 | Screens II | ✅ Complete | Health (4 async diagnostics), Progress (spinner + progress bar with async msgs), Shortcuts overlay |
| PR5 | Components II | ✅ Complete | Toast (auto-hide with TTL), Search (textinput + filter), root model integration |

**Post-overhaul**: 5 wiring gaps identified (wizard screen, restore/profiles routing, post-TUI dispatch, dashboard search integration, toast triggering) — resolved by the separate `tui-wiring-gaps` change (already archived).

## File Inventory

### Source Files (22)

| Package | Files | Description |
|---------|-------|-------------|
| `internal/tui/styles/` | `theme.go`, `styles.go`, `logo.go`, `frame.go`, `screens.go` | Rose Pine palette, 13 package-level styles, ASCII logo, frame helper, screen-specific styles |
| `internal/tui/components/` | `menu.go`, `checkbox.go`, `radio.go`, `help.go`, `toast.go`, `search.go` | Reusable render functions and sub-models |
| `internal/tui/screens/` | `menu.go`, `welcome.go`, `cloud.go`, `shortcuts.go`, `dashboard.go`, `progress.go`, `settings.go`, `health.go` | 8 screen implementations |
| `internal/tui/` | `model.go`, `keys.go`, `deps.go` | Root model with screen router, keybinding constants, DI struct |
| `cmd/` | `tty.go`, `root.go` (modified) | TTY detection, TUI launch wiring |

### Test Files (14)

| Package | Files |
|---------|-------|
| `internal/tui/styles/` | `styles_test.go`, `logo_test.go` |
| `internal/tui/components/` | `components_test.go`, `toast_test.go`, `search_test.go` |
| `internal/tui/screens/` | `menu_test.go`, `welcome_test.go`, `cloud_test.go`, `shortcuts_test.go`, `dashboard_test.go`, `progress_test.go`, `settings_test.go`, `health_test.go` |
| `internal/tui/` | `model_test.go` |
| `cmd/` | `tty_test.go` |

## Coverage

| Package | Coverage | Target |
|---------|----------|--------|
| `internal/tui` (model) | 91.1% | ≥80% ✅ |
| `internal/tui/components` | 97.9% | ≥80% ✅ |
| `internal/tui/screens` | 94.0% | ≥80% ✅ |
| `internal/tui/styles` | 90.5% | ≥80% ✅ |

## Verification

- **Status**: PASS
- All 27+ packages pass `go test ./... -count=1`
- `go vet ./...` clean
- `go build ./...` clean
- All 22 spec requirements verified (see verify-report.md)

## Dependencies Added

| Package | Version | Used By |
|---------|---------|---------|
| `charm.land/bubbletea/v2` | v2.0.7 | Root program, all models (already in go.mod) |
| `charm.land/lipgloss/v2` | v2.0.3 | All styles and rendering (already in go.mod) |
| `charm.land/bubbles/v2` | v2.1.0 | table (dashboard), textinput (search), spinner (progress) |

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Model structure | Root model + sub-models for complex screens | Root stays under 500 lines; gentle-ai reference was ~3100 lines monolithic |
| Style location | Package-level `var` in `styles/` | Zero per-frame allocation; enforced by AGENTS.md rules |
| DI pattern | Struct field injection (function fields) | Matches existing `cmdDeps` pattern in bak-cli |
| Screen routing | Switch on `Screen` enum in root Update/View | Shared state (window size, theme) without complex router |
| bubbletea version | Stay on v2 (`charm.land/bubbletea/v2`) | Already migrated; 18+ existing tests use v2 API |
| Delivery strategy | 5 chained PRs (stacked-to-main) | Each under review budget; independently revertable |

## Known Wiring Gaps (Resolved)

The following 5 gaps were identified at tui-overhaul completion and resolved by the separate `tui-wiring-gaps` change (already archived):

| Gap | Description | Resolution |
|-----|-------------|------------|
| 11.1 | Wizard screen — `ScreenWizard` declared but no `screens/wizard.go` | Resolved by tui-wiring-gaps |
| 11.2 | Restore and Profiles menu items are no-ops in `handleMenuEnter()` | Resolved by tui-wiring-gaps |
| 11.3 | Post-TUI action dispatch — `Selection()` exists but `defaultRunTUI()` ignores it | Resolved by tui-wiring-gaps |
| 11.4 | Dashboard search — Search component doesn't filter table rows | Resolved by tui-wiring-gaps |
| 11.5 | Toast triggering — nothing calls `Show()` | Resolved by tui-wiring-gaps |

## Lessons Learned

1. **Chained PRs work well for TUI work** — each PR delivered standalone value and was independently reviewable. The 400-line budget kept reviews focused.
2. **Package-level styles are non-negotiable** — defining lipgloss styles as `var` at package scope eliminates per-frame allocation and makes the theme trivially changeable.
3. **Sub-model composition keeps root model small** — dashboard and progress use bubbles sub-models (lazy-initialized on first visit), keeping the root model under 500 lines.
4. **Wiring gaps are inevitable in large TUI overhauls** — components built in isolation need integration work. Tracking them explicitly in tasks.md Phase 11 prevented them from being forgotten.
5. **bubbletea v2 API is significantly different from v1** — `tea.KeyPressMsg{Code: 'q'}` vs `tea.KeyMsg{Type: tea.KeyRunes}`. The exploration phase caught this early and the project was already on v2.
6. **Test coverage exceeded targets across all packages** — table-driven tests for Update()/View() as pure functions worked well. The ≥80% target was conservative; actual coverage ranged from 90.5% to 97.9%.

## Recommendations for Future Work

1. **Consider `bubbles/viewport` for long content** — help text, backup logs, or changelog screens would benefit from scrollable viewports.
2. **Mouse support** — `tea.WithMouseCellMotion()` is available but not yet enabled. Could add click-to-select for menu items.
3. **Golden file tests for View() output** — current tests assert substrings. Golden files would catch unintended visual regressions more precisely.
4. **Windows terminal testing** — box-drawing characters and Unicode art should be verified on Windows ConHost vs Windows Terminal.
5. **Theme customization** — Rose Pine is hardcoded. A theme abstraction (e.g., `styles.Theme()` returning a config struct) would enable user-selectable themes.

## Archive Contents

| Artifact | Status |
|----------|--------|
| `proposal.md` | ✅ Present |
| `spec.md` | ✅ Present (flat file, no delta specs subdirectory) |
| `design.md` | ✅ Present |
| `exploration.md` | ✅ Present |
| `tasks.md` | ✅ Present (Phases 1-10 complete; Phase 11 gaps resolved by tui-wiring-gaps) |
| `tasks-pr2.md` | ✅ Present (all tasks complete) |
| `tasks-pr3.md` | ✅ Present (all tasks complete) |
| `tasks-pr4.md` | ✅ Present (all tasks complete) |
| `tasks-pr5.md` | ✅ Present (all tasks complete) |
| `verify-report.md` | ✅ Present (PASS) |
| `verify-report-pr2.md` | ✅ Present |
| `verify-report-pr3.md` | ✅ Present |
| `verify-report-pr4.md` | ✅ Present |
| `archive-report.md` | ✅ This file |

## Reconciliation Note

Phase 11 of `tasks.md` contains 5 unchecked items (11.1-11.5). These are wiring gaps that were explicitly scoped as remaining work at the time of tui-overhaul completion. They were resolved by the separate `tui-wiring-gaps` change, which has been fully implemented, verified, and archived. The unchecked boxes are preserved as historical record of what was identified during the tui-overhaul cycle.

## SDD Cycle Status

This change has been fully planned, implemented, verified, and archived. The change directory remains at `openspec/changes/tui-overhaul/` as an audit trail (not moved to `archive/` per orchestrator instruction).
