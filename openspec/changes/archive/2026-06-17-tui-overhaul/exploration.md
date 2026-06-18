# Exploration: TUI Overhaul — bak-cli Interactive Experience

## Current State

### bak-cli TUI Architecture

bak-cli has **3 bubbletea models**, all in `cmd/` (no `internal/tui/` package):

| Model | File | Lines | Purpose |
|-------|------|-------|---------|
| `pickModel` | `cmd/pick.go` | 170 | Category picker checklist |
| `wizardModel` | `cmd/wizard.go` | 332 | 5-step profile creation wizard |
| (wizard reused) | `cmd/login.go` | 113 | Login provider selection |

**Entry point**: `bak` with no args → standard cobra help text (no interactive mode). `rootCmd` in `cmd/root.go` has no `RunE`.

**Pattern**: Cobra `RunE` → TTY check (`isTTY()` var) → `tea.NewProgram(model).Run()` → type-assert result → extract data → business logic via `internal/actions/`.

**Boundary**: `internal/actions/` defines callback types (`Picker`, `WizardRunner`) that `cmd/` implements with bubbletea. Clean separation — actions never import bubbletea.

**Dependencies** (from `go.mod`):
- `github.com/charmbracelet/bubbletea v1.3.10` (v1 API)
- `github.com/charmbracelet/lipgloss v1.1.0` (v1 API)
- `github.com/mattn/go-isatty v0.0.20`
- No `bubbles` dependency — everything hand-rolled

### ⚠️ CRITICAL: bubbletea v1 vs v2

The project uses **bubbletea v1** (`github.com/charmbracelet/bubbletea`). Context7 docs reference **v2** (`charm.land/bubbletea/v2`) with breaking changes:

| Feature | v1 (current) | v2 |
|---------|--------------|-----|
| Import path | `github.com/charmbracelet/bubbletea` | `charm.land/bubbletea/v2` |
| Key press msg | `tea.KeyMsg` (struct) | `tea.KeyPressMsg` (concrete type) |
| Key type check | `msg.Type == tea.KeyRunes` | `msg.String()` switch |
| Key constants | `tea.KeyEnter`, `tea.KeyCtrlC` | Same but via `KeyPressMsg` |
| KeyMsg | Struct with `Type`, `Runes` | Interface (press + release) |

**Decision: Stay on v1.** Rationale:
- Migration to v2 changes ALL existing TUI tests (which construct `tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}`)
- v1 is stable and well-documented
- v2 import path also changes lipgloss to `charm.land/lipgloss/v2`
- No functional benefit for bak-cli's use case
- Can reconsider v2 migration as a separate change later

### Visual State

- **No borders, boxes, or visual structure** — plain text lists
- **Inline styles** recreated every `View()` call — `lipgloss.NewStyle()` called inside render loops
- **No `bubbles` dependency** — everything hand-rolled
- **No `tea.WindowSizeMsg`** handling — not responsive to terminal resize
- **No loading indicators** — backup ops block silently
- **No main menu** — `bak` shows cobra help
- **Duplicated style code** — `pick.go` and `wizard.go` each define their own `cursorStyle`, `checkedStyle`, etc.

### Current Color Scheme (ANSI basic colors)

| Role | ANSI Code | Usage |
|------|-----------|-------|
| Title | `"12"` (bright blue) | Bold |
| Step indicator | `"8"` (dark gray) | Muted |
| Help text | `"8"` (dark gray) | Muted |
| Checked | `"10"` (bright green) | `[x]` |
| Unchecked | `"8"` (dark gray) | `[ ]` |
| Cursor | `"11"` (bright yellow) | `> ` |

### Existing Test Coverage

- `cmd/pick_test.go` — 10 tests: Init, Update (quit/cursor/toggle/confirm), View, Selected, cmd structure
- `cmd/wizard_test.go` — 8 tests: Init, step transitions, Ctrl+C/Esc, View, provider selection, isTTY
- All tests use v1 patterns: `tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}`
- Tests follow AGENTS.md rule: test `Update()`/`View()` as pure functions, never `Program.Run()`

---

## gentle-ai TUI Architecture (Reference)

### Pattern: Single Root Model + Stateless Render Functions

```
internal/tui/model.go          ← Root Model (Init/Update/View, ~3100 lines)
internal/tui/router.go         ← Screen navigation graph
internal/tui/screens/          ← Pure render functions per screen
internal/tui/styles/           ← Rose Pine theme + logo
```

Each screen is a **pure function**: `func RenderWelcome(cursor int, ...) string`

