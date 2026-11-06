## MODIFIED Requirements

### Requirement: Filterable review section
The system SHALL offer filters for all open PRs, PRs assigned to the current user, PRs requesting the current user's review, and PRs authored by the current user. It SHALL show the active filter and rank rows by an explainable default priority with stable tie breaking.

#### Scenario: Default authored filter
- **WHEN** the application starts
- **THEN** the authored-by-me filter is active

#### Scenario: Filtered list count
- **WHEN** a repository list is loaded or filter data changes
- **THEN** the repository summary shows the total open PRs, the number in the active filter, and how many of those remain pending

#### Scenario: Assignment data is prefetched for filtering
- **WHEN** a repository's open PR list loads
- **THEN** the list response includes the viewer and each PR's assignees so confirmed non-matches leave the assigned-to-me view immediately, while incomplete results remain labeled pending

#### Scenario: Review requested filter
- **WHEN** the user selects review requested from me
- **THEN** only PRs with current review requests for that user appear, while incomplete review-request data is labeled pending rather than silently excluded

#### Scenario: Authored by me filter
- **WHEN** the user selects authored by me
- **THEN** only PRs whose author login matches the current viewer appear, while an unavailable viewer or author is labeled pending

## ADDED Requirements

### Requirement: Review filters cycle in a predictable order
The system SHALL cycle through filters in this order: all open PRs, authored by me, assigned to me, review requested from me, starred PRs, and snoozed PRs. The startup filter SHALL be authored by me. The user SHALL use the filter cycle to reach starred and snoozed PRs rather than dedicated filter keys.

#### Scenario: Cycle review filters
- **WHEN** the user advances through the review filters
- **THEN** the sequence is all open PRs, authored by me, assigned to me, review requested from me, starred PRs, snoozed PRs, then all open PRs again

#### Scenario: Show starred PRs
- **WHEN** the active filter is starred PRs
- **THEN** only locally starred, non-snoozed PRs appear

#### Scenario: Show snoozed PRs
- **WHEN** the active filter is snoozed PRs
- **THEN** snoozed PRs appear with their wake times and the list provides an unsnooze action

#### Scenario: Filter navigation help
- **WHEN** the user views the list or keyboard help
- **THEN** filter navigation is shown as a cycle and no dedicated key is shown for starred or snoozed filters
