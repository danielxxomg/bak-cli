# Proposal: dry-extraction

## Intent

Extract duplicated helper functions into shared locations to eliminate DRY violations across the `internal/adapters/` and `internal/cloud/` packages.

## Scope

### Phase 1: Shared adapter utilities (`internal/adapters/util.go`)

`copyFile()` and `fileHash()` are duplicated across **9 locations**:
- `internal/adapters/yaml.go` (package `adapters`)
- `internal/adapters/opencode/adapter.go` (package `opencode`)
- `internal/adapters/cursor/adapter.go` (package `cursor`)
- `internal/adapters/pidev/adapter.go` (package `pidev`)
- `internal/adapters/windsurf/adapter.go` (package `windsurf`)
- `internal/adapters/claudecode/adapter.go` (package `claudecode`)
- `internal/adapters/kiro/adapter.go` (package `kiro`)
- `internal/adapters/kilocode/adapter.go` (package `kilocode`)
- `internal/adapters/codex/adapter.go` (package `codex`)

Extract `CopyFile(src, dst string) error` and `FileHash(path string) (hash string, size int64, err error)` into `internal/adapters/util.go` (package `adapters`) as exported functions. Update all callers and tests. Remove the duplicated implementations.

### Phase 2: Shared HTTP helper (`internal/cloud/httputil.go`)

HTTP request boilerplate (build URL → create request → set headers → execute → read body → check status → parse error) is repeated across `github_repo.go` and `gitea.go`. Extract a shared `doRequest()` + `apiError()` helper into `internal/cloud/httputil.go`.

### Out of scope
- No behavioral changes
- No new features
- No new dependencies

## Approach

1. Create shared utility files with the best version of each function (proper error wrapping per AGENTS.md)
2. Update all callers to use the shared functions
3. Remove duplicated implementations
4. Run `go test ./...` to verify no regressions
5. Run `golangci-lint run` and `go vet ./...`

## Rollback Plan

If tests fail after extraction, revert the change file by file. Each file change is self-contained and can be reverted independently.

## Risks
- **Low**: Pure refactoring, no behavioral changes
- Tests in sub-package adapter test files that call `fileHash()` directly will need to be updated to call `adapters.FileHash()`
