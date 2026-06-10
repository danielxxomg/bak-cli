# Tasks: TUI Overhaul — PR2 Main Menu

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~420 (model_test: 120, deps+keys+model: 140, screen tests: 100, screen impl: 80, cmd wiring: 45, tty helper: 15) |
| 400-line budget risk | Medium (slightly over budget; test-heavy, production code ~265) |
| Chained PRs recommended | Yes (PR2 of 5, stacked-to-main) |
| Suggested split | PR1 ✅ → PR2 (Main Menu) → PR3 (Refactor) → PR4 (Dashboard) → PR5 (Polish) |
| Delivery strategy | chained-prs (pre-decided) |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Medium

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Root model + deps + keys (with tests) | PR2 | ~260 lines; screen router, WindowSizeMsg, minimum guard |
| 2 | Menu screen + welcome screen (with tests) | PR2 | ~180 lines; pure render functions using PR1 components |
| 3 | Root command wiring (RunE + isTTY) | PR2 | ~60 lines; TUI launch on no-args in TTY |

## Phase 1: Root Model — RED (TDD)

**Write `model_test.go` FIRST. Tests must fail (no model.go yet).**

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

- [x] 2.1 Create `internal/tui/deps.go`: define `Deps` struct (`Version string`, `ListBackups func()`, `RunBackup func()`, `ConfigExists func() bool`), `MenuSelection` struct (`Cursor int`, `Item string`), `BackupInfo` struct, `ProgressUpdate` struct. Add godoc comments.
- [x] 2.2 Create `internal/tui/keys.go`: define shared key constants (`KeyQuit rune = 'q'`, `KeyDown = 'j'`, `KeyUp = 'k'`, `KeyEnter = '\r'`, `KeyEsc = 27`). Add godoc comments.
- [x] 2.3 Create `internal/tui/model.go`: define `Screen` enum (`ScreenMenu`, `ScreenDashboard`, `ScreenProgress`, `ScreenWizard`, `ScreenSettings`), `Model` struct (screen, width, height, cursor, tooSmall, deps, menuItems), `NewModel(deps Deps) Model`, `Init() (tea.Model, tea.Cmd)`, `Update(msg tea.Msg) (tea.Model, tea.Cmd)` with switch on msg type (WindowSizeMsg, KeyPressMsg) and screen routing, `View() string` with screen routing and tooSmall guard, `Selection() MenuSelection`. Menu items: `[]string{"Create backup", "Restore", "Browse backups", "Cloud sync", "Profiles", "Settings", "Quit"}`.
- [x] 2.4 **VERIFY** — Run `go test ./internal/tui/...` — all tests pass. Run `go test ./...` — zero regressions. Verify coverage ≥80% with `go test -cover ./internal/tui/`.
- [x] 2.5 **COMMIT** — `feat(tui): add root model with screen router and key navigation`

## Phase 3: Screen Tests — RED (TDD)

**Write screen tests FIRST. Tests must fail (no screen files yet).**

- [x] 3.1 **RED** — Create `internal/tui/screens/menu_test.go` with table-driven tests:
  - `TestRenderMainMenu`: verify output contains logo (when width ≥ 40), menu items, cursor indicator, help bar keys
  - `TestRenderMainMenu_NarrowTerminal`: verify logo omitted when width < 40
  - `TestRenderMainMenu_CursorPositions`: verify cursor indicator at correct position for cursor 0, 3, 6
- [x] 3.2 **RED** — Create `internal/tui/screens/welcome_test.go` with table-driven tests:
  - `TestRenderWelcome`: verify output contains welcome message, setup prompt
  - `TestShouldShowWelcome`: verify returns true when `configExists` is false, false when true

## Phase 4: Screen Implementation — GREEN

