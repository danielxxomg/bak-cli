## Exploration: v2-migration — bubbletea v1→v2 & lipgloss v1→v2

### Current State

**Current dependency versions (go.mod):**
- `github.com/charmbracelet/bubbletea v1.3.10`
- `github.com/charmbracelet/lipgloss v1.1.0`

**Target dependency versions (verified available):**
- `charm.land/bubbletea/v2 v2.0.7` (latest stable)
- `charm.land/lipgloss/v2 v2.0.3` (latest stable)
- `charm.land/bubbles/v2 v2.1.0` (available, NOT currently used by project)

**Key discovery:** Charmbracelet moved their Go module path from `github.com/charmbracelet/*` to `charm.land/*` for v2. This is a completely new import path, not just a version bump.

**Files using bubbletea/lipgloss (6 files, all in `cmd/`):**

| File | bubbletea | lipgloss | Complexity |
|------|-----------|----------|------------|
| `cmd/pick.go` | ✅ Model + Update + View + NewProgram | ✅ Styles | High |
| `cmd/pick_test.go` | ✅ KeyMsg construction | — | High |
| `cmd/wizard.go` | ✅ Model + Update + View + Key constants | ✅ Styles | High |
| `cmd/wizard_test.go` | ✅ KeyMsg construction | — | High |
| `cmd/login.go` | ✅ NewProgram + Run only | — | Low |
| `cmd/profile.go` | ✅ NewProgram + Run only | — | Low |

**Files NOT affected:** `cmd/root.go`, `cmd/deps.go`, all `internal/` packages — zero bubbletea/lipgloss imports.

---

### v2 Feature Benefits

1. **Declarative View** — `View()` returns `tea.View` struct with `AltScreen`, `MouseMode`, etc. No more program-level options like `tea.WithAltScreen()`.
2. **Key press/release separation** — `tea.KeyMsg` is now an interface with `tea.KeyPressMsg` and `tea.KeyReleaseMsg` concrete types. Enables key release handling if needed.
3. **Better key matching** — `msg.Code` (rune) + `msg.Mod` (modifier flags) for precise key detection. `msg.Text` (string) replaces `msg.Runes` ([]rune).
4. **Mouse event split** — `tea.MouseMsg` → `tea.MouseClickMsg`, `tea.MouseReleaseMsg`, etc. More precise mouse handling.
5. **lipgloss Color as `color.Color`** — Standard library interop. `lipgloss.Color("12")` now returns `color.Color` instead of a custom string type.

---

### Breaking Changes Inventory (File-by-File)

#### 1. `cmd/pick.go` — 8 changes

| Line(s) | Current (v1) | Required (v2) | Why |
|---------|-------------|---------------|-----|
| 7 | `tea "github.com/charmbracelet/bubbletea"` | `tea "charm.land/bubbletea/v2"` | Import path changed |
| 8 | `"github.com/charmbracelet/lipgloss"` | `"charm.land/lipgloss/v2"` | Import path changed |
| 53 | `msg.(tea.KeyMsg)` | `msg.(tea.KeyPressMsg)` | KeyMsg is now interface; PresserMsg for presses |
| 83 | `func (m pickModel) View() string` | `func (m pickModel) View() tea.View` | View return type changed |
| 85 | `return ""` | `return tea.NewView("")` | Must return tea.View |
| 116 | `return b.String()` | `return tea.NewView(b.String())` | Must return tea.View |
| 90-96 | `lipgloss.Color("12")`, etc. | No change needed | `lipgloss.Color()` now returns `color.Color`; Foreground() accepts it |
| 151 | `tea.NewProgram(m)` | `tea.NewProgram(m)` | ✅ No change |

**Net: 5 line changes + 2 import changes**

#### 2. `cmd/pick_test.go` — 7 changes

