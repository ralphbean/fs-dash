## Context

See `proposal.md` for motivation. The list currently moves selection one row at a time with `j`/`k` or the arrow keys. Selection also drives viewport rendering and prioritizes detail prefetch, so boundary jumps should use the existing selection path.

## Goals / Non-Goals

**Goals:** Recognize the Vim-style `gg` sequence and uppercase `G` in PR list mode, and keep boundary navigation consistent with current selection behavior.

**Non-Goals:** Add paging or viewport-only scrolling, change filter navigation, or bind these keys in help, detail, snooze, or log-console modes.

## Decisions

Treat a `g` key as the start of a two-key sequence and jump to the first row only when the next key completes `gg`. Bind uppercase `G` directly to the last row. Reserve both `g` and `G` in custom binding validation so the built-in navigation cannot be shadowed. This follows Vim convention and preserves the distinction between `g` and `G`; `Home`/`End` were considered but would duplicate the same actions with terminal-dependent key reporting.

Route a successful jump through the model's existing selection update behavior so the viewport and selected-row prefetch follow the new selection. If the active section has no rows, keep selection empty. Only interpret these shortcuts in PR list mode; other modes retain their existing key behavior.

## Risks / Trade-offs

- Existing configurations that bind `g` or `G` will fail validation after these keys become reserved → identify the conflicting key in the validation error so the user can rebind that command.
- A lone `g` could otherwise appear to do nothing → show the `gg` chord in list help and consume an incomplete chord without changing the selection.
