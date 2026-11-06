## MODIFIED Requirements

### Requirement: Keyboard navigation is stable during updates
The system SHALL retain the selected PR by stable identity when rows refresh or reorder, and SHALL identify the active mode and available keys. In PR list sections, `gg` SHALL select the first row and `G` SHALL select the last row; a boundary jump SHALL keep the selected row visible and SHALL do nothing when the section has no rows.

#### Scenario: Rows reorder after refresh
- **WHEN** new data changes row order
- **THEN** selection remains on the same PR if it is still present

#### Scenario: Jump to the first PR
- **WHEN** the user presses `gg` while viewing a PR list section with rows
- **THEN** the first row is selected and remains visible

#### Scenario: Jump to the last PR
- **WHEN** the user presses `G` while viewing a PR list section with rows
- **THEN** the last row is selected and remains visible

#### Scenario: Boundary jump on an empty section
- **WHEN** the user presses `gg` or `G` while the current PR list section has no rows
- **THEN** the selection remains empty
