## Context

See proposal.md for motivation. The current `pr-triage` activity field reduces trusted Fullsend status-comment evidence to one `ActivityEvidence` per PR, using the latest recognized marker overall. Fullsend status comments include a run marker, a human-readable agent role, and a `View workflow run` link; comments are already fetched as part of bounded PR activity enrichment. The selected PR's activity evidence also links to its issue comment.

## Goals / Non-Goals

**Goals:** Preserve separate concurrent runs as structured activity evidence; render them under their PR; keep workflow links attached to the evidence that supplied them; update elapsed labels locally while the section is visible.

**Non-Goals:** Query GitHub Actions separately, infer active runs from PR authorship or branch names, or change the existing six-hour stale threshold and PR-level activity lifecycle.

## Decisions

1. **Group trusted status evidence by run ID.** Build one run record per recognized Fullsend run marker and resolve its current state from the latest status evidence for that run. A terminal marker closes only its own run, allowing another non-terminal run to remain visible. Keep PR-level activity classification and its unknown, stale, failure, and inactive states intact. Carry the comment connection's completeness separately from PR-level status certainty so a partial scan cannot imply that every run was found.
2. **Use evidence already present in the status comment.** Extract the readable agent label and safe workflow-run link from the comment body, and use the comment's creation time as the start time. Do not derive a workflow URL from the PR URL or make one Actions API request per run; the explicit link identifies the originating workflow repository and server.
3. **Keep elapsed time presentation-only.** Store the start timestamp with run evidence and derive elapsed duration from the UI clock. Reuse or add a lightweight Bubble Tea tick while active run children are visible; this must not trigger GitHub reads or mutate the cached evidence.
4. **Render children with the PR row's hierarchy.** Keep each PR as the parent item in the Active Fullsend Agents section and render indented, connector-marked child rows beneath it. Start the tree at the dynamically calculated PR identity column, after the Star column and its separator. Preserve one-line rows, terminal hyperlink behavior, and the existing compact layout rules; truncate labels to available width rather than wrapping.

## Risks / Trade-offs

- **Older or incomplete comments omit a workflow link or recognizable agent role** → show the run with a readable fallback derived from the trusted bot identity, and omit the hyperlink when no safe workflow URL is available.
- **A comment page omits an earlier concurrent run** → show the known active runs with an explicit incomplete-history marker; do not claim that a partial comment scan found every run.
- **Per-second repainting costs terminal work** → tick only while the active-agent section is visible and use the coarsest interval that keeps the elapsed display useful.

## Migration Plan

No persisted data or external schema changes. Existing activity cache entries can be refreshed through the current activity field freshness lifecycle; rollback removes child run rendering and restores the existing PR-level summary.
