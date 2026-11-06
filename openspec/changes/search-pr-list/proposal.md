## Why

Finding a pull request by its title or people attached to it currently requires scanning the list manually. Directional search lets users locate matching rows quickly while keeping the existing list and selection model.

## What Changes

- Add `/` for forward search and `?` for reverse search over the active PR list, matching case-insensitive substrings in PR titles, author logins, and assignee logins.
- Use Enter to run the entered query, `n` to move to the next match in the search direction, and wrap at either end of the active list.
- Keep the last matching row selected when `Esc` exits search; preserve the current selection when there are no matches.
- Move the help shortcut from `?` to `F1`, and reserve search/help keys from custom command bindings.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `pr-list`: Define directional search, match fields, result traversal, wrapping, and search exit behavior.
- `user-config`: Reject custom command bindings that conflict with the search and help shortcuts.

## Impact

Changes PR list search state, key handling, list footer/help, and configuration validation, with corresponding UI and config coverage. “Summary” is interpreted as the PR title shown in the list; author and assignee names are the GitHub logins currently displayed there. Help moves to `F1` because `?` becomes reverse search.
