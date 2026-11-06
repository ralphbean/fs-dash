## 1. Table layout and cells

- [x] 1.1 Add reusable table column definitions and ANSI-aware width/alignment helpers for the PR list.
- [x] 1.2 Render the clickable `<org>/<repo>#<PR>` link, Star, title, author, assignees, relative Updated time, compact Checks and Review states, and unresolved Threads count in the agreed order.
- [x] 1.3 Apply full-row selection highlighting and render the same columns in the Review and Active sections while keeping activity out of the table.
- [x] 1.4 Add the final Wake column to the Snoozed view.
- [x] 1.5 Hide lower-priority trailing columns on narrow terminals while preserving the PR link and title and keeping rows single-line.

## 2. Background summary prefetch

- [x] 2.1 Queue check, review, and unresolved-thread summaries for all listed PRs without delaying initial table rendering.
- [x] 2.2 Bound summary fetch concurrency, promote visible and selected rows ahead of off-screen rows, and avoid duplicate queued or in-flight requests.
- [x] 2.3 Render pending, current, stale, and failed summary states without treating incomplete data as a negative result.

## 3. Validation

- [x] 3.1 Add or update UI tests for column order, alignment, links, row selection, narrow layouts, and Snoozed view Wake cells.
- [x] 3.2 Add or update model tests for background prefetch ordering, concurrency limits, and terminal summary states.
- [x] 3.3 Run the Go build, relevant tests, and OpenSpec validation.
