## MODIFIED Requirements

### Requirement: Unresolved thread walkthrough
Pressing `T` in PR details SHALL open a navigable sequence of unresolved review threads. Each thread SHALL show file path, current or outdated diff context, comments in chronological order, authors, and timestamps. When a thread is outdated, the UI SHALL display a warning-colored `⚠ Outdated` marker near the top of the thread view beside its location information. The marker SHALL remain understandable when color is disabled. Diff context SHALL color added, removed, and hunk-header lines distinctly when color is enabled. Each comment SHALL appear inside its own visible frame. In the thread/diff view, `gg` SHALL scroll to the top of the current thread content and `G` SHALL scroll to its bottom without changing the selected thread. The thread footer and keyboard help SHALL show these shortcuts.

#### Scenario: Thread on an outdated diff
- **WHEN** a selected thread is marked outdated
- **THEN** the thread view shows a prominent warning-colored `⚠ Outdated` marker near the top, alongside the original line information and available historical context

#### Scenario: Current thread has no outdated marker
- **WHEN** a selected thread is current
- **THEN** the thread view does not show the outdated marker

#### Scenario: Diff lines are colorized
- **WHEN** a thread view displays diff context
- **THEN** added lines, removed lines, and hunk headers use distinct colors while preserving their text and prefixes when color is disabled

#### Scenario: Comments have separate frames
- **WHEN** a thread view displays multiple comments
- **THEN** each comment's author, timestamp, and body appear within its own visible frame

#### Scenario: Jump to the top of thread content
- **WHEN** the user presses `gg` in the thread/diff view
- **THEN** the current thread content scrolls to its top and the selected thread does not change

#### Scenario: Jump to the bottom of thread content
- **WHEN** the user presses `G` in the thread/diff view
- **THEN** the current thread content scrolls to its bottom and the selected thread does not change
