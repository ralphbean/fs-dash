## ADDED Requirements

### Requirement: Toggle PR table emoji and text labels
The PR list SHALL provide the `e` key to toggle compact status columns between emoji and text representations. The selected mode SHALL be saved between application runs and restored at startup. When no mode has been saved, emoji mode SHALL be active. Keyboard hints and help SHALL identify the key and its action. Toggling SHALL apply to the table in every PR list section and SHALL NOT change the underlying status values. If saving a mode change fails, the selected mode SHALL still apply for the current run and the failure SHALL be reported.

#### Scenario: Start in emoji mode without a saved preference
- **WHEN** the PR list first appears and no display mode has been saved
- **THEN** compact status columns use emoji headers and values

#### Scenario: Restore the saved mode
- **WHEN** the PR list first appears and a display mode has been saved
- **THEN** compact status columns use the saved emoji or text headers and values

#### Scenario: Toggle to explanatory text
- **WHEN** the user presses `e` while viewing the PR list
- **THEN** compact status columns use descriptive text headers and values, the preference is saved, and the keyboard hint reflects the available toggle

#### Scenario: Toggle back to emoji mode
- **WHEN** the user presses `e` while compact status columns use text
- **THEN** compact status columns return to emoji headers and values and the preference is saved

#### Scenario: Preference save fails
- **WHEN** saving the selected display mode fails
- **THEN** the selected mode remains active for the current run and the user sees a save failure

#### Scenario: Mode is consistent across PR list sections
- **WHEN** the user switches between Review, Active, or Snoozed list sections after toggling
- **THEN** the selected emoji or text mode is used in each section
