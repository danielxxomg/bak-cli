# Delta for ci-consistency

## ADDED Requirements

### REQ-CI-009: Test Isolation from External Services

Unit tests MUST NOT make real network calls. Tests that exercise external services (OAuth device flow, cloud APIs) MUST inject a fake or stub via the existing `Deps` dependency-injection pattern used by `runLoginWithDeps`.

#### Scenario: TestRunLogin_EmptyToken completes quickly with fake DeviceLogin

- GIVEN a test that calls `runLoginWithDeps` with a `DeviceLogin` dependency returning an error for empty tokens
- WHEN the test executes
- THEN the test MUST complete in under 2 seconds
- AND no real HTTP request MUST be made to any OAuth endpoint

### REQ-CI-010: Production Timeout Safety for External Calls

External calls that block on user interaction or remote services (specifically the OAuth device flow) MUST wrap their context with `context.WithTimeout` to prevent indefinite hangs.

#### Scenario: DeviceLogin times out after safety limit

- GIVEN the production `DeviceLogin` implementation is invoked with a valid but non-completing device flow
- WHEN 60 seconds elapse without user authorization
- THEN the call MUST return a context deadline exceeded error
- AND no goroutine MUST remain blocked indefinitely

### REQ-CI-011: CI Race Detector Scoping

The `-race` flag MUST run only on `ubuntu-latest` in the per-PR test matrix. A weekly scheduled job MUST run `-race` on all three operating systems (ubuntu, windows, macos) as a safety net.

#### Scenario: Per-PR ubuntu job uses race detector

- GIVEN a pull request triggers the CI workflow
- WHEN the test job runs on `ubuntu-latest`
- THEN the `go test` invocation MUST include the `-race` flag

#### Scenario: Per-PR mac/windows jobs skip race detector

- GIVEN a pull request triggers the CI workflow
- WHEN the test job runs on `windows-latest` or `macos-latest`
- THEN the `go test` invocation MUST NOT include the `-race` flag

#### Scenario: Weekly cron runs race detector on all OS

- GIVEN the weekly scheduled trigger fires (cron)
- WHEN the test matrix executes
- THEN all three OS runners MUST execute `go test` with the `-race` flag

### REQ-CI-012: Go Module and Build Cache in CI

CI MUST cache Go modules (`GOMODCACHE`) and the Go build cache (`GOCACHE`) across runs using `actions/setup-go` caching or an equivalent mechanism.

#### Scenario: Second CI run uses cached modules

- GIVEN a CI run has completed successfully and populated the cache
- WHEN a subsequent CI run executes on the same `go.sum`
- THEN the module download step MUST be skipped or significantly faster than a cold run

### REQ-CI-013: GGA Job Functional Execution

The GGA CI job MUST install the OpenCode CLI binary before running `gga run`. The GGA job MUST execute functionally (producing review output). The GGA job MUST remain advisory — findings MUST NOT fail the workflow (`continue-on-error: true`).

#### Scenario: OpenCode binary available for GGA

- GIVEN the GGA workflow job has completed its setup steps
- WHEN `which opencode` or `opencode --version` is invoked
- THEN the command MUST succeed and the binary MUST be in `$PATH`

#### Scenario: GGA produces review output

- GIVEN the OpenCode CLI is installed and `gga` is available
- WHEN `gga run --pr-mode --diff-only` executes
- THEN the step MUST produce visible output (review comments or summary)

#### Scenario: GGA findings do not fail the workflow

- GIVEN GGA reports findings or violations
- WHEN the GGA step completes
- THEN the overall job MUST still succeed (continue-on-error absorbs failures)
