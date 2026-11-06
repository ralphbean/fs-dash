## Purpose

Defines repository contribution practices that keep implementation planning and commit history consistent and reviewable.

## ADDED Requirements

### Requirement: Repository changes use OpenSpec
Agents SHALL use OpenSpec to plan and track repository changes, and SHALL install the required OpenSpec skills or plugins when they are unavailable.

#### Scenario: Agent begins a repository change
- **WHEN** an agent starts a change in this repository
- **THEN** it follows the applicable OpenSpec workflow and installs missing required skills or plugins before proceeding

### Requirement: Commit subjects use Conventional Commits
Every commit subject SHALL follow the Conventional Commits form `<type>[optional scope][!]: <description>`, and the repository commit-message check SHALL reject subjects that do not conform.

#### Scenario: Valid commit subject
- **WHEN** a commit subject uses a type, optional scope and breaking marker, followed by a non-empty description
- **THEN** the commit-message check accepts it

#### Scenario: Invalid commit subject
- **WHEN** a commit subject does not use the required form
- **THEN** the commit-message check rejects the commit and explains the expected format
