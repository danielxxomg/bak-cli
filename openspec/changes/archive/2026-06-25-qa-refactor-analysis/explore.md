# Exploration: qa-refactor-analysis

**Change**: qa-refactor-analysis
**Mode**: openspec, interactive, Strict TDD ACTIVE
**Date**: 2026-06-24
**Explorer**: sdd-explore (fresh-context, evidence-based)
**Tools**: golangci-lint v2.12.2 (gocognit, funlen, nestif); manual code read

## Executive Summary

Ran gocognit/funlen/nestif live against the current tree. The current
**non-test** violation count is **31** (15 funlen + 8 gocognit + 8 nestif),
NOT the 44 quoted in the prior-cycle hand-off. The reconciliation: the prior
"44 test-excluded" figure appears to have been measured before the 3 DRY
consolidations of ci-hardening-v2 (cloud Pull, cmd loadExcludes, tui
mapBackupInfo) removed several funlen-class offenders, and/or counted
test-included files. **This exploration uses the current ground-truth list.**

The headline finding: the codebase has **a duplicated backup engine**. There
are TWO parallel ~230-line orchestrators of the same backup workflow:
`internal/backup/engine.go Engine.Run` (gocognit 79) used by the TUI picker
path, and `internal/actions/backup.go BackupAction.Run` (gocognit 80) used by
the CLI path. They are semantic clones — `dupl` cannot see them because they
differ structurally (`a.FS` vs `os.*`, helper signatures, error prefixes).
This single consolidation removes ~150 duplicated lines and TWO of the three
SEVERE functions.

After reading all offenders, I count **7 genuine refactoring opportunities**
(tiered 1–3) and 3 prior GGA findings (still open). Two SEVERE functions are
ACCIDENTAL complexity (duplicated/forwarding boilerplate); one (TUI
`Model.Update` = 84) is a hybrid — inherent domain branching amplified by
accidental forwarding duplication. **Ready for proposal: YES.**

---

## 1. Full Current Violation List (raw data)

Run: `golangci-lint run --default none --enable-only gocognit,funlen,nestif
--max-issues-per-linter=0 --max-same-issues=0 ./...` (default thresholds:
gocognit >30, funlen >60 lines OR >40 statements, nestif >5).

### 1a. Non-test violations (31) — baseline for this change

| # | File:Line | Function | Linter | Score |
|---|-----------|----------|--------|-------|
| 1 | internal/actions/backup.go:58 | `BackupAction.Run` | gocognit | **80** |
| 2 | internal/backup/engine.go:62 | `Engine.Run` | gocognit | **79** |
| 3 | internal/tui/model.go:127 | `Model.Update` | gocognit | **84** |
| 4 | internal/actions/restore.go:52 | `RestoreAction.Run` | gocognit | 48 |
| 5 | internal/cloud/pack.go:42 | `tarGzDir` | gocognit | 43 |
| 6 | internal/tui/model.go:361 | `Model.handleKey` | gocognit | 40 |
| 7 | internal/cloud/pack.go:140 | `untarGzDir` | gocognit | 39 |
| 8 | internal/actions/push.go:50 | `PushAction.Run` | gocognit | 32 |
| 9 | internal/tui/model.go:521 | `Model.View` | funlen | 71 lines |
| 10 | internal/actions/diff_backups.go:21 | `DiffBackupsAction.Run` | funlen | 68 lines |
| 11 | cmd/backup.go:55 | `runBackupWithDeps` | funlen | 76 lines |
| 12 | internal/cloud/oauth_device.go:75 | `DeviceClient.RequestToken` | funlen | 49 stmts |
| 13 | cmd/profile.go:201 | `runProfileCreateInteractiveWithDeps` | funlen | 49 stmts |
| 14 | internal/actions/list_local.go:18 | `RunListLocal` | funlen | 45 stmts |
| 15 | internal/actions/cleanup.go:30 | `CleanupAction.Run` | funlen | 47 stmts |
| 16 | internal/actions/pull.go:57 | `PullAction.Run` | funlen | 55 stmts |
| 17 | internal/actions/pick_backup.go:83 | `PickBackupAction.Run` | funlen | 43 stmts |
| 18 | internal/adapters/generic.go:349 | `scanRootFiles` | funlen | 64 lines |
| 19 | internal/actions/export.go:69 | `CreateTarGz` | funlen | 62 lines |
| 20 | internal/tui/screens/wizard.go:235 | `WizardModel.View` | funlen | 47 stmts |
| 21 | internal/tui/screens/profiles.go:79 | `ProfilesModel.Update` | funlen | 42 stmts |
| 22 | internal/tui/screens/restore.go:96 | `RestoreModel.Update` | funlen | 42 stmts |
| 23 | internal/tui/screens/shortcuts.go:19 | `RenderShortcuts` | funlen | 65 lines |
| 24 | internal/tui/model.go:523 | `if m.tooSmall` | nestif | 16 |
| 25 | internal/adapters/yaml.go:84 | `if cp.IsDir` | nestif | 8 |
| 26 | cmd/backup.go:81 | `if backupProfile != ""` | nestif | 8 |
| 27 | cmd/profile.go:211 | `if name != ""` | nestif | 6 |
| 28 | cmd/restore.go:54 | `if len(args) == 0` | nestif | 6 |
| 29 | internal/tui/screens/settings.go:105 | `if m.cursor >= 0...` | nestif | 6 |
| 30 | internal/tui/screens/profiles.go:108 | `if m.Modal != nil` | nestif | 5 |
| 31 | internal/actions/push.go:119 | `if a.HostnameFn != nil` | nestif | 5 |

