## Context

See proposal.md for the observed stall. Background enrichment can queue several field keys for every listed PR, while scheduling decisions run synchronously in the UI update loop.

## Goals / Non-Goals

**Goals:**
- Make queue prioritization scale linearly with queued fields and interactive PRs.
- Build one PR projection per priority calculation and reuse it for selection and visible-row decisions.

**Non-Goals:**
- Change which fields are prefetched, request limits, or offscreen pacing.
- Change row scoring, filters, or API behavior.

## Decisions

- Build a set of selected and visible PR identities once for each field or activity scheduling pass. Queue and in-flight membership checks then use direct map lookups rather than rebuilding the projection for every key.
- Share one projected section set between selected-row lookup and viewport prefetch selection. This removes duplicate sorting within a scheduling pass while preserving the existing priority order.
- When selection changes, build a set of currently listed PR revisions once and use it to retain valid queued summaries in one pass over the queue.

## Risks / Trade-offs

- A shared priority set can become stale if selection changes during a pass → each scheduling pass is synchronous and does not mutate selection, so the set is discarded when the pass returns.
