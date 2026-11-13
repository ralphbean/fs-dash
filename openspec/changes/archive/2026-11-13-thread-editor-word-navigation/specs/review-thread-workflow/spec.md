## MODIFIED Requirements

### Requirement: Reply editor preserves drafts
The system SHALL provide a multi-line editor for a reply to the selected review thread, keep the selected thread and its context visible while editing, require an explicit post action with `ctrl+d`, and preserve the draft when posting fails. Pressing `c` in the thread walkthrough SHALL open the editor for that thread. Comment editors SHALL support killing to the end of the current line with `ctrl+k`, deleting the previous word with `ctrl+w`, moving the cursor to the current line's end with `ctrl+e`, moving to the previous or next word boundary with `ctrl+left` or `ctrl+right`, and moving to the beginning of the current line with `ctrl+a`.

#### Scenario: Open an inline thread comment
- **WHEN** the developer presses `c` while viewing an unresolved review thread
- **THEN** the thread history and context remain visible with an editable comment draft for that thread

#### Scenario: Reply fails
- **WHEN** GitHub rejects a reply
- **THEN** the editor retains the draft, shows the error, and permits retry

#### Scenario: Edit a comment draft with shortcuts
- **WHEN** the developer uses `ctrl+k`, `ctrl+w`, or `ctrl+e` in a comment editor
- **THEN** the editor kills the line remainder, deletes the previous word, or moves to the line end, respectively

#### Scenario: Submit a comment draft
- **WHEN** the developer presses `ctrl+d` in a comment editor
- **THEN** the comment is submitted and `ctrl+s` does not submit it

#### Scenario: Navigate by word or line
- **WHEN** the developer presses `ctrl+left`, `ctrl+right`, or `ctrl+a` in a comment editor
- **THEN** the cursor moves to the previous word boundary, next word boundary, or beginning of the current line, respectively
