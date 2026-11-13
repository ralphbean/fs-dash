## 1. Add classification data

- [x] 1.1 Include `isDraft` in lightweight PR list responses and the domain PR model, with GitHub decoding coverage.
- [x] 1.2 Add bounded timeline actor enrichment that finds the newest non-bot activity, identifies the current viewer as actor when applicable, and reports pending, incomplete, and failed states without changing the existing Updated timestamp.
- [x] 1.3 Add tests for human and bot actor decoding, pagination/completeness, and unavailable actor data.
- [x] 1.4 Ignore GitHub subscription markers that have no actor or timestamp and do not represent PR work.

## 2. Partition the filtered PR list

- [x] 2.1 Apply the active PR filter, then partition each non-snoozed PR exactly once using Drafts, Active Fullsend Agents, Last touched by me, and Default precedence.
- [x] 2.2 Preserve existing priority ordering within sections and selection identity as asynchronous classification moves rows between sections.
- [x] 2.3 Add projection tests for all sections, precedence combinations, bot activity, pending activity, filter changes, and snoozed-view isolation.

## 3. Render the ordered sections

- [x] 3.1 Render nonempty sections in order with shared table columns.
- [x] 3.2 Keep the combined list navigable and selection visible across section boundaries and terminal resizes; keep search in rendered section order.
- [x] 3.3 Add UI coverage for section order, empty sections, selected-row movement, and compact terminal layouts.
- [x] 3.4 Omit empty sections and add a blank line before each subsequent displayed section heading.

## 4. Validate the change

- [x] 4.1 Run focused UI and GitHub enrichment tests, then the full Go test suite.
- [x] 4.2 Sync the `pr-triage` and `tui-visual-system` delta specs and run strict OpenSpec validation.

## 5. Temporary classification diagnosis

- [x] 5.1 Add temporary list columns for the latest non-bot actor and timestamp, sourced from the existing timeline actor enrichment.
- [x] 5.2 Remove the temporary diagnostic columns and delta requirement after the user verifies the displayed data, before syncing or archiving the change.
