# Design: cmd-di-refactor

## Technical Approach

Inject dependencies at the `cmd/` layer using a `CmdDeps` struct passed to internal `runXWithDeps` functions. The cobra `RunE` handlers call these with package-level `defaultDeps`, while tests inject mocks. This isolates the 11 `config.Load()` calls and direct `os.Stdin`/`os.Stdout` usage from the real filesystem.

Additionally, fix `canonicalPath()` in `internal/diff/diff.go` to use `strings.ReplaceAll(p, "\\", "/")` instead of `filepath.ToSlash(p)` per AGENTS.md security rules.

## Architecture Decisions

### Decision: CmdDeps struct location and visibility

**Choice**: Unexported `cmdDeps` struct in `cmd/deps.go`  
**Alternatives considered**: 
- Exported `CmdDeps` in `cmd` package (rejected: not needed outside cmd)
- Inline deps in each cmd file (rejected: duplication, inconsistent)

**Rationale**: AGENTS.md says "MUST NOT export types unless they need to be used outside the package". The deps struct is only used within `cmd/` for testing. Keeping it unexported enforces encapsulation.

### Decision: Dependency injection pattern

**Choice**: `runX(cmd, args)` calls `runXWithDeps(cmd, args, defaultDeps)` where `defaultDeps` is a package-level var  
**Alternatives considered**:
- Closure-based injection in RunE (rejected: verbose, harder to test)
- Global setter functions (rejected: not thread-safe, harder to reason about)
- Pass deps through cobra context (rejected: over-engineering, type-unsafe)

**Rationale**: The wrapper pattern keeps cobra's `RunE` signature unchanged while enabling test injection. Package-level `defaultDeps` with sensible defaults (os.Stdout, os.Stderr, os.Stdin, RealConfigLoader) means zero-value is usable per AGENTS.md. Tests override by calling `runXWithDeps` directly with custom deps.

### Decision: ConfigLoader interface reuse

**Choice**: Reuse `actions.ConfigLoader` interface from `internal/actions/interfaces.go`  
**Alternatives considered**:
- Define new `cmdConfigLoader` interface in cmd package (rejected: duplication)
- Use concrete `*config.Config` type (rejected: not mockable)

**Rationale**: AGENTS.md says "MUST place interfaces in the consumer package". The `actions.ConfigLoader` interface already exists and is used by `BackupAction`, `LoginAction`, etc. Reusing it avoids duplication and maintains consistency. The `cmd` package is a consumer of this interface.

### Decision: canonicalPath fix approach

**Choice**: Replace `filepath.ToSlash(p)` with `strings.ReplaceAll(p, "\\", "/")`  
**Alternatives considered**:
- Use `filepath.FromSlash` (rejected: OS-dependent, opposite of what we need)
- Keep `filepath.ToSlash` (rejected: violates AGENTS.md, fails on Linux)

**Rationale**: AGENTS.md explicitly states: "MUST use `path.Clean` + `strings.ReplaceAll(path, "\\", "/")` for canonical path comparison — NOT `filepath.ToSlash` (OS-dependent, fails on Linux)". `filepath.ToSlash` is a no-op on Linux/macOS, so tests pass locally but fail in CI. `strings.ReplaceAll` is OS-agnostic and always normalizes backslashes to forward slashes.

### Decision: Scope of refactoring

**Choice**: Refactor all 13 cmd files for consistency  
**Alternatives considered**:
- Only refactor the 7 that call config.Load() (rejected: inconsistent pattern)
- Only refactor the 7 that fail tests (rejected: same issue)

**Rationale**: Even though only 11 calls to `config.Load()` exist across 6 files (backup, profile, login, list, schedule), refactoring all 13 commands establishes a consistent pattern. Commands like `diff`, `restore`, `undo` don't call `config.Load()` now but might in the future. A uniform pattern is easier to understand and maintain. The proposal explicitly states "Refactor all 13 cmd files".

## Data Flow

### Current flow (fails on Linux):
```
cobra RunE → runBackup(cmd, args)
  → config.Load() [reads ~/.config/bak/config.json]
  → actions.BackupAction{Config: cfg}
  → action.Run(cmd, args)
```

### New flow (testable):
```
cobra RunE → runBackup(cmd, args)
  → runBackupWithDeps(cmd, args, defaultDeps)
    → deps.ConfigLoader.Load() [injectable]
    → actions.BackupAction{Config: cfg}
    → action.Run(cmd, args)

Test:
  deps := cmdDeps{ConfigLoader: mockLoader, Stdout: buf, ...}
  → runBackupWithDeps(cmd, args, deps)
```

