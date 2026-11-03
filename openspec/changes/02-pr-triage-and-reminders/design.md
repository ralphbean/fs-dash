## Context

This change builds on the PR list, scheduler, and field lifecycle from change 01. See proposal.md and the `pr-triage` and `local-reminders` specs.

## Goals / Non-Goals

**Goals:** Make ranking explainable and keep slow Fullsend activity detection independent of list readiness; persist only user-owned reminders.

**Non-Goals:** Infer an agent run from PR authorship alone, synchronize stars or snoozes to GitHub, or claim a universal Fullsend run API.

## Decisions

1. **Pure triage projections.** Filters and rankings consume field snapshots and produce sections with reason codes. The default order prioritizes direct review request, failed checks, unresolved review threads, and staleness, with deterministic source/repo/number tie breaking; unknown signals remain visible and do not score as false. Allow weights to evolve via config after initial behavior is measured. A hidden opaque score was rejected because it cannot explain attention order.
2. **Activity detection is a provider policy.** Put a `FullsendActivity` port behind the scheduler. Start with GitHub-visible evidence such as current agent-authored status or workflow markers and record the exact evidence URL/time. Establish the concrete marker contract against the target Fullsend installation during implementation; if evidence is absent or inconclusive, return unknown or inactive according to whether the query completed. Do not call an unverified endpoint or equate historical bot participation with current work.
3. **Composite review fields use dependency versions.** Effective review state is calculated from current review history per reviewer, then bot approval from that result. The projection records the versions of reviews and user identity that produced it; changes invalidate both it and rankings. This avoids a stale true value after dismissal or a newer review.
4. **Local state uses an atomic file store.** Keep stars and snoozes in an XDG state path keyed by source host, repository, and PR number. Use a versioned JSON document and atomic replace on write. Only one local process is assumed for v1; concurrent-process locking can be added if needed. Snoozes store absolute wake timestamps, and a clock tick plus startup load restores due items.

## Risks / Trade-offs

- Fullsend activity markers may differ by deployment → isolate detection and surface evidence plus unknown state.
- Expensive enrichment may exhaust GitHub quota → prioritize visible rows, coalesce requests, and cache with freshness bounds.
- A corrupt local state file may hide reminders → retain a backup/error message and start with visible PRs; never silently discard the file.
- Priority changes during enrichment may reorder rows → preserve selected identity and show reason codes.

## Migration Plan

Add the local state file on first star/snooze action. Existing users have no migration. Rollback leaves the file unused but intact.
