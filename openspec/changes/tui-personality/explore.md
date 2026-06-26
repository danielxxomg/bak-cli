# Exploration: tui-personality

**Change**: tui-personality
**Mode**: openspec, interactive, Strict TDD ACTIVE
**Date**: 2026-06-26
**Explorer**: sdd-explore (fresh-context, evidence-based)
**Skills loaded**: sdd-explore, bubbletea, bubbletea-code-review, building-glamorous-tuis (partial), golang-pro
**Context7 docs queried**: bubbletea/v2 upgrade guide (View fields, removed commands), bubbles/v2 (spinner, progress, table), lipgloss/v2 (Blend1D/Blend2D gradients)
**Go**: 1.26.4 via mise

## Executive Summary

bak-cli's TUI already sits higher than the brief assumed. I read every non-test TUI file (43 files) and verified imports: it uses **4 of the ~15 Charm feature families** today — `bubbles/textinput` (search), `bubbles/table` (dashboard), `bubbles/spinner` + `bubbles/progress` (progress screen), plus alt screen (already enabled via `v.AltScreen = true` in `model.go:621`). The prompt's "only ~2 families / no spinner / no progress / no alt screen" audit is **stale and wrong**. TheRose Pine palette is genuinely solid but corporations-conservative in *application*: static step indicators, no window title, no mouse, no viewport for long content (dry-run diffs wrap/clutter), no logo animation, no microinteractions, no global status bar. This change targets the *personality deltas* that actually move the needle for a backup/restore CLI: **window title + status bar + viewport-driven preview + spinning step indicators + mouse list nav + a startup logo flourish + styled empty states**, not a feature-bloat catalogue.

Personality is currently ~4/10. This plan brings it to **8/10** by adding 9 concrete affordances (5 Tier-1, 3 Tier-2, 1 Tier-3), each tied to a real bak-cli use case, each testable as a pure Update/View function per AGENTS.md TUI rules.

**Ready for proposal: YES.**

---

## Section 1 — Current TUI State Audit (evidence-based)

### File inventory (non-test line counts)

| Area | Files | Largest | Notes |
|------|-------|---------|-------|
| Root model | `model.go` (897), `dispatch.go` (30), `keys.go` (25), `deps.go` (115) | 897 | **subModel dispatch map already landed** (qa-refactor-analysis, `subEntries` map + `forwardTo`). NOT the 856-line god-function the brief feared. |
| Screens | 13 `.go` files | `wizard.go` (341), `restore.go` (337), `profiles.go` (303), `dashboard.go` (192), `health.go` (178), `progress.go` (201), `cloud.go` (170), `settings.go` (168), `welcome.go` (57), `menu.go` (55), `shortcuts.go` (86), `goconst_constants.go` (16) | — | Each screen implements its own `Init/Update/View` + `WindowSizeMsg`. |
| Components | 7 `.go` files | `modal.go` (158), `search.go` (120), `toast.go` (72), `menu.go` (47), `help.go` (31), `checkbox.go` (27), `radio.go` (27) | — | All **stateless render fns** except `modal`, `search`, `toast` (own sub-models). |
| Styles | 5 `.go` files | `screens.go` (106), `styles.go` (108), `theme.go` (41), `logo.go` (55), `frame.go` (11) | — | All package-level `var` (AGENTS.md compliant). |

### Screens that exist
`ScreenMenu`, `ScreenWelcome`, `ScreenDashboard`, `ScreenProgress`, `ScreenSettings`, `ScreenCloud`, `ScreenShortcuts`, `ScreenHealth`, `ScreenRestore`, `ScreenProfiles`. (`ScreenWelcome`, `ScreenMenu`, `ScreenShortcuts` are stateless renderers; the other 7 own sub-models dispatched via `subEntries`.)

### Components that exist
`RenderMenu`, `RenderCheckbox`, `RenderRadio`, `RenderHelp`, `Search` (sub-model, wraps `textinput`), `Modal` (sub-model, centered bordered), `Toast` (sub-model, auto-hide).

### Styles that exist
Rose Pine semantic colors (Base/Surface/Overlay/Muted/Subtle/Text/Love/Gold/Rose/Pine/Lavender) in `theme.go`. Shared styles (`Title/Heading/Selected/Frame/Panel/Help/Checked/Unchecked/RadioSelected/Toast/ScreenTitle/Search`) + per-screen style blocks (dashboard, progress, cloud) all package-scope. Logo: 5-line ASCII "bak" with a **static 5-band Rose Pine gradient** (Love→Gold→Rose→Pine→Lavender per line). Frame helper wraps content in `DoubleBorder`.

