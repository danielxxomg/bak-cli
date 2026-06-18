# Archive Report: critical-fixes

- **Change**: `critical-fixes`
- **Branch**: `fix/critical-fixes`
- **Archived**: 2026-06-18
- **Verdict**: PASS (verify-report.md)
- **Tasks**: 32/32 complete

## Summary

Fixed 9 issues from manual testing: backup bloat (scanRootFiles ignoring ScanOptions, missing codex whitelist, incomplete DefaultExcludes), TUI `q` dispatching backup, and 6 missing CLI affordances (--version, config show/set/get, restore no-arg picker, profile create wizard, footer ? hint, cleanup retention). Also surfaced OAuth `error_description`.

## Specs Synced

| Domain | Action | Details |
|--------|--------|---------|
| backup-engine | Updated | 3 added requirements (scanRootFiles ScanOptions, DefaultExcludes runtime artifacts, Codex whitelist) |
| bak-cli | Updated | 4 added requirements (--version, restore picker, profile wizard, footer hint) + 1 modified (TUI dispatch gate) |
| backup-retention | Created | 3 requirements (cleanup --keep N, confirmation gate, --dry-run) |
| config-cli | Created | 3 requirements (config show, config set, token redaction) |
| tui-dispatch | Created | 2 requirements (RouteSelection gate, quit no-action) |

## Artifacts

- proposal.md — 9 issues, scope, success criteria
- explore.md — root cause analysis with file:line evidence
- design.md — 5 detailed fix decisions + data flows
- tasks.md — 32 tasks across 10 phases, all complete
- specs/ (5 domains) — 16 requirements, 41 scenarios
- verify-report.md — PASS, all quality gates green

## Verification Summary

- `go test -race -count=1 ./...` — exit 0, 29 packages pass
- `go vet ./...` — exit 0
- `golangci-lint run` — 0 issues
- Runtime spot-checks — all 9 fixes verified

## Warnings (non-blocking)

- W1: `internal/config` coverage 70.6% (<80%) — pre-existing
- W4: `internal/adapters/opencode` coverage 70.6% (<80%) — mirror lacks package-local test
- W2: Test name deviations from tasks.md (behavior tested, names differ)
- W3: `*.sqlite` vs `*.sqlite*` pattern literal (behaviorally equivalent)

## SDD Cycle

Planned → Spec'd → Designed → Tasked → Applied → Verified → **Archived**
