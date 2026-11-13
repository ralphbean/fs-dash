## ADDED Requirements

### Requirement: List filters contain only current open PRs
The system SHALL build every PR list section and filter only from PRs returned by the latest successful open-PR list results. A PR absent from those results SHALL NOT be synthesized from reminder state, cached enrichment, or prior list contents.

#### Scenario: A merged PR is removed by refresh
- **WHEN** a successful open-PR refresh no longer returns a previously listed PR
- **THEN** that PR is absent from every list filter and section

#### Scenario: Saved reminder exists for a removed PR
- **WHEN** a PR is absent from the latest open-PR results but local star or snooze state still exists for it
- **THEN** the local state does not create a list row or make the PR match any filter
