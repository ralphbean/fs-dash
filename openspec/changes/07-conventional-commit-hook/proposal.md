## Why

Consistent Conventional Commit subjects make the repository history easier to scan and automate. A commit-message hook prevents nonconforming subjects from being recorded.

## What Changes

- Require Conventional Commit subjects in repository contributor guidance.
- Add a `commit-msg` pre-commit hook that rejects invalid subjects and install it in the current checkout.

## Capabilities

### New Capabilities

- `contributor-workflow`: Repository contribution conventions and local commit-message enforcement.

### Modified Capabilities

None.

## Impact

Adds repository guidance, a pre-commit configuration, and a shell validator. Commits without a Conventional Commit subject are rejected by the installed local hook.
