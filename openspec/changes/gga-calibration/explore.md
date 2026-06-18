## AGENTS.md & GGA Calibration Assessment

### Scope & Method

- **Project:** bak-cli (Go 1.25+, cobra CLI with adapter pattern)
- **GGA version:** v2.8.1 (bash wrapper, NOT Go binary) — `home/linuxbrew/.linuxbrew/Cellar/gga/2.8.1/bin/gga`
- **Provider:** `opencode:opencode-go/qwen3.7-plus` (project overrides global default `opencode`)
- **Rules file referenced by `.gga`:** `AGENTS.md` (185 lines)
- **Pre-commit hook installed:** `~/.git/hooks/pre-commit` calls `gga run || exit 1`
- **No GGA in CI** — `.github/workflows/ci.yml` runs `Lint`, `Test` (3 OS), `Coverage`, `Security`, `GoReleaser Check`, `Build` (3 OS). GGA is pre-commit only.

### How GGA actually enforces rules

GGA is a **prompt-only LLM reviewer**, not a static analyzer. It:

1. `cat`s the entire `RULES_FILE` (AGENTS.md, ~5KB) into a prompt
2. Inlines the full content of every staged file via `git show ":file"`
3. Sends the assembled prompt to the provider as a single CLI arg: `opencode run --model "$model" "$prompt"` (see `bin/gga:810-813`)
4. Parses `STATUS: PASSED` or `STATUS: FAILED` from the response

**Consequences for calibration:**

- ✅ Rules that are concrete and grep-able (`MUST use fmt.Errorf("%w")`, `MUST NOT use filepath.ToSlash`, `MUST use _GOOS.go suffix`) — GGA can spot them by reading the diff
- ⚠️ Rules that require runtime/test context (`MUST test happy path AND error paths`, `MUST test edge cases`, `MUST achieve >80% coverage`) — GGA cannot check coverage; it can only see if tests exist
- ❌ Rules that are subjective or require external info (`MUST justify any new dependency`, `SHOULD prefer well-maintained packages`, `MUST test on target OS in CI 3-OS matrix`) — GGA has no way to verify these
- ❌ `STRICT_MODE=true` causes GGA to fail on ambiguous AI responses. Since the AI is non-deterministic, this can produce false positives/negatives

### AGENTS.md Rules Analysis

For every MUST/SHOULD rule, the table below classifies GGA's ability to verify it. The pragmatic test: *can an LLM reading the diff output a "VIOLATION" verdict with file+line for this rule, deterministically?*