- [x] 4.1 Create `internal/tui/screens/menu.go`: implement `RenderMainMenu(version string, banner string, cursor int, width int) string` that composes: `styles.RenderLogo(width)` (if width ≥ 40), version subtitle via `styles.TitleStyle`, `components.RenderMenu(items, cursor)` with 7 menu items, `components.RenderHelp(keys)` with contextual keys (`↑/↓ navigate • enter select • q quit`). Add godoc comment.
- [x] 4.2 Create `internal/tui/screens/welcome.go`: implement `ShouldShowWelcome(configExists func() bool) bool` and `RenderWelcome(width int) string` that renders a welcome message with setup prompt using `styles.Frame()` and `styles.HeadingStyle`. Add godoc comments.
- [x] 4.3 **VERIFY** — Run `go test ./internal/tui/screens/...` — all tests pass. Run `go test ./...` — zero regressions. Verify coverage ≥80% per package.
- [x] 4.4 **COMMIT** — `feat(tui): add main menu screen and first-run welcome detection`

## Phase 5: Root Command Wiring

- [x] 5.1 Create `cmd/tty.go`: implement `isTTY() bool` using `github.com/mattn/go-isatty` on `os.Stdout.Fd()`. Add injectable `var runTUI func(deps tui.Deps) error = defaultRunTUI` for testing. Implement `defaultRunTUI` that creates `tui.NewModel(deps)` and runs `tea.NewProgram(m, tea.WithAltScreen())`. Add godoc comments.
- [x] 5.2 Modify `cmd/root.go`: add `RunE` to `rootCmd` — if `len(args) == 0 && isTTY()`, build `tui.Deps{Version: version, ConfigExists: ...}`, call `runTUI(deps)`, return error. Otherwise fall through to cobra help. Import `internal/tui`. Preserve `--help` behavior (cobra handles this before RunE).
- [x] 5.3 Add test in `cmd/root_test.go` or `cmd/tty_test.go`: verify `isTTY()` returns false in test environment (piped stdout). Verify `runTUI` injection point works (override with mock that records call).
- [x] 5.4 **VERIFY** — Run `go test ./...` — all tests pass. Run `go vet ./...` — clean. Run `go build ./...` — succeeds.
- [x] 5.5 **COMMIT** — `feat(cli): wire root command for interactive TUI launch on no-args`

## Phase 6: Final Verification

- [x] 6.1 Run `go test ./...` — zero failures across all packages.
- [x] 6.2 Run `go vet ./...` — clean.
- [x] 6.3 Verify `internal/tui/` coverage ≥80% (`go test -cover ./internal/tui/...`).
- [x] 6.4 Verify no existing tests broken (18 pre-existing tests still pass).
- [x] 6.5 Manual smoke test: `go run .` in TTY launches TUI main menu with Rose Pine theme.
- [x] 6.6 Manual smoke test: `go run . --help` shows cobra help (TUI NOT launched).
- [x] 6.7 Manual smoke test: `go run . | cat` (non-TTY) shows cobra help.
- [x] 6.8 Create PR: `feat(tui): main menu with screen router and TUI launch (PR2/5)`.

## Implementation Notes

**Bubbletea v2 API**: Use `charm.land/bubbletea/v2`. Key events: `tea.KeyPressMsg{Code: 'q'}`. Model interface: `Init() (tea.Model, tea.Cmd)`, `Update(msg tea.Msg) (tea.Model, tea.Cmd)`, `View() string`. Use `tea.NewProgram(m, tea.WithAltScreen())`.

**DI Pattern**: Match existing `cmdDeps` pattern. `Deps` struct with function fields. `runTUI` var in `cmd/tty.go` for test injection.

**Menu Items**: Exactly 7 items: "Create backup", "Restore", "Browse backups", "Cloud sync", "Profiles", "Settings", "Quit".

**Screen Routing**: Switch on `m.screen` in both `Update()` and `View()`. PR2 only implements `ScreenMenu`. Other screens are stubs for PR4-5.

**Minimum Size Guard**: If `width < 20 || height < 10`, set `m.tooSmall = true` and render "Terminal too small" message instead of normal layout.

**Help Bar Context**: Per-screen help keys. Menu screen: `↑/↓ navigate • enter select • q quit`. Welcome screen: `enter continue • q quit`.

**First-Run Detection**: `ShouldShowWelcome(configExists func() bool)` checks if config directory exists. Wire to `Deps.ConfigExists` in root command.

**GGA Batches**: Each commit has 1-3 files (max 4). Total 5 commits for PR2.
