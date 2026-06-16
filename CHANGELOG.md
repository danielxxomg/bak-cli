# Changelog

All notable changes to bak-cli will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.4.1] — 2026-06-16

### Fixed

- Corrected misleading E2E testscripts (`backup_restore_roundtrip`, `diff_two_backups`,
  `backup_verify_roundtrip`) that asserted stale behavior instead of exercising real
  round-trips. All three now run actual commands and verify output.
- goimports formatting in `cmd/schedule_test.go`.

### Added

- **Cloud sync integration test** — `internal/actions/cloud_sync_test.go` exercises
  full push→pull round-trip with `MockProvider`, including error paths (401, 404).
- **TUI smoke tests** — `tests/e2e/smoke_test.go` verifies binary launches without
  panic (`--help`, no-args fallback, unknown command).
- **Schedule happy-path tests** — table-driven tests with mock scheduler injection
  for create, list, and remove operations.
- **Error path tests** — `TestRunExport_CreateError` and `TestCreateTarGz_GzipCloseError`
  covering previously untested failure modes.
- **Config isolation** — all schedule tests now use `cmdDeps` injection instead of
  depending on real `~/.config/bak/`.

### Docs

- Fixed test badge, scoop bucket URL, security version references, and project
  structure in README.

## [1.4.0] — 2026-06-16

### Added

- **Full interactive TUI** — bare `bak` (no args) launches a Rose Pine-themed
  dashboard with keyboard-driven navigation across 8 screens:
  - **Dashboard** — table view of local backups with search filtering
  - **Cloud** — provider status, sync state, push/pull actions
  - **Health** — async checks for config, git, cloud connectivity
  - **Settings** — toggle options (provider, theme, auto-sync, verbose)
  - **Progress** — spinner + progress bar for async operations
  - **Shortcuts** — keybinding reference overlay
  - **Welcome** — first-run onboarding screen
  - **Pick/Wizard** — migrated to shared TUI theme
- **Toast notifications** — non-blocking feedback for actions (backup complete,
  push success, errors) with auto-hide after 3 seconds.
- **Search filtering** — `SetFilter()` on dashboard with case-insensitive substring
  matching across all columns. Activated with `/`, deactivated with `Esc`.
- **Action dispatch** — `RouteSelection()` pure function routes TUI menu selections
  to cobra actions (backup, restore, push, pull) after TUI exit.
- **`internal/tui/` package** — organized into:
  - `styles/` — Rose Pine palette (11 semantic colors), package-level lipgloss styles,
    ASCII logo with gradient, frame helper
  - `components/` — reusable stateless renderers (menu, checkbox, radio, help, toast, search)
  - `screens/` — screen-specific models and layout logic
  - `model.go` — root model with 8-screen routing, lazy sub-model init, WindowSizeMsg forwarding
  - `dispatch.go` — `RouteSelection()` pure function for post-TUI action routing
  - `deps.go` — `Deps` struct for dependency injection, `MenuSelection` type
- **Bubbletea v2 integration** — `charm.land/bubbletea/v2 v2.0.7`,
  `charm.land/lipgloss/v2 v2.0.3`, `charm.land/bubbles/v2 v2.1.0` (table, spinner,
  progress, textinput).
- **TUI responsiveness** — handles `tea.WindowSizeMsg`, stores width/height, adapts
  bordered content to terminal dimensions, shows "terminal too small" below 20×10.

### Changed

- `cmd/root.go` — bare `bak` now launches TUI when no args and isTTY, falls back
  to cobra help otherwise.
- `cmd/tty.go` — `defaultRunTUI` creates `tui.NewModel(deps)`, runs `tea.Program`,
  then dispatches via `tui.RouteSelection()`.
- `cmd/pick.go` and `cmd/wizard.go` migrated to shared TUI theme and components
  (visual consistency, no behavior change).

## [1.3.0] — 2026-06-08

### Added

