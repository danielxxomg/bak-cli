# Spec: gga-bypass

## Purpose

Documented escape hatch for GGA pre-commit when technical failures prevent review. Prevents silent rule erosion by making bypasses honest and traceable.

## Requirements

### REQ-BYPASS-001: Commit body MUST contain NO-VERIFY reason
**Priority**: must

**Scenario**: GGA fails due to technical limitation
- GIVEN a commit triggers GGA pre-commit validation
- WHEN GGA fails due to ARG_MAX overflow, provider outage, or scope-of-change mismatch
- THEN the commit body MUST contain a line `NO-VERIFY: <technical reason>`
- AND the reason MUST name the specific failure mode (e.g., `NO-VERIFY: ARG_MAX overflow on 11 working-tree files`)

**Acceptance criteria**:
- [ ] `git log --format=%B` shows `NO-VERIFY:` line in commit body
- [ ] Reason names a specific technical failure, not convenience

**Scenario**: Normal commit without bypass
- GIVEN GGA pre-commit passes successfully
- WHEN the commit is created
- THEN no `NO-VERIFY:` line is present in the commit body

**Acceptance criteria**:
- [ ] Clean commits remain clean — no spurious bypass markers

---

### REQ-BYPASS-002: Follow-up fix commit required in same PR
**Priority**: must

**Scenario**: Bypass used in a pull request
- GIVEN a commit with `NO-VERIFY:` in its body
- WHEN the PR containing that commit is open
- THEN a follow-up fix commit MUST be created within the same PR
- AND the fix MUST resolve the violations GGA would have caught

**Acceptance criteria**:
- [ ] Same PR contains both the bypass commit and the fix commit
- [ ] `gga run` passes on the fix commit (or the fix commit also documents remaining known violations)

**Scenario**: Pre-existing violations out of scope
- GIVEN `NO-VERIFY:` documents pre-existing violations outside the change scope
- WHEN the fix commit addresses those violations
- THEN the fix commit MUST fix all violations named in the bypass reason

**Acceptance criteria**:
- [ ] Every violation named in `NO-VERIFY:` reason is resolved by the fix commit

---

### REQ-BYPASS-003: Bypass documents technical failure only
**Priority**: must

**Scenario**: Accepted bypass reasons
- WHEN `NO-VERIFY:` is used
- THEN the reason MUST be one of: ARG_MAX overflow, provider outage, scope-of-change mismatch
- AND convenience reasons (e.g., "too slow", "in a hurry") MUST NOT be accepted

**Acceptance criteria**:
- [ ] AGENTS.md rule #41 lists the three accepted bypass reasons
- [ ] PR review rejects convenience-only bypasses
- [ ] Bypass without `NO-VERIFY:` line is treated as a rule violation
