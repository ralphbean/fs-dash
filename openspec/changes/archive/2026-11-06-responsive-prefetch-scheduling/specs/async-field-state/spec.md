## ADDED Requirements

### Requirement: Background prefetch preserves interactive UI responsiveness
The system SHALL keep spinner animation and visible PR enrichment responsive while background enrichment is queued for other PRs, even when the queue contains many fields.

#### Scenario: Large background queue
- **WHEN** background enrichment is queued for many PRs while a user views the list
- **THEN** queue prioritization does not pause UI updates, and selected or visible PR enrichment remains prioritized
