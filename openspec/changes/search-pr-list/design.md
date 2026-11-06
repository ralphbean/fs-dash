## Context

See `proposal.md` for motivation and scope. PR titles and authors are available in the lightweight list data; assignee logins may be incomplete and can be enriched later. The list is already ordered into visible sections by the active filter.

## Goals / Non-Goals

**Goals:** Search the current filtered PR rows in their displayed order, preserve the selection on search exit, and make results repeatable with `n`.

**Non-Goals:** Search PR descriptions, repository names, issue lists, hidden filters, or detail text. Search does not filter or reorder rows.

## Decisions

Use case-insensitive substring matching against the same PR title, author login, and assignee login values shown in the table. Treat the visible sections for the active filter as one ordered search sequence. Forward search begins after the selected row; reverse search begins before it. If there is no selected row, forward search starts at the first row and reverse search at the last. Repeating `n` starts after the current hit in the last search direction and wraps. If only one row matches, `n` may select that row again after a full cycle.

Update selection as the query is typed, scanning from the selection captured when search began. Keep the most recent match selected when `Esc` closes the prompt, and retain the query and direction so `n` can continue afterward. Enter accepts the query and closes the prompt with the same selection. An empty query does not move selection. A query with no matches leaves selection unchanged and shows no-match feedback.

Search known assignee logins immediately. If no row matches while some assignee lists are incomplete, indicate that coverage is partial; if assignee data arrives while the prompt is open, reevaluate the current query against the updated values. Move help to `F1` so `?` can open reverse search, and reject custom bindings for `/`, `n`, and `F1` (with `?` remaining reserved).

## Risks / Trade-offs

- Existing `?` help users need a replacement shortcut → keep help available through `F1` and update the footer and help text.
- Incomplete assignee data can hide a match temporarily → report partial coverage when there is no known match and assignee lists remain incomplete.
- Search results can move while assignee data updates → recompute from the search-start selection while the prompt is open, and preserve the latest hit on exit.