### 1b. Test-files violations

Tests produce **~40 additional** gocognit/funlen/nestif violations
(table-driven setup tests routinely exceed funlen; test helpers are inherently
branchy). Per AGENTS.md context, the analysis baseline is test-excluded; the
enablement strategy (§7) MUST add a test exclusion rule for these three
linters, otherwise enabling them is unworkable.

### Reconciliation with prior cycle

Prior hand-off quoted "44 test-excluded" (gocognit 24, nestif 15, funlen 5).
Current ground truth is 31 test-excluded. The delta is explained by:
1. ci-hardening-v2 consolidated 3 DRY clones that each contributed funlen
   weight (the cloud Pull dedup, cmd loadExcludes, tui mapBackupInfo).
2. The prior split (goc 24 / nestif 15 / funlen 5) does not match the current
   per-linter ratio (goc 8 / nestif 8 / funlen 15), strongly suggesting the
   prior numbers were collected with different thresholds or test-included.
This change tiers off ACTUAL current data, not the stale 44.

---

## 2. Qualitative Analysis — SEVERE (gocognit >70)

### SEVERE-A: `Model.Update` (tui/model.go:127, gocognit 84, +funlen? no, +nestif 16 via View)

**What makes it complex.** A 230-line type-switch over `tea.Msg` with a
NESTED switch over `m.screen` for `tea.WindowSizeMsg` (lines 129–177), a
SECOND duplicated screen-forward switch at the bottom (lines 297–348) for
"all other messages", a `screenChangeMsg` case doing lazy-init per screen
(lines 198–249), and `ProgressStepMsg`/`ProgressDoneMsg` special handling
(lines 266–294). The replayed pattern is:
```go
case ScreenX:
    if m.x != nil {
        newX, cmd := m.x.Update(msg)
        nx := newX.(screens.XModel)   // type assert
        m.x = &nx                       // reassign
        return m, cmd
    }
```
This 5-line block is copy-pasted **7 times for WindowSizeMsg**, **7 times
for the fallback switch**, **7 times in handleKey** → ~21 copies.

**Inherent vs accidental.** HYBRID. The `tea.Msg` type-switch and the
per-screen dispatch are inherent (Bubble Tea routing genuinely requires
this). The 21 repetitions of the "type-assert + reassign + return" block
are pure ACCIDENTAL complexity — a missing `subModel` abstraction.

**Refactoring approach.** Introduce a small unexported interface in the `tui`
package:
```go
type subModel interface {
    Update(tea.Msg) (tea.Model, tea.Cmd)
}
type forwarder func(tea.Msg) (tea.Model, tea.Cmd)
```
Replace the 7 typed sub-model fields with a `map[Screen]subModel` populated
at lazy-init, plus ONE helper `forwardTo(screen Screen, msg tea.Msg) (tea.Model, tea.Cmd, bool)` that performs the type-assert-reassign generically. `Update`
becomes: type-switch on msg → for the generic-forward cases, call the helper.
Drops Update from ~230 to ~80 lines and removes the funlen on `View`
(collapse the View switch the same way via a `view(screen) (string,bool)`
helper). Removes nestif 16.

