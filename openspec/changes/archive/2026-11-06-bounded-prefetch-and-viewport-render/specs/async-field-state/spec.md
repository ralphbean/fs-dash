## ADDED Requirements

### Requirement: Large enrichment queues yield to UI updates
The system SHALL process queued field and activity enrichment in bounded batches and allow rendering and input updates between batches.

#### Scenario: Large prefetch backlog
- **WHEN** background enrichment is queued for many PRs
- **THEN** each UI update processes only a bounded amount of queue work and continues remaining work in a later update

### Requirement: List rendering is limited to the visible viewport
The system SHALL format only PR rows that are visible within the current terminal viewport.

#### Scenario: List contains more rows than fit on screen
- **WHEN** the projected PR list is taller than the available terminal height
- **THEN** only the rows selected for the visible viewport are rendered