`View()` dispatches via switch on `m.Screen` enum (40+ screens).

### Visual Design System

**Rose Pine palette**:
```
ColorBase     = "#191724"  // Deep background
ColorSurface  = "#1f1d2e"  // Surface
ColorOverlay  = "#6e6a86"  // Borders
ColorText     = "#e0def4"  // Primary text
ColorSubtext  = "#908caa"  // Secondary
ColorLavender = "#c4a7e7"  // Accent (titles, cursor, frame)
ColorGreen    = "#9ccfd8"  // Success
ColorPeach    = "#f6c177"  // Progress
ColorRed      = "#eb6f92"  // Errors
ColorBlue     = "#31748f"  // Info
ColorMauve    = "#ebbcba"  // Headings
```

**Key visual elements**:
- `╔═╗║╚╝` double-border frame (`lipgloss.DoubleBorder()`)
- `▸` Unicode cursor indicator
- `█░` progress bar
- `⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏` braille spinner
- ASCII art logo with 5-band gradient

### Key Patterns Worth Borrowing

1. **DI via function fields** — `ExecuteFn`, `RestoreFn` injected at construction, tests swap mocks
2. **Common render helpers** — `renderOptions()`, `renderCheckbox()`, `renderRadio()`
3. **Goroutine + tea.Cmd** for async ops with `tea.Batch` for concurrent spinner
4. **Package-level function overrides** for testing (swap `os.Stat`, etc.)
5. **Manual scroll state** with `ScrollOffset` + `↑ more`/`↓ more` indicators

---

## Context7 Documentation Findings

### bubbletea v1 API (verified against current codebase)

**Core pattern** (from `/charmbracelet/bubbletea` docs):
```go
type Model interface {
    Init() Cmd
    Update(msg Msg) (Model, Cmd)
    View() string
}
```

**Key APIs for bak-cli**:
- `tea.NewProgram(model, opts...)` — create program with options like `tea.WithAltScreen()`, `tea.WithMouseCellMotion()`
- `tea.Cmd` — `func() tea.Msg` — function that returns a message
- `tea.Batch(cmds...)` — run multiple commands **concurrently**
- `tea.Sequence(cmds...)` — run commands **sequentially** (each waits for previous)
- `tea.Quit` — built-in command to exit
- `tea.WindowSizeMsg` — sent on terminal resize, contains `Width` and `Height`
- `tea.KeyMsg` (v1) — struct with `Type tea.KeyType` and `Runes []rune`

**Async pattern** (for backup progress):
```go
// Start async work in a goroutine, send result back as tea.Msg
func doBackup() tea.Msg {
    result, err := runBackup()
    return backupDoneMsg{result: result, err: err}
}

// In Init or Update:
return m, tea.Batch(
    doBackup,           // runs async
    m.spinner.Tick,     // runs concurrently for animation
)
```

### lipgloss v1 API (verified)

**Package-level styles** (best practice — define once, reuse):
```go
var (
    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#c4a7e7"))

    boxStyle = lipgloss.NewStyle().
        BorderStyle(lipgloss.DoubleBorder()).
        BorderForeground(lipgloss.Color("#6e6a86")).
        Padding(1, 2)
)
```

**Layout functions**:
- `lipgloss.JoinHorizontal(pos VerticalPosition, strs ...string) string` — join side by side. `pos`: `lipgloss.Top`, `lipgloss.Center`, `lipgloss.Bottom`, or float64
- `lipgloss.JoinVertical(pos HorizontalPosition, strs ...string) string` — join stacked. `pos`: `lipgloss.Left`, `lipgloss.Center`, `lipgloss.Right`
- `lipgloss.Place(width, height int, hPos, vPos Position, str string) string` — place text in a cell
- `lipgloss.Size(str string) (int, int)` — get rendered width/height

**Border types**: `lipgloss.NormalBorder()`, `lipgloss.RoundedBorder()`, `lipgloss.DoubleBorder()`, `lipgloss.HiddenBorder()`, `lipgloss.ThickBorder()`

**Table package** (lipgloss built-in):
```go
import "github.com/charmbracelet/lipgloss/table"

t := table.New().
    Border(lipgloss.RoundedBorder()).
    StyleFunc(func(row, col int) lipgloss.Style { ... }).
    Headers("ID", "Date", "Size").
    Rows(rows...)
```

### bubbles Components (from `/charmbracelet/bubbles` docs)

All follow the sub-model pattern: `component.New()`, `component.Update(msg) (Model, Cmd)`, `component.View() string`.

