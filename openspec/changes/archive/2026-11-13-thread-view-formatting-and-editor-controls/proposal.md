## Why

Review threads currently show an uncolored diff and unframed comment bodies, which makes code changes and comment boundaries hard to scan. The comment editor also lacks cursor-oriented editing shortcuts and uses a submission key the user wants to change.

## What Changes

- Color added, removed, and hunk-header lines in thread diff context.
- Draw a distinct window frame around each comment in a review thread.
- Add `ctrl+k` to kill to the end of the current line, `ctrl+w` to delete the previous word, and `ctrl+e` to move to the current line's end in comment editors.
- Change comment submission to `ctrl+d`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `review-thread-workflow`: specify diff/comment presentation and comment editor controls.

## Impact

- Thread and comment rendering in `internal/ui/details.go` and `internal/ui/visual.go`.
- Comment editor input and cursor state in `internal/ui/details.go` and `internal/ui/model.go`.
- In-app help and README keybinding text.
- No new dependencies or API changes.
