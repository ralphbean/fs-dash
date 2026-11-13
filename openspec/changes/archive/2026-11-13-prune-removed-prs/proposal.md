## Why

After a successful list refresh removes a merged PR, saved snooze state can currently synthesize a placeholder PR row. That row has incomplete identity data and can leak into filters such as authored by me, even though the PR is no longer open.

## What Changes

- Build list projections only from PRs returned by the latest successful open-PR list fetches.
- Stop fabricating placeholder rows from reminder state when a PR is absent from open-list results.
- Keep reminders attached to a PR while it is present in the open list, including normal snooze and unsnooze behavior.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `pr-list`: Require every list filter and section to contain only current open-list results.
- `local-reminders`: Clarify that reminder state does not reintroduce a PR absent from current open-list results.

## Impact

Changes the PR projection in `internal/ui/triage.go` and its behavior specifications. No API, storage format, or remote behavior changes.
