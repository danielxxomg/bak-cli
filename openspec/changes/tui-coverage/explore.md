# Exploration: Wizard Tests & TUI Coverage

## Current Coverage Snapshot (measured against `main`)

| Package                    | Coverage | Target | Gap   | Status |
|----------------------------|----------|--------|-------|--------|
| `internal/tui/`            | 63.1%    | 80%    | -16.9 | FAIL   |
| `internal/tui/screens/`    | 63.8%    | 80%    | -16.2 | FAIL   |
| `internal/tui/components/` | 95.1%    | 80%    | +15.1 | PASS   |
| `internal/tui/styles/`     | 90.9%    | 80%    | +10.9 | PASS   |
| `cmd/`                     | 51.0%    | E2E    | n/a   | n/a*   |

*`cmd/` per AGENTS.md: "MUST NOT unit-test `os.Exit` paths — test via integration/E2E only" and E2E tests live in `tests/e2e/`.

**Root cause of the gap** — `internal/tui/screens/wizard.go` is at **0% coverage across all 11 functions** because the existing wizard tests live in `cmd/wizard_test.go` (black-box from `package cmd`). They cover `internal/tui/screens/` only indirectly via cross-package call, which Go's coverage tool does NOT attribute to the screens package. The tests need to MOVE, not be duplicated.

## Current State

### Where the wizard tests live today

`cmd/wizard_test.go` (312 lines, 17 test functions) — all 16 of them construct and drive a `screens.WizardModel`. Only **one** test (`TestIsTTY_NotTerminal`) targets a cmd-package symbol (`isTTY()` from `cmd/wizard.go`).

The architectural smell: every test in `cmd/wizard_test.go` except one exercises code in a different package. That artificially inflates `cmd/` coverage while leaving `internal/tui/screens/wizard.go` entirely uncovered.

### What `internal/tui/screens/wizard.go` exports

```go
type WizardStep int
const (StepName; StepProvider; StepPreset; StepAdapters; StepCategories; StepConfirm)

type WizardModel struct { ... }  // 14 fields, used via pointer

type ToggleItem struct { Name string; Checked bool }

func NewWizardModel(mode string, providers []string) *WizardModel
func (m *WizardModel) CurrentStep() WizardStep
func (m *WizardModel) ProfileName() string
func (m *WizardModel) Init() tea.Cmd
func (m *WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m *WizardModel) View() tea.View
func MoveCursor(cursor *int, max int, key string)  // package-level, exported

// unexported
func (m *WizardModel) handleEnter() (tea.Model, tea.Cmd)
func (m *WizardModel) handleNavigation(msg tea.KeyPressMsg) (tea.Model, tea.Cmd)
func (m *WizardModel) renderCheckboxList(items []ToggleItem, cursor int) string
func (m *WizardModel) renderConfirmSummary() string
```

Coverage today (`go test -coverprofile ./internal/tui/screens/`):

```
wizard.go:67   NewWizardModel        0.0%
wizard.go:102  CurrentStep           0.0%
wizard.go:109  ProfileName           0.0%
wizard.go:120  Init                  0.0%
wizard.go:126  Update                0.0%
wizard.go:150  handleEnter           0.0%
wizard.go:183  MoveCursor            0.0%
wizard.go:197  handleNavigation      0.0%
wizard.go:235  View                  0.0%
wizard.go:307  renderCheckboxList    0.0%
wizard.go:318  renderConfirmSummary  0.0%
```

## Affected Areas

