# machine-profiles Specification

## Purpose
Defines named machine profiles for scoping backups, categories, presets, providers, and encryption settings.

## Requirements

### Requirement: Profile CRUD
The system MUST support `bak profile create|list|show|delete`.

#### Scenario: Create
- GIVEN a valid provider exists in config
- WHEN `bak profile create work-laptop` runs
- THEN the profile is persisted with its preset and provider

#### Scenario: List
- GIVEN multiple profiles exist
- WHEN `bak profile list` runs
- THEN all profiles display with names and providers

#### Scenario: Show
- GIVEN a profile exists
- WHEN `bak profile show <name>` runs
- THEN adapters, categories, preset, provider, and encryption status are shown

#### Scenario: Delete
- GIVEN a profile exists
- WHEN `bak profile delete <name>` runs
- THEN the profile is removed from config

### Requirement: Profile Scoping
The system MUST scope `bak backup`, `bak push`, and `bak pull` to the active profile when `--profile` is provided.

#### Scenario: Backup scoped
- GIVEN `--profile work-laptop` is passed
- WHEN `bak backup` runs
- THEN the backup uses the profile's adapters, categories, and preset

#### Scenario: Push scoped
- GIVEN `--profile work-laptop` is passed
- WHEN `bak push` runs
- THEN the push uses the profile's provider and encrypts if enabled

### Requirement: Provider Validation
The system MUST validate that a profile's provider exists and its token is configured at creation time.

#### Scenario: Missing provider
- GIVEN provider "foo" does not exist
- WHEN `bak profile create test --provider foo` runs
- THEN an error is returned

#### Scenario: Missing token
- GIVEN provider exists but token is unset
- WHEN `bak profile create test --provider <provider>` runs
- THEN an error is returned
