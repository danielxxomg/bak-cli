# Delta for cloud-sync

## Purpose

Change from GitHub Gist-only to pluggable provider backend.

## MODIFIED Requirements

### Requirement: Pluggable provider sync

The system MUST push/pull backups to a pluggable provider backend. GitHub Gist MUST remain the default provider. The system MUST support `--provider` flag on `push` and `pull` commands.
(Previously: System only supported GitHub Gist hardcoded backend.)

#### Scenario: Push round-trip with default provider

- GIVEN backup exists and `GITHUB_TOKEN` set
- WHEN `bak push` then `bak pull`
- THEN identical backup restored via GitHub Gist

#### Scenario: Push to GitHub Repo provider

- GIVEN backup exists and `GITHUB_TOKEN` set
- WHEN `bak push --provider github-repo` runs
- THEN backup pushed to GitHub repository instead of Gist

#### Scenario: Pull from Codeberg

- GIVEN backup exists on Codeberg and `CODEBERG_TOKEN` set
- WHEN `bak pull --provider codeberg <id>` runs
- THEN backup pulled from Codeberg

#### Scenario: Invalid provider

- GIVEN `--provider unknown` specified
- WHEN `bak push` runs
- THEN error returned and no network request made

#### Scenario: List backups across providers

- GIVEN `GITHUB_TOKEN` and `CODEBERG_TOKEN` configured
- WHEN `bak list --provider codeberg` runs
- THEN backups from Codeberg displayed

## REMOVED Requirements

### Requirement: GitHub Gist sync

(Reason: Superseded by pluggable provider sync. GitHub Gist is now the default provider implementation, not the only option.)
(Migration: Remove references to "GitHub Gist only" in docs; update CLI help to mention `--provider` flag.)
