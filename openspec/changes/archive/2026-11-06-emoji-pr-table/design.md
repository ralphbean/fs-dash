## Context

The table currently defines one text header per column, while Checks, Review, and Threads compute display strings directly from fetched field state. Column widths are selected before the row cells render. See proposal.md for motivation and the change specs for the visible behavior.

## Goals / Non-Goals

**Goals:**

- Keep data interpretation independent from whether the table shows text or emoji.
- Let a column provide its compact header and cell representation through the shared table presentation model, so future compact columns can join without adding special cases to the table renderer.
- Calculate layout from the active representation and preserve the existing roles, alignment, links, and counts.
- Persist the mode choice and keep local state files named for their contents.

**Non-Goals:**

- Add a configuration option for the mode.
- Change status derivation, GitHub requests, ranking, other columns, or detail views.

## Decisions

1. **Persist the display mode separately from per-PR triage data.** Store it in `preferences.json` under the existing fs-dash state directory, using the same `XDG_STATE_HOME` resolution and home-directory fallback as local triage data. Default to emoji when the preference is absent. Save each toggle atomically; if saving fails, keep the new mode for the current run and show the user an error.

2. **Keep semantic status separate from formatted output.** Checks, Review, and Threads should resolve fetched data into semantic states and associated values such as counts and completeness. Column presentation metadata should provide text and emoji encoders for those values. The renderer selects a representation from the active mode; it should not infer meaning by inspecting display strings. Shared field lifecycle states can use common semantic states while columns provide their own outcome mappings. Emoji loading cells use the current frame of the animated spinner, and ticking continues while any compact status field is active.

3. **Make compact presentation extensible per column.** Extend the existing column presentation data to describe text and emoji headers and cell formatting. Columns without an emoji representation retain their existing text output. Adding a future compact column should require presentation metadata and a semantic formatter, not a separate mode branch in table layout or input handling.

4. **Derive widths after selecting representation.** Build headers and cells using the active mode, then calculate widths and truncation from those rendered values; emoji mode must not retain the wider text-mode minimums. Continue using ANSI-aware terminal width measurement so emoji, variation selectors, and styled text do not skew alignment. Preserve counts in both modes, including running/failing check counts and incomplete thread counts.

5. **Show the active action in keyboard help.** Add `e` to the PR-list hint and help text, describing the switch between emoji and text labels. Toggling changes presentation only and applies across all PR-list sections.

6. **Name per-PR state for its scope and migrate existing data.** Store stars and snoozes together in `triage.json`, since both are keyed by PR identity. If `triage.json` is absent and `reminders.json` exists, read the legacy file and atomically write its contents to `triage.json`. Prefer `triage.json` on later runs and leave the legacy file untouched so a failed migration or downgrade cannot destroy the original data.

## Risks / Trade-offs

- **Emoji width varies across terminals** → Measure rendered output with the terminal-width helper and cover emoji and text layouts at common and narrow widths.
- **Symbols may be unfamiliar** → Keep text mode one keypress away and make the current toggle visible in the keyboard hints/help.
- **Legacy state migration could be interrupted** → Write the new file atomically, retain `reminders.json`, and prefer a successfully written `triage.json` on subsequent runs.

## Migration Plan

New preferences are written to `preferences.json`, while per-PR state is written to `triage.json` in the existing fs-dash state directory. Existing `reminders.json` data is copied to `triage.json` only when the destination is absent; the original file is retained. A saved display mode is restored on startup, with emoji mode as the first-run default.
