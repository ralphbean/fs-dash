## Context

The thread view currently runs one resolve mutation at a time, waits for the mutation result, and only then reloads the unresolved thread list. The detail model already rejects stale fetch generations and the GitHub client exposes the existing resolve mutation and paginated thread-list readback.

## Goals / Non-Goals

**Goals:**
- Move navigation ahead immediately when a thread is queued for resolution.
- Keep resolution and verification independent per thread so users can queue several.
- Preserve verified readback as the authority for removing a thread permanently.

**Non-Goals:**
- Change GitHub mutation or query APIs.
- Resolve threads automatically or alter unresolved-thread ordering.

## Decisions

- Track pending resolutions by PR and thread ID, and filter those IDs from the current walkthrough. Keep each pending entry until a complete thread-list readback confirms removal, confirms it remains unresolved, or verification fails.
- Allow separate resolve mutations to run through the existing scheduler without using the single-action lock. After each mutation result, force a thread-list refresh and associate the pending entry with that detail generation so older reads cannot settle newer actions.
- Treat an incomplete readback as a verification failure and preserve the last complete thread list. If a complete readback still contains a pending thread, restore it and report that success was not verified. A mutation error still triggers readback, since the persisted state may already have changed.
- Preserve the currently selected next thread by ID when a refreshed list changes around it; use the current index as a fallback if that thread itself disappears.

## Risks / Trade-offs

- [A thread may be hidden while GitHub is slow] → Show a background-resolution status and restore it when verification fails or remains unresolved.
- [Concurrent refreshes can complete out of order] → Use detail generations and each pending resolution's required verification generation.
