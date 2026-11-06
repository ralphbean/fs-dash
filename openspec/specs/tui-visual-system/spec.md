# TUI Visual System Specification

## Purpose

Gives the dashboard a consistent visual language that helps developers locate their selection and interpret field, fetch, and diagnostic states at a glance.

## Requirements

### Requirement: Start in a clean fullscreen terminal view
The application SHALL use the terminal's alternate screen while the dashboard is running, so launch begins with a clean view and exiting restores the prior terminal contents.

#### Scenario: Launch and exit the dashboard
- **WHEN** the user starts the dashboard in a terminal and then exits
- **THEN** the dashboard starts on a clean fullscreen view and the prior terminal contents are restored

### Requirement: Semantic color communicates status consistently
The system SHALL use distinct, consistent colors for focus, neutral text, links, active fetching, current success, stale values, retry cooldown, warnings, and terminal failures across list, details, help, and logs. A selected row SHALL retain the visible status of its fields.

#### Scenario: Mixed fetch states
- **WHEN** one section is fetching, another is in cooldown, and a retained value is stale
- **THEN** each state has a distinct color treatment and a visible text or symbol label

#### Scenario: Selection on an error row
- **WHEN** a row with a failed field becomes selected
- **THEN** focus styling identifies the row while the failed field still reads as an error

### Requirement: Framed views establish hierarchy
The system SHALL use borders or dividers to distinguish the app header, PR sections, selected-item context, details content, log console, and key hints. Panel titles SHALL identify the current mode and scope.

#### Scenario: Open logs from a PR list
- **WHEN** the user opens the log console
- **THEN** a clearly framed log view replaces the list and closing it restores the prior selection

### Requirement: Layout fits the terminal
The system SHALL account for border and padding width when truncating text and limiting rows. At common terminal sizes, including 80 columns by 24 rows, the active view SHALL fit without unintended line wraps or content lost below the terminal. Repository status summaries SHALL share one horizontal row above the PR table, regardless of the number of configured repositories. When quota information is available, the active dashboard view SHALL keep a compact quota indicator accessible without disrupting the PR table or key hints.

#### Scenario: Resize a populated list
- **WHEN** the terminal shrinks while a PR is selected
- **THEN** the selected PR remains visible and its title or metadata is truncated as needed inside the panel

#### Scenario: Very narrow terminal
- **WHEN** there is insufficient room for a full framed layout
- **THEN** the UI uses a compact readable layout and keeps navigation and status accessible

#### Scenario: Multiple repository summaries use one row
- **WHEN** the PR list has status summaries for multiple configured repositories
- **THEN** the summaries appear left-to-right together on one row above the PR table

#### Scenario: Horizontal summaries exceed frame width
- **WHEN** the combined repository summary row exceeds the frame's inner width
- **THEN** it remains on one line and is truncated within the frame

#### Scenario: Quota status remains accessible in a compact layout
- **WHEN** the PR list is shown in a narrow terminal and quota information is available
- **THEN** the quota indicator remains visible in a compact form while the PR table and keyboard hints remain accessible

### Requirement: Color is optional and never sole meaning
The system SHALL provide dark and light palettes, an automatic color-capability mode, and a no-color option. It SHALL honor `NO_COLOR`; status and focus SHALL remain understandable through words, symbols, and borders when color is disabled.

#### Scenario: No-color environment
- **WHEN** `NO_COLOR` is set
- **THEN** the UI emits no color styling and still distinguishes focus, pending, cooldown, stale, and failure states

### Requirement: Log severity is scannable
The log console SHALL give debug, info, warning, and error levels distinct semantic treatments while preserving readable timestamps, operation, and message.

#### Scenario: Error after retry warning
- **WHEN** the console contains a retry warning and a terminal error
- **THEN** the two events are visually distinct and their level labels remain visible

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

### Requirement: PR table adapts to narrow terminals
When the terminal cannot fit every column, the UI SHALL hide lower-priority columns while keeping the PR link and title visible. Rows SHALL remain single-line and the selected row SHALL remain visually identifiable.

#### Scenario: Narrow terminal hides lower-priority columns
- **WHEN** the terminal width is insufficient for the full PR table
- **THEN** lower-priority columns are hidden before the PR link or title, without horizontal scrolling or wrapped rows
