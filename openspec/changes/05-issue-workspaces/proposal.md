## Why

The same attention workflow should eventually cover GitHub and Jira issues. A separate change keeps the first release focused on PRs while proving the list and detail interfaces work across item types.

## What Changes

- Add GitHub issue and Jira issue list modes with source-specific filters and ranking inputs.
- Add issue details using the shared navigation, field-state, diagnostics, and command-binding contracts.
- Preserve source-specific identity, links, permissions, and partial-data states.

## Capabilities

### New Capabilities

- `issue-workspaces`: GitHub and Jira issue list and details modes.

### Modified Capabilities

None.

## Impact

Adds GitHub issue and Jira adapters and issue-specific domain views. Jira authentication and project selection are configurable and independent from GitHub. No issue mutations are included in this slice.
