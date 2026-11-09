## Context

The list and triage changes provide stable PR identity and progressive fields. This change adds heavy detail reads and GitHub writes. See proposal.md and the three capability specs.

## Goals / Non-Goals

**Goals:** Make review context accurate at a specific head revision; preserve drafts through failures; verify writes against GitHub.

**Non-Goals:** A full terminal diff editor, automatic thread resolution, or bypassing repository merge policy.

## Decisions

1. **Details are separate field groups.** Description, reviews, review threads, checks, commits, and conversation have independent requests and freshness. Cache by source host/repository/PR/head SHA, with per-group invalidation. A monolithic detail fetch was rejected because the slowest field would block all context.
2. **Thread walkthrough uses stable GitHub thread IDs.** Fetch unresolved thread IDs and comments, then diff context for the selected thread. Retain original path and line metadata; when the current diff no longer contains that hunk, label it outdated and show available original context. Navigate thread order by file/path/line with a stable ID tie break. Preserve position when new data arrives.
3. **Editors own drafts until verified.** The micro editor supports multiline text, cancel, and explicit submit. A posting error keeps the draft and action context. After a successful mutation response, refetch the relevant thread or conversation and match the new comment ID; only then display completion. This separates server acceptance from visible persistence.
4. **Write commands are explicit use cases.** GitHub mutation port exposes reply, PR comment, merge or queue, and close operations. The UI presents exact target and action, checks current capability, asks for confirmation on close and merge/queue, and handles changed remote state. Merge queue support is discovered from current repository/branch policy rather than guessed from a keypress.
5. **Actions do not trust stale readiness.** Before merge/queue/close, refresh or validate remote PR state and report a changed head, policy, or permission error. UI state is refreshed after every successful mutation. Command availability is a hint, not authorization.

## Risks / Trade-offs

- Large diffs and threads can stall rendering → fetch selected context, cap displayed size with an explicit expansion path, and keep scrolling incremental.
- GitHub comments can move with force pushes → keep stable thread IDs and show outdated context honestly.
- Merge queue APIs and permissions vary → capability probe and informative error; never silently fall back to direct merge.
- Duplicate submissions after timeout → use returned IDs when possible, re-query before retry, and keep draft until outcome is clear.

## Migration Plan

No data migration. Release behind details-mode keybindings; a failed adapter leaves the existing list usable.
