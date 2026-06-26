# Design: fix-ci-hygiene

## Technical Approach

Three P0 CI fixes, each independent and independently shippable. A.1 removes a
network-dependent flaky test by (a) giving the device-login poller a real,
overridable timeout and (b) converting the cmd-level test to exercise the
deterministic manual/timeout error path instead of `rootCmd.Execute()`. A.2
installs the OpenCode CLI into the GGA workflow so `gga run` can shell out to
`opencode`. A.3 drops `-race` from the per-PR Windows/macOS matrix legs (Linux
keeps it) and moves full `-race` coverage to a weekly schedule, adds the
`setup-go` build cache, and removes the unused `tparse` install.

> Note: the proposal's A.1 sketch referenced `Deps.DeviceLogin` and
> `runLoginWithDeps(cmd, reader, deps)`. Those names do NOT exist in this
> codebase. Actual shapes: `DeviceClient.RequestToken() (string, error)`
> (no ctx), `LoginAction.OAuthClient` (interface `oauthTokenRequester`),
> `runLoginWithDeps(cmd, args, deps)`. This design follows the existing
> overridable-package-var seam pattern (`DeviceLoginBase`, `sleepFn`) rather
> than fabricating the proposed-but-absent `DeviceLogin` field, per AGENTS.md
> DRY / "follow existing pattern" rules.

## Architecture Decisions

| # | Decision | Option A (chosen) | Option B (rejected) | Rationale |
|---|----------|-------------------|---------------------|-----------|
| 1 | 60s login timeout | Package var `deviceLoginTimeout` + internal `context.WithTimeout` in `RequestToken` | Change `RequestToken(ctx)` signature | B breaks the `oauthTokenRequester` interface + `cmd/login.go` wiring; A matches the existing `DeviceLoginBase`/`sleepFn` override seam and needs no interface change. |
| 2 | Cmd test form | Call `runLoginWithDeps(rootCmd, nil, depsFromCmd(rootCmd))` with `DeviceLoginBase`→`httptest` server + tiny `deviceLoginTimeout` | Keep `rootCmd.Execute()` | `rootCmd.Execute` hits the network (the bug); `runLoginWithDeps` is a pure seam; deterministic, AGENTS "don't unit-test cmd entry via Execute" spirit. |
| 3 | GGA install | `curl -fsSL https://opencode.ai/install \| bash` before `gga run` | Vendor a binary / `go install` | Script already detects `GITHUB_ACTIONS=true` and appends `$HOME/.opencode/bin` to `$GITHUB_PATH`; run via `bash -s -- --no-modify-path --version <pin>` to avoid mutating shell rc files on the runner. |
| 4 | Per-PR race matrix | `[{os: ubuntu-latest, race: true}, {windows-latest, false}, {macos-latest, false}]` | Keep `-race` on all three | Windows `-race` dominates runner minutes; Linux race catches the data races that matter on PR-sized diffs. Full sweep moves to weekly. |
| 5 | Full-race safety net | New `ci-full-race.yml` on `schedule: cron: "0 6 * * 1"` + `workflow_dispatch` | Trust Linux-only race | Defends the cross-OS race surface without burning per-PR minutes; manual trigger for hotfix verification. |
| 6 | Go cache | `actions/setup-go@v5` with `cache: true` | Manual `~/.cache/go-build` cache | setup-go v5 caches both modules and build cache keyed on `go.sum`; zero config. |
| 7 | tparse | Remove install step | Keep / cache it | Grep shows no `tparse` consumer in the repo; dead weight on the runner. |

## Data Flow

A.1 (production login):

    RequestToken()
      └─ context.WithTimeout(bg, deviceLoginTimeout) ──► ctx
      └─ poll loop: sleep(interval) ──► POST token ──► ctx.Err()? ──► return "login: timed out: %w"

A.1 (test):

    TestRunLogin_EmptyToken
      ├─ DeviceLoginBase = httptest server (always "authorization_pending", interval=1, expires_in=600)
      ├─ deviceLoginTimeout = 50ms        └─ runLoginWithDeps(rootCmd, nil, depsFromCmd(rootCmd))
      └─ assert err contains "token" / "timed out"

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/cloud/oauth_device.go` | Modify | Add `var deviceLoginTimeout = 60 * time.Second`. In `RequestToken`, wrap poll with `ctx, cancel := context.WithTimeout(context.Background(), deviceLoginTimeout)`; check `ctx.Err()` each iteration → `fmt.Errorf("login: timed out waiting for authorization: %w", ctx.Err())`. |
| `internal/cloud/oauth_device_test.go` | Modify | [RED then GREEN] Add table test: stub `DeviceLoginBase`→pending httptest server, set `deviceLoginTimeout=50ms`, assert `RequestToken` returns timeout error and does not exceed e.g. 1s wall clock. |
| `cmd/login_test.go` | Modify | Rewrite `TestRunLogin_EmptyToken` to call `runLoginWithDeps(rootCmd, nil, depsFromCmd(rootCmd))` with `DeviceLoginBase`→pending httptest + `deviceLoginTimeout=50ms`; defer reset of both vars. Drop the `rootCmd.Execute()` network path. |
| `.github/workflows/gga.yml` | Modify | Add step `Install OpenCode CLI` before `gga run`: `run: curl -fsSL https://opencode.ai/install \| bash -s -- --no-modify-path --version <pin>` (strip the literal backslash-pipe: real pipe). |
| `.github/workflows/ci.yml` | Modify | Matrix → `include: [{os: ubuntu-latest, race: true}, {os: windows-latest, race: false}, {os: macos-latest, race: false}]`; test step `go test {{if race==true}}-race{{end}} -shuffle=on ./...`; pin `actions/setup-go@v5` with `cache: true`; remove `tparse` install. |
| `.github/workflows/ci-full-race.yml` | Create | Weekly full `-race` on all 3 OS + manual `workflow_dispatch`; reuse the test step with `-race` unconditional. |

## Interfaces / Contracts

```go
// internal/cloud/oauth_device.go — overridable, no signature change
var deviceLoginTimeout = 60 * time.Second // tests may lower to force cancellation

// internal/actions/login.go — UNCHANGED
type oauthTokenRequester interface { RequestToken() (string, error) }
```

## Testing Strategy

| Layer | What | How |
|-------|------|-----|
| Unit | `RequestToken` honors `deviceLoginTimeout` | httptest pending server + 50ms timeout; assert error + wall-clock bound |
| Unit | `TestRunLogin_EmptyToken` deterministic error path | `runLoginWithDeps` direct call; assert error contains "token"/"timed out" |
| Unit | Timeout var is restorable | `t.Cleanup`/defer resets `deviceLoginTimeout`, `DeviceLoginBase` |
| CI | GGA workflow installs `opencode` and `gga run` completes | GHA workflow run on a sample PR |
| CI | Per-PR matrix: `-race` only on Linux; full suite green | GHA matrix run |
| CI | Weekly full-race passes on all 3 OS | Scheduled run + `workflow_dispatch` smoke |

## Migration / Rollout

No data migration. A.3 matrix change is backward compatible. Pin OpenCode CLI
version in `gga.yml` to avoid installer drift; bump deliberately. Add
`OPENCODE_API_KEY` secret confirmed present (current GGA already references it).

## Open Questions

- [ ] Pin exact OpenCode CLI version for reproducible GGA runs (default rolling).
- [ ] Confirm `depsFromCmd(rootCmd)` is safe to use directly in unit test (it reads
      cobra-level flags; may need `rootCmd.ResetFlags`/`SetArgs([])` setup).