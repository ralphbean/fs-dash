## Context

See proposal.md for the remaining UI stalls. Field and activity queues retain one item per requested PR field, while list rendering currently formats all projected rows and slices the resulting strings to terminal height.

## Goals / Non-Goals

**Goals:**
- Cap queue-entry inspection per UI update and resume outstanding work through scheduled continuation messages.
- Keep selected and visible work prioritized without scanning the full queue.
- Preserve list layout and clipping while formatting only visible row content.

**Non-Goals:**
- Reduce the set of fields prefetched or change API pacing.
- Change projection, filtering, sorting, or scoring behavior.

## Decisions

- Queue entries carry generations so an item promoted for immediate work can remain in the background FIFO safely; stale FIFO references are discarded in bounded chunks, including after a later requeue of the same field.
- Selected and visible items enter a separate priority queue. Both priority and background queues share a per-update inspection budget. A short scheduled continuation resumes work when the budget is exhausted and request slots remain.
- List content is represented as text lines or row descriptors. The existing selected-row centering and available-height calculation determine the visible slice before row descriptors are formatted.

## Risks / Trade-offs

- Continuation messages add queue latency when a large backlog contains stale references → use a short continuation delay and keep request concurrency limits unchanged.
- Table widths are measured from visible rows, so narrow columns can adjust as the viewport moves → preserve the existing width constraints and keep off-screen row formatting out of each render pass.
