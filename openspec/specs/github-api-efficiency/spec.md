# GitHub API Efficiency Specification

## Purpose

Reduces repeated and bursty GitHub API work while preserving accurate per-PR summaries, visible fetch progress, and useful cached data.

## Requirements

### Requirement: Fetch PR status summaries together
The system SHALL fetch each PR's check, review, review-request, and unresolved review-thread summaries together in one GitHub GraphQL request where those fields are available. It SHALL preserve each summary's independent completeness and meaning when decoding the shared response.

#### Scenario: Combined status query completes
- **WHEN** the combined query for a PR returns check, review, review-request, and review-thread data
- **THEN** the corresponding table summaries are populated from that response without issuing one request per summary

#### Scenario: A combined response contains incomplete connections
- **WHEN** one or more returned connections indicate additional pages or incomplete data
- **THEN** the affected summary remains incomplete while other complete summaries from the same response remain usable

### Requirement: Reuse fresh PR enrichment from a local cache
The system SHALL cache enrichment results across application runs with freshness metadata. A cache entry SHALL be keyed by GitHub host, repository, PR number, head SHA, and summary field family so data from a different PR revision is never treated as the current revision's result.

#### Scenario: Reopen with fresh cached summaries
- **WHEN** the application starts and a matching cached summary is still fresh
- **THEN** the summary is available without an immediate GitHub request

#### Scenario: Reopen with stale cached summaries
- **WHEN** the application starts and a matching cached summary has expired
- **THEN** the stale value remains identifiable while a refresh is scheduled according to the fetch priority rules

#### Scenario: PR head changes
- **WHEN** a PR is observed with a head SHA different from the cached entry
- **THEN** the old entry is not used as a current summary for that PR revision

#### Scenario: No matching cached summary exists
- **WHEN** a requested summary has no matching cache entry
- **THEN** it follows the normal asynchronous fetch lifecycle and remains pending until data arrives
