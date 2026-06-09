# Proposal: Generic Adapter Base Struct

## Intent

6–7 adapter packages (`codex`, `kiro`, `kilocode`, `pidev`, `windsurf`, `cursor`, `claudecode`) are ~95% identical copy-paste (~170 lines each, ~1020 lines total). They differ only in `adapterName`, `configRelPath`, `categoryMap`, and a one-line error string. Extract a `GenericAdapter` base to eliminate duplication, enforce the DRY rule in AGENTS.md, and make adding future adapters a ~20-line task.

## Scope

### In Scope
- Create `GenericAdapter` struct in `internal/adapters/` with configurable name, config path, and category map
- Refactor 7 structurally-identical adapters to delegate to `GenericAdapter`
- Preserve all existing behavior exactly — zero behavioral changes
- Keep all existing tests passing without modification

### Out of Scope
- Refactoring `opencode` adapter (has `rootConfigFiles` whitelist and extra category logic — fundamentally different `scanRootFiles`)
- Changing the `Adapter` interface or registry pattern
- Modifying test structure or adding new test files
- Touching `yaml.go` (ConfigAdapter) — different pattern

## Capabilities

### New Capabilities
- `generic-adapter`: Base struct providing Detect, ListItems, Backup, Restore for adapters that follow the standard scan-dir + scan-root-files pattern

### Modified Capabilities
None — this is a pure internal refactor. No spec-level behavior changes.

## Approach

1. Define `GenericAdapter` struct in `internal/adapters/generic.go` with fields: `AdapterName`, `ConfigRelPath`, `Categories map[string]CategoryDir`, `DetectErrContext string`
2. Implement all `Adapter` interface methods on `GenericAdapter` using the shared logic currently duplicated across the 7 packages
3. Each adapter package becomes a thin wrapper: define constants, construct a `GenericAdapter`, delegate interface methods
4. Export `CategoryDir` type from `internal/adapters/` (currently unexported in each package)

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/adapters/generic.go` | New | GenericAdapter struct + methods |
| `internal/adapters/codex/adapter.go` | Modified | Delegate to GenericAdapter (~20 lines) |
| `internal/adapters/kiro/adapter.go` | Modified | Delegate to GenericAdapter (~20 lines) |
| `internal/adapters/kilocode/adapter.go` | Modified | Delegate to GenericAdapter (~20 lines) |
| `internal/adapters/pidev/adapter.go` | Modified | Delegate to GenericAdapter (~20 lines) |
| `internal/adapters/windsurf/adapter.go` | Modified | Delegate to GenericAdapter (~20 lines) |
| `internal/adapters/cursor/adapter.go` | Modified | Delegate to GenericAdapter (~20 lines) |
| `internal/adapters/claudecode/adapter.go` | Modified | Delegate to GenericAdapter (~20 lines) |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Subtle behavioral change in error messages | Low | Keep per-adapter `DetectErrContext` string; verify with `go test ./...` |
| Breaking adapter registration | Low | Each package still exports `Adapter` type satisfying `adapters.Adapter`; register.go unchanged |
| Future opencode divergence | Low | Explicitly excluded from scope; opencode keeps its own implementation |

## Rollback Plan

Revert the single refactor commit. Each adapter package remains self-contained after revert — no cross-package dependencies introduced beyond the new `generic.go` file.

## Dependencies

- None — pure internal refactor, no new dependencies

## Success Criteria

- [ ] `go test ./...` passes with zero failures
- [ ] `go build ./...` succeeds
- [ ] Net line reduction ≥ 700 lines across the 7 adapter packages
- [ ] Each refactored adapter package ≤ 30 lines
- [ ] No behavioral changes detectable by existing tests
