## Context

The shared comment editor stores a rune cursor and supports single-character movement, line-end movement, and deletion shortcuts. See the proposal for the motivation behind adding word-level and line-start navigation.

## Goals / Non-Goals

**Goals:**
- Move the caret between word boundaries with `ctrl+left` and `ctrl+right`.
- Move the caret to the current logical line's start with `ctrl+a`.
- Keep draft text and caret visibility correct after navigation.

**Non-Goals:**
- Change line editing, comment submission, or posting behavior.
- Add word selection or multi-cursor editing.

## Decisions

- Treat runs of Unicode whitespace as word separators, matching the existing `ctrl+w` deletion behavior.
- `ctrl+left` moves backward across whitespace and the preceding word; `ctrl+right` moves forward across the current word and following whitespace to the next word start.
- `ctrl+a` moves to the logical line start by scanning backward to the prior newline. The existing `home` key remains supported.
- Exercise the movements through Bubble Tea key messages so the tests verify the same key names received by the UI.

## Risks / Trade-offs

- Word boundaries are whitespace-based rather than punctuation-aware → this matches `ctrl+w` and gives predictable behavior for code-like text.
