# Design: Quality & UX Overhaul

## Architecture Overview

Four chained PRs, each independently revertable. PR1 wires the dead-end TUI flows through the existing `Deps` function-field DI pattern (`internal/tui/deps.go:8`). PR2 centralizes cross-screen UX. PR3 threads `ScanOptions` + `progressFn` through the adapter/engine layers. PR4 adds RFC 8628 OAuth alongside the existing PAT flow. All new TUI screens follow the established sub-model pattern (`Init`/`Update`/`View` on `screens.*Model`), lazily initialized from `model.go:screenChangeMsg`.

```
cmd/root.go ──inject──→ tui.Deps ──→ Model.handleMenuEnter ──→ Screen sub-models
      │                       │                                  │
      └── runBackup/runRestore adapters wrap → actions.*Action → backup.Engine
                                  ↑ progressFn        ↑ ScanOptions
                            (PR3)  │             (PR3) │
                          chan ProgressUpdate   config/ignore.go
                                  │
                            TUI ProgressStepMsg
```

## Architecture Decisions

| Decision | Choice | Alternatives | Rationale |
|----------|--------|--------------|-----------|
| DI mechanism | Function fields on `tui.Deps` (extend existing) | Interfaces per screen | Matches `deps.go:8` + `cmdDeps` pattern; AGENTS.md mandates struct-field injection |
| Progress bridge | Persistent `chan ProgressUpdate` + `chan error` on `Model`; `tea.Cmd` drains one msg, `Update` re-issues | `progressFn` directly into TUI | Channels are already the `Deps.RunBackup` contract (`deps.go:17`); Elm architecture forbids callbacks crossing `Update` |
| ScanOptions threading | Optional `adapters.ScanConfigurable` interface + exported `ScanOpts` field | Change `Adapter.ListItems` signature | Mocks unchanged → zero-value = current behavior (spec requirement); 7 delegating adapters add 3-line forwarders |
| Active profile | New top-level `Config.ActiveProfile string` | Settings field | Spec locks Settings to 7 fields; `ActiveProfile` parallels top-level `Profiles` map |
| OAuth transport | Hand-rolled `net/http` + `encoding/json`, `DeviceLoginBase` var overridable like `GistAPIBase` | `golang.org/x/oauth2` | AGENTS.md: prefer stdlib; consistent with `cloud/github_gist.go` pattern; ~150 lines |
| Modal result | `ModalResultMsg{Confirmed bool}` emitted to parent screen | `OnConfirm`/`OnCancel` callbacks | Elm architecture: parent dispatches, not callbacks; testable as pure `Update` |
| Terminal minimums | `MinWidth=30, MinHeight=15` (amend tui delta spec) | Keep spec's 40×12 | Proposal success criteria line 102 requires 30×15 to render the TUI; 40×12 contradicts acceptance — see Open Questions |

## PR1: Unblock Core (~600 lines)

### Wiring (`cmd/root.go`, `tui/deps.go`, `tui/model.go`)

Extend `Deps` (`deps.go:8`) with:
```go
RunRestore        func(backupID string, dryRun bool, out io.Writer, ch chan<- ProgressUpdate) error
ListProfiles      func() ([]ProfileInfo, error)
GetCloudStatus    func() (screens.CloudInfo, error)
SaveSetting       func(key string, value any) error
SaveProfile       func(name string, wiz actions.ProfileCreateFromWizard) error
DeleteProfile     func(name string) error
SetActiveProfile  func(name string) error
```
`cmd/root.go:34-38` injects all fields with adapters that wrap `actions.*Action` (reuse `actions.ProfileCreateInteractive`, `actions.ProfileDelete`, `actions.ProfileList`). `RunBackup` already exists (`deps.go:17`); only the injection was missing.

