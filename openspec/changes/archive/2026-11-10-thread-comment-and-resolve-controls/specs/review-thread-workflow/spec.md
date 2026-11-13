## MODIFIED Requirements

### Requirement: Reply editor preserves drafts
The system SHALL provide a multi-line editor for a reply to the selected review thread, keep the selected thread and its context visible while editing, require an explicit post action, and preserve the draft when posting fails. Pressing `c` in the thread walkthrough SHALL open the editor for that thread.

#### Scenario: Open an inline thread comment
- **WHEN** the developer presses `c` while viewing an unresolved review thread
- **THEN** the thread history and context remain visible with an editable comment draft for that thread

#### Scenario: Reply fails
- **WHEN** GitHub rejects a reply
- **THEN** the editor retains the draft, shows the error, and permits retry

## ADDED Requirements

### Requirement: Selected review thread can be resolved
The system SHALL let the developer resolve the selected unresolved review thread with `r` and SHALL verify the persisted resolution state before reporting success.

#### Scenario: Resolve a thread
- **WHEN** the developer presses `r` on an unresolved thread and GitHub confirms it is resolved on readback
- **THEN** the thread is removed from the unresolved walkthrough and the UI reports that resolution was verified

#### Scenario: Resolution cannot be verified
- **WHEN** the resolve request or its readback fails
- **THEN** the UI does not report success, keeps the selected-thread context available, and explains that the result needs verification
