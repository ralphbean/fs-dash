## MODIFIED Requirements

### Requirement: PR list uses a justified table across repositories
The list SHALL show rows from all configured repositories in one priority-sorted table, while retaining separate Review and Active sections and the existing repository summaries. The columns SHALL appear in this order: Star, PR link, Title, Author, Assignees, Updated, Checks, Review, and Threads. The Star column header SHALL be empty. The Snoozed view SHALL add Wake as the final column. Updated SHALL show the relative time of the newest visible item in the PR's GitHub timeline, including commits, comments, reviews, and timeline events. Changes to GitHub's PR `updatedAt` that do not create a visible timeline item SHALL NOT change the Updated value.

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
- **THEN** its assignees are listed and truncated to fit, Updated uses the relative time of the newest visible PR timeline item, Checks shows a compact state, Review shows one combined current state, and Threads shows the unresolved count

#### Scenario: Updated follows visible timeline activity
- **WHEN** a new commit, comment, review, or other visible timeline event is added to a PR
- **THEN** Updated reflects the relative time of that newest timeline item

#### Scenario: Updated ignores non-timeline changes
- **WHEN** GitHub's PR `updatedAt` changes without a new visible timeline item
- **THEN** Updated continues to reflect the newest visible timeline item

#### Scenario: Timeline activity loads without blocking the list
- **WHEN** the PR list has loaded but timeline activity has not
- **THEN** the list remains usable and the Updated cell indicates that its value is pending

#### Scenario: Timeline timestamp unavailable
- **WHEN** the timeline lookup fails or GitHub does not provide a timeline timestamp
- **THEN** Updated indicates that the timeline time is unavailable instead of showing GitHub's generic PR `updatedAt`

#### Scenario: Star and selection are distinct
- **WHEN** a PR row is starred or selected
- **THEN** a star emoji appears in the leftmost Star column only for starred PRs, and selection highlights the full row

#### Scenario: Selected row remains identifiable without color
- **WHEN** color output is disabled and a PR row is selected
- **THEN** the selected row has a plain text focus marker and the view emits no ANSI styling

#### Scenario: Active and snoozed rows share the table layout
- **WHEN** the Active or Snoozed section is shown
- **THEN** its rows use the same column alignment, with Wake present only in the Snoozed view
