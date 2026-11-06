## Why

Moving through a long PR list one row at a time with `j`/`k` is slow when the desired item is at an edge. Vim-style boundary keys provide a direct way to reach either end while preserving the existing selection-based navigation model.

## What Changes

- Add `gg` to select the first row and `G` to select the last row in PR list sections.
- Show both shortcuts in the list footer/help and keep the selected row visible after a jump.
- Reserve `g` and `G` from custom command bindings so the navigation keys cannot be silently shadowed.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `pr-list`: Define keyboard shortcuts for jumping to the first and last row of a PR list.
- `user-config`: Reject custom command bindings that conflict with the new navigation keys.

## Impact

Updates list-mode key handling and keyboard help in `internal/ui/model.go`, custom binding validation in `internal/config/config.go`, with coverage in the corresponding tests and deltas to the `pr-list` and `user-config` specifications. The mappings follow Vim convention: lowercase `gg` moves to the top; uppercase `G` moves to the bottom.
