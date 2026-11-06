## Why

The authored-by-me filter should be the default and first personal view, while starred and snoozed PRs should be reachable through the same filter cycle. This gives the list one consistent navigation path for its views.

## What Changes

- Move authored-by-me to the first personal filter in the cycle, immediately after all open PRs.
- Make authored-by-me the startup filter.
- Add starred and snoozed PRs as filters in the same cycle so users reach them with `f`; remove the dedicated `S` shortcut for the snoozed view.
- Keep `s` for starring a PR, `z` for choosing its snooze duration, and `u` for unsnoozing.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `pr-triage`: Define the startup filter, available filter cycle, and filter navigation affordance.

## Impact

- Affects TUI filter initialization, projection, keyboard handling, and help text in `internal/ui/model.go`, `internal/ui/triage.go`, and `internal/triage/projection.go`, plus the PR triage specification.
