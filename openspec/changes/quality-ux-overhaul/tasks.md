# Tasks: Quality & UX Overhaul

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1700 (PR1 ~600, PR2 ~300, PR3 ~400, PR4 ~400) |
| 400-line budget risk | High (all 4 PRs) |
| Chained PRs recommended | Yes |
| Suggested split | PR1 → PR2 → PR3 → PR4 |
| Delivery strategy | ask-on-risk |
| Chain strategy | feature-branch-chain |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | PR | Notes |
|------|------|----|-------|
| 1 | Unblock core TUI flows | PR1 | ~600 lines; modal, restore, profiles, settings, cloud, deps |
| 2 | UX polish | PR2 | ~300 lines; terminal guard, welcome, toast, help overlay |
| 3 | Backup size & progress | PR3 | ~400 lines; exclude engine, scan options, progress callbacks |
| 4 | OAuth login | PR4 | ~400 lines; device flow, browser, token storage |

## PR1: Unblock Core (~600 lines)

### Phase 1.1: Modal Component
- [x] 1.1.1 Write `internal/tui/components/modal_test.go` — ModalResultMsg on Enter/Esc, Tab cycling, narrow width
- [x] 1.1.2 Write `internal/tui/components/modal.go` — ModalModel{Title,Message,Buttons,cursor}, Init/Update/View, centered bordered overlay

### Phase 1.2: Settings Persistence
- [x] 1.2.1 Write `internal/config/config_test.go` — Settings Load/Save round-trip, defaults (preset=quick, auto_sync=false, max_file_size=1048576, confirm_destructive=true), ActiveProfile
- [x] 1.2.2 Modify `internal/config/config.go` — add Settings struct (7 fields) + ActiveProfile + Load/Save
- [x] 1.2.3 Write `internal/tui/screens/settings_test.go` — Init loads from deps, toggle calls SaveSetting
- [x] 1.2.4 Modify `internal/tui/screens/settings.go` — real Settings-backed toggles, persist per-toggle

### Phase 1.3: Restore Screen
- [x] 1.3.1 Write `internal/tui/screens/restore_test.go` — states (list→dryrun→confirm→running), empty state, diff preview, modal confirm, success/error toast
- [x] 1.3.2 Write `internal/tui/screens/restore.go` — RestoreModel with table, deps.ListBackups, deps.RunRestore(dryRun), modal confirm

### Phase 1.4: Profiles Screen
- [x] 1.4.1 Write `internal/tui/screens/profiles_test.go` — list/create(n)/switch(enter)/delete(d), active-profile guard, wizard injection
- [x] 1.4.2 Write `internal/tui/screens/profiles.go` — ProfilesModel with table(Name/Provider/Preset/Active), deps.RunWizard from cmd/

### Phase 1.5: Cloud Screen
- [x] 1.5.1 Write `internal/tui/screens/cloud_test.go` — Init calls deps.GetCloudStatus, renders provider + token validity
- [x] 1.5.2 Modify `internal/tui/screens/cloud.go` — CloudModel sub-model, replace empty CloudInfo{} with real data

### Phase 1.6: Deps Wiring
- [x] 1.6.1 Modify `internal/tui/deps.go` — add RunRestore, ListProfiles, GetCloudStatus, SaveSetting, SaveProfile, DeleteProfile, SetActiveProfile, RunWizard + ProfileInfo type
- [x] 1.6.2 Write `internal/tui/model_test.go` — handleMenuEnter cases 0/1/4, backup channel drain
- [x] 1.6.3 Modify `internal/tui/model.go` — ScreenRestore/ScreenProfiles/ScreenWelcome enums, handleMenuEnter real dispatch, backupCh/backupDone channels, startBackupCmd
- [x] 1.6.4 Modify `cmd/root.go` — inject all Deps fields wrapping actions.*Action

### Phase 1.7: Quality Gates
- [x] 1.7.1 `go test -race ./...` + `go vet ./...` + `golangci-lint run` — all clean

## PR2: UX Polish (~300 lines)

### Phase 2.1: Terminal Guard
- [x] 2.1.1 Write `internal/tui/styles/styles_test.go` — IsTooSmall at 30×15, 20×10, 80×24
- [x] 2.1.2 Modify `internal/tui/styles/styles.go` — IsTooSmall(w,h), MinWidth=30, MinHeight=15
- [x] 2.1.3 Modify `screens/{dashboard,settings,health,progress}.go` + `cmd/wizard.go` — replace 5 local checks with styles.IsTooSmall

