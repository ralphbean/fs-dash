## Purpose

Makes GitHub API capacity visible so users can anticipate rate-limit blocks and understand when their available quota will reset.

## ADDED Requirements

### Requirement: Display the latest known GitHub API quota
The application SHALL capture GitHub's primary API quota limit, remaining amount, usage, and reset time from responses it already receives. It SHALL display the latest known quota and when that information was observed, without making a separate polling request solely to read quota status.

#### Scenario: Capture quota metadata from an API response
- **WHEN** a GitHub API response includes primary quota metadata
- **THEN** the application records the limit, remaining amount, usage, reset time, and observation time

#### Scenario: Show current quota status
- **WHEN** the application has observed quota metadata
- **THEN** the dashboard shows the amount used, amount remaining, total limit, and reset time in a compact status indicator

#### Scenario: Warn when quota is low
- **WHEN** the latest observed remaining quota is below 20% of its total limit
- **THEN** the indicator highlights the remaining quota and reset time as a warning

#### Scenario: Show exhausted quota
- **WHEN** the latest observed remaining quota is zero
- **THEN** the indicator identifies the quota as exhausted and shows the reset time

#### Scenario: Quota information is unavailable or old
- **WHEN** the application has no quota observation or its last observation is more than five minutes old
- **THEN** the dashboard shows an unknown or stale status with the last observation time and does not present missing data as current quota
