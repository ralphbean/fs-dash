## Why

Even after removing repeated projection work, large prefetch queues can require substantial synchronous processing, and the list view formats every PR row before clipping to the terminal viewport. Both operations run on Bubble Tea's UI goroutine and can still delay animation and input.

## What Changes

- Bound field and activity queue processing per UI update and continue remaining work in later updates.
- Render only list rows that fit in the visible terminal viewport.

## Capabilities

### New Capabilities

### Modified Capabilities
- `async-field-state`: Large background queues must yield to UI updates, and list rendering must limit row formatting to visible rows.

## Impact

Changes affect the UI model's queue state and list rendering. No API, cache format, or dependency changes are expected.
