# Design: TUI UX Fixes

## 1. Arrow Key Constants

**File**: `internal/tui/keys.go`

Add arrow key constants alongside existing j/k:

```go
const (
    // ... existing constants ...

    // KeyDownArrow navigates down (down arrow, escape sequence).
    KeyDownArrow = "down"
    // KeyUpArrow navigates up (up arrow, escape sequence).
    KeyUpArrow = "up"
)
```

Note: bubbletea v2 `tea.KeyPressMsg.Code` is a `rune` for single chars but arrow keys come as string representations. The handleKey switch must use `msg.Code` for runes (j/k) and `msg.String()` for arrow keys. See gentle-ai `model.go` line 1288 which uses `keyStr := key.String()` and switches on `"up"`, `"down"`, `"j"`, `"k"`.

**Approach**: In each `handleKey`/`Update` that handles `tea.KeyPressMsg`, extract `keyStr := msg.String()` and match both `string(KeyDown)` and `"down"` (and up equivalent). This avoids type mismatch between rune constants and string key names.

## 2. Wrap-Around Navigation

### Main Menu (`internal/tui/model.go` handleKey, ScreenMenu case)

Replace bounded cursor movement with modular arithmetic:

```go
case KeyDown: // 'j'
    m.cursor = (m.cursor + 1) % len(m.menuItems)
case KeyUp: // 'k'
    m.cursor = (m.cursor - 1 + len(m.menuItems)) % len(m.menuItems)
```

Add arrow key handling in the same switch:

```go
// After extracting keyStr := msg.String():
case "down":
    m.cursor = (m.cursor + 1) % len(m.menuItems)
case "up":
    m.cursor = (m.cursor - 1 + len(m.menuItems)) % len(m.menuItems)
```

Reference: gentle-ai `model.go` lines 1299-1306 (up wrap) and 1326-1331 (down wrap). The gentle-ai pattern uses explicit if/else with `isScrollableScreen()` guard. For bak-cli, all menu screens are non-scrollable, so modular arithmetic is simpler and equivalent.

### Settings (`internal/tui/screens/settings.go` Update)

Same modular arithmetic pattern:

```go
case 'j':
    m.cursor = (m.cursor + 1) % len(m.options)
case 'k':
    m.cursor = (m.cursor - 1 + len(m.options)) % len(m.options)
case "down":
    m.cursor = (m.cursor + 1) % len(m.options)
case "up":
    m.cursor = (m.cursor - 1 + len(m.options)) % len(m.options)
```

### Dashboard

No wrap-around. The dashboard uses `bubbles/table` which manages its own cursor. Arrow keys are forwarded to the table sub-model via the existing default forwarding path.

## 3. Help Bar Persistence

**Pattern**: Each screen's `View()` appends `components.RenderHelp(keys)` after its content, using context-appropriate key bindings. The `components.RenderHelp` function already exists and is used by `RenderMainMenu`.

### Settings (`internal/tui/screens/settings.go` View)

```go
// After existing content:
b.WriteString("\n\n")
helpKeys := []components.HelpKey{
    {Key: "↑/↓", Desc: "navigate"},
    {Key: "enter", Desc: "toggle"},
    {Key: "q", Desc: "back"},
}
b.WriteString(components.RenderHelp(helpKeys))
```

### Dashboard (`internal/tui/screens/dashboard.go` View)

Add help bar after ALL exit paths (error, empty, populated):

```go
// Helper to append help bar:
func (m DashboardModel) renderHelp() string {
    helpKeys := []components.HelpKey{
        {Key: "↑/↓", Desc: "navigate"},
        {Key: "/", Desc: "search"},
        {Key: "q", Desc: "back"},
    }
    return components.RenderHelp(helpKeys)
}
```

Call `b.WriteString("\n\n" + m.renderHelp())` before each `return tea.NewView(...)` in View().

### Health (`internal/tui/screens/health.go` View)

Replace inline `"q quit • enter rerun"` text with `components.RenderHelp`:

```go
// Idle state:
helpKeys := []components.HelpKey{
    {Key: "enter", Desc: "run"},
    {Key: "q", Desc: "back"},
}
b.WriteString(components.RenderHelp(helpKeys))

// After checks complete:
helpKeys := []components.HelpKey{
    {Key: "q", Desc: "back"},
    {Key: "enter", Desc: "rerun"},
}
b.WriteString(components.RenderHelp(helpKeys))
```

