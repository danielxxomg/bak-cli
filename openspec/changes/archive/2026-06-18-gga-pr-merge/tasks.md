# Tasks: GGA PR Merge

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~230 (fixes ~200 + CI ~30) |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | single-pr |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Low

## Phase 1: Fix pull.go Wiring (WS2a — TDD)

- [x] 1.1 **RED** Write `internal/actions/pull_test.go` — test that `PullAction.Stdout`/`Stderr` receive output via `bytes.Buffer`; assert zero `fmt.Printf` calls remain
- [x] 1.2 **GREEN** Add `Stdout io.Writer` and `Stderr io.Writer` fields to `PullAction` struct in `internal/actions/pull.go`; add nil-fallback to `os.Stdout`/`os.Stderr`
- [x] 1.3 **GREEN** Replace `fmt.Printf` at L89/130/138/139 with `fmt.Fprintf(out, ...)`; replace `fmt.Fprintf(os.Stderr, ...)` at L73/118 with `fmt.Fprintf(errOut, ...)`
- [x] 1.4 **REFACTOR** Run `go test -race ./internal/actions/...` — all pass; `grep "fmt.Printf" internal/actions/pull.go` returns 0 matches

## Phase 2: Convert engine_test.go to Table-Driven (WS2b — TDD)

- [x] 2.1 **RED** Read `internal/backup/engine_test.go` — catalog all 14 test funcs and their assertions
- [x] 2.2 **GREEN** Rewrite `internal/backup/engine_test.go` — group related tests into table-driven funcs using `tests := []struct{ name string; ... }` with `t.Run(tt.name, ...)` subtests
- [x] 2.3 **REFACTOR** Run `go test -race -cover ./internal/backup/...` — coverage >= pre-refactor baseline; all 14 original assertions preserved

## Phase 3: Calibrate AGENTS.md Rule #41 (WS1)

- [x] 3.1 Edit `AGENTS.md` rule #41 — add bypass clause: "If GGA fails due to technical limitation (ARG_MAX, provider outage, scope mismatch), developer MAY use `--no-verify` WITH: (a) `NO-VERIFY: <reason>` in commit body, (b) follow-up fix commit in same PR"
- [x] 3.2 Update `openspec/specs/bak-cli/spec.md` GGA scenario (L24-28) — verify bypass path matches rule #41 text

## Phase 4: Add GGA to CI (WS3 — TDD)

- [x] 4.1 **RED** Write `.github/workflows/gga.yml` — `gga-review` job: triggers on `pull_request`, runs `gga run --pr-mode --diff-only`, `continue-on-error: true`, env `OPENCODE_API_KEY: ${{ secrets.OPENCODE_API_KEY }}`, same Go version as CI
- [x] 4.2 **GREEN** Verify workflow YAML is valid — `actionlint .github/workflows/gga.yml` or manual review

## Phase 5: Commit, Push & Merge Chain (WS4)

- [x] 5.1 Commit Phase 1+2 fixes to `fix/wiring-fixes` branch — `git checkout fix/wiring-fixes && git add -A && git commit`
- [x] 5.2 Push `fix/wiring-fixes` — verify PR #26 CI passes with fixes
- [x] 5.3 Merge #27 into #26 — `gh pr merge 27 --merge`; verify GGA passes
- [x] 5.4 Merge #26 into #25 — `gh pr merge 26 --merge`; verify GGA passes
- [x] 5.5 Merge #25 into #24 — `gh pr merge 25 --merge`; verify GGA passes
- [x] 5.6 Merge #24 into main — `gh pr merge 24 --merge`; verify GGA passes

## Phase 6: Archive & Quality Gates

- [x] 6.1 `git mv openspec/changes/quality-ux-overhaul/ openspec/changes/archive/2026-06-18-quality-ux-overhaul/`
- [x] 6.2 Commit archive move and push
- [x] 6.3 `go test -race ./...` — all pass
- [x] 6.4 `go vet ./...` — clean
- [x] 6.5 `golangci-lint run` — exit 0
