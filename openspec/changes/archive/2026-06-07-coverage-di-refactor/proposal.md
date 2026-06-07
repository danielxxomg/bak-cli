# Proposal: Coverage DI Refactor

## Intent

CI coverage gate fails at 68.6% (threshold: 80%). Three packages drag the average down: `internal/adapters/` (41%), `internal/actions/` (60%), `cmd/` (48%). Root cause: direct OS calls (`os.Open`, `os.UserHomeDir`, `os.Stdin`, `bubbletea.Program.Run()`) block test isolation. Targeted dependency injection will make these packages testable without changing external behavior.

## Scope

### In Scope
- DI refactoring of `internal/adapters/` — inject `homeDir` into `LoadYAMLAdapters`
- DI refactoring of `internal/actions/` — route `restoreFile()` through injected `FileSystem`, inject `HostnameFunc` and `ProviderFactory`
- DI refactoring of `cmd/` — extract business logic from cobra `RunE` into testable functions
- E2E guardrail: backup→restore round-trip with real files (PR0)
- CI fixes: lint version mismatch, `parseSchtasksCSV` build tag, rate-limit workaround
- Restore coverage threshold to 80% after PR3

### Out of Scope
- New features (encryption, content diff, new adapters)
- Multi-OS install options (scoop, paru, brew) — separate change
- Full DI overhaul (Strategy 3)
- Coverage for packages already ≥80%

## Capabilities

### New Capabilities
None — this is a refactoring change, no new user-facing behavior.

### Modified Capabilities
None at spec level — all external behavior (CLI commands, manifest format, restore semantics) remains identical. Changes are internal wiring only.

## Approach

**Strategy 2: Targeted DI Refactoring** — 5 chained PRs on `feature/coverage-refactor`:

| PR | Phase | Key Changes | Expected Coverage |
|----|-------|-------------|-------------------|
| PR0 | Guardrail | E2E backup→restore test + threshold→70% | No change |
| PR1 | Adapters | `yaml_test.go`, refactor `LoadYAMLAdapters(homeDir)`, `register_test.go` | adapters/ 41%→~80% |
| PR2 | Actions | `ProviderFactory` interface, `restoreFile`→`a.FS`, `HostnameFunc`, tests | actions/ 60%→~85% |
| PR3 | Cmd | Extract `RunE` logic to testable funcs, tests, threshold→80% | cmd/ 48%→~75-80%, total≥80% |
| PR4 | CI Fixes | Lint Go version, `parseSchtasksCSV` build tag, rate-limit fix | No coverage change |

**Constraints**: Strict TDD (`go test ./...`), hand-rolled test doubles, `t.TempDir()` isolation, GGA pre-commit.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/adapters/yaml.go` | Modified | `LoadYAMLAdapters` accepts `homeDir string` parameter |
| `internal/adapters/register.go` | Modified | Registration uses injected home dir |
| `internal/actions/restore.go` | Modified | `restoreFile()` uses `a.FS` instead of `os.Open`/`os.Create` |
| `internal/actions/push.go` | Modified | Uses `ProviderFactory` for provider creation |
| `internal/actions/pull.go` | Modified | Uses `ProviderFactory` for provider creation |
| `internal/actions/actions.go` | Modified | `HostnameFunc` field added |
| `cmd/*.go` | Modified | `RunE` delegates to extracted functions in `internal/actions/` |
| `.github/workflows/` | Modified | Lint Go version, build tags, rate-limit workaround |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| `restoreFile()` refactor breaks critical path | Medium | E2E guardrail in PR0 catches regressions; strict TDD |
| `LoadYAMLAdapters` signature change ripples to callers | Low | Only 2 call sites (cmd/backup.go, register/register.go) — update in same PR |
| `rootCmd.Execute()` global state leaks between cmd tests | Medium | Reset `rootCmd` state in `TestMain` or per-test setup |

## Rollback Plan

Each PR in the chain reverts independently via `git revert`. PR0's threshold change is the safest rollback point — if later PRs fail, revert them and keep the guardrail tests at 70% threshold. No data migration or schema changes to undo.

## Dependencies

- Go 1.24+ toolchain (already in go.mod)
- No new external dependencies

## Success Criteria

- [ ] `go test -cover ./...` reports ≥80% total coverage
- [ ] `internal/adapters/` coverage ≥80%
- [ ] `internal/actions/` coverage ≥80%
- [ ] `cmd/` coverage ≥75%
- [ ] CI passes on all 3 OS (Windows, macOS, Linux)
- [ ] E2E backup→restore round-trip test passes
- [ ] Zero GGA violations against updated AGENTS.md
