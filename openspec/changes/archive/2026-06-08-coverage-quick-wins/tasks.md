# Tasks: coverage-quick-wins

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~200 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Delivery strategy | single-pr |
| Decision needed before apply | No |

## Phase 1: actions/ tests

- [x] 1.1 Add `TestParseCSV` to `internal/actions/profile_test.go` — 7 table-driven cases
- [x] 1.2 Add `TestProfileValidateForCreation` to `internal/actions/profile_test.go` — 4 table-driven cases

## Phase 2: Discovery (5 of 7 already tested)

- [x] 2.1 Verify `FormatBackupIDError` already covered in export_test.go
- [x] 2.2 Verify `CountByStatus` already covered in dryrun_test.go
- [x] 2.3 Verify `ResolveBackup` already covered in restore_test.go
- [x] 2.4 Verify `formatSize` already covered in backup_test.go and list_test.go
- [x] 2.5 Verify `RunExport` error paths already covered in export_test.go

## Phase 3: Verification

- [x] 3.1 Run `go test ./...` — 1128 passed
- [x] 3.2 Run `go vet ./...` — clean
