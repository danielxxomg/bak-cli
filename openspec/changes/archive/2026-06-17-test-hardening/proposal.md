# Proposal: Test Hardening — Fix Misleading Testscripts and Close Coverage Gaps

## Intent

The test suite has four categories of gaps that create a false sense of security. Testscripts are named after scenarios they never execute. Cloud sync has zero integration-level testing. The TUI has 1435 lines of model tests but has never been launched as a binary. Schedule happy paths are untested. This change fixes the misleading tests and adds targeted tests for the real gaps.

## Scope

### In Scope
1. **Fix misleading testscripts** — rewrite 3 txtar files to actually exercise what their names promise
2. **Cloud sync integration test** — add a push→pull round-trip test using `httptest.NewServer` as a realistic API simulator
3. **TUI launch smoke test** — add a binary-level test that proves `bak` launches without panic
4. **Schedule happy path** — add a testscript that exercises `schedule create` → `schedule list` → `schedule remove` through the action layer with a mock scheduler

### Out of Scope
- Rewriting the entire E2E suite
- Adding real GitHub API integration tests (requires tokens)
- Testing TUI interaction flows (key presses, screen transitions) — those are covered by model tests
- Windows schtasks happy path (requires admin privileges)
- Adding coverage for untested edge cases in adapters

## Problem Analysis

### Gap 1: Misleading Testscripts (HIGH RISK)

Three testscripts have names that overstate what they test:

| Script | Name Claims | Actually Does |
|--------|-------------|---------------|
| `backup_restore_roundtrip.txtar` | backup + restore roundtrip | backup + `restore --help` + error path |
| `diff_two_backups.txtar` | diff between two backups | two backups + `diff --help` + error path |
| `backup_verify_roundtrip.txtar` | backup + verify roundtrip | two backups + `bak list` (never runs verify) |

**Risk**: A regression in `restore`, `diff`, or `verify` would go undetected because these scripts never exercise those commands.

### Gap 2: Cloud Sync — Zero Integration (MEDIUM RISK)

All cloud tests (`gist_test.go`, `github_gist_test.go`, `github_repo_test.go`, `gitea_test.go`) use `httptest.NewServer` at the individual provider level. This is good unit testing. However, no test exercises the **full action flow**: `bak backup` → `cloud.TarGzDirectory()` → `Provider.Push()` → `Provider.Pull()` → `cloud.UntarGz()` → verify content.

The closest is `TestGitHubGistProvider_PushIntegration` and `TestGitHubRepoProvider_PushIntegration`, which test push→pull at the provider level. What's missing is the **action-level** integration: does `actions.PushAction` correctly orchestrate backup→pack→push?

### Gap 3: TUI Launch — Never Tested as Binary (LOW RISK)

1435 lines of model tests cover `Update()` and `View()` as pure functions. `cmd/tty_test.go` tests the injection point. But no test proves that `go build . && ./bak` (with a TTY) launches without panic.

**Risk**: A nil pointer in `defaultRunTUI` or a missing dependency wiring would only be discovered by a user running the binary.

### Gap 4: Schedule Happy Path (MEDIUM RISK)

`internal/actions/schedule_test.go` has excellent coverage of error paths (invalid interval, profile not found, config load error, scheduler error). The happy path (`TestScheduleCreate_Success`) uses a `mockScheduler` and verifies the call was made.

However, the **testscript** `schedule_create_list.txtar` only tests error paths (missing flag, invalid interval, nonexistent profile). It never exercises a successful create→list→remove cycle.

**Risk**: A regression in the cobra→action wiring for schedule commands would go undetected at the E2E level.

## Approach

### Fix 1: Rewrite Misleading Testscripts

**`backup_restore_roundtrip.txtar`**: Add a real restore step.
- Create backup with `quick` preset
- Parse backup ID from output
- Run `bak restore <id> --dry-run` (safe, no file changes)
- Assert dry-run output shows expected files
- Then run `bak restore <id> --force` for real
- Assert restored files exist

**`diff_two_backups.txtar`**: Add a real diff step.
- Create two backups with different content (modify fixture between backups)
- Parse both backup IDs
- Run `bak diff <id1> <id2>`
- Assert output shows file-level differences

**`backup_verify_roundtrip.txtar`**: Add a real verify step.
- Create backup with `quick` preset
- Parse backup ID
- Run `bak verify <id>`
- Assert verify output shows checksums OK

**Effort**: ~60 lines of txtar changes (20 per script)

### Fix 2: Cloud Sync Integration Test

