# Delta for cloud-sync

## ADDED Requirements

### Requirement: OAuth token support

The system MUST accept OAuth access tokens obtained via Device Flow (RFC 8628) in addition to personal access tokens. Both token types SHALL be stored in `github.token` and validated via `cloud.ValidateToken`.

#### Scenario: OAuth token used for push

- GIVEN the user logged in via `bak login` OAuth flow
- WHEN `bak push` executes
- THEN the OAuth token SHALL be used for GitHub API authentication

#### Scenario: PAT still works

- GIVEN the user has a manually configured PAT in `github.token`
- WHEN `bak push` executes
- THEN the PAT SHALL work as before (no regression)

### Requirement: Login dispatch

The system MUST dispatch to OAuth Device Flow when `BAK_GITHUB_OAUTH_CLIENT_ID` is set, and fall back to manual PAT paste when it is not.

#### Scenario: OAuth available

- GIVEN `BAK_GITHUB_OAUTH_CLIENT_ID` is set
- WHEN `bak login` is invoked
- THEN the OAuth Device Flow SHALL be initiated

#### Scenario: OAuth not available

- GIVEN `BAK_GITHUB_OAUTH_CLIENT_ID` is not set
- WHEN `bak login` is invoked
- THEN the manual PAT paste prompt SHALL be displayed (existing behavior)

## MODIFIED Requirements

### Requirement: GitHub Gist sync

The system MUST push/pull backups to private GitHub Gists. Authentication SHALL support both personal access tokens and OAuth access tokens.

(Previously: only personal access tokens were supported for authentication)

#### Scenario: Push round-trip

- GIVEN backup exists and a valid token (PAT or OAuth) is configured
- WHEN `bak push` then `bak pull`
- THEN identical backup restored

#### Scenario: Push with OAuth token

- GIVEN backup exists and the user authenticated via OAuth Device Flow
- WHEN `bak push` executes
- THEN the push SHALL succeed using the OAuth token