| Line(s) | Current (v1) | Required (v2) | Why |
|---------|-------------|---------------|-----|
| 7 | `tea "github.com/charmbracelet/bubbletea"` | `tea "charm.land/bubbletea/v2"` | Import path changed |
| 52 | `tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}` | `tea.KeyPressMsg{Code: 'q', Text: "q"}` | KeyMsg→KeyPressMsg; Type→Code; Runes→Text(string) |
| 74 | `tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}` | `tea.KeyPressMsg{Code: 'j', Text: "j"}` | Same pattern |
| 92 | `tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}` | `tea.KeyPressMsg{Code: 'k', Text: "k"}` | Same pattern |
| 110 | `tea.KeyMsg{Type: tea.KeySpace}` | `tea.KeyPressMsg{Code: ' ', Text: " "}` | KeySpace → literal ' ' |
| 129 | `tea.KeyMsg{Type: tea.KeyEnter}` | `tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\r"}` | KeyEnter constant may still exist as rune |

**⚠️ GOTCHA:** Exact `KeyPressMsg` struct field names must be verified after `go get`. The fields `Code` (rune), `Text` (string), and `Mod` (modifier) are confirmed from the upgrade guide, but there may be additional required fields. **Implementation phase MUST run `go doc charm.land/bubbletea/v2.KeyPressMsg` first.**

**Net: 6 struct constructions + 1 import**

#### 3. `cmd/wizard.go` — 12+ changes

| Line(s) | Current (v1) | Required (v2) | Why |
|---------|-------------|---------------|-----|
| 8 | `tea "github.com/charmbracelet/bubbletea"` | `tea "charm.land/bubbletea/v2"` | Import path |
| 9 | `"github.com/charmbracelet/lipgloss"` | `"charm.land/lipgloss/v2"` | Import path |
| 100-101 | `msg.(tea.KeyMsg)` + `switch msg.Type` | `msg.(tea.KeyPressMsg)` + `switch msg.String()` | KeyMsg→KeyPressMsg; Type field removed |
| 102 | `case tea.KeyCtrlC, tea.KeyEsc:` | `case "ctrl+c", "esc":` | Key constants removed; use string matching |
| 106 | `case tea.KeyEnter:` | `case "enter":` | String matching |
| 145 | `func ... handleNavigation(msg tea.KeyMsg)` | `func ... handleNavigation(msg tea.KeyPressMsg)` | Parameter type |
| 148,160,172,188 | `switch msg.String()` | `switch msg.String()` | ✅ No change (already uses String()) |
| 207 | `func (m *wizardModel) View() string` | `func (m *wizardModel) View() tea.View` | View return type |
| 209 | `return ""` | `return tea.NewView("")` | Must return tea.View |
| 256 | `return b.String()` | `return tea.NewView(b.String())` | Must return tea.View |

**Net: ~10 logic changes + 2 import changes**

#### 4. `cmd/wizard_test.go` — 7 changes

| Line(s) | Current (v1) | Required (v2) | Why |
|---------|-------------|---------------|-----|
| 7 | `tea "github.com/charmbracelet/bubbletea"` | `tea "charm.land/bubbletea/v2"` | Import path |
| 35 | `tea.KeyMsg{Type: tea.KeyEnter}` | `tea.KeyPressMsg{Code: tea.KeyEnter, Text: "\r"}` | KeyMsg→KeyPressMsg |
| 44 | `tea.KeyMsg{Type: tea.KeyCtrlC}` | `tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}` | Ctrl key matching changed |
| 58 | `tea.KeyMsg{Type: tea.KeyEsc}` | `tea.KeyPressMsg{Code: tea.KeyEsc, Text: "\x1b"}` | KeyEsc → Code field |
| 99 | `tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}` | `tea.KeyPressMsg{Code: 'j', Text: "j"}` | Runes→Text |
| 106 | `tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}` | `tea.KeyPressMsg{Code: 'k', Text: "k"}` | Runes→Text |

