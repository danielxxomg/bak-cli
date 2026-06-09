# Tasks: ci-fix (CI Blocking Issues)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~100–150 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |

Decision needed before apply: No
Chained PRs recommended: No
400-line budget risk: Low

## Fixes

- [x] Fix 1: errcheck violations in backup.go (6 violations — unchecked `fmt.Fprintf`)
- [x] Fix 2: errcheck violations in adapters/util.go (3 violations — unchecked `Close()`)
- [x] Fix 3: goimports formatting (all Go files)
- [x] Fix 4: Cross-platform test — DI approach (StatFn injection for GenericAdapter)
- [x] Fix 5: Commit `fix: resolve CI lint violations and cross-platform test flakiness`

## Verification

- [x] `golangci-lint run ./...` — zero violations
- [x] `go test ./... -count=1` — 1194 passed
- [x] `go vet ./...` — clean
- [x] `go build ./...` — success
