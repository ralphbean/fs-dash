## ADDED Requirements

### Requirement: PR list uses a justified table across repositories
The list SHALL show rows from all configured repositories in one priority-sorted table, while retaining separate Review and Active sections and the existing repository summaries. The columns SHALL appear in this order: Star, PR link, Title, Author, Assignees, Updated, Checks, Review, and Threads. The Star column header SHALL be empty. The Snoozed view SHALL add Wake as the final column.

#### Scenario: PR identity is a clickable repository link
- **WHEN** a PR row is rendered
- **THEN** its first identity column displays `<org>/<repo>#<PR number>` as a clickable link to that PR

#### Scenario: Table aligns row data
- **WHEN** a table row is rendered
- **THEN** text columns are left-aligned, counts and relative times are right-aligned, and Star and compact status columns are centered

#### Scenario: Star column has no header label
- **WHEN** a table header is rendered
- **THEN** the header cell above the Star column is empty

#### Scenario: Table shows row summaries
- **WHEN** a PR row is rendered
- **THEN** its assignees are listed and truncated to fit, Updated uses relative time, Checks shows a compact state, Review shows one combined current state, and Threads shows the unresolved count

#### Scenario: Star and selection are distinct
- **WHEN** a PR row is starred or selected
- **THEN** a star emoji appears in the leftmost Star column only for starred PRs, and selection highlights the full row

#### Scenario: Selected row remains identifiable without color
- **WHEN** color output is disabled and a PR row is selected
- **THEN** the selected row has a plain text focus marker and the view emits no ANSI styling

#### Scenario: Active and snoozed rows share the table layout
- **WHEN** the Active or Snoozed section is shown
- **THEN** its rows use the same column alignment, with Wake present only in the Snoozed view

### Requirement: PR table adapts to narrow terminals
When the terminal cannot fit every column, the UI SHALL hide lower-priority columns while keeping the PR link and title visible. Rows SHALL remain single-line and the selected row SHALL remain visually identifiable.

#### Scenario: Narrow terminal hides lower-priority columns
- **WHEN** the terminal width is insufficient for the full PR table
- **THEN** lower-priority columns are hidden before the PR link or title, without horizontal scrolling or wrapped rows
