# PR Triage Specification

## Purpose

Turns PR facts into transparent, actionable views so the developer can see review work and currently active Fullsend work separately.

## Requirements

### Requirement: Filterable review section
The system SHALL offer filters for all open PRs, PRs assigned to the current user, PRs requesting the current user's review, and PRs authored by the current user. It SHALL show the active filter and rank rows by an explainable default priority with stable tie breaking.

#### Scenario: Default assigned filter
- **WHEN** the application starts
- **THEN** the assigned-to-me filter is active

#### Scenario: Filtered list count
- **WHEN** a repository list is loaded or filter data changes
- **THEN** the repository summary shows the total open PRs, the number in the active filter, and how many of those remain pending

#### Scenario: Assignment data is prefetched for filtering
- **WHEN** a repository's open PR list loads
- **THEN** the list response includes the viewer and each PR's assignees so confirmed non-matches leave the assigned-to-me view immediately, while incomplete results remain labeled pending

#### Scenario: Review requested filter
- **WHEN** the user selects review requested from me
- **THEN** only PRs with current review requests for that user appear, while incomplete review-request data is labeled pending rather than silently excluded

#### Scenario: Authored by me filter
- **WHEN** the user selects authored by me
- **THEN** only PRs whose author login matches the current viewer appear, while an unavailable viewer or author is labeled pending

### Requirement: Remote Fullsend activity section
The system SHALL place PRs with current, source-backed remote Fullsend agent activity in a separate section and SHALL show unknown, stale, and failed activity detection distinctly from inactive.

#### Scenario: Activity prefetch after list load
- **WHEN** a repository's open PR list loads
- **THEN** the system starts fetching activity for its PRs in list order, with no more than five activity requests in flight across the application

#### Scenario: Selected PR is waiting in activity queue
- **WHEN** the user selects a PR whose activity request is still queued
- **THEN** that PR moves to the front of the waiting activity queue and starts as soon as an activity request slot is available, without issuing a duplicate request

#### Scenario: Selection changes while detail enrichment is queued
- **WHEN** the user moves to another PR while the previous PR has detail fields waiting to fetch
- **THEN** the system drops the previous PR's unstarted detail requests and keeps no more than two detail requests in flight

#### Scenario: Activity query pending
- **WHEN** activity detection has not completed for a PR
- **THEN** the PR remains visible in the review section with an activity pending indicator

#### Scenario: Activity confirmed
- **WHEN** current evidence indicates a remote Fullsend agent is working on the PR
- **THEN** the PR appears in the active-agent section with evidence time and source

### Requirement: Derived review signals
The system SHALL derive bot approval from current effective reviews and SHALL distinguish unknown from false. It SHALL update derived ranking when review, check, or activity inputs change.

#### Scenario: Reviews incomplete
- **WHEN** review history is still fetching
- **THEN** bot approval remains unknown and cannot lower the PR's priority as though no bot approved
