## ADDED Requirements

### Requirement: Visible PR summaries refresh automatically
The system SHALL automatically refresh selected and visible PR summaries before their freshness deadlines. Automatic refresh work SHALL use the existing priority, request concurrency, offscreen pacing, and UI queue-yield controls. Offscreen enrichment SHALL continue to follow the existing low-priority background policy until it becomes selected or visible.

#### Scenario: Visible summaries approach expiration
- **WHEN** one or more selected or visible PR summaries are 20 seconds from expiration
- **THEN** the system queues their refreshes with selected and visible work ahead of offscreen work

#### Scenario: The selected PR changes during a scheduled refresh
- **WHEN** the user selects another PR while automatic refreshes are queued
- **THEN** newly selected PR work is promoted using the existing priority rules and no duplicate request is issued

#### Scenario: Offscreen summary expires
- **WHEN** an offscreen PR summary expires
- **THEN** it remains eligible for the existing low-priority background refresh and is promoted if the PR becomes selected or visible
