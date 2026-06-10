# Tasks: v2-migration — bubbletea & lipgloss v1 → v2

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~55 source lines + go.mod/go.sum (~20 dep lines) ≈ 75 total |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR (all changes coupled — code does not compile with mixed v1/v2) |
| Delivery strategy | single-pr |
| Chain strategy | n/a |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: n/a
400-line budget risk: Low

## Phase 1: Foundation — Dependencies & API Verification

- [x] 1.1 Run `go get charm.land/bubbletea/v2@v2.0.7 charm.land/lipgloss/v2@v2.0.3` to fetch v2 modules
  - **Files**: `go.mod`, `go.sum`
  - **Lines**: ~4 (go.mod) + go.sum additions

- [x] 1.2 Run `go doc charm.land/bubbletea/v2.KeyPressMsg` and `go doc charm.land/bubbletea/v2.View` to verify struct fields (`Code`, `Text`, `Mod`) and View accessor method before writing code
  - **Files**: none (verification gate — abort if fields differ from exploration)

- [x] 1.3 Run `go mod tidy` to clean up indirect dependencies after adding v2 modules
  - **Files**: `go.mod`, `go.sum`

## Phase 2: Import Updates (Simple Files)

- [x] 2.1 Update `cmd/login.go` line 6: `tea "github.com/charmbracelet/bubbletea"` → `tea "charm.land/bubbletea/v2"`
  - **Files**: `cmd/login.go`
  - **Lines**: 1

- [x] 2.2 Update `cmd/profile.go` line 6: `tea "github.com/charmbracelet/bubbletea"` → `tea "charm.land/bubbletea/v2"`
  - **Files**: `cmd/profile.go`
  - **Lines**: 1

## Phase 3: Core Implementation — pick.go

- [x] 3.1 Update imports in `cmd/pick.go` lines 7-8: bubbletea → `charm.land/bubbletea/v2`, lipgloss → `charm.land/lipgloss/v2`
  - **Files**: `cmd/pick.go`
  - **Lines**: 2

- [x] 3.2 Change key type assertion on line 53: `msg.(tea.KeyMsg)` → `msg.(tea.KeyPressMsg)`
  - **Files**: `cmd/pick.go`
  - **Lines**: 1

- [x] 3.3 Change `View()` return type on line 83: `func (m pickModel) View() string` → `func (m pickModel) View() tea.View`; wrap line 85 `return ""` → `return tea.NewView("")`; wrap line 116 `return b.String()` → `return tea.NewView(b.String())`
  - **Files**: `cmd/pick.go`
  - **Lines**: 3

## Phase 4: Core Implementation — wizard.go

- [x] 4.1 Update imports in `cmd/wizard.go` lines 8-9: bubbletea → `charm.land/bubbletea/v2`, lipgloss → `charm.land/lipgloss/v2`
  - **Files**: `cmd/wizard.go`
  - **Lines**: 2

- [x] 4.2 Restructure `Update()` key handling (lines 100-111): change `msg.(tea.KeyMsg)` → `msg.(tea.KeyPressMsg)`; replace `switch msg.Type` with `switch msg.String()`; change `case tea.KeyCtrlC, tea.KeyEsc:` → `case "ctrl+c", "esc":`; change `case tea.KeyEnter:` → `case "enter":`
  - **Files**: `cmd/wizard.go`
  - **Lines**: 4

- [x] 4.3 Change `handleNavigation` parameter on line 145: `msg tea.KeyMsg` → `msg tea.KeyPressMsg`
  - **Files**: `cmd/wizard.go`
  - **Lines**: 1

- [x] 4.4 Change `View()` return type on line 207: `func (m *wizardModel) View() string` → `func (m *wizardModel) View() tea.View`; wrap line 209 `return ""` → `return tea.NewView("")`; wrap line 256 `return b.String()` → `return tea.NewView(b.String())`
  - **Files**: `cmd/wizard.go`
  - **Lines**: 3

## Phase 5: Test Adaptation — pick_test.go

- [x] 5.1 Update import in `cmd/pick_test.go` line 7: bubbletea → `charm.land/bubbletea/v2`
  - **Files**: `cmd/pick_test.go`
  - **Lines**: 1

- [x] 5.2 Rewrite 5 KeyMsg constructors to KeyPressMsg:
  - Line 52: `tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}` → `tea.KeyPressMsg{Code: 'q'}`
  - Line 74: `tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}` → `tea.KeyPressMsg{Code: 'j'}`
  - Line 92: `tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}` → `tea.KeyPressMsg{Code: 'k'}`
  - Line 110: `tea.KeyMsg{Type: tea.KeySpace}` → `tea.KeyPressMsg{Code: ' '}`
  - Line 129: `tea.KeyMsg{Type: tea.KeyEnter}` → `tea.KeyPressMsg{Code: tea.KeyEnter}`
  - **Files**: `cmd/pick_test.go`
  - **Lines**: 5

- [x] 5.3 Fix View() test assertions (lines 150-159): `m.View()` now returns `tea.View` — extract string via `.Content` accessor for `strings.Contains` checks
  - **Files**: `cmd/pick_test.go`
  - **Lines**: 2

## Phase 6: Test Adaptation — wizard_test.go

- [x] 6.1 Update import in `cmd/wizard_test.go` line 7: bubbletea → `charm.land/bubbletea/v2`
  - **Files**: `cmd/wizard_test.go`
  - **Lines**: 1

- [x] 6.2 Rewrite 5 KeyMsg constructors to KeyPressMsg:
  - Line 35: `tea.KeyMsg{Type: tea.KeyEnter}` → `tea.KeyPressMsg{Code: tea.KeyEnter}`
  - Line 44: `tea.KeyMsg{Type: tea.KeyCtrlC}` → `tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}`
  - Line 58: `tea.KeyMsg{Type: tea.KeyEsc}` → `tea.KeyPressMsg{Code: tea.KeyEsc}`
  - Line 99: `tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}` → `tea.KeyPressMsg{Code: 'j'}`
  - Line 106: `tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}` → `tea.KeyPressMsg{Code: 'k'}`
  - **Files**: `cmd/wizard_test.go`
  - **Lines**: 5

- [x] 6.3 Fix View() test assertions (lines 70-74, 81-84): `m.View()` now returns `tea.View` — extract string via `.Content` accessor for `strings.Contains` and equality checks
  - **Files**: `cmd/wizard_test.go`
  - **Lines**: 2

## Phase 7: Verification

- [x] 7.1 Run `go mod tidy` — ensure clean dependency graph with no v1 remnants
  - **Verify**: `grep charmbracelet/bubbletea go.mod` returns nothing; `grep charmbracelet/lipgloss go.mod` returns nothing

- [x] 7.2 Run `go build ./...` — must compile with zero errors
  - **Verify**: exit code 0

- [x] 7.3 Run `go test ./cmd/... -count=1` — all existing tests pass with no test logic changes beyond API adaptation
  - **Verify**: all PASS, zero failures

- [x] 7.4 Run `go vet ./...` — clean, no warnings
  - **Verify**: exit code 0, no output

## Commit

Single atomic commit after all phases pass:

```
chore: migrate bubbletea and lipgloss from v1 to v2
```

All changes are coupled — code does not compile with mixed v1/v2 imports. One commit keeps `git bisect` clean and enables trivial `git revert` rollback.
