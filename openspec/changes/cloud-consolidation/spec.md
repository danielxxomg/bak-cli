# Delta Spec: Cloud Consolidation

## ADDED Requirements

### Requirement: Shared content API types

The `internal/cloud` package SHALL provide unexported `contentRequest`, `contentResponse`, and `contentFile` types for use by all Contents API-based providers (GitHub, Gitea).

#### Scenario: contentRequest has required fields

- GIVEN a provider needs to create or update a file via a Contents API
- WHEN it constructs a `contentRequest`
- THEN the struct SHALL have fields: `Message`, `Content`, `Branch`, `SHA` (omitempty)
- AND JSON tags SHALL match the API expectation (`message`, `content`, `branch`, `sha`)

#### Scenario: contentResponse parses file metadata

- GIVEN a Contents API returns a JSON response with name, path, sha, size, and nested content
- WHEN unmarshalled into `contentResponse`
- THEN all fields SHALL be accessible including the nested `contentFile` with encoding and content

### Requirement: Shared getFileSHA helper

The `internal/cloud` package SHALL provide a `getFileSHA` function that fetches file metadata from a Contents API endpoint and returns the SHA.

#### Scenario: File exists

- GIVEN a file exists at the given API URL
- WHEN `getFileSHA(client, token, url)` is called
- THEN it SHALL return the file's SHA and nil error

#### Scenario: File does not exist (404)

- GIVEN no file exists at the given API URL
- WHEN `getFileSHA(client, token, url)` is called
- THEN it SHALL return empty string and nil error

#### Scenario: API returns error status

- GIVEN the API returns a non-2xx, non-404 status
- WHEN `getFileSHA(client, token, url)` is called
- THEN it SHALL return an error wrapping `formatAPIError` output

### Requirement: Shared writeContentFile helper

The `internal/cloud` package SHALL provide a `writeContentFile` function that sends a create/update request to a Contents API endpoint.

#### Scenario: Create or update file succeeds

- GIVEN valid content, message, branch, and optional SHA
- WHEN `writeContentFile(client, token, method, url, req)` is called
- THEN it SHALL marshal the request, send it, and return nil on 2xx

#### Scenario: API returns error

- GIVEN the API returns a non-2xx status
- WHEN `writeContentFile` is called
- THEN it SHALL return an error wrapping `formatAPIError` output

### Requirement: gist.go uses shared HTTP helpers

All HTTP operations in `gist.go` SHALL use `newRequest`, `doRequest`, and `formatAPIError` from `httputil.go` instead of manual request construction.

#### Scenario: CreateGist uses shared helpers

- GIVEN a valid token and files
- WHEN `CreateGist` is called
- THEN it SHALL use `newRequest` to build the request and `doRequest` to execute it
- AND SHALL NOT call `http.NewRequest` directly

#### Scenario: Error formatting preserved

- GIVEN the GitHub API returns a 422 error with message "Invalid request"
- WHEN a gist operation fails
- THEN the returned error SHALL contain status code 422 and the message "Invalid request"

## MODIFIED Requirements

### Requirement: Shared cloud HTTP utilities

The `internal/cloud` package SHALL provide internal helper functions to reduce HTTP request boilerplate across ALL cloud providers, including gist.go.

(Previously: Only GitHub and Gitea providers used shared HTTP helpers; gist.go had its own `gistAPI()` function)

#### Scenario: doRequest executes an authenticated API call

- GIVEN a valid HTTP method, URL, token, and optional body
- WHEN `doRequest(client, method, url, token, headers, body)` is called
- THEN an authenticated HTTP request SHALL be sent
- AND the response body, status code, and any error SHALL be returned

#### Scenario: GitHub provider uses shared HTTP helpers

- GIVEN the `GitHubRepoProvider`
- WHEN `getFileSHA`, `putFile`, or `Pull` methods are called
- THEN HTTP request construction SHALL use the shared `newRequest` or `doRequest` helper
- AND duplicate HTTP boilerplate SHALL be removed

#### Scenario: Gitea provider uses shared HTTP helpers

- GIVEN the `GiteaProvider`
- WHEN `getFileSHA`, `writeFile`, or `Pull` methods are called
- THEN HTTP request construction SHALL use the shared `newRequest` or `doRequest` helper
- AND duplicate HTTP boilerplate SHALL be removed

#### Scenario: Gist operations use shared HTTP helpers

- GIVEN the gist API functions in `gist.go`
- WHEN `CreateGist`, `UpdateGist`, `GetGist`, `DeleteGist`, or `List` are called
- THEN HTTP request construction SHALL use `newRequest` and `doRequest`
- AND the `gistAPI` function SHALL NOT exist
