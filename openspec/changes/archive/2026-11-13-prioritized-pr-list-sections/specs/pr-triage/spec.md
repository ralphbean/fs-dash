## ADDED Requirements

### Requirement: PR attention sections classify filtered rows by precedence
The system SHALL partition the non-snoozed PRs matching the active list filter into exactly one of four sections, displayed in this order: Default, Last touched by me, Active Fullsend Agents, and Drafts. Classification SHALL apply precedence from the bottom section upward: a draft PR belongs in Drafts; otherwise a PR with confirmed current Fullsend agent activity belongs in Active Fullsend Agents; otherwise a PR whose newest non-bot timeline activity was performed by the current viewer belongs in Last touched by me; all remaining PRs belong in Default. The existing snoozed view SHALL remain separate from these sections. Rows within each section SHALL retain their existing priority ordering, and selection SHALL remain attached to the same PR when it moves between sections.

#### Scenario: Unclassified PR appears in Default
- **WHEN** a PR matches the active filter but is neither a draft, confirmed active-agent PR, nor last touched by the current viewer
- **THEN** it appears in Default

#### Scenario: Current viewer's latest human activity is grouped separately
- **WHEN** the newest non-bot timeline activity on a PR was performed by the current viewer
- **THEN** the PR appears in Last touched by me unless it is a draft or has confirmed current Fullsend agent activity

#### Scenario: Bot activity does not change the last human actor
- **WHEN** a bot creates newer timeline activity after the current viewer's most recent human activity
- **THEN** the bot activity is ignored for section classification and the PR appears in Last touched by me unless a lower section takes precedence

#### Scenario: Another person's activity remains in Default
- **WHEN** the newest non-bot timeline activity was performed by someone other than the current viewer
- **THEN** the PR appears in Default unless it is a draft or has confirmed current Fullsend agent activity

#### Scenario: Active agent takes precedence over last touched
- **WHEN** the current viewer performed the newest non-bot activity and current Fullsend agent activity is confirmed
- **THEN** the PR appears only in Active Fullsend Agents

#### Scenario: Draft takes precedence over all other sections
- **WHEN** a PR is a draft and also has current Fullsend agent activity or was last touched by the current viewer
- **THEN** the PR appears only in Drafts

#### Scenario: Timeline actor data is pending or incomplete
- **WHEN** the system cannot yet determine the newest non-bot timeline actor because actor data is pending, unavailable, or incomplete
- **THEN** the PR remains visible in Default with its classification state identified as pending or unavailable, unless draft or confirmed active-agent status places it in a lower section

#### Scenario: Unattributed subscription marker does not block classification
- **WHEN** a timeline contains a `SubscribedEvent` without an actor or timestamp
- **THEN** the marker is ignored and the latest attributable non-bot timeline activity determines the section

#### Scenario: Section classification follows the active list filter
- **WHEN** the user changes the active PR filter
- **THEN** each section is rebuilt only from PRs matching that filter, and the existing filter and total-count behavior remains available across all four sections
