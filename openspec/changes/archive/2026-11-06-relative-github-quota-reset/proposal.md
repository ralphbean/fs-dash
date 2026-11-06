## Why

Concurrent GitHub responses can arrive out of order, allowing an older quota snapshot to replace a newer one and make the displayed remaining amount jump upward. The reset clock is also harder to interpret than a countdown.

## What Changes

- Keep the most advanced `used` counter for the same quota resource and reset window, while accepting observations for a newly reset window.
- Show the reset as a relative countdown with an appropriate time unit and refresh it as the countdown changes.

## Capabilities

### New Capabilities

### Modified Capabilities
- `github-api-quota`: Keep current-window quota observations from moving backward and show reset time as a relative countdown.

## Impact

- Quota observation selection in `internal/ui/model.go`.
- Reset countdown formatting and refresh timing in `internal/ui/triage.go` and `internal/ui/model.go`.
- Quota behavior tests and the GitHub API quota specification.