### Cloud (`internal/tui/screens/cloud.go` RenderCloudStatus)

```go
// At end of function, before return:
helpKeys := []components.HelpKey{
    {Key: "q", Desc: "back"},
}
content += "\n\n" + components.RenderHelp(helpKeys)
```

Note: cloud is rendered from `model.go` View() for `ScreenCloud`, not a sub-model. The help bar is appended to the string returned by `RenderCloudStatus`.

## 4. ListBackups Wiring

**File**: `cmd/root.go` line 31-34

Current code:
```go
deps := tui.Deps{
    Version:      Version,
    ConfigExists: configExists,
}
```

Add `ListBackups`:
```go
deps := tui.Deps{
    Version:      Version,
    ConfigExists: configExists,
    ListBackups:  listBackups,
}
```

New function in `cmd/root.go` (or `cmd/tty.go`):
```go
func listBackups() ([]tui.BackupInfo, error) {
    // Use existing actions.ListBackups or backup.ListManifests
    // Convert to []tui.BackupInfo
    dir, err := config.BackupDir()
    if err != nil {
        return nil, fmt.Errorf("backup dir: %w", err)
    }
    manifests, err := backup.ListManifests(dir)
    if err != nil {
        return nil, fmt.Errorf("list backups: %w", err)
    }
    var result []tui.BackupInfo
    for _, m := range manifests {
        result = append(result, tui.BackupInfo{
            ID:     m.ID,
            Date:   m.CreatedAt.Format("2006-01-02"),
            Size:   formatSize(m.TotalSize),
            Status: "ok",
            Cloud:  m.CloudProvider,
        })
    }
    return result, nil
}
```

The exact implementation depends on what `backup.ListManifests` returns. The key point: `deps.ListBackups` must be non-nil so `model.go` `initDashboard` actually calls it instead of returning `nil, nil`.

**Existing nil guard** (`model.go` line 425-427):
```go
if m.deps.ListBackups == nil {
    return nil, nil
}
```
This stays for test isolation — tests that don't wire ListBackups get empty dashboard.

## 5. Terminal Minimums

**File**: `internal/tui/model.go` lines 48-51

Change from:
```go
const (
    minWidth  = 20
    minHeight = 10
)
```

To:
```go
const (
    minWidth  = 40
    minHeight = 12
)
```

Rationale:
- Width 40: logo is already hidden below 40 cols (`styles.RenderLogo` guards at width<40). 40 cols fits the narrowest meaningful content (menu items + cursor + padding).
- Height 12: menu has 7 items + logo (2 lines) + version (1 line) + help bar (1 line) + spacing (3 lines) = ~14 lines ideal. 12 is the minimum that doesn't feel broken.

**Sub-screen guards** must be updated to match:
- `settings.go` line 85: `m.width < 20 || m.height < 10` → `m.width < minWidth || m.height < minHeight`
- `dashboard.go` line 147: same change
- `health.go` line 124: same change

To avoid duplicating constants, define `MinWidth` and `MinHeight` as exported from `tui` package, or define them in a shared location. Since `screens` package cannot import `tui` (circular), define the constants in `internal/tui/styles/` or duplicate them in `screens` package with a comment referencing the source of truth.

**Recommended**: Add to `internal/tui/styles/constants.go` (or existing file):
```go
const (
    MinWidth  = 40
    MinHeight = 12
)
```

Both `tui/model.go` and `screens/*.go` import `styles`, so this avoids circular imports and keeps a single source of truth.

## File Change Summary

| File | Change |
|------|--------|
| `internal/tui/keys.go` | Add `KeyDownArrow`, `KeyUpArrow` constants |
| `internal/tui/model.go` | Wrap-around in ScreenMenu; arrow key handling; lower minWidth/minHeight |
| `internal/tui/styles/` | Add `MinWidth=40`, `MinHeight=12` constants |
| `internal/tui/screens/settings.go` | Arrow keys + wrap-around; help bar in View; use styles.MinWidth/MinHeight |
| `internal/tui/screens/dashboard.go` | Help bar in View (all paths); use styles.MinWidth/MinHeight |
| `internal/tui/screens/health.go` | Replace inline help with RenderHelp; use styles.MinWidth/MinHeight |
| `internal/tui/screens/cloud.go` | Add help bar via RenderHelp |
| `cmd/root.go` | Wire `ListBackups` into `tui.Deps` |
