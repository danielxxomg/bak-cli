# Backup Retention Specification

## Purpose

CLI command for pruning old backups with dry-run safety and confirmation gates.

## Requirements

### Requirement: `bak cleanup --keep N` retains N newest

The system MUST keep the N most recent backups and delete the rest. Backups are sorted by ID descending (lexicographic = chronological).

#### Scenario: Keep 3 of 10

- GIVEN 10 backups exist
- WHEN `bak cleanup --keep 3 --force` runs
- THEN the 3 newest backups MUST remain
- AND 7 backups MUST be deleted

#### Scenario: Keep more than exist

- GIVEN 2 backups exist
- WHEN `bak cleanup --keep 5 --force` runs
- THEN no backups are deleted
- AND a message indicates nothing to clean

#### Scenario: Default keep value

- GIVEN backups exist
- WHEN `bak cleanup --force` runs without `--keep`
- THEN the system uses a default keep value

### Requirement: Confirmation required without `--dry-run`

Without `--dry-run`, the system MUST prompt for confirmation on TTY or require `--force`. Non-TTY without `--force` MUST error.

#### Scenario: TTY prompt

- GIVEN TTY is available and backups to delete exist
- WHEN `bak cleanup --keep 3` runs
- THEN the system MUST prompt `Delete N backups? [y/N]`
- AND deletion proceeds only on `y`

#### Scenario: Non-TTY without force errors

- GIVEN non-TTY environment
- WHEN `bak cleanup --keep 3` runs without `--force`
- THEN the command MUST error with a helpful message

#### Scenario: Force skips prompt

- GIVEN `--force` flag is set
- WHEN `bak cleanup --keep 3 --force` runs
- THEN no prompt is shown and deletions proceed

### Requirement: `--dry-run` shows plan without deleting

`--dry-run` MUST list backups that would be deleted without removing any files.

#### Scenario: Dry-run lists deletions

- GIVEN 10 backups, `--keep 3`
- WHEN `bak cleanup --keep 3 --dry-run` runs
- THEN 7 backup IDs MUST be listed as "would delete"
- AND 0 files are removed from disk

#### Scenario: Dry-run with no deletions

- GIVEN 3 backups, `--keep 5`
- WHEN `bak cleanup --keep 5 --dry-run` runs
- THEN output indicates nothing to delete

#### Scenario: Dry-run then force

- GIVEN `--dry-run` showed 7 deletions
- WHEN `bak cleanup --keep 3 --force` runs next
- THEN exactly those 7 backups are deleted
