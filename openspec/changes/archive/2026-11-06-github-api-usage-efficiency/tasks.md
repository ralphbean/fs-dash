## 1. Consolidate per-PR summary requests

- [x] 1.1 Build one GraphQL operation for checks, reviews, review requests, and unresolved review threads while preserving each result's completeness state.
- [x] 1.2 Decode the combined result into the existing independent summary values and verify partial connections do not corrupt complete summaries.

## 2. Prioritize and cache enrichment

- [x] 2.1 Schedule selected and visible PR enrichment first, then pace offscreen work at low priority without treating unrequested values as inactive or negative.
- [x] 2.2 Add a bounded local enrichment cache keyed by host, repository, PR number, head SHA, and field family, with atomic writes and expiry pruning.
- [x] 2.3 Restore fresh cached summaries at startup and keep expired values visibly stale while refreshes run.

## 3. Coordinate GitHub rate-limit cooldowns

- [x] 3.1 Extend shared response metadata handling with `Retry-After`, primary quota reset, and remaining-quota information.
- [x] 3.2 Add one request gate shared by list, summary, activity, and manual-refresh work; queue requests until the global cooldown expires.
- [x] 3.3 Apply server-provided waits and bounded exponential backoff for secondary limits without retrying during the active cooldown.

## 4. Verify request efficiency and rate-limit behavior

- [x] 4.1 Add tests for combined-query decoding, cache key/freshness/revision behavior, and offscreen queue priority.
- [x] 4.2 Add tests for shared primary and secondary cooldowns, `Retry-After`, and manual refresh while cooling down.
- [x] 4.3 Run the Go tests, build, formatting checks, diff checks, and strict OpenSpec validation.
