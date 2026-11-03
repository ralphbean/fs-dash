## Context

Changes 01–03 establish list/detail navigation and source-neutral field state for PRs. This change adds two issue sources. See proposal.md and the issue-workspaces spec.

## Goals / Non-Goals

**Goals:** Reuse the navigation and diagnostics contracts while keeping source-specific issue semantics clear.

**Non-Goals:** Forcing GitHub and Jira into identical field sets, issue mutations, or a universal cross-source priority score in the first issue release.

## Decisions

1. **A small common item envelope.** Share `ItemID` (source, host, stable ID), title, URL, source label, field status, list navigation, and selected-item command context. Keep GitHub issue and Jira issue detail models separate. A single giant issue struct was rejected because Jira status and GitHub labels have different meaning.
2. **Source adapters own pagination and authentication.** GitHub issue reads use the existing GitHub session; Jira reads use a separately configured base URL and credential mechanism. Each adapter maps source errors into the same retry/terminal categories, without leaking credentials to logs.
3. **Independent sections and filters.** PR, GitHub issue, and Jira issue modes have separate scopes, cursors, refresh schedules, and rankings. A Jira outage cannot stop GitHub views. The active source and filter remain visible in the header.
4. **Details use shared shell, specific content.** Reuse back navigation, command hotkeys, and field-state display, but render source-specific fields and links. Keep issue writes out until their workflows and permissions are specified.

## Risks / Trade-offs

- Jira custom fields vary by project → map a small core set, show unsupported fields as unavailable, and allow later field configuration.
- Mixed source identity can collide → namespace all cache and local state keys by source and host.
- Large issue sets can be slow → paginate, load visible rows first, and preserve selection across pages.

## Migration Plan

Add optional Jira configuration; existing PR behavior and local data remain valid. Missing Jira configuration hides or labels the Jira mode as unconfigured.
