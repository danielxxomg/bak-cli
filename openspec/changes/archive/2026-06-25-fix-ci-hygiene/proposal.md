# Proposal: fix-ci-hygiene

## Intent

Three P0 CI blockers prevent the product deep audit from being actionable:

1. **TestRunLogin_EmptyToken** (`cmd/login_test.go:68-85`) calls `rootCmd.Execute()` which builds a real `cloud.DeviceClient` with a hardcoded GitHub OAuth `client_id` (`2472954d6d1c0de9be29`). The test sets `loginProvider = nil`, so `runLogin` (`cmd/login.go:107-110`) creates a live client that polls `github.com/login/device/code` for ~15 minutes across all 3 CI OS before `go test` kills it. This is the single biggest CI time sink.
2. **GGA gate is theater**: `gga.yml` runs `gga run --pr-mode --diff-only` but the runner has no `opencode` binary. GGA uses `PROVIDER="opencode:opencode-go/qwen3.7-plus"` (`.gga:5`) which shells out to the OpenCode CLI. The job fails silently under `continue-on-error: true` — the gate provides zero review value.
3. **CI timing waste**: `-race` runs on all 3 OS (~50% overhead on mac/windows where races rarely manifest differently), no Go module/build cache, and `tparse` is re-installed every run (`go install github.com/mfridman/tparse@latest` — uncached).

## Scope

### In Scope
- **A.1**: Inject fake `DeviceClient` into `TestRunLogin_EmptyToken` via the existing `runLoginWithDeps`/`actions.Deps` pattern. Add 60s `context.WithTimeout` to real `DeviceLogin` in `internal/cloud/oauth_device.go`.
- **A.2**: Install OpenCode CLI in `gga.yml` before `gga run`. Verify binary availability for `ubuntu-latest`.
- **A.3**: Move `-race` to ubuntu-latest only in per-OS matrix. Add weekly full-race cron. Add Go module + build cache to `ci.yml`. Audit redundant steps.

### Out of Scope
- Enabling new linters (goconst, wrapcheck, etc.) — **Change B**
- TUI personality features (spinners, progress bars, dashboard) — **Change C**
- Unit tests for `cmd/` `os.Exit` paths (out of scope per AGENTS.md)
- Modifying the 3-OS matrix (cross-platform is a product requirement)
- Removing lint, coverage gate, govulncheck, or security jobs

## Capabilities

### New Capabilities
- `ci-timing-optimization`: Go cache, race-flag scoping, weekly full-race cron

### Modified Capabilities
- `ci-consistency`: test job matrix changes (race flag), cache additions
- `gga-bypass`: GGA workflow now functional (installs OpenCode CLI)

## Approach

### A.1 — Fix TestRunLogin_EmptyToken

**Root cause**: `cmd/login_test.go:79` calls `rootCmd.Execute()` which triggers the real `runLogin` path (`cmd/login.go:107-130`). The test sets `loginProvider = nil`, so `runLogin` creates a fresh `&cloud.DeviceClient{HTTPClient: http.DefaultClient}` and calls `DeviceLoginBase` → `DevicePoll` which polls github.com for ~15 minutes.

**Fix (TDD)**:
1. **[RED]** Add test: `TestRunLogin_WithMockClient_CompletesFast` — calls `runLoginWithDeps` directly with `deps.DeviceLogin = fakeDeviceLoginOK`. Asserts completion in <2s.
2. **[GREEN]** Refactor `TestRunLogin_EmptyToken` to use `runLoginWithDeps` with a fake `deps.DeviceLogin` that returns `context.DeadlineExceeded` immediately. Remove `rootCmd.Execute()` from this test.
3. **[REFACTOR]** Add `context.WithTimeout(ctx, 60*time.Second)` wrapper in `cloud.DeviceLogin` (`internal/cloud/oauth_device.go:130`) so even production code can't hang forever. The timeout value is a package-level `var DeviceLoginTimeout = 60 * time.Second` for testability.

**Files**: `cmd/login_test.go`, `cmd/login.go` (minor — expose `runLoginWithDeps` if needed), `internal/cloud/oauth_device.go` (timeout wrapper).

### A.2 — Install OpenCode CLI in GGA

**Root cause**: `gga.yml:27-36` checks out code and runs `gga run` but never installs the `opencode` binary that GGA shells out to.

