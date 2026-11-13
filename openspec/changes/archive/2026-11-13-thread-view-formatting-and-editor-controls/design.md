## Context

Thread context currently renders as plain wrapped text inside the thread panel. Comments are sequential text blocks, and the shared comment editor appends input to the end of its rune buffer. See the proposal for the problem being addressed.

## Goals / Non-Goals

**Goals:**
- Use the existing theme roles to distinguish additions, removals, and hunk headers, while retaining readable plain output without color.
- Give every thread comment an individual border frame.
- Track a cursor in the comment buffer and render editing operations at that position.
- Apply the same comment editing controls to PR comments and thread replies.

**Non-Goals:**
- Add syntax highlighting for source code beyond diff line categories.
- Change GitHub comment or resolve APIs.
- Add clipboard integration or comment history editing.

## Decisions

- Split the diff hunk into lines and paint `+` additions, `-` removals, and `@@` hunk headers with semantic theme roles; leave context text neutral. This keeps the implementation independent of source-language parsers.
- Render each thread comment in a compact border frame with the author and timestamp in its heading. The outer thread panel remains the boundary for the selected thread.
- Store the editor cursor as a rune offset. Insertions, backspace, line kill, backward-word deletion, and line-end movement update this offset. Render a caret at the current location and keep it in the visible editor window.
- Reserve `ctrl+d` as the primary comment submission key and remove `ctrl+s` as a submit alias. Keep the existing `ctrl+enter` alternate to avoid removing a second working submit gesture.

## Risks / Trade-offs

- Nested comment frames use more vertical space → keep the comment heading concise and preserve the existing scroll behavior.
- Rune offsets differ from terminal cell widths for wide glyphs → use rendered line widths for wrapping and caret visibility, while retaining rune offsets for editing.
- `ctrl+d` can be interpreted as EOF in some terminals → handle the explicit Bubble Tea key event only while a comment editor is active.