- **Difficulty: M**
- **Risk: Medium** — touches core TUI routing. **Mitigated: strong test coverage exists (tui 80.3%, screens 83%)** and AGENTS.md mandates unit-testing Update/View as pure functions; TDD red/green will catch regressions. Outer behavior (cmd dispatch via `Selection()`/`handleMenuEnter`) untouched.

### SEVERE-B: `BackupAction.Run` (actions/backup.go:58, gocognit 80)

**What makes it complex.** 8 sequential numbered phases (resolve categories →
detect adapters → apply excludes → create dir → save fail-fast manifest →
collect+count → backup+secret-scan+build-manifest-items with path-traversal
validation → env.example → save+report). Each phase has its own error
wrapping and the file-walk loop at lines 195–251 nests progress callback,
secret removal, path validation, and manifest construction.

**Inherent vs accidental.** The per-file work (progress, path-traversal
guard, manifest item build) is inherent to the domain. The OVERALL
orchestration is a near-verbatim clone of `Engine.Run` (see SEVERE-C).

**Refactoring approach.** SEE Opportunity #1 (the duplicate-engine
consolidation). `BackupAction.Run` should delegate to the canonical engine
or both should delegate to a shared `runBackupWorkflow(ctx)` helper that
owns the 8 phases; the per-file body becomes a `buildManifestItem(item)
(manifest.Item, error)` method.

- **Difficulty: L** (cross-package: actions↔backup)
- **Risk: Medium-High** — BOTH the CLI path (`cmd/backup.go`) and the TUI
  picker path (`pick_backup.go`) depend on identical semantics. Must
  preserve: secret removal behavior (the two engines differ slightly —
  `BackupAction` removes secret files via `a.FS.RemoveAll`; `Engine` via
  `os.Remove` AND skips the item via a `secretRelPaths` map; this is a real
  behavioral delta to reconcile). Strong test suite on both packages is the
  safety net (verify both `internal/actions/backup_test.go` and
  `internal/backup/engine_test.go` stay green).

### SEVERE-C: `Engine.Run` (backup/engine.go:62, gocognit 79)

Identical structure to SEVERE-B. Same phases, same ordering, same per-file
loop. The differences are mechanical:
- `Engine` uses `os.*` directly; `BackupAction` uses `a.FS` (injected).
- `Engine` hostnames via bare `os.Hostname()`; `BackupAction` via
  `HostnameFn` with nil-fallback.
- Secret handling diverges (see SEVERE-B risk note).
- `Engine` returns `*Result`; `BackupAction` returns `error` and prints a
  report.

This is the **single biggest DRY violation in the codebase** and dupl
cannot detect it (semantic duplication across struct-field vs pkg-call
boundaries). Consolidating removes ONE full SEVERE function outright and
halves the other.

- **Difficulty / Risk**: same as SEVERE-B; they are ONE refactor.

---

## 3. Qualitative Analysis — Remaining (gocognit <70, funlen, nestif)

Grouped by file/package. Categories: **(a)** quick win (<30 min, simple
extraction), **(b)** medium (restructure), **(c)** skip (inherent /
acceptable).

### internal/actions/ (engine-adjacent)

- **restore.go:52 `RestoreAction.Run` (goc 48)** — **(b)** medium. Long
  sequential pipeline (resolve → dry-run diff → confirm → exec → log → git
  safety). Extract phase methods `prepare()`, `showDryRun()`,
  `confirmExec()`, `execRestore()`, `writeLog()`. Pattern: extract-method.
- **diff_backups.go:21 `Run` (funlen 68)** — **(a)** quick win. The body is
  load→compare→group→print. Extract `printDiffGroups(out, groups, order)`
  and `printDiffSummary(out, counts, total)`. Halves the line count.
- **cleanup.go:30 `Run` (funlen 47 stmt)** — **(a)** quick win. Extract
  `printDryRunPlan(out, toDelete, keep)` and `confirmOrProceed(force,
  confirm) error`. The deletion loop already reads cleanly.
