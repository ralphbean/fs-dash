## Why

A plain PR list still leaves the developer deciding where to look first. This slice adds priority views, Fullsend activity detection, and local reminders without making slow enrichment block the list.

## What Changes

- Add review, assignment, and ownership filters plus prioritized PR slices.
- Add a second section for PRs with evidence that a remote Fullsend agent is currently working, with explicit unknown and stale states.
- Derive review and bot approval signals from underlying review data.
- Add local stars and time-based snoozes, with automatic reappearance and persistence.

## Capabilities

### New Capabilities

- `pr-triage`: Filters, ranking, and remote agent activity section.
- `local-reminders`: Stars and snoozes on PRs.

### Modified Capabilities

None.

## Impact

Extends the shared field scheduler, GitHub provider, PR list presentation, and local storage. The exact Fullsend activity signal is an adapter policy and must include source evidence and freshness; the UI does not infer activity solely from a bot author.
