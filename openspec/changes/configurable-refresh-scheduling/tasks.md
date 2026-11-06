## 1. Add refresh configuration

- [ ] 1.1 Add the `[refresh]` configuration model and current-behavior defaults, including defaults for partially specified tables.
- [ ] 1.2 Validate concurrency, batch-size, and millisecond ranges with errors that identify the invalid setting.
- [ ] 1.3 Document all refresh settings and their relationships in `config.example.toml`.
- [ ] 1.4 Add config coverage for omitted values, partial settings, valid overrides, and invalid ranges.

## 2. Apply request scheduling settings

- [ ] 2.1 Configure the shared GitHub scheduler from the global request concurrency and start-interval settings while preserving shared server cooldowns.
- [ ] 2.2 Apply detail and activity queue limits plus one shared offscreen concurrency cap and each work kind's offscreen pacing interval.
- [ ] 2.3 Apply queue batch size and yield interval to background scheduling while retaining selected/visible priority and UI responsiveness.
- [ ] 2.4 Add scheduling coverage for defaults, global and per-kind caps, offscreen cross-kind limits, pacing, and queue yielding.

## 3. Finalize the change

- [ ] 3.1 Sync the `user-config`, `github-api-efficiency`, `pr-triage`, and `async-field-state` specs and run strict OpenSpec validation.