- **list_local.go:18 `RunListLocal` (funlen 45 stmt)** — **(a)** quick win.
  Extract `formatBackupRow(m, backupID) string` and dedup the two "No
  backups found" branches (lines 22–30 mirror 45–50) into `noBackupsMsg`.
- **pull.go:57 `Run` (funlen 55 stmt)** — **(b)** medium. DRY with push.go
  (see §4). Extract `loadConfigOr inject()` helper that removes the
  duplicated `if ConfigLoader != nil { ... } else { config.Load() }`
  block (appears in pull.go:71–83, push.go:175–185).
- **push.go:50 `Run` (goc 32, +nestif 5)** — **(b)** medium. Same config
  disp; the hostname-resolution nestif is the same clone as in backup.go
  (see §4 DRY-hostname). Extract `resolveHostname()` once.
- **pick_backup.go:83 `Run` (funlen 43 stmt)** — **(c)** SIMPLICITY WIN, not
  funlen win. `NewRegistry` default registers ONLY opencode whereas
  `cmd/backup.go` registers ALL built-ins via `register.All`. either a bug
  or intentional simplification — flag for clarification (see §5). Quick
  fix: delegate to `register.All` for consistency (Tier 3).

### internal/cloud/

- **pack.go:42 `tarGzDir` (goc 43) & `untarGzDir` (goc 39)** — **(c)** SKIP.
  Both are `filepath.WalkDir` closures with the inherent header-writing /
  security-header / Symlink handling the domain demands. The complexity is
  INHERENT. Document with a `//nolint:gocognit` guard noting the
  tar/gzip walk is a fixed algorithm. DO NOT restructure — risk outweighs
  value.
- **oauth_device.go:75 `RequestToken` (funlen 49 stmt)** — **(b)** medium.
  Implements RFC 8628 Device Flow (3 phases: request code → display+open
  browser → poll). Extract `requestDeviceCode()`, `pollForToken()` phases.
  The phases are real domain steps (inherent), but splitting them improves
  readability and matches the godoc comment structure.

### internal/adapters/

- **generic.go:349 `scanRootFiles` (funlen 64)** — **(c)→(a)** borderline.
  The per-entry work is: whitelist-resolve category → exclude check →
  stat → size guard → hash → append. Extract `buildRootItem(entry, cat)
  (Item, bool, error)` (bool = skip). Mild win; low risk; heavily tested
  (generic_test.go large). Tier 3 quick win IF time allows.
- **yaml.go:84 `if cp.IsDir` (nestif 8)** — **(a)** quick win. The
  `ListItems` IsDir-branch already delegates dir scans to `scanCategoryDir`
  but inline-handles RootFiles. Extract `scanRootFilesYAML(cp, configDir,
  cat) ([]Item, error)` to mirror the generic adapter. Minor.

### internal/tui/

- **model.go:361 `handleKey` (goc 40)** — folds INTO SEVERE-A refactor. The
  same `forward-to-screens` map fixes this too.
- **model.go:523 `View` nestif 16 & funlen 71** — **(a)** quick win,
  independent of SEVERE-A. Extract `renderScreen(screen) string` from the
  big switch, and `styles.RenderTooSmall(w,h) string` (this guard is ALSO
  duplicated in `screens/settings.go:128` View — §4 DRY). View shrinks to:
  guard → render → overlay help → overlay toast.
- **screens/wizard.go:235 `View` (funlen 47 stmt)** — **(a)** quick win.
  Extract `renderStep()` switch arm helpers. The wizard is a step-state
  machine; each step's render is ~8 lines.
- **screens/profiles.go:79 & restore.go:96 `Update` (funlen 42 stmt each)**
  — **(c)** SKIP individually; revisit AFTER SEVERE-A lands a shared
  base-screen helper. They share the `WindowSizeMsg` + `LoadMsg` +
  `ModalResultMsg` + `KeyPressMsg` skeleton; a `baseScreenModel` would cut
  both, but that's a larger migration than this change should bundle.
  Tier 4 — document.
- **screens/shortcuts.go:19 `RenderShortcuts` (funlen 65)** — **(c)** SKIP.
  Static key-bindings table; long but trivially linear. Acceptable.
