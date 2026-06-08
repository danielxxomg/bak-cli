# Proposal: Why bak? Comparison Section

## Intent

Users comparing CLI backup tools need to understand what makes bak unique. Currently the README lists features but doesn't contrast bak against popular alternatives (chezmoi, mackup, stow, dotbot). This makes it harder for potential users to choose bak over tools they already know.

## Scope

### In Scope
- Add "Why bak?" section after Features, before Installation
- Comparison table: bak vs chezmoi vs mackup vs stow
- Focus on AI-agent-aware differentiators

### Out of Scope
- Detailed feature-by-feature tutorials
- Benchmarks or performance comparisons
- Adding dotbot column (less popular, similar to stow)

## Capabilities

### New Capabilities
- None

### Modified Capabilities
- `readme-structure`: ADDED requirement — README MUST include a "Why bak?" comparison table after the Features section

## Approach

Insert a markdown comparison section (~30 lines) into README.md after line 37 (end of Features list). Table columns: Feature | bak | chezmoi | mackup | stow. Rows highlight unique bak differentiators: AI agent auto-detection, cloud sync breadth, encryption, machine profiles, secret detection, mandatory dry-run, git-backed undo, YAML extensibility.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `README.md` | Modified | Insert comparison section after Features |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Table doesn't render correctly on GitHub | Low | Validate with local markdown preview |

## Rollback Plan

Revert the git commit — single-file change, trivial rollback.

## Dependencies

None.

## Success Criteria

- [ ] "Why bak?" section visible between Features and Installation
- [ ] Comparison table correctly formatted (renders on GitHub)
- [ ] All 7 unique differentiators present in table rows
