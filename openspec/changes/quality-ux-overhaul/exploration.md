# Exploration: bak-cli Quality & UX Gaps

**Status:** investigation complete
**Scope:** Gap 1 (Command/TUI audit), Gap 2 (UX/UI), Gap 3 (Login), Gap 4 (Profiles/Settings)
**Project:** bak-cli v0.1.0
**Date:** 2026-06-17

---

## Executive Summary

bak-cli's recent SDD cycles (`tui-overhaul`, `tui-wiring-gaps`, `tui-ux-fixes`) shipped a lot of UI surface area: 8 TUI screens, a Rose Pine theme, a toast system, a search component, and a wizard. But the wiring is incomplete in 4 critical places that the user is hitting in real use:

1. **The "Create backup" TUI flow is a dead end** — `cmd/root.go` never injects `Deps.RunBackup`, the progress screen is reached but never receives any `ProgressStepMsg`, and `RouteSelection` (which COULD call RunBackup) only runs after the TUI exits, not when navigating to the progress screen.
2. **The TUI shows 4 hardcoded placeholders for "Restore", "Profiles", "Settings", and the cloud screen** — they don't work end-to-end.
3. **Backup size bloat (8.4MB) is a real root cause** — the OpenCode adapter walks `skills/`, `commands/`, `plugins/`, `agent/` recursively with zero exclusions (no `.gitignore`, no max-size, no extension filter).
4. **No OAuth and no browser** — the login flow is "paste a GitHub PAT"; the user has to do everything manually.

This exploration details each gap with line numbers and concrete remediation paths so the orchestrator can drive a chained-PR cycle.

---

## 1. Command Audit

### Trace methodology
I read every cobra command, every TUI screen, the dispatch layer, and the action layer to map what actually happens when a user invokes a feature.

### `bak` (no args → TUI) — `cmd/root.go:29-44`

| Aspect | Status | Evidence |
|--------|--------|----------|
| Launch TUI when no args + TTY | ✅ working | `isTTY()` check on `cmd/root.go:33` |
| Launch TUI when no args + non-TTY | ✅ falls through to cobra help | `cmd/root.go:42` |
| Inject `RunBackup` into `Deps` | ❌ **MISSING** | `cmd/root.go:34-38` only sets `Version`, `ConfigExists`, `ListBackups` |
| Inject `RunRestore` into `Deps` | ❌ MISSING | `tui.Deps` has no `RunRestore` field at all (`internal/tui/deps.go:8-22`) |
| Inject `OpenWelcomeScreen` | ⚠️ exists but not called | `screens.ShouldShowWelcome` is in code but never invoked from `model.go` |

**Critical wiring gap:** `Deps.RunBackup` is declared (`internal/tui/deps.go:17`) and `RouteSelection` would call it for cursor 0 (`internal/tui/dispatch.go:18-20`) — but `RouteSelection` only runs **after** `tea.Quit`. By that point, the user is exiting the TUI to run a single shell command. The progress screen reached via `screenChangeMsg{ScreenProgress}` is decoupled from `RunBackup` entirely.

### Main menu: "Create backup" — `internal/tui/model.go:323-325`

| Aspect | Status | Evidence |
|--------|--------|----------|
| Menu shows "Create backup" | ✅ working | `DefaultMenuItems[0]` in `internal/tui/deps.go:52` |
| Enter key on cursor 0 | ❌ **BUG** — goes to progress screen with no work happening | `model.go:323-325` returns `screenChangeMsg{screen: ScreenProgress}` instead of starting a backup |
| Backup actually runs | ❌ NO | `model.go` has no `RunBackup` invocation path; only `dispatch.go:18-20` does, and only post-`Quit` |
| Spinner animates | ❌ NO | `m.running` defaults to `false`; spinner tick is blocked (`screens/progress.go:108-110`) until a `ProgressStepMsg` arrives, which never happens |
| Progress bar updates | ❌ NO | Same root cause as above |
| User can quit with 'q' | ✅ working when `m.running=false` | `screens/progress.go:103-104` — but feels "stuck" because nothing animates |
| `ProgressDoneMsg` ever fired | ❌ NO in production | The engine has no progress emission at all (see Gap 2.2) |

**Root cause:** The TUI navigates to the progress screen but never starts the backup. There is no `tea.Cmd` returned that calls `RunBackup` and emits progress messages. Two-line fix needs: (a) wire `RunBackup` in `cmd/root.go:34-38`, (b) refactor `handleMenuEnter` case 0 to return a `tea.Cmd` that runs the backup and pipes `ProgressStepMsg` updates.

### Main menu: "Restore" — `internal/tui/model.go:326-328`

| Aspect | Status | Evidence |
|--------|--------|----------|
| Menu shows "Restore" | ✅ working | `DefaultMenuItems[1]` |
| Enter key on cursor 1 | ❌ **STUB** — shows "Restore: coming soon" toast | `model.go:326-328` — explicit placeholder |
| `RunRestore` in `Deps` | ❌ field doesn't exist | `internal/tui/deps.go:8-22` has no `RunRestore` |
| CLI restore works | ✅ partially — `cmd/restore.go` is functional | But requires knowing backup ID, no TUI integration |

