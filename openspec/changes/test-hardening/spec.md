# Spec: Test Hardening — Fix Misleading Testscripts and Close Coverage Gaps

## Capability: testscript-restore-roundtrip

The `backup_restore_roundtrip.txtar` testscript MUST exercise the full backup→restore cycle, not just `--help` and error paths.

### Scenario: restore with dry-run shows expected files

- **Given** a sandboxed HOME with OpenCode config fixtures (`opencode.json`, `AGENTS.md`, skills)
- **When** `bak backup --preset quick` is run and the backup ID is captured
- **Then** the backup directory `.bak/backups/<id>` MUST exist
- **And** `bak restore <id> --dry-run` MUST exit 0
- **And** stdout MUST contain the names of backed-up files (e.g. `opencode.json`)

### Scenario: restore with force actually restores files

- **Given** a backup has been created and the backup ID is known
- **When** `bak restore <id> --force` is run
- **Then** exit code MUST be 0
- **And** stdout MUST contain a success message (e.g. "Restored" or "restore complete")
- **And** the restored files MUST exist at their original source paths under `$HOME`

### Scenario: restore with invalid ID fails gracefully

- **Given** no backup with ID `20000101-000000` exists
- **When** `bak restore --dry-run 20000101-000000` is run
- **Then** exit code MUST be non-zero
- **And** stderr MUST contain "not found"

---

## Capability: testscript-diff-two-backups

The `diff_two_backups.txtar` testscript MUST exercise the `bak diff` command with real backup IDs, not just `--help`.

### Scenario: diff between two backups shows differences

- **Given** two backups have been created with different content (fixture modified between backups)
- **When** `bak diff <id1> <id2>` is run
- **Then** exit code MUST be 0
- **And** stdout MUST show file-level differences (added, removed, or modified files)

### Scenario: diff with identical backups shows no differences

- **Given** two backups created from identical fixtures
- **When** `bak diff <id1> <id2>` is run
- **Then** exit code MUST be 0
- **And** stdout MUST indicate no differences (or show empty diff)

### Scenario: diff with invalid IDs fails gracefully

- **Given** no backups exist with IDs `fake-id-1` and `fake-id-2`
- **When** `bak diff fake-id-1 fake-id-2` is run
- **Then** exit code MUST be non-zero
- **And** stderr MUST contain "not found"

---

## Capability: testscript-verify-roundtrip

The `backup_verify_roundtrip.txtar` testscript MUST exercise `bak verify` to check backup integrity, not just `bak list`.

### Scenario: verify confirms backup integrity

- **Given** a backup has been created with `quick` preset and the backup ID is known
- **When** `bak verify <id>` is run
- **Then** exit code MUST be 0
- **And** stdout MUST contain checksum verification output (e.g. "OK", "verified", or file-level checksum results)

### Scenario: verify with tampered file detects corruption

- **Given** a backup has been created and a backed-up file has been modified on disk
- **When** `bak verify <id>` is run
- **Then** exit code MUST be non-zero
- **And** stderr MUST indicate checksum mismatch or verification failure

### Scenario: verify with invalid ID fails gracefully

- **Given** no backup with ID `20000101-000000` exists
- **When** `bak verify 20000101-000000` is run
- **Then** exit code MUST be non-zero
- **And** stderr MUST contain "not found"

---

## Capability: cloud-sync-integration

A Go integration test MUST exercise the full push→pull cycle at the action level through a mock HTTP server.

### Scenario: action-level push creates gist and pull retrieves content

- **Given** a mock HTTP server simulating the GitHub Gist API (using `setupMockGistAPI`)
- **And** a `PushAction` configured with the mock server URL and a valid token
- **And** fixture files exist in a temp home directory
- **When** `PushAction.Push()` is called with a backup ID
- **Then** the mock server MUST receive a POST to `/gists` with a `backup.tar.gz` file
- **And** the returned ID MUST match the gist ID from the mock response
- **When** `PullAction.Pull()` is called with that ID
- **Then** the returned data MUST be the base64-encoded archive that was pushed
- **And** decoding and untarring MUST produce files matching the originals

