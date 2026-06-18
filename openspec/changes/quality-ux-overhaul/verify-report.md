# Verify Report — quality-ux-overhaul

**Change:** quality-ux-overhaul
**Mode:** openspec (file persistence)
**Artifact set:** Full (proposal + design + tasks + 11 specs)
**Verification date:** 2026-06-17
**Verifier:** sdd-verify executor (source inspection + runtime execution)

## Verdict: FAIL

The change is broad and much of it is well-built at the component/unit level, but
several **core product goals are not realized in the production wiring**, and the
`golangci-lint` quality gate does not pass. Per the SDD verify decision gates,
spec scenarios that describe end-to-end system behavior but have no passing
covering test at the integration level are CRITICAL, and a non-zero quality gate
is CRITICAL. These block archive readiness.

**Quality gates**

| Gate | Result | Evidence |
|------|--------|----------|
| `go test -race -count=1 ./...` | PASS | exit 0; all 28 packages pass (cmd 19.7s, actions, adapters×8, backup, cloud, config, tui, tui/{components,screens,styles}, e2e) |
| `go vet ./...` | PASS | exit 0, clean |
| `golangci-lint run` | **FAIL** | exit 1 — 4 issues: `gocritic ifElseChain` (profiles.go:193, restore.go:226), `goimports` (login_test.go:61, oauth_device_test.go:136) |

**Task completeness:** 50/50 tasks marked `[x]` in tasks.md. However, the
"Quality Gates" sub-tasks (1.7.1, 2.5.1, 3.5.1, 4.4.1) each claim
"`golangci-lint run` — all clean", which is **not true** (exit 1). Several
implementation tasks are checked off but the production wiring they describe is
missing or stubbed (see CRITICAL findings).

---

## PR1: Unblock Core

### Capability: tui-modal — PASS
- [PASS] REQ-MODAL confirm dialog: `ModalModel{Title,Message,Buttons,cursor}`, centered bordered overlay via `lipgloss.Place`, Enter on index 0 → `ModalResultMsg{Confirmed:true}` (`modal.go:77-79`).
- [PASS] REQ-MODAL alert dialog: single-button modal; Enter (cursor 0) and Esc both close (`ModalResultMsg` emitted).
- [PASS] REQ-MODAL keyboard nav: Tab/Shift+Tab cycle buttons (`modal.go:84-93`), Enter activates focused, Esc cancels.
- [PASS] REQ-MODAL layout/styling: `modalFrameStyle` uses `lipgloss.DoubleBorder()` + `styles.ColorOverlay`; centered; narrow terminal adapts width (`modal.go:114-121`); too-small guard at <20×10.
- Tests: `modal_test.go` table-driven (Enter/Esc/Tab cycling, empty buttons, narrow, cursor highlight, WindowSize). Thorough.
- **Design deviation (accepted):** spec text mentions `OnConfirm`/`OnCancel` callbacks; implementation emits `ModalResultMsg{Confirmed bool}` (Elm architecture). Documented in `design.md` Architecture Decisions. Behavior equivalent; parent screens handle `ModalResultMsg`. Non-blocking.

