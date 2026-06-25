# Proposal: qa-refactor-analysis

## Intent

Eliminate 31 non-test gocognit/funlen/nestif violations and the CRITICAL duplicated backup engine. Two parallel ~230-line orchestrators (`BackupAction.Run` gocognit 80, `Engine.Run` gocognit 79) are semantic clones with a behavioral delta in secret-removal. Consolidating removes ~150 duplicated lines and two of three SEVERE functions. Enable gocognit/funlen/nestif linters at ratcheted thresholds to prevent regression.

## Scope

### In Scope
- **Tier 1 (4 items):** Engine consolidation, cloud `List()` dedup, unexport `tui.Screen`, `styles.RenderTooSmall` helper
- **Tier 2 (3 items):** `Model.Update` submodel map, `resolveBackupID` canonical, hostname+loadConfig helpers
- **Tier 3 (4 items):** cmd/profile wizard-launch dedup, diff/cleanup/list extract-methods, cmd/backup `applyProfileOverrides`, generic/yaml scanRootFiles split
- **Linter enable:** gocognit 35, funlen 80/50, nestif 6 + test exclusion; remove 3 dead `//nolint:maintidx`

### Out of Scope
- Tier 4 items (6): pack.go tarGz/untar (inherent), RenderShortcuts (static), screen base-model migration (cosmetic), cmd/restore guard ladder (necessary), gist CRUD scaffold (premature), pick_backup opencode-only registry (needs intent clarification)
- TestRunLogin_EmptyToken flaky fix (separate follow-up)
- OpenCode CLI install in CI (separate follow-up)
- Any new features — REFACTORING ONLY, preserving behavior

## Capabilities

### New Capabilities
None — this is pure refactoring, no new behavior.

### Modified Capabilities
- `backup-engine`: Consolidate `BackupAction.Run` + `Engine.Run` into single implementation; reconcile secret-removal delta (see Decision Required)
- `tui-dispatch`: Replace type-assert+reassign forwarding with submodel interface + map dispatch in `Model.Update`/`handleKey`/`View`
- `internal-adapters`: Extract `scanRootFiles` helper in generic adapter; mirror in yaml adapter
- `cloud-providers`: Deduplicate `List()` across GiteaProvider and GitHubRepoProvider
- `lint-config`: Enable gocognit/funlen/nestif in `.golangci.yml` with test exclusion

## Approach

**Tier 1 — Engine consolidation (centerpiece):**
- Extract shared `runBackupWorkflow(ctx BackupContext) (*Result, error)` in `internal/backup` package
- `BackupContext` carries: FS (injected), HomeDir, BakDir, Registry, Preset, AdapterFilter, BakVersion, Verbose, SecretPatterns, CustomCategories, ProgressFn, ExcludesLoader, HostnameFn, Stdout, Stderr
- Reconcile secret-removal delta: adopt Engine's `secretRelPaths` skip-map (correct), use `RemoveAll` for safety (BackupAction's approach)
- `BackupAction.Run` delegates to `runBackupWorkflow`, prints report, returns error
- `Engine.Run` delegates to `runBackupWorkflow`, returns `*Result`
- TDD: both `internal/actions/backup_test.go` and `internal/backup/engine_test.go` must stay green

**Tier 1 — Cloud `List()` dedup:**
- Extract `listContentsDir(client, url, token, acceptHeader, errPrefix, urlBuilder) ([]BackupMeta, error)` in `internal/cloud/httputil.go`
- Parameterize: URL template, accept header, error prefix, BackupMeta.URL builder
- GiteaProvider.List and GitHubRepoProvider.List become thin wrappers

**Tier 1 — Unexport `tui.Screen`:**
- Rename `type Screen int` → `type screen int` in `internal/tui`
- Constants stay exported if referenced externally (verify first)
- No cmd/ references found (grep confirmed)

**Tier 1 — `styles.RenderTooSmall`:**
- Extract `RenderTooSmall(width, height int) string` in `internal/tui/styles`
- Replace duplicated guards in `model.go:523`, `screens/dashboard.go:150`, `screens/health.go:127`
- `Model.View` extracts `renderScreen(screen) string` from big switch → fixes nestif 16 + funlen 71

**Tier 2 — `Model.Update` submodel map:**
- Introduce `type subModel interface { Update(tea.Msg) (tea.Model, tea.Cmd) }` in `internal/tui`
- Replace 7 typed sub-model fields with `map[Screen]subModel` populated at lazy-init
- Extract `forwardTo(screen Screen, msg tea.Msg) (tea.Model, tea.Cmd, bool)` helper
- `Update` becomes: type-switch on msg → for generic-forward cases, call helper
- Drops Update from ~230 to ~80 lines; fixes gocognit 84 + gocognit 40 (handleKey)

