# Delta for pull-fix

## ADDED Requirements

### REQ-PULL-001: pull.go MUST use fmt.Fprintf with injectable writer
**Priority**: must

All output in `internal/actions/pull.go` MUST use `fmt.Fprintf` with an injectable `io.Writer` instead of `fmt.Printf`.

**Scenario**: Status output during pull
- GIVEN `internal/actions/pull.go` lines 89, 130, 138, 139
- WHEN status messages are emitted to the user
- THEN `fmt.Fprintf(w, ...)` MUST be used where `w` is an injectable `io.Writer`
- AND zero `fmt.Printf` calls MUST remain in the file

**Acceptance criteria**:
- [ ] `grep -n "fmt.Printf" internal/actions/pull.go` returns zero matches
- [ ] All four lines (89, 130, 138, 139) converted to `fmt.Fprintf`
- [ ] Writer parameter is injectable (function parameter or struct field)

---

### REQ-PULL-002: All output MUST go through deps.Stdout/deps.Stderr
**Priority**: must

The Pull action MUST route all output through injected dependencies, consistent with the `CmdDeps` pattern.

**Scenario**: Informational output
- GIVEN a `Pull` action with injected `deps`
- WHEN download/extract/success messages are emitted
- THEN output MUST go to `deps.Stdout`

**Scenario**: Warning and error output
- GIVEN a `Pull` action with injected `deps`
- WHEN warnings or errors are emitted
- THEN output MUST go to `deps.Stderr`

**Acceptance criteria**:
- [ ] Pull function signature accepts `Stdout io.Writer` and `Stderr io.Writer` (or uses `deps` struct)
- [ ] Tests verify output via `bytes.Buffer` injected as writer
- [ ] No direct `os.Stdout` or `os.Stderr` references in `pull.go`
