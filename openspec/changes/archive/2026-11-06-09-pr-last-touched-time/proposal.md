## Why

The PR table currently labels GitHub's generic `updatedAt` timestamp as “Updated.” That timestamp can move without a visible change in the PR conversation, making an untouched PR appear recently active; developers need the time of the latest visible timeline activity instead.

## What Changes

- Define the table's Updated value using GitHub's PR timeline `updatedAt` field, rather than the PR object's `updatedAt` field.
- Ignore background changes that do not appear as PR timeline activity, such as check-status refreshes.
- Fetch the timeline timestamp asynchronously so it does not delay the initial PR list.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `tui-visual-system`: define the Updated column as the time since the latest visible PR timeline activity.

## Impact

- Affected code: GitHub PR enrichment and the PR table's Updated column.
- Each PR's timeline timestamp will use the existing bounded background fetch queue; the initial list must still render without waiting for these reads.