### Charm feature usage today (corrected from the brief)
| Feature | In use? | Where | Quality |
|--------|---------|-------|---------|
| `bubbles/textinput` | ✅ | `components/search.go` | search box only |
| `bubbles/table` | ✅ | `screens/dashboard.go` | hardcoded width 76/height 20, resizes on WindowSizeMsg |
| `bubbles/spinner` | ✅ | `screens/progress.go` | spinner ticks, but **only the standalone spinner row animates** |
| `bubbles/progress` | ✅ | `screens/progress.go` | bar with `WithDefaultBlend`, 40-wide |
| alt screen | ✅ | `model.go:621` `v.AltScreen = true` | **already enabled** |
| `bubbles/list` | ❌ | — | restore/dashboard use hand-rolled cursor lists |
| `bubbles/viewport` | ❌ | — | dry-run diff and shortcuts are raw multi-line strings; long content wraps/clutters |
| `bubbles/key` | ❌ | — | help is hand-rolled `HelpKey{Key,Desc}` table |
| `glamour` | ❌ | — | help is plain text |
| `harmonica` | ❌ (indirect dep only) | go.mod has `charmbracelet/harmonica v0.2.0` indirect | unused |
| `lipgloss` gradients | ⚠️ partial | `logo.go` uses 5 fixed styles, **not the `Blend1D` gradient API** | no smooth/animated gradient |
| `tea.View.WindowTitle` | ❌ | — | terminal title never set |
| `tea.View.MouseMode` | ❌ | — | no mouse |
| `tea.View.ForegroundColor/BackgroundColor` | ❌ | — | no terminal theme detection; Rose Pine always hardcoded |
| paste / bracketed-paste | ❌ | — | wizard `StepName` reads char-by-char, no paste support |

### The current user experience (why it's 4/10)
1. **No global context bar.** Once you leave the menu there's no persistent reminder of app name, version, active screen, or current operation. You're dropped into a bare heading line.
2. **Static "running" indicators.** `progress.go:50` and `health.go:175` render the running step as the *literal string* `"⠹"` — it never rotates. The live `spinner.Model` ticks but only in a disconnected row. A backup looks frozen at the per-step level even though it isn't.
3. **No terminal title.** When minimized or in a tab strip, the terminal shows whatever the shell set — no "bak — Backup 3/7".
4. **Dry-run diff is unscrollable.** `restore.go:305 renderDryRun` dumps `m.DryRunOutput` verbatim. A real restore diff for a config-rich machine is dozens of lines; it wraps and pushes the help bar off-screen.
5. **No mouse.** Lists (restore picker, dashboard table) are keyboard-only. Mouse scroll is the expected affordance in 2026 terminals.
6. **No startup flourish.** Logo renders instantly, static. A tool that markets "pack your setup, move anywhere" deserves a one-time reveal.
7. **Empty states are bland.** "No backups found" / "(no dry-run available)" are dead strings — a chance for personality (recoverable empty state with a CTA).
8. **No screen-transition feedback.** `screenChangeMsg` swaps content in a single frame — jarring on slow terminals.

### Correctness risks found during read (not personality, but worth flagging)
- `dashboard.go:65-72` table created with `WithWidth(76)/WithHeight(20)` **before** WindowSizeMsg arrives; first paint on a 120-col terminal is narrow until resize. Low-impact, fixable alongside this change.
- `cloud.go:84-93` View renders `RenderCloudStatus(CloudInfo{}, …)` in **both** the no-provider and error branches — an error currently shows the empty-state string, hiding the error. Pre-existing bug; this change's status bar can surface it.

### Affected Areas
- `internal/tui/model.go` — add global status bar field, window title, transition state, mouse mode on View.
- `internal/tui/screens/progress.go` + `health.go` — drive step indicators off the live spinner tick, kill static `"⠹"`.
- `internal/tui/screens/restore.go` — replace raw `DryRunOutput` dump with a `viewport.Model`.
- `internal/tui/screens/dashboard.go` + `restore.go` — opt into `MouseModeCellMotion` on the screen View; route `tea.MouseWheelMsg` to the table/list.
- `internal/tui/styles/logo.go` — switch from 5 fixed `Foreground()` calls to `lipgloss.Blend1D` + optional startup reveal frames.
- `internal/tui/components/` — new `statusbar` component (stateless render) + `viewport` wrapper component or direct screen use.
- `go.mod` — `bubbles/v2` already present; `glamour` NOT required (viewport plain text suffices for diffs; markdown is a Tier-5 maybe-have).

