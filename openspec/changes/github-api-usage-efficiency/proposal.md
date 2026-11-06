## Why

fs-dash currently makes several separate GraphQL calls for each PR and eagerly enriches offscreen rows, so a dashboard with many open PRs can spend its shared user quota quickly. Reusing fresh results, combining related summaries, and coordinating retries will reduce request volume and prevent parallel repository fetches from repeatedly hitting an exhausted quota.

## What Changes

- Combine per-PR check, review, review-request, and unresolved-thread summaries into one query.
- Prioritize selected and visible PRs, deferring offscreen enrichment to a paced, low-priority background queue while preserving pending/unknown states.
- Cache enrichment results across runs with freshness rules and keys that include repository, PR number, head SHA, and field family.
- Coordinate a shared GitHub API cooldown across repositories and fields, honoring `Retry-After` and primary reset metadata; manual refreshes wait behind that cooldown.

## Capabilities

### New Capabilities
- `github-api-efficiency`: Reduce duplicate, bursty, and repeated GitHub API requests while preserving accurate partial-data behavior.

### Modified Capabilities
- `async-field-state`: Define cached field freshness and a shared source-level cooldown, including retry timing.
- `pr-triage`: Defer background summary and activity enrichment for offscreen PRs while keeping their incomplete state explicit.

## Impact

- GitHub query construction and result decoding in `internal/github`.
- Fetch scheduling, field lifecycle, and PR queue prioritization in `internal/fetch` and `internal/ui`.
- A local cache under the fs-dash state directory.
- No new external dependency or GitHub API polling operation.
