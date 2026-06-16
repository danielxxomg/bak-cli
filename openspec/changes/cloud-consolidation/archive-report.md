# Archive Report: cloud-consolidation

## Change Summary

Three-part DRY refactoring of `internal/cloud/` to eliminate code duplication across GitHub, Gitea, and Gist providers. Consolidated duplicate HTTP helpers, content API types, and file operation functions into shared package-level utilities.

**Status**: Verified PASS — 17/17 tasks complete, 1173 tests pass, `go vet` clean, 82.6% cloud package coverage.

## Files Created

| File | Description |
|------|-------------|
| `internal/cloud/content_types.go` | Shared types (`contentRequest`, `contentResponse`, `contentFile`) + helpers (`getFileSHA`, `writeContentFile`) |
| `internal/cloud/content_types_test.go` | 6 table-driven tests for shared helpers |

## Files Modified

| File | Description |
|------|-------------|
| `internal/cloud/gist.go` | Removed `gistAPI()` and `apiError` type; all operations now use `newRequest`/`doRequest`/`formatAPIError` |
| `internal/cloud/github_gist.go` | Updated `List()` to use shared helpers |
| `internal/cloud/github_repo.go` | Removed `githubContent*` duplicate types; delegates to shared helpers |
| `internal/cloud/gitea.go` | Removed `giteaContent*` duplicate types; delegates to shared helpers |
| `internal/cloud/github_repo_test.go` | Renamed type references to shared types |
| `internal/cloud/gitea_test.go` | Renamed type references to shared types |

## Warnings Documented

### 1. `writeContentFile` accept parameter deviation (WARNING)

**Design signature**: `writeContentFile(client, token, method, url, req)`
**Actual signature**: `writeContentFile(client, token, method, accept, url, req)`

The implementation adds an explicit `accept` parameter to allow GitHub to send `application/vnd.github+json` while Gitea sends `application/json`. This does not break any spec scenario (the spec already references accept headers) and improves API flexibility. No functional impact.

### 2. Commit order deviation (WARNING)

**Design order**: (1) extract types, (2) migrate gist, (3) add tests
**Actual order**: (1) migrate gist, (2) extract types, (3) add helpers + tests

Gist migration was done first per user request. Implementation is coherent and all tests pass.

### 3. GGA bypass noted (SUGGESTION)

`apply-progress.md` mentions a GGA bypass for pre-existing git index corruption and architectural concerns. This should be reviewed separately to ensure pre-commit validation integrity.

## Lessons Learned

1. **Shared helper parameterization works well**: Adding `accept` as a parameter to `writeContentFile` was a pragmatic deviation that improved flexibility without breaking the spec contract.
2. **Existing tests as approval tests**: Refactoring tasks (Phases 1-2) leveraged existing test suites as approval tests, avoiding redundant test writing while maintaining safety.
3. **Atomic commits within a single PR**: Three atomic commits in one PR kept the change reviewable while maintaining git history clarity.

## Recommendations for Future Work

1. **Consolidate `auth.go`**: `auth.go` still uses direct `http.NewRequest`/`httpClient.Do` calls. A future refactor could extend DRY coverage by migrating it to shared helpers.
2. **Review GGA bypass**: Ensure the GGA bypass documented in apply-progress is addressed to maintain pre-commit validation standards.

## Spec Sync

No domain-scoped delta specs to sync. The change's `spec.md` is a flat delta spec at the change root. No matching main spec exists in `openspec/specs/` for the `cloud` domain. Main spec registry unchanged.

## Archive Contents

- proposal.md ✅
- spec.md ✅
- design.md ✅
- tasks.md ✅ (17/17 tasks complete)
- apply-progress.md ✅
- verify-report.md ✅
