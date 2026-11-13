## ADDED Requirements

### Requirement: Loaded list data refreshes before it expires
The system SHALL schedule a refresh for each loaded list section and its selected or visible enrichment values 20 seconds before their recorded freshness deadline while the list view is active. A scheduled refresh SHALL preserve the last successful value while work is queued or in flight and SHALL use the normal retry and shared GitHub cooldown behavior. The system SHALL stop scheduling list refreshes outside the list view and reevaluate freshness when the list view becomes active again.

#### Scenario: List section approaches its freshness deadline
- **WHEN** a loaded list section is 20 seconds from its freshness deadline while the list view is active
- **THEN** the section refresh is queued automatically before the value becomes stale

#### Scenario: Visible enrichment approaches its freshness deadline
- **WHEN** a selected or visible PR enrichment value is 20 seconds from its freshness deadline while the list view is active
- **THEN** the value is refreshed automatically while its last successful value remains available

#### Scenario: Refresh fails or is rate limited
- **WHEN** an automatic list or enrichment refresh fails or receives a GitHub cooldown
- **THEN** the existing failure, retry, and shared cooldown behavior applies and retained values remain identifiable

#### Scenario: List view becomes active with due values
- **WHEN** the user returns to the list view and a loaded list or selected or visible enrichment value is due for refresh
- **THEN** the system schedules the refresh immediately without requiring another key press
