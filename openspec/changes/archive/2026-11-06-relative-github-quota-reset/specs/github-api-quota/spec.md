## MODIFIED Requirements

### Requirement: Display the latest known GitHub API quota
The application SHALL capture GitHub's primary API quota limit, remaining amount, usage, reset time, and observation time from responses it already receives. It SHALL display the latest known quota without making a separate polling request solely to read quota status. The reset indicator SHALL show a relative countdown. Within the same quota resource and reset window, the displayed used amount SHALL NOT decrease and the remaining amount SHALL NOT increase when responses arrive out of order; an observation for a later reset window SHALL replace the prior window's value.

#### Scenario: Capture quota metadata from an API response
- **WHEN** a GitHub API response includes primary quota metadata
- **THEN** the application records the limit, remaining amount, usage, reset time, resource, and observation time

#### Scenario: Show current quota status
- **WHEN** the application has observed quota metadata
- **THEN** the dashboard shows the amount used, amount remaining, total limit, and relative reset countdown in a compact status indicator

#### Scenario: Show reset countdown in readable units
- **WHEN** the reset time is in the future
- **THEN** the dashboard shows the remaining duration as `reset in N seconds`, `reset in N minutes`, `reset in N hours`, or `reset in N days`, rounding up to a whole unit and using singular wording for one

#### Scenario: Reset time has passed
- **WHEN** the reset timestamp is at or before the current time
- **THEN** the dashboard shows `reset now`

#### Scenario: Avoid moving quota backward on a late response
- **WHEN** a response for the current quota resource and reset window arrives with a lower used amount or higher remaining amount than the displayed observation
- **THEN** the dashboard keeps the displayed observation

#### Scenario: Accept the next quota window
- **WHEN** a response reports a later reset time than the displayed observation
- **THEN** the dashboard replaces the previous window's quota with the new observation

#### Scenario: Warn when quota is low
- **WHEN** the latest observed remaining quota is below 20% of its total limit
- **THEN** the indicator highlights the remaining quota and relative reset countdown as a warning

#### Scenario: Show exhausted quota
- **WHEN** the latest observed remaining quota is zero
- **THEN** the indicator identifies the quota as exhausted and shows the relative reset countdown

#### Scenario: Quota information is unavailable or old
- **WHEN** the application has no quota observation or its last observation is more than five minutes old
- **THEN** the dashboard shows an unknown or stale status with the last observation time and does not present missing data as current quota
