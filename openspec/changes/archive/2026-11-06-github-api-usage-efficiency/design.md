## Context

See proposal.md - Why. The current list fetch paginates all open PRs. Once it completes, the UI queues separate per-PR GraphQL requests for checks, reviews, review requests, review threads, timeline time, and activity. Detail and activity use separate concurrency schedulers, and field freshness currently expires after two minutes.

## Goals / Non-Goals

**Goals:**
- Reduce per-PR API calls for table status summaries.
- Reuse fresh enrichment across process restarts while retaining revision and freshness correctness.
- Prioritize interactive rows and smooth background work.
- Prevent any GitHub request from bypassing a shared primary or secondary rate-limit cooldown.

**Non-Goals:**
- Change the PR list query's scope or filter semantics.
- Remove eventual activity detection for offscreen PRs; defer and pace it instead.
- Add quota status UI here; that is specified by `github-api-quota-display`.

## Decisions

- Replace separate checks, reviews, review-requests, and review-thread summary calls with one combined per-PR GraphQL operation. Preserve the current connection bounds and decode each summary's completeness independently. Keep timeline and Fullsend comment activity separate because they have different data and freshness needs.
- Keep selected and visible work at the front of the queue. Do not start offscreen enrichment while visible or selected work is queued or in flight. After that work settles, process offscreen rows through a low-priority paced queue so activity coverage still completes without an initial request burst.
- Persist derived enrichment results in a dedicated `github-cache.json` under the fs-dash state directory, separate from user stars, snoozes, and preferences. Key entries by host, repository, PR number, head SHA, and field family; store fetch and expiry times. Reuse entries inside their freshness window, retain expired values as stale during refresh, and prune expired or least-recently-used entries under a bounded cache size. Start with the existing two-minute freshness interval, centralized by field family for later tuning.
- Reuse the GitHub response metadata parsing from `github-api-quota-display`. Extend its metadata to include `Retry-After`; use that duration for secondary limits, and use the primary reset time when the primary quota is exhausted. If a secondary limit has neither header, wait at least one minute and increase repeated retry delays with backoff.
- Put one shared rate-limit gate in front of all GitHub API work, including list, detail, activity, and forced refresh calls. A cooldown observed by any request pauses the other queues too. A manual refresh may enqueue work during cooldown but may not send it until the gate opens.
- Use the same global start-rate and concurrency controls for activity and detail requests instead of treating their separate queues as separate allowances. Preserve visible/selected prioritization inside those controls.

## Risks / Trade-offs

- [A combined query can have a larger response and higher per-query point cost] -> Request only fields used by the summaries, keep existing connection limits, and validate query cost against representative PRs.
- [A cached result can become stale before its time-to-live expires] -> Keep the initial short freshness interval, retain provenance and expiry, and show stale values as stale while revalidating.
- [Pacing offscreen activity detection can delay discovery of an active PR] -> Keep unscanned rows pending rather than inactive, prioritize selected/visible PRs, and continue the background scan when interactive work is clear.
- [A global cooldown delays even manual refresh] -> Queue the user's refresh request and execute it after the server-provided wait instead of issuing a request that can extend the rate limit.

## Migration Plan

- The cache starts empty on upgrade; existing stars, snoozes, and preferences are untouched.
- Write cache updates atomically and tolerate malformed or older cache entries by discarding them and fetching fresh data.
- No remote schema or credential migration is required.
