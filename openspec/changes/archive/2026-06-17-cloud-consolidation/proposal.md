# Proposal: Cloud Provider Code Consolidation

## Intent

`internal/cloud/` violates the DRY rule (AGENTS.md) in three places:
1. `gist.go:gistAPI()` reimplements HTTP request/response logic that `httputil.go` already provides (`newRequest`, `doRequest`, `formatAPIError`)
2. `github_repo.go` and `gitea.go` define nearly identical types (`contentRequest`, `contentResponse`, `contentFile`)
3. `getFileSHA()` and the write-file pattern are duplicated between GitHub and Gitea providers with >80% shared logic

## Scope

### In Scope
- **Part A**: Migrate `gist.go` to use `httputil.go` helpers — eliminate `gistAPI()` function
- **Part B**: Extract shared content API types into `internal/cloud/content_types.go`
- **Part C**: Extract shared `getFileSHA` and `writeFile` patterns into reusable functions

### Out of Scope
- Changing the `Provider` interface
- Changing public API or exported types (`GistFile`, `Gist`, etc.)
- Refactoring `github_gist.go` provider wrapper (it already delegates correctly)
- Adding new cloud providers

## Capabilities

### New Capabilities
- `cloud-content-types`: Shared types and helper functions for Contents API operations (getFileSHA, writeFile) used by GitHub and Gitea providers

### Modified Capabilities
- `dry-extraction`: gist.go now uses shared HTTP helpers (completes REQ-DRY-003 for gist)

## Approach

- **Part A**: Replace `gistAPI()` body with calls to `newRequest()` + `doRequest()` + `formatAPIError()`. Keep the `apiError` type for backward compat with tests — wrap `formatAPIError` result or adapt.
- **Part B**: Create `content_types.go` with `contentRequest`, `contentResponse`, `contentFile` — remove prefixed duplicates from both files.
- **Part C**: Extract `getFileSHA(client, token, url) -> (sha, error)` and `writeContentFile(client, token, method, url, req) -> error` as package-level functions parameterized by URL and HTTP method.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/cloud/gist.go` | Modified | Remove `gistAPI()`, use `newRequest`/`doRequest`/`formatAPIError` |
| `internal/cloud/github_repo.go` | Modified | Remove local types, use shared `contentRequest`/`contentResponse`/`contentFile` |
| `internal/cloud/gitea.go` | Modified | Remove local types, use shared types + shared helpers |
| `internal/cloud/content_types.go` | New | Shared types and helper functions |
| `internal/cloud/gist_test.go` | Modified | Update tests if `apiError` type changes |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| `apiError` type removal breaks test assertions | Low | Keep `apiError` type, just construct it from `formatAPIError` output |
| Error message format changes break user-facing output | Low | Verify error strings match via test comparison |
| Import cycle from new file | Low | Same package — no new imports needed |
| Gitea vs GitHub API differences in content response | Low | Fields are identical; verified by reading both structs |

## Rollback Plan

Revert the single refactoring commit. All changes are internal — no data migration or config changes needed. `git revert <sha>` restores prior state.

## Dependencies

- None — pure internal refactoring

## Success Criteria

- [ ] `gist.go` has zero direct `http.NewRequest` / `httpClient.Do` calls
- [ ] `githubContentRequest`/`giteaContentRequest` types consolidated into one `contentRequest`
- [ ] `getFileSHA` exists as a single shared function (not duplicated)
- [ ] `go test ./internal/cloud/...` passes with no regressions
- [ ] `go vet ./...` clean
