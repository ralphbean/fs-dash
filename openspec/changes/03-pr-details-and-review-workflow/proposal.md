## Why

Once a PR needs attention, the developer should be able to understand its state and work through unresolved review threads without leaving the terminal. Review comments and merge or close actions need clear context and confirmation.

## What Changes

- Add a PR details mode with lazily loaded review, check, commit, and conversation data.
- Add `T` to step through unresolved review threads with diff context, author and timestamp history, and a micro editor for replies.
- Add PR-level comments, merge or merge-queue actions, and confirmed close.
- Refresh authoritative GitHub state after successful writes and show failures without losing drafts.

## Capabilities

### New Capabilities

- `pr-details`: Detailed PR state and navigation.
- `review-thread-workflow`: Unresolved thread walkthrough and replies.
- `pr-actions`: PR comments, merge or queue, and close.

### Modified Capabilities

None.

## Impact

Adds GitHub detail and mutation ports, diff rendering, editors, and action confirmations. A supported GitHub merge queue is used only when available; actions are never inferred successful from a command exit alone.
