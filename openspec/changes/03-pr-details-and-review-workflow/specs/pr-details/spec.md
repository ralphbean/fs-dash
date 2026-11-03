## Purpose

Shows enough current pull request context to choose the next action without delaying the lightweight list view.

## ADDED Requirements

### Requirement: PR details load progressively
The system SHALL open a details mode for the selected PR and load description, commits, checks, review state, and conversation independently, showing field-level pending, stale, and error states.

#### Scenario: Slow review history
- **WHEN** review history is slow but description and checks are ready
- **THEN** the ready details remain usable and review history shows pending

### Requirement: Detail identity is stable
The system SHALL associate fetched details with repository, PR number, and head revision, and SHALL disregard stale responses after a revision change.

#### Scenario: New commit during load
- **WHEN** a PR head changes before a prior detail response returns
- **THEN** outdated detail cannot appear as current
