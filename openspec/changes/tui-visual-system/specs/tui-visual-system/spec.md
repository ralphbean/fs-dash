## Purpose

Gives the dashboard a consistent visual language that helps developers locate their selection and interpret field, fetch, and diagnostic states at a glance.

## ADDED Requirements

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
The system SHALL account for border and padding width when truncating text and limiting rows. At common terminal sizes, including 80 columns by 24 rows, the active view SHALL fit without unintended line wraps or content lost below the terminal.

#### Scenario: Resize a populated list
- **WHEN** the terminal shrinks while a PR is selected
- **THEN** the selected PR remains visible and its title or metadata is truncated as needed inside the panel

#### Scenario: Very narrow terminal
- **WHEN** there is insufficient room for a full framed layout
- **THEN** the UI uses a compact readable layout and keeps navigation and status accessible

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
