## Why

The single Review column combines approvals, requested changes, and comments from people and bots. Separate columns make the two sources of review feedback visible at a glance while retaining the existing review-state meanings.

## What Changes

- Replace the combined Review column with adjacent Human and Bot review columns.
- Derive each column from the current effective reviews for its author type and render both with the existing review states and data-freshness behavior.
- Keep the two columns represented in both text and emoji table modes, with compact labels in emoji mode.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `pr-triage`: Expose distinct current human-review and bot-review states while preserving incomplete-history handling.
- `tui-visual-system`: Replace the combined Review table column with Human and Bot columns and define their text and emoji labels.

## Impact

- Updates table column definitions, width selection, and review-cell projection in `internal/ui/table.go`.
- Reuses the existing `ReviewSet` author-type classification and effective-review aggregation; no GitHub API or persistence changes are needed.
