# AGENTS.md — bak-cli Code Review Rules

## Project Identity
- **Name**: bak-cli
- **Binary**: `bak`
- **Purpose**: Backup and restore OpenCode AI coding configurations across machines
- **Language**: Go 1.24+
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
- MUST use `path.Clean` + `filepath.ToSlash` for canonical path comparison
- MUST NOT use `filepath.Clean` for cross-platform canonical paths

### Cross-Platform
- MUST handle Windows (`\`), macOS (`/`), and Linux (`/`) path separators
- MUST use `path.Clean` (not `filepath.Clean`) for canonical path normalization
- MUST test path operations on all three OS representations
- MUST NOT assume case-sensitive filesystems
- SHOULD use `strings.EqualFold` or `strings.ToLower` for case-insensitive comparison

### CLI Patterns
- MUST use cobra for command structure
- MUST provide `--help` for every command
- MUST return exit code 0 on success, 1 on error
- SHOULD provide `--verbose` flag for debugging
- MUST provide `--dry-run` for any destructive operation (restore)

### Testing
- MUST achieve >80% coverage for new code
- MUST test happy path AND error paths
- MUST test edge cases: empty input, missing files, permission errors
- MUST use `t.TempDir()` for test isolation — never write to real filesystem
- SHOULD test cross-platform path behavior

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

### Performance
- SHOULD avoid unnecessary allocations in hot paths
- SHOULD use `strings.Builder` for string concatenation in loops
- MUST NOT block indefinitely — use context or timeouts for external calls

### Commits
- MUST follow Conventional Commits: `feat:`, `fix:`, `test:`, `chore:`, `docs:`
- MUST keep commits atomic — one logical change per commit
- MUST NOT include AI attribution (Co-Authored-By) in commit messages
