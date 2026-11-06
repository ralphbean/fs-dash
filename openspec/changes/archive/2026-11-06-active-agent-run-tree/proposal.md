## Why

The Active Fullsend Agents section identifies which PRs have agent activity, but it hides the individual runs when more than one agent is working on a PR. Showing each active run beneath its PR makes concurrent work, its destination, and its elapsed time visible without opening the PR first.

## What Changes

- Expand each active PR into a compact process-tree style list of its currently active Fullsend agent runs.
- Show a readable agent name, a link to the corresponding workflow run, and a live relative elapsed-time display for each run.
- Indicate when the fetched activity comments do not cover the complete run history.
- Preserve the existing PR-level activity section and its loading, stale, unknown, and failure distinctions.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `pr-triage`: Active Fullsend PR entries include their current agent runs and expose each run's identity, workflow link, and elapsed time.

## Impact

- Updates GitHub activity parsing and the Active Fullsend Agents list projection/rendering.
- Uses status-comment evidence already fetched for PR activity, including per-run status markers, timestamps, agent labels, and workflow links; no new external service is required.
- Adds periodic UI refresh behavior for elapsed-time labels while active runs are visible.
