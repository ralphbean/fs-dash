## Context

The repository contains only OpenSpec planning files. The first implementation must create a runnable Go TUI and a reliable boundary between GitHub I/O and presentation. See proposal.md and the four capability specs for behavior.

## Goals / Non-Goals

**Goals:** Keep the UI responsive under slow or failed requests; make every asynchronous state explainable; allow adapters and ranking rules to change independently.

**Non-Goals:** A generic event bus, a persistent remote-data database, and a comprehensive GitHub mirror.

## Decisions

1. **Ports around domain use cases.** Put source-neutral item identity, `Field[T]`, fetch state, and domain projections in internal domain packages. Define small ports for PR listing, optional enrichment, clock, config, logging, and command execution. Keep `gh`/GitHub decoding and Bubble Tea rendering in adapters. An all-purpose repository interface was considered but hides which operations are slow and makes partial data hard to express.
2. **Separate value from request lifecycle.** A field has optional value plus provenance (`source`, `fetchedAt`, source version) and a state such as unrequested, queued, fetching, ready, cooldown, or terminal failure. A stale value can coexist with fetching or error. Each request carries a generation token; completion is accepted only for the matching identity and generation. Derived fields subscribe to dependency versions and use three-valued logic where an unknown input could change a boolean. A single nullable value was rejected because it confuses absent, false, loading, and failed.
3. **One scheduler owns fetch transitions.** The scheduler coalesces duplicate requests by item and field, caps concurrency per source, applies timeout and retry with jitter, respects rate-limit reset times, and emits success *and* failure messages. Bubble Tea receives typed messages and owns the current view snapshot. No network work occurs in `View` or synchronously in `Update`. A fetch attempt always has a terminal message for that attempt, including cancellation.
4. **Progressive GitHub queries.** Fetch minimal open-PR identity first, then request expensive fields for visible or selected rows. The GitHub adapter may use `gh` authentication and GraphQL/REST as appropriate, but its port returns typed records and classified errors. Track pagination and incomplete results explicitly. A single all-fields query was rejected because its latency and failure rate govern the whole screen.
5. **UI state follows domain state.** Spinner ticks only when at least one visible field is queued/fetching; cooldown and terminal failure use distinct indicators. Local fetch progress has visual priority over GitHub's own pending CI status so a pending check is not mistaken for an in-flight local request. Selection is keyed by source/repository/number.
6. **Config and commands.** Load a versioned TOML config at XDG config home (`~/.config/fs-dash` by default). Bind keys to an executable, argument vector with placeholders, optional working directory, and launch mode. Execute without implicit shell interpolation. Default `o` launches `xdg-open {url}` on Linux; other platforms can use a platform opener. Command start/exit errors become UI events and logs.
7. **Diagnostics.** Use structured events with level, time, operation, item, attempt, duration, and redacted error. Keep a bounded ring buffer for the in-app console, with optional file logging configured later. Correlation IDs connect scheduler transitions to provider calls.

## Risks / Trade-offs

- GitHub API rate limits and 5xx bursts → bounded concurrency, backoff, visible cooldown, and manual retry.
- The `gh` CLI might be unavailable or unauthenticated → startup diagnosis and a recoverable auth error, with no infinite spinner.
- Fast row updates can move focus → preserve selection by stable identity and test reorder behavior.
- Terminal glyph width and emoji support vary → use width-aware rendering and ASCII fallback for essential status.
- Logging can expose sensitive data → redact at event creation and avoid raw HTTP payloads.

## Migration Plan

Greenfield: create the module and executable, then verify with a fake provider before a live GitHub smoke run. No data migration is required.
