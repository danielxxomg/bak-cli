# Design: Cloud Provider Code Consolidation

## Technical Approach

Three-part internal refactoring of `internal/cloud/` to eliminate DRY violations. Each part is an independent commit. No behavioral changes — all existing tests must pass without modification (except `gist_test.go` if `apiError` changes shape).

## Architecture Decisions

### Decision: Where shared types live

| Option | Tradeoff | Decision |
|--------|----------|----------|
| `content_types.go` (new file) | Clean separation, easy to find | ✅ **Chosen** |
| Top of `github_repo.go` | No new file, but GitHub-biased naming | Rejected |
| `provider.go` | Already has shared types | Rejected — wrong abstraction level |

**Choice**: New file `internal/cloud/content_types.go` with unexported types `contentRequest`, `contentResponse`, `contentFile`.

### Decision: How to handle gist.go's apiError type

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Remove apiError, use formatAPIError directly | Cleanest, but may break test assertions on type | ✅ **Chosen** (with test audit) |
| Keep apiError, construct from formatAPIError | Preserves type but adds indirection | Rejected — unnecessary |
| Make formatAPIError return apiError | Changes httputil.go contract | Rejected — affects all callers |

**Choice**: Replace `apiError` with `formatAPIError`. Audit `gist_test.go` — if tests assert on `apiError` type, update them to check error message strings instead.

### Decision: Extracting getFileSHA and writeContentFile

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Package-level functions with URL param | Simple, no interface needed | ✅ **Chosen** |
| Embed in a base struct | OOP-style, but Go prefers composition | Rejected |
| Interface + implementation | Over-engineered for 2 callers | Rejected |

**Choice**: Package-level functions in `content_types.go`:
```go
func getFileSHA(client *http.Client, token, url string) (string, error)
func writeContentFile(client *http.Client, token, method, url string, req contentRequest) error
```

Each provider constructs its own URL (different API patterns) and calls these helpers.

## Data Flow

```
GitHubRepoProvider.Push()
  ├── getFileSHA(client, token, githubURL)  ← shared helper
  │     └── newRequest() → doRequest() → formatAPIError()
  └── writeContentFile(client, token, "PUT", githubURL, req)  ← shared helper
        └── newRequest() → doRequest() → formatAPIError()

GiteaProvider.Push()
  ├── getFileSHA(client, token, giteaURL)   ← same shared helper
  │     └── newRequest() → doRequest() → formatAPIError()
  └── writeContentFile(client, token, method, giteaURL, req) ← same helper
        └── newRequest() → doRequest() → formatAPIError()

CreateGist() / UpdateGist() / GetGist() / DeleteGist()
  └── newRequest() → doRequest() → formatAPIError()  ← now uses shared helpers
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/cloud/content_types.go` | Create | Shared types (`contentRequest`, `contentResponse`, `contentFile`) + helpers (`getFileSHA`, `writeContentFile`) |
| `internal/cloud/gist.go` | Modify | Remove `gistAPI()`, `apiError` type; use `newRequest`/`doRequest`/`formatAPIError` |
| `internal/cloud/github_repo.go` | Modify | Remove `githubContentRequest`, `githubContentResponse`, `githubContentFile`; use shared types. Replace `getFileSHA`/`putFile` bodies with shared helpers |
| `internal/cloud/gitea.go` | Modify | Remove `giteaContentRequest`, `giteaContentResponse`, `giteaContentFile`; use shared types. Replace `getFileSHA`/`writeFile` bodies with shared helpers |
| `internal/cloud/gist_test.go` | Modify | Update any `apiError` type assertions to error message checks |

## Interfaces / Contracts

```go
// content_types.go — unexported, package-level

type contentRequest struct {
    Message string `json:"message"`
    Content string `json:"content"`
    Branch  string `json:"branch"`
    SHA     string `json:"sha,omitempty"`
}

type contentResponse struct {
    Name    string       `json:"name"`
    Path    string       `json:"path"`
    SHA     string       `json:"sha"`
    Size    int64        `json:"size"`
    Content contentFile  `json:"content"`
}

type contentFile struct {
    Name     string `json:"name"`
    Path     string `json:"path"`
    SHA      string `json:"sha"`
    Size     int64  `json:"size"`
    Encoding string `json:"encoding"`
    Content  string `json:"content"`
}

func getFileSHA(client *http.Client, token, url string) (string, error)
func writeContentFile(client *http.Client, token, method, accept, url string, req contentRequest) error
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | `getFileSHA` / `writeContentFile` | Table-driven tests with `httptest.Server` |
| Unit | gist.go functions after migration | Existing tests — verify they still pass |
| Integration | `go test ./internal/cloud/...` | All existing tests must pass |
| Quality | `go vet ./...` + `golangci-lint run` | Must be clean |

## Migration / Rollout

No migration required. Pure internal refactoring — no config, data, or API changes.

## Commit Strategy

Three atomic commits:
1. `refactor(cloud): extract shared content API types and helpers` — create `content_types.go`, update `github_repo.go` and `gitea.go` to use shared types/helpers
2. `refactor(cloud): migrate gist.go to shared HTTP helpers` — remove `gistAPI()`, use `newRequest`/`doRequest`/`formatAPIError`
3. `test(cloud): add tests for shared content helpers` — table-driven tests for `getFileSHA`/`writeContentFile`

## Open Questions

- None — all code has been read and duplication verified.
