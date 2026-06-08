# Tasks: Generic Adapter Base Struct

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~270 add (commit 1) + ~195 each (commits 2–8) = ~1635 total, each commit ≤270 |
| 400-line budget risk | Low (per commit) |
| Chained PRs recommended | Yes (8 commits map to 8 reviewable work units) |
| Suggested split | 8 stacked commits; group into PRs only if maintainer requests |
| Delivery strategy | ask-always |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | GenericAdapter + tests | Commit 1 | ~270 lines; foundation for all migrations |
| 2–8 | One adapter migration each | Commits 2–8 | ~195 lines each; simplest-first order |

## Phase 1: Foundation (Commit 1)

**TDD: Write `generic_test.go` FIRST (RED), then `generic.go` (GREEN), then refactor.**

- [x] 1.1 **RED** — Create `internal/adapters/generic_test.go` with table-driven tests: Detect (dir exists, missing, not-dir, error), ListItems (all categories, empty, unknown cat), Backup/Restore (file copy, dir mkdir, copy error). Use `t.TempDir()`.
- [x] 1.2 **GREEN** — Create `internal/adapters/generic.go`: export `CategoryDir` struct (`SubPath string`, `IsDir bool`), define `GenericAdapter` struct (`AdapterName`, `ConfigRelPath`, `Categories map[string]CategoryDir`, `DetectErrContext`), implement all 5 `Adapter` interface methods. Add `var _ Adapter = (*GenericAdapter)(nil)`. Extract `scanDir`/`scanRootFiles` as unexported package-level functions. Use `filepath.ToSlash` for RelPath (preserve existing behavior).
- [x] 1.3 **VERIFY** — Run `go test ./internal/adapters/...` — all new tests pass. Run `go test ./...` — zero regressions. Run `go build ./...`.
- [x] 1.4 **COMMIT** — `refactor: add GenericAdapter base struct`

## Phase 2: Adapter Migrations (Commits 2–8)

**Per-adapter pattern** (repeat for each):
1. Replace `adapter.go` body: keep package doc, imports, constants, `type Adapter struct{}`, compile-time check. Add `var base = adapters.GenericAdapter{...}`. Replace all methods with 5 one-liner delegations to `base`.
2. Delete all `scanDir`, `scanRootFiles`, `categoryDir` type — now in `generic.go`.
3. Run `go test ./...` — zero failures (existing tests must pass unmodified).

- [ ] 2.1 **Migrate codex** — `internal/adapters/codex/adapter.go`: adapterName=`"codex"`, configRelPath=`".codex"`, categories={config, instructions}, DetectErrContext=`"stat codex config dir"`. Verify: `go test ./internal/adapters/codex/... ./...`
- [ ] 2.2 **COMMIT** — `refactor: migrate codex adapter to GenericAdapter`
- [ ] 2.3 **Migrate kiro** — `internal/adapters/kiro/adapter.go`: adapterName=`"kiro"`, configRelPath=`".kiro"`, categories={config, hooks}, DetectErrContext=`"stat kiro config dir"`. Verify: `go test ./...`
- [ ] 2.4 **COMMIT** — `refactor: migrate kiro adapter to GenericAdapter`
- [ ] 2.5 **Migrate kilocode** — `internal/adapters/kilocode/adapter.go`: adapterName=`"kilocode"`, configRelPath=`".kilocode"`, categories={config, rules}, DetectErrContext=`"stat kilocode config dir"`. Verify: `go test ./...`
- [ ] 2.6 **COMMIT** — `refactor: migrate kilocode adapter to GenericAdapter`
- [ ] 2.7 **Migrate pidev** — `internal/adapters/pidev/adapter.go`: adapterName=`"pidev"`, configRelPath=`".pi"`, categories={config, agents}, DetectErrContext=`"stat pidev config dir"`. Verify: `go test ./...`
- [ ] 2.8 **COMMIT** — `refactor: migrate pidev adapter to GenericAdapter`
- [ ] 2.9 **Migrate windsurf** — `internal/adapters/windsurf/adapter.go`: adapterName=`"windsurf"`, configRelPath=`".codeium/windsurf"` (nested path), categories={config, rules}, DetectErrContext=`"stat windsurf config dir"`. Verify: `go test ./...`
- [ ] 2.10 **COMMIT** — `refactor: migrate windsurf adapter to GenericAdapter`
- [ ] 2.11 **Migrate cursor** — `internal/adapters/cursor/adapter.go`: adapterName=`"cursor"`, configRelPath=`".cursor"`, categories={config, extensions}, DetectErrContext=`"stat cursor config dir"`. Verify: `go test ./...`
- [ ] 2.12 **COMMIT** — `refactor: migrate cursor adapter to GenericAdapter`
- [ ] 2.13 **Migrate claudecode** — `internal/adapters/claudecode/adapter.go`: adapterName=`"claude-code"`, configRelPath=`".claude"`, categories={config, skills, commands} (3 categories), DetectErrContext=`"stat claude-code config dir"`. Verify: `go test ./...`
- [ ] 2.14 **COMMIT** — `refactor: migrate claudecode adapter to GenericAdapter`

## Phase 3: Final Verification

- [ ] 3.1 Run `go test ./...` — zero failures across all packages.
- [ ] 3.2 Run `go vet ./...` — clean.
- [ ] 3.3 Verify `register/register.go` is **unchanged** — still uses `&codex.Adapter{}` etc.
- [ ] 3.4 Verify no test files were modified (`git diff --name-only | grep _test.go` should be empty except `generic_test.go`).
- [ ] 3.5 Verify net line reduction ≥700 lines (`git diff --stat main`).
