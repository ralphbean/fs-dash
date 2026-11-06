## Why

The in-TUI help currently shows executable paths for configured shortcuts, which are implementation details rather than useful descriptions of what each key does. A short, user-defined summary makes custom bindings easier to recognize.

## What Changes

- Add an optional `summary` field to configured bindings and show it in the help view instead of the executable path.
- Fall back to the executable for existing bindings that do not yet define a summary.
- Add a summary to the commented custom-binding example and to the live bindings.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `user-config`: Configured command bindings may provide human-readable summaries shown beside their hotkeys in the in-TUI help.

## Impact

Affects `internal/config`, the help view in `internal/ui`, `config.example.toml`, and the user's live fs-dash config.
