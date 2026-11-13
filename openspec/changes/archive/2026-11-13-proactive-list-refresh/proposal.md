## Why

List data and enrichment values become visibly stale after their freshness windows expire, but the application only refreshes them after user input. This makes users manually press `r` or revisit rows to keep an open list useful. The list view should schedule refreshes before loaded data expires.

## What Changes

- Schedule refreshes for list sections and selected or visible enrichment values shortly before their recorded freshness deadline while the list view is active.
- Keep showing the last successful values while a scheduled refresh is queued or in flight.
- Route scheduled work through existing selected/visible priority, request limits, pacing, and GitHub cooldown handling.
- Stop scheduling list refresh work while the user is in a detail workflow; refresh due data when the list view becomes active again.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `async-field-state`: Schedule refreshes before values expire and preserve retained values during them.
- `pr-triage`: Keep the active list and visible enrichment current automatically while preserving existing prioritization.

## Impact

Changes the Bubble Tea UI freshness scheduler and list/enrichment queues in `internal/ui`. It uses existing `FreshUntil` provenance and shared GitHub scheduling, with no configuration or API changes.
