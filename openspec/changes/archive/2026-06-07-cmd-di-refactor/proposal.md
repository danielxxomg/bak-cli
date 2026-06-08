# Proposal: cmd-di-refactor

## Intent

Achieve full test isolation for the `cmd/` package through dependency injection. Currently 7 tests fail on Linux CI because command functions call `config.Load()` which reads the real filesystem via `os.UserConfigDir()`. This refactor injects all dependencies through a `CmdDeps` struct, enabling cross-platform test reliability without changing user-facing behavior.

## Scope

### In Scope
- Create `cmd/deps.go` with `CmdDeps` struct (ConfigLoader, Stdout, Stderr, Stdin)
- Refactor all 13 cmd files to use `runXWithDeps` pattern
- Fix `canonicalPath()` in diff.go to use `strings.ReplaceAll` (Phase 1 from exploration)
- Create test helper `setupTestEnv(t)` using injected deps
- Update all failing tests to use injected deps
- Verify in Docker (task test:linux) and locally (task test)

### Out of Scope
- New features or functionality
- Coverage improvement (this is about test isolation, not coverage)
- Changing AGENTS.md rules or conventions
- Modifying internal packages beyond diff.go canonicalPath fix

## Capabilities

### New Capabilities
- `cmd-dependency-injection`: Injectable dependency structure for all command functions, enabling test isolation and cross-platform reliability

### Modified Capabilities
None (pure refactor — no spec-level behavior changes)

## Approach

Inject `CmdDeps` struct into all command functions using the wrapper pattern:
- `runX(cmd, args)` → public entry point (unchanged signature)
- `runXWithDeps(cmd, args, deps)` → implementation with injected dependencies

Dependencies injected:
- ConfigLoader interface (replaces direct `config.Load()` calls)
- Stdout, Stderr, Stdin writers (replaces direct os.Stdout/Stderr/Stdin)

Phase 1 fix: Replace `filepath.ToSlash()` with `strings.ReplaceAll()` in `canonicalPath()` to eliminate OS-dependent path separators.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/deps.go` | New | CmdDeps struct and constructor (~30 lines) |
| `cmd/*.go` (13 files) | Modified | Refactor to runXWithDeps pattern |
| `internal/diff/diff.go` | Modified | Fix canonicalPath() OS-dependent bug |
| `cmd/*_test.go` (7-10 files) | Modified | Update tests to use injected deps |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Breaking user-facing behavior | Low | No signature changes to public functions; comprehensive test suite validates behavior |
| Incomplete DI coverage | Medium | GGA pre-commit validation; Docker test verification |
| Test helper complexity | Low | Follow existing patterns in AGENTS.md; keep setupTestEnv minimal |

## Rollback Plan

Revert the feature branch `feature/cmd-di-refactor`. All changes are in isolated files with no external dependencies. No database migrations or config changes required.

## Dependencies

- Existing test infrastructure (task test, task test:linux)
- AGENTS.md conventions for DI patterns and test doubles
- Exploration findings (topic: sdd/bak-cli/cross-platform-test-isolation)

## Success Criteria

- [ ] All tests pass locally: `task test` (Windows)
- [ ] All tests pass in Docker: `task test:linux` (Linux CI)
- [ ] Zero user-facing behavior changes (verified by existing test suite)
- [ ] No increase in test failures or flakiness
- [ ] Code follows AGENTS.md DI patterns and conventional commits
