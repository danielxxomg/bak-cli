# AGENTS.md — bak-cli Code Review Rules

## Project Identity
- **Name**: bak-cli
- **Binary**: `bak`
- **Purpose**: Backup and restore OpenCode AI coding configurations across machines
- **Language**: Go 1.25+
- **Architecture**: CLI (cobra) with adapter pattern

## Code Review Rules

### Go Idioms
- MUST use `fmt.Errorf("context: %w", err)` for error wrapping — never bare `errors.New` for wrapped errors
- MUST use table-driven tests (`[]struct{ name string; ... }`) for unit tests
- MUST use `filepath.Join` for OS-specific paths, `path.Clean` for canonical paths
- SHOULD prefer interfaces over concrete types for testability
- MUST NOT use `panic` in library code — return errors
- MUST handle ALL returned errors — no `_ =` for error returns

### Error Message Formatting
- MUST start error messages with lowercase (e.g., `"backup dir: %w"`, not `"Backup dir: %w"`)
- MUST include context in error messages (what was being done when it failed)
- MUST NOT include sensitive data (tokens, paths with usernames) in error messages
- SHOULD include the operation that failed (e.g., `"read config: %w"`, `"create target: %w"`)

### Logging Standards
- MUST use `fmt.Fprintf(os.Stderr, ...)` for warnings and errors visible to users
- SHOULD use `verbose` flag to gate debug/diagnostic output
- MUST NOT log sensitive data (tokens, API keys, passwords)
- SHOULD prefix verbose messages with context (e.g., `"warning: hostname: %v"`)
- MUST NOT use `fmt.Println` for error output — use stderr

### Security
- MUST validate all paths stay under user home directory (path traversal prevention)
- MUST NOT include secrets/API keys/tokens in backup by default
- MUST redact sensitive patterns (ghp_*, sk-*, sk-ant-*) in any output
- MUST use `os.UserHomeDir()` — never hardcode home paths
- MUST use `path.Clean` + `strings.ReplaceAll(path, "\\", "/")` for canonical path comparison — NOT `filepath.ToSlash` (OS-dependent, fails on Linux)
  - ❌ Violation: `path.Clean(filepath.ToSlash(x))`
  - ✅ Correct: `path.Clean(strings.ReplaceAll(x, "\\", "/"))`
- MUST NOT use `filepath.Clean` for cross-platform canonical paths

