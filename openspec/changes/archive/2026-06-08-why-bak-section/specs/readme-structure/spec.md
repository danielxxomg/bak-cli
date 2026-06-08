# Delta for README Structure

## ADDED Requirements

### Requirement: Why Bak Comparison Section

The README MUST include a "Why bak?" section positioned after the Features section and before the Installation section. The section MUST contain a comparison table contrasting bak against generic dotfile managers to highlight bak's AI-agent-aware differentiators.

(Previously: No comparison section existed)

#### Scenario: Section placed correctly

- GIVEN a user scrolls through README.md
- WHEN they pass the Features list
- THEN a "Why bak?" heading MUST appear before the Installation heading
- AND the section MUST contain a comparison table

#### Scenario: Comparison table has required columns

- GIVEN the "Why bak?" comparison table
- WHEN rendered
- THEN columns MUST be: Feature | bak | chezmoi | mackup | stow
- AND the table MUST include at least these rows: AI agent auto-detection, Cloud sync, Encryption at rest, Machine profiles, Secret detection, Mandatory dry-run, Git-backed undo

#### Scenario: Table highlights bak differentiators

- GIVEN the comparison table
- WHEN a user scans bak's column
- THEN at least 5 rows MUST show ✅ for bak where competitors show ❌
- AND the unique differentiators MUST be visually distinct (checkmark vs cross icons)
