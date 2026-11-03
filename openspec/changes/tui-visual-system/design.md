## Context

The current `internal/ui/model.go` builds all four views with plain string builders. It already exposes field phases, a selected row, a bounded list viewport, and log levels. See proposal.md and the visual-system spec for observable behavior. This change follows the completed PR list foundation and precedes richer triage views.

## Goals / Non-Goals

**Goals:** Make status and navigation faster to scan, preserve the existing state meanings, and give future views reusable styling primitives.

**Non-Goals:** New GitHub fields, changes to fetch scheduling, configurable user-authored color codes, or a wholesale rewrite of Bubble Tea state handling.

## Decisions

1. **Central semantic theme.** Define role tokens such as `focus`, `accent`, `muted`, `link`, `pending`, `success`, `stale`, `cooldown`, `warning`, and `failure`, plus border and surface roles. Ship vivid dark and light palettes. Map `domain.Phase` and log levels to these roles in one place. Keep status text and symbols in the component, so color never carries the only signal. Direct color literals scattered through view functions were rejected because later PR and issue modes would drift.
2. **Panel components.** Use the existing Lip Gloss dependency for reusable header, section, row, details, context, console, and footer renderers. Double or heavy borders frame the active pane; lighter dividers separate subordinate regions. Keep borders around sections rather than every row to avoid visual noise and lost width. A single border around the entire screen was considered but gives little hierarchy.
3. **Focus and status are separate layers.** Apply a strong row background or left rail to the selected row, while preserving status-colored badges and symbols inside it. Use a distinct accent for links and hotkeys. A blanket selected-row color override was rejected because it would hide errors.
4. **Measure before styling.** Compute the content rectangle from terminal width and height, border thickness, padding, header/footer, and section count; truncate plain content to that width before styling. Keep the selected row in the viewport after resize. Use a compact layout below a practical width threshold, with simple dividers and fewer metadata columns. Avoid counting ANSI escape bytes as columns.
5. **Color negotiation and preference.** Add `theme = "dark" | "light"` and `color = "auto" | "never"` to TOML config. Default to dark/auto; honor `NO_COLOR` regardless of theme. Let the terminal renderer reduce color depth when needed. Avoid an `always` mode in v1 because this TUI is interactive and forcing escapes into unsupported terminals adds little value.
6. **Central render tests and live inspection.** Test semantic style mapping, no-color output, width/height at 80x24 and a narrow size, and selected-row visibility after resize. Then run the actual TUI with live PR data in both palettes and in no-color mode to verify contrast and borders in a terminal.

## Risks / Trade-offs

- Rich borders consume rows and columns → measure the content rectangle and switch to compact framing in narrow terminals.
- Terminal palettes and true-color support vary → ship dark/light choices, use terminal color negotiation, and retain labels.
- ANSI styling can break truncation and alignment → truncate before applying style and test rendered cell width.
- Bright color across every datum can obscure priority → reserve high-saturation colors for focus, actionable links, and exceptional states; use muted text for context.

## Migration Plan

Add optional theme/color keys with defaults so existing config files continue to load. No data migration is needed. Rollback restores the prior plain renderers without changing fetch or local data.
