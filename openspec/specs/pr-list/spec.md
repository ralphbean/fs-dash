# pr-list Specification

## Purpose

Provides a fast terminal list of open GitHub pull requests that a developer can navigate while slower attributes continue to load.

## Requirements

### Requirement: Open PR list appears from lightweight data
The system SHALL display configured repositories' open PR identity, title, author, updated time, and URL without waiting for review threads, check details, or other expensive enrichment.

#### Scenario: Partial list readiness
- **WHEN** the lightweight request finishes while detail fields remain pending
- **THEN** PR rows are navigable and pending columns show their own state

### Requirement: List loading and failure are section scoped
The system SHALL show loading, empty, retry cooldown, and error states for each list section independently and allow a manual refresh.

#### Scenario: Initial list request fails
- **WHEN** GitHub returns an error before any rows arrive
- **THEN** the affected section stops spinning and shows the error and retry option

### Requirement: Keyboard navigation is stable during updates
The system SHALL retain the selected PR by stable identity when rows refresh or reorder, and SHALL identify the active mode and available keys.

#### Scenario: Rows reorder after refresh
- **WHEN** new data changes row order
- **THEN** selection remains on the same PR if it is still present
