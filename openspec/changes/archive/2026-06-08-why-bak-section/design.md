# Design: Why bak? Comparison Section

## Technical Approach

Insert a markdown `## Why bak?` section between Features (ends line 37) and Installation (starts line 39) in README.md. The section includes a brief intro paragraph and a comparison table using standard GitHub-flavored markdown.

## Architecture Decisions

| Option | Tradeoff | Decision |
|--------|----------|----------|
| Place after Features vs after Installation | After Features = user sees value prop before install instructions; After Installation = user already committed | After Features (better conversion flow) |
| 4 columns (bak, chezmoi, mackup, stow) vs 5 (add dotbot) | 5 columns may overflow on mobile; dotbot is less popular | 4 columns (remove dotbot) |
| Table headers bold vs pipe table | Pipe tables are simpler and GitHub-native | Pipe table |
| Emoji checkmarks (✅/❌) vs text (Yes/No) | Emojis are more scannable; text is more accessible | ✅/❌ for visual scanability |

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `README.md` | Modify | Insert section block at line 38 (between Features and Installation) |

## Data Flow

No data flow — this is a static documentation change. No Go code, no API, no compilation needed.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Manual | Markdown rendering | Preview on GitHub, verify table columns/rows |
| Manual | Section placement | Confirm heading appears in correct order |
| Manual | Link integrity | Verify any relative links resolve |

No automated tests — this is a documentation change with no executable code.

## Migration / Rollout

No migration required. Single-file change, revert with `git revert`.

## Open Questions

None.
