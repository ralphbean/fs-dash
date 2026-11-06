## 1. Capture quota metadata from GitHub responses

- [ ] 1.1 Add a shared parser for `gh api --include` response headers and GraphQL quota metadata, including paginated and error responses.
- [ ] 1.2 Carry valid quota observations from list and enrichment results into the application model without replacing them with missing or malformed data.

## 2. Present quota status in the dashboard

- [ ] 2.1 Render remaining quota, total limit, reset time, observation age, low-quota warning, and exhausted state in the shared status area.
- [ ] 2.2 Keep unknown and stale observations distinct and preserve access to the indicator in compact terminal layouts.

## 3. Verify quota status behavior

- [ ] 3.1 Add parser and model tests for successful, paginated, malformed, and rate-limited responses.
- [ ] 3.2 Add UI coverage for unknown, current, low, exhausted, and stale quota states at common and narrow widths.
- [ ] 3.3 Run the Go tests, build, formatting checks, diff checks, and strict OpenSpec validation.
