# Design: Test Hardening — Fix Misleading Testscripts and Close Coverage Gaps

## Technical Approach

Fix four test gaps by rewriting misleading testscripts in-place, adding an action-level cloud sync round-trip test, a binary smoke test, and cmd-level schedule happy-path tests. All changes reuse existing mock infrastructure (`MockProvider`, `MockProviderFactory`, `mockScheduler`, `buildTarGz`, `sandboxEnv`) — no new test frameworks or dependencies.

## Architecture Decisions

### Decision: Testscript Rewrite Strategy

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Rewrite in-place | Filenames match behavior; no orphan files | **Chosen** — spec scenarios align with filenames |
| New files alongside | Old misleading files remain confusing | Rejected — violates "tests MUST accurately describe what they test" |

### Decision: Cloud Sync Test Location

| Option | Tradeoff | Decision |
|--------|----------|----------|
| New `cloud_sync_test.go` in `internal/actions/` | Cross-cutting test spans push+pull; clear name | **Chosen** — doesn't bloat either file |
| Extend `push_test.go` | push_test.go already 781 lines | Rejected — too large, pull is half the test |

Reuse `MockProvider`/`MockProviderFactory` from `mock_impl_test.go` and `buildTarGz`/`verifyExtractedFiles` from `pull_test.go`.

### Decision: TUI Smoke Test Location

| Option | Tradeoff | Decision |
|--------|----------|----------|
| New `tests/e2e/smoke_test.go` | Extensible for future binary-level checks | **Chosen** — keeps roundtrip_test.go focused |
| Extend `roundtrip_test.go` | Reuses `sandboxEnv` directly | Rejected — conceptually different (launch vs backup flow) |

Reuse `sandboxEnv` by exporting it or duplicating the small helper.

### Decision: Schedule Happy Path — cmd-level with Mock Injection

| Option | Tradeoff | Decision |
|--------|----------|----------|
| cmd-level test with `NewScheduler` injection | No real crontab; works in CI | **Chosen** — spec requires mock injection |
| Fix testscript | Would invoke real crontab/schtasks in CI | Rejected — fails without OS scheduler access |

Requires extending `cmdDeps` with a `NewScheduler` field and passing it through `runSchedule*WithDeps` to `ScheduleAction.NewScheduler`.

## Data Flow

### Cloud Sync Integration Test

```
PushAction.Run([backupID])
  ├── resolveBackupID → "20260101-120000"
  ├── cloud.TarGzDirectory(backupPath) → base64 string
  ├── base64 decode → rawArchive []byte
  └── MockProvider.Push(rawArchive, meta) → captures bytes, returns "mock-id"

MockProvider (shared state)
  ├── stored archive = base64 encode(rawArchive)   ← simulates gist storage
  └── PullFn("mock-id") → returns stored archive   ← simulates gist retrieval

PullAction.Run(["mock-id"])
  ├── MockProvider.Pull("mock-id") → base64 archive []byte
  ├── cloud.UntarGz(string(archiveData), backupPath)
  └── extracts files to ~/.bak/backups/<timestamp>/

Verify: extracted files content == original fixture content
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `tests/e2e/testdata/backup_restore_roundtrip.txtar` | Modify | Add real `bak restore <id> --dry-run` and `bak restore <id> --force` with file assertions |
| `tests/e2e/testdata/diff_two_backups.txtar` | Modify | Create two backups with different fixtures, run `bak diff <id1> <id2>`, assert file-level diffs |
| `tests/e2e/testdata/backup_verify_roundtrip.txtar` | Modify | Add `bak verify <id>` after backup, assert checksum output; add tampered-file scenario |
| `tests/e2e/testdata/tui_launch.txtar` | Create | Run `bak --help` and `bak` (no args), assert exit 0 and help output |
| `tests/e2e/smoke_test.go` | Create | `TestBinaryHelp`, `TestBinaryNoArgs`, `TestBinaryUnknownCommand` — build binary, run with `exec.Command` |
| `internal/actions/cloud_sync_test.go` | Create | `TestCloudSync_PushPullRoundTrip`, `TestCloudSync_InvalidToken`, `TestCloudSync_NotFound` — full push→pull with `MockProvider` |
| `cmd/deps.go` | Modify | Add `NewScheduler func() schedule.Scheduler` field to `cmdDeps` |
| `cmd/schedule.go` | Modify | Pass `deps.NewScheduler` to `ScheduleAction.NewScheduler` in all three `runSchedule*WithDeps` functions |
| `cmd/schedule_test.go` | Modify | Add `TestScheduleCreate_HappyPath`, `TestScheduleList_HappyPath`, `TestScheduleRemove_HappyPath` with mock scheduler via `cmdDeps` |

## Interfaces / Contracts

### Extended cmdDeps

```go
type cmdDeps struct {
    ConfigLoader func() (*config.Config, error)
    Stdout       io.Writer
    Stderr       io.Writer
    Stdin        io.Reader
    NewScheduler func() schedule.Scheduler // NEW — nil falls through to production default
}
```

`cmd/schedule.go` passes `deps.NewScheduler` to `ScheduleAction.NewScheduler`. When nil, `ScheduleAction.sched()` already falls through to `schedule.NewScheduler()` — no behavior change for production.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| E2E (testscript) | Restore round-trip, diff two backups, verify integrity, TUI launch | Rewrite 3 txtar in-place + 1 new txtar; assert real command output |
| E2E (Go) | Binary launches with `--help`, no args, unknown command | `exec.Command` in `smoke_test.go`; reuse `sandboxEnv` pattern |
| Integration | Push→Pull round-trip at action level | `cloud_sync_test.go` with `MockProvider` capturing archive bytes; verify extracted content matches originals |
| Unit (cmd) | Schedule create/list/remove happy path | Table-driven tests in `schedule_test.go` with mock `Scheduler` injected via extended `cmdDeps` |

### Edge Cases

- **Testscripts**: invalid backup ID for restore/diff/verify (already partially covered — extend assertions)
- **Cloud sync**: push with empty backup dir (covered by existing `TestPushAction_NoBackupsFound`); pull with corrupted archive (covered by existing `TestPullAction_MockProvider_PullError`)
- **TUI smoke**: non-TTY environment (CI) — `isTTY()` returns false, binary falls through to help
- **Schedule**: mock scheduler error propagation (already covered in `schedule_test.go`); cmd→action wiring with valid profile

## Migration / Rollout

No migration required. All changes are test-only except `cmd/deps.go` and `cmd/schedule.go` (adding an optional field — backward compatible, zero-value is nil which preserves existing behavior).

## Risk Assessment

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| txtar `exec bak restore --force` modifies sandbox unexpectedly | Low | Testscript `$HOME` is isolated; restore writes to sandbox only |
| `bak diff` output format unstable | Low | Assert on structural content (file names present), not exact format |
| `bak verify` output wording changes | Low | Assert on exit code + presence of checksum-like patterns |
| `cloud_sync_test.go` archive round-trip fails on Windows path separators | Medium | Use `buildTarGz` with forward-slash paths (already cross-platform) |
| `cmdDeps.NewScheduler` nil breaks existing tests | None | `ScheduleAction.sched()` already handles nil → production default |

## Open Questions

- [ ] Should `sandboxEnv` in `tests/e2e/roundtrip_test.go` be exported (rename to `SandboxEnv`) for reuse in `smoke_test.go`, or is a small duplicate acceptable?
