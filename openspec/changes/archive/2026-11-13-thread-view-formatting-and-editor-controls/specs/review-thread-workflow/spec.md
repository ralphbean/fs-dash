## MODIFIED Requirements

### Requirement: Unresolved thread walkthrough
Pressing `T` in PR details SHALL open a navigable sequence of unresolved review threads. Each thread SHALL show file path, current or outdated diff context, comments in chronological order, authors, and timestamps. Diff context SHALL color added, removed, and hunk-header lines distinctly when color is enabled. Each comment SHALL appear inside its own visible frame.

#### Scenario: Thread on an outdated diff
- **WHEN** a thread's original code location no longer exists in the current diff
- **THEN** the UI labels the thread outdated and shows available original context rather than a misleading current location

#### Scenario: Diff lines are colorized
- **WHEN** a thread view displays diff context
- **THEN** added lines, removed lines, and hunk headers use distinct colors while preserving their text and prefixes when color is disabled

#### Scenario: Comments have separate frames
- **WHEN** a thread view displays multiple comments
- **THEN** each comment's author, timestamp, and body appear within its own visible frame

### Requirement: Reply editor preserves drafts
The system SHALL provide a multi-line editor for a reply to the selected review thread, keep the selected thread and its context visible while editing, require an explicit post action with `ctrl+d`, and preserve the draft when posting fails. Pressing `c` in the thread walkthrough SHALL open the editor for that thread. Comment editors SHALL support killing to the end of the current line with `ctrl+k`, deleting the previous word with `ctrl+w`, and moving the cursor to the current line's end with `ctrl+e`.

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
