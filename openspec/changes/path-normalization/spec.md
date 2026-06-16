# Path Normalization Specification

## Purpose

Centralize OS-independent path slash conversion so canonical path comparisons behave identically on Windows, macOS, and Linux.

## Requirements

### Requirement: Slash Helper

The system MUST export a `paths.Slash(p string) string` helper that replaces every `\` with `/` using `strings.ReplaceAll`.

#### Scenario: Windows path on Linux

- GIVEN a Windows-style path `C:\Users\alice\.config`
- WHEN `paths.Slash` is called
- THEN the result is `C:/Users/alice/.config`

#### Scenario: Empty string

- GIVEN an empty string
- WHEN `paths.Slash` is called
- THEN the result is an empty string

#### Scenario: Unix path passthrough

- GIVEN a path containing only forward slashes `/home/alice/.config`
- WHEN `paths.Slash` is called
- THEN the result is unchanged

### Requirement: filepath.ToSlash Replacement

The system MUST NOT contain any `filepath.ToSlash` calls inside `internal/` after the change. All existing call sites MUST use `paths.Slash` or an inline `strings.ReplaceAll` equivalent when importing `paths` would create a cycle.

#### Scenario: Zero violations

- GIVEN the codebase after migration
- WHEN `grep -r 'filepath\.ToSlash' internal/` is executed
- THEN no matches are found

#### Scenario: Import-cycle avoidance

- GIVEN a package that cannot import `internal/paths` due to a dependency cycle
- WHEN `filepath.ToSlash` is replaced
- THEN an inline `strings.ReplaceAll(p, "\\", "/")` is used with a code comment explaining the exception

### Requirement: Canonical Path Form

The system MUST apply `path.Clean` after slash conversion whenever a canonical cross-platform path form is required.

#### Scenario: Canonical normalization

- GIVEN a raw path string with mixed separators and redundant segments
- WHEN canonical form is computed
- THEN `path.Clean(strings.ReplaceAll(p, "\\", "/"))` is used, not `filepath.ToSlash`

### Requirement: Test Compatibility

The system MUST pass all existing tests. Mechanical renames of replaced helpers (e.g., `filepath.ToSlash` → `paths.Slash`) in test files are permitted as long as test logic and assertions remain unchanged.

#### Scenario: Existing suite passes

- GIVEN the current test suite
- WHEN tests are executed on Windows, macOS, and Linux
- THEN all existing tests pass

#### Scenario: Mechanical renames permitted

- GIVEN a test file that calls `filepath.ToSlash`
- WHEN the helper is migrated to `paths.Slash`
- THEN the test file MAY be updated to use the new helper, provided test logic and assertions are unchanged

### Requirement: Cross-Platform Slash Tests

The system MUST include new table-driven tests for `paths.Slash` that exercise Windows-style inputs on every OS.

#### Scenario: Windows inputs on all platforms

- GIVEN inputs `C:\Users\foo`, `a\\b\\c`, `\`, and `no-backslash`
- WHEN `paths.Slash` is invoked
- THEN results are `C:/Users/foo`, `a//b//c`, `/`, and `no-backslash` on all platforms

### Requirement: AGENTS.md Compliance

The change MUST satisfy the AGENTS.md rule that prohibits `filepath.ToSlash` for canonical path comparison.

#### Scenario: GGA validation

- GIVEN a GGA or linter rule that bans `filepath.ToSlash` in `internal/`
- WHEN the rule is evaluated against the codebase
- THEN zero violations are reported
