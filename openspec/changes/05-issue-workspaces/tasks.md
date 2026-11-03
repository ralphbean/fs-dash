## 1. Shared item shell

- [ ] 1.1 Extend source-scoped item identity and selected-item commands to GitHub and Jira issues without sharing source-specific detail models.
- [ ] 1.2 Add issue list/detail mode switching, stable selection, source labels, and per-mode loading/error states.

## 2. GitHub issues

- [ ] 2.1 Add paginated GitHub issue list and detail reads through the existing scheduler.
- [ ] 2.2 Add GitHub issue filters and explainable ranking with partial-data indicators.
- [ ] 2.3 Verify PR and issue modes operate independently during one source request failure.

## 3. Jira issues

- [ ] 3.1 Add Jira base URL, project scope, and authentication configuration with credential-safe diagnostics.
- [ ] 3.2 Add paginated Jira issue list and detail reads mapped to a small core field set.
- [ ] 3.3 Add Jira issue filters and source-specific priority reasons, with unconfigured/auth-error states.
- [ ] 3.4 Verify colliding IDs remain separate and a Jira outage does not block GitHub modes.
