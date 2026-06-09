# Design: actions-di (Dependency Injection for Actions Package)

## Technical Approach

Remove the `github.com/spf13/cobra` dependency from `internal/actions/` by injecting `io.Writer` fields for stdout/stderr output, following the existing `RestoreAction.Stdin` pattern. This enforces the architecture boundary defined in AGENTS.md: actions must accept `io.Writer`/`io.Reader` and plain parameters, not framework types.

The approach extends the `cmdDeps` pattern from the previous `cmd-di-refactor` cycle. The `cmd/` package remains the only layer that translates cobra types to action parameters.

## Architecture Decisions

### Decision 1: How do actions get stdout/stderr?

| Option | Tradeoff | Decision |
|--------|----------|----------|
| **A: Add Stdout/Stderr fields to action structs** | Follows existing DI pattern (RestoreAction.Stdin), struct field injection per AGENTS.md, zero-value fallback to os.Stdout/os.Stderr | ✅ **CHOSEN** |
| B: Accept io.Writer parameters in Run() | Explicit at call site, but breaks existing pattern and AGENTS.md DI rules | ❌ Rejected |
| C: Use a dependency struct (like cmdDeps) | Groups dependencies, but overkill for 2 writers and inconsistent with RestoreAction.Stdin | ❌ Rejected |

**Rationale**: Option A follows the established `RestoreAction.Stdin` pattern, adheres to AGENTS.md struct field injection rules, and maintains consistency across all three actions. The zero-value fallback (nil → os.Stdout/os.Stderr) provides backward compatibility and testability.

### Decision 2: Run signature changes

| Action | Current Signature | New Signature | Rationale |
|--------|------------------|---------------|-----------|
| BackupAction | `Run(cmd *cobra.Command, args []string) error` | `Run() error` | Both `cmd` and `args` are unused in the function body |
| PushAction | `Run(cmd *cobra.Command, args []string) error` | `Run(args []string) error` | `cmd` is unused, but `args[0]` is used for backup ID in `resolveBackupID()` |
| RestoreAction | `Run(cmd *cobra.Command, args []string) error` | `Run() error` | Both `cmd` and `args` are unused; backupID comes from struct field `a.BackupDir` |

**Rationale**: Remove unused parameters to simplify the API. Keep `args` only where it's actually used (PushAction). This makes the action interface cleaner and removes the cobra dependency entirely.

### Decision 3: How do cmd/ callers pass writers?

**Choice**: cmd/ callers pass `deps.Stdout` and `deps.Stderr` when constructing action structs, using the existing `depsFromCmd(cmd)` helper.

**Rationale**: This follows the pattern already established in `cmd/restore.go` which passes `Stdin: deps.Stdin`. The `cmdDeps` struct already extracts writers from cobra via `cmd.OutOrStdout()` / `cmd.ErrOrStderr()`, preserving cobra's testability. No new infrastructure needed.

### Decision 4: What happens to existing tests?

**Choice**: Tests inject `io.Discard` or `bytes.Buffer` for Stdout/Stderr. Nil writers fall back to os.Stdout/os.Stderr for backward compatibility.

**Rationale**: 
- Tests currently call `Run(nil, nil)` or `Run(nil, args)` with nil cmd
- After the change, tests call `Run()` or `Run(args)` without the cmd parameter
- Injecting `io.Discard` suppresses output in tests that don't verify it
- Injecting `bytes.Buffer` allows tests to capture and verify output
- Nil fallback ensures existing tests don't break immediately, but they should be updated for proper isolation

## Data Flow

```
cmd/backup.go                    internal/actions/backup.go
┌─────────────────┐              ┌──────────────────────────┐
│ runBackup(cmd)  │              │ BackupAction             │
│   ↓             │              │ ├─ FS FileSystem         │
│ deps =          │              │ ├─ Stdout io.Writer ─────┼──→ fmt.Fprintf(a.Stdout, ...)
│   depsFromCmd() │              │ ├─ Stderr io.Writer ─────┼──→ fmt.Fprintf(a.Stderr, ...)
│   ↓             │              │ └─ ...other fields       │
│ action =        │              │                          │
│   &BackupAction │              │ Run() error              │
│   ├─ Stdout:    │──────────────│   ├─ fmt.Fprintf(Stdout) │
│   │  deps.Stdout│              │   └─ fmt.Fprintf(Stderr) │
│   └─ Stderr:    │              │                          │
│      deps.Stderr│              └──────────────────────────┘
│   ↓             │
│ action.Run()    │
└─────────────────┘
```

