## Purpose

Gives an optional concise summary of recent pull request activity with clear freshness and provenance so developers can navigate long histories faster.

## ADDED Requirements

### Requirement: Optional summary generation
The system SHALL generate a summary only when enabled and sufficient current PR data is available. Summary loading or failure SHALL not block factual details or actions.

#### Scenario: Disabled generation
- **WHEN** summary generation is disabled
- **THEN** no model subprocess runs and details remain usable

### Requirement: State-keyed summary cache
The system SHALL reuse a cached summary only when its input fingerprint matches the current PR revision and relevant review, comment, and check state.

#### Scenario: New review comment
- **WHEN** a new review comment changes the input fingerprint
- **THEN** the old summary is labeled stale and a new summary can be generated

### Requirement: Summary provenance and failure state
The UI SHALL identify generated text as a summary, show source time and model configuration, and expose generation errors without presenting stale text as current.

#### Scenario: Generation fails
- **WHEN** the local model command fails
- **THEN** details show a summary error and the last successful summary, if any, is labeled stale