### Capability: tui-restore-screen — FAIL (screen model OK; production wiring stubbed)
- [PASS] REQ-RESTORE backup list display (screen): `RestoreModel` renders table with ID/Date/Size/Cloud; empty state "No backups found. Create one first." (`restore.go:257-265`).
- [WARN] REQ-RESTORE "sorted by date descending": `renderBackupList` renders in `listBackups` order; **no sorting**. `cmd/root.go listBackupsFrom` does not sort. Scenario "Populate from disk" (sorted by date descending) unmet.
- [FAIL] REQ-RESTORE dry-run diff preview (production): screen calls `deps.RunRestore(id, true)` and displays output (`restore.go:202-209`). **But `cmd/root.go tuiRunRestore` is a stub** returning the hardcoded string `"dry-run: no changes detected"` — it never calls `actions.RestoreAction` and produces no real diff. Scenario "Preview selected backup" / "Diff shows file changes" unmet in production. This also violates AGENTS.md "MUST NOT restore without mandatory dry-run diff" (the diff is faked).
- [PASS] REQ-RESTORE confirm before execute (screen): dry-run → Enter → `ModalModel` "Confirm Restore" (`restore.go:183-189`); confirm → `runRestoreCmd`; cancel → back to list.
- [FAIL] REQ-RESTORE restore execution (production): confirm calls `deps.RunRestore(id, false)`, but `tuiRunRestore` is a stub returning `"restored successfully"` **without executing any restore**. Scenario "User confirms" (execute the restore) unmet in production.
- [WARN] REQ-RESTORE "success toast SHALL appear" / "return to main menu": implementation shows an inline `restoreStateDone` view ("Restore completed successfully." / "Error: …") and requires a keypress (q/enter/space) to emit `ScreenBackMsg`. No toast is shown (restore never emits `actionResultMsg`); no auto-return on success. Feedback exists but not per spec mechanism. "Restore error" stays on restore screen ✓ (done state is restore).
- [WARN] REQ-RESTORE "SHALL display progress during restore execution": `restoreStateRunning` renders a static "Restoring backup X…" — no per-file progress. `RunRestore` signature has no progress channel, so restore progress is not bridged to the TUI (contrast: backup bridge works).
- Tests: `restore_test.go` covers states, empty, dry-run, confirm, cancel, error, views — all with a **mock** `runRestoreFn`. Tests pass but do not exercise the production stub.

### Capability: tui-profiles-screen — FAIL (screen model OK; wizard stubbed)
- [PASS] REQ-PROFILES list display: `ProfilesModel` table with name/provider/preset + `*` active marker; empty state "No profiles yet. Press 'n' to create one." (`profiles.go:212-224`).
- [FAIL] REQ-PROFILES create via wizard (production): screen calls `deps.RunWizard()` on 'n' (`profiles.go:173-179`). **But `cmd/root.go tuiRunWizard` is a stub** returning a hardcoded `ProfileInfo{Name:"default", Provider:"github", Preset:"quick"}` — it never launches the 5-step wizard. Scenario "Complete wizard" unmet in production (pressing 'n' appends a fake "default" profile). "Cancel wizard" scenario cannot occur (no wizard to cancel).
- [PASS] REQ-PROFILES switch active: Enter → `deps.SetActiveProfile` + local active marker update (`profiles.go:150-160`).
- [PASS] REQ-PROFILES delete with confirmation: 'd' → modal; active profile blocked with `m.Msg = "Cannot delete the active profile"` (`profiles.go:161-172`); confirm → `deps.DeleteProfile` + list refresh.
- [WARN] AGENTS.md violation: `_ = m.setActive(name)`, `_ = m.deleteProfile(name)`, `_ = m.SaveProfile(...)` explicitly discard error returns (`profiles.go:154,111,101`). Rule: "MUST handle ALL returned errors — no `_ =` for error returns". Not flagged by golangci (errcheck allows explicit `_ =`).
- Tests: `profiles_test.go` covers list/empty/switch/delete-active-guard/create-via-wizard/list-error — all with mock deps.

### Capability: tui-welcome-screen — FAIL (routing PASS; content mismatch)
- [PASS] REQ-WELCOME first-run detection: `NewModel` checks `deps.ConfigExists`; false → `ScreenWelcome` (`model.go:101-105`). `cmd/root.go configExists` stat's config path.
- [PASS] REQ-WELCOME navigate to menu: Enter → `ScreenMenu`; q/Esc → `tea.Quit` (`model.go:372-379`). Tests `TestModel_Welcome_EnterToMenu`/`_Quit`/`_EscQuit` pass.
- [FAIL] REQ-WELCOME welcome content: scenario requires ASCII logo + `"Pack your AI coding setup. Move anywhere."` + `"Press Enter to get started"`. `RenderWelcome` (`welcome.go`) renders `"Welcome to bak!"`, first-time text, and `"Press enter to continue, or q to quit."`. **No ASCII logo** (logo lives in `styles/logo.go` and is only used by `RenderMainMenu`), **tagline missing**, prompt wording differs. `TestModel_Welcome_View` only checks substrings "Welcome"/"enter" (loose), so it passes despite the mismatch.