**Tier 2 — `resolveBackupID` canonical:**
- Consolidate 3 forms into `backup.LatestBackupID(backupsDir) (string, error)` + `backup.ListBackupIDs(backupsDir) ([]string, error)`
- Call sites: `actions/pick_backup.go`, `actions/push.go`, `actions/cleanup.go`, `actions/diff_backups.go`
- `backup.ResolveBackupID(id)` (different signature, returns backupDir) stays separate

**Tier 2 — hostname + loadConfig helpers:**
- Extract `resolveHostname(fn HostnameFunc, verbose bool, errOut io.Writer) string` in `internal/actions`
- Extract `loadConfigOr(loader ConfigLoader) (*config.Config, error)` method on actions
- Replace 3 hostname sites (backup.go, push.go, engine.go) and 2 loadConfig sites (pull.go, push.go)

**Tier 3 — cmd/profile wizard-launch dedup:**
- Extract `launchWizard(cfg, name, providers) (actions.ProfileCreateFromWizard, *screens.WizardModel, error)`
- Removes funlen 49 + nestif 6, ~45 deleted lines

**Tier 3 — diff/cleanup/list extract-methods:**
- `diff_backups.go`: extract `printDiffGroups(out, groups, order)` + `printDiffSummary(out, counts, total)`
- `cleanup.go`: extract `printDryRunPlan(out, toDelete, keep)` + `confirmOrProceed(force, confirm) error`
- `list_local.go`: extract `formatBackupRow(m, backupID) string` + dedup "No backups found" branches
- `settings.go:105`: collapse cursor-guard to early-continue or `toggleCurrent() error`

**Tier 3 — cmd/backup `applyProfileOverrides`:**
- Extract `applyProfileOverrides(deps, cfg) (preset, cats, adapters, error)` from `runBackupWithDeps`
- Do AFTER Tier 1 engine consolidation to avoid churn

**Tier 3 — generic/yaml scanRootFiles split:**
- Extract `buildRootItem(entry, cat) (Item, bool, error)` from `scanRootFiles`
- Mirror in yaml adapter: `scanRootFilesYAML(cp, configDir, cat) ([]Item, error)`

**Linter enable:**
- Add to `.golangci.yml`: `gocognit` (min-complexity 35), `funlen` (lines 80, statements 50), `nestif` (min-complexity 6)
- Add test exclusion rule: `path: '(.+_test\.go)'`, linters: `[gocognit, funlen, nestif]`
- Remove 3 `//nolint:maintidx` from SEVERE functions (backup.go:58, engine.go:62, model.go:127)
- Add `//nolint:gocognit // inherent: tar/gzip walk is fixed algorithm` to `tarGzDir`, `untarGzDir`
- Commit linter config in SAME PR as refactors (no window where linters enabled but violations exist)

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/backup/engine.go` | Modified | Extract `runBackupWorkflow`; `Engine.Run` becomes thin wrapper |
| `internal/actions/backup.go` | Modified | `BackupAction.Run` delegates to `runBackupWorkflow`; remove `//nolint:maintidx` |
| `internal/tui/model.go` | Modified | Submodel map in `Update`/`handleKey`/`View`; remove `//nolint:maintidx` |
| `internal/tui/styles/` | Modified | Add `RenderTooSmall` helper |
| `internal/tui/screens/dashboard.go` | Modified | Use `styles.RenderTooSmall` |
| `internal/tui/screens/health.go` | Modified | Use `styles.RenderTooSmall` |
| `internal/cloud/gitea.go` | Modified | `List()` delegates to `listContentsDir` |
| `internal/cloud/github_repo.go` | Modified | `List()` delegates to `listContentsDir` |
| `internal/cloud/httputil.go` | Modified | Add `listContentsDir` helper |
| `internal/actions/pick_backup.go` | Modified | Use `backup.LatestBackupID` |
| `internal/actions/push.go` | Modified | Use `resolveHostname` + `loadConfigOr` |
| `internal/actions/pull.go` | Modified | Use `loadConfigOr` |
| `internal/actions/cleanup.go` | Modified | Use `backup.LatestBackupID`; extract `printDryRunPlan` |
| `internal/actions/diff_backups.go` | Modified | Extract `printDiffGroups` + `printDiffSummary` |
| `internal/actions/list_local.go` | Modified | Extract `formatBackupRow` |
| `internal/adapters/generic.go` | Modified | Extract `buildRootItem` |
| `internal/adapters/yaml.go` | Modified | Extract `scanRootFilesYAML` |
| `cmd/backup.go` | Modified | Extract `applyProfileOverrides` |
| `cmd/profile.go` | Modified | Extract `launchWizard` |
| `internal/tui/types.go` (or similar) | Modified | Unexport `Screen` → `screen` |
| `.golangci.yml` | Modified | Enable gocognit/funlen/nestif + test exclusion |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Engine consolidation breaks CLI or TUI path | Medium | TDD red/green on both test suites; reconcile secret-removal delta explicitly (see Decision Required) |
| Model.Update submodel map breaks TUI routing | Medium | Strong Update/View unit tests (80%+ coverage); keep ProgressStep/Done branches explicit |
| Cloud List() abstraction obscures intent | Low | Parameterize URL/header/prefix; review gitea/github_repo tests carefully |
| tui.Screen unexport breaks external refs | Low | Grep confirmed no cmd/ references; constants stay exported if needed |
| Linter enable produces false positives | Low | Ratcheted thresholds (35/80-50/6) + test exclusion; commit alongside refactors |