**⚠️ GOTCHA:** `tea.KeyCtrlC` is gone. Must use `Code: 'c', Mod: tea.ModCtrl`. Need to verify `tea.KeyEsc` and `tea.KeyEnter` still exist as rune constants in v2.

**Net: 6 struct constructions + 1 import**

#### 5. `cmd/login.go` — 1 change

| Line(s) | Current (v1) | Required (v2) | Why |
|---------|-------------|---------------|-----|
| 6 | `tea "github.com/charmbracelet/bubbletea"` | `tea "charm.land/bubbletea/v2"` | Import path |
| 87 | `tea.NewProgram(m)` | `tea.NewProgram(m)` | ✅ No change |
| 88 | `p.Run()` | `p.Run()` | ✅ No change |

**Net: 1 import change only**

#### 6. `cmd/profile.go` — 1 change

| Line(s) | Current (v1) | Required (v2) | Why |
|---------|-------------|---------------|-----|
| 6 | `tea "github.com/charmbracelet/bubbletea"` | `tea "charm.land/bubbletea/v2"` | Import path |
| 211 | `tea.NewProgram(m)` | `tea.NewProgram(m)` | ✅ No change |
| 212 | `p.Run()` | `p.Run()` | ✅ No change |

**Net: 1 import change only**

---

### Key Field Mapping (v1 → v2)

Critical reference for test rewrites:

| v1 Field/Constant | v2 Equivalent | Notes |
|-------------------|---------------|-------|
| `tea.KeyMsg` (type) | `tea.KeyPressMsg` (concrete) | KeyMsg is now an interface |
| `msg.Type` | `msg.Code` (rune) | Field rename + type change |
| `msg.Runes` ([]rune) | `msg.Text` (string) | Type changed |
| `msg.Alt` | `msg.Mod` | Use `msg.Mod.Contains(tea.ModAlt)` |
| `tea.KeyRunes` | Removed | Check `len(msg.Text) > 0` |
| `tea.KeyCtrlC` | String `"ctrl+c"` or `Code:'c', Mod:tea.ModCtrl` | Constants removed |
| `tea.KeyEsc` | String `"esc"` or `Code:tea.KeyEsc` | Verify constant exists |
| `tea.KeyEnter` | String `"enter"` or `Code:tea.KeyEnter` | Verify constant exists |
| `tea.KeySpace` | String `"space"` or `Code:' '` | Verify constant exists |
| `View() string` | `View() tea.View` | Return type changed |
| `return "str"` | `return tea.NewView("str")` | Wrap in tea.View |

---

### lipgloss Changes

**Import path:** `github.com/charmbracelet/lipgloss` → `charm.land/lipgloss/v2`

