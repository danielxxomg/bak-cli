# Tasks: fix-ci-hygiene

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~215 (range 180–250) |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR (3 independent fixes, all small) |
| Delivery strategy | ask-always (cached) |
| Chain strategy | pending (not needed) |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

## Phase 1: Fix TestRunLogin_EmptyToken (A.1 — TDD)

- [x] 1.1 **[RED]** In `cmd/login_test.go`, write `TestRunLogin_EmptyToken_WithFakeOAuth` that builds a `cmdDeps` with a fake `OAuthClient` (inline struct implementing `RequestToken() (string, error)` returning a token instantly), calls `runLoginWithDeps(nil, nil, deps)`, and asserts: completes in <2s (`testing.Deadline` or `time.After`), `config.Load().Token == fakeToken`, zero real network calls. Must fail now because `runLoginWithDeps` doesn't accept `OAuthClient` from deps yet.

- [x] 1.2 **[GREEN — refactor test]** Replace the existing `TestRunLogin_EmptyToken` body: instead of `rootCmd.SetArgs` + `rootCmd.Execute()`, construct a `cmdDeps` with `Stdout: &buf, Stderr: &buf, Stdin: strings.NewReader("1\n")`, `ConfigLoader: config.Load`, `ConfigSaver: config.Save`, `TokenValidator: cloud.ValidateToken`, and the fake `OAuthClient`. Call `runLoginWithDeps(nil, nil, deps)` directly. Assert exit 0 and token saved.

- [x] 1.3 **[GREEN — production code]** In `cmd/deps.go`, add unexported interface `type oauthTokenRequester interface { RequestToken() (string, error) }` and field `OAuthClient oauthTokenRequester` to `cmdDeps`. In `cmd/login.go`, add package-level vars `var DeviceLoginBase = cloud.DeviceLoginBase` and `var sleepFn = time.Sleep`. In `runLoginWithDeps`, when building the `DeviceClient`, use `DeviceLoginBase` instead of the const; if `deps.OAuthClient != nil`, pass it to `cloud.NewDeviceClientWithOAuth` (or construct `DeviceClient{Base: DeviceLoginBase, OAuth: deps.OAuthClient}`); wire `sleepFn` into the polling loop.

- [x] 1.4 **[RED]** In `internal/cloud/oauth_device_test.go`, write `TestDeviceLogin_ContextTimeout` that calls `DeviceLogin` with a `context.WithTimeout(ctx, 100*time.Millisecond)` and a bogus `Base` URL. Assert the returned error wraps `context.DeadlineExceeded` (or `ctx.Err()`). Must fail because current `DeviceLogin` ignores context.

- [x] 1.5 **[GREEN]** In `internal/cloud/oauth_device.go`, add `var deviceLoginTimeout = 60 * time.Second`. Wrap the `DeviceLogin` body: `ctx, cancel := context.WithTimeout(ctx, deviceLoginTimeout); defer cancel()`. Propagate `ctx` to all `http.NewRequestWithContext` calls in the polling loop. Verify `TestDeviceLogin_ContextTimeout` now passes.

- [x] 1.6 **[VERIFY]** Run `go test -race -count=1 ./cmd/ ./internal/cloud/`. Confirm: all pass, `TestRunLogin_EmptyToken_WithFakeOAuth` completes in <2s, no real network calls (verify via fake OAuthClient call count). Run `go vet ./...`.

## Phase 2: GGA OpenCode CLI Install (A.2 — CONFIG)

- [x] 2.1 **[CONFIG]** In `.github/workflows/gga.yml`, add a step before `Run GGA review`:
  ```yaml
  - name: Install OpenCode CLI
    uses: opencode-ai/action@v1
  ```
  Add `continue-on-error: true` to the `Run GGA review` step. Pin `OPENCODE_MODEL` env var to the design-specified value.

- [x] 2.2 **[VERIFY]** Validate YAML syntax: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/gga.yml'))"`. Confirm step ordering: checkout → setup Go → install GGA → install OpenCode → gga run.

## Phase 3: CI Timing Optimization (A.3 — CONFIG)

- [x] 3.1 **[CONFIG]** In `.github/workflows/ci.yml`, replace the flat `os` matrix with an `include` matrix:
  ```yaml
  strategy:
    matrix:
      include:
        - os: ubuntu-latest
          race: true
        - os: windows-latest
          race: false
        - os: macos-latest
          race: false
  ```
  Update the test step to conditionally apply `-race`: `go test ${{ matrix.race && '-race' || '' }} -coverprofile=coverage.out -covermode=atomic ./...`.

