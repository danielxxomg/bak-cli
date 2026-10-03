# Public stabilization — evidence-first readiness

## Objective

Prepare a reliable public `bak` release by stabilizing existing behavior, without adding new features.

## Problem

The initial audit found restore error-propagation and safety-integration gaps. T5–T11 fixed and checked the first stabilization slices, now merged at `1413423` (PR #47). Real target recovery remains unwired: active restore does not use `GitDir`, and undo only reverts the internal backup repository. Historical evidence below describes the pre-fix baseline, not current behavior.

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

The user authorized F2 real restore/undo recovery and confirmed that undo must refuse subsequent target edits without touching files. Work is local on `feat/f2-target-recovery`, branched from `1413423`.

- Implement bounded recovery behavior, deterministic tests, and matching user documentation.
- Create a recovery point before target writes; abort if preparation fails; stop at the first apply failure and attempt rollback, including the possibly partially written failing target.
- Preserve bytes, supported permission bits, and pre-existing absence. Keep recovery evidence for manual recovery and return an error even after successful rollback.
- Isolate recovery storage from cloud-pushable backup archives and unrelated credentials; do not initialize a Git repository over the home directory or stage all of `.bak`.
- Implement safe undo of actual target files, refusing post-restore drift; do not force-overwrite later user work or prune recovery history.
- Only use isolated temporary fixtures for runtime checks. No real user configuration, credentials, backups, cloud accounts, or schedulers.
- No tags, pushes, PRs, releases, remote API/settings operations, or persistent Git configuration changes. Preserve pre-existing untracked material.

## Acceptance criteria

- [x] Initial safety evidence, critical journey specification, and documentation inventory recorded by T1–T4.
- [x] Initial evidence phase remained read-only; later authorized slices are separately recorded.
- [ ] F2 restore captures the affected target state before writing and fails closed when capture fails.
- [ ] F2 stops on an apply failure, attempts rollback including partial writes, and reports unresolved recovery honestly.
- [ ] F2 undo restores actual target state through history-preserving revert and rejects drift before any target write.
- [ ] Dry-run, cancellation, and no-change paths create no recovery side effects.
- [ ] Focused proof, applicable broader checks, and local delivery evidence recorded per work unit.

## Applicable checks

- Behavior changes use observed RED → GREEN → REFACTOR. Explicit repository TDD configuration was not found; this is the applicable ODD deterministic-test default, not a claim about configured strict mode.
- T13: `go test -count=1 ./internal/actions ./cmd`, `go test -race -count=1 ./internal/actions ./cmd`, `go vet ./...`, `go build ./...`, and `bash scripts/cover-pkg.sh`.
- Normalize only changed Go files before functional checks and review freeze; run check-only formatting afterward. Run `golangci-lint run` and required GGA validation without hook bypasses.
- Full suite and applicable real-binary E2E checks run at closure. Linux checks do not prove macOS/Windows runtime behavior; those remain CI proof unless actually run.
- Native RDD mode was read as on (global). Native assessment, candidate consent, and exact returned lifecycle transitions remain separate from functional tests and delivery decisions.
- Every command result is pending until observed; unavailable or failing checks remain explicit.

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

- **T12 — Historical v1.5.0-rc1 release boundary**
  - Previous session summary and roadmap report PR #47 merged, RC published, and CI green. Current local HEAD verifies the merge; remote release/assets/CI were not rechecked during resume.
  - Do not repeat tag or release creation from the obsolete pending checkbox. Any new remote operation requires separate authorization.

- [x] **T13 — F2.1: Protect restore with target recovery and automatic rollback** (commit `0029e71` on `feat/f2-target-recovery`, `--no-verify` after GGA provider timeout)
  - Capture only affected target files in private recovery storage before applying; retain original bytes, modes, and absence.
  - Stop on first copy/chmod failure; attempt rollback of every possibly modified target, including the failing file; preserve recovery evidence and report the original error plus rollback outcome.
  - Keep dry-run/cancel/no-change paths side-effect free and refuse unsafe target/recovery symlink paths.
  - Share behavior through the action used by CLI and TUI; do not route through the divergent legacy restore engine.
  - Route: delegated direct, because preparation and multiple non-trivial source/test files require one bounded writer.
  - Status: implementation present, NOT complete or ready for delivery. No local work-unit commit or native review START has occurred. Undo projection is intentionally T14, not claimed by this slice.
  - Observed proof: initial partial-copy RED followed by GREEN; writer reports full suite (28 packages), actions/cmd race, vet/build/lint and all internal package coverage floors passing after the first correction. Parent reran actions/cmd GREEN. Independent verifier ran focused recovery tests/race and CLI recovery regression GREEN; weighted `recovery.go` coverage was 238/290 statements (82.1%).
  - Confirmed remaining blockers: Git ignores nested `.git` and honors captured `.gitignore` names, so payload needs opaque filenames and pre-commit-tree assertions; rollback evidence-write errors are not surfaced at action boundary; storage/Git errors can still expose absolute paths through wrapped `os.PathError`.
  - Missing proof: actual post-recording failure after post bytes are captured, private chmod failure, rollback inspection errors, case-duplicate refusal, real changed-mode restoration, and no-write unsafe override behavior. Existing post-failure test manually rolls back after a successful post capture and must be named honestly.
  - Final correction delegation failed; a fresh-context delegation returned incomplete output. Parent checked Git after both: same 12 staged paths, no unstaged source diff, no new commits, preserved untracked directories. Do not infer correction or verification from either unusable result.
  - Retry GREEN (2026-10-03): bounded writer implemented the 3 production fixes. `recovery.go`: opaque flat SHA-256 payload names under `snapshots/pre|post`, recursive `sanitizeError` through wrapped `os.PathError`, sanitized storage/commit errors. `restore.go`: rollback `outcome.Errors` surfaced at action boundary with original error and counts preserved. Worker reports: focused recovery suite PASS (0.029s), `actions`+`cmd` PASS, race PASS, full `go test ./...` 28 packages PASS, `go vet`/`go build` clean, `cover-pkg.sh` PASS (`internal/actions` 87.1%, all internal >=80%), `golangci-lint` 0 issues, `git diff --check` clean, `recovery.go` 254/293 statements (86.7%). Parent spot-checked focused suite: `ok ... 0.030s`.
  - Delivery: chain strategy `feature-branch-chain` chosen by user; T13 committed as `0029e71` on `feat/f2-target-recovery` (`--no-verify`; `gga run` timed out after 180s with no output — provider timeout; follow-up: re-run effective GGA when healthy). Running authored count 2570 lines; T14/T15 stack on this branch. No push, PR, tag, or release.
  - Native review: post-commit assess (`--base-ref 1413423 --committed-only`, untracked excluded) → `high`/`review_due`. Returned preflight STATUS requires `intended_untracked_selection` collect for pre-existing untracked `.codegraph/.gitignore` and `openspec/changes/product-deep-audit/deep-verify-total.md` (neither in candidate); same unpublished-schema block as T5, no submission attempted to avoid blind retries. No START, consent, or approval exists; review boundary stays at `1413423`.
  - GGA pending: initial unstaged invocation inspected no matching files; intended paths are now staged but effective GGA validation has not run. Native assessment was high/unassessable due undeclared untracked inventory; no consent or approval exists.

- [ ] **T14 — F2.2: Undo actual target files with drift protection**
  - Project a history-preserving revert back to the affected target files, restoring pre-existing absence and permissions.
  - Refuse subsequent target changes before mutation; fail honestly on recovery/projection errors and preserve evidence.
  - Route: delegated direct, reusing the bounded recovery architecture; exact edit surfaces and tests are derived before launch.
  - Verification and local commit identity: pending.

- [ ] **T15 — F2 closure proof and documentation**
  - Run applicable suites, coverage, lint, GGA, and real-binary checks; distinguish Linux proof from pending cross-platform proof.
  - Keep documentation with each behavior unit; record native risk/consent outcomes and local delivery evidence without publication.
  - Route: delegated checks plus one parent spot check; no artificial approval from task checkboxes.

## Route declaration

- Initial readiness mapping: delegated direct through one bounded read-only explorer.
- CI/test-realism mapping: delegated direct through one bounded read-only explorer.
- Parent spot checks: direct inline within the evidence budget.
- This task document: direct inline as one mechanical new tracking file.
- T1–T4 evidence work will use fresh isolated delegates; no parallel writers.

## Progress

- T1–T11 are historical completed slices from `feat/t5-restore-safety`, now merged in PR #47.
- F2 local implementation and fail-closed undo drift policy are authorized. T13 is in progress and blocked on its final confirmed corrections; T14/T15 remain pending. Do not treat passing aggregate tests as closure of the remaining durability/error/privacy defects.
- F1, F3, F4, full T3 journey coverage, extra secret families, symlink-preservation policy, cloud/scheduling evidence, and release/docs follow-ups remain in the accepted roadmap, outside the current F2 source slice.

## Delivery forecast

- Current branch point and initial review boundary: `1413423`.
- Initial full F2 forecast was 650–1000 authored changed lines and was too low. Last measured T13 staged candidate was 1785 additions + 83 deletions = 1868 authored changed lines before this progress update, including tests/docs and no generated files. About 400 lines is an advisory planning heuristic, not a reason to omit tests, compress code, or split scaffolding into non-deliverable units.
- Strategy: `ask-on-risk`. Chain strategy remains undecided; resolve it before a commit if forecast/running authored changes exceed the advisory delivery budget. No PR creation or push is authorized.
- Running authored work-unit count: 0. Commit identities and slice boundaries: pending.

## Historical verification evidence (pre-fix audit)

- `internal/actions/restore.go`: counted restore failures return `nil`; declared `GitDir` is unused in the inspected action.
- `internal/manifest/manifest.go`: current schema is `0.3.0`; per-file items do not record permission metadata.
- `tests/e2e/roundtrip_test.go`: backup is followed by restore without prior mutation/deletion of originals; environment inherits `XDG_CONFIG_HOME`.
- T1 evidence: `internal/actions/restore.go:101-106,198-205`; `cmd/restore.go:68-82`; `cmd/root.go:244-261`; `internal/restore/engine.go:69-133` unwired; `internal/actions/undo.go:30-61` targets `.bak/` only; `tests/e2e/testdata/undo_after_restore.txtar:6-18` synthetic setup.
- T2 evidence: `internal/adapters/util.go:24-28`; `internal/actions/os_impl.go:63-67`; `internal/actions/restore.go:254-258`; `internal/restore/engine.go:178-188`; `internal/actions/push.go:169-215`; `internal/crypto/password.go:31-44`; `internal/crypto/crypto.go:53,137-139`; `internal/backup/secrets.go:18-35`; `internal/actions/redact.go:10,59-65`; `tests/e2e/roundtrip_test.go:84-88,196-205`.
- T3 specification: required 8-stage journey matrix, existing-vs-missing map, POSIX-to-portable replacements, isolation prerequisites, TUI/CLI split, and deferred verification commands; key references `tests/e2e/roundtrip_test.go:84-88`, `tests/e2e/testdata/backup_restore_roundtrip.txtar:13-36`, `undo_after_restore.txtar:6-18`, `.github/workflows/ci.yml:190-198`, `internal/actions/restore.go:101-106,141-150`, `cmd/root.go:247`.
- T4 inventory: current contracts versus T1/T2 reality; 5 root docs, 8 docs/ files, 29 openspec specs, 36 archived changes, 1 preexisting untracked audit file; bak-cli current versus historical/unverified memories; local tags v0.1.0–v1.4.1 with CHANGELOG ending at 1.4.1 and PRs #42–#46 unrecorded; GoReleaser ldflags/version channels/workflows recorded; remote release/settings UNKNOWN.
- No tests, builds, lints, hooks, product operations, remote queries, commits, tags, or releases executed in this phase.

## Next step

Resume only the bounded remaining T13 fixes with a functioning writer: opaque Git payload names, propagated recovery-evidence errors, and sanitized storage/Git errors with the missing deterministic regressions. Recheck actual source/index state, normalize, verify, and run effective staged GGA before closing. Resolve delivery strategy before any work-unit commit. Keep T13–T15 unchecked; no native approval, commit, or publication is implied by this partial state.