**API compatibility (for this project's usage):**

| API | v1 | v2 | Change Needed? |
|-----|----|----|----------------|
| `lipgloss.NewStyle()` | ✅ | ✅ | No |
| `.Bold(true)` | ✅ | ✅ | No |
| `.Foreground(lipgloss.Color("12"))` | ✅ | ✅ | No (Color returns color.Color now, Foreground accepts it) |
| `lipgloss.Color("12")` | Returns `lipgloss.Color` (string type) | Returns `color.Color` (interface) | Transparent |

**This project only uses:** `NewStyle()`, `Bold()`, `Foreground()`, `Color()` — all compatible.

**⚠️ No lipgloss code logic changes needed beyond the import path.**

---

### go.mod Changes

```diff
 require (
-    github.com/charmbracelet/bubbletea v1.3.10
-    github.com/charmbracelet/lipgloss v1.1.0
+    charm.land/bubbletea/v2 v2.0.7
+    charm.land/lipgloss/v2 v2.0.3
 )
```

**Indirect dependency changes expected:**
- `github.com/charmbracelet/x/ansi` → may update
- `github.com/charmbracelet/x/cellbuf` → may update
- `github.com/charmbracelet/x/term` → may update
- `github.com/charmbracelet/colorprofile` → may update
- `github.com/muesli/termenv` → may be replaced by charmbracelet's colorprofile
- New indirect deps from `charm.land/*` modules

After updating imports, run `go mod tidy` to resolve.

---

### Migration Complexity Assessment

**Overall: MEDIUM**

| Category | Rating | Rationale |
|----------|--------|-----------|
| Import changes | Trivial | Mechanical find-replace across 6 files |
| Key handling (pick.go) | Low | Already uses `msg.String()` matching — just change type assertion |
| Key handling (wizard.go) | Medium | Uses `msg.Type` switch with constants — must restructure to string matching |
| View return type | Low | Wrap returns with `tea.NewView()` — mechanical |
| Test constructors | Medium-High | All `tea.KeyMsg{...}` must be rewritten; exact API must be verified post-install |
| lipgloss | Trivial | Import path only |
| go.mod / dependencies | Low | `go get` + `go mod tidy` |

**Estimated effort:** 2-3 focused hours for implementation + test verification.

---

### Risk Matrix

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| `KeyPressMsg` fields differ from expected | Medium | Medium | Run `go doc` immediately after `go get`; verify struct fields before writing code |
| `tea.KeyEsc`/`tea.KeyEnter` constants removed in v2 | Low | Low | Fall back to string matching (`"esc"`, `"enter"`) which is documented as canonical |
| `charm.land` module proxy issues on Windows | Low | Low | Verified `go list -m -versions` works; modules exist in proxy |
| `tea.View` struct access in tests (for View() assertions) | Medium | Low | `tea.View` likely has `.String()` or content accessor; verify with `go doc` |
| Indirect dependency conflicts (termenv vs colorprofile) | Low | Medium | `go mod tidy` should resolve; may need `go get` for specific versions |
| `wizardModel` pointer receiver + `tea.Model` interface | None | — | v2 Model interface is same `(Init, Update, View)` — pointer receivers work |

---

### Recommendation

**Proceed with migration.** The changes are well-scoped and mechanical. The project's TUI usage is relatively simple (no mouse handling, no alt screen, no bubbles components), which keeps risk low.

**Suggested implementation order:**
1. `go get charm.land/bubbletea/v2@v2.0.7 charm.land/lipgloss/v2@v2.0.3`
2. Run `go doc charm.land/bubbletea/v2.KeyPressMsg` to verify struct fields
3. Update `go.mod` imports (remove old, add new)
4. Update `cmd/login.go` and `cmd/profile.go` (import-only changes — quick wins)
5. Update `cmd/pick.go` (key handling + View return type)
6. Update `cmd/wizard.go` (most complex — key type switch restructure + View)
7. Update `cmd/pick_test.go` (KeyMsg construction)
8. Update `cmd/wizard_test.go` (KeyMsg construction)
9. `go mod tidy` + `go build ./...` + `go test ./cmd/...`
10. Fix any compilation errors from unexpected API differences

---

### Skills to Inject in Later Phases

| Phase | Skill | Why |
|-------|-------|-----|
| Apply | `bubbletea` | v2 patterns, View struct, KeyPressMsg handling |
| Apply | `golang-pro` | Go idioms, error handling, test patterns |
| Verify | `bubbletea-code-review` | Elm architecture compliance, false-positive avoidance |
| Verify | `go-testing` | Table-driven tests, test verification |

---

### Ready for Proposal

**Yes.** The exploration is complete with:
- ✅ All 6 affected files identified and inventoried
- ✅ Exact line-by-line change mapping with before/after
- ✅ v2 versions verified as available (`v2.0.7` / `v2.0.3`)
- ✅ Risk matrix with mitigations
- ✅ Implementation order defined
- ⚠️ One open item: `KeyPressMsg` exact struct fields need verification post-`go get` (low risk, documented fields are `Code`, `Text`, `Mod`)
