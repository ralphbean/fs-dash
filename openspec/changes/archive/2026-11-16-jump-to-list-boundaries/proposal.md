## Why

Moving through a long PR list one row at a time with `j`/`k` is slow when the desired item is at an edge, and scrolling a long review-thread diff one line at a time has the same problem. Vim-style boundary keys provide direct access to either end in both views.

## What Changes

- In PR list sections, add `gg` to select the first row and `G` to select the last row.
- In the review-thread diff view, add `gg` to scroll to the top of the thread content and `G` to scroll to the bottom.
- Show both shortcuts in the list and thread footers/help; keep the selected row visible after a list jump.
- Reserve `g` and `G` from custom command bindings so the navigation keys cannot be silently shadowed.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `pr-list`: Define keyboard shortcuts for jumping to the first and last row of a PR list.
- `review-thread-workflow`: Define keyboard shortcuts for jumping to the top and bottom of thread diff content.
- `user-config`: Reject custom command bindings that conflict with the new navigation keys.

## Impact

Updates list and thread-mode key handling and keyboard help in `internal/ui/model.go`, `internal/ui/details.go`, and `internal/ui/triage.go`, plus custom binding validation in `internal/config/config.go`. The mappings follow Vim convention: lowercase `gg` moves to the top; uppercase `G` moves to the bottom.
