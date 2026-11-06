## Why

Refresh concurrency and background queue behavior are currently fixed in code, so users cannot tune responsiveness against GitHub request pressure. Putting the related limits in TOML makes the trade-offs adjustable while preserving today's behavior for existing configurations.

## What Changes

- Add a `[refresh]` TOML section for the shared GitHub request concurrency and minimum request-start interval.
- Make detail, activity, and offscreen concurrency limits configurable, along with their offscreen pacing intervals.
- Make the queue batch size and continuation yield interval configurable so users can tune background scheduling responsiveness.
- Default omitted values to today's behavior: 2 shared requests, 150 ms request-start interval, 2 detail requests, 5 activity requests, 1 offscreen request, 500/750 ms detail/activity offscreen intervals, 64 queue entries per update, and a 5 ms continuation delay.
- Document every setting and its relationship to the other limits in `config.example.toml`; keep GitHub-provided primary and secondary cooldowns authoritative.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `user-config`: Define optional refresh settings, defaults, validation, backward compatibility, and documented TOML comments.
- `github-api-efficiency`: Make the shared request concurrency and request-start pacing configurable while preserving the shared cooldown.
- `pr-triage`: Make detail, activity, and offscreen request limits and pacing configurable while preserving selected/visible priority.
- `async-field-state`: Make background queue batch size and yield interval configurable without allowing background scheduling to block interaction.

## Impact

Adds a refresh configuration model and validation in `internal/config`, applies it to the scheduler and UI queues in `internal/fetch` and `internal/ui`, and documents the settings in `config.example.toml`. Existing TOML files continue to use current defaults when `[refresh]` is absent.
