## 1. Add list boundary navigation

- [ ] 1.1 Handle `gg` and `G` in PR list mode, keeping empty-list behavior unchanged and routing successful jumps through existing selection updates.
- [ ] 1.2 Add `gg` and `G` to the list footer and keyboard help.
- [ ] 1.3 Add UI coverage for top/bottom selection, empty sections, list-only handling, and selected-row visibility after jumping.

## 2. Protect key bindings

- [ ] 2.1 Reserve `g` and `G` in custom command binding validation with a clear validation error.
- [ ] 2.2 Add configuration coverage for rejecting both reserved keys.

## 3. Finalize the change

- [ ] 3.1 Sync the `pr-list` and `user-config` specs and run strict OpenSpec validation.
