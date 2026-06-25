# Delta for ci-consistency

## ADDED Requirements

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