- **Config-driven plugin system** — Custom backup presets and adapters via YAML
  declarations. No Go code required to extend bak-cli with new tools or categories.
  - `~/.config/bak/presets/*.yaml` — define custom backup presets with name and
    category list. Use `bak backup --preset <name>` to invoke.
  - `~/.config/bak/adapters/*.yaml` — define custom adapters with config paths
    and category patterns. Auto-detected alongside built-in adapters.
  - `--override` flag on `backup` and `restore` commands — prefer custom YAML
    presets/adapters over same-named built-ins.
  - See `examples/presets/custom.yaml` and `examples/adapters/myapp.yaml` for
    annotated samples.
- **`internal/presets` package** — YAML preset loader (`LoadFromDir`) with type
  definitions (`YAMLPreset`, `YAMLMetadata`). `ResolveAll()` merges custom and
  built-in presets with conflict detection.
- **`internal/adapters` YAML support** — `ConfigAdapter` implementing the
  `Adapter` interface for tools declared in YAML. `LoadYAMLAdapters()` scans
  and parses adapter definitions. `RegisterOrReplace()` handles override logic.
- **`internal/adapters/register` package** — `LoadYAMLAdapters()` wraps
  `adapter.LoadYAMLAdapters()` with registry integration and override warnings.
- **Backup scheduling** — `bak schedule` commands manage OS-native backup schedules
  using crontab on Unix and schtasks on Windows. Schedules run `bak backup && bak push`
  for a profile at configurable intervals.
  - `bak schedule create <profile> --every daily|weekly|every-12h|every-6h`
  - `bak schedule list` — table view of all active bak-cli schedules
  - `bak schedule remove <profile>` — delete a scheduled task
- **`internal/schedule` package** — `Scheduler` interface with platform-specific
  implementations (`CronScheduler` on Unix, `SchtasksScheduler` on Windows) using
  build tags. Includes `MockScheduler` for testing.
- **Schedule configuration** — `config.ScheduleConfig` struct with `Enabled` and
  `Interval` fields stored per profile. Schedule state is persisted in config.json
  alongside the OS-native task.
- **Interactive wizard** — `bak profile create --interactive` and `bak login --interactive`
  launch a step-by-step TUI wizard (Bubble Tea) for provider selection, preset choice,
  adapter toggling, and category selection. Keyboard-driven with arrow keys, space to
  toggle, enter to advance.
- **`cmd/wizard.go`** — `wizardModel` (bubbletea.Model) with 5-step flow: provider →
  preset → adapters → categories → confirm. Shared by profile create and login commands.
- **`--interactive` flag** on `profile create` and `login` commands — launches the
  wizard instead of requiring CLI flags. Provider list auto-populated from configured
  providers.
- **`bak verify <id>` command** — verifies backup integrity by checking SHA-256 hashes
  of every file in the manifest against files on disk. Exits 0 on success, 1 on first
  hash mismatch. Supports `--verbose` for per-file progress.
- **`bak diff <id1> <id2>` command** — compares two backups and shows file-level
  differences grouped by category: Added, Removed, Modified, and Unchanged.
- **`internal/diff` package** — `Compare()` function that flattens two manifests into
  canonical path maps and categorizes items by presence and SHA-256 hash comparison.
- **`internal/backup.ResolveBackupID()` shared helper** — validates backup IDs with
  path traversal prevention, replacing duplicated logic in `restore` command.

### Changed

- `config.ProfileConfig` now includes `Schedule *ScheduleConfig` field for persisting
  schedule state per profile.
- `cmd/restore.go` refactored to use shared `ResolveBackupID()` instead of inline
  BakDir + traversal guard + existence check. Behavior-preserving.
- `cmd/backup.go` — thin wire updated to call `presets.ResolveAll()` and
  `register.LoadYAMLAdapters()` for YAML integration.
- `cmd/restore.go` — supports `--override` flag.
- `internal/adapters/registry.go` — added `RegisterOrReplace()` for conflict
  resolution with YAML adapter overrides.

## [1.2.0] — 2026-06-07

