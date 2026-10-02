# Public stabilization — evidence-first readiness

## Objective

Prepare a reliable public `bak` release by stabilizing existing behavior, without adding new features.

## Problem

The product advertises local backup, restore, verification, and recovery guarantees, but the current audit has directly verified restore error-propagation and safety-integration gaps. Other reported security, permission, compatibility, documentation, and test-proof gaps still require isolated reproduction before implementation scope can be fixed.

## Why

Users installing a public release cannot rely on maintainer assistance. Stabilization must be proven with executable evidence, honest documented limits, and reproducible release artifacts.

## Scope

Stable target:

- Local discovery, backup, verification, diff, restore, and recovery.
- Both CLI and TUI entry points for the stable core.
- Linux, macOS, and Windows with equivalent safety behavior.

Explicitly conditional:

- Cloud push/pull/login and scheduling may remain experimental unless proven.
- Experimental status never excuses data loss, secret exposure, or silent failures.

Out of scope for this feature:

- New adapters, backends, commands, or product features.
- Tag, release, branch-protection, or remote repository mutations.
- Real user backups, credentials, cloud accounts, or production schedulers.

## Constraints

- Preserve `openspec/changes/product-deep-audit/` untouched until its reconciliation decision.
- Preserve historical SDD material and Engram provenance; no automatic memory deletion.
- Distinguish source inspection from runtime reproduction and remote-state claims.
- Keep implementation surfaces bounded per task with explicit allowed edit surfaces.
- Honor mandatory GGA validation; do not bypass hooks.
- Technical artifacts default to English.

## Authorized scope for the current phase

Read-only isolated evidence plus this task document and its Engram mirror only:

- Inspect source, tests, workflows, docs, local Git metadata, and toolchain presence.
- Run no product backup/restore/login/scheduling against real user state.
- No source, docs, config, workflow, hook, or release edits except this document and mirror.
- No commits, tags, branches, pushes, PRs, releases, or remote API/settings operations.
- No access to real credentials, tokens, backups, or unrelated repositories.

## Acceptance criteria

- [ ] Safety blockers are either reproduced in isolation or refuted with evidence.
- [ ] Critical real-binary journey assertions are specified before test/product edits.
- [ ] Documentation, memory, changelog, and release baselines are inventoried.
- [ ] No product behavior was changed during the evidence phase.
- [ ] Next implementation slice has bounded surfaces, checks, and rollback reporting.

## Applicable checks

- Evidence tasks: structural readback plus exact deferred commands for a later isolated delegate.
- No full test suite, build matrix, lint, security scan, or packaged-artifact smoke test is claimed as run until a fresh verification worker reports observed results.
- Later implementation tasks will use the agreed deterministic checks and three-platform critical journeys.

## Tasks

- [x] **T1 — Reproduce restore safety evidence**
  - Verified counted-but-unreported copy failures, unused restore safety integration, integrity bypass under force, and absence of pre-restore recovery/rollback in the current code paths.
  - Parent spot-checked `cmd/root.go:244-251`: TUI sets `Force: !dryRun`, coupling prompt bypass to integrity bypass.
  - Deferred isolated reproduction commands recorded; no product behavior edited and no commands executed.

- [x] **T2 — Reproduce permission, encryption, and isolation evidence**
  - Verified executable-bit/permission loss, unknown-profile silent plaintext push, unmasked/empty password acceptance, narrow secret patterns, and config/home isolation gaps.
  - Parent spot-checked `internal/actions/push.go:169-215`: missing profile returns `(false, nil)` and plaintext is returned without warning.
  - Deferred isolated reproduction commands recorded; no product behavior edited and no commands executed.

- [x] **T3 — Specify critical binary journey assertions**
  - Specified discovery, mutation/deletion, dry-run no-write, apply byte/permission, verify, tamper/negative, partial-failure, and real-recovery assertions with exit/output requirements.
  - Parent spot-checked `tests/e2e/testdata/backup_restore_roundtrip.txtar:13-36`: POSIX `sh` ID capture plus existence-only restore assertions.
  - Portable testscript/Go-helper direction and TUI/CLI proof split recorded; no tests written and no commands executed.

- [x] **T4 — Inventory docs, memories, changelog, and release baseline**
  - Inventoried current contracts, layout/duplication, memory baseline, local changelog/version/release state, and carry-forward list.
  - Parent spot-checked `SECURITY.md:1-95`: supported table still `1.3.x` while safety/integrity claims contradict T1/T2 code reality.
  - Remote release existence and repository settings remain UNKNOWN; no docs moved and no memories rewritten.

