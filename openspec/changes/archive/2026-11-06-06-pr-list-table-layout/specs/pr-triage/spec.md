## ADDED Requirements

### Requirement: Table triage summaries load in the background
The system SHALL populate check, review, and unresolved review-thread summaries for PR rows asynchronously with bounded concurrency. The list SHALL render from the initial PR data without waiting for these summaries, and visible rows SHALL be prioritized in the background queue.

#### Scenario: List appears before summaries are ready
- **WHEN** the initial PR list loads while check, review, or thread summaries are unavailable
- **THEN** the table renders immediately with pending indicators and summary fetches continue in the background

#### Scenario: Visible row is waiting for summaries
- **WHEN** a visible PR has queued summary fetches
- **THEN** its fetches are prioritized ahead of off-screen PRs without duplicating requests

#### Scenario: Summary fetch finishes or fails
- **WHEN** a summary fetch completes or reaches a terminal error
- **THEN** the corresponding table indicator updates to its current result or failure state, and unknown data is never shown as a confirmed negative

### Requirement: Table review state combines current human and bot reviews
The system SHALL provide one overall current review state derived from effective human and bot reviews, while keeping incomplete review history distinct from a definitive state.

#### Scenario: Current review state is known
- **WHEN** review history is complete for the current PR revision
- **THEN** the table review summary reflects the overall current review state

#### Scenario: Review history is incomplete
- **WHEN** review history is pending, incomplete, or failed
- **THEN** the review summary remains pending or failed rather than appearing as no review
