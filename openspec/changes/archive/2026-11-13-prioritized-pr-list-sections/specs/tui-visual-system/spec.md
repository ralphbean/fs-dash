## ADDED Requirements

### Requirement: PR attention sections share an ordered responsive table layout
The PR list SHALL render nonempty sections from Default, Last touched by me, Active Fullsend Agents, and Drafts in that vertical order. Each displayed section SHALL use the existing PR table columns and row styling, with a visible section heading. The sections SHALL form one navigable list, keep the selected PR visible while scrolling, and adapt to terminal height and width without wrapping PR rows or hiding the selected row. A blank line SHALL separate each displayed section after the first.

#### Scenario: Render the four attention sections
- **WHEN** the non-snoozed PR list is displayed
- **THEN** each nonempty section appears in the specified order and each row uses the shared PR table layout

#### Scenario: No rows match a section
- **WHEN** one of the four sections has no PRs
- **THEN** the section heading, table header, and empty-state explanation are omitted

#### Scenario: Separate displayed sections
- **WHEN** more than one section contains PRs
- **THEN** one blank line appears before each displayed section heading after the first

#### Scenario: Selected PR moves between sections
- **WHEN** updated classification data moves the selected PR to another section
- **THEN** the same PR remains selected and visible in the combined list

#### Scenario: Compact terminal displays multiple sections
- **WHEN** terminal height or width is too small to show all section rows at once
- **THEN** the list scrolls as one view, retains the section order, keeps rows on one line, and keeps the selected PR visible
