# Design: qa-refactor-analysis

## Technical Approach

Pure refactoring (behavior-preserving) except one confirmed bug fix: the CLI
backup path's manifest no longer references secret files removed from disk.
Strategy: extract a single canonical backup workflow in the leaf package
`internal/backup`, then make `BackupAction.Run` (CLI) and `Engine.Run` (TUI
picker via `pick_backup.go:144`) thin delegators. Apply the same extract-helper
pattern to cloud `List()`, TUI message dispatch, `resolveBackupID`, and the
Tier-3 long functions. Enable `gocognit`/`funlen`/`nestif` in the SAME PR that
lands the refactors, at ratcheted thresholds, with mandatory test exclusion.

## Architecture Decisions

### Decision: Where the canonical backup workflow lives

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `backup.Run(ctx)` in `internal/backup` (leaf) | `actions` already imports `backup`; backup cannot import actions — leaf is the only place both callers can share without a cycle | ✅ Chosen |
| `runBackupWorkflow` in `internal/actions` | `Engine` (in `backup`) couldn't import it | Rejected (import cycle) |
| Inject `actions.FileSystem` into `Engine` | `backup` can't import `actions.FileSystem` | Rejected |

**Rationale:** `internal/actions` already imports `internal/backup`
(`backup.go:15`). Import direction rules out any shared code in `actions`. The
workflow MUST live in `internal/backup`.

### Decision: FS abstraction for the shared workflow

`Engine` uses `os.*` directly; `BackupAction` uses injected `a.FS`
(actions.FileSystem). The shared workflow needs one FS contract.

**Choice:** Define `backup.FS` (subset of `actions.FileSystem`) in
`internal/backup`; provide `osFS` for the OS path; `BackupAction.FS`
structurally satisfies `backup.FS` and is passed straight through.

**Rationale:** Identical method names/signatures → Go structural typing means
`actions.OSFileSystem` and `MockFileSystem` satisfy `backup.FS` without
adapters. `Engine.Run` sets `ctx.FS = osFS{}` (nil-safe default preserves all
existing `&backup.Engine{...}` instantiations in tests/picker).

### Decision: Secret-removal reconciliation (bug fix)

**Choice:** Adopt `Engine.Run`'s `secretRelPaths` skip-map (manifest excludes
secrets) **and** `BackupAction`'s `FS.RemoveAll` (handles directories).
**Rationale:** Fixes the CLI dangling-reference bug; `RemoveAll` is the safer
removal. This is the ONLY behavior change; spec scenario "manifest Items count
excludes secrets" codifies it.

### Decision: resolveHostname location (spec correction)

**Choice:** `ResolveHostname(fn func()(string,error), verbose bool, errOut io.Writer) string`
lives in `internal/backup`, NOT `internal/actions`.
**Rationale:** The workflow (in `backup`) owns hostname logic; only `push.go`
remains an out-of-workflow caller. Spec named `internal/actions/`, but that
package cannot be imported by `backup`. `loadConfigOr` stays in
`internal/actions` (pull/push only, no cycle). **This is a spec-location delta
to flag to the user.**

### Decision: Model.Update dispatch structure

**Choice:** `type subModel interface { Update(tea.Msg)(tea.Model,tea.Cmd) }` +
`m.subs map[screen]subModel` (lazily populated, holding the SAME pointers the
7 typed fields hold today) + `forwardTo` helper. Update/handleKey/View migrate
to pointer receivers so the map's `set` closures persist typed fields.
**Rationale:** Removes ~21 type-assert-reassign blocks. Pointer receivers keep
map-mutation stable; existing Update/View tests (white-box, package-internal)
stay valid since they access fields directly.

