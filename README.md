<p align="center">
  <img src="docs/brand/banner/bak-github-banner.png" alt="bak — Pack your AI coding setup. Move anywhere." width="100%">
</p>

<p align="center">
  <a href="https://goreportcard.com/report/github.com/danielxxomg/bak-cli"><img src="https://goreportcard.com/badge/github.com/danielxxomg/bak-cli" alt="Go Report Card"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"></a>
  <a href="https://github.com/danielxxomg/bak-cli/releases/latest"><img src="https://img.shields.io/github/v/release/danielxxomg/bak-cli" alt="Release"></a>
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go 1.25+">
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey" alt="Platform">
  <img src="https://img.shields.io/badge/tests-1235+-brightgreen" alt="Tests">
</p>

<p align="center">
  <strong>bak</strong> is a CLI tool that backs up, restores, and syncs your AI coding configuration across machines. Originating with OpenCode and expanded to support 8 AI coding tools — Claude Code, Cursor, Codex, Windsurf, Kiro, KiloCode, pi.dev, and OpenCode. Never lose your skills, MCP servers, plugins, agents, or config files again.
</p>

## Supported Platforms

| Platform | Install Method | Package Format |
|----------|---------------|----------------|
| macOS (arm64, amd64) | Homebrew, Go | `brew install --cask`, `go install` |
| Linux (arm64, amd64) | Homebrew, .deb, .rpm, Go | `brew install --cask`, `.deb`, `.rpm`, `go install` |
| Windows (arm64, amd64) | Scoop, Go | `scoop install`, `go install` |

## Features

- 🤖 **Multi-Agent Support** — Auto-detects 8 AI coding tools (originating with OpenCode): Claude Code, Cursor, Codex, Windsurf, Kiro, KiloCode, pi.dev, and OpenCode
- 🔄 **Backup & Restore** — Preset-based backups (quick, full, skills) with interactive confirmation and dry-run preview before restore
- 🔒 **Secret Detection & Redacted-in-Place Backup** — Automatically detects recognized secret families (GitHub, OpenAI, Anthropic, Slack, AWS, GCP, Stripe secret keys, connection strings with inline credentials, Bearer tokens). Instead of dropping files containing secrets, bak preserves configuration files structurally by replacing secrets in-place with `<YOUR_SECRET>` placeholders in the backup payload, labels them in the manifest schema 0.5.0, and generates `.env.example` templates. *Tradeoff:* Restoring a redacted file **overwrites the live file's real secrets with placeholders**. The user must re-enter them.
- ☁️ **Multi-Cloud Sync** — Push/pull backups to GitHub Gist, GitHub Repo, Codeberg, Gitea/Forgejo, and rclone (Google Drive, S3, etc.)
- 🔐 **Cloud Archive Encryption** — AES-256-GCM encryption with Argon2id key derivation for cloud archives (opt-in per profile; local backups under ~/.bak remain plaintext)
- 👤 **Machine Profiles** — `bak profile` commands to scope backups per machine with independent adapter, category, preset, provider, and encryption settings
- 🖥️ **Cross-Platform** — Works on Windows, macOS, and Linux with path normalization
- 🎯 **Interactive Picker** — TUI with bubbletea for selective category backup
- ↩️ **Undo** — Target-level undo with fail-closed drift protection and Git-backed rollback in ~/.bak via `bak undo`
- 📦 **Export** — Export backups as portable tar.gz archives

## Why bak?

There are many dotfile managers. **bak** is not one of them — it's purpose-built for AI coding setups, with features generic tools don't cover.

| Feature | bak | chezmoi | mackup | stow |
|---------|-----|---------|--------|------|
| AI agent auto-detection (8 agents) | ✅ | ❌ | ❌ | ❌ |
| Cloud sync (Gist, Codeberg, Gitea, rclone) | ✅ | ✅ (git) | ✅ (iCloud, etc.) | ❌ |
| Cloud archive encryption (AES-256-GCM) | ✅ | ❌ | ❌ | ❌ |
| Machine profiles | ✅ | ✅ (templates) | ❌ | ❌ |
| Secret detection (auto-exclude tokens) | ✅ | ❌ | ❌ | ❌ |
| Mandatory dry-run before restore | ✅ | ❌ | ❌ | ❌ |
| Git-backed undo | ✅ | ✅ (git) | ❌ | ❌ |
| YAML extensibility (presets, adapters) | ✅ | ❌ | ❌ | ❌ |

