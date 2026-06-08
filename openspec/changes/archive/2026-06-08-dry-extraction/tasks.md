# Tasks: dry-extraction

## Review Workload Forecast

- Decision needed before apply: No
- Chained PRs recommended: No
- 400-line budget risk: Medium (estimated ~500 lines changed, but mostly deletions)

## Phase 1: Shared adapter utilities

- [x] 1.1 Create `internal/adapters/util.go` with exported `CopyFile` and `FileHash`
- [x] 1.2 Create `internal/adapters/util_test.go` with moved tests from `yaml_test.go`
- [x] 1.3 Update `internal/adapters/yaml.go` to use shared `CopyFile`/`FileHash`, remove local implementations
- [x] 1.4 Update `internal/adapters/yaml_test.go` to remove moved tests
- [x] 1.5 Update `internal/adapters/opencode/adapter.go` to call `adapters.CopyFile`/`adapters.FileHash`, remove local dupes
- [x] 1.6 Update `internal/adapters/opencode/adapter_test.go` to reference `adapters.FileHash`
- [x] 1.7 Update `internal/adapters/cursor/adapter.go` to call shared functions, remove dupes
- [x] 1.8 Update `internal/adapters/cursor/adapter_test.go` to reference `adapters.FileHash`
- [x] 1.9 Update `internal/adapters/pidev/adapter.go` and `adapter_test.go`
- [x] 1.10 Update `internal/adapters/windsurf/adapter.go` and `adapter_test.go`
- [x] 1.11 Update `internal/adapters/claudecode/adapter.go` and `adapter_test.go`
- [x] 1.12 Update `internal/adapters/kiro/adapter.go` and `adapter_test.go`
- [x] 1.13 Update `internal/adapters/kilocode/adapter.go` and `adapter_test.go`
- [x] 1.14 Update `internal/adapters/codex/adapter.go` and `adapter_test.go`

## Phase 2: Shared HTTP helper

- [x] 2.1 Create `internal/cloud/httputil.go` with `newRequest`, `doRequest` and `formatAPIError` helpers
- [x] 2.2 Update `internal/cloud/github_repo.go` to use shared helpers
- [x] 2.3 Update `internal/cloud/gitea.go` to use shared helpers

## Phase 3: Verification

- [x] 3.1 Run `go test ./...` — 1113 tests pass
- [x] 3.2 Run `go vet ./...` — no issues
- [x] 3.3 Run `golangci-lint run` — minor goimports issues (formatting only)
- [x] 3.4 Verify no unused functions left behind — grep confirms zero `func copyFile` / `func fileHash` remain
