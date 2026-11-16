## Why

The PR table shows when a PR was last touched but not who did it. The existing last-touched-by-me pane already identifies the latest human actor, so surfacing that value beside Updated makes the activity useful without another GitHub query.

## What Changes

- Add a `By` column immediately after `Updated`.
- Show the latest human timeline actor from the same data used to classify the Last touched by me pane.
- Keep bot activity out of the displayed actor and show pending, unavailable, and stale states consistently with the existing field lifecycle.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `tui-visual-system`: Add the By column and define its value, state, position, and responsive layout behavior.

## Impact

Changes table column specifications and cell rendering in `internal/ui/table.go`, plus the existing column-order expectation. It reuses existing timeline enrichment and does not add a GitHub request.
