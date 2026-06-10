# Proposal: TUI Overhaul — Interactive bak-cli Experience

## Intent

Transform bak-cli from plain-text checklist TUIs into a beautiful interactive experience. Running `bak` with no args currently shows cobra help text — it should launch a Rose Pine-themed main menu with ASCII art, box-drawing frames, and vim-style navigation. Existing screens (pick, wizard) get restyled with shared components. New screens (dashboard, progress, search) fill major UX gaps.

## Scope

### In Scope
- Rose Pine theme system with 11 semantic colors + ASCII art logo
- Shared TUI components (menu, checkbox, radio, frame, help bar)
- Interactive main menu on `bak` (no args) with screen routing
- First-run setup wizard with styled multi-step flow
- Backup dashboard (table), styled progress (spinner + bar), fuzzy search
- Refactor existing pick/wizard to shared theme/components
- AGENTS.md TUI rules (package org, styling, testing conventions)
- 16 features across 5 chained PRs, each under review budget

### Out of Scope
- bubbletea v2 migration (COMPLETE — v2.0.7 already in go.mod)
- Changes to `internal/actions/` business logic (untouched)
- `bubbles/list` — lists are simple enough to hand-roll
- Cloud provider implementation changes (only TUI layer)

## Capabilities

### New Capabilities
- `tui-theme`: Rose Pine palette, ASCII art logo, package-level lipgloss styles
- `tui-components`: Reusable menu, checkbox, radio, frame, help bar renderers
- `tui-main-menu`: Root model with screen router, `bak` no-args TUI launch
- `tui-dashboard`: Backup list table with `bubbles/table`
- `tui-progress`: Styled backup progress with `bubbles/spinner` + `bubbles/progress`

### Modified Capabilities
- `bak-cli`: Add `RunE` to rootCmd for TUI launch, add `tea.WindowSizeMsg` handling to existing models

## Approach

**Architecture: Single root model + sub-models for complex screens.**

Root model (`internal/tui/model.go`) owns screen enum, shared state (window size, theme), and routes Update/View to screen-specific handlers. Simple screens (menu, settings) are pure render functions in `internal/tui/screens/`. Complex screens (dashboard, progress) use bubbles sub-models composed into the root.

**Key decisions:**
1. Sub-model composition for dashboard/progress — root model stays under 500 lines
2. Add bubbles dependencies per-PR (spinner+progress in PR4, table in PR4, textinput in PR5)
3. AGENTS.md TUI rules in PR1 (prerequisite for all subsequent work)
4. TUI launch guard: `len(args) == 0 && isTTY()` — preserves `--help` via `bak --help`

## PR Breakdown

| PR | Features | ~Lines | Dependencies |
|----|----------|--------|--------------|
| PR1 Foundation | F8 (logo), theme, shared components, AGENTS.md rules | 500 | None |
| PR2 Main Menu | F1 (main menu), F5 (first-run wizard), F16 (help bar) | 400 | PR1 |
| PR3 Refactor | Migrate pick.go + wizard.go to shared theme/components | 300 | PR1 |
| PR4 Dashboard | F2 (table), F3 (progress), F6 (cloud status), F11, F12 | 400 | PR2 |
| PR5 Polish | F4 (search), F7 (health), F9 (settings), F10 (toast), F14, F15 | 300 | PR2 |

PR3 and PR4/PR5 can run in parallel after PR2.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/tui/styles/` | New | Theme palette, logo, package-level styles |
| `internal/tui/components/` | New | Menu, checkbox, radio, frame, help bar |
| `internal/tui/screens/` | New | Pure render functions per screen |
| `internal/tui/model.go` | New | Root model with screen router |
| `cmd/root.go` | Modified | Add `RunE` for TUI launch on no-args |
| `cmd/pick.go` | Modified | Use shared theme/components |
| `cmd/wizard.go` | Modified | Use shared theme/components |
| `AGENTS.md` | Modified | Add 6 TUI rule sections |
| `go.mod` | Modified | Add `charmbracelet/bubbles` (PR4) |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Windows ConHost can't render Unicode | Med | Detect terminal capabilities, document Windows Terminal requirement |
| Root model grows too large | Low | Sub-model composition for complex screens, 500-line budget |
| Existing test breakage (PR3) | Med | Preserve v2 message patterns, run `go test ./...` after each file |
| Styles created per-frame (perf) | Low | Package-level constants enforced by AGENTS.md rules |

## Rollback Plan

Each PR is independently revertable. PR1-PR2 add new code with no existing behavior changes. PR3 refactors are guarded by existing 18 tests. PR4-PR5 add opt-in screens behind menu navigation — removing them doesn't break existing commands.

## Dependencies

- `charmbracelet/bubbles` — add in PR4 (spinner, progress, table); PR5 adds textinput
- `ggprompts/tfe@bubbletea` skill — install before PR1 for best practices

## Success Criteria

- [ ] `bak` with no args launches themed main menu in TTY
- [ ] All 16 features implemented and tested
- [ ] Existing 18 tests pass + new TUI tests at ≥80% coverage
- [ ] All lipgloss styles are package-level constants (no per-frame allocation)
- [ ] Each PR under 400-line review budget
- [ ] `go test ./...` passes on all 5 PRs
