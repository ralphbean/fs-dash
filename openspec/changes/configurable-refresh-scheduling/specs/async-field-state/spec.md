## MODIFIED Requirements

### Requirement: Large enrichment queues yield to UI updates
The system SHALL process queued field and activity enrichment in batches of at most the configured `queue_batch_size` and SHALL allow rendering and input updates between batches by waiting the configured `queue_yield_interval_ms` before continuing.

#### Scenario: Large prefetch backlog
- **WHEN** background enrichment is queued for many PRs
- **THEN** each UI update processes no more than `queue_batch_size` items and continues remaining work after `queue_yield_interval_ms`, allowing rendering and input updates between batches
