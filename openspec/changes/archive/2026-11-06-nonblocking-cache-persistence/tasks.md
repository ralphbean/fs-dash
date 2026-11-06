## 1. Async cache persistence

- [x] 1.1 Move enrichment cache snapshot saving out of UI updates and serialize writes with generation-based coalescing.
- [x] 1.2 Keep cache pruning off the UI update path and optimize PR access-time updates with a single cache pass.
- [x] 1.3 Preserve visible save-failure warnings and avoid retry loops.

## 2. Verification

- [x] 2.1 Format the changed Go files, build the application, and validate the OpenSpec change.
