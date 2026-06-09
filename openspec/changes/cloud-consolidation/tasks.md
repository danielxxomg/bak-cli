# Tasks: Cloud Provider Code Consolidation

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~280–340 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR (3 atomic commits) |
| Delivery strategy | single-pr |
| Chain strategy | N/A |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: N/A
400-line budget risk: Low

## Phase 1: Migrate gist.go to httputil.go

- [x] 1.1 Replace `gistAPI()` with `newRequest()`+`doRequest()`+`formatAPIError()` in `CreateGist`
- [x] 1.2 Replace `gistAPI()` in `UpdateGist`, `GetGist`, `DeleteGist`
- [x] 1.3 Update `github_gist.go` `List()` to use shared helpers instead of `gistAPI()`
- [x] 1.4 Remove `apiError` type and `gistAPI()` function from `gist.go`
- [x] 1.5 Verify all gist tests pass after migration

## Phase 2: Extract shared content API types

- [x] 2.1 Create `internal/cloud/content_types.go` with `contentRequest`, `contentResponse`, `contentFile`
- [x] 2.2 Update `github_repo.go` to use shared types; remove `githubContent*` prefixed types
- [x] 2.3 Update `gitea.go` to use shared types; remove `giteaContent*` prefixed types
- [x] 2.4 Update test files — rename `githubContent*` / `giteaContent*` references to `content*`
- [x] 2.5 All github_repo and gitea tests must pass

## Phase 3: Extract shared getFileSHA + writeFile helpers

- [x] 3.1 Add `getFileSHA(client, token, url)` function to `content_types.go`
- [x] 3.2 Add `writeContentFile(client, token, method, accept, url, req)` to `content_types.go`
- [x] 3.3 Update `github_repo.go` `getFileSHA`/`putFile` to delegate to shared helpers
- [x] 3.4 Update `gitea.go` `getFileSHA`/`writeFile`/`postFile`/`putFile` to delegate to shared helpers
- [x] 3.5 Write table-driven unit tests for `getFileSHA` and `writeContentFile`

## Phase 4: Verify

- [x] 4.1 Run `go test ./...` — zero failures (1173 passed)
- [x] 4.2 Run `go vet ./...` — clean
