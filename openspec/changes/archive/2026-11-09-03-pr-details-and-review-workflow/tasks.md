## 1. Progressive PR details

- [x] 1.1 Add detail read ports and GitHub adapters for description, commits, checks, reviews, conversation, and review threads.
- [x] 1.2 Cache each detail group by source/repo/PR/head and reject responses from older heads.
- [x] 1.3 Build details mode with back navigation, independent field states, scrolling, and visible action keys.
- [x] 1.4 Verify that a slow or failed group leaves ready groups usable and selection/back navigation intact.

## 2. Review thread walkthrough

- [x] 2.1 Fetch unresolved thread IDs/history and selected-thread diff context, including outdated-location handling.
- [x] 2.2 Build `T` walkthrough with stable thread order, history metadata, next/previous navigation, and empty state.
- [x] 2.3 Add multiline reply editor with explicit submit/cancel and draft retention on failed post.
- [x] 2.4 Implement reply mutation and post-write thread verification; test timeout and ambiguous outcome behavior.

## 3. PR actions

- [x] 3.1 Add PR-level comment editor, mutation, retained draft, and refreshed-conversation verification.
- [x] 3.2 Add merge/queue capability discovery, pre-action state validation, confirmation, submission, and state verification.
- [x] 3.3 Add close confirmation, cancel path, pre-action validation, and verified state refresh.
- [x] 3.4 Exercise read-only details against a live PR and exercise writes with a controlled test repository or fake provider before enabling live use.
