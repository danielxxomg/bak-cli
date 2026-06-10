# Tasks: TUI Overhaul — PR1 Foundation

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~590 (theme: 30, styles: 40, logo: 60, frame: 30, components: 150, tests: 200, AGENTS.md: 80) |
| 400-line budget risk | Medium (foundation PR, slightly over budget but acceptable) |
| Chained PRs recommended | Yes (5 PRs total, this is PR1) |
| Suggested split | PR1 (Foundation) → PR2 (Main Menu) → PR3 (Refactor) → PR4 (Dashboard) → PR5 (Polish) |
| Delivery strategy | chained-prs (already decided in proposal) |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Rose Pine theme + styles + logo + frame | PR1 | ~160 lines; foundation for all TUI rendering |
| 2 | Shared components (menu, checkbox, radio, help) | PR1 | ~150 lines; reusable render functions |
| 3 | Tests for all components (≥80% coverage) | PR1 | ~200 lines; table-driven tests |
| 4 | AGENTS.md TUI rules (6 sections) | PR1 | ~80 lines; conventions for PR2-5 |

## Phase 1: Theme & Styles (TDD)

**TDD: Write `styles_test.go` FIRST (RED), then implement (GREEN).**

- [x] 1.1 **RED** — Create `internal/tui/styles/styles_test.go` with table-driven tests: verify all 11 Rose Pine colors exist as `lipgloss.Color` constants (ColorBase, ColorSurface, ColorLavender, etc.), verify package-level styles exist (TitleStyle, HeadingStyle, SelectedStyle, FrameStyle, PanelStyle, HelpStyle), verify Frame() produces DoubleBorder characters (╔, ╚, ╗,╝). Use substring assertions.
- [x] 1.2 **GREEN** — Create `internal/tui/styles/theme.go`: define 11 semantic colors as package-level `var` (ColorBase `#191724`, ColorSurface `#1f1d2e`, ColorOverlay `#26233a`, ColorMuted `#6e6a86`, ColorSubtle `#908caa`, ColorText `#e0def4`, ColorLove `#eb6f92`, ColorGold `#f6c177`, ColorRose `#ebbcba`, ColorPine `#31748f`, ColorLavender `#c4a7e7`). Add godoc comments.
- [x] 1.3 **GREEN** — Create `internal/tui/styles/styles.go`: define package-level `var` styles using lipgloss v2 API: `TitleStyle` (Bold, Foreground ColorLavender), `HeadingStyle` (Bold, Foreground ColorGold), `SelectedStyle` (Foreground ColorRose), `FrameStyle` (Border lipgloss.DoubleBorder(), BorderForeground ColorMuted), `PanelStyle` (Padding 1, Border lipgloss.NormalBorder(), BorderForeground ColorOverlay), `HelpStyle` (Foreground ColorMuted). Define `CursorIndicator = "▸ "`. Add godoc comments.
- [x] 1.4 **GREEN** — Create `internal/tui/styles/frame.go`: implement `Frame(content string, width int) string` that wraps content in DoubleBorder using FrameStyle, respects width parameter. Add godoc comment.
- [x] 1.5 **VERIFY** — Run `go test ./internal/tui/styles/...` — all tests pass. Run `go test ./...` — zero regressions. Verify coverage ≥80% with `go test -cover ./internal/tui/styles/...`.
- [x] 1.6 **COMMIT** — `feat(tui): add Rose Pine theme and package-level styles`

## Phase 2: ASCII Art Logo (TDD)

**TDD: Write `logo_test.go` FIRST (RED), then implement (GREEN).**

- [x] 2.1 **RED** — Create `internal/tui/styles/logo_test.go` with table-driven tests: verify RenderLogo(width) returns non-empty string, verify logo contains "bak" text, verify logo uses 5 distinct color codes (Rose Pine gradient), verify narrow terminal (width < 40) returns truncated/empty logo.
- [x] 2.2 **GREEN** — Create `internal/tui/styles/logo.go`: define ASCII art for "bak-cli" (5 lines tall, ~35 chars wide), implement `RenderLogo(width int) string` that applies 5-band Rose Pine gradient (ColorLove → ColorGold → ColorRose → ColorPine → ColorLavender), returns empty string if width < 40. Add godoc comment.
- [x] 2.3 **VERIFY** — Run `go test ./internal/tui/styles/...` — all tests pass. Verify coverage ≥80%.
- [x] 2.4 **COMMIT** — `feat(tui): add ASCII art logo with Rose Pine gradient`

## Phase 3: Shared Components (TDD)

**TDD: Write `components_test.go` FIRST (RED), then implement (GREEN).**

