## Context

See proposal.md - Why. The GitHub adapter uses `gh api` for paginated PR lists and per-PR enrichment, but currently discards response headers. The UI receives those results asynchronously and has a shared status area suitable for a compact, app-wide indicator.

## Goals / Non-Goals

**Goals:**
- Capture primary GraphQL quota metadata from responses already being made, including rate-limited responses.
- Keep one latest-known quota snapshot across repositories and enrichment operations.
- Show the remaining budget, reset time, warning state, and observation age without adding API calls.

**Non-Goals:**
- Change request scheduling, batching, cache policy, or the shared cooldown behavior; those belong to the separate API efficiency change.
- Persist quota snapshots across application runs.
- Display or predict GitHub secondary-rate-limit budgets before GitHub reports one.

## Decisions

- Capture `x-ratelimit-limit`, `x-ratelimit-remaining`, `x-ratelimit-used`, `x-ratelimit-reset`, and `x-ratelimit-resource` from `gh api --include` output. Response headers are available on successful and failed API responses, avoiding a separate quota query.
- Parse transport metadata separately from the JSON response body so existing GraphQL decoders continue to consume JSON. For paginated operations, retain the most conservative remaining amount from the response pages and its associated reset time; record when that observation was received.
- Pass an optional quota observation alongside list and field completion messages into one model-level snapshot. Ignore malformed or missing metadata rather than replacing a valid observation with fabricated values.
- Show a compact indicator in the shared dashboard status area. Use a warning state below 20% remaining and an exhausted state at zero; include the reset time and observation age so users can judge freshness.
- Leave the indicator unknown until a response provides metadata. Keep the last valid observation visible when later requests fail without quota headers.

## Risks / Trade-offs

- [The `gh api --include` output format can vary across pagination and error responses] -> Parse HTTP header blocks separately from JSON and cover single-page, multi-page, and API-error output with fixtures.
- [A last-known quota value can become stale while no request is active] -> Display its observation age and mark old snapshots stale instead of implying they are current.
- [A fixed warning threshold may not suit every user's request pattern] -> Start with a 20% presentation threshold and keep threshold policy isolated so it can be tuned later without changing the data contract.