### Scenario: push with invalid token returns authentication error

- **Given** a mock HTTP server that returns 401 for invalid tokens
- **And** a `PushAction` configured with an invalid token
- **When** `PushAction.Push()` is called
- **Then** the error MUST contain "401" or "unauthorized"

### Scenario: pull for non-existent gist returns not found error

- **Given** a mock HTTP server that returns 404 for unknown gist IDs
- **And** a `PullAction` configured with the mock server
- **When** `PullAction.Pull("nonexistent-id")` is called
- **Then** the error MUST contain "not found" or "404"

---

## Capability: schedule-happy-path

The schedule command happy path MUST be tested end-to-end through the cobra→action→scheduler chain.

### Scenario: schedule create succeeds with valid profile and interval (cmd level)

- **Given** a `cmdDeps` with a `ConfigLoader` returning a config with profile "work"
- **And** a mock `Scheduler` injected via `ScheduleAction.NewScheduler`
- **When** `runScheduleCreateWithDeps` is called with args `["work"]` and `--every daily`
- **Then** the mock scheduler's `Create` MUST be called with `("work", "daily")`
- **And** stdout MUST contain "Schedule created"
- **And** the profile's `Schedule` config MUST be updated to `{Enabled: true, Interval: "daily"}`

### Scenario: schedule list shows active schedules (cmd level)

- **Given** a mock `Scheduler` returning two entries: `{Profile: "work", Interval: "daily"}` and `{Profile: "home", Interval: "weekly"}`
- **When** `runScheduleListWithDeps` is called
- **Then** stdout MUST contain a table with "work", "daily", "home", "weekly"

### Scenario: schedule remove succeeds (cmd level)

- **Given** a mock `Scheduler` and a config with profile "work" having `Schedule.Enabled = true`
- **When** `runScheduleRemoveWithDeps` is called with args `["work"]`
- **Then** the mock scheduler's `Remove` MUST be called with `"work"`
- **And** stdout MUST contain "Schedule removed"
- **And** the profile's `Schedule` config MUST be set to nil

---

## Capability: tui-launch-smoke

A smoke test MUST prove the bak binary launches without panic.

### Scenario: bak binary with --help exits cleanly

- **Given** a compiled bak binary
- **When** `bak --help` is executed
- **Then** exit code MUST be 0
- **And** stdout MUST contain "Backup and restore your AI coding setup"

### Scenario: bak binary with no args in non-TTY shows help

- **Given** a compiled bak binary running in a non-TTY environment (piped stdin)
- **When** `bak` is executed with no arguments
- **Then** exit code MUST be 0
- **And** stdout MUST contain help output (since `isTTY()` returns false in test environments)

### Scenario: bak binary with unknown subcommand fails gracefully

- **Given** a compiled bak binary
- **When** `bak nonexistent-command` is executed
- **Then** exit code MUST be non-zero
- **And** stderr MUST contain an error message about unknown command

---

## Cross-Cutting Constraints

### Performance
- No individual test SHALL take more than 10 seconds
- The full test suite (`go test ./...`) SHALL complete in under 60 seconds

### Isolation
- All testscripts MUST use the sandboxed `$HOME` provided by `setupEnv`
- All Go tests MUST use `t.TempDir()` for filesystem isolation
- Cloud tests MUST use `httptest.NewServer` — no real network calls
- Schedule tests MUST inject a mock scheduler — no real crontab/schtasks calls

### Naming
- Test names MUST accurately describe what they test
- Testscript filenames MUST match the scenario they exercise
- If a test cannot exercise the full scenario, the filename MUST reflect the limited scope (e.g. `backup_restore_help.txtar`)

### Platform Compatibility
- All testscripts MUST pass on Linux, macOS, and Windows
- Schedule happy path tests MUST use mock injection (not real OS schedulers)
- TUI smoke tests MUST work in non-TTY environments (CI)
