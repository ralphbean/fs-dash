## Context

See proposal.md for motivation. GitHub review enrichment already returns a complete `ReviewSet` with each review's author and bot classification. `EffectiveReviews` keeps the latest review by each author and refuses to report a definitive state when the history is incomplete. The current table aggregates that effective set into one review cell.

## Goals / Non-Goals

**Goals:** Derive Human review and Bot review states from the same current review snapshot, preserve existing state precedence and async/freshness indicators, and fit both columns into the current responsive table system.

**Non-Goals:** Change GitHub queries, review completeness rules, bot-approval ranking, review actions, or detailed review history.

## Decisions

1. **Reuse the existing effective review snapshot.** Resolve effective reviews once from the current `ReviewSet`, then partition by the existing author classification. `IsBot` reviews populate Bot; other user-authored reviews populate Human. Each partition uses the existing precedence: changes requested, approved, commented, then awaiting. If the review field is pending, stale, failed, or incomplete, both columns retain the same status indicator.
2. **Use one rendering path for both categories.** Parameterize the review-cell calculation by source category so the two columns share state mapping, colors, and failure behavior. No new domain data or GitHub request is required.
3. **Keep the columns adjacent where Review currently appears.** Use descriptive text headers `Human review` and `Bot review`, with `👤` and `🤖` in emoji mode. Column widths follow the active display mode. The existing narrow-table policy remains in effect and may hide trailing columns, including Bot review, before PR identity or title.

## Risks / Trade-offs

- **Adding a column uses more horizontal space** → recalculate widths in both display modes and retain the existing responsive hiding order.
- **A reviewer author is unavailable or has changed type** → rely on the `IsBot` classification already present in the fetched `ReviewSet`; the same complete-history behavior continues to guard both states.

## Migration Plan

No persisted data or API migration is needed. The combined Review cell is replaced in place by Human review and Bot review cells.
