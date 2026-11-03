## Purpose

Lets a developer inspect recent background fetch and action events in the terminal without losing the current dashboard context.

## ADDED Requirements

### Requirement: In-app log console
The system SHALL provide a toggleable, scrollable console for timestamped debug, info, warning, and error events and SHALL preserve the current selection when it closes.

#### Scenario: Inspect a failed fetch
- **WHEN** a background fetch fails and the user opens the console
- **THEN** the console shows the source, operation, item identity, attempt, and sanitized error

### Requirement: Logs are bounded and safe
The system SHALL bound in-memory log history and SHALL redact credentials and authorization material from visible and persisted diagnostics.

#### Scenario: Long-running session
- **WHEN** event volume exceeds the configured history limit
- **THEN** oldest events are dropped without blocking new fetches or UI rendering
