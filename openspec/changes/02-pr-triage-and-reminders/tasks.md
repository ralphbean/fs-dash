## 1. Triage inputs

- [ ] 1.1 Add deferred review, review-request, check, and activity field requests through the existing scheduler.
- [ ] 1.2 Implement effective-review and bot-approval derivation with invalidation and unknown-state tests.
- [ ] 1.3 Establish and document the concrete Fullsend activity evidence contract for the target installation, including freshness and negative-result rules.
- [ ] 1.4 Implement the Fullsend activity adapter and tests for active, inactive, unknown, stale, and failed results.

## 2. Prioritized list

- [ ] 2.1 Implement pure section/filter/ranking projections with reason codes and deterministic ties.
- [ ] 2.2 Add all, assigned-to-me, and review-requested-from-me filters plus active-agent section and status indicators.
- [ ] 2.3 Verify selection stability and ranking changes as deferred fields arrive or fail.

## 3. Local reminders

- [ ] 3.1 Implement a versioned atomic local state store keyed by source host/repo/PR.
- [ ] 3.2 Add star toggle, visible marker, and persistence across restarts.
- [ ] 3.3 Add snooze selection, hidden-items view, automatic wake-up, and manual unsnooze.
- [ ] 3.4 Verify restart, clock rollover, and corrupt-state behavior without mutating GitHub.
