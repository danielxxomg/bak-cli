# Design: Coverage DI Refactor

## Technical Approach

Targeted dependency injection across 3 packages to close the coverage gap (68.6% → ≥80%). Each PR is independently revertable. The core strategy: replace direct `os.*` calls with injected interfaces/struct fields following the existing `BackupAction` pattern (struct field injection, hand-rolled mocks, consumer-side interfaces).

## Architecture Decisions

### PR1 — Adapters

| Decision | Options | Choice | Rationale |
|---|---|---|---|
| `LoadYAMLAdapters` homeDir | Add param vs inject FS vs remove check | **Add `homeDir string` param** | Minimal change; 2 callers updated in same PR; path traversal check preserved |
| `ConfigAdapter` testability | Inject FS into adapter vs test via tempDir | **Test via `t.TempDir()` as homeDir** | Adapter interface already receives `homeDir`; real `os.*` calls work against temp dirs; no interface change needed |
| `register.LoadYAMLAdapters` signature | Add homeDir param vs accept FS | **Add `homeDir string` param** | Consistent with `adapters.LoadYAMLAdapters`; callers (cmd) already know homeDir |

### PR2 — Actions

| Decision | Options | Choice | Rationale |
|---|---|---|---|
| `restoreFile` copy mechanism | `a.FS.CopyFile()` vs add Open/Create to FS | **`a.FS.CopyFile(src, dst)`** | Already exists on `FileSystem` interface and `OSFileSystem`; single call replaces 15 lines |
| `ProviderFactory` location | `actions/interfaces.go` vs new file | **`actions/interfaces.go`** | Consumer-side interface per AGENTS.md; `cloud` import is acceptable (no cycle) |
| `HostnameFunc` pattern | Struct field vs package var | **Struct field `HostnameFunc func() (string, error)`** | Per-action control; follows struct field injection rule; tests set per-case |
| `PushAction` provider wiring | Inject `ProviderFactory` vs inject `Provider` | **Inject `ProviderFactory`** | Action needs to build provider with config; factory pattern allows lazy construction |

### PR3 — Cmd

| Decision | Options | Choice | Rationale |
|---|---|---|---|
| Profile CRUD extraction | New action structs vs extracted functions | **Extract functions in `cmd/`** | Profile ops are thin (load config, mutate, save); full action structs would be over-engineering |
| `config.Load()` in cmd | Interface injection vs env override | **Keep direct calls; test via `t.TempDir()` + `BAK_CONFIG_DIR`** | Profile/list/verify/diff are thin wrappers; AGENTS.md says "no logic in cmd" but CRUD is wiring, not business logic |
| Login stdin injection | `io.Reader` field on action vs package var | **`Stdin io.Reader` field on `LoginAction`** | Follows struct field injection pattern; `os.Stdin` is default in production wiring |
| Bubbletea `Program.Run()` | Test model Update/View vs inject runner | **Test model logic only; skip `Program.Run()`** | Per AGENTS.md: "MUST NOT test bubbletea.Program.Run() directly" |

## Data Flow

### Before: restoreFile (PR2)
```
RestoreAction.restoreFile()
    ├─ os.Open(src)           ← untestable
    ├─ os.Create(dst)         ← untestable
    └─ io.Copy(dst, src)
```

### After: restoreFile (PR2)
```
RestoreAction.restoreFile()
    ├─ path validation (src under backupDir)
    ├─ path validation (dst under homeDir)
    ├─ a.FS.MkdirAll(parent)  ← injected
    └─ a.FS.CopyFile(src,dst) ← injected
```

### Before: Push provider wiring
```
cmd/runPush → PushAction.Run()
                  ├─ cloud.NewProviderRegistry()   ← hardcoded
                  ├─ cloud.NewGitHubGistProvider()  ← hardcoded
                  └─ reg.Register(...)
```

### After: Push provider wiring
```
cmd/runPush → PushAction.Run()
                  └─ a.Factory.CreateProvider()    ← injected
```

