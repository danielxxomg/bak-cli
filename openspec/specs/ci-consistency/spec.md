# Delta Spec: ci-consistency

## ADDED Requirements

### REQ-CI-001: Go Version Consistency

All CI jobs in `.github/workflows/ci.yml` MUST use the same Go version, and that version MUST match the `go` directive in `go.mod`.

**Rationale**: Version mismatches between jobs cause inconsistent behavior — lint/test may pass on 1.25 while security/build silently use 1.24, masking compatibility issues.

#### Scenarios

**Scenario: All jobs use matching Go version**
- Given `go.mod` declares `go 1.25.0`
- When any CI job sets up Go
- Then the `go-version` value MUST be `'1.25'`

**Scenario: go.mod version changes**
- Given the `go` directive in `go.mod` is updated
- Then all CI job `go-version` values MUST be updated to match in the same change

---

### REQ-CI-002: Cross-Platform Binary Verification

The CI build verification step MUST use the correct binary name for each operating system.

**Rationale**: Go produces `bak` on Linux/macOS and `bak.exe` on Windows. Running `./bak.exe` on Unix fails because the file does not exist.

#### Scenarios

**Scenario: Unix runner verification**
- Given the CI runner OS is Linux or macOS
- When the binary verification step executes
- Then it MUST run `./bak version` (no `.exe` extension)

**Scenario: Windows runner verification**
- Given the CI runner OS is Windows
- When the binary verification step executes
- Then it MUST run `.\bak.exe version`

---

### REQ-CI-003: Taskfile Cross-Platform Binary Name

The Taskfile build output MUST produce the correct binary name for the host operating system.

**Rationale**: Hardcoding `bak.exe` breaks local development and CI on Linux/macOS where `.exe` is not the conventional extension.

#### Scenarios

**Scenario: Build on Unix**
- Given the host OS is Linux or macOS
- When `task build` executes
- Then the output binary MUST be named `bak`

**Scenario: Build on Windows**
- Given the host OS is Windows
- When `task build` executes
- Then the output binary MUST be named `bak.exe`

---

### REQ-CI-004: GGA PR Review job
**Priority**: should

The CI pipeline MUST include a non-blocking GGA review job that runs on pull requests using `--pr-mode --diff-only` flags.

**Scenario**: PR opened or updated
- GIVEN a pull request targets `main`
- WHEN the PR is opened or new commits are pushed
- THEN CI runs `gga run --pr-mode --diff-only` as a dedicated job
- AND the job result MUST NOT block merge (warn-only via `continue-on-error: true`)

**Acceptance criteria**:
- [ ] A GGA review workflow exists under `.github/workflows/`
- [ ] Job triggers on `pull_request` event
- [ ] Job invokes `gga run --pr-mode --diff-only`
- [ ] Job is configured with `continue-on-error: true`
- [ ] Job uses the same Go version as other CI jobs (per REQ-CI-001)

**Scenario**: GGA provider unavailable in CI
- GIVEN the `gga-review` job is running
- WHEN the AI provider times out or returns an error
- THEN the job MUST NOT fail the CI pipeline (non-blocking)

**Acceptance criteria**:
- [ ] Provider timeout does not block merge
- [ ] CI logs show the GGA failure reason for debugging
