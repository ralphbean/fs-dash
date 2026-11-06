## Context

See proposal.md for the motivation. The UI model owns the in-memory cache, and the store's atomic file save performs encoding, filesystem writes, and sync operations.

## Goals / Non-Goals

**Goals:**
- Keep disk I/O out of Bubble Tea's synchronous update path.
- Prevent concurrent saves from replacing a newer snapshot with an older one.
- Coalesce mutations that arrive while a save is active.

**Non-Goals:**
- Change cache freshness, retention, or on-disk format.
- Make cache writes a prerequisite for displaying fetched data.

## Decisions

- The model tracks dirty and completed cache generations plus whether a save command is active. A cache mutation marks the state dirty; the update handler starts a save only when none is active.
- Each command receives a snapshot of cache entries and calls the store outside the update loop. Encoded values are immutable after insertion, so the snapshot can safely share their byte slices. A completion message clears the active save and starts one more save if newer cache state accumulated.
- Cache pruning and snapshot ordering happen before the store call, in the command, so the update loop does not sort the full cache on every result. A successful completion replaces the in-memory cache with its pruned snapshot only when no newer mutation has occurred.
- Failed saves are reported through the existing local warning and are not retried in a tight loop. A later cache mutation can retry the latest state.
- Cache access-time updates use a set of listed PR identities to avoid scanning the entire cache once per PR.

## Risks / Trade-offs

- A process exit can occur before an outstanding asynchronous cache write finishes → fetched values remain usable in the current session, and the cache is an optimization that can be rebuilt on a later run.
- A failed latest snapshot remains dirty but is not retried until another cache mutation → surface the warning and avoid an unbounded retry loop.

## Migration Plan

No on-disk migration is needed. Existing cache files continue to load and new snapshots retain the current format.
