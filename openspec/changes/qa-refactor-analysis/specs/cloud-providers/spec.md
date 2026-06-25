# Delta for cloud-providers

## ADDED Requirements

### Requirement: Cloud List() consolidation

Duplicate `List()` logic across `GiteaProvider` and `GitHubRepoProvider` MUST be consolidated into a shared helper `listContentsDir(client, url, token, acceptHeader, errPrefix, urlBuilder) ([]BackupMeta, error)` in `internal/cloud/httputil.go`. Both providers MUST delegate to this helper, parameterizing URL template, accept header, error prefix, and BackupMeta URL builder.

(Previously: ~50-line near-identical `List()` implementations in `gitea.go:143-191` and `github_repo.go:100-147` differing only in URL template, accept header, and error prefix.)

#### Scenario: GiteaProvider.List returns correct items via shared helper

- GIVEN a Gitea instance with 3 backup directories in the gist
- WHEN `GiteaProvider.List()` is called
- THEN it delegates to `listContentsDir` with Gitea-specific URL, accept header, and error prefix
- AND returns 3 `BackupMeta` entries with correct URLs

#### Scenario: GitHubRepoProvider.List returns correct items via shared helper

- GIVEN a GitHub repo with 5 backup directories
- WHEN `GitHubRepoProvider.List()` is called
- THEN it delegates to `listContentsDir` with GitHub-specific URL, accept header, and error prefix
- AND returns 5 `BackupMeta` entries with correct URLs

#### Scenario: shared logic parameterized by URL/headers/prefix

- GIVEN `listContentsDir` receives different URL templates and accept headers
- WHEN called by each provider
- THEN the HTTP request uses the provider-specific URL and headers
- AND the error messages use the provider-specific prefix

#### Scenario: HTTP error propagated with correct prefix

- GIVEN the API returns a 404 response
- WHEN `listContentsDir` is called by `GiteaProvider`
- THEN the returned error contains the Gitea-specific error prefix
- AND the same call via `GitHubRepoProvider` contains the GitHub-specific prefix

#### Scenario: empty directory list returns empty slice

- GIVEN a provider with no backup directories
- WHEN `List()` is called
- THEN it returns an empty `[]BackupMeta` slice and nil error
