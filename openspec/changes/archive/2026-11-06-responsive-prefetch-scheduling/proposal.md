## Why

When a repository has many open PRs, background enrichment creates a large queue. The UI scheduler repeatedly rebuilds and sorts the full PR projection while checking queued items, blocking animation and input.

## What Changes

- Compute the interactive PR set once per scheduling pass and use constant-time membership checks while scanning queues.
- Build the listed PR revision set once when dropping stale queued fields, instead of searching every PR for each queued field.
- Reuse the same projected sections when determining the selected PR and visible rows for prefetch priority.

## Capabilities

### New Capabilities

### Modified Capabilities
- `async-field-state`: Background prefetch must not block spinner animation or interactive field updates as the PR queue grows.

## Impact

Changes affect UI prefetch scheduling and projection helpers. No API, cache format, or dependency changes are expected.
