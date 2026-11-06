## ADDED Requirements

### Requirement: Enrichment cache persistence does not block UI progress
The system SHALL keep rendering, spinner animation, and input updates responsive while persisting enrichment cache snapshots. Cache snapshots SHALL be written in order, and changes made during a write SHALL be persisted in a later snapshot. Cache write failures SHALL remain visible to the user.

#### Scenario: Cache save is slow
- **WHEN** a cache save takes longer than one spinner interval
- **THEN** UI updates and spinner animation continue while the save is in progress

#### Scenario: Cache changes during a save
- **WHEN** enrichment results update the cache while an earlier snapshot is being saved
- **THEN** the earlier write completes without overlapping a second write, and a later write persists the newest cache state

#### Scenario: Cache save fails
- **WHEN** persisting a cache snapshot fails
- **THEN** the application displays its local cache warning without blocking UI progress