**spinner** — for loading indicators:
```go
import "github.com/charmbracelet/bubbles/spinner"

// In model:
s := spinner.New()
s.Spinner = spinner.Dot  // or spinner.Line, spinner.MiniDot, etc.
s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#c4a7e7"))

// Init: return s.Tick()
// Update: handle spinner.TickMsg → m.spinner, cmd = m.spinner.Update(msg)
// View: m.spinner.View()
```

**progress** — for progress bars:
```go
import "github.com/charmbracelet/bubbles/progress"

p := progress.New(
    progress.WithDefaultGradient(),
    progress.WithWidth(40),
)
// Set value: m.progress.SetValue(0.65)
// Update: handle progress.FrameMsg
// View: m.progress.View()
```

**table** — for backup list:
```go
import "github.com/charmbracelet/bubbles/table"

cols := []table.Column{
    {Title: "ID", Width: 8},
    {Title: "Date", Width: 12},
    {Title: "Size", Width: 8},
}
t := table.New(
    table.WithColumns(cols),
    table.WithRows(rows),
    table.WithFocused(true),
    table.WithHeight(10),
)
// Update: m.table, cmd = m.table.Update(msg)
// Handle tea.WindowSizeMsg: m.table.SetWidth(msg.Width), m.table.SetHeight(msg.Height)
// View: m.table.View()
```

**textinput** — for search:
```go
import "github.com/charmbracelet/bubbles/textinput"

ti := textinput.New()
ti.Placeholder = "Search..."
ti.Focus()
// Init: return ti.Focus() (or ti.Blink)
// Update: m.input, cmd = m.input.Update(msg)
// View: m.input.View()
```

**viewport** — for scrollable content:
```go
import "github.com/charmbracelet/bubbles/viewport"
// Useful for help text, long backup logs
```

**list** — for menu items with filtering:
```go
import "github.com/charmbracelet/bubbles/list"
// list.New(items, delegate, width, height)
// Built-in filtering, pagination, keybindings
```

---

## Skills Discovery

### Found (via `npx skills find`)

| Skill | Installs | URL | Relevance |
|-------|----------|-----|-----------|
| `ggprompts/tfe@bubbletea` | 340 | skills.sh/ggprompts/tfe/bubbletea | ⭐ Best for bubbletea patterns |
| `existential-birds/beagle@bubbletea-code-review` | 178 | skills.sh/existential-birds/beagle/bubbletea-code-review | Code review for bubbletea |
| `pedronauck/skills@bubbletea` | 107 | skills.sh/pedronauck/skills/bubbletea | General bubbletea |
| `dicklesworthstone/meta_skill@building-glamorous-tuis` | 63 | skills.sh/dicklesworthstone/meta_skill/building-glamorous-tuis | TUI design patterns |
| `gentleman-programming/engram@gentleman-bubbletea` | 59 | skills.sh/gentleman-programming/engram/gentleman-bubbletea | From our org |

### Already Installed

| Skill | Path | Relevance |
|-------|------|-----------|
| `golang-pro` | `.agents/skills/golang-pro/` | Go patterns, testing, concurrency |
| `go-testing` | `.config/opencode/skills/go-testing/` | Go test patterns |

### Not Found

- No dedicated **lipgloss** skill exists
- No **bubbles** component skill exists

### Recommendation

Install before implementation phase:
```bash
npx skills add ggprompts/tfe@bubbletea -g -y
```

The `bubbletea-code-review` skill could be useful for the verify phase.

---

## AGENTS.md Audit — TUI-Specific Gaps

### Existing TUI Rules (✅ Present)

1. `MUST NOT test bubbletea.Program.Run() directly — test model Update()/View() logic instead`
2. `MUST test TUI model Update() and View() methods — they contain business logic that deserves coverage`

### Missing Rules (❌ Need Addition)

#### 1. TUI Package Organization
```markdown
### TUI Code Organization
- MUST place all bubbletea models under `internal/tui/` — NOT in `cmd/`
- MUST separate concerns: `internal/tui/styles/` (theme), `internal/tui/components/` (reusable widgets), `internal/tui/screens/` (pure render functions)
- `cmd/` SHOULD only wire cobra → TUI launch → result extraction
- MUST NOT import `internal/tui/` from `internal/actions/` — actions define callback types, cmd/ implements them with TUI
```