### canonicalPath data flow:
```
Input: "C:\\Users\\alice\\.config\\opencode\\config.yaml"
  → strings.ReplaceAll(p, "\\", "/")
  → "C:/Users/alice/.config/opencode/config.yaml"
  → path.Clean(...)
  → "C:/Users/alice/.config/opencode/config.yaml"

Input: "/home/alice/.config/opencode/config.yaml"
  → strings.ReplaceAll(p, "\\", "/")
  → "/home/alice/.config/opencode/config.yaml" [no change]
  → path.Clean(...)
  → "/home/alice/.config/opencode/config.yaml"
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/deps.go` | Create | Define `cmdDeps` struct with ConfigLoader, Stdout, Stderr, Stdin fields. Initialize `defaultDeps` with os.Stdout, os.Stderr, os.Stdin, and `actions.RealConfigLoader{}`. (~40 lines) |
| `cmd/backup.go` | Modify | Split `runBackup` into `runBackup` (calls `runBackupWithDeps`) and `runBackupWithDeps(cmd, args, deps)`. Replace `config.Load()` with `deps.ConfigLoader.Load()`. Replace `os.Stderr` with `deps.Stderr`. |
| `cmd/profile.go` | Modify | Split 4 `runProfileX` functions into wrappers + `runProfileXWithDeps`. Replace 5 `config.Load()` calls with `deps.ConfigLoader.Load()`. |
| `cmd/login.go` | Modify | Split `runLogin` and `runLoginInteractive`. Replace 2 `config.Load()` calls. Replace `os.Stdin` with `deps.Stdin`. |
| `cmd/list.go` | Modify | Split `runListLocal` and `runListCloud`. Replace 1 `config.Load()` call. Replace `os.Stdout`/`os.Stderr` with `deps.Stdout`/`deps.Stderr`. |
| `cmd/schedule.go` | Modify | Split `runScheduleCreate` and `runScheduleList`. Replace 2 `config.Load()` calls. |
| `cmd/diff.go` | Modify | Split `runDiff` into wrapper + `runDiffWithDeps`. No config.Load() but refactor for consistency. |
| `cmd/restore.go` | Modify | Split `runRestore` for consistency. |
| `cmd/undo.go` | Modify | Split `runUndo` for consistency. |
| `cmd/verify.go` | Modify | Split `runVerify` for consistency. |
| `cmd/pick.go` | Modify | Split `runPick` for consistency. |
| `cmd/export.go` | Modify | Split `runExport` for consistency. |
| `cmd/version.go` | Modify | Split `runVersion` for consistency (no deps needed, but uniform pattern). |
| `cmd/testhelper_test.go` | Create | `setupTestDeps(t)` returns `cmdDeps` with temp config file, mock loader, and buffers. (~50 lines) |
| `cmd/*_test.go` (7-10 files) | Modify | Update tests to call `runXWithDeps(cmd, args, deps)` instead of going through `rootCmd.Execute()`. Use `setupTestDeps(t)` for common setup. |
| `internal/diff/diff.go` | Modify | Fix `canonicalPath()`: replace `filepath.ToSlash(p)` with `strings.ReplaceAll(p, "\\", "/")`. Add import for `"strings"`. |

## Interfaces / Contracts

### cmdDeps struct (unexported)

```go
// cmd/deps.go
package cmd

import (
    "io"
    "os"
    "github.com/danielxxomg/bak-cli/internal/actions"
)

// cmdDeps holds injectable dependencies for command execution.
// Tests override these to isolate from the real filesystem.
type cmdDeps struct {
    ConfigLoader actions.ConfigLoader
    Stdout       io.Writer
    Stderr       io.Writer
    Stdin        io.Reader
}

// defaultDeps provides production defaults.
// Zero-value is usable: os.Stdout, os.Stderr, os.Stdin, RealConfigLoader.
var defaultDeps = cmdDeps{
    ConfigLoader: &actions.RealConfigLoader{},
    Stdout:       os.Stdout,
    Stderr:       os.Stderr,
    Stdin:        os.Stdin,
}
```

### Wrapper pattern example

```go
// cmd/backup.go
func runBackup(cmd *cobra.Command, args []string) error {
    return runBackupWithDeps(cmd, args, defaultDeps)
}

func runBackupWithDeps(cmd *cobra.Command, args []string, deps cmdDeps) error {
    // ... existing logic ...
    cfg, err := deps.ConfigLoader.Load()
    if err != nil {
        return fmt.Errorf("load config for profile: %w", err)
    }
    // ... rest of logic using deps.Stderr instead of os.Stderr ...
}
```

### Test helper

