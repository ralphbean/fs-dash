## Purpose

Extends the dashboard's attention workflow to GitHub and Jira issues while retaining source identity and the shared partial-data behavior.

## ADDED Requirements

### Requirement: Separate issue list modes
The system SHALL offer GitHub issue and Jira issue lists with visible source, scope, filters, priority attributes, and per-section loading and failure states.

#### Scenario: Jira unavailable
- **WHEN** Jira authentication or network access fails
- **THEN** the Jira list shows its error while GitHub PR and issue lists remain usable

### Requirement: Issue details and navigation
The system SHALL open source-specific details for a selected issue and return to the same list selection. Shared selected-item URL command bindings SHALL work for both issue sources.

#### Scenario: Open Jira issue
- **WHEN** a Jira issue is selected and the user opens details
- **THEN** the UI shows its key, title, status, assignee, links, and partial-data states without treating it as a GitHub issue

### Requirement: Source-scoped identity
The system SHALL distinguish items by source and stable source ID so identical numeric IDs or titles never share cache or local state.

#### Scenario: Colliding IDs
- **WHEN** a GitHub issue and Jira issue have matching visible numbers
- **THEN** their details and command URLs remain independent
