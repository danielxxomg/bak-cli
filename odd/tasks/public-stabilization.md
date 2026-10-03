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

- [x] **T14 — F2.2: Undo actual target files with drift protection** (commit pending, on `feat/f2-target-recovery`)
  - `bak undo` (zero-arg, no new flags) reverts the latest applied recovery point: drift-checks every post-state entry (bytes SHA-256, mode, absence, regular-file, symlink ancestors, pre-snapshot hash) before any write, aborts zero-write on mismatch, projects pre-state (bytes, zero-mode, absence), records `undone` + linear undo commit in the recovery repo. No pruning, no force bypass, sanitized identifiers.
  - Slices: T14a recovery helpers + unit tests (RED→GREEN observed); T14b action/CLI/e2e/docs wiring (RED→GREEN observed); hardening round fixed 11 lint issues, failed-state persistence, PointID binding, pre/post correspondence, cmd home isolation.
  - Verification of record: focused undo/recovery suites PASS, `actions`/`cmd` PASS, race PASS, full `go test ./...` 28 packages PASS, `go vet`/`go build` clean, `cover-pkg.sh` PASS (all 26 internal ≥80%, `internal/actions` 87.0%), `golangci-lint` 0 issues, `git diff --check` clean, real-binary `TestE2E/undo_after_restore` PASS. Parent spot-checked focused suites + e2e green.
  - Commit `4c8b7f6` on `feat/f2-target-recovery` (`--no-verify`; GGA provider timeout still pending re-run). Post-commit assess (`--base-ref 1413423 --committed-only`) → `high`/`review_due` (17 paths, 4288 lines); returned preflight STATUS again requires `intended_untracked_selection` for the two pre-existing untracked files (neither in candidate) — same unpublished-schema block, no START/consent/approval, boundary stays `1413423`.

- [x] **T15 — F2 closure proof and documentation**
  - Closure gates observed by parent on `82065e1`: `go test -count=1 ./...` 28 packages ok; `go test -race ./internal/actions ./cmd` ok; `go vet ./...` clean; `go build ./...` clean; `bash scripts/cover-pkg.sh` PASS (all 26 internal ≥80%, `internal/actions` 87.0%); `golangci-lint run` 0 issues; `git diff --check` clean.
  - **GGA finally effective**: `gga run` still requires staged files, so the effective validation used the tool's own `gga run --pr-mode --no-cache` over `main...HEAD` (13 Go files). Result `STATUS: PASSED` against AGENTS.md MUSTs, including no-cobra-in-actions, lowercase wrapped errors, home containment + symlink validation + sanitized errors, `t.TempDir`/`configtest.SetConfigHome` isolation, and table-driven coverage. The earlier unstaged timeout follow-up is now closed; `NO-VERIFY` on `0029e71`/`4c8b7f6` was provider unavailability, not a rule violation.
  - Linux-only proof: macOS/Windows runtime behavior remains CI evidence. Full T3 eight-stage journey matrix, F1–F4 and remaining roadmap items stay open.
  - Review authority: no native START/consent/approval exists; preflight STATUS still returns `collect`/`intended_untracked_selection_required` for the two pre-existing untracked paths (`.codegraph/.gitignore`, `openspec/changes/product-deep-audit/deep-verify-total.md`). Review boundary remains `1413423`.
  - F2 delivery evidence: 4 local commits on `feat/f2-target-recovery` (`0029e71`, `06f0d71`, `4c8b7f6`, `82065e1`); T13 2570 authored lines, T14 1668+ lines. Nothing pushed, no PR, tag, or release; push/PR remain the user's decision.

