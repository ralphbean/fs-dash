## Why

The Checks, Review, and Threads columns consume valuable width with repeated labels even though their states and values are compact. Emoji headers and state marks let the default table fit more useful PR information, while a text mode gives users a built-in way to learn the symbols.

## What Changes

- Start the PR table in emoji mode, remember the user's selected mode between runs, and add `e` to toggle between emoji and text labels and values for compact status columns.
- Give Checks, Review, and Threads concise emoji headers and map every rendered state to a defined emoji representation, using the animated spinner for loading cells. Preserve useful counts, including check counts and unresolved thread counts.
- Keep status meaning independent from its text or emoji rendering so additional compact columns can add representations without duplicating status logic.
- Include the toggle in visible keyboard help. Store global display preferences in `preferences.json` and per-PR stars and snooze times in `triage.json` within the fs-dash state directory.
- Migrate existing per-PR state from `reminders.json` to `triage.json` when the new file is absent, preserving existing stars and snoozes.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `tui-visual-system`: Change the PR table's Checks, Review, and Threads headers and cell representations, and define the emoji/text display mode.
- `pr-triage`: Add and document the `e` keyboard toggle for the table's emoji/text display mode.

## Impact

- Affects PR table column specifications and cell rendering, model state and key handling, local state files and migration, and keyboard hints/help.
- No GitHub data, ranking, filtering, or details-view behavior changes.