`handleMenuEnter` (`model.go:322`): case 0 returns `tea.Batch(screenChangeCmd, m.startBackupCmd())` instead of a bare `screenChangeMsg`. `startBackupCmd` creates `m.backupCh = make(chan ProgressUpdate, 32)` + `m.backupDone = make(chan error, 1)`, spawns `go deps.RunBackup(nil, m.backupCh)` writing the error to `backupDone`, then returns a `tea.Cmd` that selects one msg from either channel. `Update` on `ProgressStepMsg` re-issues the drain cmd; on `actionResultMsg` sends `ProgressDoneMsg` + shows toast. Cases 1 and 4 return `screenChangeMsg{ScreenRestore}` / `{ScreenProfiles}`.

### tui-restore-screen (`internal/tui/screens/restore.go` — new)
`RestoreModel` sub-model. States: `stateList → stateDryRun → stateConfirm → stateRunning`. `Init` calls `deps.ListBackups` into a `bubbles/table` (reuse `dashboardColumns`). Enter on row → `deps.RunRestore(id, true, &diffBuf, nil)` → render `diffBuf` string. Confirm modal → `deps.RunRestore(id, false, nil, ch)` with progress bridge (same channel pattern as backup). Success → `ScreenBackMsg` + success toast; error → stay + error toast. Empty state per spec. `ScreenRestore` enum added to `model.go:17`.

### tui-profiles-screen (`internal/tui/screens/profiles.go` — new)
`ProfilesModel` with `bubbles/table` (Name/Provider/Preset/Active columns). `deps.ListProfiles` returns `[]ProfileInfo{Name, Provider, Preset, Active}` built from `cfg.Profiles` + `cfg.ActiveProfile`. `n` → launches `wizardModel` (reuse `cmd/wizard.go` via a `Deps.RunWizard func() (actions.ProfileCreateFromWizard, error)` injected from `cmd/`) → on confirm `deps.SaveProfile(name, wiz)`. `enter` → `deps.SetActiveProfile(name)`. `d` → modal confirm → `deps.DeleteProfile(name)`; active profile blocked with toast. `ScreenProfiles` enum added.

### tui-modal (`internal/tui/components/modal.go` — new)
`ModalModel{Title, Message, Buttons []string, cursor int, width, height}`. `View` renders centered bordered box over dimmed background via `lipgloss.Place`. Enter emits `ModalResultMsg{Confirmed: cursor==0}`; Esc emits `Confirmed:false`; Tab cycles. Parent screens own a `*ModalModel`, render it on top when non-nil, and handle `ModalResultMsg` in their `Update`. Narrow terminal (width<40): modal width = `width-4`, min 2-col padding.

### Real Settings & Cloud (`bak-cli` delta)
`internal/config/config.go`: add `Settings Settings` struct (7 fields per spec) + `ActiveProfile string`. `Settings.Load()` defaults: `default_preset="quick"`, `auto_sync=false`, `max_file_size=1048576`, `confirm_destructive=true`. `SettingsModel.Init` calls `deps.SaveSetting`-bound loader; toggle in `Update` calls `deps.SaveSetting(key, value)` immediately (persists per-toggle). Cloud: `model.go:369` replaces `RenderCloudStatus(CloudInfo{}, …)` with `m.cloud.View()` where `CloudModel.Init` calls `deps.GetCloudStatus` → reads `cfg.Providers["github"].Token` + `cloud.ValidateToken`.

## PR2: UX Polish (~300 lines)

### Terminal guard (`styles/styles.go`)
Add `func IsTooSmall(w, h int) bool { return w < MinWidth || h < MinHeight }`. Set `MinWidth=30, MinHeight=15`. Remove local checks in `dashboard.go:148`, `settings.go:81`, `health.go:125`, `progress.go:145`, `wizard.go:223` — all call `styles.IsTooSmall`. `model.go:351` too-small view handles `KeyQuit` → `tea.Quit` (add to the `handleKey` root fallthrough).

