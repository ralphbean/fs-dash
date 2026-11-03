## Why

PR details can be long, and a concise account of recent changes would help the developer choose a next action. Summaries should stay optional and must not delay factual PR data.

## What Changes

- Optionally run a local Codex session using a configured Luna model to summarize recent PR updates.
- Cache summaries against a fingerprint of the PR state and recompute only when relevant inputs change.
- Show summary freshness, provenance, loading, and errors in details mode.

## Capabilities

### New Capabilities

- `pr-activity-summaries`: Optional generated summaries and state-keyed caching.

### Modified Capabilities

None.

## Impact

Adds a bounded local subprocess adapter and summary cache. No summary is treated as the source of truth for merge readiness or review state.
