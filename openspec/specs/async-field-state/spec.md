# async-field-state Specification

## Purpose

Makes the availability, freshness, and fetch lifecycle of every remote or derived field visible so partial data and failures remain understandable.

## Requirements

### Requirement: Field values expose availability and fetch progress
The system SHALL distinguish an unrequested field, a queued or in-flight fetch, a fresh value, a stale retained value, a retry cooldown, and a terminal failure. A retained value SHALL remain available with its freshness and error status.

#### Scenario: Initial fetch
- **WHEN** a field has been requested but no response has arrived
- **THEN** consumers see no value and an active pending state suitable for a spinner

#### Scenario: Refresh with retained value
- **WHEN** an existing value is being refreshed
- **THEN** consumers see the retained value and an active refresh state

### Requirement: Failed fetches settle visibly
Every fetch attempt SHALL end in success, retry cooldown, cancellation, or terminal failure. A failed list or field fetch SHALL never leave its loading indicator active after the attempt ends. A GitHub rate-limit response SHALL establish one shared cooldown for all GitHub requests in the application, including requests for other repositories and manual refreshes.

#### Scenario: Transient failure
- **WHEN** a request receives a retryable network, rate-limit, or server error
- **THEN** its state shows the error, next retry time, and any retained value while the spinner stops during cooldown

#### Scenario: Terminal failure
- **WHEN** a request cannot be retried without changed credentials, permissions, or input
- **THEN** its state shows a terminal failure and a manual retry path

#### Scenario: Primary GitHub quota is exhausted
- **WHEN** a GitHub response reports that the primary quota is exhausted and provides a reset time
- **THEN** all queued and newly requested GitHub work waits until that reset time before sending another request

#### Scenario: Secondary rate limit provides Retry-After
- **WHEN** a GitHub response reports a secondary rate limit with a Retry-After duration
- **THEN** all GitHub work waits for that duration before sending another request

#### Scenario: Secondary rate limit omits Retry-After
- **WHEN** a GitHub response reports a secondary rate limit without a Retry-After duration or an exhausted primary quota
- **THEN** the application applies a shared cooldown of at least one minute and increases subsequent retry delays with backoff

#### Scenario: Manual refresh during a shared cooldown
- **WHEN** the user requests a manual refresh while a GitHub cooldown is active
- **THEN** the refresh remains queued and sends no GitHub request before the shared cooldown expires

### Requirement: Derived fields preserve dependency uncertainty
A derived field SHALL expose which inputs determine it and SHALL not claim a definitive result when a missing or stale input could change that result. It SHALL recompute when a relevant input version changes.

#### Scenario: Bot approval from incomplete reviews
- **WHEN** review data has not arrived
- **THEN** bot approval is unknown, even if the current partial review slice contains no bot approval

#### Scenario: Bot approval from complete reviews
- **WHEN** complete current review data contains an effective bot approval
- **THEN** bot approval is true and cites that review data version

### Requirement: Stale responses do not overwrite newer state
The system SHALL discard a response whose request generation or PR identity no longer matches the active field request.

#### Scenario: Selection changes during fetch
- **WHEN** a delayed response for an older PR or head SHA arrives after a newer request
- **THEN** it cannot replace the newer field state

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

### Requirement: Background prefetch preserves interactive UI responsiveness
The system SHALL keep spinner animation and visible PR enrichment responsive while background enrichment is queued for other PRs, even when the queue contains many fields.

#### Scenario: Large background queue
- **WHEN** background enrichment is queued for many PRs while a user views the list
- **THEN** queue prioritization does not pause UI updates, and selected or visible PR enrichment remains prioritized

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