### Welcome screen (`model.go`, `screens/welcome.go`)
Add `ScreenWelcome` enum. `NewModel`: if `deps.ConfigExists != nil && !deps.ConfigExists()` set `m.screen = ScreenWelcome`. `handleKey` ScreenWelcome: Enter → `ScreenMenu`; `q` → `tea.Quit`. `View` calls existing `screens.RenderWelcome(width)`.

### Toast positioning (`components/toast.go`, `model.go:385`)
`ToastStyle` gains `Border(lipgloss.NormalBorder()).BorderForeground(ColorGold).Background(ColorSurface)`. `model.go:View`: if `m.width >= 50`, `content = lipgloss.Place(m.width, m.height, lipgloss.Right, lipgloss.Bottom, toastView)`; else keep inline append (narrow fallback per spec). `?` overlay: `handleKey` adds `case '?'` for every screen → sets `m.showHelp bool`; `View` overlays `screens.RenderShortcuts(width)` when set.

## PR3: Backup Size & Progress (~400 lines)

### Exclusion engine (`internal/config/ignore.go` — new, ~120 lines)
Gitignore-compatible parser: `ParseIgnore(path string) ([]Pattern, error)`, `Pattern.Match(relPath string, isDir bool) bool` (supports `node_modules`, `*.lock`, `dir/`, `!negation`). `LoadExcludes(configDir string, settings Settings) ([]string, int64, error)` merges defaults + `~/.config/bak/ignore` + `settings.ExcludePatterns` (empty array = clear defaults; non-empty = replace defaults per spec). `MaxFileSize` from settings (default 1 MiB). Reloaded per backup run (no caching) per "ignore file reload" scenario.

### ScanOptions (`internal/adapters`)
New `ScanOptions{Excludes []string, MaxFileSize int64}` + optional interface `ScanConfigurable interface { SetScanOptions(ScanOptions) }`. `GenericAdapter` + `opencode.Adapter` gain exported `ScanOpts ScanOptions` + `SetScanOptions` method; `scanDir` reads `opts.Excludes` (skip dirs/files via `Pattern.Match`) and `opts.MaxFileSize` (skip + stderr warning). 7 delegating adapters (windsurf/codex/cursor/claudecode/kilocode/kiro/pidev) add `func (a *Adapter) SetScanOptions(o adapters.ScanOptions) { a.base.ScanOpts = o }`. `Engine.Run` + `BackupAction.Run` build `ScanOptions` from `config.LoadExcludes` and call `SetScanOptions` on each detected adapter before `ListItems`. Zero-value `ScanOptions{}` → current behavior (mocks unaffected).

### Progress reporting (`internal/backup/engine.go`, `internal/actions/{backup,restore}.go`)
`Engine` gains `ProgressFn func(currentFile string, done, total int)` (nil-safe: guard before each call). Called once per file inside the `detected` loop after `Backup`. `BackupAction.ProgressFn` forwards to engine. `RestoreAction.ProgressFn` called per `restoreFile`. `cmd/root.go` `runBackup`/`runRestore` adapters convert `progressFn` → `chan<- ProgressUpdate` sends (`ProgressUpdate{Step: currentFile, Current: done, Total: total}`; final `Done: true`). Cloud push/pull: `PushAction`/`PullAction` gain optional `ProgressFn` (SHOULD per spec) — single "Packaging" + "Uploading" coarse callbacks.

## PR4: OAuth Login (~400 lines)