### Cross-Platform
- MUST handle Windows (`\`), macOS (`/`), and Linux (`/`) path separators
- MUST use `path.Clean` (not `filepath.Clean`) for canonical path normalization
- MUST test path operations on all three OS representations
- MUST NOT assume case-sensitive filesystems
- SHOULD use `strings.EqualFold` or `strings.ToLower` for case-insensitive comparison

### Platform-Specific Code
- MUST use `_GOOS.go` suffix for OS-specific files (e.g., `scheduler_windows.go`)
- MUST use `//go:build GOOS` tags for platform-restricted code
- MUST inject OS calls via variables for testability (e.g., `var execCommand = exec.Command`, `var isAdminFn = isAdmin`)
- MUST test platform-specific code on the target OS in CI (3-OS matrix)

### CLI Patterns
- MUST use cobra for command structure
- MUST provide `--help` for every command
- MUST return exit code 0 on success, 1 on error
- SHOULD provide `--verbose` flag for debugging
- MUST provide `--dry-run` for any destructive operation (restore)
- MUST delegate all business logic from cobra `RunE` to `internal/actions/` — no logic in `cmd/`

### Architecture Boundaries
- `internal/actions/` MUST NOT import `github.com/spf13/cobra` — cobra is a `cmd/` concern only
- Actions MUST accept `io.Writer`/`io.Reader` and plain parameters, not framework types
- `cmd/` is the ONLY package that translates cobra types to action parameters
- `internal/cloud/` MUST reuse existing helpers (`httputil.go`) — MUST NOT reimplement HTTP request/response logic

### Testing
- MUST achieve >80% coverage for new code
- MUST test happy path AND error paths
- MUST test edge cases: empty input, missing files, permission errors
- MUST use `t.TempDir()` for test isolation — never write to real filesystem
- MUST isolate config from real user config via dependency injection or `t.Setenv` — tests MUST NOT depend on `~/.config/bak/` existing
- MUST skip Windows-specific test cases on non-Windows via `runtime.GOOS` check
- MUST use `setConfigHome(t, dir)` helper for config isolation — handles `XDG_CONFIG_HOME` (Linux), `APPDATA` (Windows), `HOME`+Library (macOS) automatically
- MUST NOT assume `os.UserConfigDir()` respects `XDG_CONFIG_HOME` on macOS — it always returns `$HOME/Library/Application Support`
- SHOULD test cross-platform path behavior
- MUST maintain per-package coverage ≥80% for `internal/` packages
- MUST NOT unit-test `os.Exit` paths — test via integration/E2E only (E2E tests in `tests/e2e/` cover cmd/ entry points)
- MUST NOT test `bubbletea.Program.Run()` directly — test model `Update()`/`View()` logic instead (these are pure functions, easy to unit-test)
- MUST test TUI model `Update()` and `View()` methods — they contain business logic that deserves coverage

### Backup/Restore Specifics
- MUST create manifest before copying files (fail-fast on invalid state)
- MUST compute SHA-256 checksums for all backed-up files
- MUST verify checksums on restore
- MUST NOT restore without mandatory dry-run diff
- MUST warn on version mismatch between backup and installed tools

### Git Safety
- MUST auto-commit before and after restore operations
- MUST NOT force-push or rewrite history
- `bak undo` MUST use `git revert` (safe, non-destructive)

### Dependency Management
- MUST prefer Go standard library over third-party packages
- MUST justify any new dependency (why stdlib is insufficient)
- MUST NOT add dependencies for trivial functionality
- SHOULD prefer well-maintained packages (>1000 stars, active commits)

### Documentation
- MUST add godoc comments on all exported types and functions
- MUST include package-level documentation in at least one file per package
- SHOULD include usage examples in godoc comments for complex functions
- MUST keep README.md in sync with CLI commands and flags

### API Design
- MUST NOT export types unless they need to be used outside the package
- SHOULD prefer concrete types for function parameters, interfaces for struct fields
- MUST use consistent naming: `Config` not `Configuration`, `Run` not `Execute`
- SHOULD return structs instead of multiple return values for complex results

### DRY (Don't Repeat Yourself)
- MUST NOT duplicate utility functions across packages — extract to a shared package (e.g., `adapters/util.go`, `cloud/httputil.go`)
- MUST use existing helpers instead of reimplementing logic (e.g., use `httputil.go` for HTTP, `adapters/util.go` for file ops)
- SHOULD extract common patterns into base implementations (e.g., generic adapter struct with configurable constants)
- If two functions share >70% of their logic, they MUST be consolidated into a parameterized implementation

### Performance
- SHOULD avoid unnecessary allocations in hot paths
- SHOULD use `strings.Builder` for string concatenation in loops
- MUST NOT block indefinitely — use context or timeouts for external calls

### Dependency Injection
- MUST place interfaces in the consumer package (e.g., `actions/interfaces.go` defines `FileSystem`, `ConfigLoader`)
- MUST use struct field injection — no constructor functions for internal packages
- MUST make zero-value structs usable when possible (default behavior without explicit init)
- SHOULD accept interfaces, return structs (Go proverb)

### Test Doubles
- MUST hand-roll test doubles — no mock generation tools
- MUST use `Mock*` prefix for reusable test doubles (e.g., `MockFileSystem`)
- MUST use descriptive suffixes for inline fakes (e.g., `homeFS`, `mkdirFailingFS`, `writeFailingFS`)
- MUST verify interface compliance at compile time (e.g., `var _ FileSystem = (*MockFileSystem)(nil)`)
- SHOULD use `t.Helper()` in shared test setup functions

### GGA Integration
- MUST run GGA (Guardian Angel) as pre-commit validation against AGENTS.md rules
- MUST fix all GGA violations before committing — no `--no-verify` bypass
- Config in `.gga` — file patterns, exclude patterns, rules file, strict mode

### Commits
- MUST follow Conventional Commits: `feat:`, `fix:`, `test:`, `chore:`, `docs:`
- MUST keep commits atomic — one logical change per commit
- MUST NOT include AI attribution (Co-Authored-By) in commit messages
