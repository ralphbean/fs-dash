## 1. Model compact column presentations

- [x] 1.1 Separate Checks, Review, and Threads semantic states and values from their text and emoji formatting, preserving counts, roles, and completeness.
- [x] 1.2 Add text and emoji header/cell representations to the shared column presentation model so future compact columns can opt in without renderer-specific mode branches.

## 2. Add emoji and text table modes

- [x] 2.1 Default the PR-list presentation mode to emoji and render the specified headers and state symbols while preserving check and thread counts.
- [x] 2.2 Add the `e` toggle and update keyboard hints/help to explain switching between emoji and text representations across all PR-list sections.
- [x] 2.3 Calculate column widths and truncation from the active representations and preserve alignment in emoji and text modes.
- [x] 2.4 Persist and restore the emoji/text mode in `preferences.json`, default to emoji when unset, and report save errors without changing the current session's selected mode.
- [x] 2.5 Store per-PR stars and snoozes in `triage.json` and migrate existing `reminders.json` data when the new file does not yet exist.

## 3. Verify the change

- [x] 3.1 Add UI tests for default and restored modes, preference save errors, animated loading, each supported state mapping, count preservation, `e` toggling, and consistent mode across sections.
- [x] 3.2 Add table layout coverage for emoji and text widths, including narrow terminal widths.
- [x] 3.3 Add coverage for legacy triage-state migration, then run the Go test suite, build, formatting and diff checks, and strict OpenSpec validation; resolve any failures.
