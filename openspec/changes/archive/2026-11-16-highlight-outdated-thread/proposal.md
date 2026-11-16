## Why

The thread view already identifies outdated review comments, but the label blends into the surrounding location details. A clear warning marker makes it easier to notice that the comment refers to historical code context.

## What Changes

- Highlight an outdated thread status near the top of the thread view with the warning color and a visible warning symbol.
- Keep the word “Outdated” in the label so its meaning remains clear when color is disabled.

## Capabilities

### New Capabilities

### Modified Capabilities
- `review-thread-workflow`: specify a prominent warning treatment for outdated threads.

## Impact

- Affects the review-thread TUI presentation in `internal/ui/details.go`.
- Uses the existing semantic warning style and the existing outdated-thread data.
