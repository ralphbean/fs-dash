## Context

See `proposal.md` for motivation. The filter cycle is defined by `Filter.Next()`; the startup filter is configured separately.

## Goals / Non-Goals

**Goals:** Make authored-by-me the startup filter, then cycle through every list view including starred and snoozed PRs.

**Non-Goals:** Change star/unstar, snooze-duration, or unsnooze actions.

## Decisions

Represent starred and snoozed views as filter states in the existing cycle. Render each filter as a tab, compacting labels for narrow terminals while keeping every tab visible. Use `h` and `l` for previous and next filter navigation and wrap at either end. Remove the old `f` cycle key and keep `s`, `z`, and `u` for mutating star and snooze state. Non-snoozed filters continue to hide snoozed PRs; the snoozed filter uses the existing wake-time table presentation.

## Risks / Trade-offs

The cycle order and snoozed-view navigation change for existing users; starring, snoozing, and unsnoozing remain available through their existing actions.
