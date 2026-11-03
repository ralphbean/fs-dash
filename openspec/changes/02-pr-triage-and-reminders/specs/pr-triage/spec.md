## Purpose

Turns PR facts into transparent, actionable views so the developer can see review work and currently active Fullsend work separately.

## ADDED Requirements

### Requirement: Filterable review section
The system SHALL offer filters for all open PRs, PRs assigned to the current user, and PRs requesting the current user's review. It SHALL show the active filter and rank rows by an explainable default priority with stable tie breaking.

#### Scenario: Review requested filter
- **WHEN** the user selects review requested from me
- **THEN** only PRs with current review requests for that user appear, while incomplete review-request data is labeled pending rather than silently excluded

### Requirement: Remote Fullsend activity section
The system SHALL place PRs with current, source-backed remote Fullsend agent activity in a separate section and SHALL show unknown, stale, and failed activity detection distinctly from inactive.

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