The same pattern applies to PushAction and RestoreAction. For RestoreAction, the existing `cmd.OutOrStdout()` / `cmd.ErrOrStderr()` calls are replaced with `a.Stdout` / `a.Stderr`.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/actions/backup.go` | Modify | Remove cobra import, add `Stdout io.Writer` and `Stderr io.Writer` fields, change `Run()` signature (no params), replace `fmt.Printf` → `fmt.Fprintf(a.Stdout, ...)`, replace `fmt.Fprintf(os.Stderr, ...)` → `fmt.Fprintf(a.Stderr, ...)`, add nil-check fallback to os.Stdout/os.Stderr |
| `internal/actions/push.go` | Modify | Remove cobra import, add `Stdout io.Writer` and `Stderr io.Writer` fields, change `Run(args []string)` signature, replace fmt calls, add nil-check fallback |
| `internal/actions/restore.go` | Modify | Remove cobra import, add `Stdout io.Writer` and `Stderr io.Writer` fields, change `Run()` signature (no params), replace `cmd.OutOrStdout()` → `a.Stdout`, replace `cmd.ErrOrStderr()` → `a.Stderr`, add nil-check fallback |
| `internal/actions/backup_test.go` | Modify | Update all `Run(nil, nil)` calls to `Run()`, inject `io.Discard` or `bytes.Buffer` for Stdout/Stderr in test structs |
| `internal/actions/push_test.go` | Modify | Update all `Run(nil, args)` calls to `Run(args)`, inject writers in test structs |
| `internal/actions/restore_test.go` | Modify | Update all `Run(nil, args)` calls to `Run()`, inject writers in test structs |
| `cmd/backup.go` | Modify | Pass `Stdout: deps.Stdout` and `Stderr: deps.Stderr` when constructing `BackupAction` struct |
| `cmd/push.go` | Modify | Pass `Stdout: deps.Stdout` and `Stderr: deps.Stderr` when constructing `PushAction` struct |
| `cmd/restore.go` | Modify | Pass `Stdout: deps.Stdout` and `Stderr: deps.Stderr` when constructing `RestoreAction` struct (already passes `Stdin: deps.Stdin`) |

## Interfaces / Contracts

No new interfaces needed. Using standard `io.Writer` from Go stdlib.

### Struct Field Additions

```go
// BackupAction (internal/actions/backup.go)
type BackupAction struct {
    FS       FileSystem
    Config   ConfigLoader
    Registry *adapters.Registry
    Stdout   io.Writer  // NEW: nil falls back to os.Stdout
    Stderr   io.Writer  // NEW: nil falls back to os.Stderr
    
    // Parameters (from CLI flags).
    Preset           string
    AdapterFilter    []string
    Verbose          bool
    BakVersion       string
    SecretPatterns   []*regexp.Regexp
    CustomCategories []string
    HostnameFn       HostnameFunc
}

// PushAction (internal/actions/push.go)
type PushAction struct {
    FS       FileSystem
    Provider string
    Profile  string
    Verbose  bool
    Stdout   io.Writer  // NEW: nil falls back to os.Stdout
    Stderr   io.Writer  // NEW: nil falls back to os.Stderr
    
    Factory      ProviderFactory
    HostnameFn   HostnameFunc
    ConfigLoader func() (*config.Config, error)
}

// RestoreAction (internal/actions/restore.go)
type RestoreAction struct {
    FS        FileSystem
    BackupDir string
    DryRun    bool
    Force     bool
    Verbose   bool
    GitDir    string
    Stdin     io.Reader  // EXISTING
    Stdout    io.Writer  // NEW: nil falls back to os.Stdout
    Stderr    io.Writer  // NEW: nil falls back to os.Stderr
}
```

### Nil-Check Helper Pattern

Each action will use this pattern at the start of `Run()`:

```go
func (a *BackupAction) Run() error {
    out := a.Stdout
    if out == nil {
        out = os.Stdout
    }
    errOut := a.Stderr
    if errOut == nil {
        errOut = os.Stderr
    }
    
    // ... rest of implementation uses out and errOut
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Actions write to injected writers | Inject `bytes.Buffer` for Stdout/Stderr, verify output captured correctly |
| Unit | Actions handle nil writers | Pass nil writers, verify fallback to os.Stdout/os.Stderr doesn't panic |
| Unit | Existing tests still pass | Update `Run()` calls to new signature, inject `io.Discard` for writers to suppress output |
| Integration | cmd/ passes writers correctly | Use `cmd.SetOut()` / `cmd.SetErr()` in cmd tests, verify action receives them via deps |

### Test Migration Pattern

**Before**:
```go
action := &BackupAction{
    FS:       setupMockFS(),
    Registry: setupBackupRegistry(),
    Preset:   "quick",
}
err := action.Run(nil, nil)
```

**After**:
```go
action := &BackupAction{
    FS:       setupMockFS(),
    Registry: setupBackupRegistry(),
    Preset:   "quick",
    Stdout:   io.Discard,  // or bytes.Buffer to capture
    Stderr:   io.Discard,
}
err := action.Run()
```

## Migration / Rollout

No migration required. This is a pure internal refactor with no behavior changes:
- No API changes for end users (CLI flags and output remain identical)
- No data migration
- No feature flags needed
- Rollback is safe: revert the commit

The change is backward compatible at the struct level (fields are additive). The `Run()` signature change is internal to the codebase and does not affect external consumers.

## Open Questions

None. The design follows established patterns from the previous `cmd-di-refactor` cycle and the existing `RestoreAction.Stdin` implementation. All decisions are straightforward applications of AGENTS.md architecture rules.