### Added

- **Extracted action structs** — Core workflows (backup, restore, push, pull)
  moved to `internal/actions/` with injectable filesystem and config dependencies.
  Enables full unit-test coverage with mock implementations.
- **`internal/actions` package** — `BackupAction`, `RestoreAction`, `PushAction`,
  `PullAction` with `FileSystem` and `ConfigLoader` interfaces. Tests use
  `MockFileSystem` and `MockConfigLoader` for deterministic, isolated coverage.

### Changed

- `cmd/backup.go` — thin wire to `BackupAction`.
- `cmd/restore.go` — thin wire to `RestoreAction`.
- `cmd/push.go` — thin wire to `PushAction`.
- `cmd/pull.go` — thin wire to `PullAction`.

## [1.1.0] — 2026-06-06

### Added

- **QA stack** — Taskfile development workflow, golangci-lint integration,
  E2E test suite, fuzz testing, and benchmark framework.

## [1.0.0] — 2026-06-05

### Added

- **Stable release** — Production-grade CLI with 8 AI coding agent adapters,
  5 cloud backends, AES-256-GCM encryption, and machine profiles.
- All features from v0.1.0 through v0.3.0 stabilized and hardened.

## [0.3.0] — 2026-06-05

### Added

- **Encryption at rest** — AES-256-GCM encryption with Argon2id key derivation (64 MB RAM,
  3 iterations, 4 parallelism). Encrypted archives are prefixed with `BAK_ENC\x01` magic
  bytes for instant detection. Encryption is opt-in per profile.
- **`internal/crypto` package** — `Encrypt()`, `Decrypt()`, `IsEncrypted()`, `DeriveKey()`
  functions with full test coverage (round-trip, wrong password, magic byte integrity).
- **Password input strategy** — `crypto.GetPassword()` checks `BAK_ENCRYPTION_PASSWORD`
  env var first, falls back to interactive stdin prompt. Errors if no terminal and no
  env var (CI-safe).
- **Machine profiles** — `bak profile` commands:
  - `bak profile create <name> --provider <name> [--preset] [--adapters] [--categories] [--encrypt]`
  - `bak profile list` — table view of all configured profiles
  - `bak profile show <name>` — full profile details
  - `bak profile delete <name>` — remove a profile
- **Profile-scoped backups** — `bak backup --profile <name>` resolves the profile's
  preset, categories, and adapter list, overriding CLI flags. Works with `bak push`
  and `bak pull` for end-to-end encrypted cloud sync.
- **Profile-aware push/pull** — `bak push --profile <name>` and `bak pull --profile <name>`
  resolve the profile's encryption settings. Encrypted archives are transparently
  encrypted on push and decrypted on pull.
- **Config migration v0.2.0 → v0.3.0** — auto-detected on `Load()`, adds empty
  `profiles` map, bumps `schema_version` to `"0.3.0"`, writes `config.json.v020.bak`
  before overwriting. All existing providers preserved.

### Changed

- `config.Config` now includes `Profiles map[string]ProfileConfig` with fields:
  `Adapters`, `Categories`, `Preset`, `Provider`, and `Encryption` (with `Enabled`
  and optional `Password`, `Iterations`, `MemoryKiB`, `Parallelism`).
- `config.EncryptionConfig` adds `Enabled bool` field for explicit encryption opt-in.
- `cmd/backup.go` adds `--profile` flag that overrides `--preset` and `--adapter`
  when set. Profile preset/categories flow into `backup.Engine`.
- `cmd/push.go` adds `--profile` flag (default `"default"`); encrypts archive
  before push when profile has encryption enabled.
- `cmd/pull.go` adds `--profile` flag; detects encrypted archives via magic bytes
  and decrypts on the fly. Plaintext archives from v0.2.0 are handled transparently.
- `internal/manifest/manifest.go` adds `Encryption` struct (algorithm, KDF, salt,
  nonce, iterations, memory, parallelism) for encrypted backup auditability.

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
