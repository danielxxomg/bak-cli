# Delta Spec: dry-extraction

## ADDED Requirements

### REQ-DRY-001: Shared adapter utility functions

The `internal/adapters` package SHALL provide exported `CopyFile` and `FileHash` utility functions usable by all adapter sub-packages.

#### Scenario: CopyFile copies a regular file
- **Given** a source file exists at `/tmp/src/file.txt`
- **When** `adapters.CopyFile("/tmp/src/file.txt", "/tmp/dst/file.txt")` is called
- **Then** the destination file SHALL exist with identical content
- **And** parent directories SHALL be created with permissions 0755

#### Scenario: CopyFile returns error for missing source
- **Given** a source file does NOT exist
- **When** `adapters.CopyFile("/nonexistent", "/tmp/dst")` is called
- **Then** an error SHALL be returned wrapping the OS error

#### Scenario: FileHash computes SHA-256 digest
- **Given** a file exists at `/tmp/test.txt` containing "hello"
- **When** `adapters.FileHash("/tmp/test.txt")` is called
- **Then** the hash SHALL be `sha256:<hex>` format
- **And** the size SHALL match the file size in bytes

#### Scenario: FileHash returns error for missing file
- **Given** a file does NOT exist
- **When** `adapters.FileHash("/nonexistent")` is called
- **Then** an error SHALL be returned

### REQ-DRY-002: Adapter sub-packages use shared utilities

All adapter sub-packages (opencode, cursor, pidev, windsurf, claudecode, kiro, kilocode, codex) and `yaml.go` SHALL use `adapters.CopyFile` and `adapters.FileHash` instead of local implementations.

#### Scenario: Sub-package calls shared CopyFile
- **Given** adapter sub-package `opencode`
- **When** a backup or restore operation copies a file
- **Then** `adapters.CopyFile(src, dst)` SHALL be called
- **And** the local `copyFile` function SHALL NOT exist in the sub-package

#### Scenario: Sub-package calls shared FileHash
- **Given** adapter sub-package `opencode`
- **When** a file hash is computed
- **Then** `adapters.FileHash(path)` SHALL be called
- **And** the local `fileHash` function SHALL NOT exist in the sub-package

### REQ-DRY-003: Shared cloud HTTP utilities

The `internal/cloud` package SHALL provide internal helper functions to reduce HTTP request boilerplate across cloud providers.

#### Scenario: doRequest executes an authenticated API call
- **Given** a valid HTTP method, URL, token, and optional body
- **When** `doRequest(client, method, url, token, headers, body)` is called
- **Then** an authenticated HTTP request SHALL be sent
- **And** the response body, status code, and any error SHALL be returned

#### Scenario: GitHub provider uses shared HTTP helpers
- **Given** the `GitHubRepoProvider`
- **When** `getFileSHA`, `putFile`, or `Pull` methods are called
- **Then** HTTP request construction SHALL use the shared `newRequest` or `doRequest` helper
- **And** duplicate HTTP boilerplate SHALL be removed

#### Scenario: Gitea provider uses shared HTTP helpers
- **Given** the `GiteaProvider`
- **When** `getFileSHA`, `writeFile`, or `Pull` methods are called
- **Then** HTTP request construction SHALL use the shared `newRequest` or `doRequest` helper
- **And** duplicate HTTP boilerplate SHALL be removed

### REQ-DRY-004: Existing tests continue to pass

All existing tests SHALL pass after the refactoring with no regressions.

#### Scenario: Full test suite passes
- **Given** the refactoring is complete
- **When** `go test ./...` is executed
- **Then** all tests SHALL pass
- **And** `go vet ./...` SHALL report no issues

#### Scenario: Adapter sub-package tests pass
- **Given** adapter test files updated to reference `adapters.FileHash`
- **When** `go test ./internal/adapters/...` is executed
- **Then** all adapter tests SHALL pass

#### Scenario: Cloud provider tests pass
- **Given** cloud provider code uses shared HTTP helpers
- **When** `go test ./internal/cloud/...` is executed
- **Then** all cloud tests SHALL pass
