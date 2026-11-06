## ADDED Requirements

### Requirement: Directional search navigates matching PR rows
The system SHALL provide case-insensitive substring search over PR titles, author logins, and assignee logins in the active PR list view. `/` SHALL start forward search and `?` SHALL start reverse search. The search prompt SHALL update the selected row to the first matching row in the chosen direction as the query is entered. `n` SHALL move to the next matching row in that direction. Search SHALL wrap at either end of the active list. Search SHALL follow the list's current filter and rendered section order, and SHALL not alter the list contents.

#### Scenario: Start forward search
- **WHEN** the user presses `/` in a PR list view
- **THEN** the search prompt opens in the forward direction

#### Scenario: Start reverse search
- **WHEN** the user presses `?` in a PR list view
- **THEN** the search prompt opens in the reverse direction

#### Scenario: Find a row by title, author, or assignee
- **WHEN** the entered query is a case-insensitive substring of a PR title, author login, or assignee login
- **THEN** the matching PR is selected without removing other rows from the list

#### Scenario: Repeat the search
- **WHEN** the user presses `n` after accepting or exiting a non-empty search query
- **THEN** selection moves to the next matching PR in the search direction

#### Scenario: Wrap forward search
- **WHEN** forward search reaches the end of the active list while another matching PR exists before the current position
- **THEN** search continues from the beginning of the active list

#### Scenario: Wrap reverse search
- **WHEN** reverse search reaches the beginning of the active list while another matching PR exists after the current position
- **THEN** search continues from the end of the active list

#### Scenario: No row matches
- **WHEN** a non-empty query has no matching PR in the active list
- **THEN** selection remains unchanged and the user sees that there are no matches

#### Scenario: Exit search and keep the last hit
- **WHEN** the user presses `Esc` while entering a query
- **THEN** the prompt closes and the most recently selected matching PR remains selected

#### Scenario: Search fields are incomplete
- **WHEN** no loaded title, author, or assignee value matches and one or more assignee lists are incomplete
- **THEN** the user is told that the search result may be incomplete

#### Scenario: Open help from the list
- **WHEN** the user presses `F1` in a PR list view
- **THEN** keyboard help opens
