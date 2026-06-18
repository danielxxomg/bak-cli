# Delta for bak-cli

## ADDED Requirements

### Requirement: Settings schema

The system MUST define a Settings struct in `internal/config/config.go` with the following fields: `default_preset` (string), `auto_sync` (bool), `exclude_patterns` ([]string), `max_file_size` (int64), `confirm_destructive` (bool), `verbose_default` (bool), `default_provider` (string). Settings MUST persist to `~/.config/bak/config.json` via Load/Save.

#### Scenario: Settings round-trip

- GIVEN the user toggles `auto_sync` to true in the TUI Settings screen
- WHEN settings are saved
- THEN `config.json` SHALL contain `"auto_sync": true`
- AND relaunching the TUI SHALL show `auto_sync` as enabled

#### Scenario: Default settings

- GIVEN no `config.json` exists
- WHEN settings are loaded
- THEN defaults SHALL be: `default_preset="quick"`, `auto_sync=false`, `max_file_size=1048576`, `confirm_destructive=true`

### Requirement: Cloud screen data

The system MUST populate the cloud screen with real provider data from config instead of rendering an empty `CloudInfo{}`.

#### Scenario: Cloud screen with provider configured

- GIVEN `github.token` is set in config
- WHEN the user navigates to the Cloud screen
- THEN the screen SHALL display the provider name, token validity status, and last sync time

#### Scenario: Cloud screen without provider

- GIVEN no token is configured
- WHEN the user navigates to the Cloud screen
- THEN the screen SHALL display "No cloud provider configured" with instructions to run `bak login`

## MODIFIED Requirements

### Requirement: Action DI wiring

The system MUST inject dependencies into `internal/actions/` and `cmd/` instead of calling `os` or `hostname` directly. The `tui.Deps` struct MUST include `RunBackup`, `RunRestore`, `ListBackups`, `ListProfiles`, `GetCloudStatus`, `SaveSetting`, and `ConfigExists` function fields. `cmd/root.go` MUST inject all fields when launching the TUI.

(Previously: `cmd/root.go:34-38` only injected `Version`, `ConfigExists`, `ListBackups`; `RunBackup` was declared but never injected; `RunRestore`, `ListProfiles`, `GetCloudStatus`, `SaveSetting` did not exist)

#### Scenario: ProviderFactory injection

- GIVEN `Push` and `Pull` actions
- WHEN provider creation is needed
- THEN `ProviderFactory` interface is used — no direct `NewGitHubProvider` call

#### Scenario: restoreFile via FS

- GIVEN a `Restore` action with injected `FileSystem`
- WHEN `restoreFile()` copies a file
- THEN it reads from and writes to the injected `a.FS` — not `os.Open`/`os.Create`

#### Scenario: HostnameFunc injection

- GIVEN an `Actions` struct
- THEN `HostnameFunc` field supplies hostname for manifest metadata
- AND tests inject a static value without calling `os.Hostname`

#### Scenario: Mock provider compliance

- GIVEN a hand-rolled `MockProvider` implementing the provider interface
- THEN compile-time check `var _ Provider = (*MockProvider)(nil)` passes

#### Scenario: RegistryFactory injection

- GIVEN `ListCloudAction` struct
- THEN it accepts `RegistryFactory func() cloud.ProviderRegistry` as a field
- AND tests inject a factory returning mock registries

#### Scenario: CmdDeps struct

- GIVEN a command needs testable dependencies
- THEN `cmdDeps` in `cmd/deps.go` provides `ConfigLoader`, `Stdout`, `Stderr`, `Stdin` fields

#### Scenario: Wrapper pattern

- GIVEN a `runX` function is the cobra `RunE` target
- WHEN it executes
- THEN it delegates to `runXWithDeps(cmd, args, deps)` passing `defaultDeps`

#### Scenario: Test isolation

- GIVEN a test uses injected `CmdDeps`
- WHEN config loading occurs
- THEN it uses the mock loader, never calling `os.UserConfigDir`
- AND output writes to `deps.Stdout`/`deps.Stderr`, not real `os.Stdout`/`os.Stderr`

#### Scenario: TUI Deps injection

- GIVEN `cmd/root.go` launches the TUI
- WHEN `tui.Deps` is constructed
- THEN ALL function fields (`RunBackup`, `RunRestore`, `ListBackups`, `ListProfiles`, `GetCloudStatus`, `SaveSetting`, `ConfigExists`) SHALL be populated with real implementations
