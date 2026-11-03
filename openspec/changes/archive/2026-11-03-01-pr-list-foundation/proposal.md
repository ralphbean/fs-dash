## Why

Developers need a fast, trustworthy view of open Fullsend project pull requests. The earlier dashboard could keep spinning after a failed list fetch, so this first slice establishes explicit state and diagnostics before adding richer triage.

## What Changes

- Create the Go/Bubble Tea application with a two-mode navigation shell and an immediately useful open PR list.
- Model remote and derived fields with explicit availability, refresh, retry, and terminal-failure states.
- Fetch lightweight GitHub list data asynchronously through a narrow provider interface; keep expensive enrichment out of the initial list request.
- Show loading animation, partial data, field-level errors, and an in-app log console.
- Load user preferences and URL command bindings from `~/.config/fs-dash/`, including `o` to open the selected PR in a browser.

## Capabilities

### New Capabilities

- `async-field-state`: Observable lifecycle for remote and derived data, including retries and stale results.
- `pr-list`: Initial GitHub pull request listing and navigation.
- `diagnostics`: In-app structured log viewing and fetch failure diagnosis.
- `user-config`: Configuration and safe selected-item command bindings.

### Modified Capabilities

None.

## Impact

New Go module, Bubble Tea UI, GitHub adapter, configuration loader, logging, and testable ports. GitHub authentication uses the user's existing `gh` session initially. No remote mutation is included.