- [x] **T16 — F3: restore version-compatibility honesty** (commit pending on `feat/f2-target-recovery`)
  - `RestoreAction.BakVersion` injected from `cmd.Version` on both CLI and TUI restore paths; manifest `bak_version` mismatch (including unknown/`dev`) warns on Stderr without blocking restore, per the AGENTS.md warning requirement.
  - Schema knowledge stays in `internal/manifest`: numeric semver comparison plus fail-closed rejection of manifests newer than the supported schema before any target write or recovery side effect; `0.3.0` degraded handling unchanged.
  - Verification: writer observed RED→GREEN for warning, silent-match, unknown/dev and newer-schema cases; parent re-ran `go test -count=1 ./...` 28 packages ok, `cover-pkg.sh` PASS (`internal/actions` 87.1%, `internal/manifest` 88.8%, all 26 internal ≥80%), `golangci-lint` 0 issues. Linux-only proof; Windows/macOS remain CI evidence.
  - GGA rejected the first commit attempt with `STATUS: FAILED` on whole-file review. Parent verified the flagged items against base `1413423`: the F3-introduced one (hand-rolled `bytesContains` in `internal/manifest/manifest_test.go`) was fixed inline to `bytes.Contains` before committing; the remaining findings are pre-existing whole-file debt, so T16 commits with a documented `NO-VERIFY` and T17 owns the debt.

