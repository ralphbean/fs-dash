## Why

Resolving a review thread currently waits for GitHub and a fresh thread-list read before moving to the next thread. That pause slows down reviewing a queue of already-addressed threads.

## What Changes

- Advance immediately to the next unresolved thread when `r` is pressed.
- Run the resolve mutation and fresh thread-list verification in the background, allowing the user to queue resolutions for subsequent threads.
- Keep verification authoritative: confirmed threads stay removed, while unresolved or unverifiable threads return to the walkthrough with a status message.

## Capabilities

### New Capabilities

### Modified Capabilities
- `review-thread-workflow`: make thread resolution non-blocking while retaining readback verification and recovery.

## Impact

- Affects asynchronous thread actions and thread-list projection in `internal/ui/details.go` and `internal/ui/model.go`.
- Uses the existing GitHub resolve mutation and detail refresh APIs.