## Rollback Plan

Revert the PR. All changes are additive (new helpers) or refactoring (no behavior change except secret-removal delta fix). Git revert restores prior state. No data migration, no schema change.

## Dependencies

- None external. All refactoring uses existing dependencies.
- Strict TDD ACTIVE: all refactors must pass existing test suites before commit.

## Success Criteria

- [ ] `golangci-lint run ./...` produces ZERO gocognit/funlen/nestif violations on non-test code at thresholds 35/80-50/6
- [ ] `go test ./...` passes with >80% coverage on all `internal/` packages
- [ ] `BackupAction.Run` and `Engine.Run` consolidated into single implementation; secret-removal delta reconciled
- [ ] 3 `//nolint:maintidx` annotations removed from SEVERE functions
- [ ] No new `//nolint` annotations except Tier 4 inherent-complexity items (tarGzDir, untarGzDir, RenderShortcuts)
- [ ] CLI backup path and TUI picker path produce identical manifests (verify via integration test or manual comparison)

## Decision Required: Secret-Removal Delta

**The two engines differ in secret-removal behavior. This is a BUG in `BackupAction.Run`.**

**BackupAction.Run (CLI path, backup.go:202-212):**
- Uses `a.FS.RemoveAll(sf)` — recursive, handles directories
- Does NOT skip secret files from the manifest
- **BUG:** manifest lists files that were removed from disk → dangling references

**Engine.Run (TUI path, engine.go:190-203):**
- Uses `os.Remove(secretFile)` — single file only
- DOES skip secret files from manifest via `secretRelPaths` map
- **CORRECT:** manifest only lists files that exist

**Recommendation:** Adopt Engine's behavior (skip from manifest) as canonical. Use `RemoveAll` for safety (BackupAction's approach, handles edge cases). The manifest should never reference files that don't exist.

**Impact:** CLI path currently produces manifests with dangling references. After consolidation, both paths produce clean manifests.

**Action:** Surface to user for confirmation before apply. This is a behavior change (bug fix) in the CLI path.

## Tier 4 Skip Rationale

| Item | Reason |
|------|--------|
| `pack.go` tarGzDir/untarGzDir (goc 43/39) | INHERENT domain complexity: tar/gzip walk is a fixed algorithm with security-header/symlink handling. Restructuring adds risk without value. Add `//nolint:gocognit` with reason. |
| `screens/shortcuts.RenderShortcuts` (funlen 65) | Static key-bindings table. Long but trivially linear. Acceptable. |
| `screens/profiles/RestoreModel.Update` deep rework | Defer to future `baseScreenModel` migration. This change shouldn't bundle a cross-screen base-type refactor. |
| `cmd/restore.go:54 nestif 6` | Acceptable guard ladder. Interactive-picker path with real early-return. Flattening adds nesting elsewhere. |
| `internal/cloud/gist.go` CRUD scaffold | Medium value, defer. Gist CRUD is not yet used in production paths. |
| `PickBackupAction NewRegistry` opencode-only | Clarify INTENT before "fixing". May be deliberate (picker is opencode-only) or leftover from before full-adapter support. Flag for product decision. |

## Estimated Changed Lines

| Tier | Items | Estimated Lines Changed |
|------|-------|------------------------|
| Tier 1 | 4 | ~400 (engine consolidation ~200, cloud List ~80, tui.Screen ~20, RenderTooSmall ~100) |
| Tier 2 | 3 | ~250 (Model.Update ~150, resolveBackupID ~50, hostname+loadConfig ~50) |
| Tier 3 | 4 | ~200 (profile wizard ~80, diff/cleanup/list ~80, cmd/backup ~30, generic/yaml ~10) |
| Linter | 1 | ~20 (.golangci.yml + nolint cleanup) |
| **Total** | **12** | **~870 lines** |

Net reduction: ~150 lines (engine consolidation) + ~50 lines (cloud List) + ~45 lines (profile wizard) = **~245 lines deleted**.
