## Purpose

Supports common pull request actions from details mode while making consequential writes explicit and their outcomes verifiable.

## ADDED Requirements

### Requirement: PR-level comments
The system SHALL let a user draft and explicitly post a PR-level comment, retaining the draft on failure and refreshing the conversation after success.

#### Scenario: Post PR comment
- **WHEN** a comment request succeeds
- **THEN** the new comment appears in refreshed conversation data

### Requirement: Merge or queue uses current GitHub capability
The system SHALL offer merge or merge-queue actions according to current GitHub repository policy, show the chosen action before submission, and refresh PR state after success.

#### Scenario: Queue required
- **WHEN** the repository requires a merge queue and the user confirms the queue action
- **THEN** the app submits the queue action and reports its verified queue state

### Requirement: Close requires confirmation
The system SHALL require explicit confirmation before closing a PR and SHALL show a clear cancel path.

#### Scenario: Cancel close
- **WHEN** the user cancels the close prompt
- **THEN** no GitHub mutation is sent
