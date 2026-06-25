# Cloud Providers Specification

## Purpose

Consolidate duplicated `List()` implementations across cloud provider adapters into a shared helper, eliminating code duplication and ensuring consistent behavior.

## Requirements

### Requirement: Cloud List() consolidation

Duplicate `List()` logic across `GiteaProvider`, `GitHubProvider`, and `DropboxProvider` MUST be extracted into a shared helper function in `internal/cloud/`. Each provider's `List()` method MUST delegate to the shared helper.

#### Scenario: GiteaProvider delegates to shared List helper

- GIVEN `GiteaProvider.List()` is called
- WHEN the method executes
- THEN it delegates to the shared `listBackups` helper
- AND no duplicate listing logic exists in the provider

#### Scenario: GitHubProvider delegates to shared List helper

- GIVEN `GitHubProvider.List()` is called
- WHEN the method executes
- THEN it delegates to the shared `listBackups` helper
- AND no duplicate listing logic exists in the provider

#### Scenario: DropboxProvider delegates to shared List helper

- GIVEN `DropboxProvider.List()` is called
- WHEN the method executes
- THEN it delegates to the shared `listBackups` helper
- AND no duplicate listing logic exists in the provider

### Requirement: Shared List helper returns empty slice on error

The shared `listBackups` helper MUST return an empty `[]BackupMeta` slice (not nil) when the cloud directory does not exist or cannot be listed.

#### Scenario: non-existent directory returns empty slice

- GIVEN a cloud provider pointing to a non-existent directory
- WHEN `List()` is called
- THEN it returns an empty `[]BackupMeta` slice and nil error
