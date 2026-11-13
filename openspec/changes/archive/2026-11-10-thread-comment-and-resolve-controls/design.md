## Context

The thread walker currently routes `r` to a separate full-screen reply editor and has no resolve action. The details view already has a separate PR-level comment action on `c`; that binding remains scoped to details mode.

## Goals / Non-Goals

**Goals:**
- Keep the active thread visible while composing a reply from `c`.
- Add a GitHub resolve mutation whose result is confirmed by a follow-up thread-context read.
- Refresh unresolved threads after a verified resolution.

**Non-Goals:**
- Editing comments that have already been posted to GitHub.
- Adding an unresolve action or changing PR-level comment behavior.

## Decisions

- Reuse the existing multiline draft buffer and reply submission/readback flow. Render the thread history with a composer section while the editor is active, so navigation context stays on screen and failed posts retain the draft.
- Add `ResolveReviewThread` to the action port and implement it with GitHub GraphQL's `resolveReviewThread` mutation. Validate the mutation payload, then query the thread context again and require `isResolved` before returning success.
- After successful resolution, refresh the unresolved thread list. Keep the existing ordered selection behavior so the next thread becomes active at the same index when possible.
- Use `R` for refreshing a missing or failed thread list, keeping lowercase `r` dedicated to resolving a selected thread.

## Risks / Trade-offs

- A resolve request can succeed while its readback fails. The UI must report an uncertain outcome, preserve the thread context, and require a refresh before claiming success.
- Inline composition uses vertical space otherwise available to thread history; cap the composer display to the viewport while retaining the full draft buffer.
