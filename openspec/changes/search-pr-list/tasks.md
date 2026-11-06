## 1. Implement list search

- [ ] 1.1 Add search prompt and retained query/direction state for forward (`/`) and reverse (`?`) search.
- [ ] 1.2 Search the active filtered PR rows in rendered order using case-insensitive substrings of title, author login, and known assignee logins.
- [ ] 1.3 Support live query edits, `n` repeat, wrapping, no-match feedback, and preserving the last selected hit when the prompt closes.
- [ ] 1.4 Surface partial search coverage when there are no known hits and assignee data is incomplete; reevaluate an open query when assignee data updates.

## 2. Update keyboard help and configuration

- [ ] 2.1 Add search and repeat-search shortcuts to the list footer and help; move help from `?` to `F1`.
- [ ] 2.2 Reserve `/`, `n`, and `f1` in custom command binding validation with a clear error.

## 3. Verify and finalize

- [ ] 3.1 Add UI and config coverage for search direction, fields, wraparound, query exit, incomplete assignees, help access, and reserved bindings.
- [ ] 3.2 Sync the `pr-list` and `user-config` specs and run strict OpenSpec validation.