---

## Section 2 — Bubbletea v2 Feature Gap Analysis (Context7-verified)

v2 shifts program options/commands to **declarative `tea.View` fields** (`AltScreen`, `MouseMode`, `WindowTitle`, `ForegroundColor/BackgroundColor`, `Cursor`, `ProgressBar`, `ReportFocus`, `DisableBracketedPasteMode`). bak-cli already returns `tea.View` from every model and sets `AltScreen` — so all gaps below are *field assignments in existing `View()` returns*, not new program options.

| # | Feature | What it does (Context7) | Concrete bak-cli use case | Where it lives | Complexity | Personality impact (1-5) |
|---|---------|------------------------|--------------------------|----------------|------------|--------------------------|
| 1 | `tea.View.WindowTitle` | Set terminal tab title declaratively | "bak — Menu" / "bak — Backup 3/7" / "bak — Restore:abc1234 (dry-run)" | `model.go View()` (per-screen) + each sub-model View | **S** | **5** |
| 2 | Spinning step indicators (`bubbles/spinner` reuse) | One spinner drives animated per-row indicator frames instead of static glyph | progress + health running rows reflect actual rotation | `progress.go`, `health.go` View() | S | 5 |
| 3 | `bubbles/viewport` | Scrollable, cursor-aware content region with `PgUp/PgDn/g/G` defaults | dry-run diff preview (restore) + help/shortcuts on tall content | `restore.go renderDryRun`, possibly `shortcuts.go` | M | 4 |
| 4 | `tea.View.MouseMode` = `MouseModeCellMotion` | Wheel-scroll + click messages | scroll dashboard table, scroll dry-run viewport, click restore list rows | dashboard/restore View() + Update MouseWheelMsg routing | M | 4 |
| 5 | Global status bar (new `components/statusbar`) | Persistent one-line bar: logo glyph + version + active screen + running op | every screen sees context; replaces mid-View reconstruction | `model.go renderContent`, new `components/statusbar.go` | S | 5 |
| 6 | `lipgloss.Blend1D` gradient | Smooth multi-stop color gradient (logo line, progress bar border, success banner) | gradient logo + "Complete!" banner in Rose Pine sweep | `styles/logo.go`, `progress.go` View | S | 4 |
| 7 | Startup logo reveal (spinner-driven frames) | Animate logo color block in on first paint | one-time flourish on `ScreenWelcome`/first menu render | `welcome.go`/`menu.go` via a short tea.Tick sequence | M | 4 |
| 8 | Styled empty + error states | Branded empty-state panel with recovery CTA instead of bare string | "No backups yet — press `b` to create one" (dashboard/restore/cloud) | dashboard, restore, cloud empty branches | S | 4 |
| 9 | `tea.View.ForegroundColor/BackgroundColor` | Detect terminal's fg/bg to decide light-vs-dark fallback | users on light terminals currently get Rose Pine dark-on-near-black → unreadable text; detect and adapt | `model.go View()` (one-time detection cmd) | M | 3 |
| 10 | `tea.View.ProgressBar` | Native terminal OSC9 progress bar in the tab badge | when minimized, terminal tab shows backup % | `model.go View()` while `ScreenProgress` running | S | 3 |
| 11 | `harmonica` spring animation | Physics smoothing for value transitions | smooth percent rise on progress bar; smooth reveal | `progress.go` (wrap progress.SetPercent in spring) | M | 2 |
| 12 | `bubbles/list` | Filterable, paginated list with built-in help | replace hand-rolled restore picker (gains `/`-filter, pagination, mouse, help) | `restore.go` list state | L | 4 (but restore picker already works; high churn) |
| 13 | `bubbles/key` + help.KeyMap | Structured bindings + generated help | auto-consistent shortcuts overlay vs hand-maintained `shortcuts.go` | `shortcuts.go`, per-screen `Update` | M | 2 (consistency, not flash) |
| 14 | `glamour` markdown rendering | Render markdown with syntax/typography | help text / changelog preview | `shortcuts.go` | M (new dep) | 2 |
| 15 | Paste (`DisableBracketedPasteMode=false`) | Multi-char paste into textinput | wizard `StepName` profile name, search box | `wizard.go`, `search.go` | S | 2 |
| 16 | `bubbles/textinput` enhancements | `EchoPassword`, `CharLimit`, `Placeholder` | wizard name field placeholder + length cap | `wizard.go` (use textinput instead of hand-rolled NameInput) | S | 2 |