### Capability: tui (delta) — PASS WITH WARNINGS
- [PASS] REQ-TUI centralized terminal guard: `styles.IsTooSmall(w,h)` with `MinWidth=30`, `MinHeight=15` (`styles.go:81-91`). Used by dashboard, health, progress, settings, wizard (`cmd/wizard.go:223`), profiles (inline equivalent).
- [PASS] REQ-TUI help overlay: `?` toggles `showHelp` globally; Esc/`?` dismiss; `View` overlays `RenderShortcuts` (`model.go:175-191,568-571`). Works on sub-screens (`TestModel_Help_ToggleOnSubScreen`).
- [PASS] REQ-TUI menu cursor 1/4 handlers: `handleMenuEnter` case 1 → `ScreenRestore`, case 4 → `ScreenProfiles` (`model.go:494-501`). Cases 0/2/3/5/6 wired.
- [PASS] REQ-TUI toast notification: `ToastStyle` has `Border` + `Background(ColorSurface)` (`styles.go:60-65`); `View` uses `lipgloss.Place(Right,Bottom)` when width≥50, inline append otherwise (`model.go:576-586`). Auto-hide via ticks. `TestModel_Toast_PositionedWide`/`_InlineNarrow` pass.
- [WARN] REQ-TUI "Quit from too-small view": when `tooSmall`, `Update` still routes through `handleKey` by screen. From `ScreenMenu`/`ScreenWelcome` (launch state) `q` quits ✓. From a sub-screen resized small, `q` may route to that screen's back handler instead of quitting. Edge case; common launch-small path works.
- [WARN] **Spec text not amended:** `tui` delta spec line 7 states `MinWidth=40, MinHeight=12`; code/tests use 30×15. `design.md` Open Questions flagged this and said the spec constant line needs amending during `sdd-apply`. **Spec text is stale.** Code follows the design resolution (30×15) which matches the proposal success criteria (30×15 renders, 20×10 too small).

### Capability: bak-cli (delta) — FAIL
- [PASS] REQ-BAK-CLI Settings schema: `Settings` struct has all 7 fields (`config.go:42-64`); persists via `Load`/`Save` (JSON `settings`). `ActiveProfile` top-level field present.
- [PASS] REQ-BAK-CLI Settings round-trip (save half): toggling auto_sync → `tuiSaveSetting` writes `config.json` (`root.go:263-292`). `TestSettings_SaveLoadRoundTrip` passes.
- [FAIL] REQ-BAK-CLI Settings round-trip (reload half): scenario "relaunching the TUI SHALL show `auto_sync` as enabled" **unmet**. `model.go:213` wires `screens.NewSettingsModel(m.deps.SaveSetting)` (defaults), NOT `NewSettingsModelWithSettings`. There is no `LoadSettings`/`GetSettings` dep. On relaunch the TUI always shows defaults (auto_sync=false) regardless of saved config. `TestSettings_LoadInitialSettings` tests `NewSettingsModelWithSettings` directly (not via model wiring), so it passes but doesn't cover the gap.
- [FAIL] REQ-BAK-CLI "Default settings": scenario requires `default_preset="quick"`, `max_file_size=1048576`, `confirm_destructive=true` when no config exists. `config.Load()` returns **zero-value** Settings (`DefaultPreset=""`, `MaxFileSize=0`, `ConfirmDestructive=false`). `DefaultSettings()` exists (`config.go:67-74`) with correct values but is **never called by `Load`/`LoadPath`**. `TestSettings_LoadDefaultsWhenMissing` encodes the **wrong** behavior (asserts `DefaultPreset == ""`). Scenario unmet.
- [PASS] REQ-BAK-CLI Cloud screen data: `CloudModel.Init` calls `deps.GetCloudStatus` → real config data (`cloud.go:47-55`, `root.go:244-260`). Replaces empty `CloudInfo{}`.
- [WARN] REQ-BAK-CLI "Cloud screen without provider" wants "instructions to run `bak login`": `cloud.go:109-117` shows "No cloud provider configured" + help bar (q back) but **no `bak login` instructions**.
- [PASS] REQ-BAK-CLI Action DI wiring: `tui.Deps` has `RunBackup,RunRestore,ListBackups,ListProfiles,GetCloudStatus,SaveSetting,ConfigExists` (+ `SaveProfile,DeleteProfile,SetActiveProfile,RunWizard`); `cmd/root.go:37-50` injects all. `TestSettings_SaveSetting_CalledOnToggle` verifies per-toggle persist.
- [WARN] `tuiCloudStatus` uses `token != ""` for `Connected` instead of `cloud.ValidateToken` (design said ValidateToken); `LastSync` hardcoded `"never"`. Provider name + token-presence status shown; "token validity status" approximated.