#### 2. Lipgloss Style Conventions
```markdown
### TUI Styling
- MUST define lipgloss styles as package-level variables — NOT inside View() or per-frame
- MUST use hex color codes (e.g., `"#c4a7e7"`) via `lipgloss.Color()`, NOT ANSI numbers (e.g., `"12"`)
- MUST centralize theme colors in `internal/tui/styles/theme.go`
- SHOULD use `lipgloss.DoubleBorder()` for main containers, `lipgloss.RoundedBorder()` for inner panels
```

#### 3. Bubbletea Version Lock
```markdown
### Bubbletea Version
- MUST use bubbletea v1 API (`github.com/charmbracelet/bubbletea`) — NOT v2 (`charm.land/bubbletea/v2`)
- MUST use `tea.KeyMsg` (struct) for key handling — NOT `tea.KeyPressMsg` (v2 only)
- MUST construct test key messages as `tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}`
```

#### 4. Bubbles Dependency
```markdown
### Bubbles Components
- MUST justify each `bubbles` component used (why hand-rolling is insufficient)
- SHOULD prefer `bubbles/spinner` and `bubbles/progress` over hand-rolled animations
- SHOULD prefer `bubbles/table` for tabular data over custom table rendering
- MUST NOT add `bubbles/list` if a simple cursor list suffices (avoid over-engineering)
```

#### 5. Responsive Layout
```markdown
### TUI Responsiveness
- MUST handle `tea.WindowSizeMsg` in models that display bordered content
- SHOULD define minimum terminal dimensions and show a resize message if too small
- MUST test View() output with different simulated window sizes
```

#### 6. TUI Testing Rules
```markdown
### TUI Testing
- MUST test Update() by constructing messages directly: `tea.KeyMsg{Type: tea.KeyEnter}`
- MUST test View() by asserting on substrings (styles produce ANSI codes, avoid exact match)
- MUST test screen transitions by verifying model state after Update() calls
- SHOULD use golden file tests for complex View() output (compare with `testdata/`)
- MUST NOT snapshot-test ANSI escape sequences — they vary by terminal
```

---

## Affected Areas

### Files to Modify
| File | Change |
|------|--------|
| `cmd/root.go` | Add `RunE` for interactive TUI when no args |
| `cmd/pick.go` | Refactor to use shared theme/components from `internal/tui/` |
| `cmd/wizard.go` | Refactor to use shared theme/components |
| `cmd/login.go` | Wire through new TUI wizard |
| `cmd/deps.go` | Extend `cmdDeps` with TUI dependencies if needed |
| `go.mod` | Add `github.com/charmbracelet/bubbles` dependency |
| `AGENTS.md` | Add TUI-specific rules (see audit above) |

### New Files
| File | Purpose |
|------|---------|
| `internal/tui/styles/theme.go` | Rose Pine palette + style definitions |
| `internal/tui/styles/logo.go` | ASCII art logo with gradient |
| `internal/tui/components/menu.go` | Reusable cursor menu renderer |
| `internal/tui/components/checkbox.go` | Reusable checkbox renderer |
| `internal/tui/components/frame.go` | Bordered container helper |
| `internal/tui/model.go` | Root TUI model (main menu + screen router) |
| `internal/tui/screens/menu.go` | Main menu screen render |
| `internal/tui/screens/dashboard.go` | Backup list/dashboard screen |
| `internal/tui/screens/progress.go` | Backup progress screen |
| `internal/tui/screens/settings.go` | Settings screen (future) |
| `internal/tui/keys.go` | Shared keybinding constants |

### Unchanged
| File | Reason |
|------|--------|
| `internal/actions/*.go` | No changes — callback types stay as-is |
| `internal/backup/*.go` | Business logic untouched |
| `internal/config/*.go` | Config untouched |

---

## Proposed Features

### F1: Interactive Main Menu (Priority: HIGH)

When user types `bak` with no args, launch a TUI main menu:

```
╔══════════════════════════════════════════════════════╗
║                                                      ║
║   [ASCII art logo with gradient]                     ║
║                                                      ║
║   bak v1.3.1 — Backup your AI coding setup           ║
║                                                      ║
║   ▸ Create backup                                    ║
║     Restore backup                                   ║
║     Browse backups                                   ║
║     Cloud sync                                       ║
║     Profiles                                         ║
║     Settings                                         ║
║     Quit                                             ║
║                                                      ║
║   j/k: navigate • enter: select • q: quit            ║
║                                                      ║
╚══════════════════════════════════════════════════════╝
```

