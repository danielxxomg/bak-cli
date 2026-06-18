# Delta for bak-cli

## MODIFIED Requirements

### Requirement: CI pipeline fixes
The system MUST fix build and lint issues blocking CI.

#### Scenario: Lint version pinning
- GIVEN `.github/workflows/ci.yml`
- THEN `golangci-lint` version matches Go version in `go.mod`

#### Scenario: Build tag compliance
- GIVEN `parseSchtasksCSV` in Windows-specific file
- THEN `//go:build windows` tag is present and the file uses `_windows.go` suffix

#### Scenario: Rate limit resilience
- GIVEN GitHub API rate limiting
- THEN task actions include retry/backoff or caching workaround

#### Scenario: All-lint-green
- GIVEN `golangci-lint run` executes
- THEN it exits 0 with zero warnings on Ubuntu, macOS, and Windows

#### Scenario: GGA pre-commit with bypass path
- GIVEN a commit is prepared
- WHEN GGA pre-commit validation runs against AGENTS.md
- THEN it passes without `--no-verify` bypass
- OR if GGA fails due to a technical failure (ARG_MAX overflow, provider outage, scope-of-change mismatch), the commit body MUST contain `NO-VERIFY: <reason>` and a follow-up fix commit MUST be created in the same PR

(Previously: GGA pre-commit scenario required passing without any `--no-verify` bypass, with no escape hatch for technical failures)

#### Scenario: Docker test pass
- GIVEN `task test:linux` (Docker)
- WHEN it executes
- THEN all tests pass inside the Linux container

#### Scenario: 3-OS CI matrix
- GIVEN GitHub Actions CI with Ubuntu, macOS, Windows
- WHEN `go test ./...` runs
- THEN all three jobs report success
