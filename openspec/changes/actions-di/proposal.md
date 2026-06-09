# Proposal: actions-di (Dependency Injection for Actions Package)

## Intent

Remove the `github.com/spf13/cobra` dependency from `internal/actions/` to enforce the architecture boundary defined in AGENTS.md: actions must accept `io.Writer`/`io.Reader` and plain parameters, not framework types. Currently 3 action files import cobra and accept `*cobra.Command` in their `Run` signatures, violating this rule.

## Scope

### In Scope
- Refactor `BackupAction.Run(cmd *cobra.Command, args []string)` → `Run()` (args unused)
- Refactor `PushAction.Run(cmd *cobra.Command, args []string)` → `Run(args []string)` (args used for backup ID)
- Refactor `RestoreAction.Run(cmd *cobra.Command, args []string)` → `Run()` (args unused, backupID from struct field)
- Add `Stdout io.Writer` and `Stderr io.Writer` fields to all 3 action structs
- Replace `fmt.Printf`/`fmt.Fprintf(os.Stderr, ...)` with `fmt.Fprintf(a.Stdout, ...)`/`fmt.Fprintf(a.Stderr, ...)`
- Replace `cmd.OutOrStdout()`/`cmd.ErrOrStderr()` with injected writers
- Update `cmd/backup.go`, `cmd/push.go`, `cmd/restore.go` to pass `deps.Stdout`/`deps.Stderr`
- Remove `github.com/spf13/cobra` imports from all 3 action files
- Update tests to inject `bytes.Buffer` or `io.Discard` for writers

### Out of Scope
- Refactoring other actions that don't import cobra (already compliant)
- Changing the `cmdDeps` pattern in `cmd/deps.go` (already correct)
- Modifying flag parsing logic (remains in `cmd/`)
- Adding new features or changing behavior

## Capabilities

### New Capabilities
- `actions-io-injection`: Injectable stdout/stderr writers for action output, enabling testability and framework-agnostic action execution

### Modified Capabilities
None (pure internal refactor, no spec-level behavior changes)

## Approach

Extend the existing `cmdDeps` pattern from `cmd-di-refactor`:

1. **Struct field injection** (per AGENTS.md DI rules): Add `Stdout io.Writer` and `Stderr io.Writer` fields to `BackupAction`, `PushAction`, `RestoreAction`
2. **Zero-value defaults**: When writers are nil, fall back to `os.Stdout`/`os.Stderr` (matches existing `Stdin` pattern in `RestoreAction`)
3. **Signature change**: Remove `cmd *cobra.Command` parameter from `Run` methods. Keep `args []string` only where used (`PushAction`)
4. **cmd/ translation layer**: `cmd/backup.go`, `cmd/push.go`, `cmd/restore.go` pass `deps.Stdout`/`deps.Stderr` when constructing actions
5. **Output replacement**: Replace all `fmt.Printf` → `fmt.Fprintf(a.Stdout, ...)`, `fmt.Fprintf(os.Stderr, ...)` → `fmt.Fprintf(a.Stderr, ...)`

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/actions/backup.go` | Modified | Remove cobra import, add Stdout/Stderr fields, change Run signature, replace fmt calls |
| `internal/actions/push.go` | Modified | Remove cobra import, add Stdout/Stderr fields, change Run signature, replace fmt calls |
| `internal/actions/restore.go` | Modified | Remove cobra import, add Stdout/Stderr fields, change Run signature, replace cmd.OutOrStdout() |
| `internal/actions/backup_test.go` | Modified | Inject test writers (bytes.Buffer), update Run() calls |
| `internal/actions/push_test.go` | Modified | Inject test writers, update Run(args) calls |
| `internal/actions/restore_test.go` | Modified | Inject test writers, update Run() calls |
| `cmd/backup.go` | Modified | Pass deps.Stdout/deps.Stderr to BackupAction |
| `cmd/push.go` | Modified | Pass deps.Stdout/deps.Stderr to PushAction |
| `cmd/restore.go` | Modified | Pass deps.Stdout/deps.Stderr to RestoreAction |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Tests break due to missing writers | Low | Add nil-check fallback to os.Stdout/os.Stderr; update all tests to inject buffers |
| Output format changes accidentally | Low | Keep fmt.Fprintf calls identical, only change writer target |
| cmd/ callers forget to pass writers | Low | Compile-time enforcement: struct fields are io.Writer, not optional |

## Rollback Plan

Revert the commit. The change is purely internal refactoring with no behavior changes, so rollback is safe and complete.

## Dependencies

- Existing `cmdDeps` pattern from `cmd-di-refactor` SDD cycle (already merged)
- `io.Writer` interface from Go stdlib

## Success Criteria

- [ ] `internal/actions/backup.go`, `push.go`, `restore.go` do NOT import `github.com/spf13/cobra`
- [ ] All 3 actions accept `Stdout io.Writer` and `Stderr io.Writer` struct fields
- [ ] `Run` signatures no longer include `*cobra.Command` parameter
- [ ] All tests pass with injected writers
- [ ] `go build ./...` succeeds
- [ ] `gga` (Guardian Angel) passes AGENTS.md architecture boundary checks
