## MODIFIED Requirements

### Requirement: Command bindings receive selected-item data safely
The system SHALL allow configured hotkeys to launch an executable with explicit argument placeholders for the selected item’s URL and metadata. A binding MAY provide an optional human-readable summary, which the in-TUI help SHALL display beside its hotkey instead of the executable name. For existing bindings without a summary, help SHALL continue to display the executable name. The system SHALL avoid shell interpretation unless the user explicitly configures a shell executable.

#### Scenario: Open selected PR
- **WHEN** the user presses `o` on a PR row
- **THEN** the configured opener receives that PR URL as one argument and execution status is reported

#### Scenario: Custom Codex launcher
- **WHEN** a configured binding references the selected PR URL and a working directory
- **THEN** those values are passed as separate validated arguments and the requested working directory is used

#### Scenario: Display a configured binding summary
- **WHEN** a configured binding has a summary and the user opens in-TUI help
- **THEN** the help lists that summary beside the hotkey instead of the executable name

#### Scenario: Keep help useful for bindings without a summary
- **WHEN** an existing configured binding has no summary and the user opens in-TUI help
- **THEN** the help lists the executable name beside the hotkey
