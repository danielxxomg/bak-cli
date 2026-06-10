# Design: TUI Overhaul

## Technical Approach

Single root model (`internal/tui/model.go`) owns a screen enum, window size, and routes `Update`/`View` to screen handlers. Simple screens are pure render functions in `screens/`; complex screens (dashboard, progress) use `bubbles` sub-models. All styles are Rose Pine package-level constants. DI via function fields matching existing `cmdDeps` pattern.

**Critical**: Codebase uses **bubbletea v2** (`charm.land/bubbletea/v2 v2.0.7`), not v1. Tests use `tea.KeyPressMsg{Code: 'q'}`.

## Architecture Decisions

| Decision | Choice | Rejected | Rationale |
|----------|--------|----------|-----------|
| Model structure | Root + sub-models for complex screens | Monolithic (gentle-ai ~3100 lines) | Root stays <500 lines |
| bubbletea version | Stay on v2 (`charm.land/bubbletea/v2`) | N/A (already v2) | 18 existing tests use v2 API |
| Style location | Package-level `var` in `styles/` | Inline in `View()` (current) | Zero per-frame allocation |
| DI pattern | Struct field injection (function fields) | Constructor functions | Matches `cmdDeps`, `PickBackupAction` |
| Screen routing | Switch on `Screen` enum in root `Update`/`View` | Per-screen models with no router | Shared state (window size, theme) |

## Data Flow

```
bak (no args) → cmd/root.go RunE → isTTY() check
    → tui.NewModel(Deps{...}) → tea.NewProgram(m).Run()
    → rootModel.Update routes by screen enum
    → tea.Quit → cmd extracts Selection → actions.Run()
```

## Package Layout

```
internal/tui/
├── model.go              # Root model: Init/Update/View, screen router
├── model_test.go
├── keys.go               # Shared keybinding constants
├── deps.go               # Deps struct + view-model types
├── styles/
│   ├── theme.go          # Rose Pine: 11 lipgloss.Color constants
│   ├── styles.go         # TitleStyle, MutedStyle, CursorStyle, etc.
│   ├── logo.go           # ASCII art logo
│   └── frame.go          # Frame(width, title, content) — DoubleBorder wrapper
├── components/
│   ├── menu.go           # RenderMenu(items, cursor) string
│   ├── checkbox.go       # RenderCheckbox(items, cursor) string
│   ├── radio.go          # RenderRadio(items, cursor) string
│   ├── helpbar.go        # RenderHelpBar(pairs...) string
│   └── components_test.go
└── screens/
    ├── menu.go           # Main menu: logo + cursor menu
    ├── dashboard.go      # bubbles/table sub-model (PR4)
    ├── progress.go       # bubbles/spinner + progress sub-model (PR4)
    └── wizard.go         # First-run styled wizard (PR2)
```

## Key Interfaces

```go
// internal/tui/deps.go
type Deps struct {
    ListBackups func() ([]BackupInfo, error)
    RunBackup   func(cats []string, ch chan<- ProgressUpdate) error
    Version     string
}

type BackupInfo struct{ ID, Date, Size, Status, Cloud string }
type ProgressUpdate struct{ Step string; Current, Total int; Done bool }
```

```go
// internal/tui/model.go
type Screen int
const ( ScreenMenu Screen = iota; ScreenDashboard; ScreenProgress; ScreenWizard )

type Model struct {
    screen    Screen
    width     int
    height    int
    cursor    int
    dashboard *screens.DashboardModel  // nil until visited
    progress  *screens.ProgressModel   // nil until visited
    deps      Deps
}

func NewModel(deps Deps) Model
func (m Model) Init() tea.Cmd
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m Model) View() tea.View
func (m Model) Selection() MenuSelection  // post-quit result
```

```go
// internal/tui/components/ — pure render functions
func RenderMenu(items []MenuItem, cursor int) string
func RenderCheckbox(items []CheckItem, cursor int) string
func RenderRadio(items []string, cursor int) string
func RenderHelpBar(pairs ...string) string  // "j/k","navigate","enter","select"
```

## File Changes

| File | Action | PR |
|------|--------|----|
| `internal/tui/styles/theme.go` | Create | 1 |
| `internal/tui/styles/styles.go` | Create | 1 |
| `internal/tui/styles/logo.go` | Create | 1 |
| `internal/tui/styles/frame.go` | Create | 1 |
| `internal/tui/components/*.go` + tests | Create | 1 |
| `AGENTS.md` | Modify: add 6 TUI rule sections | 1 |
| `internal/tui/model.go` + `deps.go` + `keys.go` + tests | Create | 2 |
| `internal/tui/screens/menu.go` | Create | 2 |
| `internal/tui/screens/wizard.go` | Create | 2 |
| `cmd/root.go` | Modify: add `RunE` for TUI launch | 2 |
| `cmd/pick.go` | Modify: use shared `styles.*` + `components.*` | 3 |
| `cmd/wizard.go` | Modify: use shared `styles.*` + `components.*` | 3 |
| `internal/tui/screens/dashboard.go` | Create | 4 |
| `internal/tui/screens/progress.go` | Create | 4 |
| `go.mod` | Modify: add `charm.land/bubbles/v2` | 4 |

## Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Components | Render output | Pure functions, assert substrings, table-driven |
| Root model | Screen transitions, quit, resize, esc-back | `Update(tea.KeyPressMsg{Code: ...})`, assert `m.screen` |
| Sub-models | Dashboard nav, progress state | `Update()`/`View()` as pure functions, mock `Deps` |
| Styles | `Frame()` produces borders | Smoke test for `╔`/`╚` chars |
| Integration | `root.go` RunE TUI launch | Override `isTTY`, assert model creation |

Coverage target: ≥80% for `internal/tui/` (per AGENTS.md).

## Migration / Rollout

Each PR is independently revertable. PR1-2 add new code only. PR3 refactors guarded by 18 existing tests. PR4-5 add opt-in screens behind menu navigation.

## Open Questions

- [ ] Verify `charm.land/bubbles/v2` import path exists (exploration referenced v1 path `github.com/charmbracelet/bubbles`)
- [ ] Windows ConHost Unicode — box-drawing may need terminal capability detection
- [ ] Logo ASCII art design TBD (must fit 40-col minimum)