- [x] 3.1 **RED** — Create `internal/tui/components/components_test.go` with table-driven tests for all 4 components: RenderMenu (verify cursor indicator `▸` at correct position, verify items rendered), RenderCheckbox (verify `[x]` for checked, `[ ]` for unchecked, verify ColorGreen for checked items), RenderRadio (verify `(•)` for selected, `( )` for unselected), RenderHelp (verify key-value pairs rendered, verify ColorMuted style). Use substring assertions.
- [x] 3.2 **GREEN** — Create `internal/tui/components/menu.go`: implement `RenderMenu(items []string, cursor int) string` that renders menu items with `CursorIndicator` (▸) at cursor position, applies SelectedStyle to cursor item. Add godoc comment.
- [x] 3.3 **GREEN** — Create `internal/tui/components/checkbox.go`: implement `RenderCheckbox(label string, checked, focused bool) string` that renders `[x]` (ColorGreen) or `[ ]` (ColorMuted), applies SelectedStyle if focused. Add godoc comment.
- [x] 3.4 **GREEN** — Create `internal/tui/components/radio.go`: implement `RenderRadio(label string, selected, focused bool) string` that renders `(•)` (ColorGold) or `( )` (ColorMuted), applies SelectedStyle if focused. Add godoc comment.
- [x] 3.5 **GREEN** — Create `internal/tui/components/help.go`: define `HelpKey` struct (`Key string`, `Desc string`), implement `RenderHelp(keys []HelpKey) string` that renders key-desc pairs separated by ` • `, applies HelpStyle. Add godoc comment.
- [x] 3.6 **VERIFY** — Run `go test ./internal/tui/components/...` — all tests pass. Run `go test ./...` — zero regressions. Verify coverage ≥80% with `go test -cover ./internal/tui/components/...`.
- [x] 3.7 **COMMIT** — `feat(tui): add shared components (menu, checkbox, radio, help)`

## Phase 4: AGENTS.md TUI Rules

- [x] 4.1 **ADD** — Append 6 new sections to `AGENTS.md` under "### TUI Rules": (1) TUI Package Organization: `internal/tui/styles/` for theme/styles, `internal/tui/components/` for reusable render functions, `internal/tui/screens/` for screen-specific logic, (2) TUI Styling: MUST use package-level `var` for all lipgloss styles (zero per-frame allocation), MUST use Rose Pine semantic colors from `styles/` package, MUST NOT use inline `lipgloss.NewStyle()` in `View()` methods, (3) Bubbletea Version Lock: MUST use `charm.land/bubbletea/v2 v2.0.7` (v2 API), MUST use `tea.KeyPressMsg{Code: 'q'}` (not v1 `tea.KeyMsg{Type: tea.KeyRunes}), (4) Bubbles Dependency: MUST justify any `charm.land/bubbles/v2` import (spinner, progress, table), MUST NOT add bubbles for trivial functionality, (5) TUI Responsiveness: MUST handle `tea.WindowSizeMsg` in all models, MUST store width/height in model struct, MUST adapt bordered content to terminal dimensions, MUST show "terminal too small" message if dimensions < 20x10, (6) TUI Testing: MUST test model `Update()` and `View()` methods (pure functions), MUST use table-driven tests for component renderers, MUST NOT test `bubbletea.Program.Run()` directly, MUST achieve ≥80% coverage for `internal/tui/` packages.
- [x] 4.2 **VERIFY** — Run `go test ./...` — zero regressions. Verify `go build ./...` succeeds.
- [x] 4.3 **COMMIT** — `docs(agents): add TUI rules for theme, components, and testing`

## Phase 5: Final Verification

- [x] 5.1 Run `go test ./...` — zero failures across all packages.
- [x] 5.2 Run `go vet ./...` — clean.
- [x] 5.3 Verify `internal/tui/styles/` coverage ≥80% (`go test -cover ./internal/tui/styles/...`).
- [x] 5.4 Verify `internal/tui/components/` coverage ≥80% (`go test -cover ./internal/tui/components/...`).
- [x] 5.5 Verify no existing tests broken (`git diff --name-only | grep _test.go` should show only new TUI test files).
- [x] 5.6 Verify AGENTS.md contains all 6 TUI rule sections.
- [ ] 5.7 Create PR: `feat(tui): foundation — Rose Pine theme, components, and conventions (PR1/5)`. Description: "PR1 of 5 chained PRs. Adds TUI foundation: Rose Pine theme (11 colors), package-level styles, ASCII art logo, shared components (menu, checkbox, radio, help), and AGENTS.md TUI rules. All tests pass, ≥80% coverage. PR2-5 will add main menu, refactor existing screens, dashboard, and polish."

## Implementation Notes

**TDD Discipline**: Follow strict RED → GREEN → REFACTOR cycle. Write failing tests first, then implement minimal code to pass, then refactor for clarity.

**Bubbletea v2 API**: Use `charm.land/bubbletea/v2` (not v1). Key differences: `tea.KeyPressMsg{Code: 'q'}` instead of `tea.KeyMsg{Type: tea.KeyRunes}`, `tea.NewView()` instead of returning string from `View()`.

**Lipgloss v2 API**: Use `charm.land/lipgloss/v2`. Package-level styles are `var` (not `const`), use `.Render()` method.

**Rose Pine Palette**: 11 semantic colors from https://rosepinetheme.com/palette/. Use hex codes, not ANSI color numbers.

**Component Signatures**: Match design.md exactly: `RenderMenu(items []string, cursor int) string`, `RenderCheckbox(label string, checked, focused bool) string`, `RenderRadio(label string, selected, focused bool) string`, `RenderHelp(keys []HelpKey) string`.

**Frame Function**: `Frame(content string, width int) string` wraps content in DoubleBorder. Use `FrameStyle` from styles package.

**ASCII Art Logo**: 5 lines tall, ~35 chars wide, fits 40-col minimum terminal. Apply 5-band Rose Pine gradient (Love → Gold → Rose → Pine → Lavender). Return empty string if width < 40.

**Next PRs**: PR2 adds main menu + wizard, PR3 refactors pick.go/wizard.go, PR4 adds dashboard + progress, PR5 adds search + polish.
