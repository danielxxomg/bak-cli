# Delta for config-migration

## Purpose

Auto-migration from v0.1.0 flat config to v0.2.0 multi-backend config.

## Requirements

### Requirement: Migration detection

`config.Load()` MUST detect v0.1.0 schema by presence of `github_token` and `gist_id` at root level.

#### Scenario: v0.1.0 config detected

- GIVEN `config.json` contains `github_token` and `gist_id` at root
- WHEN `config.Load()` called
- THEN migration triggered automatically

#### Scenario: v0.2.0 config skipped

- GIVEN `config.json` contains `schema_version: "0.2.0"`
- WHEN `config.Load()` called
- THEN no migration runs

### Requirement: Migration transformation

The system MUST transform flat keys to nested `providers.github.token` and `providers.github.gist_id`.

#### Scenario: Auto-migrate

- GIVEN v0.1.0 config with `github_token: "ghp_xxx"`
- WHEN `Load()` completes
- THEN `providers.github.token` set to `"ghp_xxx"`

### Requirement: Backup preservation

The system MUST preserve the original config as `config.json.v010.bak` before writing migrated config.

#### Scenario: Backup created

- GIVEN v0.1.0 config exists
- WHEN migration writes new config
- THEN `.v010.bak` exists with original content

#### Scenario: Rollback

- GIVEN `.v010.bak` exists and migration caused issues
- WHEN user replaces `config.json` with `.v010.bak`
- THEN bak reads v0.1.0 config successfully

### Requirement: Schema marker

Migrated config MUST include `schema_version: "0.2.0"` to prevent re-migration.

#### Scenario: Idempotent migration

- GIVEN config already migrated to v0.2.0
- WHEN `Load()` called again
- THEN `schema_version` remains `"0.2.0"` and no duplicate backup created