- [x] 3.2 **[CONFIG]** Create `.github/workflows/ci-full-race.yml` — weekly cron (`cron: '0 6 * * 1'`), manual dispatch, 3-OS matrix all with `-race`, same test/build/install steps as ci.yml. No `tparse` install (audit result: unused, grep confirms zero consumers).

- [x] 3.3 **[VERIFY]** Confirm `actions/setup-go@v5` with `cache: true` is already present in ci.yml (it is — no change needed). Validate both YAML files: `python3 -c "import yaml; yaml.safe_load(open(f)) for f in ['.github/workflows/ci.yml', '.github/workflows/ci-full-race.yml']"`. Confirm `tparse` has zero references in `.github/workflows/` (already clean).

## Implementation Notes (apply batch)

deviation source: tasks 1.1–1.6 text describes the **proposal** approach (fake
`cmdDeps.OAuthClient` field, `cloud.NewDeviceClientWithOAuth`, a `DeviceLogin`
function, a `sleepFn` package-var in `cmd/login.go`). Those symbols do not exist
in the codebase. The design.md **correction note** overrides the proposal; this
apply batch implements the design's corrected approach.

phase 1 (REQ-CI-009 / REQ-CI-008):
- 1.1–1.2 `TestRunLogin_EmptyToken` rewritten deterministically: redirects
  `cloud.DeviceLoginBase` to a local `httptest` pending server (expires_in=1,
  interval=1) and calls `runLoginWithDeps(&cobra.Command{}, nil, deps)` with
  `setupTestDeps` (mock config → AGENTS.md isolation). Asserts <2s, no real
  network, error mentions "timed out"/"token". RED: old body hit real github.com
  (HTTP/2 readLoop captured) and hung >8s. GREEN: completes in ~1s.
- 1.3–1.5 added `var deviceLoginTimeout = 60 * time.Second`
  (`internal/cloud/oauth_device.go`). `RequestToken` now wraps the flow in
  `context.WithTimeout(context.Background(), deviceLoginTimeout)`; the loop
  checks `ctx.Err()`; `requestDeviceCode`/`pollAccessToken`/`postOAuthForm`
  thread `ctx` via `http.NewRequestWithContext`. The 600s server-declared
  expires_in can no longer hang the client. Extracted `pollForAccessToken` to
  satisfy golangci-lint `funlen`.
- cloud RED→GREEN: `TestRequestToken_DeviceLoginTimeout` — slow /access_token
  endpoint + deviceLoginTimeout=50ms. RED: hung 3s past the 2s guard. GREEN:
  cancelled via ctx, returns `context.DeadlineExceeded` in <1s.
- 1.6 `go test ./cmd/ ./internal/cloud/` green; `golangci-lint run ./...` 0 issues.

phase 2 (REQ-CI-011):
- 2.1 added `Install OpenCode CLI` step (`curl -fsSL https://opencode.ai/install
  | bash` + `~/.opencode/bin` → `$GITHUB_PATH`) before `gga run`, plus a
  `Verify OpenCode CLI` (`opencode --version`) step. `continue-on-error` is
  already on the job, satisfying REQ-CI-011 core-requirement.
- 2.2 YAML validated; step order: setup Go → install GGA → verify GGA → install
  OpenCode → verify OpenCode → gga run.

phase 3 (REQ-CI-010):
- 3.1 `ci.yml` test job uses `include` matrix (os paired with `race`); test step
  is `go test -shuffle=on ${{ matrix.race && '-race' || '' }} ./...`. `-race`
  now only on ubuntu-latest per PR.
- 3.2 added `.github/workflows/ci-full-race.yml` — `cron: '0 6 * * 1'` (Monday
  06:00 UTC) + `workflow_dispatch`, 3-OS matrix, `go test -race -shuffle=on
  ./...`.
- 3.3 `setup-go@v5` default `cache: true` already applies (no explicit cache
  key) — no change.
- 3.4 `tparse` absent from all workflows/Taskfile — no change.
- 3.5 all three YAML files parse (python yaml.safe_load).

TDD cycle evidence (Strict TDD):
| task | RED evidence | GREEN evidence |
|------|--------------|----------------|
| 1.1/1.2 | old `TestRunLogin_EmptyToken` made real HTTP/2 calls, hung >8s | rewritten test PASS in 1.02s, no real net |
| 1.4/1.5 | `TestRequestToken_DeviceLoginTimeout` FAIL: "hung >2s" (3.00s) | same test PASS, ctx cancelled <1s, DeadlineExceeded |
