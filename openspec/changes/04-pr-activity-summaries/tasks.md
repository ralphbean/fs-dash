## 1. Summary inputs and cache

- [ ] 1.1 Add opt-in config for summary executable, model ID, timeout, and size limits.
- [ ] 1.2 Build bounded factual PR snapshots and a fingerprint covering head, reviews, comments, checks, prompt, and model.
- [ ] 1.3 Implement a local summary cache with exact-match reuse and stale-result labeling.

## 2. Generation and UI

- [ ] 2.1 Implement a bounded `codex exec` adapter; confirm installed CLI flags and `gpt-6-luna` access at implementation time.
- [ ] 2.2 Coalesce duplicate requests, cancel obsolete generations, and keep detail fetches independent.
- [ ] 2.3 Render summary, provenance, loading, stale state, and sanitized failure in PR details.
- [ ] 2.4 Verify no model call when disabled or cache hits, invalidation on new comments/checks, and safe behavior on timeout.