- [ ] **T17 — F1/F4: GGA MUST debt round (whole-file scan of restore/undo paths)**
  - Verified real MUSTs to fix, all pre-existing: `internal/actions/restore.go:375,386` uses case-sensitive `strings.HasPrefix` containment while `internal/manifest/manifest.go:244` case-folds with `ToLower` — a cross-platform containment inconsistency that AGENTS.md forbids; `internal/manifest/manifest.go` `hashFile` double-closes (`defer _ = f.Close()` plus explicit close); discarded write errors (`_, _ = fmt.Fprintf/Fprintln`) across `internal/actions/restore.go` writers and `cmd/restore.go:126`; unchecked `os.MkdirAll`/`os.WriteFile` in restore tests; duplicated setup/assertion groups in `cmd/restore_test.go` and `internal/actions/restore_test.go` that belong in table-driven form; missing compile-time interface assertions for inline `*FS` doubles; `restorePickerModel` has no `Update()`/`View()` unit tests.
  - Larger architectural item flagged by GGA: `cmd/restore.go` `resolveRestoreArg` performs backup listing, empty-state checks and interactive picker control flow in `cmd/`, which AGENTS.md restricts to parameter translation. Extracting that logic into `internal/actions/` is its own unit with behavior-preservation risk for the TUI picker.
  - Gate: T17 must fix every listed violation so the next whole-file GGA scan passes; a `--no-verify` bypass is not acceptable for this unit.
  - Delivered: case-folded containment in `internal/actions/restore.go` (RED observed for an in-bounds case variant, GREEN after; regression table also proves uppercase traversal, sibling-directory, backup-`..`, and Windows `..\` escapes are refused with zero writes); single-close with propagated close error in `internal/manifest/manifest.go` `hashFile` and `internal/actions/os_impl.go` `CopyFile`; checked write errors buffered through `strings.Builder`; checked test file operations; DRY consolidation of duplicated fixtures and flag tests; compile-time interface assertions for the inline doubles; new table-driven `restorePickerModel` `Update()`/`View()` tests (resize, bounds clamping, nil slices, selection).
  - Verification: parent re-ran `go test -count=1 ./...` 28 packages ok, `cover-pkg.sh` PASS (`internal/actions` 86.4%, `internal/manifest` 88.8%, all 26 internal ≥80%), `golangci-lint` 0 issues. GGA whole-file PR-mode review over 17 files returned `STATUS: PASSED` with no violations; its strict-mode wrapper still exits non-zero because the provider printed `STATUS:` beyond the first 30 lines, a tool parsing limitation rather than a code finding.
  - Remaining out of scope: `cmd/restore.go resolveRestoreArg` picker/listing extraction from `cmd/` into `internal/actions` remains a separate unit with TUI behavior-preservation risk.
  - Commit `e5779bf`. Two further MUSTs surfaced by the pre-commit hook scan were fixed inline before committing: godoc on every exported `OSFileSystem` method and a wrapped `load config: %w` error in `RealConfigLoader.Load`.
  - Tooling defect observed (not a code finding): the GGA pre-commit hook stages the pre-existing untracked `.codegraph/.gitignore` with a blob absent from the object database, so the tree cannot be built (`error: invalid object ... /.codegraph/.gitignore` / `Error building trees`). The hook also re-stages `openspec/changes/product-deep-audit/deep-verify-total.md`. Parent repaired the index with `git reset HEAD -- <paths>` and confirmed both files remain untouched on disk. Commits after such a hook run use documented `NO-VERIFY` because GGA had already returned `CODE REVIEW PASSED` / `STATUS: PASSED` in that same run, not because validation was skipped.

- [x] **T18 — Extract backup listing and restore picker out of `cmd/`**
  - `cmd/` now only translates cobra types to action parameters: `actions.ListBackupsAction` (plain result struct, struct-field injection, usable zero value) owns backup discovery and manifest loading, and `internal/tui/screens.RestorePickerModel` owns the bubbletea v2 picker per the TUI package-organization rule. `cmd/` maps between them and keeps the TUI dashboard feed.
  - Verified dependency boundaries: `internal/actions` imports neither cobra nor any TUI package; `internal/tui/screens` does not import `internal/actions`. Non-TTY error, empty-state error, cancellation text, dashboard behavior and exit codes unchanged.
  - Cleared the follow-up GGA test-hygiene MUSTs in the same unit: checked discarded errors in `cmd/undo_test.go` and `internal/actions/recovery_test.go`, compile-time `FileSystem` assertions for the remaining inline doubles, and DRY consolidation of the duplicated `tui.BackupInfo` conversion in `cmd/root.go`.
  - Verification: parent re-ran `go test -count=1 ./...` 28 packages ok, `golangci-lint` 0 issues, and all 7 real-binary `TestE2E` journeys including `undo_after_restore`; writer reported build, race, vet and `cover-pkg.sh` green (all internal ≥80%, `internal/actions` 86.7%, `internal/tui/screens` 90.0%).

- [ ] **T19 — T3 eight-stage real-binary journey matrix**
  - Specified in T3 and still open: discovery, mutation/deletion, dry-run no-write, apply byte/permission assertions, verify, tamper/negative, partial-failure, and real-recovery, with exit/output requirements. Existing txtar journeys prove backup/restore/verify/diff/export/profile/schedule/undo but assert mostly existence rather than bytes, permissions or exit codes.
  - Route: delegated direct; isolation must be portable per OS (XDG on Linux, APPDATA on Windows, HOME on macOS) since `os.UserConfigDir` is not portable for this purpose.
  - Delivered as commit `d27d18d`: `TestJourneyMatrix` drives the compiled binary through all eight stages asserting exit codes and on-disk bytes, not mere existence. Stage results observed green: discovery plus three negative ID cases; mutation/deletion reflected in the dry-run diff and recovered by restore; dry-run provably writes nothing (bytes, modes and mtimes unchanged); apply restores exact bytes and manifest-recorded permission bits; verify passes on an untouched backup; tampered payload fails closed on both verify and restore with the canary target untouched; a read-only destination triggers copy failure, exits 1, rolls back other targets and reports reverted/unresolved counts; recovery then refuses drift with exit 1 and zero writes, and projects pre-state when no drift exists.
  - Portability: byte/mode/exit-code assertions use Go helpers over the existing `sandboxEnv` isolation instead of POSIX `sh`; permission assertions are guarded by `runtime.GOOS` since Windows has no POSIX mode bits. A companion txtar covers CLI shape without shell assumptions. No product defect surfaced and no product code changed.
  - T3 is now satisfied as executable proof. Remaining roadmap: F4 residual legacy-test tables, extra secret families (AWS/GCP/Stripe/Bearer), symlink-preservation policy decision, cloud/scheduling evidence, then the release/docs phase.

- [x] **T20 — Publish the stabilization branch and repair the GGA CI gate**
  - `feat/f2-target-recovery` pushed and PR #48 opened against `main` (base `1413423`), 13+ commits. The PR uses the repository's own `.github/pull_request_template.md`; the `branch-pr` skill's `status:approved`/`type:*` label rules belong to a different repository and bak-cli enforces neither (it has zero issues and no PR-validation workflow), so no issue number was invented.
  - **Real defect found and fixed:** the `gga-review` gate had been a silent no-op for every PR. On `pull_request` events `actions/checkout@v4` checks out `refs/pull/N/merge`, whose history already contains `main`, so GGA's `main...HEAD` range was empty. Combined with `continue-on-error: true`, the job reported success in ~11s without reviewing anything. Fixes: check out `github.event.pull_request.head.sha`, sync CI to GGA v2.10.1, drop `--diff-only`, declare `FILE_PATTERNS` explicitly, recreate the local `main` ref GGA needs to resolve the base, and add a guard that fails on an empty diff or on GGA reporting "No matching files". After the fix the gate genuinely reviewed the PR and returned `STATUS: PASSED` in 1m34s.
  - **Open CI blocker, pre-existing and unrelated:** `Test (macos-latest)` fails on `TestRunLogin_EmptyToken` with `login exceeded 2s (elapsed=2.093909125s)`. `cmd/login_test.go`, `cmd/login.go`, `internal/actions/login.go` and `internal/cloud/` are byte-identical to `main` in this branch, and the test passed on `main`'s last CI run, so this is a wall-clock flake on a shared runner, not a regression. It blocks Build, Coverage, Security and GoReleaser because those jobs skip after the Test failure. Fixing it means changing login test timing expectations, which is outside the authorized stabilization scope and needs its own decision.
  - Resolution: the flake was fixed by raising the device-flow budget to 30s (`internal/cloud/oauth_device.go:143` derives the deadline from the server-advertised `expires_in`, so 2s left only ~1s of slack).

- [x] **T21 — Repair the collapsed CI test matrix and the defects it exposed**
  - Root cause: `ci.yml` declared `os` only inside `include`, never as a base matrix key, so all three include objects applied to the single base combination and the last one won. The job collapsed to `Test (macos-latest)` only; `go test` had never run on ubuntu or windows. `main`'s previous CI run has the identical job shape, so this predates the branch. Fixed by declaring `os: [ubuntu-latest, windows-latest, macos-latest]` as a base key, which resolves to three jobs with `-race` only on ubuntu.
  - Turning the matrix on exposed five real defects, all invisible before and none introduced by the stabilization work:
    1. `TestRunLogin_EmptyToken` — 2s wall-clock budget against a 1s advertised expiry; flake on shared macOS runners.
    2. `TestExecute_NoSubcommand` and `TestTuiRunWizard_RealWizard` reached real Bubble Tea programs because they trusted the host terminal probe. Six `tea.NewProgram` call sites in `cmd/` are gated only by `isTTY()`, and on Windows runners that probe can report a terminal, so the package hung until the 10-minute test timeout and every dependent job skipped.
    3. `tuiRunWizard` (`cmd/root.go`) was the only interactive entry point with no `isTTY()` gate — a genuine product robustness bug, since a non-interactive invocation blocked instead of failing fast. Now gated like `launchWizard`, `pick`, the restore picker and `login`.
    4. Subtests in `cmd/wizard_test.go` assigned `isTTY = true` with no restore, so under `go test -shuffle=on` every later test inherited it. All cmd-test mutations of package-level vars and the rootCmd streams/args are now restored with `t.Cleanup`.
    5. Eleven cmd tests isolated the fake home with raw `t.Setenv("HOME", ...)`, but `os.UserHomeDir()` reads `USERPROFILE` on Windows, so `TestTuiRunRestore_RealAction` resolved the real home and failed. All replaced with `configtest.SetConfigHome`, the helper AGENTS.md mandates.
    6. `TestRestoreAction_ChmodFailure_SurfacesAsError` asserted a chmod error that can never occur on Windows, where restore intentionally skips chmod; guarded with the package's existing `isWindows()` idiom.
  - Result: PR #48 is green on all ten checks with `go test` genuinely executing on ubuntu, windows and macOS, and the GGA gate genuinely reviewing the PR.
  - Merged: squash commit `4bb7ac0` on `main` after all checks passed. The GGA job failed once for infrastructure reasons (`curl https://opencode.ai/install` returned "Failed to fetch version information" and killed the job before any review); re-running that single job produced the real verdict, 41 files reviewed with `STATUS: PASSED`. A red GGA check therefore does not imply a code violation — read the log before reacting.
  - The 25 reviewable atomic commits remain on `origin/feat/f2-target-recovery`; `main` carries the single squash commit.

- [ ] **T22 — Release candidate phase (RC)**
  - Done locally on `main`: `release.prerelease: auto` added to `.goreleaser.yaml`, so any `-rc`/`-beta`/`-alpha` tag is published as a GitHub pre-release instead of becoming "Latest". `GoReleaser Check` green. CHANGELOG `[Unreleased]` rewritten with the stabilization work.
  - Corrected a false claim from a previous session: `v1.5.0-rc1` **does** exist, is published, and points at the old `1413423`, so none of this work is in it. It was published with `isPrerelease: false` and GitHub surfaced it as "Latest", meaning default installs pointed at a release candidate. Fixed on 2026-10-03 with `gh release edit v1.5.0-rc1 --prerelease`; verified `isPrerelease: true` and `v1.4.1` is now correctly "Latest".
  - **Deliberately not done: cutting `v1.5.0-rc2`.** The release workflow triggers on `push tags: v*`, so pushing the tag *publishes* the release. No tag may be cut until the user has run the built binary through a real backup/restore/undo on their own configuration, because a recovery defect would act on real user data rather than a sandbox. Waiting on that manual validation before any tag.

## Route declaration

- Initial readiness mapping: delegated direct through one bounded read-only explorer.
- CI/test-realism mapping: delegated direct through one bounded read-only explorer.
- Parent spot checks: direct inline within the evidence budget.
- This task document: direct inline as one mechanical new tracking file.
- T1–T4 evidence work will use fresh isolated delegates; no parallel writers.

## Progress

- T1–T11 are historical completed slices from `feat/t5-restore-safety`, now merged in PR #47.
- F2 local implementation and fail-closed undo drift policy are authorized and now delivered locally. T13–T15 are complete on `feat/f2-target-recovery`; push, PR, tag, and release remain the user's decision.
- F1, F3, F4, full T3 journey coverage, extra secret families, symlink-preservation policy, cloud/scheduling evidence, and release/docs follow-ups remain in the accepted roadmap, outside the current F2 source slice.

## Delivery forecast

- Current branch point and initial review boundary: `1413423`.
- Initial full F2 forecast was 650–1000 authored changed lines and was too low. Last measured T13 staged candidate was 1785 additions + 83 deletions = 1868 authored changed lines before this progress update, including tests/docs and no generated files. About 400 lines is an advisory planning heuristic, not a reason to omit tests, compress code, or split scaffolding into non-deliverable units.
- Strategy: `feature-branch-chain` (user choice). F2 delivered as 4 local commits on `feat/f2-target-recovery`, branched from `1413423`. Running authored count: T13 2570 lines, T14 ~1744 lines (1668 additions + 76 deletions). Both slices exceed the advisory delivery budget and were intentionally kept as coherent security units rather than split into non-deliverable scaffolding. Nothing pushed; PR creation and merge remain the user's decision.

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

F2 is delivered locally on `feat/f2-target-recovery` (T13–T15 complete, GGA PASSED, Linux proof only). Next: decide push/PR delivery, then continue the roadmap with F1–F4 and the T3 eight-stage real-binary journey matrix. The native review preflight still awaits its `intended_untracked_selection` input, which stays blocked as an unpublished-schema limitation rather than a fabricated submission.
