# Delta for Config and Manifest

## ADDED Requirements

### Requirement: v0.2.0→v0.3.0 Config Migration
The system MUST additively migrate existing v0.2.0 configs: add an empty `profiles` map, bump `schema_version` to "0.3.0", preserve all existing providers, and write a `config.json.v020.bak` backup.

#### Scenario: Additive migration
- GIVEN a v0.2.0 config with providers
- WHEN the system loads the config
- THEN `profiles` is added, `schema_version` equals "0.3.0", providers are preserved, and a `.v020.bak` backup exists

### Requirement: Manifest Encryption Metadata
The system MUST include an optional `Encryption` struct in the manifest containing algorithm, KDF params, salt, and nonce.

#### Scenario: Encrypted manifest
- GIVEN an encrypted push
- WHEN the manifest is generated
- THEN the `Encryption` struct is present with algorithm, KDF, salt, and nonce

#### Scenario: Plaintext manifest
- GIVEN an unencrypted push
- WHEN the manifest is generated
- THEN the `Encryption` field is absent

## MODIFIED Requirements

### Requirement: Config Schema Version
The system MUST report schema_version "0.3.0" after migration.
(Previously: schema_version was "0.2.0")

#### Scenario: Version bump
- GIVEN a migrated config
- THEN `schema_version` equals "0.3.0"

## REMOVED Requirements

### Requirement: No Encryption at Rest
The prohibition on encryption at rest is removed.
(Reason: v0.3.0 introduces encryption-at-rest.)
(Migration: Update documentation and constraints.)