## Data Flow

    cmd/backup.go ──▶ BackupAction.Run ─┐
                                       ├─▶ backup.Run(Context) ─▶ Result
    pick_backup.go ──▶ Engine.Run ──────┘       (uses Context.FS)
        (osFS)                                      │
                                                   ▼
                              adapters → manifest (via FS.WriteFile)
                              secret scan → secretRelPaths → FS.RemoveAll

    cloud:   GiteaProvider.List ─┐▶ listContentsDir(client,url,token,accept,errPrefix,urlBuilder)
              GitHubRepoProvider.List ─┘

    tui:     msg ─▶ Model.Update ─▶ m.subs[screen].Update ─▶ set(closure) ─▶ typed field

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/backup/workflow.go` | Create | `type FS interface{...}` (subset), `osFS`, `type Context`, `func Run(Context)(*Result,error)` — the 8 phases |
| `internal/backup/engine.go` | Modify | `Engine.Run` builds `Context{FS:e.FS or osFS{}, HostnameFn:nil,...}` → `Run(ctx)`. Drop `//nolint:maintidx`. Keep `scanBackupForSecrets` (used by workflow) |
| `internal/backup/hostname.go` | Create | `ResolveHostname(fn func()(string,error), verbose bool, errOut io.Writer) string` |
| `internal/backup/resolve.go` | Modify | Add `LatestBackupID(backupsDir)(string,error)` + `ListBackupIDs(backupsDir)([]string,error)` (descending). `ResolveBackupID(id)` stays |
| `internal/actions/backup.go` | Modify | `BackupAction.Run` → build `backup.Context{FS:a.FS, HostnameFn:a.HostnameFn, Stderr:errOut,...}`, call `backup.Run`, print report from `Result`. Drop `//nolint:maintidx`. Delete `saveManifest`/`scanBackupForSecrets`/hostname inline (workflow owns) |
| `internal/actions/push.go` | Modify | Use `backup.ResolveHostname` + `backup.LatestBackupID`; add `loadConfigOr` use |
| `internal/actions/pick_backup.go` | Modify | `ResolveBackupID` delegates to `backup.LatestBackupID`; `NewRegistry` default unchanged (Tier 4) |
| `internal/actions/pull.go` | Modify | `loadConfigOr` method |
| `internal/actions/config.go` (or actions.go) | Create | `loadConfigOr(loader ConfigLoader)(*config.Config,error)` shared method |
| `internal/actions/diff_backups.go` | Modify | Extract `printDiffGroups`,`printDiffSummary` |
| `internal/actions/cleanup.go` | Modify | Use `backup.LatestBackupID`; extract `printDryRunPlan`,`confirmOrProceed` |
| `internal/actions/list_local.go` | Modify | Extract `formatBackupRow`; dedup "No backups found" |
| `internal/cloud/httputil.go` | Modify | Add `listContentsDir(client,url,token,accept,errPrefix,urlBuilder)([]BackupMeta,error)` |
| `internal/cloud/gitea.go`,`github_repo.go` | Modify | `List()` → thin delegators; token/repo guards stay in callers |
| `internal/tui/model.go` | Modify | `subModel`+`subs`+`forwardTo`; rename `Screen`→`screen`; `View`→`renderScreen`; drop `//nolint:maintidx` |
| `internal/tui/styles/styles.go` | Modify | Add `RenderTooSmall(width,height int) string` |
| `internal/tui/screens/{dashboard,health,settings,progress,wizard}.go` | Modify | Use `styles.RenderTooSmall` |
| `internal/adapters/generic.go` | Modify | Extract `buildRootItem(entry,cat)(Item,bool,error)` from `scanRootFiles` |
| `internal/adapters/yaml.go` | Modify | Extract `scanRootFilesYAML` |
| `cmd/backup.go` | Modify | Extract `applyProfileOverrides(deps,cfg)(preset,cats,adapters,error)` |
| `cmd/profile.go` | Modify | Extract `launchWizard(cfg,name,providers)(ProfileCreateFromWizard,*WizardModel,error)` |
| `.golangci.yml` | Modify | Enable 3 linters + thresholds + test exclusion |

## Interfaces / Contracts

