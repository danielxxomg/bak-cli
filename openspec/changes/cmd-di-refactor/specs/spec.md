# Delta for cmd-di-refactor

## ADDED Requirements

### Requirement: CmdDeps struct

The system MUST provide a `CmdDeps` struct in `cmd/deps.go` with `ConfigLoader`, `Stdout`, `Stderr`, and `Stdin` fields.

#### Scenario: Default dependencies

- GIVEN a command needs production behavior
- WHEN `defaultDeps()` is called
- THEN it returns `CmdDeps` with real `config.Load`, `os.Stdout`, `os.Stderr`, and `os.Stdin`

#### Scenario: Test dependencies

- GIVEN a test needs isolated config
- WHEN `CmdDeps` is constructed manually
- THEN it accepts a mock `ConfigLoader` and `bytes.Buffer` for I/O

### Requirement: Wrapper pattern

The system MUST refactor every `runX` function to delegate to `runXWithDeps(cmd, args, deps)`.

#### Scenario: Entry point unchanged

- GIVEN `runX` is the cobra `RunE` target
- WHEN it executes
- THEN it calls `runXWithDeps` passing `defaultDeps()`

#### Scenario: Implementation uses deps

- GIVEN `runXWithDeps` is called
- WHEN config is loaded
- THEN it uses `deps.ConfigLoader` instead of direct `config.Load()`

### Requirement: Test isolation

The system MUST use injected `CmdDeps` in all `cmd/` tests instead of real filesystem calls.

#### Scenario: Config isolation

- GIVEN a test uses injected `CmdDeps`
- WHEN config loading occurs
- THEN it uses the mock loader, never calling `os.UserConfigDir`

#### Scenario: I/O capture

- GIVEN a test exercises a command with injected buffers
- WHEN output is written
- THEN it writes to `deps.Stdout`/`deps.Stderr`, not real `os.Stdout`/`os.Stderr`

## MODIFIED Requirements

### Requirement: Cross-platform path normalization

The system MUST use `strings.ReplaceAll(path, "\\", "/")` in `canonicalPath()` instead of `filepath.ToSlash()`.
(Previously: `filepath.ToSlash()` — OS-dependent, fails on Linux)

#### Scenario: Windows path on Linux

- GIVEN a Windows-style path `C:\Users\foo`
- WHEN `canonicalPath()` runs on Linux
- THEN backslashes are replaced with forward slashes

#### Scenario: Unix path unchanged

- GIVEN a Unix path `/home/foo`
- WHEN `canonicalPath()` runs
- THEN it remains `/home/foo`

### Requirement: Platform-specific test skipping

The system MUST skip Windows-specific tests on non-Windows via `runtime.GOOS` check.

#### Scenario: Windows-only test on Linux

- GIVEN a test requires Windows registry or `schtasks`
- WHEN `runtime.GOOS != "windows"`
- THEN it calls `t.Skip()` immediately

## Verification Requirements

### Requirement: CI pass

The system MUST pass `task test:linux` in Docker after the refactor.

#### Scenario: Linux CI

- GIVEN the change is applied
- WHEN `task test:linux` runs
- THEN all tests pass with zero failures

#### Scenario: Local Windows

- GIVEN the change is applied on Windows
- WHEN `task test` runs
- THEN all tests pass with zero new failures
