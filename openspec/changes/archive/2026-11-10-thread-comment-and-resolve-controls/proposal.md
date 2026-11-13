## Why

The review-thread workflow currently puts replies on `r`, while `c` only starts a PR-level comment from the details screen. Making thread comments and resolution direct, easy-to-find actions will reduce key confusion and let developers compose a reply without losing the thread context.

## What Changes

- Bind `c` in the thread walkthrough to an inline comment composer for the selected thread.
- Bind `r` in the thread walkthrough to resolve the selected thread, then verify the persisted state with GitHub.
- Keep the current `c` PR-level comment action on the PR details screen.
- Preserve the draft and selected-thread context if posting or resolution verification fails.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `review-thread-workflow`: define thread comment and resolve bindings, in-place editing, and verified resolution.

## Impact

- `internal/ui/details.go` and thread workflow help text in `internal/ui/model.go`, `internal/ui/triage.go`, and `README.md`.
- `internal/ports/ports.go`, `internal/domain/details.go`, and `internal/github/details.go` for the resolve operation and readback.
- UI and GitHub adapter behavior; no new dependencies.