- **screens/settings.go:105 nestif 6** — **(a)** quick win. The toggle
  cursor-guard `if m.cursor>=0 && m.cursor<len(...)` already has a guard
  above; collapse to early-continue or a `toggleCurrent() error` method.

### cmd/

- **cmd/backup.go:55 `runBackupWithDeps` (funlen 76, +nestif 8 on
  backupProfile)** — **(b)** medium. The `backupProfile != ""` block
  (resolve preset/categories/adapter from profile + verbose log) is a
  self-contained concern → extract `applyProfileOverrides(deps, cfg)
  (preset, cats, adapters, error)`. Also the registry-build snapshot at
  lines 60–71 is a candidate for the registry helper (§4). This is the
  cmd-layer mirror of the Engine/BackupAction split —- refactors here reduce
  AFTER the engine consolidation.
- **cmd/profile.go:201 `runProfileCreateInteractiveWithDeps` (funlen 49
  stmt, +nestif 6)** — **(a)** quick win (HIGH value). The wizard-launch
  + model-extraction + adapter/category collection is copy-pasted TWICE
  (the `name != ""` branch lines 211–253 and the `name == ""` branch
  256–296). Extract `launchWizard(cfg, name, providers) (actions.
  ProfileCreateFromWizard, *screens.WizardModel, error)`. Removes BOTH the
  funlen AND the nestif, ~45 deleted lines.
- **cmd/restore.go:54 nestif 6** — **(c)** SKIP. The block is an
  interactive-picker path with a real early-return ladder; flattening adds
  nesting elsewhere. Acceptable.

---

## 4. DRY Analysis Beyond dupl

`dupl` only catches token-level clones. The semantic duplication dupl
misses:

### #1 (CRITICAL) Two parallel backup engines — SEE §2 SEVERE-B/C.
~150 duplicated lines. dupl score: invisible (struct-field vs pkg-call).

### #2 resolveBackupID / "latest backup" pattern — THREE+ forms
- `internal/actions/push.go:201 resolveBackupID` (verbose-aware, returns
  latest sorted desc).
- `internal/actions/pick_backup.go:33 ResolveBackupID` (backupsDir+args,
  returns latest sorted asc).
- `internal/backup/resolve.go:14 ResolveBackupID` (different signature:
  returns backupDir, single id arg → no "find latest").
- `internal/actions/cleanup.go:60` repeats the `sort`-descending + take-N
  id-collection.

Consolidate into ONE `backup.LatestBackupID(backupsDir) (string, error)` +
`backup.ListBackupIDs(backupsDir) ([]string, error)` in `internal/backup`.
Difficulty S, risk Low, tests exist on all three call sites.

### #3 hostname-resolution boilerplate — 3 sites
`backup.go:137–147`, `push.go:119–131` (identical), `engine.go:127–133`
(simpler variant). Pattern:
```go
hostname := "unknown"
if a.HostnameFn != nil {
    if h, err := a.HostnameFn(); err == nil { hostname = h }
    else if a.Verbose { warnf(errOut, "warning: hostname: %v\n", err) }
} else if h, err := os.Hostname(); err == nil { hostname = h }
else if a.Verbose { warnf(..., err) }
```
Extract `resolveHostname(fn HostnameFunc, verbose bool, errOut io.Writer)
string` in `internal/actions` (reuse the existing `HostnameFunc` in
interfaces.go:58). Difficulty S, risk Low.

### #4 (GGA finding, OPEN) cloud `List()` dedup
`GiteaProvider.List` (gitea.go:143–191) and `GitHubRepoProvider.List`
(github_repo.go:100–147) are ~50-line near-identical clones. Differences
are parameterizable: URL template, accept header, error prefix, URL
builder for BackupMeta.URL. Prior consolidation already extracted
`getFileSHA`, `writeContentFile`, `pullContentFromAPI` — but `List` was
left duplicated (the prior-cycle finding was deferred as pre-existing).
Extract a `contentsListProvider` base or a `listContentsDir(client, url,
token, acceptHeader, errPrefix, urlBuilder) ([]BackupMeta, error)` helper.
Difficulty M, risk Medium (cloud tests are integration-style; review
gitea_test/github_repo_test carefully).

