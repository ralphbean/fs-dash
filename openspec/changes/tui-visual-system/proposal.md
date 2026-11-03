## Why

The working PR dashboard is readable but visually flat. Color and framed regions can make selection, fetch state, and log severity quicker to scan as the dashboard gains more data.

## What Changes

- Introduce a consistent semantic color system for focus, links, neutral data, pending work, stale data, cooldown, success, warning, and failure.
- Add bordered panels and dividers to the PR list, selected-item context, details shell, help, and log console.
- Keep status labels and symbols alongside color, and provide a usable no-color presentation.
- Make panel layout responsive so borders do not hide rows or create unwanted wrapping in common terminal sizes.

## Capabilities

### New Capabilities

- `tui-visual-system`: Semantic styling, panel hierarchy, and terminal-size behavior across existing views.

### Modified Capabilities

None.

## Impact

Changes presentation code in `internal/ui`, theme preferences in local config, and UI tests. Uses the existing terminal styling dependency. Fetch behavior, data contracts, and GitHub requests remain as defined by the foundation change.