- `cmd/wizard_test.go` — 17 tests, 16 of which must move out
- `cmd/wizard.go` — `isTTY()` (stays in cmd, tested by `TestIsTTY_NotTerminal`)
- `internal/tui/screens/wizard_test.go` — NEW file to host the moved tests
- `internal/tui/model.go` — `initRestore` (0%), `initProfiles` (0%), `initCloud` (25%), `Update` (66.7%), `handleKey` (67.1%), `View` (79.4%), `initSettings` (83.3%)
- `internal/tui/screens/cloud.go` — `Update` (53.8%), `View` (50.0%)
- `internal/tui/screens/profiles.go` — `Update` (62.2%), `handleKey` (72%), `View` (53.8%), `renderError` (0%)
- `internal/tui/screens/restore.go` — `Update` (58.3%), `handleKey` (47.8%), `View` (47.1%), `renderErrorState` (0%), `renderBackupList` (0%), `renderRunning` (0%), `renderDone` (0%)
- `internal/tui/screens/settings.go` — `Init` (0%)
- `internal/tui/screens/health.go` — `Init` (0%)
- `internal/tui/screens/menu.go` — `RenderMainMenu` (87.5%, only on missing-banner branch)
- `internal/tui/components/search.go` — `IsActive` (0%)
- `internal/tui/components/toast.go` — `Update` (88.9%)
- `internal/tui/components/modal.go` — `NewModalWithFooter`/`Show`/`Update` (83-92%, edge cases missing)
- `internal/tui/styles/logo.go` — `RenderLogo` (86.7%)

## Wizard Tests to Migrate

All tests in `cmd/wizard_test.go` import `screens` and construct `screens.NewWizardModel`. The split:

| Test                          | Action   | Rationale |
|-------------------------------|----------|-----------|
| `TestWizardModel_Init`        | MOVE     | Tests `screens.WizardModel` |
| `TestWizardModel_StepTransitions` | MOVE | Tests `screens.handleEnter` step transitions |
| `TestWizardModel_ExitKeys`    | MOVE     | Tests `screens.Update` for `ctrl+c`/`esc` |
| `TestWizardModel_NameStep_FirstStep` | MOVE | Tests `screens.NewWizardModel` initial step |
| `TestWizardModel_NameStep_EnterAdvances` | MOVE | Tests `screens.handleEnter` name→provider |
| `TestWizardModel_NameStep_Typing` | MOVE | Tests `screens.handleNavigation` name input |
| `TestWizardModel_NameStep_Backspace` | MOVE | Same, backspace path |
| `TestWizardModel_NameStep_BackspaceOnEmpty` | MOVE | Edge: empty + backspace |
| `TestWizardModel_NameStep_NamePersistsAcrossSteps` | MOVE | State-persist regression |
| `TestWizardModel_ProfileName` | MOVE     | Pure function `ProfileName()` (3 sub-cases) |
| `TestWizardModel_View_ContainsTitle` | MOVE | Tests `screens.View` |
| `TestWizardModel_View_QuittingEmpty` | MOVE | Tests `screens.View` quitting branch |
| `TestWizardModel_ProviderSelection` | MOVE | Tests `screens.MoveCursor` + provider step |
| `TestIsTTY_NotTerminal`       | **STAY** | Tests `cmd.isTTY()` — cmd-package symbol |
| `TestWizardModel_Update_WindowSize` | MOVE | Tests `screens.Update` for `WindowSizeMsg` |
| `TestWizardModel_Update_WindowSize_SecondResize` | MOVE | Same, second-resize regression |
| `TestMoveCursor`              | MOVE     | Tests `screens.MoveCursor` directly (13 sub-cases) |

**Result**: 16 tests move to `internal/tui/screens/wizard_test.go`; 1 test stays in `cmd/wizard_test.go` (it is the only thing in `cmd/wizard.go` and currently the only reason `cmd/wizard_test.go` exists — can be moved to `cmd/tty_test.go` or left where it is; recommend renaming the file to `cmd/wizard_test.go` → `cmd/tty_test.go` since the file is then about TTY detection only, or keep it for proximity to `cmd/wizard.go`).