### Phase 2.2: Welcome Screen
- [x] 2.2.1 Write `internal/tui/model_test.go` — ConfigExists=false→welcome, Enter→menu, q→quit
- [x] 2.2.2 Modify `internal/tui/model.go` — ScreenWelcome enum, NewModel checks ConfigExists, handleKey routes

### Phase 2.3: Toast Positioning
- [x] 2.3.1 Write `internal/tui/components/toast_test.go` — bordered toast at bottom-right (80×24), inline fallback (30×15)
- [x] 2.3.2 Modify `internal/tui/components/toast.go` — Border + Background on ToastStyle
- [x] 2.3.3 Modify `internal/tui/model.go` — lipgloss.Place when width≥50, inline otherwise

### Phase 2.4: Help Overlay
- [x] 2.4.1 Write `internal/tui/model_test.go` — '?' toggles showHelp on every screen, dismiss with '?'/Esc
- [x] 2.4.2 Modify `internal/tui/model.go` — showHelp bool, '?' handler, View overlays RenderShortcuts

### Phase 2.5: Quality Gates
- [x] 2.5.1 `go test -race ./...` + `go vet ./...` + `golangci-lint run` — all clean

## PR3: Backup Size & Progress (~400 lines)

### Phase 3.1: Exclusion Engine
- [x] 3.1.1 Write `internal/config/ignore_test.go` — ParseIgnore (wildcards, dir/, !negation), LoadExcludes merge, empty-array-clears-defaults
- [x] 3.1.2 Write `internal/config/ignore.go` — ParseIgnore, Pattern.Match, LoadExcludes(configDir, settings)

### Phase 3.2: ScanOptions Plumbing
- [x] 3.2.1 Write `internal/adapters/adapter_test.go` — ScanConfigurable compliance, SetScanOptions
- [x] 3.2.2 Modify `internal/adapters/adapter.go` — ScanOptions struct + ScanConfigurable interface
- [x] 3.2.3 Modify `internal/adapters/generic.go` — ScanOpts field, excludes+MaxFileSize in scanDir
- [x] 3.2.4 Modify `internal/adapters/opencode/adapter.go` — same ScanOpts + scanDir filtering
- [x] 3.2.5 Modify 7 delegating adapters — SetScanOptions forwarders

### Phase 3.3: Progress Callback
- [x] 3.3.1 Write `internal/backup/engine_test.go` — ProgressFn called N times incrementing, nil-safe
- [x] 3.3.2 Modify `internal/backup/engine.go` — ProgressFn field, per-file call (nil guard)
- [x] 3.3.3 Modify `internal/actions/{backup,restore}.go` — ProgressFn forwarding
- [x] 3.3.4 Modify `internal/actions/{push,pull}.go` — optional coarse ProgressFn

### Phase 3.4: TUI Progress Bridge
- [x] 3.4.1 Write `internal/tui/model_test.go` — progressFn→chan→ProgressStepMsg bridge, ProgressDoneMsg
- [x] 3.4.2 Modify `cmd/root.go` — runBackup/runRestore adapters: progressFn → chan<- ProgressUpdate

### Phase 3.5: Quality Gates
- [x] 3.5.1 `go test -race ./...` + `go vet ./...` + `golangci-lint run` — all clean

## PR4: OAuth Login (~400 lines)

### Phase 4.1: OAuth Device Flow
- [ ] 4.1.1 Write `internal/cloud/oauth_device_test.go` — device code request, polling (success/expire/deny/slow_down), headless fallback via httptest.Server
- [ ] 4.1.2 Write `internal/cloud/oauth_device.go` — DeviceClient, RequestToken() RFC 8628 (POST device/code → poll oauth/access_token)

### Phase 4.2: Browser Opener
- [ ] 4.2.1 Write `internal/cloud/browser_test.go` — openBrowserOS per GOOS, DISPLAY guard on Linux
- [ ] 4.2.2 Write `internal/cloud/browser.go` — openBrowserOS(url) via runtime.GOOS switch, DISPLAY check

### Phase 4.3: Token Storage & Dispatch
- [ ] 4.3.1 Write `internal/actions/login_test.go` — OAuth dispatch when clientID set, PAT fallback, token validate+save
- [ ] 4.3.2 Modify `internal/actions/login.go` — OAuthClient field, dispatch to DeviceClient.RequestToken, PAT fallback
- [ ] 4.3.3 Modify `cmd/login.go` — wire DeviceClient with BAK_GITHUB_OAUTH_CLIENT_ID env, injectable OpenBrowser/Clipboard

### Phase 4.4: Quality Gates
- [ ] 4.4.1 `go test -race ./...` + `go vet ./...` + `golangci-lint run` — all clean
