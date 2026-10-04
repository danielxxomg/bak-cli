# Security Policy

## Supported Versions

Security updates are provided for the latest release only. Active development is on an unreleased stabilization line; security fixes apply to the stabilization line and the latest release.

| Version | Supported          |
|---------|--------------------|
| Latest release (unreleased stabilization line) | :white_check_mark: |

When a new major version is released, the previous version stops receiving security patches.

## Reporting a Vulnerability

**Do not open a public issue for security vulnerabilities.**

### Private Disclosure

Email security reports to the maintainer at the address listed on the [GitHub profile](https://github.com/danielxxomg).

Please include:

1. A clear description of the vulnerability
2. Steps to reproduce (proof-of-concept if possible)
3. Affected version(s)
4. Potential impact

### What to Expect

- **Acknowledgment** within 48 hours
- **Status update** within 7 days of acknowledgment
- **Fix timeline** depends on severity:
  - **Critical** (data loss, token leakage): patch within 24–72 hours
  - **High** (bypass of safety guarantees): patch within 7 days
  - **Medium/Low**: addressed in the next scheduled release

### Disclosure Policy

- The reporter will be credited in the release notes (unless they request anonymity)
- A CVE will be requested for critical vulnerabilities
- Public disclosure will be coordinated with the fix release

## Security Features

bak-cli includes multiple layers of safety by design:

### Path Traversal Prevention

All file paths written during restore operations are validated to stay within the user's home directory. The restore engine resolves and canonicalizes every path before writing, rejecting any path that escapes the home directory boundary.

```
Implementation:
- os.UserHomeDir() for the base directory (never hardcoded)
- path.Clean + strings.ReplaceAll(path, "\\", "/") for canonical path comparison
- Reject paths that do not start with the canonical home prefix
```

### Symlink Traversal and Containment

During backup scanning, adapters follow symbolic links that stay within the user's home directory boundary:
- **Directory symlinks**: Symlinks targeting directories within the home directory are traversed recursively, enabling backup of shared skill directories, command libraries, or plugin directories linked across agents.
- **File symlinks**: Symlinks targeting regular files within the home directory are resolved, hashed, and backed up as regular files.
- **Broken and escaping symlinks**: Broken or unresolvable symlinks, and symlinks pointing outside the user's home directory, are skipped (and warned on `stderr` when verbose mode is enabled) rather than aborting the backup or copying content outside home. Cycle detection prevents infinite traversal on recursive symlink loops.
- **Tradeoff**: Following symlinks means the backup can include content stored outside the adapter's own configuration directory (for example, shared skills located in `~/.agents/skills/`), provided the target remains safely within the user's home directory.

### Secret Detection and Redacted-in-Place Backup

The backup engine detects common secret patterns and redacts them in-place with placeholders:

| Pattern | Description |
|---------|-------------|
| `ghp_*` | GitHub personal access tokens |
| `gho_*` | GitHub OAuth tokens |
| `ghu_*` | GitHub user-to-server tokens |
| `ghs_*` | GitHub server-to-server tokens |
| `ghr_*` | GitHub refresh tokens |
| `sk-*` | OpenAI API keys |
| `sk-ant-*` | Anthropic API keys |
| `xoxb-*` | Slack bot tokens |
| `xoxp-*` | Slack user tokens |
| `AKIA*`, `ASIA*` | AWS access key IDs (AKIA long-lived, ASIA temporary/STS) |
| `AIza*` | Google Cloud API keys |
| `sk_*`, `rk_*` | Stripe secret and restricted keys (publishable `pk_*` keys are excluded) |
| Connection strings | DSNs carrying inline credentials (`user:password@`) |
| `Bearer *` | HTTP Bearer authentication tokens |

Instead of dropping files containing secrets, bak preserves configuration files structurally by replacing secrets in-place with `<YOUR_SECRET>` placeholders in the backup payload and generating a companion `.env.example` template:
- **Redacted-in-place payload**: Config files containing recognized secret families are written to the backup payload with every matched secret replaced by `<YOUR_SECRET>`. Surrounding structure (JSON/YAML formatting, sibling keys, comments) survives.
- **Manifest schema 0.5.0 tracking**: Manifest `Item` entries record `redacted: true` and `secret_count`. Stored SHA-256 hashes and file sizes describe the redacted content actually written to disk, so integrity checks pass on redacted backups.
- **Companion `.env.example`**: Generated directly from source files with home-relative section headers and `<YOUR_SECRET>` placeholders to list what needs re-entering.
- **Transparent summary reporting**: The backup summary distinguishes excluded files from files backed up with secrets redacted using home-relative paths (`~/...`).
- **Honest dry-run classification**: On restore, dry-run labels redacted files as `[redacted]`, distinctly from clean and missing files.
- **Restore report transparency**: The restore report lists every restored file that contains placeholders and reminds the user that secrets must be re-entered by hand.
- **Critical tradeoff**: Restoring a redacted file **overwrites the live file's real secrets with placeholders**. The user must re-enter them. This is strictly better than having no backup of the file, but it is a real footgun and users must be aware that restoring onto a live system will replace working credentials with placeholders until re-entered.

Stripe publishable `pk_` keys are deliberately not treated as secrets: Stripe documents them as safe for client-side use, and redacting them would replace working configuration with placeholders on restore.

Unrecognized secret formats, custom token formats, or keys from unlisted providers outside these families are not detected and must be managed or excluded manually.

### Checksum and Manifest Integrity

- **SHA-256 checksums**: Every backed-up file gets a SHA-256 checksum computed at backup time and stored in `manifest.json`.
- **Integrity verification**: On restore, every file is verified against its stored checksum before being written. Checksum mismatches block the restore and produce a clear error message.
- **Mandatory under `--force`**: Manifest and checksum integrity verification cannot be bypassed. The `--force` flag skips interactive confirmation only, never integrity checks.
- **Permission preservation (0.4.0+)**: Manifest schema `0.4.0` and `0.5.0` preserve portable file permission bits (`Mode`) at backup time and reapply them during restore. Any chmod failures are reported as restore errors.
- **Degraded 0.3.0 handling**: Legacy `0.3.0` manifests lacking mode metadata are loaded and restored in degraded mode, explicitly warning the user that restored files lack original mode metadata rather than claiming exact permission restoration.
- **Version compatibility warning**: On restore, `bak` compares the backup tool version (`bak_version`) with the running tool version. Any mismatch (including unknown, development, or empty versions on either side) produces a clear warning on stderr naming the backup ID and both versions without blocking restore.
- **Schema version gating**: Manifest schema versions newer than supported (`0.5.0`) cannot be safely interpreted and fail closed with an actionable error before any target write or recovery preparation, instructing the user to upgrade `bak`. Legacy known versions (`0.3.0` and `0.4.0`) continue to restore.
- **Real-binary journey verification**: An eight-stage real-binary journey matrix (`tests/e2e/journey_matrix_test.go`) characterizes discovery, mutation/deletion diff recovery, dry-run zero-write guarantees, apply correctness, manifest verification, tamper fail-closed rejection, partial failure rollback, and target undo drift protection.

### Target Recovery and Automatic Rollback

Restore operations are protected by private local target recovery snapshots:

- **Pre-restore target capture**: Before any target file is modified during restore, `bak` captures the affected target files (original bytes, permission mode bits, and absence status) in private local recovery storage under `~/.bak/recovery/<point-id>`.
- **Fail-closed preparation**: If target inspection, recovery staging, or the pre-restore commit fails, the restore operation aborts immediately before any target file is modified.
- **Automatic rollback on failure**: If writing any target file or applying file permissions fails, the restore halts immediately and attempts rollback of every attempted target file (including partially written files). Targets that did not exist before restore are deleted; targets that previously existed are restored to their original bytes and permissions.
- **Preserved recovery evidence**: Recovery evidence is retained locally under `~/.bak/recovery/<point-id>` with restricted permissions (0700/0600) for diagnostics and manual recovery. The operation returns an error reporting the original failure and rollback outcome.
- **Private local plaintext storage**: Recovery data is stored unencrypted in local private storage under `~/.bak/recovery/` with restrictive permissions. It is never pushed to cloud providers or included in cloud backup archives.
- **Best-effort limitations**: Rollback makes a best-effort attempt to revert target files across failures, but cannot promise race-proof atomicity against external concurrent processes or OS-level permission revocations during rollback.

### Git Safety Net

Local backup operations are protected by Git history tracking within `~/.bak`:

- **Scoped to `~/.bak`**: Git tracking is initialized within `~/.bak` to track local backup metadata and snapshots. Target configuration directories (such as `~/.config/opencode/`) are not auto-committed to Git by bak during restore operations (`GitDir` integration on target configs is unwired).
- **`bak undo`**: Reverts the last restore operation by restoring target configuration files to their pre-restore state using recovery point metadata, and records a `git revert` commit in the `~/.bak` repository — safe, non-destructive, and history-preserving.
- **No force-push**: The tool never force-pushes or rewrites Git history.

### Dry-Run and Confirmation

`bak restore` requires confirmation before writing any files:

- **Interactive confirmation**: Running `bak restore <backup-id>` without flags displays a dry-run diff previewing exactly which files would be written, skipped, or modified, and prompts the user for interactive confirmation before applying.
- **`--dry-run` flag**: Previews exactly which files would be written and where, exiting without writing files or prompting.
- **`--force` flag**: Skips the interactive confirmation prompt (useful for scripting and non-interactive workflows). Manifest integrity and checksum checks remain strictly mandatory under `--force`.

This prevents accidental overwrites. There is no silent restoration path.

### Output Sanitization

- Error messages **never** include sensitive data (tokens, API keys, passwords)
- Secret patterns are redacted in all output (`ghp_***`, `sk-***`, etc.)
- Verbose mode (`--verbose`) gates diagnostic output, preventing accidental leakage

### Adapter Boundaries and Runtime State Exclusion

To prevent backup bloat, database corruption, and accidental leakage of ephemeral runtime execution logs, tool adapters use strict allowlists for root configuration files:

- **Codex adapter boundary**: The built-in `codex` adapter strictly allowlists conventional configuration, instruction, and hook files (`config.toml`, `config.json`, `config.yaml`, `config.yml`, `instructions.md`, `INSTRUCTIONS.md`, `AGENTS.md`, `agents.md`, `hooks.json`, `hooks.toml`, `hooks.yaml`, `hooks.yml`, and `mcp.json`).
- **Excluded state & size rationale**: Runtime state files (`*.sqlite`, `*.sqlite-wal`, `*.sqlite-shm`, `history.jsonl`, `session_index.jsonl`, `models_cache.json`, `installation_id`, `version.json`) are deliberately excluded. Because SQLite database files and session logs are actively written by running agent processes and regenerated at runtime, backing them up produced massive backups (e.g., a single `logs_2.sqlite` exceeding 140 MB, inflating backups to 145 MB) and risked writing inconsistent WAL/SHM state across machines.
- **Custom file escape hatch**: For non-standard or user-specific files (such as custom `*.config.toml` variations), users can override the built-in adapter by dropping a YAML adapter into `~/.config/bak/adapters/*.yaml` matching the adapter name (`codex`). The YAML schema allows defining custom `config_path`, and per-category `sub_path`, `is_dir`, `patterns` (globbing), and `root_files`.

## Known Limitations

- **Local Git required for undo**: The `bak undo` feature requires Git to be installed and operates on the `~/.bak` repository.
- **Target undo & drift protection**: `bak undo` projects pre-restore snapshots back to target configuration files and requires an exact match with post-restore state. If any target file has drifted (content changed, deleted, permission modified, or replaced with a directory or symlink), `bak undo` refuses all changes before writing any file. Best-effort limits: target undo requires un-drifted local state and a valid applied recovery point; concurrent external modifications during undo are not race-proof.
- **Token in environment**: `GITHUB_TOKEN` and other cloud provider credentials passed via environment variables are readable by any process with access to the user's environment.
- **Local backups at rest**: Backups stored locally under `~/.bak/backups/` are **never** encrypted on disk. AES-256-GCM encryption applies exclusively to cloud push/pull archives when configured per profile. Users must rely on OS filesystem permissions and full-disk encryption for local backup confidentiality.
- **Platform limits on permission assertions**: File permission preservation (`0.4.0` mode bits) is verified on non-Windows platforms (Linux and macOS); Windows filesystems do not support POSIX permission bits. Local automated suite proof is executed on Linux; Windows and macOS runtime behaviors remain CI evidence.

## Dependencies

Dependencies are reviewed before addition. The project policy is:

- Prefer Go standard library over third-party packages
- New dependencies must be justified (why stdlib is insufficient)
- Prefer well-maintained packages (>1000 stars, active commits)
- Dependencies are pinned with `go.sum` checksums

Run `go mod verify` to validate module integrity.