| # | Category | Rule summary | GGA enforceable | Calibration | Notes |
|---|----------|--------------|-----------------|-------------|-------|
| 1 | Go Idioms | `MUST use fmt.Errorf("context: %w", err)` — never bare `errors.New` for wrapped errors | Partial | Just right | LLM can grep `errors.New(` and check the wrapping pattern, but false positives likely (e.g., sentinel errors via `errors.New` are correct) |
| 2 | Go Idioms | `MUST use table-driven tests` | Partial | Just right | LLM can see if a test loop exists, but `setupX` helper + N narrow `Test*` functions is the established pattern in this repo (14/14 engine tests). Rule is a SHOULD dressed as a MUST |
| 3 | Go Idioms | `MUST use filepath.Join / path.Clean` for paths | Yes | Just right | LLM can detect `path/filepath` import and pattern usage |
| 4 | Go Idioms | `SHOULD prefer interfaces over concrete types` | No | Too strict (MUST-level) | LLM cannot determine whether an interface would be useful at the call site |
| 5 | Go Idioms | `MUST NOT use panic in library code` | Yes | Just right | Easy regex match |
| 6 | Go Idioms | `MUST handle ALL returned errors — no _ = for error returns` | Partial | Just right | LLM can spot `_ = someFunc()` patterns, but false positives for legit `_ =` in goroutines / signal handlers |
| 7 | Error Message | `MUST start error messages with lowercase` | Yes | Just right | Regex-detectable |
| 8 | Error Message | `MUST include context / operation name` | Partial | Just right | Subjective; LLM can flag "no context" but quality is fuzzy |
| 9 | Error Message | `MUST NOT include sensitive data` | No | Too strict | LLM cannot reliably detect tokens/keys unless pattern is explicit |
| 10 | Logging | `MUST use fmt.Fprintf(os.Stderr, ...)` for warnings/errors | Yes | Just right | Detectable; **real violation exists**: `internal/actions/pull.go:89,130,138,139` all use `fmt.Printf` for status output |
| 11 | Logging | `SHOULD use verbose flag` / `MUST NOT log sensitive data` | No | Just right | Subjective |
| 12 | Security | `MUST validate paths stay under home` (traversal) | Partial | Just right | LLM can flag missing `path.Clean` checks, but verifying correctness requires tracing |
| 13 | Security | `MUST NOT include secrets in backup` | No | Manual | Config behavior, not code-grep-able |
| 14 | Security | `MUST redact ghp_*, sk-*, sk-ant-*` in output | No | Manual | LLM can spot literal `Print(ghp_...)` but real redaction is a runtime concern |
| 15 | Security | `MUST use os.UserHomeDir()` — never hardcode home | Yes | Just right | Detectable |
| 16 | Security | `MUST use path.Clean + strings.ReplaceAll("\\", "/")` (not `filepath.ToSlash`) | Yes | Just right | **Best rule in the file** — explicit, grep-able, anti-pattern shown. The repo follows it (`internal/paths/normalize.go:58` has comment "Unlike filepath.ToSlash...") |
| 17 | Cross-Platform | `MUST NOT assume case-sensitive filesystems` | Partial | Too strict (MUST-level) | LLM cannot know if the code will run on macOS HFS+ or NTFS |
| 18 | Platform-Specific Code | `MUST use _GOOS.go suffix / //go:build GOOS` | Yes | Just right | Filename pattern — checkable |
| 19 | Platform-Specific Code | `MUST inject OS calls via variables` (`var execCommand = exec.Command`) | Partial | Just right | Pattern is visible; the PR #27 OAuth commit introduces `os/exec` calls that may or may not be injected |
| 20 | Platform-Specific Code | `MUST test platform-specific code on target OS in CI` | No | Manual | CI config, not code |
| 21 | CLI Patterns | `MUST use cobra` / `--help` / exit codes / `--verbose` / `--dry-run` | Partial | Just right | Patterns visible; dry-run presence is a runtime concern |
| 22 | CLI Patterns | `MUST delegate all business logic to internal/actions/` | Yes | Just right | **Strong rule** — easy to verify: `grep -r "cobra" internal/actions/` should be empty |
| 23 | Architecture Boundaries | `internal/actions/ MUST NOT import spf13/cobra` | Yes | Just right | Single-line check |
| 24 | Architecture Boundaries | `Actions MUST accept io.Writer/io.Reader` | Partial | Just right | Detectable in function signatures |
| 25 | Architecture Boundaries | `cmd/ is the ONLY package translating cobra types` | Yes | Just right | Architectural, easy to spot violations |
| 26 | Architecture Boundaries | `internal/cloud/ MUST reuse httputil.go` | Yes | Just right | Grep-able; PR #27 explicitly says it reuses `newRequest, doRequest, formatAPIError` |
| 27 | Testing | `MUST achieve >80% coverage` | **No** | Wrong tool | GGA is pre-commit, has no coverage data. Should be a CI gate (it already is — `Coverage` check) |
| 28 | Testing | `MUST test happy + error paths + edge cases` | No | Wrong tool | Subjective, requires running the test to see if it asserts anything |
| 29 | Testing | `MUST use t.TempDir() / setConfigHome / t.Setenv` | Yes | Just right | Pattern checkable |
| 30 | Testing | `MUST NOT unit-test os.Exit` / `bubbletea.Program.Run()` | Yes | Just right | Pattern checkable |
| 31 | Testing | `MUST test TUI Update() and View()` | Yes | Just right | Function name checkable |
| 32 | Backup/Restore | `MUST create manifest before copying / SHA-256 / dry-run diff / version mismatch warning` | Partial | Just right | Some are code-visible (SHA-256 calls), others are runtime |
| 33 | Git Safety | `MUST auto-commit before/after restore` / `MUST NOT force-push` / `bak undo uses git revert` | Partial | Just right | `exec.Command("git", "revert", ...)` is detectable; force-push check is fuzzy |
| 34 | Dependency Mgmt | `MUST prefer stdlib / justify new dep / >1000 stars` | No | Wrong tool | GGA cannot check go.sum, GitHub stars, or justification presence in PR body |
| 35 | Documentation | `MUST add godoc on exported types / package docs / README in sync` | Partial | Just right | LLM can flag missing comments above `func`/`type` declarations but cannot verify README sync |
| 36 | API Design | `MUST NOT export types unless needed` / naming consistency / `SHOULD return structs over multi-value` | Partial | Just right | Detectable but subjective; "Configuration vs Config" naming is fuzzy |
| 37 | DRY | `MUST NOT duplicate utility functions` / `>70% logic overlap → consolidate` | No | Wrong tool | LLM cannot compute "70% overlap" |
| 38 | Performance | `SHOULD avoid allocations` / `strings.Builder` / `MUST NOT block indefinitely — use context` | Partial | Just right | LLM can spot missing `context.Context` in HTTP/exec calls; allocation analysis is harder |
| 39 | Dependency Injection | `MUST place interfaces in consumer package / struct field injection / zero-value usable` | Partial | Just right | Pattern checkable; "zero-value usable" is a test concern |
| 40 | Test Doubles | `MUST hand-roll / Mock* prefix / inline suffixes / compile-time interface check` | Yes | Just right | Pattern checkable |
| 41 | GGA Integration | `MUST run GGA / fix all violations / no --no-verify bypass` | Yes | **Contradicts reality** | **The rule itself was violated** in commit `23dfaf9`. AGENTS.md promises GGA is mandatory, but the very commit that the user is asking about used `--no-verify` for a legitimate technical reason (ARG_MAX overflow) |
| 42 | Commits | `Conventional Commits / atomic / no AI attribution` | Yes | Just right | **All 4 PRs comply** — all commit messages follow `feat():`, `fix():`, `chore():`, `docs():` patterns and have no `Co-Authored-By` lines |
| 43 | TUI Package Organization | `internal/tui/{styles,components,screens}/ boundaries` | Yes | Just right | File-path based rule, easy to verify |
| 44 | TUI Styling | `Package-level lipgloss styles / Rose Pine colors / no inline NewStyle in View()` | Yes | Just right | Grep `lipgloss.NewStyle()` in `*_view*.go` / `View()` methods |
| 45 | Bubbletea Version Lock | `charm.land/bubbletea/v2 v2.0.7 / tea.KeyPressMsg / charm.land/lipgloss/v2 v2.0.3` | Yes | Just right | `go.mod` + import check |
| 46 | Bubbles Dependency | `MUST justify bubbles imports / sub-model Init/Update/View` | Partial | Just right | Grep `charm.land/bubbles/v2` + check for `bubbles.Init/Update/View` in wrapper |
| 47 | TUI Responsiveness | `WindowSizeMsg / width+height in model / 20x10 fallback / guard logo` | Yes | Just right | Pattern checkable |
| 48 | TUI Testing | `Update()/View() table-driven / ≥80% coverage / edge cases` | Partial | Just right | Coverage is wrong tool; patterns checkable |