If you back up AI coding configs, bak is the only tool that auto-detects your agents, encrypts your data, and syncs across clouds — all with safety guarantees built in.

## Installation

### macOS / Linux (Recommended)

```bash
brew install --cask danielxxomg/tap/bak
```

### Windows (Recommended)

```bash
scoop bucket add danielxxomg https://github.com/danielxxomg/bak-cli-tap
scoop install bak
```

<details>
<summary>Alternative install methods</summary>

### Debian/Ubuntu

Download the `.deb` file from [GitHub Releases](https://github.com/danielxxomg/bak-cli/releases) and install:

```bash
sudo dpkg -i bak_*.deb
```

### RHEL/Fedora

Download the `.rpm` file from [GitHub Releases](https://github.com/danielxxomg/bak-cli/releases) and install:

```bash
sudo rpm -i bak-*.rpm
```

### Go

```bash
go install github.com/danielxxomg/bak-cli@latest
```

### From Source

```bash
git clone https://github.com/danielxxomg/bak-cli.git
cd bak-cli
go build -o bak .
```

</details>

## Quick Start

```bash
# Create a backup
bak backup

# Create a backup scoped to a machine profile
bak profile create work --provider github-gist --preset full --encrypt
bak backup --profile work

# Preview what would be restored
bak restore --dry-run 20260604-150405

# Restore a backup (shows diff and prompts for confirmation)
bak restore 20260604-150405

# Skip interactive confirmation (integrity checks remain mandatory)
bak restore --force 20260604-150405

# Undo the last restore (reverts ~/.bak history)
bak undo

# Sync to cloud (GitHub Gist, Codeberg, Gitea, rclone, etc.)
bak login
bak push --provider github-gist
bak pull

# Verify backup integrity
bak verify 20260604-150405
bak verify --verbose 20260604-150405

# Compare two backups
bak diff 20260604-150405 20260605-080000
```

## Commands

| Command | Description |
|---------|-------------|
| `bak backup [--preset quick\|full\|skills] [--profile <name>]` | Create a backup |
| `bak restore [--dry-run] [--force] <id>` | Restore a backup (shows diff and prompts; warns on version mismatch, fails on newer schema; --force skips confirmation only) |
| `bak undo` | Revert the last restore operation on target files and in ~/.bak with drift protection |
| `bak list [--provider <name>]` | List local or cloud backups |
| `bak pick` | Interactive TUI picker |
| `bak push [id] [--provider <name>] [--profile <name>]` | Push to a cloud backend |
| `bak pull [id] [--provider <name>] [--profile <name>]` | Pull from a cloud backend |
| `bak export <id> [--output path]` | Export as tar.gz |
| `bak login [--provider <name>]` | Authenticate with a cloud provider |
| `bak profile create\|list\|show\|delete` | Manage machine profiles |
| `bak verify [--verbose] <id>` | Verify backup integrity |
| `bak diff <id1> <id2>` | Show file-level differences between two backups |
| `bak version` | Show version info |
| `bak schedule create\|list\|remove` | Manage OS-native backup schedules |
| `bak wizard` | Launch the interactive profile/backup wizard |

## Configuration

### Storage Location

Backups are stored in `~/.bak/backups/<id>/`:

```
~/.bak/
├── config.json          # bak configuration
└── backups/
    └── 20260604-150405/
        ├── manifest.json
        ├── .env.example
        └── opencode/
            ├── skills/
            ├── commands/
            ├── plugins/
            └── config files...
```

### GitHub Token

For cloud sync, configure a GitHub token:

```bash
# Option 1: Interactive (GitHub only)
bak login

# Option 2: Environment variable
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx

# Option 3: Config file
bak config set github.token ghp_xxxxxxxxxxxx
```

### Cloud Providers

Use `--provider` to select a cloud backend for push/pull/list:

| Provider | Flag | Config Key | Env Token |
|----------|------|------------|-----------|
| GitHub Gist | `github-gist` (default) | `providers.github.token` | `GITHUB_TOKEN` |
| GitHub Repo | `github-repo` | `providers.github.token` + `.repo` | `GITHUB_TOKEN` |
| Codeberg | `codeberg` | `providers.codeberg.token` + `.repo` | `CODEBERG_TOKEN` |
| Gitea / Forgejo | `gitea` | `providers.gitea.token` + `.repo` + `.base_url` | `GITEA_TOKEN` |
| Rclone | `rclone` | `providers.rclone.remote` | — |

```bash
# Push to a specific provider
bak push --provider codeberg

# List cloud backups
bak list --provider github-gist

# Configure non-GitHub providers
bak config set providers.codeberg.token <your-token>
bak config set providers.codeberg.repo owner/backups
```

### Machine Profiles

Profiles let you scope backups to specific machines with independent settings
for adapters, categories, preset, provider, and encryption.

```bash
# Create a profile for your work laptop
bak profile create work-laptop --provider github-gist --preset full --encrypt

# Create a lightweight profile for your home PC
bak profile create home-pc --provider github-repo --preset quick

# Create a profile that only backs up OpenCode and Cursor config
bak profile create dev-box --provider codeberg --adapters opencode,cursor --categories config,skills

# List all profiles
bak profile list

# Show full profile details
bak profile show work-laptop

# Delete a profile
bak profile delete old-machine
```

Use a profile with `--profile` on `backup`, `push`, or `pull`:

```bash
bak backup --profile work-laptop
bak push --profile work-laptop
bak pull --profile work-laptop
```

When `--profile` is set, its preset, categories, and adapter list override
the equivalent CLI flags.

### Custom Presets

You can define custom backup presets as YAML files under
`~/.config/bak/presets/`. Each file defines a preset with a name and
category list.

**Example** (`~/.config/bak/presets/my-full.yaml`):

```yaml
name: my-full
categories:
  - config
  - skills
  - commands
  - plugins
  - agents

metadata:
  description: "Custom full preset without MCP servers"
  author: "you"
```

If a custom preset has the same name as a built-in (quick, full, skills),
use `--override` to prefer the custom version:

```bash
bak backup --preset full --override
```

Without `--override`, name conflicts produce an error so you don't
accidentally replace built-in behavior.

Custom presets are merged with built-ins: any preset name not matching
a built-in is treated as a custom preset loaded from YAML.

### Custom Adapters

You can register adapters for new tools without writing Go code by
placing YAML declarations in `~/.config/bak/adapters/`.

**Example** (`~/.config/bak/adapters/myapp.yaml`):

```yaml
name: myapp
config_path: .config/myapp

categories:
  - name: config
    root_files:
      - config.yaml
      - settings.json

  - name: skills
    sub_path: skills
    is_dir: true

  - name: commands
    sub_path: commands
    is_dir: true
```

`bak backup` will auto-detect your custom adapter if the `config_path`
directory exists under your home directory. Use `--adapter myapp` to
force it.

Custom adapters that share a name with a built-in adapter (e.g. `codex`, `opencode`)
replace the built-in adapter when `--override` is passed:

```bash
bak backup --adapter codex --override
```

The YAML schema supports `config_path`, and for each category:
- `sub_path`: relative path under `config_path`
- `is_dir`: boolean indicating if `sub_path` is a directory to recursively scan
- `patterns`: glob patterns to match files (e.g. `["*.config.toml"]`)
- `root_files`: explicit filenames at the config root

See `examples/presets/` and `examples/adapters/` for annotated samples.

### Backup Scheduling

Schedule automatic backups using OS-native task schedulers (crontab on
Linux/macOS, schtasks on Windows).

```bash
# Create a daily scheduled backup for a profile
bak schedule create work --every daily

# List all active bak-cli schedules
bak schedule list

# Remove a schedule
bak schedule remove work
```

Supported intervals: `daily`, `weekly`, `every-12h`, `every-6h`.

Each schedule runs `bak backup --profile <name> && bak push --profile <name>`
at the configured interval.

### Interactive Wizard

Use `--interactive` on `profile create` or `login` to launch a step-by-step
TUI wizard powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea).

```bash
# Create a profile interactively (no flags needed)
bak profile create my-machine --interactive

# Login with provider selection wizard
bak login --interactive
```

The wizard walks through provider selection, preset choice, adapter toggling,
and category selection with keyboard navigation.

### Encryption

Encryption is enabled per profile with the `--encrypt` flag on `bak profile create`.
Encrypted archives use **AES-256-GCM** with **Argon2id** key derivation (64 MB RAM,
3 iterations, 4 parallelism).

> **Note**: Encryption applies exclusively to cloud push/pull archives. Backups stored locally under `~/.bak/backups/` remain unencrypted on disk; rely on OS filesystem permissions or full-disk encryption for local confidentiality.

| Feature | Detail |
|---------|--------|
| Algorithm | AES-256-GCM |
| Key derivation | Argon2id (64 MB, 3 iter, 4 parallel) |
| Magic bytes | `BAK_ENC\x01` — instant detection without parsing |
| Password input | Masked interactive prompt (stdin) or `BAK_ENCRYPTION_PASSWORD` env var |
| Backward compat | Plaintext archives from v0.2.0 are detected and handled automatically |
| Scope | Cloud archives only (local backups under `~/.bak/` remain plaintext) |

**Push flow**: `bak push --profile work` encrypts the tar.gz archive before upload.
**Pull flow**: `bak pull` detects magic bytes, prompts for password, decrypts on the fly.

```bash
# Set password via environment variable (CI/scripts)
export BAK_ENCRYPTION_PASSWORD="your-secure-password"
bak push --profile work

# Or use interactive prompt (no env var set)
bak push --profile work
# → Enter encryption password: ********
```

Encryption metadata (algorithm, KDF, salt, nonce) is stored in the backup manifest
for auditability. The password itself is never persisted to disk.

### Supported AI Coding Agents

Originating with OpenCode and expanded to 8 tools, `bak backup` auto-detects installed agents in priority order:

| Agent | Path | Priority |
|-------|------|----------|
| Claude Code | `~/.claude/` | 1 |
| Cursor | `~/.cursor/` | 2 |
| Codex | `~/.codex/` | 3 |
| Windsurf | `~/.codeium/windsurf/` | 4 |
| Kiro | `~/.kiro/` | 5 |
| KiloCode | `~/.kilocode/` | 6 |
| pi.dev | `~/.pi/` | 7 |
| OpenCode | `~/.config/opencode/` | 8 |

Force a specific adapter:
```bash
bak backup --adapter cursor
```

#### Codex Configuration & State Boundary

The built-in Codex adapter (`~/.codex/`) uses a strict allowlist to back up meaningful configuration while preventing backup bloat from runtime state:

- **Covered configuration**:
  - Tool configuration: `config.toml`, `config.json`, `config.yaml`, `config.yml` (`config` category)
  - Instructions: `instructions.md`, `INSTRUCTIONS.md`, `AGENTS.md`, `agents.md` (`config` category, so the default `quick` preset covers them); the `agents` category is the `agent/` directory of sub-agent definitions
  - Hooks: `hooks.json`, `hooks.toml`, `hooks.yaml`, `hooks.yml` (`config` category)
  - Model Context Protocol: `mcp.json` (`mcp` category)
- **Deliberately excluded runtime state**:
  - SQLite databases: `*.sqlite`, `*.sqlite-wal`, `*.sqlite-shm` (e.g., `logs_2.sqlite`, `state_5.sqlite`)
  - Session history: `history.jsonl`, `session_index.jsonl`
  - Caches & ephemeral metadata: `models_cache.json`, `installation_id`, `version.json`
- **Size rationale**: Runtime session databases and execution logs are dynamically regenerated at runtime. In older releases without an allowlist, capturing unmanaged SQLite files caused backups to balloon to ~145 MB (with a single `logs_2.sqlite` exceeding 140 MB). The allowlist ensures compact, fast, reproducible backups.
- **Escape hatch for custom files**: If your configuration includes additional non-standard files (such as custom `*.config.toml` variants), define a custom adapter in `~/.config/bak/adapters/codex.yaml` and pass `--override` to replace the built-in adapter. The YAML schema supports `patterns` (globs) and `root_files` without requiring built-in code changes.

## Architecture

```
bak-cli/
├── cmd/                    # CLI commands (cobra)
├── internal/
│   ├── adapters/           # Agent adapters (8 supported: Claude Code, Cursor, Codex,
│   │   │                   #   Windsurf, Kiro, KiloCode, pi.dev, OpenCode)
│   │   └── register/       # RegisterAll() wire-up
│   ├── backup/             # Backup engine + presets + secrets
│   ├── restore/            # Restore engine + dry-run + git safety
│   ├── manifest/           # Manifest schema + validation
│   ├── cloud/              # Cloud provider abstraction (GitHub Gist, GitHub Repo,
│   │                       #   Codeberg, Gitea/Forgejo, Rclone)
│   ├── crypto/             # AES-256-GCM encryption + Argon2id key derivation
│   ├── paths/              # Cross-platform path normalization
│   ├── git/                # Git operations (go-git)
│   ├── config/             # Configuration management + v0.1.0 → v0.3.0 migration
│   ├── presets/            # Preset definitions
│   └── schedule/           # OS-native task scheduling (crontab / schtasks)
├── .goreleaser.yaml        # Cross-platform release config
└── Taskfile.yml             # Development workflow targets
```

### Data Flow

```mermaid
graph LR
    subgraph "bak backup"
        A[Detect Adapters] --> B[Resolve Preset]
        B --> C[Copy Files]
        C --> D[Scan Secrets]
        D --> E[Generate Manifest]
        E --> F[Auto-commit]
    end

    subgraph "bak restore"
        G[Load Manifest] --> H[Validate Checksums]
        H --> I[Compute Dry-run]
        I --> J{User Confirms?}
        J -->|Yes| K[Apply Restore]
        J -->|No| L[Cancel]
    end

    subgraph "Cloud Sync"
        N[bak push] --> O[Package tar.gz]
        O --> P[Upload to Gist]
        Q[bak pull] --> R[Download Gist]
        R --> S[Extract & Restore]
    end

    subgraph "Safety"
        T[bak undo] --> U[git revert HEAD]
    end
```

### Adapter Pattern

```mermaid
classDiagram
    class Adapter {
        <<interface>>
        +Name() string
        +Detect(homeDir) bool
        +ListItems(homeDir, categories) []Item
        +Backup(homeDir, backupDir, items) error
        +Restore(homeDir, backupDir) error
    }

    class OpenCodeAdapter {
        +Name() "opencode"
        +Detect() ~/.config/opencode
    }

    class Registry {
        +Register(adapter)
        +DetectAll(homeDir) []DetectedAdapter
        +Get(name) Adapter
    }

    Adapter <|.. OpenCodeAdapter
    Registry --> Adapter
```

## Safety Guarantees

- ✅ **Interactive confirmation & dry-run** — Always preview changes before restore; interactive confirmation required unless bypassed with `--force`
- ✅ **Mandatory integrity** — SHA-256 checksum and manifest integrity checks cannot be bypassed, even under `--force` (which only skips the confirmation prompt)
- ✅ **Target recovery & automatic rollback** — Before modifying target files on restore, captures affected target pre-state in private local recovery storage (`~/.bak/recovery/<point-id>`) with restricted permissions (0700/0600); stops at first copy or chmod failure and attempts automatic rollback of all attempted targets
- ✅ **Permission preservation (0.4.0)** — Manifest schema 0.4.0 records portable file permission mode bits and reapplies them on restore; legacy 0.3.0 manifests restore in degraded mode with an explicit warning
- ✅ **Target-level undo & drift protection** — `bak undo` restores actual target configuration files to their pre-restore state using local recovery snapshots, while creating a revert commit in `~/.bak`. If any target file was modified, deleted, created, or replaced since the restore, undo fails closed before writing any files to protect subsequent user work
- ✅ **Version compatibility & schema gating** — Warns on `stderr` when restoring backups created by a different or unversioned/development `bak` version; fails closed before any target write or recovery preparation if the manifest schema is newer than supported (`0.4.0`), prompting the user to upgrade
- ✅ **Secret exclusion** — Automatically detects recognized token families (GitHub `ghp_*`, `gho_*`, `ghu_*`, `ghs_*`, `ghr_*`, OpenAI `sk-*`, Anthropic `sk-ant-*`, Slack `xoxb-*`, `xoxp-*`, AWS access key IDs `AKIA*` / `ASIA*`, GCP API keys `AIza*`, Stripe secret keys `sk_*` / `rk_*`, connection strings with inline credentials, and Bearer tokens), names excluded files in the backup summary, and generates `.env.example` templates with redacted placeholders instead of storing real secrets. Stripe publishable `pk_*` keys are explicitly not treated as secrets (preserving working client-side configuration). Tradeoff: every added pattern excludes matching files from the backup, so a false positive silently removes a file from protection. Restore dry-run distinguishes secret-excluded files and reminds users to re-enter values manually
- ✅ **Symlink traversal & containment** — Scanning follows symlinks within the user home directory; symlinks pointing to directories are traversed recursively, and symlinks to regular files are hashed and backed up as files. Broken symlinks or symlinks pointing outside the home directory are skipped (with a warning when verbose mode is enabled). *Tradeoff:* Following symlinks allows backing up shared or centralized configurations (such as skills shared across agents) stored outside an adapter's own config directory, as long as the target remains under the user home.
- ✅ **Path validation** — Prevents path traversal attacks by validating that all restored paths stay within the user home directory
- ✅ **Executable journey matrix proof** — Validated by an eight-stage real-binary journey test suite (`tests/e2e/journey_matrix_test.go`): discovery, mutation/deletion diff recovery, dry-run zero-write guarantees, apply correctness (with POSIX permission bit preservation on non-Windows platforms), checksum verification, tamper fail-closed rejection, partial failure rollback, and target undo drift protection (verified locally on Linux; Windows and macOS behaviors are validated in CI)

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, code style, adapter implementation guide, and PR process.

Quick start: fork → branch → commit (conventional commits) → push → PR.

## Next Steps

- **Interactive setup?** Run `bak wizard` for a step-by-step TUI setup.
- **Automated backups?** Set up a [schedule](#backup-scheduling) with `bak schedule create`.
- **Custom backup presets?** Add YAML files to `~/.config/bak/presets/`. See [Custom Presets](#custom-presets).
- **Support a new tool?** Register a [custom adapter](#custom-adapters) in `~/.config/bak/adapters/`.

## Roadmap

### Completed ✅
- v1.3.0 — Multi-OS package manager support (Homebrew, Scoop, deb, rpm)
- v1.2.x — DI refactor, CI hardening, test coverage bump
- v1.1.0 — QA stack (Taskfile, golangci-lint, E2E, fuzz, benchmarks)
- v1.0.0 — Stable release (8 adapters, 5 cloud backends, encryption, profiles)
- v0.3.0 — Encryption at rest + machine profiles
- v0.2.0 — Multi-agent + cloud backends

### Future
- [ ] homebrew-core submission
- [ ] scoop-extras submission
- [ ] winget, AUR, nix support
- [ ] Plugin system for custom backup strategies

<details>
<summary>Brand Assets</summary>

Visual assets are in `docs/brand/`:

| Asset | File | Usage |
|-------|------|-------|
| Wordmark (color) | `logo/bak-wordmark-color.png` | Primary brand mark |
| Wordmark (mono) | `logo/bak-wordmark-mono-white.png` | Dark backgrounds, print |
| GitHub Banner | `banner/bak-github-banner.png` | Social preview, README |
| Icon (geometric) | `icon-secondary/bak-icon-geometric.png` | Official icon, favicons |
| Icon (friendly) | `icon-secondary/bak-icon-friendly.png` | Stickers, swag, presentations |
| Favicon 32px | `favicon/bak-favicon-32.png` | Browser tab, small icon |
| Favicon 16px | `favicon/bak-favicon-16.png` | Browser tab (tiny) |

</details>

## License

MIT License — see [LICENSE](LICENSE) for details.
