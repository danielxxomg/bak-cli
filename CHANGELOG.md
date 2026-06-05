# Changelog

All notable changes to bak-cli will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] — 2026-06-05

### Added

- **Multi-agent backup** — `bak backup` now auto-detects 8 AI coding tools in priority order:
  Claude Code, Cursor, Codex, Windsurf, Kiro, KiloCode, pi.dev, and OpenCode.
  Each agent backs up its own config files, skills, commands, rules, and extensions.
- **Cloud provider abstraction** — `Provider` interface and `ProviderRegistry` pattern
  (`internal/cloud/provider.go`) decouples push/pull/list from any single backend.
- **New cloud providers**:
  - `github-repo` — GitHub repository Contents API (file-level push/pull)
  - `codeberg` — Codeberg API (Gitea-compatible, `CODEBERG_TOKEN` env support)
  - `gitea` — Self-hosted Gitea/Forgejo with configurable base URL (`GITEA_TOKEN` env)
  - `rclone` — Shells out to `rclone` binary; supports Google Drive, OneDrive, S3, etc.
- **`--provider` flag** on `push`, `pull`, `list`, and `login` commands:
  `bak push --provider codeberg`, `bak list --provider github-gist`
- **Per-provider token resolution** via `ResolveProviderToken()` — supports
  `GITHUB_TOKEN`, `CODEBERG_TOKEN`, `GITEA_TOKEN` env vars and nested config keys.
- **`bak login --provider`** — interactive GitHub login; other providers directed to
  `bak config set providers.<name>.token`.
- **Cloud listing** — `bak list --provider <name>` displays backups stored on the
  selected cloud backend with IDs, dates, hosts, and sizes.
- **Config schema migration v0.1.0 → v0.2.0** — auto-detected on `Load()`,
  writes `config.json.v010.bak` before migrating flat `github_token`/`gist_id`
  to nested `providers.github.token`/`providers.github.gist_id`.
- **7 new agent adapters** under `internal/adapters/` with full test coverage:
  `claudecode`, `cursor`, `codex`, `windsurf`, `kiro`, `kilocode`, `pidev`.
- **`register.All()`** wire-up function in `internal/adapters/register/` for
  priority-ordered adapter registration.

### Changed

- `cmd/backup.go` now uses `register.All(reg)` instead of manual adapter registration.
- `cmd/push.go` and `cmd/pull.go` use `ProviderRegistry` with `--provider` flag
  (default `github-gist`) instead of direct Gist calls.
- `cmd/login.go` accepts `--provider` flag; non-GitHub providers redirect to `bak config set`.
- `cmd/list.go` has `--provider` flag for cloud listing; defaults to local listing.
- `config.Config` now includes `SchemaVersion` and `Providers map[string]ProviderConfig`.
- `config.Get()`/`Set()` support nested keys (`providers.github.token`, etc.).
- Existing `GitHubToken`/`GistID` fields kept as compat shim; auto-migrated on Load.

## [0.1.0] — 2026-06-04

### Added

- **Backup with presets** (`quick`, `full`, `skills`) via `bak backup [--preset]` — scans and copies AI coding configuration files into `~/.bak/backups/<id>/`
- **Restore with mandatory dry-run** via `bak restore [--dry-run] [--force] <id>` — previews changes before applying; bare `bak restore` requires explicit `--force` without `--dry-run`
- **GitHub Gist cloud sync** via `bak push [id]` and `bak pull [id]` — push backups to private Gists, pull them to a new machine
- **Interactive TUI picker** (`bak pick`) — powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea), select backup presets and categories interactively
- **Export as tar.gz** via `bak export <id> [--output path]` — portable archive for offline transfers
- **Git-backed safety** — auto-commit before and after restore operations; `bak undo` reverts via `git revert` (safe, non-destructive)
- **Secret detection and exclusion** — automatically detects patterns like `ghp_*`, `sk-*`, `sk-ant-*` and generates `.env.example` templates instead of backing up real secrets
- **Cross-platform path normalization** — native support for Windows (`\`), macOS, and Linux (`/`) path separators using canonical path comparison
- **Comprehensive CLI** with `bak backup`, `bak restore`, `bak list`, `bak push`, `bak pull`, `bak pick`, `bak export`, `bak login`, `bak undo`, `bak version`
- **Adapter architecture** — `adapters.Adapter` interface supports future AI coding tools (OpenCode v1, designed for Claude Code, Cursor, etc.)
- **Manifest system** — JSON manifest with SHA-256 integrity checksums for every backed-up file
- **Authentication** — `bak login` for interactive GitHub token setup, plus `GITHUB_TOKEN` env var and config file support
- **Version info** — `bak version` reports binary version, commit hash, and build date
- **`--verbose` flag** on all commands for diagnostic output
- **CI pipeline** — GoReleaser cross-platform build matrix (Linux, macOS, Windows × amd64, arm64)
- **Full test suite** with table-driven tests, >80% coverage target, and cross-platform path tests

## Roadmap

See [README.md](README.md#roadmap) for planned features in v0.2.0 and v1.0.0.
