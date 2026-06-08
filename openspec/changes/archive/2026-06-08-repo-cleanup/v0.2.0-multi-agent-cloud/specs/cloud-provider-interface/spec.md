# Delta for cloud-provider-interface

## Purpose

Provider interface abstracting cloud push/pull from GitHub Gist-specific code.

## Requirements

### Requirement: Provider interface contract

The `Provider` interface MUST define `Push(archive, meta) error`, `Pull(id) (archive, error)`, and `List() ([]BackupMeta, error)`.

#### Scenario: Push contract

- GIVEN a valid Provider implementation
- WHEN `Push` called with archive and metadata
- THEN archive uploaded to backend and metadata returned

#### Scenario: Pull contract

- GIVEN a backup ID exists on backend
- WHEN `Pull` called with that ID
- THEN archive bytes returned with matching checksum

#### Scenario: List contract

- GIVEN multiple backups exist on backend
- WHEN `List` called
- THEN all backup metadata returned in reverse chronological order

### Requirement: Provider registration

The system MUST register providers by name and route `push`/`pull` commands to the selected provider.

#### Scenario: Register GitHub Repo provider

- GIVEN `GitHubRepoProvider` implements `Provider`
- WHEN registered under name `github-repo`
- THEN `bak push --provider github-repo` uses it

#### Scenario: Unknown provider

- GIVEN provider name `gitlab` not registered
- WHEN `bak push --provider gitlab` runs
- THEN error returned: "unknown provider: gitlab"

### Requirement: Provider authentication

Each provider MUST support independent authentication via config or environment.

#### Scenario: Codeberg auth

- GIVEN `CODEBERG_TOKEN` set
- WHEN `bak push --provider codeberg` runs
- THEN request authenticated with token

#### Scenario: Missing auth

- GIVEN no token configured for provider
- WHEN `bak push` runs
- THEN error returned: "provider auth: token not found"
