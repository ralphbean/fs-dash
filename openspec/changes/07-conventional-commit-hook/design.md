## Context

See proposal.md for motivation. `AGENTS.md` now specifies OpenSpec and Conventional Commit practices. The repository had no commit-message hook, while `pre-commit` is available in the current checkout.

## Goals / Non-Goals

**Goals:** Enforce commit subject format in the current checkout and provide versioned guidance and hook configuration for future checkouts.

**Non-Goals:** Validate commit body or footer structure, require a fixed set of commit types, or change existing commits.

## Decisions

1. Use a local `commit-msg` hook in `.pre-commit-config.yaml`. This keeps enforcement declarative and lets contributors install it with the standard `pre-commit install --hook-type commit-msg` command.
2. Use a small POSIX shell validator instead of a remote hook repository. It avoids a runtime dependency and accepts lowercase types, optional scopes, an optional breaking marker, and a non-empty description.
3. Add the installation command to `AGENTS.md` so agents can activate the hook in new checkouts.

## Risks / Trade-offs

- A new checkout may not have the local hook installed → contributor guidance includes the installation command.
- The validator checks the subject line only → commit body and footer remain unrestricted, as scoped in the proposal.

## Migration Plan

Install the hook in a checkout with `pre-commit install --hook-type commit-msg`. No existing commit is rewritten.
