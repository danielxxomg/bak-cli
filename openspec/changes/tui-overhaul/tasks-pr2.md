# Tasks: TUI Overhaul — PR2 Main Menu — ✅ COMPLETE

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~420 (model_test: 120, deps+keys+model: 140, screen tests: 100, screen impl: 80, cmd wiring: 45, tty helper: 15) |
| 400-line budget risk | Medium (slightly over budget; test-heavy, production code ~265) |
| Chained PRs recommended | Yes (PR2 of 5, stacked-to-main) |
| Suggested split | PR1 ✅ → PR2 ✅ → PR3 ✅ → PR4 ✅ → PR5 ✅ |
| Delivery strategy | chained-prs (pre-decided) |
| Chain strategy | stacked-to-main |
| Status | **ALL TASKS COMPLETE** |

## Phase 1: Root Model — RED (TDD)

- [x] 1.1 **RED** — Create `internal/tui/model_test.go` with table-driven tests:
  - `TestNewModel`: verify default screen is `ScreenMenu`, cursor 0, deps stored
  - `TestModel_Init`: verify returns nil cmd
  - `TestModel_Update_Quit`: send `tea.KeyPressMsg{Code: 'q'}` on `ScreenMenu`, verify `tea.Quit` cmd returned
  - `TestModel_Update_NavigateDown`: send `tea.KeyPressMsg{Code: 'j'}` on `ScreenMenu`, verify cursor increments (clamp at menu len-1)
  - `TestModel_Update_NavigateUp`: send `tea.KeyPressMsg{Code: 'k'}`, verify cursor decrements (clamp at 0)
  - `TestModel_Update_WindowSize`: send `tea.WindowSizeMsg{Width: 120, Height: 40}`, verify model stores dimensions
  - `TestModel_Update_MinSizeGuard`: send `tea.WindowSizeMsg{Width: 10, Height: 5}`, verify `m.tooSmall = true`
  - `TestModel_View_Menu`: verify View() output contains menu items ("Create backup", "Quit")
  - `TestModel_View_TooSmall`: verify View() shows "Terminal too small" when dimensions < 20×10
  - `TestModel_Selection`: verify Selection() returns MenuSelection with cursor index

## Phase 2: Root Model — GREEN

- [x] 2.1 Create `internal/tui/deps.go`: define `Deps` struct (`Version string`, `ListBackups func()`, `RunBackup func()`, `ConfigExists func() bool`), `MenuSelection` struct (`Cursor int`, `Item string`), `BackupInfo` struct, `ProgressUpdate` struct, `DefaultMenuItems` var.
- [x] 2.2 Create `internal/tui/keys.go`: define shared key constants (`KeyQuit rune = 'q'`, `KeyDown = 'j'`, `KeyUp = 'k'`, `KeyEnter = '\r'`, `KeyEsc = 27`).
- [x] 2.3 Create `internal/tui/model.go`: define `Screen` enum (8 screens), `Model` struct (screen, width, height, cursor, tooSmall, deps, menuItems, search, toast), `NewModel(deps Deps) Model`, `Init()`, `Update()` with WindowSizeMsg and KeyPressMsg routing, `View()` with screen routing and tooSmall guard, `Selection()`.
- [x] 2.4 **VERIFY** — `go test ./internal/tui/...` — all pass. Coverage ≥80%.
- [x] 2.5 **COMMIT** — `feat(tui): add root model with screen router and key navigation`

## Phase 3: Screen Tests — RED (TDD)

- [x] 3.1 **RED** — Create `internal/tui/screens/menu_test.go` with table-driven tests:
  - `TestRenderMainMenu`: verify output contains logo (when width ≥ 40), menu items, cursor indicator, help bar keys
  - `TestRenderMainMenu_NarrowTerminal`: verify logo omitted when width < 40
  - `TestRenderMainMenu_CursorPositions`: verify cursor indicator at correct position for cursor 0, 3, 6
- [x] 3.2 **RED** — Create `internal/tui/screens/welcome_test.go` with table-driven tests:
  - `TestRenderWelcome`: verify output contains welcome message, setup prompt
  - `TestShouldShowWelcome`: verify returns true when `configExists` is false, false when true

## Phase 4: Screen Implementation — GREEN

- [x] 4.1 Create `internal/tui/screens/menu.go`: implement `RenderMainMenu(version string, banner string, cursor int, width int) string` that composes: `styles.RenderLogo(width)` (if width ≥ 40), version subtitle via `styles.TitleStyle`, `components.RenderMenu(items, cursor)` with 7 menu items, `components.RenderHelp(keys)` with contextual keys.
- [x] 4.2 Create `internal/tui/screens/welcome.go`: implement `ShouldShowWelcome(configExists func() bool) bool` and `RenderWelcome(width int) string`.
- [x] 4.3 **VERIFY** — All tests pass, coverage ≥80%.
- [x] 4.4 **COMMIT** — `feat(tui): add main menu screen and first-run welcome detection`

## Phase 5: Root Command Wiring

- [x] 5.1 Create `cmd/tty.go`: implement `isTTY() bool`, injectable `var runTUI func(deps tui.Deps) error = defaultRunTUI`, `defaultRunTUI` that creates model and runs `tea.NewProgram(m).Run()`.
- [x] 5.2 Modify `cmd/root.go`: add `RunE` — if `len(args) == 0 && isTTY()`, build `tui.Deps`, call `runTUI(deps)`, return error. Otherwise fall through to help.
- [x] 5.3 Add `cmd/tty_test.go`: verify `isTTY()` returns false in test environment, verify `runTUI` injection point works.
- [x] 5.4 **VERIFY** — `go test ./...` — all pass. `go vet ./...` — clean. `go build ./...` — succeeds.
- [x] 5.5 **COMMIT** — `feat(cli): wire root command for interactive TUI launch on no-args`

## Phase 6: Final Verification

- [x] 6.1 Run `go test ./...` — zero failures across all packages.
- [x] 6.2 Run `go vet ./...` — clean.
- [x] 6.3 Verify `internal/tui/` coverage ≥80%.
- [x] 6.4 Verify no existing tests broken.
- [x] 6.5 Manual smoke test: `go run .` in TTY launches TUI main menu with Rose Pine theme.
- [x] 6.6 Manual smoke test: `go run . --help` shows cobra help (TUI NOT launched).
- [x] 6.7 Manual smoke test: `go run . | cat` (non-TTY) shows cobra help.
- [x] 6.8 Create PR: `feat(tui): main menu with screen router and TUI launch (PR2/5)`.

## Status

**ALL TASKS COMPLETE**. PR2 delivered the root model, screen router, key navigation, main menu screen, welcome screen, and root command TUI wiring. All tests pass with ≥80% coverage.

See `tasks.md` (Phase 11) for remaining wiring gaps that span beyond PR2 scope.
