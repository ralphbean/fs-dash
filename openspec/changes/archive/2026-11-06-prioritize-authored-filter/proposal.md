## Why

The authored-by-me filter should be the default and first personal view, while starred and snoozed PRs should be reachable through the same filter cycle. This gives the list one consistent navigation path for its views.

## What Changes

- Move authored-by-me to the first personal filter in the cycle, immediately after all open PRs.
- Make authored-by-me the startup filter.
- Show the filters together in a horizontal tab strip, with the active filter highlighted; navigate left and right with `h` and `l`.
- Remove the one-way `f` filter shortcut and avoid repeating the active filter name or item-action hints in the PR attention pane.
- Keep `s` for starring a PR, `z` for choosing its snooze duration, and `u` for unsnoozing.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `pr-triage`: Define the startup filter, available filter cycle, and filter navigation affordance.

## Impact

- Affects TUI filter initialization, projection, keyboard handling, tab rendering, and help text in `internal/ui/model.go`, `internal/ui/triage.go`, and `internal/triage/projection.go`, plus `internal/config/config.go` and the PR triage specification.
