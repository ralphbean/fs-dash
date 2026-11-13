## Why

The current PR list mixes work that may need attention with PRs waiting on other people, PRs being handled by Fullsend agents, and drafts. Grouping the current filtered results into ordered sections makes those distinctions visible and lets the user scan the likely work first.

## What Changes

- Partition the current PR filter into four mutually exclusive sections, in display order: Default, Last touched by me, Active Fullsend Agents, and Drafts.
- Apply classification precedence from bottom to top: drafts go to Drafts; otherwise PRs with active Fullsend work go to Active Fullsend Agents; otherwise PRs whose newest non-bot timeline activity was by the current user go to Last touched by me; all remaining PRs go to Default.
- Obtain draft state and timeline actor information needed to classify rows. Keep rows visible in Default while classification data is pending or unavailable, with the existing pending or failure treatment.
- Preserve current PR filtering, row ordering within each section, selection, search, snooze behavior, and active-run details.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `pr-triage`: Define the four mutually exclusive PR sections, their precedence, and actor-aware classification that ignores bots.
- `tui-visual-system`: Define the vertical section order and consistent PR table presentation across the sections.

## Impact

Changes the PR list projection and rendering, the lightweight PR list data needed for draft classification, and timeline enrichment needed to identify the latest non-bot actor. Search and selection continue to operate over the current filtered results in rendered order.
