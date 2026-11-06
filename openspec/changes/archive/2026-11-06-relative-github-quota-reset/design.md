## Context

See proposal.md - Why. The quota model stores response counters and reset timestamps, and the UI currently accepts any observation received later by the local clock. Concurrent requests can therefore finish out of order. The current reset value is rendered as a local wall-clock time.

## Goals / Non-Goals

**Goals:** Keep the displayed counter stable within one resource/reset window and make the remaining wait easy to read in both regular and compact layouts.

**Non-Goals:** Track quotas for multiple GitHub credentials or API resources at once, change request scheduling, or add quota polling.

## Decisions

- Compare quota observations by resource and reset window. For the same resource and reset timestamp, reject a response if its `used` value is lower or its `remaining` value is higher than the displayed observation. Prefer observations with a higher `used` value, and retain the latest observation time when counters are equal. Accept an observation with a later reset timestamp as the next window; ignore responses from an earlier window.
- Render a future reset duration by rounding up to whole seconds below one minute, minutes below one hour, hours below one day, and days thereafter. Use full unit names in the regular layout and compact suffixes in narrow layouts. Show `reset now` when the timestamp has passed.
- Refresh the countdown periodically while the quota observation is displayed, using minute intervals for longer waits and second intervals inside the final minute. Stop the timer once the reset time has passed.

## Risks / Trade-offs

- [A counter can change because the authenticated identity or quota resource changed] -> Treat a resource change as a separate context selected by the latest observation; all current dashboard requests use the GraphQL resource and one `gh` authentication context.
- [A countdown timer can continue while the dashboard is idle] -> Limit updates to once per minute, increasing to once per second only in the final minute before reset.

## Migration Plan

No data migration is needed. The observation remains in memory and is replaced when a later quota window is observed.
