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
  - Review: assess `medium`/`under_budget` — no native review due for this slice; boundary stays pending.
  - Commit bypass: `--no-verify` with documented `NO-VERIFY` (GGA whole-file scope mismatch); GGA-new-code findings fixed before commit.
  - Follow-ups (deferred, not this slice): F1 fmt write-error idiom decision; F2 Git auto-commit wiring; F3 backup-version warning; F4 remaining old-test table refactors.

## Route declaration

- Initial readiness mapping: delegated direct through one bounded read-only explorer.
- CI/test-realism mapping: delegated direct through one bounded read-only explorer.
- Parent spot checks: direct inline within the evidence budget.
- This task document: direct inline as one mechanical new tracking file.
- T1–T4 evidence work will use fresh isolated delegates; no parallel writers.

## Progress

- Shared ten-point readiness understanding confirmed.
- Evidence-first next phase authorized; direct product edits remain unauthorized.
- T1–T5 completed; T5 work-unit commit `419c473` on `feat/t5-restore-safety`. Follow-ups F1–F4 deferred to future slices.

## Verification evidence

- `internal/actions/restore.go`: counted restore failures return `nil`; declared `GitDir` is unused in the inspected action.
- `internal/manifest/manifest.go`: current schema is `0.3.0`; per-file items do not record permission metadata.
- `tests/e2e/roundtrip_test.go`: backup is followed by restore without prior mutation/deletion of originals; environment inherits `XDG_CONFIG_HOME`.
- T1 evidence: `internal/actions/restore.go:101-106,198-205`; `cmd/restore.go:68-82`; `cmd/root.go:244-261`; `internal/restore/engine.go:69-133` unwired; `internal/actions/undo.go:30-61` targets `.bak/` only; `tests/e2e/testdata/undo_after_restore.txtar:6-18` synthetic setup.
- T2 evidence: `internal/adapters/util.go:24-28`; `internal/actions/os_impl.go:63-67`; `internal/actions/restore.go:254-258`; `internal/restore/engine.go:178-188`; `internal/actions/push.go:169-215`; `internal/crypto/password.go:31-44`; `internal/crypto/crypto.go:53,137-139`; `internal/backup/secrets.go:18-35`; `internal/actions/redact.go:10,59-65`; `tests/e2e/roundtrip_test.go:84-88,196-205`.
- T3 specification: required 8-stage journey matrix, existing-vs-missing map, POSIX-to-portable replacements, isolation prerequisites, TUI/CLI split, and deferred verification commands; key references `tests/e2e/roundtrip_test.go:84-88`, `tests/e2e/testdata/backup_restore_roundtrip.txtar:13-36`, `undo_after_restore.txtar:6-18`, `.github/workflows/ci.yml:190-198`, `internal/actions/restore.go:101-106,141-150`, `cmd/root.go:247`.
- T4 inventory: current contracts versus T1/T2 reality; 5 root docs, 8 docs/ files, 29 openspec specs, 36 archived changes, 1 preexisting untracked audit file; bak-cli current versus historical/unverified memories; local tags v0.1.0–v1.4.1 with CHANGELOG ending at 1.4.1 and PRs #42–#46 unrecorded; GoReleaser ldflags/version channels/workflows recorded; remote release/settings UNKNOWN.
- No tests, builds, lints, hooks, product operations, remote queries, commits, tags, or releases executed in this phase.

## Next step

Propose the next slice (permissions manifest schema, encryption gating, or remaining GGA follow-ups) for user authorization; no further source edits without it.
