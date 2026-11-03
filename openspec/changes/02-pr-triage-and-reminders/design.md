## Context

This change builds on the PR list, scheduler, and field lifecycle from change 01. See proposal.md and the `pr-triage` and `local-reminders` specs.

## Goals / Non-Goals

**Goals:** Make ranking explainable and keep slow Fullsend activity detection independent of list readiness; persist only user-owned reminders.

**Non-Goals:** Infer an agent run from PR authorship alone, synchronize stars or snoozes to GitHub, or claim a universal Fullsend run API.

## Decisions

1. **Pure triage projections.** Filters and rankings consume field snapshots and produce sections with reason codes. The initial PR list query includes viewer identity and a bounded assignee connection, letting the assigned-to-me filter resolve from the first list response; incomplete assignee data stays unknown and can use the selected-PR detail query as a fallback. The default order prioritizes direct review request, failed checks, unresolved review threads, and staleness, with deterministic source/repo/number tie breaking; unknown signals remain visible and do not score as false. Allow weights to evolve via config after initial behavior is measured. A hidden opaque score was rejected because it cannot explain attention order.
2. **Activity detection is a provider policy with bounded prefetch.** The target contract is the GitHub PR issue-comment protocol in `fullsend-ai/fullsend` main at `90b7310ff8751a3c70e5d68214770dc168a1cbe7`: trusted agent status comments are authored by `fullsend-ai-review`, `fullsend-ai-coder`, `fullsend-ai-triage`, `fullsend-ai-retro`, or `fullsend-ai-prioritize`; each contains `<!-- fullsend:agent-status:<runID> -->`; and `<!-- fullsend:status:terminal -->` marks a terminal run. The newest recognized comment by `createdAt` wins. A non-terminal start younger than six hours is active; an older non-terminal start is stale. A terminal latest comment or a complete scan with no recognized marker is inactive. A comment fetch that fails or cannot establish completeness is failed or unknown, never inactive. Store the comment URL and timestamp as evidence; successfully fetched field snapshots stay fresh for two minutes, after which they are refreshed on selection. After each open-PR list load, enqueue activity reads in list order and run at most five at once. Selecting a queued PR promotes it to the front; an in-flight request is reused rather than duplicated. Use a dedicated scheduler lane so other enrichment does not reduce the activity limit. For other selected-PR detail fields, keep at most two requests in flight and drop queued work for a previous selection. This bounds scrolling bursts without delaying initial activity prefetch. This follows the current `nextwork` default `--stale-hours 6` and its marker rules; do not poll GitHub Actions or infer activity from PR authorship.
3. **Composite review fields use dependency versions.** Effective review state is calculated from current review history per reviewer, then bot approval from that result. The projection records the versions of reviews and user identity that produced it; changes invalidate both it and rankings. This avoids a stale true value after dismissal or a newer review.
4. **Local state uses an atomic file store.** Keep stars and snoozes in an XDG state path keyed by source host, repository, and PR number. Use a versioned JSON document and atomic replace on write. Only one local process is assumed for v1; concurrent-process locking can be added if needed. Snoozes store absolute wake timestamps, and a clock tick plus startup load restores due items.

## Risks / Trade-offs

- Fullsend activity markers may differ by deployment → isolate detection and surface evidence plus unknown state.
- Activity prefetch increases GitHub reads → cap it at five concurrent requests, prioritize the selected row, coalesce requests, and cache with freshness bounds. Detail reads stay bounded to two active requests and stale queued selection work is discarded.
- A corrupt local state file may hide reminders → retain a backup/error message and start with visible PRs; never silently discard the file.
- Priority changes during enrichment may reorder rows → preserve selected identity and show reason codes.

## Migration Plan

Add the local state file on first star/snooze action. Existing users have no migration. Rollback leaves the file unused but intact.
