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
The system SHALL retain the selected PR by stable identity when rows refresh or reorder, and SHALL identify the active mode and available keys. In PR list sections, `gg` SHALL select the first row and `G` SHALL select the last row; a boundary jump SHALL keep the selected row visible and SHALL do nothing when the section has no rows. The list footer and keyboard help SHALL show these shortcuts.

#### Scenario: Rows reorder after refresh
- **WHEN** new data changes row order
- **THEN** selection remains on the same PR if it is still present

#### Scenario: Jump to the first PR
- **WHEN** the user presses `gg` while viewing a PR list section with rows
- **THEN** the first row is selected and remains visible

#### Scenario: Jump to the last PR
- **WHEN** the user presses `G` while viewing a PR list section with rows
- **THEN** the last row is selected and remains visible

#### Scenario: Boundary jump on an empty section
- **WHEN** the user presses `gg` or `G` while the current PR list section has no rows
- **THEN** the selection remains empty

#### Scenario: List boundary shortcuts are visible
- **WHEN** the PR list footer or keyboard help is shown
- **THEN** it identifies `gg` as top and `G` as bottom

### Requirement: List filters contain only current open PRs
The system SHALL build every PR list section and filter only from PRs returned by the latest successful open-PR list results. A PR absent from those results SHALL NOT be synthesized from reminder state, cached enrichment, or prior list contents.

#### Scenario: A merged PR is removed by refresh
- **WHEN** a successful open-PR refresh no longer returns a previously listed PR
- **THEN** that PR is absent from every list filter and section

#### Scenario: Saved reminder exists for a removed PR
- **WHEN** a PR is absent from the latest open-PR results but local star or snooze state still exists for it
- **THEN** the local state does not create a list row or make the PR match any filter