```go
// cmd/testhelper_test.go
func setupTestDeps(t *testing.T) (cmdDeps, *bytes.Buffer, *bytes.Buffer) {
    t.Helper()
    
    // Create temp config file
    tmpDir := t.TempDir()
    cfgPath := filepath.Join(tmpDir, "config.json")
    os.WriteFile(cfgPath, []byte(`{"schema_version":"0.3.0"}`), 0600)
    
    // Mock loader that reads from temp path
    loader := &mockConfigLoader{path: cfgPath}
    
    stdout := new(bytes.Buffer)
    stderr := new(bytes.Buffer)
    
    deps := cmdDeps{
        ConfigLoader: loader,
        Stdout:       stdout,
        Stderr:       stderr,
        Stdin:        strings.NewReader(""), // empty stdin for tests
    }
    
    return deps, stdout, stderr
}

// mockConfigLoader implements actions.ConfigLoader for tests.
type mockConfigLoader struct {
    path string
}

func (m *mockConfigLoader) Load() (*actions.Config, error) {
    cfg, err := config.LoadPath(m.path)
    if err != nil {
        return nil, err
    }
    return &actions.Config{SchemaVersion: cfg.SchemaVersion}, nil
}
```

### canonicalPath fix

```go
// internal/diff/diff.go
import (
    "path"
    "strings" // add this
    // remove "path/filepath" if unused
)

// canonicalPath normalizes a path for cross-platform comparison.
func canonicalPath(p string) string {
    return path.Clean(strings.ReplaceAll(p, "\\", "/"))
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `cmdDeps` default values | Verify `defaultDeps` has os.Stdout, os.Stderr, os.Stdin, RealConfigLoader |
| Unit | `canonicalPath` cross-platform | Table-driven test with Windows, macOS, Linux paths. Verify all normalize to forward slashes. |
| Unit | Each `runXWithDeps` function | Call directly with mock deps. Verify config loader is called, output goes to deps.Stdout/Stderr. |
| Integration | Existing cmd tests | Update to use `setupTestDeps(t)` and call `runXWithDeps`. Verify no regression in behavior. |
| E2E | Docker test (task test:linux) | Run full test suite in Linux container. Verify all tests pass without real `~/.config/bak/`. |

### Test isolation verification

1. **Before refactor**: Tests fail on Linux because `config.Load()` reads real `~/.config/bak/config.json`
2. **After refactor**: Tests pass on all OS because `deps.ConfigLoader.Load()` reads from temp file
3. **Verification**: Run `task test:linux` — all tests should pass

### canonicalPath test cases

```go
func TestCanonicalPath(t *testing.T) {
    tests := []struct{
        name     string
        input    string
        expected string
    }{
        {"windows path", `C:\Users\alice\.config\opencode`, "C:/Users/alice/.config/opencode"},
        {"unix path", "/home/alice/.config/opencode", "/home/alice/.config/opencode"},
        {"mixed slashes", `C:/Users\alice/.config\opencode`, "C:/Users/alice/.config/opencode"},
        {"relative path", "../config", "../config"},
        {"empty", "", "."},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := canonicalPath(tt.input)
            if got != tt.expected {
                t.Errorf("canonicalPath(%q) = %q, want %q", tt.input, got, tt.expected)
            }
        })
    }
}
```

## Migration / Rollout

No migration required. This is a pure refactor with zero user-facing behavior changes.

**Rollout plan**:
1. Create `cmd/deps.go` with `cmdDeps` struct and `defaultDeps`
2. Refactor one cmd file (e.g., `backup.go`) as a proof of concept
3. Update its tests to use `setupTestDeps(t)`
4. Verify tests pass locally and in Docker
5. Refactor remaining 12 cmd files using the same pattern
6. Fix `canonicalPath()` in `internal/diff/diff.go`
7. Run full test suite: `task test` (Windows) and `task test:linux` (Docker)
8. Commit with conventional commit message: `refactor(cmd): inject dependencies for test isolation`

**Rollback**: Revert the feature branch `feature/cmd-di-refactor`. No database migrations or config changes.

## Open Questions

- [ ] Should `cmdDeps` include a `BakDir` field to inject the backup directory path, or is mocking `ConfigLoader` sufficient? (Current leaning: ConfigLoader is enough; BakDir is derived from home dir which tests already override via env vars)
- [ ] Should we create a `MockConfigLoader` in `internal/actions/mock_impl.go` for reuse, or keep it local to `cmd/testhelper_test.go`? (Current leaning: keep local to cmd tests since it's cmd-specific)
- [ ] For commands that don't call `config.Load()` (diff, restore, undo, verify, pick, export, version), should we still create the `runXWithDeps` wrapper even if it just passes through? (Current leaning: yes, for consistency)
