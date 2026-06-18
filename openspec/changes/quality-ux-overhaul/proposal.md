# Proposal: Quality & UX Overhaul

## Intent

The TUI ships 8 screens and a Rose Pine theme, but the core flows are dead ends: "Create backup" navigates to a progress screen that never animates because `Deps.RunBackup` is never injected (`cmd/root.go:34-38`, `model.go:323-325`); "Restore" and "Profiles" show "coming soon" toasts (`model.go:326-335`); "Settings" has 4 toggles that do nothing (`settings.go:31-40`); the cloud screen always shows empty state (`model.go:369` invokes `RenderCloudStatus` with `CloudInfo{}`). On the data side, full-preset backups balloon to 8.4 MB because `internal/adapters/generic.go:160-198` and `internal/adapters/opencode/adapter.go:129-169` walk all files with zero exclusions. Login is a manual PAT paste with no browser flow. This change unblocks the product end-to-end across 4 chained PRs.

## Scope

### In Scope
- **Phase 1 — Unblock core:** wire `Deps.RunBackup` + `Deps.RunRestore`; restore-picker screen; real `Profiles` screen reusing `cmd/wizard.go`; real `Settings` backed by config; live `Cloud` screen with push/pull.
- **Phase 2 — UX polish:** fix toast positioning (`components/toast.go:67-71`); centralize terminal guards (`styles/styles.go:78-82`, 6 re-implementations); activate `Welcome` (`screens/welcome.go`); restore-from-dashboard row; `?` help overlay on every screen; modal for destructive confirmations; footer with version + profile + cloud status.
- **Phase 3 — Size & progress:** default exclude list (`node_modules`, `.git`, `*.lock`, `*.log`, binaries > 1 MB); `~/.config/bak/ignore` gitignore syntax; `MaxFileSize` cap; `progressFn` callback on `Engine`, `BackupAction`, `RestoreAction`, push, pull; bridge `chan<- ProgressUpdate` → `ProgressStepMsg`.
- **Phase 4 — OAuth:** RFC 8628 Device Flow for GitHub (~150 lines, stdlib only); auto-open browser via `os/exec`; auto-copy `user_code` (`atotto/clipboard` already a dep); graceful manual-paste fallback when `DISPLAY` is absent.

### Out of Scope
- Activity log, side-by-side diff viewer, schedule editor TUI, plugin marketplace, encryption at rest, Codeberg/Forgejo OAuth (deferred to v0.3+).
- E2E test harness is disabled in `config.yaml` — we will not enable it; unit + integration tests only.

## Capabilities

### New Capabilities
- `tui-restore-screen`: restore picker with backup list, dry-run preview, confirm modal
- `tui-profiles-screen`: list/create/show/delete profiles, reuses `wizardModel`
- `tui-welcome-screen`: first-run detection via `Deps.ConfigExists`, single-screen onboarding
- `tui-modal`: reusable confirmation dialog component
- `backup-exclude-rules`: default + `~/.config/bak/ignore` merging, `MaxFileSize` cap
- `progress-reporting`: `progressFn func(string)` on engine + actions; TUI bridge
- `oauth-device-flow`: RFC 8628 client (device code, user code, polling, browser open, clipboard)

### Modified Capabilities
- `tui`: `Deps` gains `RunRestore`, `ListProfiles`, `GetCloudStatus`, `SaveSetting`; `handleMenuEnter` cases 0/1/4 dispatch to real flows
- `bak-cli`: `cmd/root.go` injects all new deps; real Settings schema (`default_preset`, `auto_sync`, `exclude_patterns`, `max_file_size`, `confirm_destructive`, `verbose_default`, `default_provider`)
- `backup-engine`: `Engine.Run` accepts `progressFn`; `generic.go:scanDir` accepts `ScanOptions{Excludes, MaxFileSize}`; OpenCode adapter forwards
- `cloud-sync`: `actions.LoginAction` dispatches OAuth vs manual paste; `cloud.ValidateToken` reused post-OAuth

## Approach

Four chained PRs, each ≤600 lines, each independently shippable and revertable:

1. **PR1 — Phase 1 (Unblock core, ~600 lines):** wire deps in `cmd/root.go`, refactor `handleMenuEnter` cases 0/1/4, real Settings + Cloud screens, restore-picker, profiles screen wrapping `wizardModel`. TDD for `SettingsModel` load/modify/save; `handleMenuEnter` driving `RunBackup` with mock `progressFn`; `ProfilesModel` table rendering.
2. **PR2 — Phase 2 (UX polish, ~300 lines):** `styles.IsTooSmall(w,h) bool` + new constants; toast repositioning via `lipgloss.Place`; activate Welcome; `?` help overlay; modal; footer. Test matrix at 30×15 / 40×12 / 80×24; toast renders bottom-right on narrow and wide.
3. **PR3 — Phase 3 (Size & progress, ~400 lines):** `internal/config/ignore.go` (gitignore parser, ~120 lines); `ScanOptions` plumbing; `progressFn` callback on engine + actions; TUI bridge goroutine. RED for ignore parsing, RED for exclude behavior, RED for progress callback ordering.
4. **PR4 — Phase 4 (OAuth, ~400 lines):** new `internal/cloud/oauth_device.go`; refactor `actions.LoginAction`; browser open helper; clipboard integration. Mock OAuth server, polling state machine, `DISPLAY`-less fallback.

