## MODIFIED Requirements

### Requirement: Remote Fullsend activity section
The system SHALL place PRs with current, source-backed remote Fullsend agent activity in a separate section and SHALL show unknown, stale, and failed activity detection distinctly from inactive.

#### Scenario: Activity prefetch after list load
- **WHEN** a repository's open PR list loads
- **THEN** the system starts activity fetches for selected and visible PRs first, then processes offscreen PRs in a low-priority, paced queue, with no more than five activity requests in flight across the application

#### Scenario: Selected PR is waiting in activity queue
- **WHEN** the user selects a PR whose activity request is queued
- **THEN** that PR moves to the front of the waiting activity queue and starts as soon as an activity request slot is available, without issuing a duplicate request

#### Scenario: Selection changes while detail enrichment is queued
- **WHEN** the user moves to another PR while the previous PR has detail fields waiting to fetch
- **THEN** the system drops the previous PR's unstarted detail requests and keeps no more than two detail requests in flight

#### Scenario: Activity query pending
- **WHEN** activity detection has not completed for a PR, including an offscreen PR waiting in the low-priority queue
- **THEN** the PR remains visible in the review section with an activity pending indicator rather than being treated as inactive

#### Scenario: Activity confirmed
- **WHEN** current evidence indicates a remote Fullsend agent is working on the PR
- **THEN** the PR appears in the active-agent section with evidence time and source

### Requirement: Table triage summaries load in the background
The system SHALL populate check, review, and unresolved review-thread summaries for PR rows asynchronously with bounded concurrency. It SHALL obtain the check, review, review-request, and review-thread summaries for each PR from one combined query where supported. The list SHALL render from the initial PR data without waiting for these summaries, and selected and visible rows SHALL be prioritized ahead of offscreen rows, which SHALL be deferred to a paced, low-priority queue.

#### Scenario: List appears before summaries are ready
- **WHEN** the initial PR list loads while check, review, or thread summaries are unavailable
- **THEN** the table renders immediately with pending indicators and combined summary fetches continue in the background

#### Scenario: Visible row is waiting for summaries
- **WHEN** a visible PR has queued summary fetches
- **THEN** its fetches are prioritized ahead of offscreen PRs without duplicating requests

#### Scenario: Offscreen summaries are deferred
- **WHEN** selected or visible PR summaries are still queued or in flight
- **THEN** offscreen PR summaries remain queued for low-priority processing and are not treated as confirmed negative results

#### Scenario: Summary fetch finishes or fails
- **WHEN** a summary fetch completes or reaches a terminal error
- **THEN** the corresponding table indicator updates to its current result or failure state, and unknown data is never shown as a confirmed negative

### Requirement: Table review state combines current human and bot reviews
The system SHALL provide one overall current review state derived from effective human and bot reviews, while keeping incomplete review history distinct from a definitive state.

#### Scenario: Current review state is known
- **WHEN** review history is complete for the current PR revision
- **THEN** the table review summary reflects the overall current review state

#### Scenario: Review history is incomplete
- **WHEN** review history is pending, incomplete, or failed
- **THEN** the review summary remains pending or failed rather than appearing as no review
