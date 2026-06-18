# Proposal: GGA PR Merge — Calibrate Rule, Fix Violations, Add CI, Merge Chain

## Intent

`quality-ux-overhaul` shipped 4 chained PRs but the chain is blocked. PR #26 used `--no-verify` (commit `23dfaf9`) — GGA's prompt exceeded `ARG_MAX` on 11 files; 2 violations deferred as "out of scope": 4× `fmt.Printf` in `internal/actions/pull.go` and 14 non-table-driven tests in `internal/backup/engine_test.go`. Fix violations, calibrate rule #41 to make `--no-verify` an honest escape hatch, add GGA to CI, then merge #27→#26→#25→#24→main.

## Scope

### In Scope
- **WS2** Fix PR #26: 4× `fmt.Printf` → `fmt.Fprintf(os.Stderr, ...)`; `engine_test.go` → table-driven
- **WS1** Calibrate `AGENTS.md` rule #41: bypass requires `NO-VERIFY:` line in commit body (allowed: ARG_MAX, provider outage, scope mismatch)
- **WS3** `gga-review` CI job — `gga run --pr-mode --diff-only`, non-blocking
- **WS4** Merge chain #27 → #26 → #25 → #24 → main; archive `quality-ux-overhaul`

### Out of Scope
- `cmd/wizard_test.go` → `internal/tui/screens/wizard_test.go` move (coverage task)
- GGA upstream patches, `STRICT_MODE` relaxation, `AGENTS.md` split

## Capabilities

### New Capabilities
- `gga-bypass`: `NO-VERIFY:` line in commit body documents reason; follow-up fix required in same change

### Modified Capabilities
- `bak-cli`: replace `Scenario: GGA pre-commit` (line 240) — (a) normal pass; (b) bypass with `NO-VERIFY:` + fix
- `ci-consistency`: add `REQ-CI-004` — `gga-review` job on `pull_request`, non-blocking warn

## Approach

User order: **WS2 → WS1 → WS3 → WS4**. WS4 depends on WS2 (PR #26 clean). Apply fix as new commit on PR #26 branch (no squash) — chain preserved.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/actions/pull.go` | Modified | L89/130/138/139: `fmt.Printf` → `fmt.Fprintf(os.Stderr, ...)` |
| `internal/backup/engine_test.go` | Modified | 14 funcs → table-driven |
| `AGENTS.md` | Modified | Rule #41 adds bypass clause |
| `.github/workflows/ci.yml` | Modified | New `gga-review` job |

## Risks

| Risk | Mitigation |
|------|------------|
| Test conversion breaks coverage | `setConfigHome` pattern proven in `wiring-fixes` |
| GGA in CI quota cost | `--diff-only` + non-blocking |
| Merge conflict on chain | Top-down merge order |
| `NO-VERIFY:` becomes rubber stamp | Follow-up fix required in same change |

## Rollback Plan

- **WS1**: revert `AGENTS.md` (single file)
- **WS2**: revert fix commit on PR #26 branch (pre-merge risk zero)
- **WS3**: remove `gga-review` job (no state)
- **WS4**: `git revert -m 1 <merge-sha>` per chain PR

## Dependencies

- `OPENCODE_API_KEY` repo secret configured before WS3
- Branch protection on `main` permits chain order

## Success Criteria

- [ ] `pull.go` has 0 `fmt.Printf`; `engine_test.go` is table-driven
- [ ] `AGENTS.md` rule #41 documents `NO-VERIFY:` escape hatch
- [ ] `ci.yml` has non-blocking `gga-review` job
- [ ] PRs #27→#24 merged; `gga run` passes on HEAD
- [ ] `quality-ux-overhaul` archived
