# user-config Specification

## Purpose

Provides predictable user preferences and selected-item shortcuts while keeping local command execution explicit and configurable.

## Requirements

### Requirement: Configuration loads from the user config directory
The system SHALL load preferences from `~/.config/fs-dash/`, validate them at startup, and show actionable errors for invalid entries. Missing configuration SHALL use documented defaults.

#### Scenario: First launch
- **WHEN** no configuration file exists
- **THEN** the app starts with default list behavior and `o` bound to open the selected PR URL with `xdg-open` on Linux

### Requirement: Repository scope is explicit
The system SHALL allow a configured set of GitHub repositories and SHALL show which repositories the current list covers. With no configured repositories, it SHALL detect the current directory's GitHub remote or request configuration instead of silently listing an arbitrary organization.

#### Scenario: No project repository
- **WHEN** the app starts outside a GitHub checkout with no repository configured
- **THEN** it shows an actionable repository setup state and sends no ambiguous global PR query

### Requirement: Command bindings receive selected-item data safely
The system SHALL allow configured hotkeys to launch an executable with explicit argument placeholders for the selected item's URL and metadata. It SHALL avoid shell interpretation unless the user explicitly configures a shell executable.

#### Scenario: Open selected PR
- **WHEN** the user presses `o` on a PR row
- **THEN** the configured opener receives that PR URL as one argument and execution status is reported

#### Scenario: Custom Codex launcher
- **WHEN** a configured binding references the selected PR URL and a working directory
- **THEN** those values are passed as separate validated arguments and the requested working directory is used