Phase 1 MUST be first — phases 2–4 build on real working flows. Phases 2–4 are independent of each other and can land in any order after PR1.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/root.go:29-44` | Modified | Inject `RunRestore`, `ListProfiles`, `GetCloudStatus`, `SaveSetting` |
| `internal/tui/deps.go:8-22` | Modified | 4 new function fields; `ProgressUpdate` already exists |
| `internal/tui/model.go:103,323-342,351-355` | Modified | `handleMenuEnter` cases 0/1/4 dispatch real flows; use `styles.IsTooSmall`; "too small" with `q`-quittable |
| `internal/tui/screens/{settings,cloud,welcome,dashboard,health,progress}.go` | Modified | Real Settings/Cloud, activate Welcome, `?` overlay, footer |
| `internal/tui/components/{toast,help,modal}.go` | Modified/New | Repositioned toast, reusable help, new modal |
| `internal/tui/styles/styles.go:78-82` | Modified | `IsTooSmall(w,h) bool`; constants `MinWidth=30, MinHeight=15` |
| `internal/tui/screens/{restore,profiles}.go` | New | `RestoreModel` picker + dry-run; `ProfilesModel` table + create/delete |
| `internal/backup/engine.go:52-230` | Modified | Add `progressFn func(string)` optional field |
| `internal/actions/{backup,restore,login}.go` | Modified | `progressFn` on backup/restore; dispatch OAuth vs manual in login |
| `internal/adapters/{generic,opencode/adapter}.go` | Modified | `ScanOptions{Excludes, MaxFileSize}` |
| `internal/config/config.go` | Modified | Settings struct (7 fields) + Load/Save |
| `internal/config/ignore.go` | New | Gitignore parser + merge with defaults |
| `internal/cloud/oauth_device.go` | New | RFC 8628 client, ~150 lines |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| `progressFn` callback refactor breaks 200+ existing tests | High | Make callback optional (nil-safe); existing callers unaffected; new test suite for ordering |
| Default excludes break users who legitimately back up `node_modules` | Med | Config opt-out (`exclude_patterns` array, user can clear defaults); documented in `bak init` |
| OAuth requires GitHub OAuth App registration | Med | Document prerequisite in PR4; ship PR4 with `--oauth-client-id` flag; manual-paste fallback always works |
| Browser auto-open on headless Linux (`DISPLAY=""`) | Med | Detect `DISPLAY`; fall back to printing URL with `atotto/clipboard` copy |
| Toast repositioning breaks narrow terminal layout | Low | Test matrix at 30×15 and 40×12; fallback to inline render when `m.width < 50` |
| Real Settings round-trip corrupts `config.json` on concurrent writes | Low | Use `Config.Load`/`Config.Save` mutex pattern already in `internal/config/config.go`; table-driven test for save/load/reload |

## Rollback Plan

Each chained PR is independently revertable. PR1 (unblock core) is the highest-risk because it touches `Deps` injection — revert restores the previous dead-end behavior, no data loss. PR2 (UX polish) is purely additive cosmetic. PR3 (size + progress) reverts cleanly: `ScanOptions` is optional with zero-value fall-through to current behavior; `progressFn` is nil-safe. PR4 (OAuth) is opt-in: `LoginAction` falls back to manual paste when `OAuthClientID == ""` env var, so the feature ships disabled and is enabled post-rollout.

If a single PR fails verification, archive that change with `archive-report.md` documenting the failure and reopen in a new `quality-ux-overhaul-phaseN` change.

## Dependencies

- **GitHub OAuth App** must be registered before PR4 merges. Client ID read from `BAK_GITHUB_OAUTH_CLIENT_ID` env var or config; manual-paste remains the default until the env var is set.
- **`atotto/clipboard`** already a transitive dep (`go.mod:21`) — no new module for clipboard.
- **Browser-opening** uses `os/exec` stdlib only — no new dep.
- **Gitignore parser** hand-rolled in `internal/config/ignore.go` (~120 lines) to honor `AGENTS.md` rule "MUST prefer Go standard library over third-party packages".
- **OAuth HTTP** hand-rolled in `internal/cloud/oauth_device.go` (~150 lines) using `net/http` + `encoding/json` — consistent with existing `internal/cloud/github_gist.go` pattern.

## Success Criteria

- [ ] `bak` TUI: Enter on "Create backup" produces a real backup with animated progress and `Done` toast
- [ ] `bak` TUI: Enter on "Restore" opens a picker; selecting a row shows dry-run diff; confirming writes a restore log and auto-commits
- [ ] `bak` TUI: "Profiles" lists existing profiles; `n` opens the 5-step wizard; created profile appears in list
- [ ] `bak` TUI: toggling "Auto-sync" in Settings persists to `config.json`; relaunching TUI shows the saved value
- [ ] `bak` TUI: "Cloud sync" shows the configured provider name and token-validity status; `p` pushes, `l` pulls
- [ ] `bak` TUI: first run (no `config.json`) shows Welcome screen; Enter navigates to main menu
- [ ] `bak backup --preset full` on a fixture with 50 MB `node_modules` produces a backup < 2 MB
- [ ] `~/.config/bak/ignore` with `*.tmp` excludes matching files; reload picks up edits
- [ ] `bak login` opens browser to `github.com/login/device`; user pastes the auto-copied code; token validates and saves without manual copy
- [ ] `bak login` on a headless server (no `DISPLAY`) prints the URL and falls back to manual paste
- [ ] Terminal 30×15 shows the TUI; 20×10 shows "Terminal too small" with working `q` to quit
- [ ] Toasts render at bottom-right with a border, not appended to the last content line
- [ ] All existing tests pass; new code ≥80% coverage (`openspec/config.yaml` threshold)
- [ ] `golangci-lint run` exits 0; `go test -race ./...` exits 0 on Linux, macOS, Windows CI
