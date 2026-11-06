## MODIFIED Requirements

### Requirement: Table review state combines current human and bot reviews
The system SHALL expose separate current review states for human reviewers and bot reviewers, derived from the effective reviews for each author type. Each state SHALL use the existing review-state precedence and freshness behavior, and incomplete review history SHALL remain distinct from a definitive state.

#### Scenario: Current human and bot review states are known
- **WHEN** review history is complete for the current PR revision
- **THEN** the Human review state reflects effective user-authored reviews and the Bot review state reflects effective bot-authored reviews

#### Scenario: A reviewer type has no current reviews
- **WHEN** review history is complete and there are no effective reviews for one reviewer type
- **THEN** that review state shows awaiting while the other reviewer type retains its own current state

#### Scenario: Review history is incomplete
- **WHEN** review history is pending, incomplete, or failed
- **THEN** both source-specific review states remain pending or failed rather than appearing as no review
