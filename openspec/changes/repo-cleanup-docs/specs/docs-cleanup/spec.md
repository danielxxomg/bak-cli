# Delta for Docs Cleanup

## ADDED Requirements

### Requirement: Documentation Accuracy

All project documentation MUST accurately reflect current standards, tools, and approved patterns.

#### Scenario: Forbidden function reference

- GIVEN `filepath.ToSlash` is a forbidden function per project rules
- WHEN any `.md` file references or recommends `filepath.ToSlash`
- THEN the reference MUST be removed or replaced with the approved alternative

#### Scenario: Go version reference

- GIVEN the project uses Go 1.25
- WHEN any `.md` file references the required Go version
- THEN it MUST state `1.25` or `1.25+`

#### Scenario: Build tool reference

- GIVEN the project uses `task` as its build tool
- WHEN CONTRIBUTING.md or any `.md` file references build commands
- THEN it MUST reference `task` commands and MUST NOT reference `make` commands

### Requirement: CHANGELOG Structure

The CHANGELOG MUST group released versions under dated versioned sections.

#### Scenario: Released versions have sections

- GIVEN versions v1.0.0 through v1.3.0 have been released
- WHEN CHANGELOG.md is inspected
- THEN each released version MUST appear under its own dated `[vX.Y.Z]` section
- AND only genuinely unreleased items MUST remain under `[Unreleased]`

### Requirement: openspec Hygiene

The openspec directory MUST remain consistent with the project state.

#### Scenario: Stale changes archived

- GIVEN completed changes exist in `openspec/changes/`
- WHEN the repository is in a clean state
- THEN stale changes MUST be moved to `openspec/changes/archive/`

#### Scenario: Config version matches runtime

- GIVEN `openspec/config.yaml` declares a Go version
- WHEN that version is compared to `go.mod`
- THEN the versions MUST match exactly

### Requirement: File Naming Consistency

File names MUST accurately describe their contents.

#### Scenario: Preset name matches filename

- GIVEN a preset file exists at `examples/presets/custom.yaml`
- WHEN the file's internal `name` field is inspected
- THEN the `name` field MUST equal the filename stem (`custom`)

#### Scenario: Test files have descriptive names

- GIVEN a test file in `cmd/` covers wiring behavior
- WHEN the file is renamed to reflect its purpose
- THEN the new filename MUST describe the behavior under test (e.g., `wiring_test.go`)

### Requirement: Review Scope

Automated review configuration MUST include all relevant source files.

#### Scenario: Test files included in review

- GIVEN `.gga` configures automated review file patterns
- WHEN the exclude patterns are inspected
- THEN `*_test.go` MUST NOT be excluded from review
