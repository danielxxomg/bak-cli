# Tasks: Test Hardening — Fix Misleading Testscripts and Close Coverage Gaps

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~280–320 |
| 400-line budget risk | Medium |
| Chained PRs recommended | No |
| Suggested split | single PR |
| Delivery strategy | ask-on-risk |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Medium

## Phase 1: Fix Misleading Testscripts

- [x] 1.1 Rewrite `tests/e2e/testdata/backup_restore_roundtrip.txtar`: after `exec bak backup --preset quick`, capture backup ID, add `exec bak restore $ID --dry-run` with stdout assertion, then `exec bak restore $ID --force` with `exists` checks for restored files
- [x] 1.2 Rewrite `tests/e2e/testdata/diff_two_backups.txtar`: create first backup, modify fixture file (e.g. `cp` new content over `opencode.json`), create second backup, add `exec bak diff $ID1 $ID2` with stdout assertion showing file differences
- [x] 1.3 Rewrite `tests/e2e/testdata/backup_verify_roundtrip.txtar`: after `exec bak backup --preset quick`, capture ID, add `exec bak verify $ID` with stdout assertion for checksum/OK output; add tampered-file scenario (modify a backed-up file, re-verify, assert non-zero exit)
- [x] 1.4 Run `go test ./tests/e2e/ -run TestE2E -count=1 -v` to verify all three testscripts pass

## Phase 2: Schedule Happy Path (cmd-level)

- [x] 2.1 Add `NewScheduler func() schedule.Scheduler` field to `cmdDeps` in `cmd/deps.go`; nil means production default (backward compatible)
- [x] 2.2 Update `cmd/schedule.go`: pass `deps.NewScheduler` through to `actions.ScheduleAction.NewScheduler` in all three `runSchedule*WithDeps` functions
- [x] 2.3 Add `TestScheduleCreate_HappyPath` in `cmd/schedule_test.go`: inject mock scheduler via `cmdDeps`, assert `Create` called with correct args, stdout contains "Schedule created"
- [x] 2.4 Add `TestScheduleList_HappyPath` in `cmd/schedule_test.go`: mock scheduler returns two entries, assert stdout contains profile names and intervals
- [x] 2.5 Add `TestScheduleRemove_HappyPath` in `cmd/schedule_test.go`: inject mock scheduler, assert `Remove` called, stdout contains "Schedule removed"
- [x] 2.6 Run `go test ./cmd/ -run TestSchedule -count=1 -v` to verify

## Phase 3: Cloud Sync Integration Test

- [x] 3.1 Create `internal/actions/cloud_sync_test.go` with `TestCloudSync_PushPullRoundTrip`: use `MockProvider` + `MockProviderFactory` from existing test helpers; push a real backup dir, capture archive bytes, pull back, untar, verify file content matches originals
- [x] 3.2 Add `TestCloudSync_PushInvalidToken` in `cloud_sync_test.go`: mock provider returns 401-style error, assert error contains "unauthorized" or "401"
- [x] 3.3 Add `TestCloudSync_Pull_NotFound` in `cloud_sync_test.go`: mock provider returns not-found error on Pull, assert error contains "not found" or "404"
- [x] 3.4 Run `go test ./internal/actions/ -run TestCloudSync -count=1 -v` to verify

## Phase 4: TUI Binary Smoke Test

- [x] 4.1 Create `tests/e2e/smoke_test.go` with `TestBinaryHelp`: build binary via `go build`, exec `bak --help`, assert exit 0 and stdout contains "Backup and restore"
- [x] 4.2 Add `TestBinaryNoArgs` in `smoke_test.go`: exec `bak` with piped stdin (non-TTY), assert exit 0 and help output shown
- [x] 4.3 Add `TestBinaryUnknownCommand` in `smoke_test.go`: exec `bak nonexistent`, assert exit 1 and stderr contains error
- [x] 4.4 Run `go test ./tests/e2e/ -run TestBinary -count=1 -v` to verify

## Phase 5: Quality Gates

- [x] 5.1 Run `go test -race ./...` — all tests pass with race detector
- [x] 5.2 Run `go vet ./...` — no issues
- [x] 5.3 Run `golangci-lint run` — no new lint violations