**Fix**:
1. Add step before `gga run`: `curl -fsSL https://opencode.ai/install | bash`. The installer auto-detects `$GITHUB_ACTIONS` and appends `~/.opencode/bin` to `$GITHUB_PATH`.
2. Add verification step: `opencode version` to confirm install.
3. The `OPENCODE_API_KEY` secret is already configured (`gga.yml:35`).

**Verification**: The install script (`https://opencode.ai/install`) downloads from `github.com/anomalyco/opencode/releases/latest/download/opencode-linux-x64.tar.gz`. Confirmed available for `ubuntu-latest` (x64 Linux). The old `opencode-ai/opencode` repo is archived; the project moved to `anomalyco/opencode`.

**Files**: `.github/workflows/gga.yml`.

### A.3 — CI Timing Optimization

**Current state** (`ci.yml:60-93`):
- `test` job: 3-OS matrix, each runs `go install tparse` (uncached), `go test -race -coverprofile=... ./...`
- No Go module cache, no build cache
- `-race` on all 3 OS

**Fix**:
1. **Race flag scoping**: Use matrix `include` to set `race: true` only for `ubuntu-latest`. Other OS run `go test -coverprofile=... ./...` without `-race`.
2. **Weekly full-race cron**: Add `schedule: - cron: '0 6 * * 1'` (Monday 6am UTC) trigger to `ci.yml`. Use a matrix override or env var to force `-race` on all OS for the weekly run.
3. **Go caches**: Add `actions/cache@v4` for `~/go/pkg/mod` (module cache) and `~/.cache/go-build` (build cache). Key: `hashFiles('**/go.sum')` + `runner.os`.
4. **tparse cache**: Either cache the `tparse` binary or replace `go install` with a direct download of the release binary.

**Files**: `.github/workflows/ci.yml`.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `cmd/login_test.go` | Modified | Refactor `TestRunLogin_EmptyToken` to use `runLoginWithDeps` with fake client |
| `cmd/login.go` | Modified | Minor: ensure `runLoginWithDeps` is testable (may already be sufficient) |
| `internal/cloud/oauth_device.go` | Modified | Add 60s `context.WithTimeout` wrapper in `DeviceLogin` |
| `.github/workflows/gga.yml` | Modified | Add OpenCode CLI install step |
| `.github/workflows/ci.yml` | Modified | Race scoping, caches, weekly cron |

**Estimated lines**: ~120-150 lines changed across 5 files.

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| OpenCode install script breaks in CI | Low | Pin to a specific version tag if available; add `continue-on-error` fallback |
| 60s timeout too aggressive for slow networks | Low | Make it configurable via `DeviceLoginTimeout` var; 60s is generous for OAuth polling |
| `-race` only on ubuntu misses OS-specific races | Low | Weekly full-race cron on all 3 OS catches these |
| Go cache keys become stale | Low | Use `go.sum` hash; cache restore with `restore-keys` fallback |
| Fake client doesn't exercise real OAuth edge cases | Med | Keep one integration-style test (or rely on e2e) for real flow |

## Rollback Plan

- **A.1**: Revert `cmd/login_test.go` and `internal/cloud/oauth_device.go`. The test returns to its current (broken but passing-via-timeout) state. No production behavior change.
- **A.2**: Remove the `curl | bash` step from `gga.yml`. GGA returns to silent-failure state (no regression — it was already failing).
- **A.3**: Revert `ci.yml` to current matrix. No production impact.

All changes are CI/test-only except the 60s timeout wrapper in `oauth_device.go`, which is a safety net that doesn't change happy-path behavior (OAuth flows complete in <30s normally).

## Dependencies

- OpenCode CLI install script (`https://opencode.ai/install`) must remain available and serve Linux x64 binaries
- `OPENCODE_API_KEY` GitHub secret must remain configured (already set)
- `actions/cache@v4` must be available (standard GitHub Actions)

## Success Criteria

- [ ] `TestRunLogin_EmptyToken` completes in <5s (currently ~15 minutes)
- [ ] `go test ./...` on CI finishes in <3 minutes total (currently ~8-10 minutes)
- [ ] GGA workflow step shows `opencode version` output confirming install
- [ ] GGA `gga run` produces actual review output (not "OpenCode CLI not found")
- [ ] Weekly cron runs `-race` on all 3 OS
- [ ] Go module cache hit rate >80% on subsequent runs
- [ ] No regression in test coverage (coverage gate still passes)
- [ ] Cross-platform build verification still works on all 3 OS