**Net effect on coverage** (estimate): `internal/tui/screens/wizard.go` 0% → ~95% (some lines in `renderConfirmSummary`/`View` with style-rendering branches may stay partially uncovered without golden-file testing, but pure-function tests will cover everything that doesn't require ANSI snapshot comparison). This single change moves `internal/tui/screens/` from **63.8% → ~85%+** and `internal/tui/` from **63.1% → ~75%**.

## Coverage Gaps Beyond the Wizard Migration

After the migration, the remaining files still below 80%:

| File                                          | Current | Missing Test Focus |
|-----------------------------------------------|---------|--------------------|
| `internal/tui/model.go` `initRestore`         | 0%      | Build a `Model` with mock `ListBackups`/`RunRestore` deps, call `initRestore()`, assert returned `RestoreModel` has wired closures |
| `internal/tui/model.go` `initProfiles`        | 0%      | Same pattern; also test `SaveProfile` field injection post-construction |
| `internal/tui/model.go` `initCloud`           | 25%     | Cover all three branches: `statusFn` set vs nil, error path from `Status` |
| `internal/tui/model.go` `Update` (66.7%)      | 66.7%   | Missing cases: unknown msg type, `ScreenShortcuts` flow, `ScreenBackMsg` from sub-models |
| `internal/tui/model.go` `handleKey` (67.1%)   | 67.1%   | Per-screen key dispatch — many `case` arms unhit (Settings, Profiles, Cloud) |
| `internal/tui/model.go` `View` (79.4%)        | 79.4%   | Per-screen view rendering — covers main 7 screens; edge branches for empty/zero width |
| `internal/tui/screens/cloud.go` `Update` (53.8%) | 53.8% | Error-from-`statusFn` path, `Status` `tea.Msg` handler with `Status=Disconnected` |
| `internal/tui/screens/cloud.go` `View` (50.0%)   | 50.0% | Both states (provider configured / not configured) |
| `internal/tui/screens/profiles.go` `Update` (62.2%) | 62.2% | `wizardResultMsg` happy path, error path, switch/delete flows |
| `internal/tui/screens/profiles.go` `View` (53.8%)  | 53.8% | Empty list, error state, populated list |
| `internal/tui/screens/profiles.go` `renderError` (0%) | 0% | Trivial — feed a non-nil `err` and assert substring |
| `internal/tui/screens/restore.go` `Update` (58.3%) | 58.3% | Dry-run command, restore command, error path, list-load error |
| `internal/tui/screens/restore.go` `View` (47.1%)  | 47.1% | All 4 view states (empty / list / dry-run / confirm / running / done) |
| `internal/tui/screens/restore.go` `renderErrorState` (0%) | 0% | One-liner |
| `internal/tui/screens/restore.go` `renderBackupList` (0%) | 0% | Empty + populated cases |
| `internal/tui/screens/restore.go` `renderRunning` (0%) | 0% | One-liner |
| `internal/tui/screens/restore.go` `renderDone` (0%)   | 0% | One-liner |
| `internal/tui/screens/settings.go` `Init` (0%) | 0% | Returns nil — needs explicit `TestSettingsModel_Init_ReturnsNil` to hit |
| `internal/tui/screens/health.go` `Init` (0%)  | 0% | Same — explicit `TestHealthModel_Init_ReturnsNil` |
| `internal/tui/screens/menu.go` `RenderMainMenu` (87.5%) | 87.5% | Banner-present branch missing (when `banner != ""`) |
| `internal/tui/components/search.go` `IsActive` (0%) | 0% | Trivial — call after `Activate()` and after `Deactivate()` |
| `internal/tui/components/toast.go` `Update` (88.9%) | 88.9% | Tick-expired path |
| `internal/tui/components/modal.go` (mixed 83-92%) | mixed | Edge cases: empty `content`, footer-only, escape dismissal |
| `internal/tui/styles/logo.go` `RenderLogo` (86.7%) | 86.7% | Width 40-49 transition (the cutoff branch) |

## Test Patterns Observed

Reference patterns from the existing test suite:

- **White-box testing** for all `internal/tui/screens/` tests — they use `package screens` (not `screens_test`), so unexported fields like `m.NameInput`, `m.ProviderCursor`, `m.Quitting` can be asserted directly. This is the standard pattern (see `cloud_test.go:4`, `dashboard_test.go:4`, etc.).
- **Pure-function `Update` / `View` testing** — drive the model with `tea.KeyPressMsg{Code: r, Text: string(r)}` and `tea.WindowSizeMsg{Width: W, Height: H}`, then read `View().Content` as a string. No `tea.Program.Run()` ever invoked (AGENTS.md: "MUST NOT test `bubbletea.Program.Run()` directly").
- **Table-driven tests** — `[]struct{ name string; ... }` per AGENTS.md rule "MUST use table-driven tests".
- **Spy closures as test doubles** — `dispatch_test.go:50-65` shows the pattern: assign a closure to `deps.RunBackup` that records the call and delegates. No mock-generation tools used.
- **String-content assertions on `View()`** — `cloud_test.go:289` and `dashboard_test.go` check that rendered output contains substrings like "github", "Connected", "12" — not exact snapshots, so the tests stay resilient to style changes.
- **`MoveCursor` test pattern** — direct function call with a pointer and a key string; the test table already in `cmd/wizard_test.go:276-311` is the canonical example and can be copied verbatim.

## Approach Options

### Approach A: Move tests as-is (lowest risk)

Copy the 16 wizard tests verbatim to `internal/tui/screens/wizard_test.go` with `package screens` and `import "github.com/danielxxomg/bak-cli/internal/tui/screens"` removed (replaced with direct field access since we're now in the same package). Then delete `cmd/wizard_test.go` and recreate it with only `TestIsTTY_NotTerminal`.

- Pros: smallest diff, preserves all 17 tests as they are, immediately lifts `wizard.go` from 0% to ~95%
- Cons: doesn't improve the `model.go` / `cloud.go` / `profiles.go` / `restore.go` gaps. Leaves `internal/tui/` aggregate at ~75% — still below 80% target.

### Approach B: Move + backfill the remaining ~5pp

Do A, then add focused table-driven tests for the low-hanging 0% functions in `model.go` (`initRestore`, `initProfiles`, `initCloud`) and the four 0% render helpers in `restore.go` / `profiles.go` / `screens/cloud.go` View + Update.

- Pros: hits the 80% target for both `internal/tui/` and `internal/tui/screens/`. Cheap because most of these are one-line render helpers or pure DI builders — test doubles are already in place.
- Cons: a bit more work; the `handleKey` dispatch (67.1%) needs per-screen cases that are time-consuming to enumerate.

### Approach C: Move + full backfill (over-scope)

Approach B plus extensive per-screen `Update` / `View` coverage for every model.

- Pros: every package at >90%
- Cons: large PR, higher risk of brittleness. AGENTS.md says "MUST achieve ≥80% coverage for all `internal/tui/` packages" — 80% is the target, not 100%. YAGNI applies.

## Recommendation

**Approach B**. The wizard move is a one-step win that fixes the most embarrassing 0% file in the repo. The remaining ~5pp on `internal/tui/` is mechanical — every 0% function identified above is either a one-line render helper or a pure dependency-injection builder that takes a `Deps` struct. Both patterns are already well-tested elsewhere in the codebase (see `dispatch_test.go` for the DI pattern, `cloud_test.go` for the render-helper pattern), so the test code is mostly copy-adapt.

Estimated final coverage after Approach B:
- `internal/tui/`: 63.1% → **~82%**
- `internal/tui/screens/`: 63.8% → **~92%**
- `internal/tui/components/`: 95.1% (unchanged, already passes)
- `internal/tui/styles/`: 90.9% (unchanged, already passes)

## Risks

- **No real coverage risk** — all existing tests are being moved, not removed. They will continue to pass and now contribute to the package they belong to.
- **Import-cycle risk if `wizard_test.go` is in `package screens` instead of `screens_test`** — none; every other screens test uses `package screens` already, so this matches the existing convention.
- **Brittleness of string-substring `View()` assertions** — low; the existing tests in `cloud_test.go` and `dashboard_test.go` use the same approach and have not been a problem.
- **`renderConfirmSummary` style paths** — lipgloss styling means some branches may stay uncovered without golden-file tests. Acceptable; the 80% threshold accommodates this.

## Ready for Proposal

**Yes.** The plan is concrete, low-risk, and the success criterion is unambiguous: `go test -cover ./internal/tui/...` reports ≥80% for the screens package after the migration, and the wizard.go file moves from 0% to ≥80% in its new home.