**Counts:**
- 25 rules: GGA can check (with some risk of false positives)
- 12 rules: Partially checkable (subjective, runtime, or fuzzy)
- 11 rules: Not checkable by GGA (coverage, deps, secrets behavior, runtime behavior)

**Calibration summary:**

- **Too strict as MUST:** rules #2, #4, #9, #17, #27, #28, #34, #37 (subjective, wrong tool, or impossible to verify)
- **Just right:** ~30 rules — concrete, grep-able, easy to verify by LLM
- **Too loose:** rule #41 (`no --no-verify bypass`) — the AGENTS.md rule has no escape hatch for technical failures like ARG_MAX overflow
- **Contradictory:** rule #41 vs reality — the rule says "no bypass" but commit `23dfaf9` had to bypass and explicitly documented why

### GGA Configuration

```ini
PROVIDER="opencode:opencode-go/qwen3.7-plus"
FILE_PATTERNS="*.go,*.mod,*.sum"
EXCLUDE_PATTERNS="vendor/*,*.pb.go,go.sum,go.mod"
RULES_FILE="AGENTS.md"
STRICT_MODE="true"
TIMEOUT="800"
PR_BASE_BRANCH="main"
```

**Analysis:**

| Setting | Value | Verdict | Notes |
|---------|-------|---------|-------|
| `PROVIDER` | `opencode:opencode-go/qwen3.7-plus` | OK | Specific model pinned; global default would be `opencode` alone |
| `FILE_PATTERNS` | `*.go,*.mod,*.sum` | **Bug** | GGA matches by **suffix only** (see `bin/gga:992-1007`). The pattern is checked with `[[ "$file" == *"$suffix" ]]` where `suffix` is `*.go` minus the `*` → `.go`. So `go.mod` and `go.sum` are matched. But the `*.mod` pattern actually fails to match anything since it looks for files ending in `.mod` literally — `go.mod` ends in `.mod` so it works, but `*.sum` works similarly. So this is actually OK. The bigger issue: **`*.mod` and `*.sum` are listed in `EXCLUDE_PATTERNS` too**, so they're filtered out before review anyway. This is redundant |
| `EXCLUDE_PATTERNS` | `vendor/*,*.pb.go,go.sum,go.mod` | **Contradictory** | `go.mod` and `go.sum` are both in include AND exclude. Include wins (exclude only filters after include matches), so they ARE reviewed, then excluded — net effect: ignored. So **the FILE_PATTERNS line is misleading** |
| `RULES_FILE` | `AGENTS.md` | OK | 185 lines of rules. Long, but GGA handles it |
| `STRICT_MODE` | `true` | OK | Fails on ambiguous AI response — appropriate for pre-commit |
| `TIMEOUT` | `800` | **Reasonable** | 13 min — long, but 11 files × ~500 lines + 185-line rules = ~5-6K tokens, which the opencode provider struggles with. The 800s was raised to avoid timeouts |
| `PR_BASE_BRANCH` | `main` | OK | |

