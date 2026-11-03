## Purpose

Lets a developer inspect and respond to unresolved GitHub review threads in sequence with relevant code and conversation context.

## ADDED Requirements

### Requirement: Unresolved thread walkthrough
Pressing `T` in PR details SHALL open a navigable sequence of unresolved review threads. Each thread SHALL show file path, current or outdated diff context, comments in chronological order, authors, and timestamps.

#### Scenario: Thread on an outdated diff
- **WHEN** a thread's original code location no longer exists in the current diff
- **THEN** the UI labels the thread outdated and shows available original context rather than a misleading current location

### Requirement: Reply editor preserves drafts
The system SHALL provide a multi-line micro editor for a thread reply, require an explicit post action, and preserve the draft when posting fails.

#### Scenario: Reply fails
- **WHEN** GitHub rejects a reply
- **THEN** the editor retains the draft, shows the error, and permits retry

### Requirement: Posted replies are verified
After a reply succeeds, the system SHALL refresh that thread and display the persisted comment before marking the action complete.

#### Scenario: Reply accepted
- **WHEN** a reply request succeeds
- **THEN** the updated thread history includes the posted comment or reports verification failure
