## Context

See `proposal.md` for motivation. Today, the shared scheduler uses a global limit of 2 requests, a 150 ms request-start interval, and a 20 second timeout. UI scheduling separately limits detail requests to 2 and activity requests to 5, serializes offscreen work, delays offscreen detail/activity by 500/750 ms, processes up to 64 queue entries per update, and waits 5 ms before a continuation.

## Goals / Non-Goals

**Goals:** Preserve those defaults while letting users independently tune wire-level concurrency, request pacing, per-kind and offscreen work, and UI queue responsiveness through TOML.

**Non-Goals:** Let configuration override GitHub's primary or secondary rate-limit cooldowns, change request contents or cache freshness, or make refresh settings dynamically adapt to quota observations.

## Decisions

Add a `RefreshConfig` under `[refresh]` with the nine settings and defaults listed in `proposal.md`. Decode user files into a config initialized with those defaults so an old file or a partial `[refresh]` table retains existing behavior. Reject values outside the ranges in the user-config spec with errors that name the setting and valid range. Keep the config version unchanged because all settings are optional and backward compatible.

Use `max_concurrent_requests` as the shared scheduler's actual request ceiling. `max_concurrent_detail_requests` and `max_concurrent_activity_requests` constrain their own work queues but do not raise the shared ceiling. `max_concurrent_offscreen_requests` is one shared cap across offscreen detail and activity work. The effective limit for a request is the lowest applicable global, work-kind, and offscreen limit. Per-kind values may exceed the global ceiling; in that case the global ceiling still wins.

Apply `request_start_interval_ms` in the shared scheduler, separately from its concurrency semaphore. Apply `offscreen_detail_interval_ms` and `offscreen_activity_interval_ms` as additional per-kind spacing for background dispatches. Zero disables only the corresponding local spacing. The existing rate-limit gate remains authoritative even when local spacing is zero.

Use `queue_batch_size` and `queue_yield_interval_ms` only for processing queue work on the UI event loop. They must not change the number of requests allowed by the other settings. Keep selected and visible requests ahead of offscreen requests at every limit.

Document the settings in one `[refresh]` section in `config.example.toml`. Explain that the shared request limit is the overall ceiling, per-kind and offscreen limits further restrict work, request spacing is separate from concurrency, and queue batch/yield settings affect UI responsiveness rather than HTTP concurrency. A single concurrency knob was considered but rejected because it cannot independently tune background drain rate or UI scheduling cost.

## Risks / Trade-offs

- Large configured limits can increase GitHub secondary limits, timeouts, or server errors → retain conservative defaults, validate upper bounds, and leave the server cooldown in force.
- Independent settings can be confusing when one cap makes another ineffective → document the effective minimum in the example comments and keep the settings grouped together.
- Large queue batches or very short yields can reduce input responsiveness → constrain their ranges and state their UI-only effect in the comments.

## Migration Plan

Existing configuration files need no migration. Missing values use current defaults; the example file adds the `[refresh]` section with comments. Users can remove the section to return to defaults.