### Explicitly NOT proposed (no concrete bak-cli value)
- `tea.View.ReportFocus` — no focus-sensitive widgets that benefit.
- `tea.View.KeyboardEnhancements` — no need for Kitty/WezTerm extended keys; vanilla keys cover all bak bindings.
- Full screen-transition animation system (fade/slide between screens) — high effort, fights the alt-screen + fast-swap model; a *contextual* status-bar transition (instant content + persistent bar) is the better UX. Listed as Tier-4 skip.

---

## Section 3 — Personality Design Vision

**Brand voice:** "Calm competence with a quiet flourish." Bak-cli moves your *coding setup* — precious, invisible infrastructure. The TUI should feel like a well-made tool, not a toy: precise Rose Pine color, generous spacing, no gratuitous animation, but **one** moment of delight on startup and **continuous** subtle life during operations. Think `lazygit`'s information density crossed with `charmbracelet/gum`'s polish — not a demo reel.

**Microinteractions:**
- **Startup:** logo gradient sweeps in band-by-band over ~600ms (4 tea.Tick frames), only on the first `ScreenMenu`/`ScreenWelcome` paint, never again in the session. Skippable (any keypress completes it instantly).
- **Operation life:** the step row that's "running" shows the spinner's current frame (`spinner.View()`), so a backup visibly breathes instead of freezing at `"⠹"`. On `StepDone` the row flips to a colored `✓` with a brief fade handled by style, not physics.
- **Completion:** a single "Complete!" line with a Rose Pine `Blend1D` gradient underline — a contained celebration, no finale.
- **Errors:** instead of `Error: <err>` dead text, a bordered Love-colored panel with the failed step, a one-line cause, and the CTA `[r] retry  [q] back`. Character via clarity, not snark.

**Transitions:** instant content swap + the new persistent status bar updates its "active screen" segment in the same frame. The bar provides continuity so the swap doesn't feel jarring — *no* fade/slide, which would add latency to a tool users run dozens of times.