---

## PR2: UX Polish

### Capability: tui (terminal guard / welcome / toast / help) — PASS WITH WARNINGS
- [PASS] Terminal guard: `IsTooSmall` 30×15; `styles_test.go TestIsTooSmall` matrix (30×15 false, 29×20 true, 50×14 true, 20×10 true, 80×24 false). Replaces local checks in 5 screens + wizard.
- [FAIL] Welcome content: see PR1 tui-welcome-screen (ASCII logo + tagline missing). Routing PASS.
- [PASS] Toast positioning: bordered + background; `lipgloss.Place` wide, inline narrow (<50). `TestModel_Toast_PositionedWide`/`_InlineNarrow`; `TestToastStyle_HasBorder`/`_HasBackground`.
- [PASS] Help overlay: `?` toggles on every screen; Esc/`?` dismiss; `RenderShortcuts` overlay. `TestModel_Help_*` (toggle, dismiss-via-esc, view-contains, sub-screen).
- [WARN] Stale `TestScreenIotaValues` only asserts ScreenMenu..ScreenHealth (0-6); does not cover new ScreenRestore/ScreenProfiles/ScreenWelcome (7-9). Cosmetic; new screens appended at end so 0-6 values unchanged.

---

## PR3: Backup Size & Progress

### Capability: backup-exclude-rules — FAIL (engine never engages exclusions)
- [PASS] Default exclusion patterns exist: `DefaultExcludes` = node_modules/, .git/, *.lock, *.log, *.png, *.jpg, *.jpeg, *.zip, *.tar, *.gz, *.exe, *.dll, *.so, *.dylib (`ignore.go:19-34`). Matches spec list (+ *.jpeg extra).
- [PASS] Custom ignore file (unit): `LoadExcludes(configDir, settings)` reads `ignore` file, merges with defaults; reload-per-call (no caching) (`ignore.go:182-213`). `TestLoadExcludesIgnoreReload` passes.
- [PASS] Opt-out via config (unit): nil `ExcludePatterns` → defaults; non-nil → replaces; empty slice → clears defaults (`ignore.go:186-190`). `TestLoadExcludes` "empty exclude_patterns clears defaults" / "override defaults" pass.
- [FAIL] **Default patterns applied in production:** `Engine.Run` (`engine.go:56-257`) and `BackupAction.Run` (`backup.go:54-263`) **never call `config.LoadExcludes` and never call `SetScanOptions` on detected adapters** before `ListItems`. Grep confirms zero production callers of `SetScanOptions`/`LoadExcludes` (only definitions + tests). Adapters run with zero-value `ScanOpts` → all files included, no exclusions, no size cap. Scenarios "node_modules excluded", "Custom pattern applied", "Large file skipped", "Override defaults" **unmet in production**. Proposal success criterion "50 MB node_modules → backup < 2 MB" unmet.
- [FAIL] **MaxFileSize default cap:** `LoadExcludes` returns `settings.MaxFileSize` as-is (0 when unset — `TestLoadExcludes` "default patterns… wantMaxSize: 0"). Combined with `Load()` not applying `DefaultSettings()` (PR1 #5), the default 1 MB cap never applies. Scenario "Large file skipped" (default 1 MB) unmet.
- [WARN] "lock files excluded" scenario names `package-lock.json` and `yarn.lock`. Default pattern `*.lock` matches `yarn.lock` ✓ but **not** `package-lock.json` (ends in `.json`). `TestPatternMatch` explicitly asserts `package-lock.json` is NOT matched. Spec internal inconsistency (pattern list vs scenario expectation); implementation follows the documented `*.lock` pattern literally.

### Capability: backup-engine (delta) — PARTIAL
- [PASS] REQ-BE progress callback: `Engine.ProgressFn func(currentFile string, filesDone, filesTotal int)`, nil-safe guard (`engine.go:42,195-197`). `TestEngine_Run_ProgressFn` verifies incrementing done, consistent total, last done==total; `TestEngine_Run_ProgressFnNilSafe` verifies nil safety.
- [PASS] REQ-BE ScanOptions struct + ScanConfigurable interface (`adapter.go:30-43`); `GenericAdapter` + `opencode.Adapter` have `ScanOpts` field + `SetScanOptions` + `scanDir` filtering + MaxFileSize stderr warning. 7 delegating adapters forward `SetScanOptions`. Compile-time checks present. Zero-value = current behavior (`testScanOptionsZeroValueFallThrough`).
- [FAIL] REQ-BE "MUST apply exclusion rules from ScanOptions (default + custom + max size) before copying files" (Preset-based backup MODIFIED): not applied — see backup-exclude-rules FAIL above. `TestEngine_Run_FullPreset` uses a fixture with no node_modules/large files, so it passes but does not cover exclusion.
- [PASS] Preset-based backup (quick/full/skills) still works; manifest created; secrets scanned.

### Capability: progress-reporting — PASS (backup); PARTIAL (restore TUI bridge)
- [PASS] REQ-PR engine progress: `Engine.ProgressFn` per-file, nil-safe.
- [PASS] REQ-PR action-level progress: `BackupAction.ProgressFn` forwards to engine (`backup.go:45,206-208`); `RestoreAction.ProgressFn` called per restored file with filesTotal/filesDone (`restore.go:30,135-149`).
- [PASS] REQ-PR TUI progress bridge (backup): `cmd/root.go tuiRunBackup` bridges `progressFn` → `chan<- ProgressUpdate` (`root.go:177-188`); `model.go drainProgressCmd` → `ProgressStepMsg`/`ProgressDoneMsg` → progress sub-model + toast. `TestBackupChannelBridge*` pass.
- [WARN] REQ-PR TUI progress bridge (restore): not implemented. `RunRestore` signature `(string,error)` has no channel; restore done state shows static message. Design said "same channel pattern as backup". Restore progress never reaches TUI.
- [PASS] REQ-PR cloud push/pull progress (SHOULD): `PushAction.ProgressFn` / `PullAction.ProgressFn` coarse milestones (Packaging/Uploading/Downloading/Complete) (`push.go:103-167`, `pull.go:90-141`). SHOULD satisfied.

---

## PR4: OAuth Login

### Capability: oauth-device-flow — PASS WITH WARNINGS
- [PASS] REQ-ODF device code request: `requestDeviceCode` POSTs `/login/device/code` with `client_id` + `scope=gist` (`oauth_device.go:169-191`). `TestDeviceClient_PollingStates` + mock server verify.
- [PASS] REQ-ODF user code display + browser open: `RequestToken` prints verification URI + user code; `OpenBrowser` called when injected (`oauth_device.go:96-112`). `TestDeviceClient_OpenBrowserCalled` passes. `browser.go` GOOS switch (darwin/`open`, linux/`xdg-open`+DISPLAY guard, windows/`rundll32`); `execCommand` injectable. `TestOpenBrowserOS_DISPLAYGuard_Linux` passes.
- [FAIL] REQ-ODF "Auto-copy user code" (SHALL): `DeviceClient.Clipboard` is injectable but **`cmd/login.go:77-81` does not inject it**; `atotto/clipboard` is **never imported** anywhere in the codebase (grep confirms). `Clipboard` nil → copy skipped (`oauth_device.go:101-105`). Scenario "the user code SHALL be copied to the clipboard via atotto/clipboard" **unmet in production**. `TestDeviceClient_ClipboardCalled` passes only with an injected fake. Design said "default clipboard.WriteAll"; impl made it nil-skip and wiring omits it.
- [PASS] REQ-ODF headless fallback: `openBrowserOS` returns error when `DISPLAY=""` on Linux; `RequestToken` prints URL + code regardless. `TestOpenBrowserOS_DISPLAYGuard_Linux` (display_unset → error "display").
- [PASS] REQ-ODF token polling: polls `/login/oauth/access_token` at `interval`; handles `authorization_pending` (continue + "Waiting for authorization…"), `slow_down` (interval+=5), `expired_token` (error), `access_denied` (error); deadline from `expires_in` (`oauth_device.go:114-165`). `TestDeviceClient_PollingStates` covers success/pending/slow_down/expired/denied.
- [PASS] REQ-ODF token storage: `LoginAction.validateAndSave` validates via `cloud.ValidateToken` then `ConfigSaver.Set("github.token", token)` (`login.go:113-129`). `TestLoginAction_OAuthDispatch_SuccessCases` verifies token saved under `github.token`.
- [WARN] "Code expired" / "Authorization denied" scenarios say "display 'Code expired. Run bak login again.'" / "'Authorization denied.'". Impl returns errors with those messages embedded; caller prints the error. Acceptable but the friendly text is in the error, not printed to `out` before returning.

### Capability: cloud-sync (delta) — PASS WITH WARNINGS
- [PASS] REQ-CS OAuth token support: OAuth tokens stored under same `github.token` key; `cloud.ValidateToken` reused; push/pull use `cloud.ResolveToken` unchanged → both token types work.
- [PASS] REQ-CS Login dispatch: `LoginAction.Run` dispatches to OAuth when `OAuthClient != nil`, else manual PAT (`login.go:80-98`). `cmd/login.go:76-82` sets `OAuthClient` only when `BAK_GITHUB_OAUTH_CLIENT_ID` env set. `TestLoginAction_OAuthDispatch_SuccessCases` (oauth_success, pat_fallback_when_no_oauth) + `TestLoginAction_OAuthDispatch_Errors` pass.
- [PASS] REQ-CS GitHub Gist sync (MODIFIED): both PAT and OAuth tokens supported (same storage path). No regression to PAT flow.
- [WARN] No integration test verifies `cmd/login.go` wires OAuth based on the env var (action-layer dispatch is tested; cmd wiring is present in source but not asserted by test).

---

## Quality Gates

- [PASS] `go test -race ./...` — exit 0, all 28 packages pass.
- [PASS] `go vet ./...` — exit 0, clean.
- [**FAIL**] `golangci-lint run` — **exit 1**, 4 issues:
  - `internal/tui/screens/profiles.go:193` — gocritic ifElseChain (rewrite if-else to switch)
  - `internal/tui/screens/restore.go:226` — gocritic ifElseChain
  - `internal/actions/login_test.go:61` — goimports (not properly formatted)
  - `internal/cloud/oauth_device_test.go:136` — goimports
  - Proposal success criterion "`golangci-lint run` exits 0" **not met**. Tasks 1.7.1/2.5.1/3.5.1/4.4.1 claim "all clean" incorrectly.
- Coverage: not separately measured; unit/integration tests are extensive and per-package suites pass with `-race`. E2E disabled per `config.yaml` (out of scope per proposal).

---

## Evidence

**Runtime commands:**
```
$ go vet ./...
=== VET EXIT: 0 ===

$ go test -race -count=1 ./...
ok  github.com/danielxxomg/bak-cli/cmd    19.702s
ok  github.com/danielxxomg/bak-cli/internal/actions    1.945s
ok  github.com/danielxxomg/bak-cli/internal/adapters   1.033s
… (all 28 packages ok)
=== TEST EXIT: 0 ===

$ golangci-lint run
internal/tui/screens/profiles.go:193:2: ifElseChain: rewrite if-else to switch statement (gocritic)
internal/tui/screens/restore.go:226:2: ifElseChain: rewrite if-else to switch statement (gocritic)
internal/actions/login_test.go:61:1: File is not properly formatted (goimports)
internal/cloud/oauth_device_test.go:136:1: File is not properly formatted (goimports)
4 issues:
* gocritic: 2
* goimports: 2
=== LINT EXIT: 1 ===
```

**Source-inspection evidence (key):**
- `grep -rn "SetScanOptions|LoadExcludes"` → only definitions (generic.go, adapter.go, opencode/adapter.go, 7 delegating adapters) + tests (adapter_test.go, ignore_test.go). **No caller in `internal/backup/engine.go`, `internal/actions/backup.go`, `internal/actions/restore.go`, or `cmd/root.go`.** Exclusion engine disconnected from production backup path.
- `grep "atotto/clipboard|clipboard.WriteAll|Clipboard:"` → only `oauth_device.go` comment + two test fakes. **`cmd/login.go` never sets `Clipboard`; `atotto/clipboard` never imported.**
- `cmd/root.go:218-223` `tuiRunRestore` returns hardcoded strings, no `actions.RestoreAction` call.
- `cmd/root.go:334-339` `tuiRunWizard` returns hardcoded `ProfileInfo{Name:"default",…}`, no wizard launch.
- `internal/tui/model.go:213` uses `screens.NewSettingsModel(m.deps.SaveSetting)` (defaults), not `NewSettingsModelWithSettings`; no load dep.
- `internal/config/config.go:123-168` `Load`/`LoadPath` never call `DefaultSettings()`; `config_test.go:771` asserts `DefaultPreset == ""` for missing settings (contradicts spec).
- `internal/tui/screens/welcome.go:23-46` `RenderWelcome` has no logo call and no "Pack your AI coding setup. Move anywhere." tagline.
- `internal/tui/screens/restore.go` done state renders inline message; no `actionResultMsg`/toast; requires keypress to return.

---

## CRITICAL Issues (block archive readiness)

1. **Exclusion engine disconnected from backup path** (PR3). `Engine.Run` and `BackupAction.Run` never call `config.LoadExcludes`/`SetScanOptions`. Adapters run with zero-value `ScanOpts`. → backup-exclude-rules (default patterns, custom ignore, max size, opt-out) and backup-engine "apply exclusion rules before copying" unmet in production; proposal "<2 MB" goal unmet. Covering integration tests absent (`TestEngine_Run_FullPreset` has no node_modules/large files).

2. **`tuiRunRestore` is a stub** (PR1, `cmd/root.go:218-223`). Returns hardcoded strings; no real dry-run diff, no real restore. → tui-restore-screen "Preview selected backup", "Diff shows file changes", "User confirms" unmet in production; mandatory dry-run diff faked.

3. **`tuiRunWizard` is a stub** (PR1, `cmd/root.go:334-339`). Returns hardcoded profile; no 5-step wizard. → tui-profiles-screen "Complete wizard"/"Cancel wizard" unmet in production.

4. **Settings round-trip reload not wired** (PR1). `model.go` uses `NewSettingsModel` (defaults); no `LoadSettings` dep. → bak-cli "Settings round-trip" reload half ("relaunching the TUI SHALL show auto_sync enabled") unmet.

5. **Config defaults not applied on `Load`** (PR1). `Load()` returns zero-value Settings; `DefaultSettings()` never called. → bak-cli "Default settings" scenario (`default_preset="quick"`, `max_file_size=1048576`, `confirm_destructive=true`) unmet; test `TestSettings_LoadDefaultsWhenMissing` encodes the wrong behavior. Compounds with #1 (MaxFileSize default cap never applies).

6. **`golangci-lint run` exits 1** (quality gate). 4 issues (2 gocritic, 2 goimports). Proposal success criterion "golangci-lint run exits 0" not met; tasks 1.7.1/2.5.1/3.5.1/4.4.1 mis-marked as clean.

---

## Warnings

7. **OAuth clipboard not wired** (PR4). `atotto/clipboard` never imported; `cmd/login.go` does not inject `Clipboard`. → oauth-device-flow "Auto-copy user code" (SHALL) unmet in production; unit test passes only with injected fake.
8. **Welcome content mismatch** (PR2). `RenderWelcome` lacks ASCII logo and tagline "Pack your AI coding setup. Move anywhere."; prompt wording differs from "Press Enter to get started". → tui-welcome-screen "Welcome renders" scenario unmet.
9. **Restore completion not via toast / no auto-return** (PR1). Done state shows inline feedback and requires keypress to return; no toast. → tui-restore-screen "Successful restore"/"Restore error" toast mechanism unmet (feedback shown inline).
10. **tui delta spec constant not amended** (PR2). Spec says `MinWidth=40,MinHeight=12`; code/tests use 30×15 per design resolution + proposal acceptance. Spec text stale.
11. **Cloud screen missing `p`/`l` push/pull + `bak login` hint** (PR1/PR2). `cloud.go` handleKey only q/esc; no p/l. Proposal success criterion "`p` pushes, `l` pulls" unmet (not in delta specs → proposal gap, not spec violation). "Cloud screen without provider" missing "run `bak login`" instructions.
12. **Restore backup list not sorted by date descending** (PR1). `renderBackupList` uses `listBackups` order; no sort. → tui-restore-screen "Populate from disk" unmet.
13. **`_ =` error discards** (PR1). `profiles.go` (`_ = m.setActive`, `_ = m.deleteProfile`, `_ = m.SaveProfile`), `settings.go` (`_ = m.saveFunc`). Violates AGENTS.md "MUST handle ALL returned errors"; not flagged by golangci (errcheck allows explicit `_ =`).
14. **`package-lock.json` not excluded by `*.lock`** (PR3). Spec scenario names it; pattern only matches `.lock` suffix. Spec internal inconsistency; test asserts non-match.
15. **Deps signature design deviations** (PR1). `RunRestore` simplified to `(string,error)` (no writer/progress channel); `SaveProfile`/`RunWizard` use `any`/`ProfileInfo` vs design's `actions.ProfileCreateFromWizard`; `GetCloudStatus` uses local `CloudStatus` vs `screens.CloudInfo`. Improve decoupling but deviate from design; restore TUI progress bridge dropped as a consequence.

---

## Suggestions (non-blocking)

16. `TestScreenIotaValues` stale — doesn't cover new `ScreenRestore`/`ScreenProfiles`/`ScreenWelcome` enum values (appended at 7/8/9; 0-6 unchanged so test passes).
17. Stale comment `model_test.go:438` ("width < 40") and `dashboard.go`/`logo.go` logo guard `width < 40` (separate from 30×15 `IsTooSmall`).
18. `tuiCloudStatus` uses `token != ""` for `Connected` and hardcodes `LastSync="never"`; design specified `cloud.ValidateToken` and real last-sync.
19. No integration test asserts `cmd/login.go` wires OAuth from `BAK_GITHUB_OAUTH_CLIENT_ID` env (action dispatch tested; cmd wiring present in source only).

---

## Deferred Items (explicitly out of scope per proposal)

- Activity log, side-by-side diff viewer, schedule editor TUI, plugin marketplace, encryption at rest, Codeberg/Forgejo OAuth (deferred to v0.3+).
- E2E test harness disabled in `config.yaml`; not enabled (unit + integration tests only).

---

## Pre-existing / Not-Part-Of-This-Change

- The existing `cmd/restore.go` CLI restore flow (real `actions.RestoreAction`) predates this change; the gap is specifically that the **TUI** `tuiRunRestore` does not reuse it.
- The existing `cmd/wizard.go` `wizardModel` (real 5-step wizard) predates this change; the gap is specifically that the **TUI** `tuiRunWizard` does not launch it.

---

## Final Verdict: **FAIL**

Six CRITICAL issues block archive readiness: the exclusion engine is built but
never connected to the backup path (PR3 core goal unmet); the TUI restore and
wizard flows are wired to hardcoded stubs instead of real actions (PR1 core goal
partially unmet); settings reload and config defaults are not wired (PR1 spec
scenarios unmet); and `golangci-lint` exits non-zero (quality gate failed).
Unit/component tests pass broadly and many capabilities (modal, terminal guard,
toast, help overlay, OAuth device flow, progress callbacks, progress bridge for
backup) are correctly implemented and tested, but the end-to-end spec scenarios
for backup size reduction, real restore/dry-run, real wizard creation, and
settings round-trip are not satisfied in production. Recommend fixing CRITICAL
items 1-6 (and ideally warnings 7-9) before re-running verification and archiving.
