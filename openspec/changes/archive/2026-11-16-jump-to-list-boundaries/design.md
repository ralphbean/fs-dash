## Context

See `proposal.md` for motivation. The list moves selection one row at a time with `j`/`k`; the thread view scrolls its diff and comments with the same keys.

## Goals / Non-Goals

**Goals:** Recognize the Vim-style `gg` sequence and uppercase `G` in PR list and thread/diff modes. Keep list jumps on the existing selection path and thread jumps on the existing content-scroll path.

**Non-Goals:** Add paging, change filter navigation, or bind these keys in help, PR details, snooze, editor, or log-console modes.

## Decisions

Treat a `g` key as the start of a two-key sequence, scoped to the current mode. Jump to the first list row or top of thread content only when the next key completes `gg`; if another key arrives, process that key normally. Bind uppercase `G` directly to the last list row or bottom of thread content. Reserve both `g` and `G` in custom binding validation so the built-in navigation cannot be shadowed. This follows Vim convention and preserves the distinction between `g` and `G`; `Home`/`End` were considered but would duplicate the same actions with terminal-dependent key reporting.

Route a list jump through the model's existing selection update behavior so the viewport and selected-row prefetch follow the new selection. If the active section has no rows, keep selection empty. In thread mode, set the existing thread scroll position to the current top or bottom of the thread content. Only interpret these shortcuts in PR list and thread modes; other modes retain their existing key behavior.

## Risks / Trade-offs

- Existing configurations that bind `g` or `G` will fail validation after these keys become reserved → identify the conflicting key in the validation error so the user can rebind that command.
- A lone `g` could otherwise appear to do nothing → show the `gg` chord in both view footers and help; a non-`g` second key remains available to its normal handler.
