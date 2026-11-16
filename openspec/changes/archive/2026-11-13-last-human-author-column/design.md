## Context

See `proposal.md` for motivation. The latest human actor is already fetched in `last_human_touch` and used by the Last touched by me section. That fetcher filters bots and paginates through the PR timeline when needed.

## Goals / Non-Goals

**Goals:** Show the actor from the existing last-human-touch value after Updated and preserve the field's current freshness and failure semantics.

**Non-Goals:** Add a query, change timeline parsing, show bot identities, or change the existing table's saved emoji/text preference.

## Decisions

Add a plain-text `By` column after `Updated`, before the compact status columns. Read `domain.LastHumanActivity` from the existing `last_human_touch` field. Show its actor when complete; show an em dash when a complete timeline has no human actor; show pending when the field is unrequested, loading, or incomplete; show unavailable after failure; and retain a known actor with stale styling when its value is stale. Reuse the table's existing responsive column sizing and truncation behavior.

The existing timeline enrichment already excludes bot actors, so the column and the Last touched by me classification remain based on the same source and cannot diverge through separate queries.

## Risks / Trade-offs

- The extra column uses terminal width → keep it between Updated and lower-priority status columns so responsive layouts can hide later columns first.
- Timeline actor data may be incomplete → display pending or unavailable instead of inferring from the PR author or another field.
