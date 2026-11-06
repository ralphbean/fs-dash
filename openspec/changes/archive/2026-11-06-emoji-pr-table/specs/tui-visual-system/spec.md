## MODIFIED Requirements

### Requirement: PR list uses a justified table across repositories
The list SHALL show rows from all configured repositories in one priority-sorted table, while retaining separate Review and Active sections and the existing repository summaries. The columns SHALL appear in this order: Star, PR link, Title, Author, Assignees, Updated, Checks, Review, and Threads. The Star column header SHALL be empty. The Snoozed view SHALL add Wake as the final column. Updated SHALL show the relative time of the newest visible item in the PR's GitHub timeline, including commits, comments, reviews, and timeline events. Changes to GitHub's PR `updatedAt` that do not create a visible timeline item SHALL NOT change the Updated value. The table SHALL restore the user's saved emoji or text mode; when no mode has been saved, it SHALL start in emoji mode. In emoji mode, Checks, Review, and Threads SHALL use compact emoji headers and state representations; text mode SHALL retain their descriptive headers and values. The two modes SHALL represent the same underlying values, and useful check and thread counts SHALL remain visible in both modes. Column widths SHALL be calculated from the active mode and SHALL NOT reserve space for text labels while emoji mode is active.

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

#### Scenario: Emoji mode uses compact labels and state marks
- **WHEN** the table is in emoji mode
- **THEN** the Checks, Review, and Threads headers use 🧪, 👀, and 🧵 respectively; pending, stale, failed, and unknown states use ⏳, 🕰️, ❌, and ❔ respectively, while loading uses the current frame of the animated spinner; check outcomes use ✅ for pass, ❌ plus the failure count for failures, 🔄 plus the count for running checks, ⚠️ for partial results, and ➖ for no checks; review outcomes use ✅ for approved, 🛑 for changes requested, 💬 for commented, and 💤 for awaiting review; and thread results show the unresolved count, retaining `+` for incomplete counts

#### Scenario: Text mode retains descriptive cell values
- **WHEN** the table is in text mode
- **THEN** Checks, Review, and Threads show their descriptive headers and the existing textual values for pending, loading, stale, failed, unknown, check outcomes, review outcomes, and unresolved thread counts

#### Scenario: Saved display mode is used by the table
- **WHEN** a saved emoji or text mode is restored
- **THEN** the table renders all compact status headers and cells in that mode

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

#### Scenario: Compact mode recalculates table widths
- **WHEN** emoji mode or text mode is active
- **THEN** column widths fit the selected headers and cell representations without wrapping rows or misaligning columns

#### Scenario: Emoji mode frees width used by text labels
- **WHEN** the table switches from text mode to emoji mode
- **THEN** Checks, Review, and Threads widths no longer reserve space for their text headers or state labels, while counts remain visible
