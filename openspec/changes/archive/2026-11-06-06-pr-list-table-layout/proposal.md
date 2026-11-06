## Why

The PR list currently compresses identity, ownership, update time, and changing status into free-form rows, with repository context repeated in the title and summary lines. A justified table will make those facts easier to scan and compare while retaining the existing priority ordering and review/active sections.

## What Changes

- Render review PRs from all configured repositories in one priority-sorted table, while keeping active Fullsend PRs in their separate section with the same columns and retaining repository summary lines.
- Use columns in this order: Star, clickable repository/PR link, title, author, assignees, relative update time, checks, overall review state, and unresolved review-thread count. Add a Wake column in the Snoozed view.
- Show checks as a compact state indicator and keep detailed check information in PR details. Use one combined review column for now; human and bot review columns may be added later.
- Keep the activity column out of scope. Retain the separate Active section.
- Align text left, counts and relative times right, and the star and compact status cells centrally. Highlight the selected row.
- On narrow terminals, hide lower-priority columns while keeping the repository/PR link and title visible.
- Keep priority visible through row order; keep priority reasons in selected-PR details.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `pr-triage`: Populate check, review, and unresolved review-thread summaries asynchronously for table rows, prioritizing visible rows without blocking the list.
- `tui-visual-system`: Use a justified, responsive table for PR lists with clickable identity links, relative timestamps, selection highlighting, and narrow-terminal column adaptation.

## Impact

Changes list projection and rendering in `internal/ui` and may adjust the presentation of existing triage data. No GitHub API or configuration changes are expected.
