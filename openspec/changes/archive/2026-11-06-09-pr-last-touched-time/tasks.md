## 1. GitHub Timeline Enrichment

- [x] 1.1 Add a timeline activity field that retrieves GitHub's `timelineItems.updatedAt` value for a PR.
- [x] 1.2 Keep timeline lookup failures and missing timestamps unavailable rather than falling back to the PR-level `updatedAt` value.

## 2. PR Table Integration

- [x] 2.1 Queue timeline activity for listed PRs through the existing bounded background prefetch, prioritizing visible rows while keeping list rendering immediate.
- [x] 2.2 Render the Updated column from timeline activity, showing pending and unavailable states until a valid timestamp is available.

## 3. Specification and Validation

- [x] 3.1 Sync the `tui-visual-system` delta into the main specification and validate the OpenSpec change.
- [x] 3.2 Build the application and confirm the worktree diff is clean of formatting errors.
