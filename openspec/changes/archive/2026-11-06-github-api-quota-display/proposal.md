## Why

When GitHub approaches its API quota, fs-dash currently gives users little warning before requests are blocked. Showing quota from responses the app already receives lets users see remaining capacity and the reset time without adding a polling request.

## What Changes

- Capture the latest GitHub API quota limit, remaining amount, reset time, and observation time from existing API responses.
- Show a compact quota indicator in the dashboard and make low or exhausted quota understandable.
- Distinguish primary quota status from request cooldown information when available.

## Capabilities

### New Capabilities
- `github-api-quota`: Capture and present the latest known GitHub API quota state.

### Modified Capabilities
- `tui-visual-system`: Keep quota status visible and legible within the dashboard layout, including compact terminal widths.

## Impact

- GitHub CLI response handling in `internal/github`.
- Dashboard status and layout in `internal/ui`.
- No new API polling endpoint or dependency; quota data comes from existing GitHub responses.
