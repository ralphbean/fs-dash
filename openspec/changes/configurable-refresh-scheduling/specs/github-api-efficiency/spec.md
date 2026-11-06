## ADDED Requirements

### Requirement: Shared GitHub request concurrency and pacing are configurable
The system SHALL apply `max_concurrent_requests` as the shared upper bound for GitHub list, detail, and activity requests. It SHALL apply `request_start_interval_ms` as the minimum local interval between request starts when the value is greater than zero. A value of zero SHALL disable only this local pacing; the system SHALL continue to honor shared primary and secondary GitHub cooldowns.

#### Scenario: Request classes share the configured concurrency ceiling
- **WHEN** list, detail, and activity requests are ready at the same time
- **THEN** no more than `max_concurrent_requests` GitHub requests execute concurrently across those request classes

#### Scenario: Request starts use the configured interval
- **WHEN** multiple GitHub requests are waiting and `request_start_interval_ms` is greater than zero
- **THEN** successive requests start no sooner than the configured interval apart

#### Scenario: Disable local start pacing
- **WHEN** `request_start_interval_ms` is zero
- **THEN** requests may start without the local interval delay while the global concurrency limit and any server-provided cooldown remain in force
