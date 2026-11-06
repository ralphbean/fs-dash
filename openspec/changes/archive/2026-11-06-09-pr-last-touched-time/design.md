## Context

See `proposal.md` - Why. The list query currently supplies GitHub's PR-level `updatedAt`, while the table renders each row immediately. GitHub also exposes `updatedAt` on the `timelineItems` connection. The existing UI already enriches PR rows asynchronously with bounded, visibility-prioritized field fetches.

## Goals / Non-Goals

**Goals:**
- Obtain GitHub's timeline `updatedAt` timestamp for each PR.
- Keep the initial list responsive and limit concurrent timeline requests.
- Display a pending or failure state when a timeline timestamp is not yet available.

**Non-Goals:**
- Treat a new head commit as the only kind of PR activity.
- Reconstruct or display the full PR timeline in the table.
- Change GitHub's PR-level `updatedAt` data used elsewhere.

## Decisions

- Add a dedicated timeline-activity enrichment field instead of reusing `PullRequest.updatedAt`. GitHub exposes a separate `timelineItems.updatedAt` value that represents when that timeline was last updated.
- Fetch the timeline connection timestamp for each PR. Do not fetch or parse timeline items because the connection already provides the needed timestamp.
- Reuse the existing bounded background fetch queue and its visible-row prioritization. The PR list renders before this enrichment finishes; the Updated cell shows a pending marker and later refreshes with the relative timestamp.
- Keep failures distinct from valid activity timestamps. A failed or missing timeline timestamp SHALL render an unavailable state instead of substituting PR `updatedAt`.

## Risks / Trade-offs

- [Risk] Timeline metadata lookup adds API requests across large lists → Use the existing concurrency cap and prioritize visible rows.
- [Risk] A newly added item may not immediately appear while a cached field is fresh → Use the existing field freshness and explicit-refresh behavior.

## Migration Plan

No data migration is required. Existing list responses continue to load as before; the Updated column uses the new enrichment when available. On rollback, remove the enrichment and restore the list's PR-level timestamp rendering.
