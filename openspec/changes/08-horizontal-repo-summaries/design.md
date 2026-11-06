## Context

See proposal.md for motivation. The PR list currently adds one frame-content line for each configured repository before the table.

## Goals / Non-Goals

**Goals:** Preserve the existing repository status text and semantic colors while displaying all configured repositories in one horizontal summary row.

**Non-Goals:** Change repository status semantics, reorder repositories, or reduce the information shown for an individual repository.

## Decisions

Join the existing colored repository/status pairs with a muted horizontal separator and append the joined string once to the frame content. Reuse the frame's existing ANSI-aware truncation so an overlong row remains a single physical line and fits the terminal.

## Risks / Trade-offs

- A large number of long repository names can exceed the frame width → truncate the combined row to the frame's available width, preserving the vertical space.
