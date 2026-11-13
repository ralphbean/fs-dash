## Context

See `proposal.md` and the two delta specs. The list currently has a triage projection with Review, Active, and Snoozed sections. The lightweight `domain.PR` has no draft flag, while the details model does. The `last_touched` enrichment supplies only the newest timeline timestamp; it does not identify an actor. That timestamp also drives the Updated column, whose meaning must continue to include all visible timeline activity.

## Goals / Non-Goals

**Goals:**
- Classify rows from the existing active filter into the four specified sections without duplicates.
- Make draft status available with the initial list data and obtain enough actor data to determine the newest non-bot timeline actor.
- Keep list rendering responsive while actor enrichment is pending, and preserve PR identity selection as rows move.

**Non-Goals:**
- Change filter definitions, filter navigation, PR actions, snooze behavior, or the meaning of the Updated column.
- Give sections independent selection or scrolling; they remain stacked regions in the existing single list viewport.

## Decisions

1. **Apply the current filter before section precedence.** The active filter determines the candidate PRs. Snoozed mode remains its existing separate view. Within the non-snoozed candidate set, assign each row once using the bottom-up precedence Drafts, Active Fullsend Agents, Last touched by me, then Default. Preserve the current priority ordering within each section. This keeps the new grouping orthogonal to filter behavior.

2. **Fetch draft state with lightweight PR identity.** Add GitHub's `isDraft` value to the initial PR list query and `domain.PR`, instead of waiting for details enrichment. Draft membership is a primary section decision, so the list should know it as soon as the row is available.

3. **Keep actor classification separate from Updated time.** Add a bounded background enrichment for the newest non-bot timeline actor. Inspect timeline events in newest-first order, ignore bot actors using GitHub actor type and the existing `[bot]` login convention, and stop when the most recent human actor is established. Ignore `SubscribedEvent` markers because GitHub does not attribute them to an actor or timestamp and they do not represent PR work. Keep paging while needed to distinguish the newest human event from newer bot events. If the scan cannot establish an actor because the history is incomplete or unavailable, report the classification as pending/unavailable and do not guess. Do not redefine `last_touched`, which continues to supply the Updated timestamp for all visible events.

4. **Use confirmed classifications and retain pending rows.** A confirmed draft or active-agent result takes its specified precedence. Otherwise, place a row with a confirmed current-viewer last-human actor in Last touched by me. Put all remaining rows, including those awaiting actor classification, in Default and retain an explicit pending or unavailable signal. This prevents incomplete enrichment from hiding work or falsely claiming the user was the last human actor.

5. **Reuse bounded enrichment scheduling and the single viewport.** Queue actor enrichment through the existing field scheduler, prioritizing selected and visible PRs and pacing offscreen work. Render nonempty sections in order with shared table columns, omitting empty sections and separating displayed headings with a blank line. Recompute the projection as data arrives while preserving the selected PR by stable identity. Search continues through rows in rendered section order.


## Risks / Trade-offs

- [Risk] A long run of bot-authored timeline events can require multiple pages before the latest human actor is known → Stop at the first non-bot event, use existing concurrency and pacing limits, and keep the row visibly pending until classification is conclusive.
- [Risk] Extra timeline reads can increase GitHub API use → Prioritize selected and visible rows and use the existing bounded, paced offscreen queue.
- [Risk] Rows can move after enrichment and surprise users → Preserve selection by PR identity and keep section order deterministic.
- [Risk] Empty headings use terminal height → Keep empty states compact and preserve the existing single scrollable viewport.

## Migration Plan

No persisted-data migration is required. Add draft state to list responses and the new actor enrichment, then enable the four-section projection. On rollback, remove the new classification field and restore the current Review and Active projection; the existing Updated timeline timestamp remains unchanged.