**Implementation**:
- New `internal/tui/model.go` — root model with `screen` enum and `Update()` router
- `cmd/root.go` gets `RunE` that checks `len(args) == 0 && isTTY()` → launches TUI
- Menu items map to existing commands: backup → `PickBackupAction`, restore → `restoreAction`, etc.
- Uses `lipgloss.DoubleBorder()` for the frame
- Handles `tea.WindowSizeMsg` for responsive width

**bubbletea pattern**: Single root model, screen enum, switch in Update/View.

### F2: Backup Dashboard (Priority: HIGH)

After selecting "Browse backups", show a styled table:

```
┌─ Recent Backups ─────────────────────────────────────┐
│                                                       │
│  ID       Date          Size    Status    Cloud       │
│  ──────── ───────────── ─────── ───────── ─────────── │
│  a3f2c1   2026-06-09    2.3 MB  ✓ OK      ↑ synced   │
│  b7d4e2   2026-06-08    2.1 MB  ✓ OK      ↑ synced   │
│  c1a9f3   2026-06-07    1.8 MB  ⚠ partial  — local    │
│                                                       │
│  d: delete • r: restore • space: select • q: back     │
└───────────────────────────────────────────────────────┘
```

**Implementation**:
- Use `bubbles/table` with custom `StyleFunc` using Rose Pine colors
- Populate from `backup.ListLocal()` (existing function)
- Handle `tea.WindowSizeMsg` → `table.SetWidth()` / `table.SetHeight()`
- `lipgloss.RoundedBorder()` for the panel

### F3: Styled Backup Progress (Priority: HIGH)

During backup/restore, show a progress screen:

```
┌─ Creating Backup ─────────────────────────────────────┐
│                                                       │
│  ⠹ Backing up...                                      │
│                                                       │
│  ████████████████░░░░░░░░░░  65%                      │
│                                                       │
│  ✓ skills          12 files                           │
│  ✓ commands         8 files                           │
│  ⠹ config           3/7 files                         │
│  ○ plugins          pending                            │
│                                                       │
└───────────────────────────────────────────────────────┘
```

**Implementation**:
- `bubbles/spinner` for the loading indicator
- `bubbles/progress` for the progress bar
- Goroutine runs backup → sends `tea.Msg` with progress updates
- `tea.Batch(spinner.Tick, doBackupStep)` for concurrent animation + work
- Custom `progressMsg`, `stepDoneMsg` types

### F4: Fuzzy Search (Priority: MEDIUM)

Add `/` to activate fuzzy search on any list:

**Implementation**: `bubbles/textinput` + custom filter logic.

### F5: First-Run Setup Wizard (Priority: MEDIUM)

When no config exists, guide user through initial setup. Reuses existing `wizardModel` pattern with new visual style.

### F6: Cloud Sync Status (Priority: MEDIUM)

Interactive view showing cloud state with options to sync/pull/push.

### F7: Backup Health Check (Priority: LOW)

Verify backup integrity with visual checklist feedback.

### F8: ASCII Art Logo with Gradient (Priority: MEDIUM)

Custom bak-cli logo using Braille dots or box drawing, with 5-band Rose Pine gradient (mauve → lavender → blue → teal → green).

### F9: Settings Screen (Priority: LOW)

Interactive settings for categories, cloud provider, theme, backup location.

### F10: Toast Notifications (Priority: LOW)

Non-intrusive success/error messages overlay.

---

## Approaches

### Approach A: Big Bang — Full TUI Package

Create `internal/tui/` with all components, screens, styles at once.

- **Pros**: Clean architecture from day one, all patterns consistent
- **Cons**: ~2000+ lines, massive PR, high review burden, hard to test incrementally
- **Effort**: High (3-5 days)

### Approach B: Incremental — Feature-Flagged Slices ⭐ RECOMMENDED

Build in 5 chained PRs, each self-contained:

1. **PR1**: Foundation — `internal/tui/styles/` (theme + logo) + `internal/tui/components/` (menu, checkbox, frame) + AGENTS.md TUI rules (~300 lines)
2. **PR2**: Main menu — `internal/tui/model.go` + root `RunE` wiring + `bak` no-args launches TUI (~400 lines)
3. **PR3**: Refactor existing — Migrate `pick.go` + `wizard.go` to use shared theme/components (~300 lines)
4. **PR4**: Dashboard + Progress — `bubbles/table` + `bubbles/spinner` + `bubbles/progress` screens (~400 lines)
5. **PR5**: Polish — Fuzzy search + settings + toast notifications (~300 lines)

