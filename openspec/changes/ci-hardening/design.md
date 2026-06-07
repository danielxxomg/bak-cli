# Design: CI Hardening (v1.2.1)

## Technical Approach

Three-area patch release: (1) fix 7 lint violations per staticcheck recommendations, (2) fix macOS CI by introducing a cross-platform `setConfigHome` test helper that sets `HOME` (macOS) + `XDG_CONFIG_HOME` (Linux) + `APPDATA` (Windows), (3) add unit tests for 7 untested action files using existing DI patterns and `t.TempDir()` fixtures.

## Architecture Decisions

### Decision: setConfigHome location

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `cmd/testhelper_test.go` | Simple but invisible to `internal/` packages | ❌ |
| `internal/config/testutil/` (exported, `package configtest`) | Cross-package import, follows Go convention for test helpers | ✅ |
| Shared `internal/testutil/` package | Generic but adds a new top-level package | ❌ |

**Rationale**: `internal/config/testutil/` with `package configtest` lets both `config` and `actions` tests import it. Setting `HOME` on macOS makes `os.UserConfigDir()` resolve to `$HOME/Library/Application Support` — the root cause fix. The helper also sets `XDG_CONFIG_HOME` (Linux) and `APPDATA` (Windows) for full 3-OS coverage.

### Decision: list_cloud.go registry injection

| Option | Tradeoff | Decision |
|--------|----------|----------|
| New `RegistryProvider` interface | Overkill — `*cloud.ProviderRegistry` is already a concrete type with methods | ❌ |
| `RegistryFactory func() *cloud.ProviderRegistry` struct field | Matches existing `NewRegistry func()` pattern in `PickBackupAction` | ✅ |

**Rationale**: `PickBackupAction.NewRegistry` already uses `func() (*adapters.Registry, error)`. Same pattern for consistency. Tests inject a factory that returns a registry pre-populated with `MockProvider` instances (already defined in `mock_impl_test.go`). Production code: `if a.RegistryFactory == nil { ... default ... }`.

### Decision: Test fixtures for verify_backup / diff_backups

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Inject `ResolveBackupID` function | Breaks production API, over-injection | ❌ |
| `t.Setenv("HOME", tempDir)` + real FS fixtures | Matches existing `setupBackupDir` pattern, no API changes | ✅ |

**Rationale**: `backup.ResolveBackupID()` → `backup.BakDir()` → `os.UserHomeDir()`. Setting `HOME` (and `USERPROFILE` on Windows) controls the full chain. Reuse `setupBackupDir` from `list_local_test.go` (already creates `<bakDir>/backups/<id>/manifest.json`). For verify: write files + compute SHA-256. For diff: create two backup dirs with different manifests.

### Decision: Commit order

5 atomic commits (Conventional Commits):

| # | Type | Scope | Content |
|---|------|-------|---------|
| 1 | `fix` | lint | SA5011 nil check, QF1012 Fprintf ×3, QF1001 De Morgan, SA9003 empty branch, SA4023 interface comparison |
| 2 | `fix` | ci | `configtest.SetConfigHome` helper + fix macOS tests in `internal/config/*_test.go` + E2E fixture |
| 3 | `test` | actions | `login_interactive_test.go`, `undo_test.go`, `schedule_test.go`, `list_cloud_test.go` (pure DI, no FS) |
| 4 | `test` | actions | `verify_backup_test.go`, `diff_backups_test.go`, `pick_backup_test.go` (FS fixtures via `setConfigHome`) |
| 5 | `chore` | actions | Inject `RegistryFactory` into `list_cloud.go` (production code change for testability) |

**Note**: Commit 5 must come before or with commit 3 (list_cloud_test.go needs the injection point). Adjusted order: 5 → 3 merge, or commit 5 first.

**Revised order**: 1 → 2 → 5 → 3 → 4.

## Data Flow

### list_cloud.go — Registry Injection

```
ListCloudAction.Run(providerName)
    │
    ├─ RegistryFactory != nil → factory() → *cloud.ProviderRegistry (mock)
    └─ RegistryFactory == nil → cloud.NewProviderRegistry() + register all
         │
         └→ registry.Get(name) → Provider → List() → []BackupMeta
```

### verify_backup / diff_backups — Test Isolation

