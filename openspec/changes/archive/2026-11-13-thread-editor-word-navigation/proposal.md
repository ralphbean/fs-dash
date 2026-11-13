## Why

The comment editor now supports cursor-aware edits, but moving through longer drafts still requires repeated arrow presses. Word-level navigation and a line-start shortcut make replies faster to review and revise before posting.

## What Changes

- Add `ctrl+left` and `ctrl+right` to move between word boundaries in comment drafts.
- Add `ctrl+a` to move to the beginning of the current line.
- Preserve the draft and keep the caret visible as these shortcuts move through the text.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `review-thread-workflow`: extend comment editor navigation controls.

## Impact

- Key handling and caret visibility in `internal/ui/details.go`.
- UI tests in `internal/ui/details_test.go`.
- Help text, README, and the main review-thread workflow spec.