### #5 stdout/stderr + IO nil-fallback pattern — pervasive
Every action opens with:
```go
out := a.Stdout; if out == nil { out = os.Stdout }
errOut := a.Stderr; if errOut == nil { errOut = os.Stderr }
```
Appears in backup.go:59, push.go:51, pull.go (has stdout()/stderr() methods
already — the BETTER pattern). Encourage the `stdout() io.Writer` method
pattern (already used in pull.go) across all actions. Difficulty S, risk
Low, but only worth doing opportunistically — Tier 4 (style consistency).

### #6 ConfigLoader nil-fallback — 2 sites identical
pull.go:71–83 and push.go:178–185 byte-identical:
```go
if a.ConfigLoader != nil { cfg, err = a.ConfigLoader() }
else { cfg, err = config.Load() }
```
Extract `a.loadConfig() (*config.Config, error)` on a shared base or per
action. Difficulty S, risk Low. Pair with #3.

### #7 TUI "terminal too small" guard — duplicated
model.go:523 View guard `fmt.Sprintf("Terminal too small (%dx%d)...")` AND
screens/settings.go:128 View guard — identical. Extract
`styles.RenderTooSmall(width, height) string` in `internal/tui/styles`
(already owns MinWidth/MinHeight). Difficulty S, risk Low.

