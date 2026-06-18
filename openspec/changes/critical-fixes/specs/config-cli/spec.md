# Config CLI Specification

## Purpose

CLI interface for viewing and modifying bak settings via dotted keys, with mandatory token redaction in output.

## Requirements

### Requirement: `bak config show` displays current settings

The system MUST display all current settings in human-readable format. Token values MUST be redacted, showing only the last 4 characters prefixed by `***`.

#### Scenario: Show redacts tokens

- GIVEN `providers.codeberg.token` is `ghp_abcdef1234567890`
- WHEN `bak config show` runs
- THEN the token MUST display as `***7890`
- AND the raw token MUST NOT appear in output

#### Scenario: Show non-secret values

- GIVEN `settings.default_preset` is `quick`
- WHEN `bak config show` runs
- THEN `quick` MUST appear in full (no redaction)

#### Scenario: Show with no config file

- GIVEN no config file exists
- WHEN `bak config show` runs
- THEN default settings MUST be displayed

### Requirement: `bak config set` persists setting

The system MUST persist the given value for a dotted key path and overwrite existing values.

#### Scenario: Set token value

- GIVEN `bak config set providers.codeberg.token xyz`
- WHEN the command executes
- THEN `providers.codeberg.token` MUST be `xyz` in the config file
- AND subsequent `bak config show` MUST redact it as `***xyz`

#### Scenario: Set nested key

- GIVEN `bak config set settings.default_preset full`
- WHEN the command executes
- THEN `settings.default_preset` MUST be `full`

#### Scenario: Set invalid key

- GIVEN an unknown dotted key path
- WHEN `bak config set` runs
- THEN a helpful error MUST be returned

### Requirement: Token redaction in output

Any config output MUST redact values for keys containing `token`, `api_key`, `secret`, or `password`, showing only the last 4 characters prefixed by `***`.

#### Scenario: Short token redacted entirely

- GIVEN a token value `ab`
- WHEN displayed in config output
- THEN it MUST show as `***ab`

#### Scenario: Multiple tokens all redacted

- GIVEN multiple providers with tokens configured
- WHEN `bak config show` runs
- THEN ALL token values MUST be redacted
