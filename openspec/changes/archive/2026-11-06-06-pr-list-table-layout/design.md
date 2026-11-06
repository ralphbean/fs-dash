## Context

See proposal.md for motivation. `listView` currently renders each PR as a free-form line containing its number, repository/title, author, update time, and a variable status string. Repository summaries appear above the rows. `triage.Project` already ranks review and active rows across the combined repository inputs. Assignees are present in the initial list result. Other enrichment fields are fetched for the selected PR, with a two-request limit; activity has a separate queue capped at five.

## Goals / Non-Goals

**Goals:** Keep the initial PR list fast, give all rows a stable table structure, and fill summary cells progressively without requiring selection.

**Non-Goals:** Add an activity column, split human and bot reviews into separate columns in this change, or change the existing priority calculation.

## Decisions

1. **Use one table renderer for each triage section.** Render the Review and Active sections with the same column definitions and preserve the existing repository summary lines. The projection already merges and priority-sorts PRs across configured repositories, so the rendering change preserves that ordering instead of adding per-repository row groups.

2. **Calculate widths from terminal cells, not string bytes.** Keep Star, PR link, author, assignees, relative update time, status, thread count, and Wake as bounded columns; give Title the remaining width. Use ANSI-aware display widths so styling and hyperlinks do not misalign cells. Left-align text, right-align counts and relative times, and center Star and compact status cells. Highlight the full selected row without a second cursor column; when color is disabled, use a plain text focus marker in the PR link cell.

3. **Use the repository/PR label as the hyperlink.** Render `<org>/<repo>#<number>` as the visible label and attach the PR URL through the terminal hyperlink escape supported by the renderer. Keep the label useful when a terminal ignores clickable hyperlinks.

4. **Hide columns from the right when space is short.** Remove the lowest-priority trailing columns first, preserving the PR link and Title and keeping rows single-line. This avoids horizontal scrolling and keeps identity available in narrow terminals.

5. **Prefetch table summaries in the background.** Enqueue check, review, and unresolved-thread summaries for every listed PR after the lightweight list arrives. Reuse the GitHub detail scheduler with an initial maximum of two in-flight requests. Promote rows in the visible viewport, including the selected row, ahead of off-screen work; do not duplicate in-flight or queued fields. Keep rendering and navigation independent of this queue, and show pending/failed/unknown states until each field settles. Keep activity on its separate five-request lane and retain the existing Active section without an activity column.

6. **Keep one aggregate review column for now.** Derive the displayed state from current effective human and bot reviews, preserving pending or failure when the source history is incomplete. Keep detailed review history in PR details. The column model can gain separate Human and Bot columns in a future change.

7. **Use relative time for Updated only.** Format recent update values as concise ages such as `5 mins ago` or `2 days ago`. Keep Wake in the existing local date-and-time format.

## Risks / Trade-offs

- Background summaries add requests proportional to the PR list size → cap concurrency at two, prioritize visible rows, share the GitHub scheduler, and retain terminal fetch states.
- Some terminals ignore hyperlink escapes → keep the full repository/PR label visible and preserve the normal selected-PR details link.
- Many columns cannot fit narrow terminals → remove trailing columns progressively while preserving the PR link and title.
- Relative times can become stale while the application is open → refresh their presentation from the current clock without refetching GitHub data.

## Migration Plan

No persisted data or configuration migration is needed. Existing list and field snapshots continue to populate the new cells; empty or incomplete summaries render as pending until fetched.
