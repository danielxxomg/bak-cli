# Proposal: CI Hardening (v1.2.1)

## Intent

CI is broken on v1.2.0: 7 lint errors block merge, macOS tests fail due to `os.UserConfigDir()` ignoring `XDG_CONFIG_HOME`, and 8 action files from the verify agent have zero test coverage. This patch release fixes all three to restore green CI across all 3 OS targets.

## Scope

### In Scope
- Fix 7 golangci-lint violations (SA5011, QF1012×3, QF1001, SA9003, SA4023)
- Fix macOS CI failures (2 tests) by resolving XDG_CONFIG_HOME path mismatch
- Add unit tests for 7 untested action files in `internal/actions/`
- Ensure GGA pre-commit passes with no `--no-verify` bypass

### Out of Scope
- New features or commands
- Refactoring existing tested code
- E2E test expansion (deferred to next cycle)
- Coverage threshold increase (stays at 80%)

## Capabilities

### New Capabilities
None — this is a quality/CI fix, no new user-facing capabilities.

### Modified Capabilities
- `engineering-quality`: Fix lint violations, add tests for untested actions, resolve macOS CI failures. Requirements affected: CI pipeline fixes, action DI wiring, adapter testability.

## Approach

**Area 1 — Lint Fixes (7 issues):**
Direct fixes per linter recommendation:
- SA5011: Add nil check before dereference in `cmd/export_test.go:38`
- QF1012 (×3): Replace `fmt.Fprint(w, fmt.Sprintf(...))` with `fmt.Fprintf(w, ...)` in `cmd/pick.go:109`, `cmd/wizard.go:283,323`
- QF1001: Apply De Morgan simplification in `internal/cloud/pack_test.go:130`
- SA9003: Remove empty branch or add comment explaining intent in `internal/config/migration_test.go:142`
- SA4023: Fix interface comparison — use `== nil` on concrete type or check interface properly in `internal/schedule/scheduler_unix_test.go:11`

**Area 2 — macOS CI Fixes (2 failures):**
Root cause: `os.UserConfigDir()` returns `~/Library/Application Support` on macOS, ignoring `XDG_CONFIG_HOME`. Tests set `XDG_CONFIG_HOME` which works on Linux but not macOS.

Decision needed:
- Option A: Dual-write — set both `XDG_CONFIG_HOME` and `HOME` in tests, let `os.UserConfigDir()` resolve naturally per OS
- Option B: Env var helper — create `configDir()` wrapper that checks `XDG_CONFIG_HOME` first on all platforms (overrides OS default)
- Option C: Refactor `ConfigDir()` — inject config directory via struct field, tests inject `t.TempDir()`

Recommended: **Option A** — minimal change, respects OS conventions, no API changes.

Files: `internal/config/*_test.go`, `testdata/e2e/profile_create_list.txtar`

**Area 3 — Actions Test Coverage (7 files):**
Untested files: `login_interactive.go`, `list_cloud.go`, `diff_backups.go`, `verify_backup.go`, `pick_backup.go`, `undo.go`, `schedule.go`

Decision needed:
- `list_cloud.go`: Inject registry interface or keep hardcoded? Recommended: **Inject** — aligns with existing DI pattern in spec.
- FS-dependent actions: Use `t.TempDir()` fixtures with real FS operations. Recommended: **t.TempDir()** — matches existing test patterns, no mock FS complexity.

Each action gets table-driven tests covering happy path + error paths per AGENTS.md rules.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/export_test.go` | Modified | Add nil check (SA5011) |
| `cmd/pick.go` | Modified | Replace Fprint+Sprintf with Fprintf (QF1012) |
| `cmd/wizard.go` | Modified | Replace Fprint+Sprintf with Fprintf (QF1012) ×2 |
| `internal/cloud/pack_test.go` | Modified | Apply De Morgan (QF1001) |
| `internal/config/migration_test.go` | Modified | Remove empty branch (SA9003) |
| `internal/schedule/scheduler_unix_test.go` | Modified | Fix interface comparison (SA4023) |
| `internal/config/*_test.go` | Modified | macOS path fix — set HOME + XDG_CONFIG_HOME |
| `testdata/e2e/profile_create_list.txtar` | Modified | macOS config path expectation |
| `internal/actions/login_interactive_test.go` | New | Unit tests for interactive login |
| `internal/actions/list_cloud_test.go` | New | Unit tests with injected registry |
| `internal/actions/diff_backups_test.go` | New | Unit tests with t.TempDir() |
| `internal/actions/verify_backup_test.go` | New | Unit tests with t.TempDir() |
| `internal/actions/pick_backup_test.go` | New | Unit tests with t.TempDir() |
| `internal/actions/undo_test.go` | New | Unit tests with t.TempDir() |
| `internal/actions/schedule_test.go` | New | Unit tests with t.TempDir() |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| macOS fix breaks Linux/Windows tests | Low | Test on all 3 OS in CI before merge; Option A respects OS conventions |
| New action tests reveal bugs | Medium | Fix bugs in same PR — this is the point of adding tests |
| `list_cloud.go` DI refactor breaks callers | Low | Inject interface, keep existing constructor signature with default |
| Lint fixes change behavior | Low | SA5011 adds nil check (defensive), QF1012 is cosmetic, others are test-only |

## Rollback Plan

Revert the single PR commit: `git revert <commit-sha>`. No migrations, no data changes, no API breaks. All changes are test/lint fixes — safe to revert if CI still fails post-merge.

## Dependencies

- Go 1.26+ (already in go.mod)
- golangci-lint (already in CI)
- No new dependencies

## Success Criteria

- [ ] `golangci-lint run` exits 0 with no warnings
- [ ] `go test ./...` passes on Ubuntu, macOS, Windows in CI
- [ ] All 7 untested action files have ≥80% coverage
- [ ] GGA pre-commit passes without `--no-verify`
- [ ] `task test:linux` (Docker) passes
- [ ] No behavior changes to existing commands (verified by existing tests)
