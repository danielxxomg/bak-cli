# Proposal: README Refresh

## Intent

Restructure README.md for better first-impression, scanability, and discoverability. Current README has flat installation hierarchy, minimal badges (3), no platform table, weak Contributing section, and a 190+ line Configuration section. No content removal — pure reorganization + additions inspired by gentle-ai README structure.

## Scope

### In Scope
- Expand badges (Go version, platform, test count)
- Restructure Installation with recommended method + collapsible alternatives
- Add Supported Platforms table
- Condense Contributing to brief section + CONTRIBUTING.md link
- Add "Next Steps" section for new users
- Make Brand Assets collapsible via `<details>`

### Out of Scope
- Removing existing content (all sections preserved)
- Rewriting Configuration section (deferred — separate change)
- Adding "What It Does" Before/After (not applicable to CLI tool docs)
- Changing Architecture or Data Flow diagrams

## Capabilities

### New Capabilities
- `readme-structure`: README layout requirements — badge completeness, install hierarchy, platform table, contributing link, next steps section

### Modified Capabilities
- `docs-cleanup`: README accuracy rules now include badge and structural requirements

## Approach

Restructure README sections following gentle-ai flow: banner → badges → one-liner → features → install (recommended + collapsible) → quick start → commands → configuration → architecture → safety → contributing (brief + link) → next steps → brand assets (collapsible) → license. All existing content preserved, reorganized for scanability.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `README.md` | Modified | Full restructure — badges, install hierarchy, platform table, collapsible sections |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Badge URLs break on release | Low | Use shields.io dynamic badges that auto-update |
| Collapsible sections hide content from mobile | Low | Keep key info visible, collapse only secondary content |

## Rollback Plan

Revert README.md to pre-change version via git. No code changes — zero runtime risk.

## Dependencies

- None

## Success Criteria

- [ ] README has ≥6 badges (Go version, platform, tests, license, release, Go Report Card)
- [ ] Installation has recommended method prominent with alternatives collapsible
- [ ] Platform support table present with macOS/Linux/Windows
- [ ] Contributing section ≤5 lines + link to CONTRIBUTING.md
- [ ] Next Steps section exists with 3+ actionable items
- [ ] Brand Assets wrapped in `<details>` block
- [ ] All existing content preserved (no deletions)