- **Pros**: Each PR reviewable under 400 lines, testable incrementally, can ship partial value, follows bak-cli's chained-PR pattern
- **Cons**: More PR overhead, some rework between PRs
- **Effort**: Medium per PR, total ~5-7 days

### Approach C: Minimal Viable TUI — Main Menu Only

Just add the main menu + styled existing screens, no new features.

- **Pros**: Fastest, lowest risk, immediate UX improvement
- **Cons**: Doesn't address dashboard, progress, search gaps
- **Effort**: Low (1-2 days)

---

## Recommendation

**Approach B: Incremental** — 5 chained PRs.

Rationale:
- 400-line review budget (user preference) makes big-bang impractical
- Each PR delivers standalone value
- Can pause between PRs if priorities shift
- Follows bak-cli's existing SDD chained-PR pattern
- PR1 unblocks all subsequent PRs (theme + components are prerequisites)

**Dependency order**: PR1 → PR2 → PR3 → PR4 → PR5 (PR3 and PR4 can be parallel after PR2).

**Pre-requisite**: Install `ggprompts/tfe@bubbletea` skill before PR1 for bubbletea best practices.

**Also required**: Update AGENTS.md with TUI rules (from audit above) as part of PR1.

---

## New Dependency: `charmbracelet/bubbles`

Required for production-ready components:

| Component | Used In | Justification (per AGENTS.md) |
|-----------|---------|-------------------------------|
| `bubbles/spinner` | F3 (progress) | Hand-rolling braille spinner animation is non-trivial |
| `bubbles/progress` | F3 (progress) | Smooth animated progress bar with blending |
| `bubbles/table` | F2 (dashboard) | Column sizing, row selection, scrolling built-in |
| `bubbles/textinput` | F4 (search) | Cursor management, input handling, focus |
| `bubbles/viewport` | Future (help) | Scrollable content area for long text |

Already a transitive dependency via bubbletea's module graph. Adding as direct is low risk.

**NOT adding**: `bubbles/list` — bak-cli's lists are simple enough to hand-roll with the shared `menu.go` component.

---

## Risks

1. **bubbletea v1 API limitations** — v1 lacks some v2 niceties (key release events, alt screen improvements). Acceptable for bak-cli's needs.
2. **Model complexity** — The gentle-ai root Model is ~3100 lines. bak-cli should keep it under 500 lines by using sub-models for complex screens (dashboard, progress).
3. **Test coverage impact** — New TUI code needs tests. Follow AGENTS.md: test `Update()`/`View()` as pure functions, never `Program.Run()`. View tests should assert substrings, not exact ANSI output.
4. **Terminal compatibility** — Box drawing and Unicode art may not render on Windows ConHost. bak-cli already uses `go-isatty` for TTY detection. Should detect terminal capabilities or document Windows Terminal requirement.
5. **Existing test stability** — Refactoring pick/wizard models (PR3) may break existing tests that construct `tea.KeyMsg` directly. Must preserve v1 message format.
6. **Performance** — Lipgloss styles MUST be package-level constants, not recreated per `View()` call. Current code violates this (creates `lipgloss.NewStyle()` inside `View()` loops).
7. **bubbles API stability** — `bubbles` v2 API differs from v1. Must use v1-compatible bubbles (`github.com/charmbracelet/bubbles` not `charm.land/bubbles/v2`).

---

## Skills to Inject in Later Phases

| Phase | Skill | Purpose |
|-------|-------|---------|
| Propose | `golang-pro` | Go architecture patterns |
| Design | `golang-pro` | Interface design, package layout |
| Apply (PR1) | `golang-pro` + `bubbletea` (install) | bubbletea model patterns, lipgloss styling |
| Apply (PR3) | `bubbletea-code-review` (install) | Review refactored models |
| Verify | `go-testing` | Test patterns, coverage |

---

## Ready for Proposal

**Yes** — exploration is complete and significantly enriched with:

1. ✅ Verified bubbletea v1 API references (not assumptions)
2. ✅ Critical v1 vs v2 decision documented
3. ✅ Skills discovered and installation recommended
4. ✅ AGENTS.md audit with 6 specific rule additions
5. ✅ bubbles dependency justified per-component
6. ✅ Accurate Context7 documentation for all three libraries
7. ✅ Existing test patterns analyzed

The orchestrator should:
1. Present this report to the user
2. Ask: "¿Querés que arranque con el PR1 (theme + componentes compartidos + AGENTS.md TUI rules) o preferís ajustar las features propuestas?"
3. If approved, run `sdd-propose` for `tui-overhaul`