### Before: cmd/profile RunE
```
runProfileCreate(cmd, args)
    ├─ config.Load()     ← real FS
    ├─ validate provider ← business logic in cmd
    ├─ mutate cfg
    └─ cfg.Save()        ← real FS
```

### After: cmd/profile RunE
```
runProfileCreate(cmd, args)
    └─ actions.ProfileCreate(name, opts)  ← extracted, testable
         ├─ config.Load()
         ├─ validate
         ├─ mutate
         └─ cfg.Save()
```

## File Changes

### PR1 — Adapters
| File | Action | Description |
|---|---|---|
| `internal/adapters/yaml.go` | Modify | `LoadYAMLAdapters(dir, homeDir string)` — add homeDir param |
| `internal/adapters/yaml_test.go` | Create | Tests for `ConfigAdapter` (Detect, ListItems, Backup, Restore), `LoadYAMLAdapters`, `copyFile`, `fileHash` |
| `internal/adapters/register/register.go` | Modify | `LoadYAMLAdapters(reg, override, homeDir string)` — add homeDir param |
| `internal/adapters/register/register_test.go` | Create | Tests for `All()` and `LoadYAMLAdapters()` |
| `cmd/backup.go` | Modify | Pass `homeDir` to `register.LoadYAMLAdapters` |

### PR2 — Actions
| File | Action | Description |
|---|---|---|
| `internal/actions/interfaces.go` | Modify | Add `ProviderFactory` interface |
| `internal/actions/restore.go` | Modify | `restoreFile()` uses `a.FS.CopyFile()`; add `Stdin io.Reader` field |
| `internal/actions/push.go` | Modify | Add `Factory ProviderFactory` + `HostnameFunc` fields; remove direct cloud.* calls |
| `internal/actions/pull.go` | Modify | Add `Factory ProviderFactory` field; inject config via `ConfigLoader` |
| `internal/actions/backup.go` | Modify | Add `HostnameFunc` field; use it instead of `os.Hostname()` |
| `internal/actions/push_test.go` | Modify | Add tests using MockProviderFactory |
| `internal/actions/pull_test.go` | Modify | Add tests using MockProviderFactory |
| `internal/actions/restore_test.go` | Modify | Add tests for restoreFile via MockFileSystem.CopyFile |
| `internal/actions/mock_impl.go` | Modify | Add `MockProviderFactory`, `MockProvider` |

### PR3 — Cmd
| File | Action | Description |
|---|---|---|
| `internal/actions/profile.go` | Create | `ProfileCreate()`, `ProfileList()`, `ProfileShow()`, `ProfileDelete()` extracted functions |
| `internal/actions/profile_test.go` | Create | Table-driven tests for all profile operations |
| `internal/actions/login.go` | Create | `LoginAction` struct with `Stdin io.Reader` field |
| `internal/actions/login_test.go` | Create | Tests with `strings.NewReader("y\n")` for stdin |
| `cmd/profile.go` | Modify | RunE delegates to `actions.Profile*()` |
| `cmd/login.go` | Modify | RunE delegates to `actions.LoginAction` |
| `cmd/push.go` | Modify | Wire `ProviderFactory` into `PushAction` |
| `cmd/pull.go` | Modify | Wire `ProviderFactory` into `PullAction` |

## Interfaces / Contracts

```go
// internal/actions/interfaces.go — ADDITIONS

// ProviderFactory creates cloud providers. Consumer-side interface
// to decouple actions from concrete cloud provider construction.
type ProviderFactory interface {
    // CreateProvider returns a configured cloud.Provider for the given name.
    CreateProvider(name string) (cloud.Provider, error)
}

// HostnameFunc returns the current hostname. Injected for testability.
type HostnameFunc func() (string, error)
```

