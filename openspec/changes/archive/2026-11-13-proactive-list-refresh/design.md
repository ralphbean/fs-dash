## Context

See `proposal.md` for the user problem. List sections and enrichment values already carry `FreshUntil` timestamps, and the UI already prioritizes selected and visible work, limits background requests, and preserves values during refresh.

## Goals / Non-Goals

**Goals:** Use freshness metadata to start automatic work before list data currently expires. Keep the scheduler active only while the list view is in use and route work through existing request controls.

**Non-Goals:** Change freshness durations, request contents, GitHub quota policy, or detail-view refresh behavior. This change does not force periodic refreshes of offscreen enrichment.

## Decisions

Schedule one Bubble Tea timer for the earliest freshness deadline among loaded list sections and selected or visible enrichment values, minus a 20-second lead. Recalculate the timer after each model update so new responses, selection changes, and viewport changes adjust the next deadline. On a timer event, refresh due list sections and queue only selected or visible enrichment values that are within the lead window.

Use existing list fetch and enrichment queue paths rather than creating a separate network worker. This preserves selection priority, request caps, pacing, cancellation generations, and rate-limit cooldowns. When the UI leaves list mode, invalidate the timer. On return, an already-due deadline produces an immediate timer event.

Offscreen enrichment remains on the existing background path. This bounds the automatic work to data that is currently presented to the user while allowing stale background values to refresh as they are selected or become visible.

## Risks / Trade-offs

- Slow requests or shared cooldowns can outlast the 20-second lead → retain values, show the existing refresh state, and honor the shared cooldown rather than bypassing it.
- Frequent list changes can move freshness deadlines → recompute from current provenance after each model update and discard obsolete timer generations.
- Automatic requests add GitHub traffic for open list views → refresh only loaded list sections and selected or visible enrichment; leave offscreen work under existing scheduling.
