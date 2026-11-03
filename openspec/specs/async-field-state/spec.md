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
Every fetch attempt SHALL end in success, retry cooldown, cancellation, or terminal failure. A failed list or field fetch SHALL never leave its loading indicator active after the attempt ends.

#### Scenario: Transient failure
- **WHEN** a request receives a retryable network, rate-limit, or server error
- **THEN** its state shows the error, next retry time, and any retained value while the spinner stops during cooldown

#### Scenario: Terminal failure
- **WHEN** a request cannot be retried without changed credentials, permissions, or input
- **THEN** its state shows a terminal failure and a manual retry path

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
