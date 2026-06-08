# Design: dry-extraction

## Architecture Decision: Where to place shared utilities

### Decision 1: `internal/adapters/util.go` (package `adapters`)

**Rationale**: `copyFile` and `fileHash` are general-purpose file utilities used by adapter implementations. Placing them in the parent `adapters` package makes them available to all sub-packages via import. Since `yaml.go` is already in `package adapters`, it can call them directly without import.

**Alternatives considered**:
- `internal/adapters/shared/` or similar sub-package: adds unnecessary nesting
- Inline in each adapter: current state, violates DRY

### Decision 2: Exported names `CopyFile` and `FileHash`

**Rationale**: Sub-packages import `adapters` and call `adapters.CopyFile()`. The callsite reads clearly and follows Go convention (package qualifier + PascalCase).

### Decision 3: Use the best implementation variant

The `yaml.go` variant has the best error wrapping (uses `fmt.Errorf` for all error paths), matching AGENTS.md mandate. We'll use that as the canonical implementation, with minor improvements:
- `copyFile`: named return `err` removed (not needed, callers don't check it)
- `fileHash`: consistent error wrapping in all branches

```go
// CopyFile copies a regular file from src to dst, creating parent
// directories as needed. It preserves no metadata beyond file content.
func CopyFile(src, dst string) error { ... }

// FileHash computes the SHA-256 hex digest and file size for the given
// regular file path.
func FileHash(path string) (hash string, size int64, err error) { ... }
```

### Decision 4: Cloud HTTP helper design

**Problem**: Both `GitHubRepoProvider` and `GiteaProvider` repeat the same HTTP request pattern:
```
build URL → http.NewRequest → set Authorization → set Accept → set User-Agent → client.Do → defer close → io.ReadAll → check status → format error
```

**Design**: Two helpers in `internal/cloud/httputil.go`:

```go
// newRequest builds an authenticated HTTP request with common headers.
func newRequest(method, url, token, accept, contentType string, body io.Reader) (*http.Request, error)

// doRequest executes an HTTP request and reads the response body.
// Returns the body, status code, and any error.
func doRequest(client *http.Client, req *http.Request) (body []byte, status int, err error)
```

**Why two functions instead of one**: Separating request building from execution allows callers to add custom headers if needed, and keeps each function focused on one responsibility.

**Tradeoffs**:
- Two-step pattern at each callsite (build + execute) vs one-step
- But each callsite's boilerplate shrinks from ~10 lines to ~4 lines
- Callers still need to check status codes themselves (404 handling differs)

## Sequence: Refactoring a sub-package adapter

```
Before:
  package opencode
  func copyFile(src, dst string) error { ... }  // 20+ lines
  func fileHash(path string) (...) { ... }       // 17+ lines
  
  func (a *Adapter) Backup(...) {
      copyFile(src, dst)   // local call
  }

After:
  package opencode
  import "github.com/danielxxomg/bak-cli/internal/adapters"
  
  // copyFile and fileHash removed
  
  func (a *Adapter) Backup(...) {
      adapters.CopyFile(src, dst)   // shared call
  }
```

## Files Changed

| File | Action | Scope |
|------|--------|-------|
| `internal/adapters/util.go` | Create | New file with `CopyFile`, `FileHash` |
| `internal/adapters/util_test.go` | Create | Moved tests from `yaml_test.go` |
| `internal/adapters/yaml.go` | Modify | Remove `copyFile`/`fileHash`, use shared |
| `internal/adapters/yaml_test.go` | Modify | Remove moved tests |
| `internal/adapters/opencode/adapter.go` | Modify | Remove dupes, call `adapters.CopyFile`/`adapters.FileHash` |
| `internal/adapters/opencode/adapter_test.go` | Modify | Update `fileHash` test calls |
| `internal/adapters/cursor/adapter.go` | Modify | Same |
| `internal/adapters/cursor/adapter_test.go` | Modify | Same |
| `internal/adapters/pidev/adapter.go` | Modify | Same |
| `internal/adapters/pidev/adapter_test.go` | Modify | Same |
| `internal/adapters/windsurf/adapter.go` | Modify | Same |
| `internal/adapters/windsurf/adapter_test.go` | Modify | Same |
| `internal/adapters/claudecode/adapter.go` | Modify | Same |
| `internal/adapters/claudecode/adapter_test.go` | Modify | Same |
| `internal/adapters/kiro/adapter.go` | Modify | Same |
| `internal/adapters/kiro/adapter_test.go` | Modify | Same |
| `internal/adapters/kilocode/adapter.go` | Modify | Same |
| `internal/adapters/kilocode/adapter_test.go` | Modify | Same |
| `internal/adapters/codex/adapter.go` | Modify | Same |
| `internal/adapters/codex/adapter_test.go` | Modify | Same |
| `internal/cloud/httputil.go` | Create | New file with `newRequest`, `doRequest` |
| `internal/cloud/github_repo.go` | Modify | Use shared helpers |
| `internal/cloud/gitea.go` | Modify | Use shared helpers |
