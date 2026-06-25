# ci-consistency Specification

## Requirements

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

---

### REQ-CI-005: Per-Package Coverage Gate

The CI pipeline MUST enforce a minimum 80% statement coverage threshold for each package under `internal/`. Packages with no test files (e.g., test helpers) and the `cmd/` package are exempted per AGENTS.md.

#### Scenario: internal/ package above 80% passes

- GIVEN an `internal/` package with ≥80.0% statement coverage
- WHEN `task cover:pkg` executes in CI
- THEN the job passes for that package

#### Scenario: internal/ package below 80% fails

- GIVEN an `internal/` package with <80.0% statement coverage
- WHEN `task cover:pkg` executes in CI
- THEN the job fails with a non-zero exit code and lists the failing package

#### Scenario: cmd/ package is exempted

- GIVEN the `cmd/` package has <80% coverage (by-design, no unit tests for os.Exit)
- WHEN `task cover:pkg` executes
- THEN the job passes (cmd/ is excluded from the per-package gate)

#### Scenario: package with no test files is exempted

- GIVEN an `internal/` package with no `_test.go` files
- WHEN `task cover:pkg` executes
- THEN the job passes (package is excluded from the gate)

---

### REQ-CI-006: golangci-lint Version Pinned

CI MUST use `golangci/golangci-lint-action@v8` with an explicit pinned `version` (e.g., `v2.12.2`). CI MUST NOT use `go install ...@latest` for golangci-lint.

#### Scenario: CI runs with pinned version

- GIVEN `.github/workflows/ci.yml` uses `golangci/golangci-lint-action@v8`
- WHEN the lint job executes
- THEN the action MUST specify `version: v2.12.2` (exact patch)

#### Scenario: cache is used

- GIVEN the lint job uses `golangci-lint-action@v8`
- WHEN the job runs on a subsequent CI invocation
- THEN the action MUST use its built-in cache (`~/.cache/golangci-lint`)

#### Scenario: no go install @latest for golangci-lint

- GIVEN the CI workflow file
- WHEN inspecting the lint job steps
- THEN there MUST NOT be a `go install .../golangci-lint@latest` step

---

### REQ-CI-007: govulncheck Blocking (Reachable Vulnerabilities)

The CI security job MUST run `govulncheck ./...` as a blocking check. Any **reachable** vulnerability (as determined by govulncheck's built-in reachability analysis) MUST fail the job. govulncheck v1.4.0 has no severity filter flag; reachability IS the filter — unreachable vulnerabilities in transitive dependencies are not reported. `gosec` MUST remain advisory (`|| true`).

#### Scenario: reachable vulnerability blocks CI

- GIVEN a dependency has a vulnerability that is reachable from bak-cli's code
- WHEN `govulncheck ./...` runs in CI
- THEN the security job fails with a non-zero exit code

#### Scenario: unreachable vulnerability does not block

- GIVEN a dependency has a vulnerability that is NOT reachable from bak-cli's code
- WHEN `govulncheck ./...` runs in CI
- THEN govulncheck does not report it (built-in reachability analysis filters it)

#### Scenario: no vulnerabilities passes

- GIVEN no known reachable vulnerabilities in dependencies
- WHEN `govulncheck ./...` runs in CI
- THEN the security job passes cleanly

#### Scenario: gosec remains advisory

- GIVEN gosec reports findings (which may include false positives G301/G306/G304)
- WHEN the security job runs
- THEN gosec output is logged but does NOT fail the job (`|| true`)

---

### REQ-CI-008: GGA Installed without brew

The GGA CI job MUST install GGA without Homebrew. GGA is a Bash application (not Go — `go install` is not possible). CI MUST use `git clone --branch <tag>` + `./install.sh`, or an equivalent binary/script release download. CI MUST NOT use `brew install gga` on Linux runners.

#### Scenario: CI installs GGA without brew

- GIVEN the `gga-review` job in `.github/workflows/gga.yml`
- WHEN the GGA installation step executes
- THEN it MUST use `git clone --branch v2.8.1 https://github.com/Gentleman-Programming/gentleman-guardian-angel && ./install.sh` or equivalent
- AND MUST NOT invoke `brew update` or `brew install`

#### Scenario: install succeeds on ubuntu-latest

- GIVEN the runner is `ubuntu-latest`
- WHEN the GGA install step completes
- THEN the `gga` binary MUST be available in `$PATH` and `gga --version` succeeds
