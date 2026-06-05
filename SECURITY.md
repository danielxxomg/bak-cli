# Security Policy

## Supported Versions

Security updates are provided for the latest release only.

| Version | Supported          |
|---------|--------------------|
| 1.0.x   | :white_check_mark: |

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
- path.Clean + filepath.ToSlash for canonical path comparison
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

Instead of backing up real secrets, bak generates a `.env.example` template with redacted placeholder values. Secrets are **never** written to the backup directory.

### Checksum Integrity

- Every backed-up file gets a **SHA-256 checksum** computed at backup time and stored in `manifest.json`
- On restore, every file is verified against its stored checksum before being written
- Checksum mismatches block the restore and produce a clear error message

### Git Safety Net

Restore operations are protected by automatic Git operations:

- **Pre-restore commit**: The current state of `~/.config/opencode/` is committed to a local Git repository before any files are changed
- **Post-restore commit**: The restored state is committed after all files are written
- **`bak undo`**: Reverts the restore commit via `git revert` — safe, non-destructive, and history-preserving
- **No force-push**: The tool never force-pushes or rewrites Git history

### Mandatory Dry-Run

`bak restore` cannot be run without either:

- `--dry-run` flag — previews exactly which files would be written and where
- `--force` flag — explicitly acknowledges the user has reviewed and accepts the changes

This prevents accidental overwrites. There is no silent restoration path.

### Output Sanitization

- Error messages **never** include sensitive data (tokens, API keys, passwords)
- Secret patterns are redacted in all output (`ghp_***`, `sk-***`, etc.)
- Verbose mode (`--verbose`) gates diagnostic output, preventing accidental leakage

## Known Limitations

- **Local Git required**: The undo feature requires Git to be installed and the config directory to be a Git repository
- **Token in environment**: `GITHUB_TOKEN` environment variable is readable by any process with access to the user's environment
- **No encryption at rest**: Backups stored in `~/.bak/backups/` are not encrypted on disk; rely on filesystem permissions and disk encryption for confidentiality

## Dependencies

Dependencies are reviewed before addition. The project policy is:

- Prefer Go standard library over third-party packages
- New dependencies must be justified (why stdlib is insufficient)
- Prefer well-maintained packages (>1000 stars, active commits)
- Dependencies are pinned with `go.sum` checksums

Run `go mod verify` to validate module integrity.
