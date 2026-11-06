## MODIFIED Requirements

### Requirement: Remote Fullsend activity section
The system SHALL place PRs with current, source-backed remote Fullsend agent activity in a separate section and SHALL show unknown, stale, and failed activity detection distinctly from inactive. For each active PR, the section SHALL show a process-tree style child entry for every currently active Fullsend agent run found in the activity evidence, including a readable agent name, a link to that run's workflow, and its elapsed running time. Child rows SHALL align under the PR identity column after the Star column. Elapsed time SHALL update while the active-agent section is visible. If the activity evidence does not cover the complete run history, the section SHALL identify the run list as potentially incomplete. Activity requests SHALL not exceed `max_concurrent_activity_requests` or the shared `max_concurrent_requests` limit. All offscreen activity and detail work together SHALL not exceed `max_concurrent_offscreen_requests`; offscreen activity requests SHALL be paced by `offscreen_activity_interval_ms`.

#### Scenario: Activity prefetch after list load
- **WHEN** a repository's open PR list loads
- **THEN** the system starts activity fetches for selected and visible PRs first, then processes offscreen PRs in a low-priority, paced queue subject to the configured concurrency and pacing limits

#### Scenario: Selected PR is waiting in activity queue
- **WHEN** the user selects a PR whose activity request is queued
- **THEN** that PR moves to the front of the waiting activity queue and starts as soon as an activity request slot is available, without issuing a duplicate request

#### Scenario: Selection changes while detail enrichment is queued
- **WHEN** the user moves to another PR while the previous PR has detail fields waiting to fetch
- **THEN** the system drops the previous PR's unstarted detail requests and keeps no more than `max_concurrent_detail_requests` detail requests in flight

#### Scenario: Activity query pending
- **WHEN** activity detection has not completed for a PR, including an offscreen PR waiting in the low-priority queue
- **THEN** the PR remains visible in the review section with an activity pending indicator rather than being treated as inactive

#### Scenario: Activity confirmed
- **WHEN** current evidence indicates a remote Fullsend agent is working on the PR
- **THEN** the PR appears in the active-agent section with evidence time and source

#### Scenario: Active PR shows agent run children
- **WHEN** current status-comment evidence identifies one or more non-terminal, non-stale Fullsend agent runs for a PR
- **THEN** the active-agent section shows the PR followed by a nested process-tree entry for each active run, with the agent's readable name, a workflow-run link, and elapsed time since that run started

#### Scenario: Active run elapsed time advances
- **WHEN** an active-agent PR remains visible while one of its runs continues
- **THEN** that run's displayed elapsed time advances without requiring another GitHub request

#### Scenario: Completed or stale run is not shown as active
- **WHEN** a run's latest status evidence is terminal or older than the existing activity freshness threshold
- **THEN** the run is omitted from the active-run children and is not presented as currently running

#### Scenario: Workflow link is unavailable
- **WHEN** a status comment identifies an active run but has no usable workflow-run URL
- **THEN** the agent name and elapsed time remain visible and the run is shown without a misleading link

#### Scenario: Activity comment history is incomplete
- **WHEN** the activity query indicates that older comments were omitted from the fetched page
- **THEN** the section shows active runs found in the fetched evidence and visibly indicates that earlier run history may be incomplete

### Requirement: Table triage summaries load in the background
The system SHALL populate check, review, and unresolved review-thread summaries for PR rows asynchronously with no more than `max_concurrent_detail_requests` detail requests in flight. It SHALL obtain the check, review, review-request, and review-thread summaries for each PR from one combined query where supported. The list SHALL render from the initial PR data without waiting for these summaries, and selected and visible rows SHALL be prioritized ahead of offscreen rows. Offscreen detail requests SHALL be paced by `offscreen_detail_interval_ms`, and together with offscreen activity requests SHALL not exceed `max_concurrent_offscreen_requests`.

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
