# Changelog

All notable changes to bak-cli will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] — 2026-06-04

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

See [README.md](README.md#roadmap) for planned features in v1.1 and v2.0.
