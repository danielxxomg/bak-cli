# Tasks: CI Hardening (v1.2.1)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1100–1250 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (lint) → PR 2 (ci+helper) → PR 3 (registry+DI tests) → PR 4 (FS tests) |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | Fix 7 lint violations | PR 1 | ~20 lines, trivial review, standalone |
| 2 | macOS CI fix + configtest helper | PR 2 | ~75 lines, depends on nothing; tests green on 3-OS |
| 3 | Registry injection + pure DI action tests | PR 3 | ~400 lines; includes list_cloud prod change + 4 test files (no FS) |
| 4 | FS-dependent action tests | PR 4 | ~450 lines; 3 test files using configtest.SetConfigHome + t.TempDir |

## Phase 1: Lint Fixes

- [x] 1.1 `cmd/export_test.go:38` — Add `return` after `t.Error` to fix SA5011 nil deref
- [x] 1.2 `cmd/pick.go:109` — Replace `b.WriteString(fmt.Sprintf(...))` with `fmt.Fprintf(&b, ...)` (QF1012)
- [x] 1.3 `cmd/wizard.go:283` — Same Fprintf fix (QF1012)
- [x] 1.4 `cmd/wizard.go:323` — Same Fprintf fix (QF1012)
- [x] 1.5 `internal/cloud/pack_test.go:130` — Apply De Morgan simplification in boolean expr (QF1001)
- [x] 1.6 `internal/config/migration_test.go:142` — Remove empty branch (SA9003)
- [x] 1.7 `internal/schedule/scheduler_unix_test.go:11` — Fix interface nil comparison with type assertion (SA4023)
- [x] 1.8 Verify: `go build ./...` + `go vet ./...` + `go test ./...` passes (golangci-lint unavailable)

## Phase 2: macOS CI Fix

- [ ] 2.1 Create `internal/config/testutil/testutil.go` — `package configtest`, export `SetConfigHome(t testing.TB, dir)` that sets HOME (macOS), XDG_CONFIG_HOME (Linux), APPDATA+USERPROFILE (Windows) via `t.Setenv`
- [ ] 2.2 Update `internal/config/config_test.go` — Replace ad-hoc `t.Setenv("XDG_CONFIG_HOME", ...)` with `configtest.SetConfigHome(t, dir)`
- [ ] 2.3 Update other `internal/config/*_test.go` files that set config env vars to use `configtest.SetConfigHome`
- [ ] 2.4 Update `tests/e2e/e2e_test.go` `setupEnv` — Set HOME alongside XDG_CONFIG_HOME so macOS resolves correctly
- [ ] 2.5 Update `testdata/e2e/profile_create_list.txtar` — Add macOS path expectation (`$HOME/Library/Application Support/bak/`)
- [ ] 2.6 Verify: `go test ./internal/config/... ./tests/e2e/...` passes on all 3 OS

## Phase 3: Registry Injection + DI Action Tests

- [ ] 3.1 `internal/actions/list_cloud.go` — Add `RegistryFactory func() *cloud.ProviderRegistry` field to `ListCloudAction`; in `Run()`, use factory if non-nil, else default `cloud.NewProviderRegistry()`
- [ ] 3.2 `cmd/list.go` — Wire `RegistryFactory` in cobra RunE (can be nil for default behavior)
- [ ] 3.3 Create `internal/actions/list_cloud_test.go` — Table-driven: MockProvider via RegistryFactory, test list output, empty state, provider-not-found error
- [ ] 3.4 Create `internal/actions/login_interactive_test.go` — Table-driven: mock ConfigLoader + mock Wizard, test happy path + auth error + cancel
- [ ] 3.5 Create `internal/actions/undo_test.go` — Table-driven: inject HomeDir/IsRepo/UndoFn, test success + no-repo + revert-fail
- [ ] 3.6 Create `internal/actions/schedule_test.go` — Table-driven: mock Scheduler via NewScheduler field, test Create/List/Remove paths
- [ ] 3.7 Verify: `go test ./internal/actions/...` passes, each new file has ≥80% coverage

## Phase 4: FS Action Tests

- [ ] 4.1 Create `internal/actions/verify_backup_test.go` — Use `configtest.SetConfigHome` + `t.TempDir()`, create fixture files with SHA-256 checksums, test pass + checksum mismatch + missing manifest
- [ ] 4.2 Create `internal/actions/pick_backup_test.go` — Use `configtest.SetConfigHome` + `t.TempDir()`, create multiple backup dirs with manifests, test selection by criteria + empty dir error
- [ ] 4.3 Create `internal/actions/diff_backups_test.go` — Use `configtest.SetConfigHome` + `t.TempDir()`, create two backup fixtures with different manifests, test diff output + identical + missing backup error
- [ ] 4.4 Verify: `go test ./internal/actions/...` passes, all 3 new files ≥80% coverage
- [ ] 4.5 Verify: `go test ./...` passes full suite, no regressions

## Phase 5: Final Verification

- [ ] 5.1 Run `golangci-lint run` — exits 0
- [ ] 5.2 Run `go test ./...` — all pass
- [ ] 5.3 Run GGA pre-commit — passes without `--no-verify`
- [ ] 5.4 Confirm no behavior changes: all pre-existing tests still pass unchanged
