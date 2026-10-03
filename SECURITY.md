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

### Secret Detection and Exclusion

The backup engine detects common secret patterns and excludes them from backups:

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

Instead of backing up real secrets, bak generates a `.env.example` template with redacted placeholder values. For files containing recognized token families (`ghp_*`, `gho_*`, `ghu_*`, `ghs_*`, `ghr_*`, `sk-*`, `sk-ant-*`, `xoxb-*`, `xoxp-*`) and standard assignment patterns, matching secrets are never written to the backup directory. Unrecognized secret formats, custom token formats, or keys from unlisted providers outside these families are not detected and must be managed or excluded manually.

### Checksum and Manifest Integrity

- **SHA-256 checksums**: Every backed-up file gets a SHA-256 checksum computed at backup time and stored in `manifest.json`.
- **Integrity verification**: On restore, every file is verified against its stored checksum before being written. Checksum mismatches block the restore and produce a clear error message.
- **Mandatory under `--force`**: Manifest and checksum integrity verification cannot be bypassed. The `--force` flag skips interactive confirmation only, never integrity checks.
- **Permission preservation (0.4.0)**: Manifest schema `0.4.0` preserves portable file permission bits (`Mode`) at backup time and reapplies them during restore. Any chmod failures are reported as restore errors.
- **Degraded 0.3.0 handling**: Legacy `0.3.0` manifests lacking mode metadata are loaded and restored in degraded mode, explicitly warning the user that restored files lack original mode metadata rather than claiming exact permission restoration.

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
- **`bak undo`**: Reverts the last state in the `~/.bak` repository via `git revert` — safe, non-destructive, and history-preserving.
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

## Known Limitations

- **Local Git required for undo**: The `bak undo` feature requires Git to be installed and operates on the `~/.bak` repository.
- **Target undo limitation (pending T14)**: Target recovery and automatic rollback protect active restore failures. However, user-initiated `bak undo` currently operates on the `~/.bak` repository only and does not yet project reverts back to external target configuration files. Target-level undo with drift protection is deferred to a future stabilization unit (T14).
- **Token in environment**: `GITHUB_TOKEN` and other cloud provider credentials passed via environment variables are readable by any process with access to the user's environment.
- **Local backups at rest**: Backups stored locally under `~/.bak/backups/` are **never** encrypted on disk. AES-256-GCM encryption applies exclusively to cloud push/pull archives when configured per profile. Users must rely on OS filesystem permissions and full-disk encryption for local backup confidentiality.

## Dependencies

Dependencies are reviewed before addition. The project policy is:

- Prefer Go standard library over third-party packages
- New dependencies must be justified (why stdlib is insufficient)
- Prefer well-maintained packages (>1000 stars, active commits)
- Dependencies are pinned with `go.sum` checksums

Run `go mod verify` to validate module integrity.
