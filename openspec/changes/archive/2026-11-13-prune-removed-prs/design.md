## Context

See `proposal.md` for the user report. The current projection appends synthetic `domain.PR` values from saved snooze entries when their IDs are absent from loaded open-PR sections. Those values have no author, head revision, or current GitHub data, but are still passed through normal filters.

## Goals / Non-Goals

**Goals:** Keep every list query and section bounded by the latest successful open-PR results, so removed PRs cannot appear through fallback state.

**Non-Goals:** Delete saved stars or snooze records, change snooze expiration, or change the list section's retained-data behavior while a refresh is pending or fails.

## Decisions

Remove the projection fallback that constructs placeholder PRs from reminder entries. Continue applying reminder state to each PR actually returned by the open-PR list. A successful list response already replaces the section's PR slice, so no extra PR membership cache or cleanup API is needed.

Keep saved reminder records when a PR is absent. This avoids changing local persistence semantics and allows the record to remain attached if the PR is listed again later, while ensuring the record alone cannot create a list row.

## Risks / Trade-offs

- Snoozed PRs that close or merge disappear from the snoozed view → this matches the open-PR-only list contract; local reminder data remains stored.
- A failed refresh cannot confirm that a PR has closed → retain the last successful list and show the existing error state until a successful response arrives.
