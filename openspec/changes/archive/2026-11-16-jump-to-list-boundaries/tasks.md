## 1. Add list boundary navigation

- [x] 1.1 Handle `gg` and `G` in PR list mode, keeping empty-list behavior unchanged and routing successful jumps through existing selection updates.
- [x] 1.2 Show `gg` and `G` in the list footer and keyboard help.

## 2. Add thread diff boundary navigation

- [x] 2.1 Handle `gg` and `G` in thread mode to scroll to the top and bottom of the thread content.
- [x] 2.2 Show `gg` and `G` in the thread footer and keyboard help.

## 3. Protect key bindings

- [x] 3.1 Reserve `g` and `G` in custom command binding validation with a clear validation error.

## 4. Finalize the change

- [x] 4.1 Sync the `pr-list`, `review-thread-workflow`, and `user-config` specs and run strict OpenSpec validation.
