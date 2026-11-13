# Local Reminders Specification

## Purpose

Lets a developer mark PRs for later and temporarily hide them without changing any remote GitHub state.

## Requirements

### Requirement: Starred PRs persist locally
The system SHALL let users toggle a star on a PR, show a visible star marker, and retain that choice across restarts using the PR's stable source identity.

#### Scenario: Star PR
- **WHEN** the user stars a PR and restarts the app
- **THEN** that PR still has a visible star marker

### Requirement: Snoozed PRs reappear automatically
The system SHALL hide snoozed PRs from normal sections until a chosen wake time, provide a way to inspect snoozed items, and restore them automatically after that time only while they remain in the latest successful open-PR list results. Reminder state SHALL NOT synthesize a PR row when the PR is absent from those results.

#### Scenario: Snooze expires while PR remains open
- **WHEN** the configured wake time passes while the app is running or before the next launch and the current open-PR results still contain the PR
- **THEN** the PR reappears in its applicable section without a remote write

#### Scenario: Manual unsnooze
- **WHEN** a user unsnoozes a PR that remains in the current open-PR results
- **THEN** it immediately returns to the applicable list

#### Scenario: Snoozed PR is absent from open-list results
- **WHEN** a successful refresh no longer returns a snoozed PR
- **THEN** the PR does not appear in the snoozed view or any other list filter, regardless of its wake time