**Effort to fix:** medium. Need to add `Deps.RunRestore`, refactor `handleMenuEnter` case 1, and add a restore-picker flow (probably reusing `screens.RenderCloudStatus` style or building on the dashboard's selected row).

### Main menu: "Browse backups" — `internal/tui/model.go:329-330`, `internal/tui/screens/dashboard.go`

| Aspect | Status | Evidence |
|--------|--------|----------|
| Menu shows "Browse backups" | ✅ working | `DefaultMenuItems[2]` |
| Enter navigates to Dashboard | ✅ working | `model.go:330` returns `screenChangeMsg{screen: ScreenDashboard}` |
| List populates from disk | ✅ working | `cmd/root.go:37` injects `ListBackups` → `model.go:423-441` builds `DashboardModel` |
| Table renders | ✅ working | `dashboard.go:30-36` columns, `dashboard.go:65-72` build |
| Search filter (`/` key) | ✅ working | `model.go:277-279` activates, `dashboard.go:121-140` filters |
| Restore selected row from table | ❌ **MISSING** | `dashboard.go:91-109` only handles j/k/q/esc, not enter |
| Delete backup from table | ❌ **MISSING** | No handler at all |
| Show backup details (manifest, files) | ❌ **MISSING** | "hello-world" — basic ID/Date/Size/Status/Cloud table only |
| Status field meaningful | ❌ always "ok" | `cmd/root.go:137` hardcodes `Status: "ok"` regardless of validation |
| Cloud column meaningful | ❌ just first adapter name | `cmd/root.go:127-131` picks first adapter name as "cloud provider hint" — misleading |

**Effort to improve:** medium. Restore-from-row, delete-from-row, and a details view would all be additive. Status field needs to come from manifest validation. Cloud column should show actual cloud provider (github-gist/codeberg/etc) not the adapter name.

### Main menu: "Cloud sync" — `internal/tui/model.go:331-332`, `internal/tui/screens/cloud.go`

| Aspect | Status | Evidence |
|--------|--------|----------|
| Menu shows "Cloud sync" | ✅ working | `DefaultMenuItems[3]` |
| Enter navigates to Cloud screen | ✅ working | `model.go:332` |
| Cloud status renders | ❌ **STUB** — always empty | `model.go:369` calls `RenderCloudStatus(screens.CloudInfo{}, m.width)` — passes empty struct |
| Real provider data | ❌ NEVER QUERIED | No code path loads config and populates `CloudInfo` |
| Push from cloud screen | ❌ NO | No `tea.Cmd` to trigger push |
| Pull from cloud screen | ❌ NO | Same |
| Auto-refresh on data change | ❌ NO | No tick/refresh logic |

**Critical:** The cloud screen is a dead shell. The user sees "No cloud provider configured" forever even when they HAVE configured one. Effort to fix: medium — add a CloudModel sub-model similar to SettingsModel that loads config in `Init()` and exposes push/pull actions.

### Main menu: "Profiles" — `internal/tui/model.go:333-335`

| Aspect | Status | Evidence |
|--------|--------|----------|
| Menu shows "Profiles" | ✅ working | `DefaultMenuItems[4]` |
| Enter key on cursor 4 | ❌ **STUB** — "Profiles: coming soon" toast | `model.go:333-335` — explicit placeholder |
| CLI profile commands | ✅ working | `cmd/profile.go:13-188` has `create/list/show/delete/interactive` |
| Profile wizard (CLI) | ✅ working | `cmd/wizard.go:1-317` (the 5-step `wizardModel`) |
| Profile TUI screen | ❌ **MISSING** | No `ScreenProfiles` in `model.go:17-32` |

**Effort to fix:** medium-large. The CLI commands are mature. Need a `ProfilesModel` sub-model + `ScreenProfiles` enum + wiring in `handleMenuEnter` case 4. Could reuse the existing `wizardModel` from `cmd/wizard.go` for the create flow.

### Main menu: "Settings" — `internal/tui/model.go:336-337`, `internal/tui/screens/settings.go`

| Aspect | Status | Evidence |
|--------|--------|----------|
| Menu shows "Settings" | ✅ working | `DefaultMenuItems[5]` |
| Enter navigates to Settings | ✅ working | `model.go:337` |
| Settings render | ✅ working (visually) | `settings.go:31-40` shows 4 hardcoded options |
| Cloud Provider toggle | ❌ **NO-OP** | Toggling changes `opt.Value` in memory only; no config write |
| Theme select | ❌ **NO-OP** | Only Rose Pine theme exists (`styles/theme.go`); toggle has no effect |
| Auto-sync toggle | ❌ **NO-OP** | No `auto_sync` config field, no behavior wired |
| Verbose toggle | ❌ **NO-OP** | `verbose` is a CLI flag only (`cmd/root.go:16`); no config persistence |
| Settings persist across runs | ❌ NO | `SettingsModel` is lazy-initialized on each visit (`model.go:154-159`) and recreated with defaults (`settings.go:31-40`) |

**Why the user said "son una mierda":** Every option is a checkbox that does NOTHING. Toggling has zero effect on the running app or future runs. This is a classic "checkbox theater" UX failure.

**Effort to fix:** medium. Need: (a) real settings backed by config (which fields? suggest: `auto_sync`, `default_preset`, `default_provider`, `theme` (when more than one exists), `verbose_default`), (b) SettingsModel gets/loads config in `Init()` and writes on enter, (c) tests for the load/modify/save round-trip.

### Main menu: "Quit" — `internal/tui/model.go:338-339`

✅ working. Returns `tea.Quit` cleanly.

### Dead/dropped components

| Component | File | Status |
|-----------|------|--------|
| `HealthModel` | `internal/tui/screens/health.go` | Wired into model but uses `tea.Tick` with hardcoded "OK" results (`health.go:115-118`) — not a real diagnostic, just a UI demo. Not in the menu either (only reachable via `ScreenHealth` enum). |
| `Welcome` | `internal/tui/screens/welcome.go` | `ShouldShowWelcome` and `RenderWelcome` exist but are NEVER called from `model.go`. First-run detection dead. |
| Shortcuts keys 1-7 | `internal/tui/screens/shortcuts.go:42-49` | Help screen advertises `1=Menu, 2=Dashboard, ..., 7=Wizard` but `model.go` handleKey has no handlers for digit keys. |
| `tui.Deps` `OpenWelcomeScreen` | (not defined) | Mentioned in proposal docs but never built. |
| `ProfileShow`, `ProfileDelete` from TUI | n/a | Only accessible from CLI. |
| `bak undo` from TUI | n/a | The undo command exists (`cmd/undo.go`) but no TUI entry. |
| `bak verify` from TUI | n/a | `cmd/verify.go` exists, no TUI entry. |
| `bak diff` from TUI | n/a | `cmd/diff.go` exists, no TUI entry. |
| `bak export` from TUI | n/a | `cmd/export.go` exists, no TUI entry. |
| `bak schedule` from TUI | n/a | `cmd/schedule.go` exists, no TUI entry. |

**Pattern:** the project has many "complete" CLI commands but the TUI is a thin shell that only exposes 7 menu items, 2 of which (Create, Cloud sync) are broken.

---

## 2. UX/UI Investigation

### 2.1 Backup size analysis (the 8.4MB question)

**User's complaint:** backups feel excessive.

**What gets included:** When you run `bak backup --preset full`, the OpenCode adapter (`internal/adapters/opencode/adapter.go`) walks ALL of these with no filtering:

| Category | Path under `~/.config/opencode/` | Recursive? | Filter? |
|----------|------------------------------------|------------|---------|
| `config` | top-level files (`opencode.jsonc`, `AGENTS.md`, `tui.json`, `mcp.json`, etc.) | no | only files in `rootConfigFiles` map (line 71-80) — actually quite tight |
| `mcp` | top-level `mcp.json` | no | only the root file |
| `skills` | `skills/` directory | **YES** | **NONE** |
| `commands` | `commands/` directory | **YES** | **NONE** |
| `plugins` | `plugins/` directory | **YES** | **NONE** |
| `agents` | `agent/` directory | **YES** | **NONE** |

**Code evidence:** `internal/adapters/opencode/adapter.go:129-169`:
```go
func scanDir(dir, category, configDir, homeDir string) ([]adapters.Item, error) {
    var items []adapters.Item
    err := filepath.WalkDir(dir, func(absPath string, d fs.DirEntry, err error) error {
        ...
        if !d.IsDir() {
            hash, sz, hashErr := adapters.FileHash(absPath)  // <-- hashes EVERY file
            ...
        }
        items = append(items, item)
        return nil
    })
    return items, err
}
```

No skip list. No size cap. No extension filter. If a user puts a 5MB file under `~/.config/opencode/skills/foo.bin`, it gets included.

**Same pattern in all other adapters** — `internal/adapters/generic.go:160-198` (used by claudecode, codex, cursor, kilocode, kiro, pidev, windsurf):
```go
func scanDir(dir, category, configDir string) ([]Item, error) {
    ...
    err := filepath.WalkDir(dir, func(absPath string, d fs.DirEntry, err error) error {
        ...
        if !d.IsDir() {
            hash, sz, hashErr := FileHash(absPath)
            ...
        }
        items = append(items, item)
    })
    return items, err
}
```

**No `MaxSize` config:** Verified by `grep -rn "MaxSize\|max-size\|max_size" --include="*.go"` → 0 matches. The only related concept is `1 MB max line` in `internal/backup/secrets.go:57` for the secret scanner, which is unrelated.

**No gitignore / no `.bakignore`:** Verified by `grep -rn "bakignore\|\.bakignore\|exclude\." --include="*.go"` → 0 matches.

**Expected vs actual size:**
- **Quick preset** (`CatConfig` only): 5-10 files at the top level. Should be ~5-50 KB. ✅
- **Skills preset** (just `CatSkills`): depends on how many skills + how large each. A single OpenCode skill is usually 5-30 KB of markdown, so 50 skills = 1-1.5 MB. Plausible.
- **Full preset** (all categories): skills + commands + plugins + agents + config. If user has a few plugins with bundled code or large assets, 8 MB is realistic. **The 8.4MB is probably full preset with non-trivial plugins/agents dirs.**

**Likely culprits** (in priority order):
1. **`plugins/` directory** — OpenCode plugins are user-installed and can include `node_modules`, binaries, or large assets.
2. **`agent/` directory** — same risk profile as plugins.
3. **`.git/` directories inside any skill/command** — git clones stored locally.
4. **`node_modules/`** if a user runs `npm install` inside a plugin.

**Recommendation:** add a default exclude list in `internal/adapters/generic.go:scanDir`:

| Pattern | Reason |
|---------|--------|
| `node_modules` | never should be backed up |
| `.git` | large, recreatable |
| `*.lock` | recreatable |
| `*.log` | transient |
| Files > 1 MB | cap individual file size |
| Files matching `*.{png,jpg,jpeg,gif,mp4,zip,tar,gz,exe,dll,dylib,so}` | binaries (unless user overrides) |

Make this configurable via a `~/.config/bak/ignore` file (gitignore syntax) that merges with defaults. Effort: medium-large (need to design ignore syntax, parser, and tests).

### 2.2 Progress reporting during backup/restore

**TUI side:** `internal/tui/screens/progress.go` is fully built and has:
- `spinner` (braille animation)
- `progress` bar (bubble v2)
- `ProgressStepMsg{Step, Current, Total}` for per-step updates
- `ProgressDoneMsg{}` for completion

**Engine side:** `internal/backup/engine.go:52-230` has NO progress emission:
- `Run()` is a single function that does detect → listItems → backup → scanSecrets → writeManifest
- No callbacks, no channels, no progress events
- Same for `internal/actions/backup.go:49-229`

**Same in restore:** `internal/actions/restore.go:48-159` and `internal/restore/dryrun.go:52-100` are blocking, no progress.

**Same in cloud push/pull:** `internal/cloud/github_gist.go:48-100` blocks on the HTTP call.

**Verification:** `grep -rn "ProgressUpdate\|ProgressStepMsg" --include="*.go"` only finds matches in the TUI itself — no engine emits them.

**Effort to fix:** medium. The pattern is well-established (Engine accepts a `progressFn func(string)` callback like `internal/actions/verify_backup.go:41-47` already does for verify). Add `progressFn` to `backup.Engine`, `RestoreAction`, push/pull actions, and a goroutine wrapper in the TUI deps to bridge `chan<- ProgressUpdate` → `ProgressStepMsg`.

### 2.3 Toast component placement

**`internal/tui/components/toast.go:67-71`:**
```go
func (t Toast) View() string {
    if !t.visible || t.message == "" {
        return ""
    }
    return styles.ToastStyle.Render(t.message)
}
```

The component docstring (`toast.go:17-18`) says "displays a message ... at the bottom-right of the screen." But the View returns just a plain styled string.

**In `model.go:385-387`:**
```go
if toastContent := m.toast.View(); toastContent != "" {
    content += "\n" + toastContent
}
```

**Reality:** The toast is **appended to the bottom of the content** as a newline-prefixed string, NOT positioned at the bottom-right corner. It's not overlaid, not wrapped, not placed with `lipgloss.Place`. On a narrow terminal, it'll be the last line of whatever content was rendered before. There's no background, no border, no visual emphasis. It looks like a plain "Restore: coming soon" line at the bottom of the menu screen.

**Effort to fix:** small. Use `lipgloss.Place(width, height, lipgloss.Right, lipgloss.Bottom, toastView)` to actually position it. Add a border background.

### 2.4 "Terminal too small" guard

**Constants:** `internal/tui/styles/styles.go:78-82`:
```go
const MinWidth = 40
const MinHeight = 12
```

**Guard:** `model.go:103`:
```go
m.tooSmall = msg.Width < styles.MinWidth || msg.Height < styles.MinHeight
```

**The message rendered** (`model.go:351-355`):
```go
content = fmt.Sprintf(
    "Terminal too small (%dx%d). Need at least %dx%d.",
    m.width, m.height, styles.MinWidth, styles.MinHeight,
)
```

**Problem 1: 12 rows is aggressive.** A split terminal pane (vim with a 50% split) typically gives you 15-20 rows on each side. But 12 still fits. HOWEVER, if the user has tmux with a status bar (1 row) + window list (1 row) + a top bar (1 row), they can easily drop below 12. Also common dev setup: iTerm split + cmd history + autocomplete popup → real estate evaporates.

**Problem 2: the guard is ALL or NOTHING.** `MinHeight = 12` and `MinWidth = 40` are hardcoded globally. If you're 39 wide or 11 tall, the entire TUI shows only the warning. There's no graceful degradation (smaller logo, vertical menu instead of framed, etc.).

**Problem 3: identical guard duplicated in sub-screens.** The same check is re-implemented in `dashboard.go:148`, `settings.go:81`, `health.go:125`, `progress.go:145`, and `cmd/wizard.go:223`. Five places. None use the root model's `tooSmall` state.

**Problem 4: minimums are inconsistent.** `progress.go:145` uses `20×10` (its own minimum), but `styles.MinWidth = 40, MinHeight = 12` (root). The dashboard and others use `styles.MinWidth/MinHeight`. So if you're 35 wide and 15 tall, the progress screen allows but the root guard blocks. Inconsistent UX.

**Recommendation:**
- Bump `MinHeight` to 15 or 18 (still aggressive but less painful).
- Lower `MinWidth` to 30.
- Make the guard progressive: show the TUI as long as you have ≥20 wide and ≥10 tall, with a slimmer layout (no logo, no border).
- Centralize the check in `styles` (already done) and make all sub-screens call a shared helper.
- Allow 'q' to quit even from the "too small" view (it doesn't currently — `model.go:228-243` has KeyQuit only for ScreenMenu, but `tea.Quit` on the root handler is reachable via the global case).

### 2.5 Other UX gaps

| Gap | Where | Impact |
|-----|-------|--------|
| **No version display on sub-screens** | Only the main menu shows `bak vX.Y.Z` (`screens/menu.go:31`) | User loses context when they drill into Settings |
| **No "Are you sure?" prompts** | Restore `cmd/restore.go:110` has confirmation but Settings toggles are instant; delete from dashboard doesn't exist | User can break state |
| **No help overlay from sub-screens** | Shortcuts screen only reachable from main menu (`model.go:240-242`) | User has to go back to menu to see keybindings |
| **No per-screen breadcrumbs** | Every sub-screen just shows its own title | User loses navigation context |
| **No loading state** | ListBackups is called sync in `model.go:422-443`; first paint on dashboard has no spinner | UI flickers on slow disks |
| **Dashboard column "Cloud" is misleading** | `cmd/root.go:127-131` picks first adapter name as cloud hint | "opencode" in Cloud column makes user think it's their cloud provider, not their local adapter |
| **No relative dates** | `cmd/root.go:120-124` formats raw `YYYYMMDD-HHMMSS` | "2 days ago" is friendlier than "20260615-143022" |
| **No backup age warning** | Manifest has `CreatedAt` (`manifest.go:53`) but no comparison logic | User can't see "this is 6 months old, consider re-backup" |

---

## 3. Login Flow Analysis

### 3.1 Current implementation

**Entry:** `bak login` (`cmd/login.go:55-75`) → `actions.LoginAction.Run()` (`internal/actions/login.go:41-98`).

**Flow:**
1. User invokes `bak login`
2. CLI prompts: `Enter GitHub personal access token:`
3. User must open browser, go to github.com/settings/tokens, create a token with `gist` scope, copy it, paste it back
4. CLI calls `cloud.ValidateToken` to verify (HTTP GET to `https://api.github.com/user`)
5. Token saved to `~/.config/bak/config.json` under `github.token`

**Source:** `internal/actions/login.go:69-89`:
```go
// 2. Prompt for token.
_, _ = fmt.Fprint(out, "Enter GitHub personal access token: ")
input, err := reader.ReadString('\n')
...
token := strings.TrimSpace(input)

if token == "" {
    return fmt.Errorf("login: token cannot be empty")
}

// 3. Validate token.
_, _ = fmt.Fprint(out, "Validating token... ")
if a.TokenValidator != nil {
    if err := a.TokenValidator(token); err != nil {
        _, _ = fmt.Fprintln(out, "❌")
        return fmt.Errorf("token validation failed: %w", err)
    }
}
_, _ = fmt.Fprintln(out, "✅")
```

**What the user asked:** "is this called OAuth?" — they want a browser-based flow. Answer: **not quite**. There are two common patterns:

| Pattern | What it does | Used by | Effort to implement |
|---------|--------------|---------|---------------------|
| **OAuth Device Flow (RFC 8628)** | CLI requests a `device_code` + `user_code` from the auth server. User visits a URL in their browser, enters the code, authorizes. CLI polls the auth server until the user completes. | GitHub CLI (`gh auth login`), OpenCode, VSCode, Tailscale | medium (need `golang.org/x/oauth2` or hand-rolled HTTP) |
| **OAuth Authorization Code + Local Server** | CLI starts a local HTTP server on a random port, redirects user to auth server, receives callback with code, exchanges for token. | Spotify CLI, AWS SSO, Google Cloud SDK | larger (need HTTP server, callback handling, port management) |
| **OAuth Authorization Code + PKCE (manual paste)** | Like above, but user pastes the redirect URL back into the CLI. | Some CLIs without browser access | smallest (no local server) |
| **PAT paste (current)** | User creates a PAT manually and pastes it. | `bak` today, `docker login` classic | trivial (already done) |

**The OpenCode-style flow the user is describing is OAuth Device Flow.** It is THE standard for CLI tools that need browser auth.

### 3.2 What's needed for Device Flow

**GitHub's Device Flow endpoints:**
- `POST https://github.com/login/device/code` → returns `device_code`, `user_code`, `verification_uri`, `interval`, `expires_in`
- `POST https://github.com/login/oauth/access_token` (poll) → returns `access_token` on success

**Required CLI client ID:** GitHub requires a registered OAuth app. For a CLI without a public client, you can use the special **public client IDs** that GitHub provides for CLI tools (e.g., the GitHub CLI's client ID is published, OpenCode has its own). bak-cli needs to register its own.

**Code shape:**
```go
// 1. Request device code
type deviceCodeResp struct {
    DeviceCode      string `json:"device_code"`
    UserCode        string `json:"user_code"`
    VerificationURI string `json:"verification_uri"`
    Interval        int    `json:"interval"`
    ExpiresIn       int    `json:"expires_in"`
}

// 2. Display to user: "Visit https://github.com/login/device, enter code XXXX-XXXX"
// 3. Open browser automatically (xdg-open / open / rundll32)
// 4. Poll token endpoint with device_code every `interval` seconds
// 5. On success, save token to config
```

**Browser opening (no extra deps needed — stdlib `os/exec`):**
```go
func openBrowser(url string) error {
    var cmd *exec.Cmd
    switch runtime.GOOS {
    case "darwin":  cmd = exec.Command("open", url)
    case "linux":   cmd = exec.Command("xdg-open", url)
    case "windows": cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
    }
    return cmd.Start()
}
```

**Dependencies to add:**
- `golang.org/x/oauth2` — official OAuth2 client. OR hand-roll the HTTP calls (only ~50 lines, simple JSON over POST). Hand-rolling is consistent with project's preference for stdlib over deps (`AGENTS.md` rule).
- `atotto/clipboard` is **already** a transitive dep (`go.mod:21`) — can use to auto-copy the `user_code`.

**Effort:** medium. ~150 lines of new code in `internal/cloud/oauth_device.go`, refactor `internal/actions/login.go` to call it.

### 3.3 What exists to build on

| Asset | Reuse potential |
|-------|-----------------|
| `cloud.ValidateToken` (`internal/cloud/auth.go:80-122`) | Reuse for post-OAuth validation |
| `cloud.ResolveToken` (`internal/cloud/auth.go:63-78`) | Reuse for token resolution precedence |
| `Config.Get`/`Config.Set` | Reuse for storage |
| `LoginAction` (`internal/actions/login.go`) | Refactor to dispatch to OAuth vs manual based on availability |
| `LoginInteractiveAction` (`internal/actions/login_interactive.go`) | Already has TUI wizard for provider selection — extend with new OAuth step |

### 3.4 What about Codeberg / Gitea / Rclone providers?

- **Codeberg** has device flow since 2023 (Forgejo supports it).
- **Gitea/Forgejo** (self-hosted) requires user-supplied `base_url` and the OAuth app has to be registered per-instance. Device flow is supported in 1.20+.
- **Rclone** is not OAuth — it uses local config files or SSH. Different auth model.

For v1, focus on GitHub Device Flow. Other providers can keep PAT-paste until the abstraction stabilizes.

---

## 4. Profiles & Settings

### 4.1 Profiles

**CLI state (works well):**
- `cmd/profile.go` has full implementation: `create`, `list`, `show`, `delete`, and `--interactive` wizard mode
- `actions/profile.go` has the business logic
- `wizardModel` (`cmd/wizard.go`) is a 5-step interactive wizard: Provider → Preset → Adapters → Categories → Confirm
- Reused by `cmd/login.go:77-113` for provider selection in login flow

**TUI state (missing):**
- `DefaultMenuItems[4]` = "Profiles"
- `handleMenuEnter` case 4 → `m.toast.Show("Profiles: coming soon", 3)` (stub)
- No `ScreenProfiles` enum value
- No `ProfilesModel` sub-model
- No way to list/create/edit profiles from the TUI

**What the TUI could/should expose:**
1. **List profiles** (read-only, similar to dashboard table)
2. **Create profile** (reuses `wizardModel` — already a 5-step flow, just needs TUI integration)
3. **Show profile** (detail view of one profile)
4. **Delete profile** (with confirmation)
5. **Select active profile** (which profile is used by default for "Create backup")

**Effort:** medium. The CLI work is done. Needs:
- `ScreenProfiles` enum + `ProfilesModel` sub-model with table
- "Create" sub-screen that wraps `wizardModel`
- Wire `handleMenuEnter` case 4 to navigate to `ScreenProfiles`
- A config field `active_profile` so the user can set which profile is "default"

### 4.2 Settings

**Current state (`internal/tui/screens/settings.go:31-40`):**
```go
options: []SettingsOption{
    {Label: "Cloud Provider", Type: "toggle", Value: false},
    {Label: "Theme", Type: "select", Value: true},
    {Label: "Auto-sync", Type: "toggle", Value: false},
    {Label: "Verbose", Type: "toggle", Value: false},
},
```

**Issues:**
- **No persistence.** Each visit re-creates the model from defaults (`model.go:154-159` lazy init).
- **No behavior.** Toggling has no effect.
- **No real options.** The 4 shown are placeholders; actual user-facing settings should be:
  - `default_preset` (quick/full/skills) — used when no `--preset` flag
  - `default_provider` — used when no `--provider` flag
  - `auto_sync` — after every backup, push to cloud
  - `default_adapters` — restrict which adapters are detected
  - `exclude_patterns` — gitignore-style file exclusions (Gap 2.1)
  - `max_file_size` — cap for individual file (Gap 2.1)
  - `verbose_default` — default `--verbose` flag state
  - `confirm_destructive` — always prompt before restore/delete

**Effort:** medium. Need to:
1. Define a real Settings struct in `internal/config/config.go`
2. Load settings in `SettingsModel.Init()` from config
3. Write changes to config in `Update()` on toggle
4. Replace hardcoded options with real, meaningful settings
5. Add tests for the load/modify/save round-trip

### 4.3 Welcome screen (first-run)

**State:** `internal/tui/screens/welcome.go` exists with `RenderWelcome` and `ShouldShowWelcome` helpers. But:
- `ShouldShowWelcome` is never called from `model.go`
- `RenderWelcome` is never used
- No `ScreenWelcome` enum value

The `Deps.ConfigExists` function is injected in `cmd/root.go:36` but never consumed.

**Effort:** small. Add a `ScreenWelcome` case in `model.go` that triggers when `ConfigExists() == false`, navigate to it on Init, and route to main menu when user presses enter.

---

## 5. UX Polish Opportunities

### 5.1 Existing components inventory

| Component | File | Used in TUI? | Used in real flow? |
|-----------|------|--------------|---------------------|
| `Toast` | `components/toast.go` | ✅ wired | ⚠️ only fires from "coming soon" stubs and `actionResultMsg` (which never fires in prod) |
| `Search` | `components/search.go` | ✅ dashboard | ✅ real |
| `Menu` | `components/menu.go` | ✅ main menu | ✅ real |
| `Checkbox` | `components/checkbox.go` | ✅ settings, wizard | ⚠️ settings is theater |
| `Radio` | `components/radio.go` | ❌ never used in production code (only in tests) | ❌ |
| `Help` | `components/help.go` | ✅ every screen | ✅ real |
| `Frame` | `styles/frame.go` | ✅ some screens | ✅ real |
| `spinner` (bubbles) | progress screen | ✅ progress | ❌ never reaches progress because no backup runs |
| `progress` (bubbles) | progress screen | ✅ progress | ❌ same |
| `table` (bubbles) | dashboard | ✅ dashboard | ✅ real |

### 5.2 Missing UI components

| Component | Purpose | Effort |
|-----------|---------|--------|
| **Aside (sidebar)** | Persistent context panel (current path, current user, current profile) | small — just lipgloss layout |
| **Breadcrumb** | Show navigation trail (Home > Settings > Profiles) | small — helper func |
| **Modal/Dialog** | Confirmation prompts ("Are you sure?"), error dialogs | medium — new sub-model |
| **Toast positioned properly** | Actually bottom-right, with border, auto-dismiss | small — `lipgloss.Place` |
| **Help overlay on any screen** | `?` key shows shortcuts on current screen | small — reuse `RenderShortcuts` |
| **Loading state** | Spinner while `ListBackups` runs | small — wrap in tea.Cmd |
| **Empty-state illustrations** | ASCII art for "no backups yet" screens | small — lipgloss art |
| **Status bar** | Bottom strip with current state, profile, cloud connection | medium — new layout |
| **Notification history** | View past toasts (like a notification center) | medium — new sub-screen |

### 5.3 Terminal guard deep-dive

**Constants:** `MinWidth=40, MinHeight=12` (`styles/styles.go:78-82`)

**Where checked (6 places — should be 1):**
1. `model.go:103` (root)
2. `screens/dashboard.go:148`
3. `screens/settings.go:81`
4. `screens/health.go:125`
5. `screens/progress.go:145` — uses `20×10`, not `40×12`
6. `cmd/wizard.go:223` — uses `20×10`, not `40×12`

**Why annoying:** If the user is in a vertically split tmux pane (very common dev setup) and drops to 11 rows tall, the ENTIRE TUI becomes "Terminal too small (NxM). Need at least 40x12." No fallback layout. The user is locked out.

**Specific issues:**
1. **Height 12 is too aggressive.** A standard terminal is 24 rows. Half-split is 12. With status bars, often less. Bump to 15 or 18.
2. **The progress screen allows 20×10** but the root blocks at 40×12. So if you're 39×15, you can't get to the progress screen even though it would fit.
3. **The "too small" view doesn't accept input.** Pressing 'q' does nothing because `KeyQuit` is only handled for `ScreenMenu` (`model.go:232-233`).

**Recommendations:**
- Centralize in one helper: `styles.IsTooSmall(w, h int) bool` with a single set of constants
- Show graceful degradation: smaller logo, no frame, vertical menu, all sub-screens still render
- Always accept 'q' to quit
- Use a more reasonable default like 30×15

---

## 6. Recommended Priority Order

Based on:
- **Impact** (how many users hit this daily)
- **Effort** (lines changed, tests, review burden)
- **Dependencies** (does it unblock other work?)

### Tier 1 — Fix the broken core (high impact, medium effort)

| # | Change | Effort | Why first |
|---|--------|--------|-----------|
| 1 | **Wire `RunBackup` in `cmd/root.go`** + refactor `handleMenuEnter` case 0 to run backup with progress | medium | "Create backup" is dead. Unblocks 4+ other gaps. |
| 2 | **Wire `RunRestore` + add restore-picker screen** | medium | "Restore" is dead from TUI. |
| 3 | **Add `Profiles` TUI screen** reusing existing `wizardModel` | medium | "Profiles" is dead from TUI. |
| 4 | **Real `Settings` backed by config** with persistence and real effects | medium | Current settings are theater. |
| 5 | **Fix cloud screen to load actual config** + add push/pull actions | medium | Cloud sync screen shows empty data forever. |

### Tier 2 — UX polish (medium impact, small-medium effort)

| # | Change | Effort |
|---|--------|--------|
| 6 | **Position toast properly** with `lipgloss.Place` + border | small |
| 7 | **Lower terminal minimums + graceful degradation** | small |
| 8 | **Activate `Welcome` screen on first run** | small |
| 9 | **Add backup age / relative date in dashboard** | small |
| 10 | **Add "Restore selected" + "Delete selected" from dashboard** | medium |
| 11 | **Wire `?` help overlay on every screen** | small |
| 12 | **Show version + profile + cloud status in a footer** | medium |
| 13 | **Add modal/dialog component for confirmations** | medium |

### Tier 3 — Backup size + progress (high user value, larger effort)

| # | Change | Effort |
|---|--------|--------|
| 14 | **Add default exclude list** in `internal/adapters/generic.go:scanDir` (node_modules, .git, *.lock, *.log, binaries > 1MB) | medium |
| 15 | **Add `~/.config/bak/ignore` gitignore-syntax** with merging | medium-large |
| 16 | **Add progress reporting to backup engine** with `progressFn func(string)` callback | medium |
| 17 | **Bridge `chan<- ProgressUpdate` → `ProgressStepMsg` in TUI** | medium |
| 18 | **Show file-by-file progress** during backup (current/total) | medium |

### Tier 4 — Login overhaul (high value, medium-large effort)

| # | Change | Effort |
|---|--------|--------|
| 19 | **Implement OAuth Device Flow for GitHub** (~150 lines) | medium |
| 20 | **Auto-open browser** via stdlib `os/exec` (macOS/Linux/Windows) | small |
| 21 | **Auto-copy user code to clipboard** (atotto/clipboard is already a dep) | small |
| 22 | **TUI login flow** with progress ("Waiting for browser authorization…") | medium |
| 23 | **Extend OAuth Device Flow to Codeberg/Forgejo** | medium |

### Tier 5 — Larger features (low priority for v0.2.0)

- Activity log / backup history view
- Side-by-side backup diff (using existing `bak diff`)
- Cloud restore (pull from cloud and restore)
- Schedule editor TUI
- Plugin marketplace (already can list plugins via adapter detection)

---

## 7. Architecture Notes

### What works well (don't break this)

- **Adapter pattern with `GenericAdapter`** is excellent (`internal/adapters/generic.go`). Adding new tools is a 5-line file.
- **`Deps` function fields in TUI** for DI (`internal/tui/deps.go`) — clean, no interface boilerplate.
- **Strict TDD with `cmdDeps` pattern** in actions. Every action has a `*WithDeps` variant for testability.
- **Manifest schema versioning + auto-migration** in `internal/config/config.go` is solid.
- **Secret scanning with redaction** in `internal/backup/secrets.go` is well thought out.

### What needs rethinking

- **Two backup engines** (`internal/backup/engine.go` and `internal/actions/backup.go` `BackupAction`). The engine in `backup/` is the older one. The action in `actions/backup.go` duplicates logic. Should consolidate.
- **No progress in business logic** is the biggest gap. Every long-running op (backup, restore, push, pull) is a blocking call with no progress feedback.
- **Two push paths**: `cmd/push.go` → `actions.PushAction` vs `cmd/pull.go` → `actions.PullAction`. These are simpler than the TUI's cloud.Provider interface. Pick one canonical path.
- **TUI components are not yet used in CLI commands** — `internal/tui/components/*.go` is exclusively for the TUI. Some, like `Toast` and `Search`, could be useful in `pick` (the old `cmd/pick.go`) but aren't.

### Wiring gaps to fix in one PR (chained)

1. **RunBackup wiring** (Tier 1, item 1)
2. **Progress emission** (Tier 3, items 16-17)
3. **Toast positioning** (Tier 2, item 6)
4. **Real Settings** (Tier 1, item 4)
5. **Lower terminal minimums** (Tier 2, item 7)

These 5 are all < 400 lines and together fix the "stuck TUI" complaint.

---

## 8. Concrete First PR (chained)

**PR1: TUI unblock — make "Create backup" actually work**
- Files: `cmd/root.go`, `cmd/backup.go`, `internal/tui/model.go`, `internal/tui/dispatch.go`, `internal/tui/deps.go`, `internal/backup/engine.go`, `internal/actions/backup.go`
- Add `progressFn` callback to `backup.Engine` and `actions.BackupAction`
- Add `RunBackup` dep injection in `cmd/root.go`
- Refactor `handleMenuEnter` case 0 to return a `tea.Cmd` that calls `RunBackup` and pipes `ProgressStepMsg`
- Tests: RED for `progressFn` callbacks, GREEN for TUI driving a real backup, integration with `t.TempDir()`

**PR2: TUI completeness — Restore, Profiles, real Settings, real Cloud**
- Files: `internal/tui/model.go`, `internal/tui/screens/*`, `internal/tui/deps.go`, `internal/config/config.go`
- Add `ScreenRestore`, `ScreenProfiles`, `ScreenCloud` real sub-models
- Add `Deps.RunRestore`, `Deps.ListProfiles`, `Deps.GetCloudStatus`
- Settings backed by config with real persistence
- Cloud screen loads real provider status

**PR3: Backup size — exclude patterns + max-size cap**
- Files: `internal/adapters/generic.go`, `internal/adapters/opencode/adapter.go`, new `internal/config/ignore.go`
- Add `scanDirOpts` struct with `ExcludePatterns []string`, `MaxFileSize int64`
- Default exclude list: node_modules, .git, *.lock, *.log
- Read `~/.config/bak/ignore` (gitignore syntax) and merge
- Tests: RED for ignore parsing, RED for exclude behavior, GREEN

**PR4: Login — OAuth Device Flow for GitHub**
- Files: new `internal/cloud/oauth_device.go`, `internal/actions/login.go`, `cmd/login.go`
- Implement RFC 8628 client (hand-rolled, no new deps)
- Auto-open browser (stdlib `os/exec`)
- Auto-copy user code (atotto/clipboard already available)
- Tests with mock OAuth server

---

## 9. Risks

| Risk | Mitigation |
|------|------------|
| Wiring changes break existing TUI tests | All TUI tests live in `internal/tui/*_test.go`; run with `-race` before commit |
| Backup engine refactor breaks 100s of existing tests | Add new `progressFn` field as optional; existing callers unaffected |
| OAuth Device Flow requires GitHub client ID | Register OAuth app first, document in `docs/` |
| Default excludes break users who legitimately back up node_modules | Make exclude list opt-out via config; default to safest |
| Toast positioning breaks narrow terminal layouts | Test with 30×15 minimums, not 80×24 |
| Browser opening on Linux servers (no DISPLAY) | Fall back to printing URL for manual copy; detect `DISPLAY` env |

---

## 10. Ready for Proposal

**Yes.** This exploration provides enough detail to drive:
- `sdd-propose` for PR1 (TUI unblock — highest priority, smallest scope)
- Then chained PRs for PR2-PR4 as separate `sdd-propose` calls

The orchestrator should present this report and ask the user which Tier 1-2 items to prioritize for v0.2.0. Tier 3-4 are v0.3+ candidates.

---

## Appendix: Key files reference

| Path | Purpose |
|------|---------|
| `cmd/root.go:29-44` | TUI launch + Deps injection (gaps here) |
| `cmd/tty.go:13-33` | `runTUI` injection point |
| `internal/tui/model.go` | Root TUI model, 448 lines |
| `internal/tui/model.go:323-342` | `handleMenuEnter` (stubs at case 1 & 4) |
| `internal/tui/dispatch.go:12-24` | `RouteSelection` (only handles cursor 0) |
| `internal/tui/deps.go:8-22` | `Deps` struct (missing `RunRestore`, etc.) |
| `internal/tui/styles/styles.go:78-82` | `MinWidth=40, MinHeight=12` |
| `internal/tui/screens/progress.go:89-141` | `Update` (handles `ProgressStepMsg` that never fires in prod) |
| `internal/tui/components/toast.go:67-71` | `Toast.View` (returns plain string, not positioned) |
| `internal/backup/engine.go:52-230` | `Engine.Run` (no progress callback) |
| `internal/actions/backup.go:49-229` | `BackupAction.Run` (no progress callback) |
| `internal/adapters/opencode/adapter.go:129-169` | `scanDir` (no exclusions) |
| `internal/adapters/generic.go:160-198` | Shared `scanDir` (no exclusions) |
| `internal/actions/login.go:41-98` | `LoginAction.Run` (manual PAT paste) |
| `internal/cloud/auth.go:80-122` | `ValidateToken` (existing OAuth-aware validator) |
| `cmd/profile.go:13-188` | Full CLI profile management (TUI doesn't use) |
| `internal/tui/screens/settings.go:31-40` | Hardcoded settings options (theater) |
| `internal/tui/screens/welcome.go:13-47` | Welcome screen (never called) |
| `cmd/wizard.go:1-317` | 5-step wizard (CLI-only, TUI doesn't reuse) |

## 11. Performance notes

- Backup engine hashes every file (`adapters.FileHash` reads + SHA-256). For 8MB of files this is ~50-200ms on modern hardware. Not a bottleneck.
- TUI package-level styles (good, zero per-frame allocation) — verified in `internal/tui/styles/styles.go` and `screens.go`.
- `spinner.Tick` (every 100ms) is cheap.
- `progress.SetPercent` is cheap.
- No obvious perf issues in the current code path.

## 12. Test coverage assessment

- `cmd/`: high coverage, TDD with `cmdDeps` pattern is well-established.
- `internal/actions/`: high coverage, similar TDD pattern.
- `internal/tui/`: high coverage, pure-function testing of `Update`/`View` works well.
- `internal/backup/`: integration tests in `integration_test.go`.
- `internal/cloud/`: tests with mock HTTP server (`setupMockGistAPI`).
- `internal/adapters/`: per-adapter tests.

The codebase has solid test coverage overall. The bug isn't a coverage gap — it's a wiring gap. Tests verify the components work in isolation, but they don't verify they connect end-to-end (e.g., "does `bak` from a shell actually result in a backup being created?").

**Recommendation:** add E2E tests in `tests/e2e/` for the new wired flows (the config marks E2E as `available: false` — could be enabled).