Add `internal/actions/push_test.go` (or extend existing) with a test that:
1. Creates fixture files in a temp home
2. Builds a `PushAction` with a mock HTTP server (via `setupMockGistAPI`)
3. Runs `PushAction.Push()` which calls backup→pack→push
4. Verifies the mock server received the expected payload
5. Runs `PullAction.Pull()` which calls pull→unpack→restore
6. Verifies restored files match originals

This uses the existing `setupMockGistAPI` pattern from `gist_test.go` but at the action level.

**Effort**: ~80 lines

### Fix 3: TUI Launch Smoke Test

Add `tests/e2e/testdata/tui_launch.txtar`:
- Run `bak` with no args in a non-TTY environment
- Assert it falls through to help output (since testscripts are non-TTY)
- This proves the binary launches and the root command wiring works

For a real TTY smoke test, add `TestTUIBinaryLaunch` in `roundtrip_test.go`:
- Build the binary
- Run it with `--help` (no TTY needed)
- Assert exit code 0 and help output contains expected text

**Effort**: ~30 lines

### Fix 4: Schedule Happy Path Testscript

Rewrite `schedule_create_list.txtar` to include a happy path:
- Create a profile first (`bak profile create work --provider github-gist --preset quick`)
- Run `bak schedule create work --every daily`
- Assert success output
- Run `bak schedule list`
- Assert work profile appears
- Run `bak schedule remove work`
- Assert removal output

**Note**: This requires the schedule command to work without real OS crontab access. The `CronScheduler` uses `execCommand` which is a package-level variable — but in E2E tests, we run the real binary, so it will try to use real `crontab`. This may fail in CI without crontab.

**Alternative**: Test the happy path at the `cmd/` level (like `schedule_test.go` already does for error paths) by injecting a mock scheduler. The existing `TestScheduleCreate_Execute` only tests error paths. Add `TestScheduleCreate_HappyPath` with a mock scheduler injection.

**Effort**: ~40 lines

## Prioritization

| Priority | Gap | Risk of Silent Breakage | Effort | Value |
|----------|-----|------------------------|--------|-------|
| 1 | Fix misleading testscripts | HIGH — restore/diff/verify regressions undetected | Low (60 LOC) | HIGH — proves core flows |
| 2 | Schedule happy path | MEDIUM — cobra→action wiring untested | Low (40 LOC) | MEDIUM — proves scheduling works |
| 3 | Cloud sync integration | MEDIUM — action-level orchestration untested | Medium (80 LOC) | HIGH — proves end-to-end cloud flow |
| 4 | TUI launch smoke | LOW — model tests cover logic thoroughly | Low (30 LOC) | LOW — catches only wiring panics |

## Risk Assessment

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Restored files differ in CI due to path resolution | Low | Use `--dry-run` first, then `--force` with known fixtures |
| `bak diff` output format changes | Low | Assert on structural content (file names), not exact format |
| `bak verify` requires checksum computation that may vary | Low | Verify uses manifest checksums, not recomputation |
| Cloud integration test becomes flaky due to HTTP timing | Low | Use `httptest.NewServer` (synchronous, in-process) |
| Schedule happy path fails in CI without crontab | Medium | Use cmd-level test with mock scheduler, not testscript |
| TUI smoke test fails on Windows due to TTY detection | Low | Test `--help` path, not TUI launch path |

## Non-Goals

- Testing real GitHub/GitLab/Codeberg APIs (requires tokens, rate limits)
- Testing TUI screen navigation (already covered by 1435 lines of model tests)
- Testing Windows schtasks happy path (requires admin)
- Achieving 100% coverage — targeting the highest-risk gaps only
- Adding new test infrastructure (reuse existing `setupMockGistAPI`, `testscript`, `roundtrip_test.go` patterns)

## Success Criteria

- [ ] `backup_restore_roundtrip.txtar` runs `bak restore` (not just `--help`)
- [ ] `diff_two_backups.txtar` runs `bak diff` with real backup IDs
- [ ] `backup_verify_roundtrip.txtar` runs `bak verify` and asserts checksum output
- [ ] Cloud sync integration test exercises push→pull at the action level
- [ ] Schedule happy path test creates, lists, and removes a schedule
- [ ] TUI binary launches without panic (smoke test)
- [ ] All existing tests continue to pass
- [ ] No test takes more than 10 seconds

## Dependencies

- No new dependencies required
- Uses existing `httptest.NewServer`, `testscript`, `exec.Command` patterns
- Builds on existing `setupMockGistAPI` and `roundtrip_test.go` infrastructure
