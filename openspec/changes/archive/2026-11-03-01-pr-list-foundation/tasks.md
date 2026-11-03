## 1. Application shell and contracts

- [x] 1.1 Create the Go module, executable, package boundaries, and a Bubble Tea shell that starts, exits, and reports version/help.
- [x] 1.2 Define source-scoped PR identity, optional field value/provenance, request lifecycle, and typed provider errors.
- [x] 1.3 Define narrow list, enrichment, clock, logger, config, and command-runner ports with fake adapters for tests.

## 2. Asynchronous data flow

- [x] 2.1 Implement field transitions and generation checks for initial fetch, refresh, cancellation, cooldown, and terminal failure.
- [x] 2.2 Implement bounded scheduling, request coalescing, timeouts, retry classification, and rate-limit-aware cooldown.
- [x] 2.3 Implement derived-field dependency versioning and three-valued boolean evaluation, including bot approval as a fixture.
- [x] 2.4 Add focused tests proving failed requests settle, retained values stay visible, and stale responses cannot overwrite current state.

## 3. GitHub PR list and UI

- [x] 3.1 Implement a lightweight paginated GitHub open-PR adapter using existing `gh` authentication and actionable auth errors.
- [x] 3.2 Build the PR list screen with stable selection, independent section states, refresh, empty state, and help.
- [x] 3.3 Add field-level spinner animation, stale/error/cooldown indicators, and a nonblocking route for later enrichment.
- [x] 3.4 Verify with a fake provider that a 5xx list failure stops its spinner and leaves the rest of the UI responsive; run a live read-only GitHub smoke check.

## 4. Configuration and diagnostics

- [x] 4.1 Load and validate XDG config with documented defaults and a sample config file.
- [x] 4.2 Add configured repository scope and current-directory remote detection with a visible setup state when neither is available.
- [x] 4.3 Add explicit argument-vector command bindings, default browser opener, working directory support, and execution status events.
- [x] 4.4 Add structured redacted logs and a bounded in-app console with level filter, scroll, and toggle key.
- [x] 4.5 Verify startup with missing/invalid config, command arguments containing spaces, and logs from a failed fetch.
