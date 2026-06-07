# Tasks — Cycle D: QA v1.1.0 (E2E testscript + fuzzing + benchmarks + GoReleaser)

## Phase 1: Testscript E2E

- [x] 1.1 Add `rogpeppe/go-internal` dependency to go.mod
- [x] 1.2 Create `tests/e2e/e2e_test.go` with setupEnv that builds bak binary
- [x] 1.3 Create `tests/e2e/testdata/backup_verify_roundtrip.txtar`
- [x] 1.4 Create `tests/e2e/testdata/backup_restore_roundtrip.txtar`
- [x] 1.5 Create `tests/e2e/testdata/diff_two_backups.txtar`
- [x] 1.6 Create `tests/e2e/testdata/profile_create_list.txtar`
- [x] 1.7 Create `tests/e2e/testdata/schedule_create_list.txtar`

## Phase 2: Directed Fuzzing

- [x] 2.1 Create `internal/presets/fuzz_test.go` — FuzzLoadFromDir
- [x] 2.2 Create `internal/manifest/fuzz_test.go` — FuzzLoad
- [x] 2.3 Create `internal/adapters/fuzz_test.go` — FuzzLoadYAMLAdapters

## Phase 3: Benchmarks

- [x] 3.1 Create `internal/backup/bench_test.go` — BenchmarkEngine_Run
- [x] 3.2 Create `internal/diff/bench_test.go` — BenchmarkCompare
- [x] 3.3 Create `internal/crypto/bench_test.go` — BenchmarkEncrypt, BenchmarkDecrypt

## Phase 4: GoReleaser Validation

- [x] 4.1 Add goreleaser-verify target to Taskfile
- [x] 4.2 Add goreleaser check job to CI workflow

## Review Workload Forecast

- 400-line budget risk: Medium
- Chained PRs recommended: No
- Decision needed before apply: No
- Chain strategy: not applicable (single PR, size within budget)