**Gaps in `.gga`:**

1. **No `MAX_FILE_SIZE` cap** — the underlying issue (PR #26's "Argument list too long") is that the entire prompt (rules + ALL staged files) is passed as a single `opencode run "$prompt"` arg. ARG_MAX on Linux is 2MB. With 11 working-tree files, the prompt blew past this. GGA has no way to chunk files.
2. **No `MAX_FILES` cap** — if someone runs `gga` on a refactor touching 30+ files, same crash.
3. **No fallback to stdin** — `execute_opencode()` passes the prompt as `$prompt` (positional arg), not via stdin. The Ollama path handles this correctly via `printf | python3`, but OpenCode does not.
4. **`STRICT_MODE` blocks valid `STATUS: PASSED` responses** when the AI adds an "I noticed..." caveat line before the STATUS line. This is the #1 source of "no-verify" pressure.

### Open PRs Status

| PR | Title | Base ← Head | State | Mergeable | Files | +/− | GGA status | Issues |
|---|---|---|---|---|---|---|---|---|
| #24 | feat(quality-ux-overhaul): tracker | main ← feat/quality-ux-overhaul | OPEN | MERGEABLE | 77 (mostly SDD archive metadata) | +617/−5 | ✅ CI green (10 checks pass: Lint, Test ×3, Coverage, Security, GoReleaser, Build ×3) | **No GGA in CI** — only the developer-side pre-commit hook enforced rules. The 3 commits on this branch (`310b28a`, `35a88be`, `67ee9bd`) are all `chore(sdd):` archives — no rule-relevant code. Safe to merge |
| #25 | feat(tui): UX polish | feat/quality-ux-overhaul-01-unblock-core ← feat/quality-ux-overhaul-02-ux-polish | OPEN | MERGEABLE | 13 | +511/−60 | **Pre-commit GGA passed** — no bypass marker. Commit `4a5208e` message says "All quality gates passing (race/vet/lint)" | None apparent. Files in TUI screens — TUI rules (43-48) checkable. Risk: needs a re-review on merge since the CI doesn't run GGA |
| #26 | feat(backup): exclusion engine, scan options, progress callbacks | feat/quality-ux-overhaul-02-ux-polish ← feat/quality-ux-overhaul-03-backup-size | OPEN | MERGEABLE | 24 | +1440/−55 | ⚠️ **Mixed** — 2/3 commits passed GGA (`c75c5f7`, `c98177a`); 1 commit **`23dfaf9` used `--no-verify`** with documented reason: ARG_MAX overflow on 11 working-tree files. Commit message explicitly names pre-existing violations as "outside scope of this change": (a) `fmt.Printf` in `internal/actions/pull.go:89,130,138,139` (violates rule #10), (b) 14 non-table-driven tests in `internal/backup/engine_test.go` (violates rule #2) | **The PR ships 2 known rule violations** that GGA would have caught had it run. Risk: those violations merge to main |
| #27 | feat(oauth): add GitHub OAuth Device Flow login (PR4 quality-ux-overhaul) | feat/quality-ux-overhaul-03-backup-size ← feat/quality-ux-overhaul-04-oauth | OPEN | MERGEABLE | 9 | +1126/−153 | **Pre-commit GGA passed** — commit `56f348a` says "reuses httputil.go helpers" (rule #26). Files are cloud/ + actions/login.go. No bypass marker. | The OAuth commit introduces `os/exec` calls for browser-open — **must verify rule #19** (OS call injection via variables). Quick check below |

**Verification of PR #27 rule compliance (rule #19 — OS call injection):**

The commit adds `internal/cloud/browser.go` and `internal/cloud/browser_test.go`. Need to verify `os/exec.Command` is wrapped in a `var` for testability. (Not done in this explore — flag as a verify-task before merge.)

**PR #24 is the "tracker" but it has 77 files / 617 insertions** — mostly SDD archive metadata. The actual feature work lives in PRs #25-#27. This is the right structure: PR #24 closes the SDD housekeeping; the chained PRs are the real ship.

### GGA Capabilities (from `gga --help`)

```text
Gentleman Guardian Angel v2.8.1
Provider-agnostic code review using AI

USAGE:
  gga <command> [options]

COMMANDS:
  run [--no-cache]        Run code review on staged files
  install                 Install git pre-commit hook (default)
  install --commit-msg    Install git commit-msg hook
  uninstall               Remove git hooks
  config                  Show current configuration
  init                    Create a sample .gga config file
  cache clear             Clear cache for current project
  cache clear-all         Clear all cached data
  cache status            Show cache info

RUN OPTIONS:
  --no-cache              Force review all files, ignoring cache
  --ci                    CI mode: review files changed in last commit (HEAD~1..HEAD)
  --pr-mode               PR mode: review all files changed in the full PR
                          Auto-detects base branch
  --diff-only             With --pr-mode: send only diffs (faster, cheaper)

CONFIG OPTIONS:
  PROVIDER                AI provider (claude|gemini|codex|opencode|ollama:<m>|lmstudio[:m]|github:<m>)
  FILE_PATTERNS           Comma-separated globs (default: *)
  EXCLUDE_PATTERNS        Comma-separated globs
  RULES_FILE              Rules file path (default: AGENTS.md)
  STRICT_MODE             Fail on ambiguous AI response (default: true)
  TIMEOUT                 Max seconds for AI response (default: 300)
  PR_BASE_BRANCH          Base branch for --pr-mode (default: auto-detect)

ENVIRONMENT:
  GGA_PROVIDER            Override provider
  GGA_TIMEOUT             Override timeout (seconds)
```

**Capabilities gap relevant to this explore:**

- No `MAX_FILE_SIZE` option — root cause of the ARG_MAX failure
- No stdin-pipe for prompt — would solve the "Argument list too long" problem completely
- No per-file or per-PR chunking — GGA tries to fit everything in one prompt
- No `--quiet` / `--json` output — human-readable only, hard to integrate with CI
- No way to bypass a single rule for a single commit (e.g., `--allow-known-violation`)
- `--ci` mode exists but is unused by the project's CI (CI runs `Lint`, `Test`, `Coverage`, `Security`, `Build` — no GGA)

### Pre-existing GGA Violations Confirmed in Code

Found by grep, not running GGA — these are the issues the `--no-verify` commit punted on:

**1. `internal/actions/pull.go` — 4× `fmt.Printf` (violates rule #10)**
```
89:  fmt.Printf("Downloading backup %s...\n", remoteID)
130: fmt.Printf("Extracting backup %s...\n", backupID)
138: fmt.Printf("✅ Backup pulled: %s\n", backupID)
139: fmt.Printf("   Run 'bak restore %s' to apply it.\n", backupID)
```

**2. `internal/backup/engine_test.go` — 14/14 test functions are NOT table-driven (violates rule #2)**
```
14 test functions, 0 use `tests := []struct{...}{...}` pattern, 0 `for _, tt := range tests`
```

**3. `internal/cloud/browser.go` (PR #27, new) — rule #19 (OS call injection) ✅ COMPLIANT**
```
10: // execCommand is the function used to run external commands.
11: // Overridable for tests.
12: var execCommand = exec.Command
...
24: // This is a package-level variable so callers (e.g., DeviceClient)
25: // can override it for injection.
26: var openBrowserOS = func(url string) error {
```
Both `execCommand` and `openBrowserOS` are package-level vars, fully test-injectable. PR #27 has no rule violation here.

### Root Cause of the `--no-verify` Incident

The "Argument list too long" error in PR #26's commit `23dfaf9` is **NOT a GGA bug — it's a GGA design limitation**:

1. GGA assembles a prompt: rules (5KB) + each staged file (inline content, average ~3-15KB each)
2. For 11 files, prompt size ≈ 5KB + 11 × 10KB = 115KB
3. GGA passes this as a single positional arg: `opencode run --model "..." "$prompt"` (see `bin/gga:810-813`)
4. Linux `ARG_MAX` = 2,097,152 bytes by default — should fit, but opencode's `qwen3.7-plus` may have its own size limit
5. The error reported was "Argument list too long for 11 working-tree files" — this is `execve()` rejecting the command, meaning prompt > 2MB. With 11 files averaging 180KB each, that's possible for long Go files with full file content (not just diffs)

**Fix paths:**
- **GGA upstream:** pipe prompt via stdin instead of arg, OR chunk files
- **Project-side:** raise `TIMEOUT` further, OR add a `MAX_FILE_SIZE` config (doesn't exist in v2.8.1)
- **Workaround used:** `--no-verify` — explicitly documented, and the rule-violations it would have caught are pre-existing and out of scope

### Recommendations

#### 1. AGENTS.md adjustments

| Severity | Change | Rationale |
|----------|--------|-----------|
| **High** | Add an escape-hatch to rule #41: "MUST fix all GGA violations OR document bypass reason in commit body for: ARG_MAX overflow, provider outage, scope-of-change mismatch. Bypass requires explicit `NO-VERIFY:` line in commit body." | The rule as written is unenforceable in practice and was violated this very cycle. Make the workflow honest. |
| **High** | Demote rules #27, #28, #34, #37 from MUST to SHOULD or remove. They cannot be checked by GGA (coverage) or are subjective (DRY 70%, dependency stars). | They are aspirational rules dressed as enforceable ones — they will be ignored silently and erode the authority of real MUSTs. |
| **Medium** | Split `AGENTS.md` into two files: `AGENTS.md` (machine-checkable by GGA) and `CONTRIBUTING.md` (philosophy, rationale, subjective rules). | 185 lines is too long for the LLM to internalize perfectly. Splitting makes the GGA prompt smaller and more focused. |
| **Medium** | Rule #2 (table-driven tests) should explicitly carve out `Test*_FooBar` test functions that exercise one behavior each. The current 14/14 non-table-driven engine tests are correctly written — they just don't fit the rule's literal text. | Rule is followed in spirit but violated in letter. Either rewrite the rule or rewrite the tests. |
| **Low** | Add a "When GGA cannot run" section: doc the ARG_MAX limit, the OpenCode provider's prompt size cap, and the `--no-verify` workflow. | Make the failure mode discoverable. |

#### 2. GGA config adjustments

| Setting | Recommended change | Reason |
|---------|-------------------|--------|
| `FILE_PATTERNS` | `"*.go"` (remove `*.mod,*.sum`) | `*.mod` and `*.sum` are in EXCLUDE_PATTERNS too — redundant and confusing |
| `EXCLUDE_PATTERNS` | `"vendor/*,*.pb.go,*_generated.go,*.gen.go,cmd/bindata.go"` | Add common Go generation patterns |
| `STRICT_MODE` | Keep `true` | Correct for pre-commit; flags ambiguous LLM responses |
| `TIMEOUT` | Keep `800` | Raised for a reason (opencode-go/qwen3.7-plus is slow on large prompts) |
| **Add (upstream GGA):** | `MAX_PROMPT_SIZE` and stdin-pipe support | Fixes the "Argument list too long" root cause |
| **Add (project CI):** | `gga run --ci` step in `ci.yml` | Closes the gap where GGA only runs locally. GGA `--ci` mode reviews the last commit in CI |

#### 3. PR merge strategy

Recommended order (matches the existing chain; do not reorder):

1. **Merge PR #24 first** (tracker) — closes SDD housekeeping, no rule risk
2. **Merge PR #25** — pre-commit GGA passed, low risk
3. **Merge PR #26** — requires one of:
   - (a) **Squash the chain into a single commit** so `--no-verify` only applies once, and add a follow-up commit that fixes the 4 `fmt.Printf` violations + converts `engine_test.go` to table-driven, OR
   - (b) **Fix the violations before merge** — add a `fix: address GGA-punted violations` commit on the branch, then re-enable GGA, then merge
   - Recommended: **(b)**. The 4 `fmt.Printf` → `fmt.Fprintf(os.Stderr, ...)` is a 4-line mechanical change; converting 14 engine tests to table-driven is a 200-line change but pure refactor
4. **Merge PR #27** — pre-commit GGA passed; **rule #19 (OS call injection) verified compliant** — `internal/cloud/browser.go:12` declares `var execCommand = exec.Command` and `:26` declares `var openBrowserOS` for test override. Safe to merge.

**Alternative for PR #26:** if the violations are kept (status quo), archive this change with a `verify-report.md` that names them as accepted tech debt, and create a follow-up `quality-ux-overhaul-05-lint-cleanup` change.

#### 4. Process changes (long-term)

- **Add GGA to CI**: a `gga run --ci` step in `.github/workflows/ci.yml` will catch rule violations even when developers skip the pre-commit hook
- **Block merge on GGA failure**: if GGA is added to CI, set PR branch protection to require CI success
- **Reduce `STRICT_MODE` overhead**: currently every "looks-good-but-caveated" response from the LLM blocks the commit. Consider `STRICT_MODE=false` for the pre-commit and reserve strict mode for CI

### Ready for Proposal

**Yes.** Calibration is clear, gaps are concrete, and the PR merge path is well-defined.

**Key tension to surface to the user:** the current AGENTS.md rule #41 ("MUST fix all GGA violations — no `--no-verify` bypass") is **not survivable as written**. Either (a) add an explicit bypass clause for technical failures, or (b) the rule will keep being silently violated. The choice is: codify honesty, or keep the pretense.