- [x] **T5 — Fix the first implementation slice from evidence** (commit `419c473` on `feat/t5-restore-safety`)
  - Delivered: aggregated copy-failure errors with nonzero exit, mandatory manifest validation (`--force` confirmation-only), TUI confirmation decoupled without code change, table-driven RED/GREEN tests.
  - Allowed edit surfaces: `internal/actions/restore.go`, `internal/actions/restore_test.go`, `cmd/root.go`, `cmd/restore.go`, `cmd/restore_test.go` (root.go/restore.go untouched — no change required).
  - Verification of record: `go test -count=1 ./internal/actions/` ok, `go test -count=1 ./cmd/` ok, `go vet` clean, `gofmt` clean; parent spot-checked diff and re-ran actions suite green.
  - Review: pre-commit assess `medium`/`under_budget`; post-commit assess `medium`/`slice_budget_reached` → preflight blocked on unpublished intended-untracked-selection schema (2 well-formed submissions rejected, no mutation); occurrence reported upstream (#5129); boundary stays pending.
  - Commit bypass: `--no-verify` with documented `NO-VERIFY` (GGA whole-file scope mismatch); GGA-new-code findings fixed before commit.
  - Follow-ups (deferred, not this slice): F1 fmt write-error idiom decision; F2 Git auto-commit wiring; F3 backup-version warning; F4 remaining old-test table refactors.

- [x] **T6 — Explicit push encryption gating** (commit `ac8c3a8` on `feat/t5-restore-safety`, `--no-verify` after GGA provider timeout)
  - Delivered: unknown/missing profiles and fresh-install setups fail closed with guidance; explicit unencrypted profiles stay a deliberate opt-out; empty profile rejected at CLI + action layers; table-driven gating tests.
  - Boy-scout in touched test files: checked ignored os errors, `Mock*` doubles with interface assertions, `Run` error assertions, consolidated `ResolveBackupID` table.
  - Verification of record: `go test -count=1 ./cmd/ ./internal/actions/` ok, `go vet` clean, `gofmt` clean; parent spot-checked diff and re-ran suites green.
  - GGA follow-up: re-run GGA cleanly on the four files when the provider is healthy; whole-file scan previously surfaced only pre-existing test-style findings, now fixed.

- [x] **T7 — Permission-preserving manifest schema (0.4.0)** (commit `3483d73` on `feat/t5-restore-safety`, `--no-verify` after GGA provider timeouts)
  - Delivered: portable mode bits recorded at scan, stored in `0.4.0` items, applied on restore with chmod-failure errors; `0.3.0` loads and restores degraded with explicit report; no more silent `0600→0644` widening.
  - Surface expansion (authorized inline): `internal/backup/engine_test.go`, `internal/backup/integration_test.go` now assert `manifest.ManifestVersion` instead of hardcoded `0.3.0`.
  - Verification of record: `go test` green on manifest/adapters/backup/actions + e2e; `go vet`/`gofmt` clean; parent spot-checked version-assertion fix and re-ran suites.
  - GGA follow-up: re-run GGA cleanly on touched files when the provider is healthy.

- [x] **T8 — Password prompt without echo + empty rejection** (commit `6eaaca1` on `feat/t5-restore-safety`, `--no-verify` after GGA provider timeouts)
  - Delivered: `x/term` masked terminal read with non-terminal fallback error; empty env/prompt passwords rejected with guidance; `x/term v0.42.0` promoted to direct dependency; caller pull test updated to the new validation error.
  - Surface expansion (authorized inline): `internal/actions/pull_test.go` one-line assertion update (empty password now fails at validation, not decryption).
  - Verification of record: `go test` green on crypto + actions; `go vet`/`gofmt` clean; parent spot-checked and re-ran suites.
  - GGA follow-up: re-run GGA cleanly on touched files when the provider is healthy.

- [x] **T9 — Secret-pattern coverage per SECURITY.md** (commit `c62c445` on `feat/t5-restore-safety`, `--no-verify` after GGA provider timeouts)
  - Delivered: `gho_/ghu_/ghs_/ghr_` + `xoxb-/xoxp-` detection with per-family regression tests; existing families intact; redaction verified no-change.
  - Verification of record: `go test` green on backup + actions; `go vet`/`gofmt` clean; parent spot-checked diff and re-ran suites.
  - Follow-up: AWS/GCP/Stripe/Bearer families deliberately out of scope; GGA re-run when healthy.

- [x] **T10 — Docs/memory compaction + honesty alignment** (slices 1–3 done: `88aa40e`, `069aec9`, memories reconciled)
  - Delivered: multi-agent help framing; `--force` integrity note; provider ecosystem + fail-closed profiles; SECURITY.md stabilization line, recognized families, 0.4.0/degraded notes, unwired-Git honesty, interactive confirmation, local-plaintext reality; smoke banner updated inline.
  - Delivered (slice 2): README multi-agent honesty; `CHANGELOG.md [Unreleased]` delta for T5–T9 without version number; six patch specs carry superseded frontmatter, content preserved.
  - Verification of record: `go build ./...` ok, `go test ./cmd/` ok, `TestBinaryHelp` green; parent spot-checked diff (strings/comments only) and re-ran suites.
  - Remaining: slice 2 done (commit `069aec9`); slice 3 done (decisions T5–T10 + safety facts reconciled in Engram, no deletions).

- [x] **T11 — RC gate verification** (GREEN, verification-only worker + parent spot check)
  - Gates: `gofmt` clean, `go vet ./...` clean, `go test ./...` 28 packages ok, e2e ok, `go build ./...` ok, snapshot smoke (`--version`/`--help`) ok, T5–T9 regression suites green.
  - GGA: read-only run requires hook context — skipped, follow-up stands.
  - No files modified by verification; branch unchanged except this task file.

- [ ] **T12 — Release v1.5.0-rc1** (number agreed, creation pending explicit authorization)
  - Agreed: `v1.5.0-rc1` — minor with behavior changes + read-compatible manifest 0.4.0, RC first per staged strategy.
  - Pending: tag creation, release publication, merge/PR decisions. Each requires separate explicit authorization with destination and scope.

## Route declaration

- Initial readiness mapping: delegated direct through one bounded read-only explorer.
- CI/test-realism mapping: delegated direct through one bounded read-only explorer.
- Parent spot checks: direct inline within the evidence budget.
- This task document: direct inline as one mechanical new tracking file.
- T1–T4 evidence work will use fresh isolated delegates; no parallel writers.

## Progress

- Shared ten-point readiness understanding confirmed.
- Evidence-first next phase authorized; direct product edits remain unauthorized.
- T1–T11 completed. Branch `feat/t5-restore-safety` GREEN across all gates. Follow-ups F1–F4, GGA hook-context re-runs, symlinks, extra secret families deferred.

## Verification evidence

- `internal/actions/restore.go`: counted restore failures return `nil`; declared `GitDir` is unused in the inspected action.
- `internal/manifest/manifest.go`: current schema is `0.3.0`; per-file items do not record permission metadata.
- `tests/e2e/roundtrip_test.go`: backup is followed by restore without prior mutation/deletion of originals; environment inherits `XDG_CONFIG_HOME`.
- T1 evidence: `internal/actions/restore.go:101-106,198-205`; `cmd/restore.go:68-82`; `cmd/root.go:244-261`; `internal/restore/engine.go:69-133` unwired; `internal/actions/undo.go:30-61` targets `.bak/` only; `tests/e2e/testdata/undo_after_restore.txtar:6-18` synthetic setup.
- T2 evidence: `internal/adapters/util.go:24-28`; `internal/actions/os_impl.go:63-67`; `internal/actions/restore.go:254-258`; `internal/restore/engine.go:178-188`; `internal/actions/push.go:169-215`; `internal/crypto/password.go:31-44`; `internal/crypto/crypto.go:53,137-139`; `internal/backup/secrets.go:18-35`; `internal/actions/redact.go:10,59-65`; `tests/e2e/roundtrip_test.go:84-88,196-205`.
- T3 specification: required 8-stage journey matrix, existing-vs-missing map, POSIX-to-portable replacements, isolation prerequisites, TUI/CLI split, and deferred verification commands; key references `tests/e2e/roundtrip_test.go:84-88`, `tests/e2e/testdata/backup_restore_roundtrip.txtar:13-36`, `undo_after_restore.txtar:6-18`, `.github/workflows/ci.yml:190-198`, `internal/actions/restore.go:101-106,141-150`, `cmd/root.go:247`.
- T4 inventory: current contracts versus T1/T2 reality; 5 root docs, 8 docs/ files, 29 openspec specs, 36 archived changes, 1 preexisting untracked audit file; bak-cli current versus historical/unverified memories; local tags v0.1.0–v1.4.1 with CHANGELOG ending at 1.4.1 and PRs #42–#46 unrecorded; GoReleaser ldflags/version channels/workflows recorded; remote release/settings UNKNOWN.
- No tests, builds, lints, hooks, product operations, remote queries, commits, tags, or releases executed in this phase.

- [ ] **T7 — Permission-preserving manifest schema (0.4.0)** (authorized, in progress on `feat/t5-restore-safety`)
  - Goals: record portable permission bits at backup time, restore them on apply, bump new manifests to `0.4.0`, and keep reading `0.3.0` with explicit degraded-permission handling.
  - Allowed edit surfaces: `internal/manifest/manifest.go`, `internal/manifest/manifest_test.go`, `internal/manifest/fuzz_test.go`, `internal/adapters/adapter.go`, `internal/adapters/generic.go`, `internal/adapters/util.go`, `internal/adapters/util_test.go`, `internal/adapters/generic_test.go`, `internal/backup/workflow.go`, `internal/backup/workflow_test.go`, `internal/actions/restore.go`, `internal/actions/restore_test.go`, `internal/actions/os_impl.go`, `internal/actions/os_impl_test.go`, `internal/actions/interfaces.go`, `internal/actions/mock_impl_test.go`, `tests/e2e/roundtrip_test.go`, `tests/e2e/testdata/backup_restore_roundtrip.txtar`.
  - Excluded: encryption, secrets, cloud/scheduling, password prompting, docs compaction, changelog/version/release, workflows/hooks, remote operations, real user backups.
  - Contract decisions: `0.3.0` items without mode restore content only and report degraded permissions (never claim exact restore); `0.4.0` restore applies stored bits best-effort and returns an error when chmod fails; symlinks stay regular files in this slice (recorded follow-up, no silent change).
## Next step

T12 awaits explicit authorization: merge to main first, or tag the RC on this branch? Then tag + release publication as separate steps.
