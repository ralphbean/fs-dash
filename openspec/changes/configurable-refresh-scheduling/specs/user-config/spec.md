## ADDED Requirements

### Requirement: Refresh scheduling settings are configurable and documented
The system SHALL accept an optional `[refresh]` TOML section with `max_concurrent_requests`, `request_start_interval_ms`, `max_concurrent_detail_requests`, `max_concurrent_activity_requests`, `max_concurrent_offscreen_requests`, `offscreen_detail_interval_ms`, `offscreen_activity_interval_ms`, `queue_batch_size`, and `queue_yield_interval_ms`. When the section or an individual setting is omitted, the system SHALL use the existing behavior's defaults: 2, 150, 2, 5, 1, 500, 750, 64, and 5 respectively. Concurrency values SHALL be integers from 1 through 64; `queue_batch_size` SHALL be an integer from 1 through 4096; request-start and offscreen intervals SHALL be from 0 through 60000 milliseconds; and `queue_yield_interval_ms` SHALL be from 1 through 1000 milliseconds. The `config.example.toml` comments SHALL explain each setting and how the global request limit, per-kind limits, offscreen limits, request pacing, and UI queue yielding interact.

#### Scenario: Existing configuration uses refresh defaults
- **WHEN** the user loads a TOML file without a `[refresh]` section or without one or more refresh settings
- **THEN** each omitted setting uses its documented default and all existing configuration remains valid

#### Scenario: User configures refresh scheduling
- **WHEN** the user supplies valid `[refresh]` values
- **THEN** the application uses those values for the corresponding scheduling controls

#### Scenario: Reject out-of-range refresh settings
- **WHEN** a refresh setting is outside its documented range or is not an integer
- **THEN** configuration loading reports the setting and its valid range

#### Scenario: Example explains setting relationships
- **WHEN** the user reads the refresh section in `config.example.toml`
- **THEN** comments explain that the shared request limit is the overall network ceiling, per-kind and offscreen limits further constrain work, request-start pacing is independent, and queue batch/yield settings affect UI scheduling rather than network concurrency