```
configtest.SetConfigHome(t, tempDir)
    → t.Setenv("HOME", tempDir) [+ USERPROFILE on Windows]
    → backup.BakDir() returns tempDir/.bak
    → create tempDir/.bak/backups/<id>/manifest.json + files
    → ResolveBackupID finds the fixture directory
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/config/testutil/testutil.go` | Create | `configtest.SetConfigHome(t, dir)` — cross-platform env var helper |
| `cmd/export_test.go` | Modify | Add nil guard before `flag.DefValue` access (SA5011) |
| `cmd/pick.go` | Modify | `fmt.Fprint(w, fmt.Sprintf(...))` → `fmt.Fprintf(w, ...)` (QF1012) |
| `cmd/wizard.go` | Modify | Same Fprintf fix ×2 (QF1012) |
| `internal/cloud/pack_test.go` | Modify | De Morgan simplification in `isBase64` (QF1001) |
| `internal/config/migration_test.go` | Modify | Remove empty branch at :142, add intent comment (SA9003) |
| `internal/schedule/scheduler_unix_test.go` | Modify | Fix `s == nil` → compare concrete `*CronScheduler` (SA4023) |
| `internal/config/*_test.go` | Modify | Replace ad-hoc env setup with `configtest.SetConfigHome` |
| `internal/actions/list_cloud.go` | Modify | Add `RegistryFactory` field, use in `Run()` with nil-default |
| `internal/actions/login_interactive_test.go` | Create | Table-driven: mock ConfigLoader + Wizard, test happy/error paths |
| `internal/actions/list_cloud_test.go` | Create | MockProvider + RegistryFactory, test table output + empty + error |
| `internal/actions/undo_test.go` | Create | Inject HomeDir/IsRepo/UndoFn, test success + no-repo + revert-fail |
| `internal/actions/schedule_test.go` | Create | Mock Scheduler via NewScheduler field, test Create/List/Remove |
| `internal/actions/pick_backup_test.go` | Create | Test ResolveBackupID + PickBackupAction with mock Picker |
| `internal/actions/verify_backup_test.go` | Create | setConfigHome + real files with SHA-256, test pass + mismatch |
| `internal/actions/diff_backups_test.go` | Create | setConfigHome + two backup fixtures, test diff output + identical |
| `testdata/e2e/profile_create_list.txtar` | Modify | macOS path expectation (`$HOME/Library/Application Support/`) |

## Interfaces / Contracts

### configtest.SetConfigHome

```go
// Package configtest provides test helpers for config isolation.
package configtest

// SetConfigHome sets OS-specific env vars so config/home lookups
// resolve under dir. Uses t.Setenv for automatic cleanup.
//   - Linux:   XDG_CONFIG_HOME
//   - macOS:   HOME (os.UserConfigDir uses $HOME/Library/Application Support)
//   - Windows: APPDATA + USERPROFILE
func SetConfigHome(t testing.TB, dir string)
```

### ListCloudAction.RegistryFactory

```go
type ListCloudAction struct {
    // ... existing fields ...

    // RegistryFactory creates the provider registry.
    // If nil, Run() uses the default registry with all providers.
    RegistryFactory func() *cloud.ProviderRegistry
}
```

## Testing Strategy

| Layer | What | Approach |
|-------|------|----------|
| Unit | 7 action files | Table-driven, hand-rolled mocks (existing `MockProvider`, `MockFileSystem`), injected functions |
| Unit | Lint fixes | Existing tests cover behavior; lint is compile-time/static |
| Unit | macOS fix | `configtest.SetConfigHome` + verify `os.UserConfigDir()` resolves correctly |
| Integration | None new | Existing E2E txtar covers integration; macOS path fix validated in CI matrix |
| E2E | txtar fixture | Update `profile_create_list.txtar` macOS expectation |

## Migration / Rollout

No migration required. All changes are test/lint fixes — zero behavior changes to production commands.

## Open Questions

- [ ] SA5011 in `export_test.go:38`: the nil check at line 35 should guard line 38 in the same function — verify if staticcheck still flags after confirming control flow. May need `if flag == nil { return }` (early return) instead of `t.Error` to satisfy the analyzer.
- [ ] Commit 5 (RegistryFactory injection) is a production code change — confirm it fits the `test:` commit scope or should be `refactor:` type.
