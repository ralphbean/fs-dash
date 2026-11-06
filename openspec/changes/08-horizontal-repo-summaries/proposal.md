## Why

The PR list spends one line per repository on status summaries, pushing the table down as repositories are added. A single horizontal summary row preserves more vertical space while keeping every repository's status together.

## What Changes

- Render the configured repository summaries on one horizontal row above the PR table, regardless of repository count.
- Keep the current per-repository counts and status indicators, with the row constrained to the frame width.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `tui-visual-system`: Repository summaries share one horizontal row in the PR list frame.

## Impact

Changes PR list layout in `internal/ui` and its UI tests. No data, API, or configuration changes are required.