**Loading states:** the existing spinner, but (a) reused as the per-step indicator (Section 2 #2) and (b) paired with rotating status verbs — "Scanning adapters…", "Hashing files…", "Writing manifest…" — drawn from the step name the engine already sends via `ProgressUpdate.Step`. No new strings to author; surface what's already transmitted.

**The "wow" moment:** launching `bak` for the first time and watching the gradient logo resolve + seeing the terminal tab read `bak — Menu`, then `bak — Backup 4/7` while it runs and you switch tabs to do other work. *Situational* delight: the tool respects your context.

---

## Section 4 — Architecture Compatibility

1. **subModel dispatch (already landed).** `model.go` routes via `subEntries` map + `forwardTo`. New per-screen View-field concerns (WindowTitle, MouseMode) fit cleanly: each sub-model's existing `View()` already returns `tea.View`, so we add field assignments there. The root `View()` can layer a *global* window title as the default and let sub-models override via a returned title. No new dispatch abstraction needed.
2. **Further extraction needed?** `model.go` is 897 lines but ~40% is already-extracted init helpers (`initDashboard/Progress/…`). The router is comfortable. We should extract a small `renderStatusBar(m)` helper (S) and a `currentWindowTitle(m) string` helper (S), but **no** structural refactor is a prerequisite. This change is additive.
3. **Where new components live.**
   - Reusable, stateless → `internal/tui/components/` (new `statusbar.go`).
   - Viewport lives *inside* the restore screen → `internal/tui/screens/restore.go` (screen-owned `viewport.Model` field), not a shared component (no other screen needs it yet — avoid premature abstraction per AGENTS.md DRY rules).
   - A shared `animate`/`startupReveal` concern, if added, belongs in `internal/tui/styles/logo.go` (render logic) or a tiny `internal/tui/components/reveal.go` if it grows state.
4. **Command structure.** Spinner already returns `spinner.Tick`; progress re-batches `tea.Batch(m.spinner.Tick, pgCmd)`. New global concerns add at most: a one-time `tea.Tick` for the startup reveal (returns `revealFrameMsg`), and a theme-detection `tea.Cmd` (returns `themeDetectedMsg{isDark bool}`). Both are standard `tea.Cmd`-returning helpers, no restructuring of the Update loop.
5. **Alt screen + WindowSizeMsg.** Already handled (`m.width/height` set on `WindowSizeMsg`, forwarded to active sub-model). Enabling `MouseMode` on a View does **not** change `WindowSizeMsg` semantics — mouse msgs (`tea.MouseWheelMsg`, `tea.MouseClickMsg`) become additional `tea.Msg` cases in the relevant screen `Update`. Root `Update` already forwards unhandled msgs to the active sub-model via `forwardTo`, so mouse routing is *mostly* free — each screen just adds a `case tea.MouseWheelMsg`.

---

## Section 5 — Implementation Priority (tiered)

### Tier 1 — High value, Low effort (do first)
- **F1 Window title** (#1) — 1 field per View, ~5 lines/screen. S.
- **F2 Spinning step indicators** (#2) — replace static glyph with `m.spinner.View()`; already ticking. S.
- **F5 Global status bar** (#5) — new stateless `components/statusbar.go` + wire into `renderContent`. S.
- **F6 Gradient logo** (#6) — swap `logo.go` 5-style array for `Blend1D(len(lines), Love,Gold,Rose,Pine,Lavender)`. S.
- **F10 Native terminal progress bar** (#10) — `v.ProgressBar` field while on ScreenProgress. S.

### Tier 2 — High value, Medium effort
- **F3 Viewport for dry-run diff** (#3) — `restore.go`: embed `viewport.Model`, size on WindowSizeMsg, wire keys. M.
- **F4 Mouse scroll/click** (#4) — enable `MouseModeCellMotion` on dashboard/restore Views, add `MouseWheelMsg` cases. M.
- **F8 Styled empty/error states** (#8) — branded panels with CTA; refactor empty branches in dashboard/restore/cloud. M (touches 3 screens).

### Tier 3 — Medium value, Low effort
- **F15 Paste support** (#15) — set `DisableBracketedPasteMode=false` (default) and ensure wizard/search consume `tea.PasteMsg`. S.
- **F9 Terminal theme detection** (#9) — one `tea.Cmd` reads default fg color; choose Rose Pine vs a light-friendly palette. M.
- **F16 textinput enhancements** (#16) — wizard `StepName` → `textinput.Model` with `Placeholder`/`CharLimit`. S.

### Tier 4 — High value, High effort (skip unless time)
- **F7 Startup logo reveal** (#7) — delightful but adds a reveal state machine + timing. Defer to a follow-up change OR ship a *static* gradient (F6) now and animate later.
- **F11 harmonica spring** (#11) — smooth progress is nice but bubbles/progress already animates; marginal. Skip.
- **F12 bubbles/list** (#12) — high churn, restore picker works. Defer.
- **F13 bubbles/key help** (#13), **F14 glamour** (#14) — consistency/convenience, not personality. Defer.

**Recommended scope for *this* change:** Tier 1 + Tier 2 + F15. That's **9 affordances**, all with concrete use cases, all testable. F7/F9/F16 optional stretch.

---

## Section 6 — Risk Assessment

1. **Alt-screen tests:** already enabled + `Program.Run()` is never unit-tested (AGENTS.md). Adding `WindowTitle`/`MouseMode`/`ProgressBar` View fields is pure-data on the View struct → **no existing test breakage**; tests assert on `View().Content` (string) and now may also assert `View().WindowTitle`. Safe.
2. **Mouse vs keyboard conflict:** none inherent — mouse msgs are a distinct `tea.Msg` type; keyboard handling is untouched. Risk is **feel**: ensure mouse wheel doesn't scroll when search input is focused (add a `if m.search.IsActive()` guard in dashboard's `MouseWheelMsg` case). Low.
3. **Spinner/progress command complexity:** the Update loop already batches `spinner.Tick` + `progress.SetPercent`. F2 reuses the same `m.spinner` — one source of ticks, no second spinner. Risk is a `TickMsg` storm if two spinners run; mitigated by sharing the single `spinner.Model`. Low.
4. **Gradient rendering across terminals:** `lipgloss.Blend1D` emits truecolor/256color/ansi based on detected profile (lipgloss auto-detects via `colorprofile`, already an indirect dep). Windows Terminal + modern macOS + Linux all support truecolor. Fallback to nearest ANSI is automatic. Risk: terminals with no-color show no gradient (acceptable — degrades to monochrome logo, still readable). Low.
5. **Coverage impact:** every new component and every modified `View`/`Update` needs ≥80% coverage per AGENTS.md. F3 (viewport) and F4 (mouse) add the most test surface. Mitigation: TDD Update/View as pure functions (the established pattern); `MouseWheelMsg` is just a `tea.Msg` struct, trivially constructible in tests. F5 statusbar is a stateless render fn → table-driven test like `help.go`/`menu.go`.

### New test surface (TDD-friendly, all pure functions)
- `components/statusbar_test.go` — table-driven render assertions (widths, screen labels, running-op text).
- `restore_test.go` — extend for viewport sizing + `PgUp/PgDn` + MouseWheel scrolling (assert `viewport.ScrollPercent`/cursor).
- `dashboard_test.go` + `restore_test.go` — assert `View().MouseMode == tea.MouseModeCellMotion` when enabled.
- `model_test.go` — assert `View().WindowTitle` per active screen; assert native `ProgressBar` set only on ScreenProgress.
- `progress_test.go` + `health_test.go` — assert running-step rendered frame equals `m.spinner.View()` (snapshot the spinner's current frame).
- `logo_test.go` — assert `Blend1D` returns `len(lines)` colors and monochrome fallback when no color.

---

## Section 7 — Dependency Audit

From `go.mod`:
- `charm.land/bubbles/v2 v2.1.0` ✅ — provides `viewport`, `list`, `key`, `spinner`, `progress`, `table`, `textinput`. **No new bubbles import needed.**
- `charm.land/bubbletea/v2 v2.0.7` ✅ — provides `tea.View` fields (`WindowTitle`, `MouseMode`, `ProgressBar`, `ForegroundColor/BackgroundColor`, `Cursor`, `DisableBracketedPasteMode`). No upgrade needed.
- `charm.land/lipgloss/v2 v2.0.3` ✅ — provides `Blend1D`/`Blend2D` (Context7-verified API). No upgrade.
- `github.com/charmbracelet/harmonica v0.2.0` — already an **indirect** dep (pulled by bubbles); importing it directly for F11 is zero new download. (F11 skipped anyway.)
- `github.com/charmbracelet/glamour` — **NOT** a dep; F14 (glamour) would add one. **F14 deferred** → no new dep in recommended scope.

**AGENTS.md "prefer stdlib" compliance:** bubbles/bubbletea/lipgloss are already first-class deps; using more of their exported API (viewport, View fields, Blend1D) is *consistent* with existing choices — no new dependency justification required for Tier 1+2. ✅

---

## Section 8 — Concrete Implementation Plan (for design phase)

### F1 — Window title (`internal/tui/model.go`, per screen)
```go
// new helper
func (m Model) currentWindowTitle() string {
    switch m.screen {
    case ScreenProgress:
        if m.progress != nil && m.progress.Running() {
            cur, tot := m.progress.Stats() // add accessor
            return fmt.Sprintf("bak — Backup %d/%d", cur, tot)
        }
        return "bak — Progress"
    case ScreenRestore:
        if m.restore != nil && m.restore.SelectedID != "" {
            return fmt.Sprintf("bak — Restore:%s", m.restore.SelectedID)
        }
        return "bak — Restore"
    // … one case per screen
    }
    return "bak"
}
// in View():
v.WindowTitle = m.currentWindowTitle()
```
Tests: `model_test.go` table asserting `View().WindowTitle` per `m.screen`. Pure.

### F2 — Spinning step indicators (`screens/progress.go`, `screens/health.go`)
```go
// progress.go View(): replace static indicator
case StepRunning:
    indicator = m.spinner.View()   // live rotating frame
// health.go needs its own spinner.Model field (currently shares none);
// add `spinner spinner.Model` to HealthModel, Init() returns m.spinner.Tick,
// Update propagates spinner.TickMsg like progress.go does.
```
Tests: in `progress_test.go`/`health_test.go`, set `m.spinner` to a known frame (call `m.spinner.Update(spinner.TickMsg{})` N times), assert running-row output contains that frame string.

### F3 — Viewport for dry-run (`screens/restore.go`)
```go
type RestoreModel struct {
    // …existing…
    viewport viewport.Model
    vpReady  bool
}
// NewRestoreModel: m.viewport = viewport.New(viewport.WithWidth(80), viewport.WithHeight(10))
// WindowSizeMsg: m.viewport.Width = msg.Width-4; m.viewport.Height = msg.Height-8; m.vpReady = true
// restoreDryRunResultMsg: m.viewport.SetContent(m.DryRunOutput)
// Update case tea.KeyPressMsg in restoreStateDryRun: forward j/k/PgUp/PgDn/g/G to m.viewport.Update
// renderDryRun: b.WriteString(m.viewport.View())
```
Tests: TDD `restore_test.go` — given DryRunOutput of 50 lines and height 10, pressing `PgDn` advances `viewport.ScrollPercent`; pressing `g` returns to top.

### F4 — Mouse (`screens/dashboard.go`, `screens/restore.go`)
```go
// in View(): 
v := tea.NewView(content)
v.MouseMode = tea.MouseModeCellMotion
return v
// in Update, add:
case tea.MouseWheelMsg:
    if m.search.IsActive() { return m, nil }   // dashboard guard
    newTbl, cmd := m.table.Update(msg)          // bubbles table handles wheel
    m.table = newTbl
    return m, cmd
```
Tests: assert `View().MouseMode == tea.MouseModeCellMotion`; construct `tea.MouseWheelMsg{Y: -1}` and assert table cursor moved.

### F5 — Status bar (new `internal/tui/components/statusbar.go`)
```go
// Package-level styles (define in styles/, not here — AGENTS.md).
// stateless render fn:
func RenderStatusBar(version, screenLabel, opLabel string, width int) string
```
Wire in `model.go renderContent()` above the screen content. Tests: `components/statusbar_test.go`, table-driven (narrow width truncates, running-op shows, labels map per screen).

### F6 — Gradient logo (`styles/logo.go`)
```go
func RenderLogo(width int) string {
    // …guard, split lines…
    stops := []lipgloss.Color{ColorLove, ColorGold, ColorRose, ColorPine, ColorLavender}
    grad := lipgloss.Blend1D(len(lines), stops...)
    // render each line i with lipgloss.NewStyle().Foreground(grad[i])
}
```
Tests: `logo_test.go` assert `len(grad) == len(lines)`; assert width<40 returns ""; assert non-empty output contains logo glyphs.

### F10 — Native progress (`model.go View()` while `ScreenProgress`)
```go
if m.screen == ScreenProgress && m.progress != nil && m.progress.Running() {
    v.ProgressBar = m.progress.Percent()  // bubbles progress exposes ratio; map to tea's bar value
}
```
Tests: assert `View().ProgressBar != 0` only on running progress; zero elsewhere.

### F8 — Styled empty states (`screens/dashboard.go`, `restore.go`, `cloud.go`)
Replace `DashboardEmptyStyle.Render("No backups found")` with a bordered panel + CTA line:
```go
styles.EmptyStatePanelStyle.Render(fmt.Sprintf("No backups yet\n\n  press %s to create one", styles.SelectedStyle.Render("b")))
```
Add `EmptyStatePanelStyle` + `EmptyStateCTAStyle` package-level vars to `styles/screens.go`. Tests: assert empty-state output contains the CTA key glyph.

### F15 — Paste (`wizard.go` `StepName`, `search.go`)
```go
// wizard.go Update: add
case tea.PasteMsg:
    m.NameInput += msg.Text
    return m, nil
// View: v.DisableBracketedPasteMode = false  (default, but explicit)
```
Tests: send `tea.PasteMsg{Text: "work-laptop"}`, assert `NameInput == "work-laptop"`.

---

## Section 9 — AGENTS.md Compliance Check

| Rule | How this change complies |
|------|--------------------------|
| Package-level `var` lipgloss styles (AGENTS.md §styles) | New `EmptyStatePanelStyle`, statusbar styles, gradient style lives in `styles/` package scope. No inline `lipgloss.NewStyle()` in any `View()`. |
| New components have Init/Update/View or are stateless render fns | statusbar = stateless render fn (like `help`/`menu`); viewport is a screen-owned sub-model (already has Update/View via bubbles). |
| `WindowSizeMsg` handled in all new/modified models | viewport sizing in restore.go WindowSizeMsg case; statusbar recomputes from m.width. |
| "Terminal too small" guard in all new screens | statusbar has no dedicated screen (overlay); restore keeps existing `IsTooSmall` guard before viewport render. |
| Test coverage ≥80% for `internal/tui/` packages | New test files listed §6; TDD red/green; pure-function Update/View tests, no Program.Run. |
| Rose Pine semantic colors from `internal/tui/styles/` | gradient stops reuse `ColorLove/Gold/Rose/Pine/Lavender`; new styles reference existing semantic colors. |
| bubbletea v2 API (`tea.KeyPressMsg{Code}`, `tea.View` fields) | all new code uses v2 View fields + KeyPressMsg.Code; no v1 `tea.KeyMsg`. |
| bubbles/v2 justification | viewport already shipped in bubbles/v2 — using more existing dep API, no new dependency. |
| No `fmt.Println` for errors; stderr for warnings | no new logging; statusbar/title/viewport write only to View(). |
| Error wrapping `fmt.Errorf("ctx: %w", err)` | restore/cloud error panels render, don't fabricate new wrapped errors (only existing `m.Err` surfaced). |
| DRY — no duplicated utility fns | statusbar is single shared fn; viewport used in one screen (no premature shared component); step-indicator logic stays per-screen (progress owns spinner, health owns its own). |

---

## Section 10 — Summary

### Top 5 features by personality impact (recommended scope)
1. **Window title** (F1) — situational delight, near-zero cost. The tab-strip moment.
2. **Spinning step indicators** (F2) — makes operations feel alive; fixes the "frozen backup" impression.
3. **Global status bar** (F5) — persistent context transforms navigation feel; the single biggest perceived-quality lift for the effort.
4. **Mouse scroll/click** (F4) — meets 2026 user expectations; unblocks scrollable content.
5. **Viewport dry-run + styled empty states** (F3 + F8) — removes the two worst concrete UX moments (unscrollable diff, dead empty strings).

### Top 3 risks
1. **Viewport test surface** (F3) — most new Update logic; mitigated by TDD pure-function tests.
2. **Mouse feel on focused search** (F4) — wheel must be suppressed when search active; simple guard, easy to forget.
3. **Scope creep toward F7 reveal animation** — ship the static gradient (F6) first; animation is a follow-up change to protect review focus (chained-PR spirit, AGENTS.md atomic commits).

### Overall effort estimate
Tier 1 (5): ~S each → ~1–1.5 days. Tier 2 (3): M each → ~2–3 days. F15: S → half a day.
**Total: ~4–5 focused days for a 9-affordance personality upgrade, TDD throughout.**

### Expected personality score
**4/10 → 8/10.** The 2-point gap to 10/10 is deliberately left: it's the startup reveal animation (F7) and spring physics (F11), which are polish-for-polish in a backup tool and better shipped as a later, smaller change to keep this PR reviewable.

---

## Result Contract

```
status: success
executive_summary: Corrected the stale brief — bak-cli's TUI already uses 4 Charm families (textinput, table, spinner, progress) AND alt screen, and the subModel dispatch map already landed (qa-refactor-analysis), so this is an additive personality change, not a refactor-preconditioned one. Mapped 16 Charm affordances against concrete bak-cli use cases via Context7 v2 docs; 9 belong in scope (Tier1: window-title, spinning step indicators, status bar, gradient logo, native terminal progress bar; Tier2: viewport dry-run, mouse scroll/click, styled empty states; Tier3: paste). 4 are explicitly skipped (startup reveal animation, harmonica springs, bubbles/list churn, glamour) to protect review focus and avoid feature-bloat in a CLI tool. Personality is architecturally cheap because v2 moved everything to declarative tea.View fields — each gap is a field assignment in an existing View(). No new dependencies (bubbles/lipgloss already shipped; harmonica already indirect). Expected lift 4/10→8/10 in ~4–5 TDD days.
artifacts:
  - openspec/changes/tui-personality/explore.md
  - engram: sdd/tui-personality/explore
next_recommended: propose
risks:
  - Viewport (F3) adds the most new Update test surface; mitigated by TDD pure-fn tests
  - Mouse wheel must be suppressed when dashboard search is active (feel bug, simple guard)
  - Scope creep toward startup-reveal animation (F7) — defer to a follow-up change; ship static gradient now
skill_resolution: paths-injected
```

<!-- KEY LEARNINGS:
1. bak-cli's TUI audit in the prompt was STALE — it already uses bubbles/textinput, table, spinner, progress (4 families) AND alt screen via v.AltScreen in model.go View(); the subModel dispatch map already landed from qa-refactor-analysis.
2. Bubbletea v2 moved program options (alt screen, mouse, window title, progress bar, paste, theme colors) to DECLARATIVE tea.View FIELDS returned from View() — so personality affordances are field assignments, not tea.WithX options or tea.EnterAltScreen commands.
3. lipgloss v2 provides Blend1D/Blend2D gradient APIs (Context7-verified); replacing bak-cli's 5-band static logo with Blend1D is a ~10-line change in styles/logo.go.
4. go.mod already has bubbles/v2 and lipgloss/v2 first-class and harmonica as an INDIRECT dep — no new dependency needed for the recommended Tier 1+2 scope; glamour would be a NEW dep and is correctly deferred.
5. The two worst concrete UX moments are STATIC step indicators (literal "⠹" string that never rotates, progress.go:50 + health.go:175) and an unscrollable raw dry-run dump (restore.go renderDryRun) — both are cheap, high-personality fixes.
-->