```go
// internal/backup/workflow.go
type FS interface {
    UserHomeDir() (string, error)
    Stat(path string) (os.FileInfo, error)
    ReadDir(string) ([]os.DirEntry, error)
    MkdirAll(path string, perm os.FileMode) error
    RemoveAll(path string) error
    WalkDir(root string, fn fs.WalkDirFunc) error
    WriteFile(string, []byte, os.FileMode) error
}
type osFS struct{}
func (osFS) UserHomeDir() (string,error){ return os.UserHomeDir() }
// ... delegates to os/* + filepath.WalkDir

type Context struct {
    FS               FS
    HomeDir, BakDir  string
    Registry         *adapters.Registry
    Preset           string
    AdapterFilter    []string
    BakVersion       string
    Verbose          bool
    SecretPatterns   []*regexp.Regexp
    CustomCategories []string
    ProgressFn       ProgressFn
    ExcludesLoader   func() (adapters.ScanOptions, error)
    HostnameFn       func() (string, error) // nil → os.Hostname
    Stderr           io.Writer               // nil → os.Stderr
}
func Run(ctx Context) (*Result, error)
```

```go
// internal/cloud/httputil.go
func listContentsDir(client *http.Client, url, token, accept, errPrefix string,
    urlBuilder func(item contentResponse) string) ([]BackupMeta, error)
```

```go
// internal/tui/model.go
type subModel interface { Update(tea.Msg) (tea.Model, tea.Cmd) }
func (m *Model) forwardTo(s screen, msg tea.Msg) (tea.Model, tea.Cmd, bool)
func styles.RenderTooSmall(width, height int) string
```

```yaml
# .golangci.yml delta
linters.enable: [+ gocognit, + funlen, + nestif]
linters.settings:
  gocognit: { min-complexity: 35 }
  funlen:   { lines: 80, statements: 50 }
  nestif:   { min-complexity: 6 }
linters.exclusions.rules[test]: linters: [+ gocognit, + funlen, + nestif]
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `backup.Run` secret-exclusion (CLI+TUI produce identical manifest, no dangling refs) | [RED] new test on `BackupAction.Run` asserting `len(manifest.Items)==8` for 10 files/2 secrets (currently 10) → [GREEN] after consolidation |
| Unit | `Engine.Run` FS-injected path stays green | Existing `engine_test.go`/`integration_test.go` + `bench_test.go` unchanged |
| Unit | `listContentsDir` gitea/github parameterizations | httptest server; assert URL/accept/prefix + 404→empty |
| Unit | `LatestBackupID`/`ListBackupIDs` | Table: latest, explicit, empty→error |
| Unit | `ResolveHostname` | nil→os.Hostname, fn returns, error→"unknown"+verbose warn |
| Unit | `Model.Update`/`View` routing | Existing `model_test.go`; add: key routes to correct sub-model, `ProgressStepMsg` handled directly, unknown screen no-panic, lazy-init populates map |
| Unit | Tier-3 extract helpers | `printDiffGroups`, `printDryRunPlan`, `formatBackupRow`, `applyProfileOverrides`, `launchWizard` |
| Integration | CLI vs TUI manifest equality | Both paths over a temp-tree fixture; assert byte-identical manifest `Items` |
| Lint | `golangci-lint run ./...` | 0 gocognit/funlen/nestif on non-test; 0 maintidx without the 3 nolints |

## Migration / Rollout

No data migration / feature flags. Single PR. Tier ordering: T1 (engine →
cloud → screen → RenderTooSmall) → T2 (Model.Update → resolveBackupID →
hostname/loadConfig) → T3 (after engine consolidation to avoid churn) → linter
config committed last in same PR. Rollback: `git revert`.

## Open Questions

- [ ] **Spec location delta (needs user confirmation):** `resolveHostname` MUST
  live in `internal/backup`, not `internal/actions` as the spec states, due to
  the import-direction constraint (`backup` cannot import `actions`). Accept
  the location correction or restructure imports?
- [ ] Should `styles.RenderTooSmall` also fold the variant in
  `screens/wizard.go:242` (slightly different format string) and
  `components/modal.go:104` ("for modal"), or only the spec-listed three?
- [ ] `Engine.Run` adding `FS`/`Stderr` fields: confirm nil-safe `osFS` default
  is acceptable vs. requiring callers (tests/picker) to set them
  (recommendation: nil-safe default — zero existing call sites change).