```go
// internal/actions/mock_impl.go — ADDITIONS

// MockProviderFactory implements ProviderFactory for testing.
type MockProviderFactory struct {
    Providers map[string]cloud.Provider
    Err       error
}
func (m *MockProviderFactory) CreateProvider(name string) (cloud.Provider, error) {
    if m.Err != nil { return nil, m.Err }
    p, ok := m.Providers[name]
    if !ok { return nil, fmt.Errorf("unknown provider: %q", name) }
    return p, nil
}

// MockProvider implements cloud.Provider for testing.
type MockProvider struct {
    MockName   string
    PushFn     func([]byte, cloud.PushMeta) (string, error)
    PullFn     func(string) ([]byte, error)
    ListFn     func() ([]cloud.BackupMeta, error)
}
// ... Name(), Push(), Pull(), List() delegate to MockName/PushFn/PullFn/ListFn
```

```go
// Struct field additions

// RestoreAction — ADD
type RestoreAction struct {
    // ... existing fields ...
    Stdin io.Reader // injectable stdin for confirmation prompt
}

// PushAction — ADD
type PushAction struct {
    // ... existing fields ...
    Factory     ProviderFactory // creates cloud providers
    HostnameFn  HostnameFunc    // returns hostname; nil = os.Hostname
}

// PullAction — ADD
type PullAction struct {
    // ... existing fields ...
    Factory  ProviderFactory
    Config   ConfigLoader      // replaces direct config.Load()
}

// BackupAction — ADD
type BackupAction struct {
    // ... existing fields ...
    HostnameFn HostnameFunc    // nil = os.Hostname
}
```

## Testing Strategy

| Layer | PR | What | Approach |
|---|---|---|---|
| Unit | PR1 | `ConfigAdapter.Detect/ListItems/Backup/Restore` | `t.TempDir()` as homeDir; create fixture YAML + config files |
| Unit | PR1 | `LoadYAMLAdapters` with injected homeDir | `t.TempDir()` with valid/invalid YAML files; test traversal rejection |
| Unit | PR1 | `register.All()`, `register.LoadYAMLAdapters()` | Verify registry state after calls |
| Unit | PR2 | `restoreFile` via `a.FS.CopyFile` | `MockFileSystem` with `CopyFile` tracking; verify path traversal guards |
| Unit | PR2 | `PushAction.Run` with `MockProviderFactory` | Verify archive creation + provider.Push call with correct meta |
| Unit | PR2 | `PullAction.Run` with `MockProviderFactory` | Verify download + extract flow without real cloud |
| Unit | PR3 | `ProfileCreate/List/Show/Delete` | `t.TempDir()` for config dir; table-driven CRUD scenarios |
| Unit | PR3 | `LoginAction` with `strings.NewReader` | Inject stdin; test token prompt + confirmation flows |
| E2E | PR0 | Backup→restore round-trip | Real files in `t.TempDir()`; verify SHA-256 checksums match |

## Migration / Rollout

**PR0**: Lower threshold to 70% + add E2E guardrail. Revert: `git revert` threshold change.
**PR1**: `LoadYAMLAdapters` signature change — update both callers in same commit. Revert: `git revert`.
**PR2**: Additive struct fields (zero-value = current behavior). Revert: `git revert`.
**PR3**: Extract functions from cmd — cmd becomes thin wrappers. Revert: `git revert`.
**PR4**: CI fixes only. Revert: `git revert`.

No data migration. No feature flags. Each PR preserves external behavior.

## Open Questions

- [ ] `PullAction` currently calls `config.Load()` directly for `cfg.Get("github.gist_id")`. Should this go through the existing `ConfigLoader` interface (which only returns `SchemaVersion`), or should we expand `ConfigLoader` to support key-value access? → **Lean: expand `ConfigLoader` with `Get(key string) (string, error)` method.**
- [ ] Login interactive mode uses `bubbletea.Program.Run()`. Per AGENTS.md we test model logic, not `Run()`. Should we extract the token-entry flow (non-interactive) only, leaving the wizard untested? → **Lean: yes, test non-interactive path only.**