### #8 TUI cursor j/k navigation — 4 screens
`case 'j', tea.KeyDown: m.cursor = (m.cursor+1)%len(...)` appears in
settings.go, restore.go, dashboard.go, profiles.go. Plus the screens
re-implement `tea.WindowSizeMsg: w,h = ...; return`. Extract a
`components.CursorNav(len, idx, msg) int` helper and a `baseScreenModel
.AutoSize(msg)`. Difficulty M, risk Medium — defer beyond this change
(behavioural; each screen's base type differs). Tier 3.

### NOT revisited (done in ci-hardening-v2)
cloud Pull dedup, cmd loadExcludes, tui mapBackupInfo — consolidated.
Thumbs-up; do not re-analyze.

---

## 5. Spaghetti / Over-engineering Analysis

**nestif verdicts:**
- `model.go:523 tooSmall=16` — ACCIDENTAL (see SEVERE-A). Real spaghetti
  in `View` (guard wraps a 50-line switch). Tier 1.
- `yaml.go:84 IsDir=8` — mild; the RootFiles sub-branch is legitimate but
  extractable. Tier 3 quick win.
- `cmd/backup.go:81`, `cmd/profile.go:211`, `cmd/restore.go:54` — real
  guard ladders, acceptable (restore) or extractable (backup/profile).
- `screens/settings.go:105`, `profiles.go:108` — minor; settings toggle is
  the only one worth touching.
- `push.go:119 HostnameFn=5` — clone of backup.go; folds into #3 DRY.

**God functions:** SEVERE-A/B/C are the only true god-functions. After the
two consolidations, the largest remaining gocognit is 48 (restore), 43
(pack), 40 (handleKey). All acceptable or skippable.

**Unnecessary indirection:** NONE found. The layers are clean: cmd →
actions → adapters/backup/cloud. No pass-through wrappers detected.

**Over-engineering:**
- **`tui.Screen` is exported but used ZERO times outside the `tui`
  package** (grep confirms). This is the GGA finding "exported Screen
  (unnecessary export)". Fix: `type screen int` (unexport). Risk: Low but
  needs care — `screens.ScreenBackMsg` and external references to the
  constants must be checked; constants stay exported if referenced. Tier 1
  quick win. **CAUTION:** verify no cmd/ reference first (this change MUST
  not break the cmd→tui `Selection()` contract).
- `adapters/util.go` `FileHash`/`CopyFile` are appropriately shared — not
  over-engineered.
- `internal/cloud` provider API (Push/Pull/List + ProviderRegistry) is
  right-sized; the duplication is the List body, not the abstraction.
- `PickBackupAction.NewRegistry` registering ONLY opencode vs `register.All`
  elsewhere — POSSIBLE over-simplification rather than over-engineering;
  flag for clarification (is the picker intentionally opencode-only, or a
  leftover from before full-adapter support?).
- `oauth_device.RequestToken` is a faithful RFC 8628 implementation — not
  over-engineered; just long. The phase-split (§3) is readability, not
  de-engineering.

---

## 6. Prioritized Refactoring Opportunities (tiered)

Tiered by Value × Risk × Effort × Testability.

### Tier 1 — HIGH value, LOW risk, tests exist → DO FIRST
1. **#1 Consolidate the two backup engines** (`Engine.Run` ↔ `BackupAction.Run`).
   Fixes 2 of 3 SEVERE. ~150 duplicated lines removed. Value: 10. Risk:
   Med-High (mitigated by dual test suites + TDD). Effort: L. Tests: strong.
   → Bundle as the **centerpiece**; do under strict TDD with one engine
   delegating. The behavioral delta (secret-removal strategy) MUST be
   reconciled explicitly (see §2 risk).
2. **#4 cloud `List()` dedup** (GGA finding, open). Value 7, Risk Med,
   Effort M. Tests: gitea/github_repo integration tests + httputil.
3. **Unexport `tui.Screen`** (GGA finding). Value 4, Risk Low, Effort S.
4. **#7 `styles.RenderTooSmall`** + extract `Model.renderScreen()` → fixes
   `View` nestif 16 + funlen 71. Value 7, Risk Low (View is pure, tested),
   Effort S-M.

### Tier 2 — HIGH value, MEDIUM risk → DO with care
5. **SEVERE-A `Model.Update` sub-model map**. Fixes goc 84 + goc 40
   (handleKey). Value 10, Risk Med (core routing), Effort M, Tests strong.
6. **#2 `resolveBackupID` consolidation** (1 canonical + list helper).
   Value 6, Risk Low-Med (3 call sites), Effort S, Tests on all sites.
7. **#3 + #6 hostname + loadConfig helpers**. Value 5, Risk Low, Effort S.

### Tier 3 — MEDIUM value, LOW effort → quick wins (opportunistic)
8. **#8 cmd/profile.go wizard-launch dedup** (removes funlen 49 + nestif 6).
   Value 7, Risk Low (logic identical in both branches), Effort S.
9. **diff_backups / cleanup / list_local / settings toggle extract-method**
   (funlen quick wins). Value 5 each, Risk Low, Effort S each, Coverage good.
10. **cmd/backup.go `applyProfileOverrides` extract** (funlen 76 + nestif 8).
    Value 6, Risk Low, Effort S. (Do AFTER Tier 1 #1 to avoid churn.)
11. **generic `scanRootFiles` + yaml `scanRootFilesYAML` split**. Value 4.

### Tier 4 — LOW value OR HIGH risk → SKIP (document why)
- `cloud/pack.go tarGzDir/untarGzDir` (goc 43/39): INHERENT domain complexity.
- `screens/shortcuts.RenderShortcuts` (funlen 65): linear static table.
- `screens/profiles/RestoreModel.Update` deep rework: defer to a future
  `baseScreenModel` migration; this change shouldn't bundle a cross-screen
  base-type refactor.
- `cmd/restore.go:54 nestif 6`: acceptable guard ladder.
- `internal/cloud/gist.go` CRUD scaffold dedup: medium value, defer.
- `PickBackupAction NewRegistry` opencode-only: clarify INTENT before
  "fixing" — may be deliberate.

---

## 7. .golangci.yml Enable-Linter Strategy

### Thresholds (after Tier 1–2 refactors land)

```yaml
linters:
  enable:
    - gocognit
    - funlen
    - nestif
  settings:
    gocognit:
      min-complexity: 35        # default 30 too aggressive; post-refactor ceiling ~48
    funlen:
      lines: 80                 # default 60; allow legitimate linear functions
      statements: 50            # default 40; allow table-ish pipelines
    nestif:
      min-complexity: 6         # default 5; tolerate shallow guards
```

Rationale: enabling AT default thresholds (30/60/5) would flag the
remaining inherent-complexity functions (pack.go, shortcuts, the
post-refactor RestoreModel) and force `//nolint` spam. Choose thresholds
that PASS cleanly on the refactored tree and tighten over time (ratchet).

### Test exclusion (MANDATORY — otherwise unworkable)

```yaml
linters:
  exclusions:
    rules:
      - path: '(.+_test\.go)'
        linters: [gocognit, funlen, nestif]   # ADD to existing rule
```
Tests legitimately have long table-driven setup and branchy helpers.
Without this, enabling produces ~40 noise violations and no-one will
respect the gate.

### Ratchet approach (prevent regression)

1. Land refactors. Run `gocognit/funlen/nestif` over non-test code →
   produces ZERO violations at the chosen thresholds.
2. Commit the `.golangci.yml` change in the SAME PR that finishes the
   refactors (no window where the linters are enabled but violations
   exist).
3. Add a per-linter `//nolint:gocognit // <reason>` ONLY for the
   explicitly-deferred Tier-4 items (`tarGzDir`, `untarGzDir`,
   `RenderShortcuts`) — each with a typed reason. Audit existing
   `//nolint:maintidx` comments: the 3 SEVERE ones may be REMOVED once the
   underlying functions are refactored (they'd otherwise be dead nolints).

### nolint policy (after refactor)

- Remove `//nolint:maintidx` from the THREE SEVERE functions once
  refactored (they will no longer exist or will be small).
- Any NEW `//nolint` for gocognit/funlen/nestit MUST cite a Tier-4 reason
  matching this explore.md (inherent complexity). CI guard: a grep-based
  check that nolint count never rises above N.

### Linter-test interaction

After enabling, run `golangci-lint run ./...` in CI exactly as
ci-hardening-v2 wired `golangci-lint-action@v8`. The test-exclusion rule
keeps the run green; the thresholds prevent churn.

---

## Risks

- **Medium-High:** Backup-engine consolidation touches the two highest-
  traffic code paths (CLI + TUI picker) with a real behavioral delta in
  secret-removal strategy (`RemoveAll` vs `Remove`+`secretRelPaths` skip-
  map). MUST reconcile deliberately, NOT silently pick one. TDD red/green
  on both `internal/actions/backup_test.go` and `internal/backup/
  engine_test.go` is mandatory and is the safety net.
- **Medium:** `Model.Update` sub-model refactor is core routing. Strong
  Update/View unit tests mitigate; risk is logic in custom `Msg` cases
  (ProgressStep/Done) that don't go through the forwarding map — must keep
  those branches explicit.
- **Medium:** cloud `List()` dedup — the two providers differ in URL
  builder + accept header + error prefix; the helper signature must carry
  these as parameters without obscuring intent. A bad abstraction here is
  worse than the duplication.
- **Low:** `tui.Screen` unexport — must first verify no external
  (cmd/) references to the exported constants; if any, keep constants
  exported and only unexport the type.
- **Process:** the prior cycle claimed 44 violations; reality is 31. The
  proposal/spec/tasks MUST plan against 31 (and the 7 opportunities), not
  44 — otherwise scope over-runs.

## Skill Resolution
`paths-injected` — loaded `sdd-explore` + `golang-pro` SKILL.md via Read
(both paths existed, no registry fallback).

## Result Contract (for orchestrator)

```
status: success
executive_summary: Ran gocognit/funlen/nestif live; current non-test violations = 31 (not the 44 quoted — prior cycle pre-dated 3 DRY consolidations and/or counted tests). Reading every offender surfaced 7 real refactoring opportunities: the CRITICAL one is two parallel backup engines (Engine.Run=79 ≈ BackupAction.Run=80, ~150 semantic-clone lines dupl can't see), plus Model.Update god-function (84), cloud List() dedup (open GGA finding), unexported-screen, and 4 helper consolidations (resolveBackupID×3, hostname×3, loadConfig×2, tooSmall×2). Tiering: 4 Tier-1 (high-value), 3 Tier-2 (care), 4 Tier-3 (quick wins), 6 Tier-4 skip (inherent). Enable-strategy: thresholds gocognit 35 / funlen 80-50 / nestif 6 + MANDATORY test exclusion, committed alongside refactors, ratchet forward.
artifacts:
  - openspec/changes/qa-refactor-analysis/explore.md
  - engram: sdd/qa-refactor-analysis/explore
next_recommended: propose
risks:
  - Medium-High backup-engine consolidation has a real secret-removal behavioral delta between the two clones; reconcile deliberately under TDD (both test suites green)
  - Medium Model.Update core-routing refactor; mitigated by 80%+ Update/View coverage
  - Medium cloud List() abstraction must parameterize URL/header/prefix without obscuring intent
  - Low tui.Screen unexport must verify no cmd/ constant references first
  - Process prior "44" overstates scope; plan against actual 31 / 7 opportunities
skill_resolution: paths-injected
```