### Device Flow (`internal/cloud/oauth_device.go` — new, ~150 lines)
```go
type DeviceClient struct {
    ClientID  string
    HTTPClient *http.Client      // defaults to cloud.httpClient
    BaseURL   string             // defaults to DeviceLoginBase ("https://github.com"), overridable for tests
    Out       io.Writer
    OpenBrowser func(url string) error  // injectable; default openBrowserOS
    Clipboard   func(s string) error    // injectable; default clipboard.WriteAll
}
```
Sequence (RFC 8628):
1. `POST {BaseURL}/login/device/code` (form: `client_id`, `scope=gist`) → `device_code, user_code, verification_uri, interval, expires_in`.
2. `Clipboard(user_code)` (best-effort) + `OpenBrowser(verification_uri)`. `openBrowserOS` switches on `runtime.GOOS`: `darwin`→`open`, `linux`→`xdg-open` (only if `os.Getenv("DISPLAY") != ""`), `windows`→`rundll32 url.dll,FileProtocolHandler`. Headless → print URL + code, skip open.
3. Poll `POST {BaseURL}/login/oauth/access_token` every `interval` until `access_token` or `error` (`expired_token`→"Code expired", `access_denied`→"Authorization denied", `slow_down`→increase interval).
4. Return token.

### Token management & dispatch (`internal/actions/login.go`, `cmd/login.go`)
`LoginAction.Run` (line 41): before the PAT prompt, if `os.Getenv("BAK_GITHUB_OAUTH_CLIENT_ID") != ""` → call `DeviceClient.RequestToken()` → `cloud.ValidateToken(token)` (reuse `auth.go:82`) → `ConfigSaver.Set("github.token", token)`. Else fall through to existing PAT paste (unchanged). `LoginAction` gains `OAuthClient *cloud.DeviceClient` field (nil = no OAuth). Token stored under same `github.token` key → `cloud.ResolveToken` (`auth.go:63`) unchanged → push/pull work with no cloud-sync code changes (cloud-sync delta's "both token types" satisfied automatically). `BAK_GITHUB_OAUTH_CLIENT_ID` absent → feature dormant (rollback-safe).

## File Changes

| File | Action | PR |
|------|--------|----|
| `internal/tui/deps.go` | Modify | 1 — 7 new function fields + `ProfileInfo` type |
| `cmd/root.go` | Modify | 1 — inject all deps; `runBackup`/`runRestore`/`runWizard` adapters |
| `internal/tui/model.go` | Modify | 1+2 — `handleMenuEnter` cases 0/1/4; `ScreenRestore`/`ScreenProfiles`/`ScreenWelcome` enums; backup channels; toast `lipgloss.Place`; `?` overlay; too-small `q` |
| `internal/tui/screens/restore.go` | Create | 1 |
| `internal/tui/screens/profiles.go` | Create | 1 |
| `internal/tui/screens/cloud.go` | Modify | 1 — `CloudModel` sub-model |
| `internal/tui/screens/settings.go` | Modify | 1 — real options + `deps.SaveSetting` |
| `internal/tui/components/modal.go` | Create | 1 |
| `internal/config/config.go` | Modify | 1 — `Settings` struct + `ActiveProfile` + Load/Save |
| `internal/tui/styles/styles.go` | Modify | 2 — `IsTooSmall`, `MinWidth=30`, `MinHeight=15`, bordered `ToastStyle` |
| `internal/tui/screens/{dashboard,health,progress,wizard}.go` | Modify | 2 — call `styles.IsTooSmall` |
| `internal/config/ignore.go` | Create | 3 |
| `internal/adapters/adapter.go` | Modify | 3 — `ScanOptions` + `ScanConfigurable` interface |
| `internal/adapters/generic.go` | Modify | 3 — `ScanOpts` field + `scanDir` opts |
| `internal/adapters/opencode/adapter.go` | Modify | 3 — `ScanOpts` field + `scanDir` opts |
| `internal/adapters/{windsurf,codex,cursor,claudecode,kilocode,kiro,pidev}/adapter.go` | Modify | 3 — `SetScanOptions` forwarders |
| `internal/backup/engine.go` | Modify | 3 — `ProgressFn` field + per-file calls |
| `internal/actions/{backup,restore,push,pull}.go` | Modify | 3 — `ProgressFn` forwarding |
| `internal/cloud/oauth_device.go` | Create | 4 |
| `internal/actions/login.go` | Modify | 4 — OAuth dispatch |
| `cmd/login.go` | Modify | 4 — wire `DeviceClient` |

## Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Unit (config) | `ignore.go` parse/match/merge; `Settings` round-trip; defaults | Table-driven; `t.TempDir` + `setConfigHome`; RED first (strict TDD) |
| Unit (adapters) | `ScanOptions` excludes dirs/files; `MaxFileSize` skip; zero-value = current | `t.TempDir` fixtures with `node_modules/`, 5MB binary; `GenericAdapter.ScanOpts` |
| Unit (engine) | `ProgressFn` called N times with incrementing `done`; nil-safe | Mock adapter returning fixed items; assert call order |
| Unit (oauth) | Device code request; polling states (success/expire/deny/slow_down); headless fallback | `httptest.Server` + injectable `OpenBrowser`/`Clipboard`; `DeviceLoginBase` override like `GistAPIBase` |
| Unit (TUI) | `RestoreModel`/`ProfilesModel`/`ModalModel` `Update`+`View`; `handleMenuEnter` 0/1/4; backup channel drain; toast placement at 30×15/40×12/80×24 | Pure-function `Update`/`View` tests; hand-rolled `MockDeps`; table-driven render matrix |
| Integration | `cmd/root.go` injection completeness; `LoginAction` OAuth vs PAT dispatch | Assert all `Deps` fields non-nil; mock OAuth server end-to-end |
| E2E | Disabled per `config.yaml` — not added | n/a |

Test doubles: hand-rolled `MockDeps` (function fields), `mockAdapter` extended with `SetScanOptions` only where testing opts. `var _ adapters.ScanConfigurable = (*GenericAdapter)(nil)` compile checks.

## Migration / Rollout

No data migration. `Settings` absent in old `config.json` → defaults apply (zero-value). `ActiveProfile` empty → no active marker. OAuth dormant until `BAK_GITHUB_OAUTH_CLIENT_ID` set. Each PR reverts independently (PR1 highest-risk: restores dead-end behavior; PR3 `ScanOptions` zero-value fall-through; PR4 env-gated).

## Risks & Mitigations

| Risk | Mitigation |
|------|-----------|
| `progressFn` refactor breaks 200+ existing tests | Nil-safe optional field; existing callers pass nil; new test suite for ordering only |
| Backup channel drain blocks TUI on empty backup | `select` on `backupDone` with timeout-free fallback; `RunBackup` always sends `Done:true` |
| `ScanOptions` field mutation on shared registry adapters | bak is single-threaded per run; tests construct fresh adapters; document non-concurrency |
| GitHub OAuth App not registered before PR4 | Env-gated; manual PAT always works; document prerequisite in PR4 |
| Browser open on headless Linux | `DISPLAY` check; print URL fallback; `atotto/clipboard` copy |
| Modal focus/keyboard conflicts with parent screen | Parent `Update` short-circuits when `modal != nil` |

## Open Questions

- [ ] **Terminal minimums conflict**: tui delta spec locks `MinWidth=40, MinHeight=12` but proposal success criteria (line 102) requires 30×15 to render the TUI. Design resolves to 30×15 (acceptance-driven); the tui delta spec constant line needs amending during `sdd-apply`. **Needs orchestrator/user confirmation.**
- [ ] `wizardModel` lives in `cmd/` package — reusing it from the TUI `ProfilesModel` (in `internal/tui/screens/`) requires either moving `wizardModel` to `internal/tui/screens/wizard.go` or injecting a `Deps.RunWizard func() (actions.ProfileCreateFromWizard, error)` adapter built in `cmd/`. Design proposes injection (keeps `cmd/` as cobra boundary); confirm.
- [ ] OAuth client ID: registered GitHub OAuth App client ID value + whether to hardcode a fallback or require env var only. Proposal says env-var-only.
