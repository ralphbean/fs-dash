## Why

The spinner pauses while enrichment results are cached because durable file writes run inside the UI update loop. Moving persistence out of that loop keeps animation and input responsive during slow storage operations.

## What Changes

- Move enrichment cache writes out of synchronous UI updates.
- Serialize writes and coalesce newer cache state while a save is in progress.
- Keep cache write failures visible through the existing local warning path.

## Capabilities

### New Capabilities

### Modified Capabilities
- `async-field-state`: Enrichment cache persistence must not block UI progress, and the newest cache snapshot must follow any in-progress write.

## Impact

Changes affect the Bubble Tea UI model and its enrichment cache tests. No public API or external dependency changes are expected.
