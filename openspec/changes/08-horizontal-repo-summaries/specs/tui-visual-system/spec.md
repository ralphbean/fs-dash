## MODIFIED Requirements

### Requirement: Layout fits the terminal
The system SHALL account for border and padding width when truncating text and limiting rows. At common terminal sizes, including 80 columns by 24 rows, the active view SHALL fit without unintended line wraps or content lost below the terminal. Repository status summaries SHALL share one horizontal row above the PR table, regardless of the number of configured repositories.

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
