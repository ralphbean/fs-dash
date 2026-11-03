## 1. Triage inputs

- [x] 1.1 Add deferred review, review-request, check, and activity field requests through the existing scheduler.
- [x] 1.2 Implement effective-review and bot-approval derivation with invalidation and unknown-state tests.
- [x] 1.3 Establish and document the concrete Fullsend activity evidence contract for the target installation, including freshness and negative-result rules.
- [x] 1.4 Implement the Fullsend activity adapter and tests for active, inactive, unknown, stale, and failed results.
- [x] 1.5 Prefetch activity for all listed PRs with a five-request concurrency cap, FIFO queueing, selected-PR promotion, and deduplicated queue refill.
- [x] 1.6 Bound selected-PR detail reads to two in flight and discard queued work when selection changes.

## 2. Prioritized list

- [x] 2.1 Implement pure section/filter/ranking projections with reason codes and deterministic ties.
- [x] 2.2 Add all, assigned-to-me, review-requested-from-me, and authored-by-me filters plus active-agent section and status indicators.
- [x] 2.3 Verify selection stability and ranking changes as deferred fields arrive or fail.
- [x] 2.4 Start with the assigned-to-me filter active.
- [x] 2.5 Include viewer identity and assignees in the initial PR list query, with per-PR fallback for incomplete results.
- [x] 2.6 Show the total open count and active-filter count in each repository summary, including pending rows.
- [x] 2.7 Add an authored-by-me filter using the PR author and viewer logins.

## 3. Local reminders

- [x] 3.1 Implement a versioned atomic local state store keyed by source host/repo/PR.
- [x] 3.2 Add star toggle, visible marker, and persistence across restarts.
- [x] 3.3 Add snooze selection, hidden-items view, automatic wake-up, and manual unsnooze.
- [x] 3.4 Verify restart, clock rollover, and corrupt-state behavior without mutating GitHub.
