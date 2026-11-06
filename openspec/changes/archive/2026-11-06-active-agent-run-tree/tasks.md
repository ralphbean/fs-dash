## 1. Preserve per-run activity evidence

- [x] 1.1 Extend the activity model to retain a collection of run records with run ID, readable agent name, safe workflow URL, and start time alongside the existing PR-level state.
- [x] 1.2 Parse trusted Fullsend status comments by run ID, resolve each run's latest status independently, exclude terminal or stale runs, and retain run-history completeness.
- [x] 1.3 Add parser tests for concurrent runs, per-run terminal status, stale runs, missing workflow links, and untrusted comment authors.

## 2. Render active run children

- [x] 2.1 Project active run records beneath their PRs in the Active Fullsend Agents section with process-tree connectors, readable labels, and an incomplete-history indicator when needed.
- [x] 2.2 Align run children under the PR identity column after Star, render available workflow URLs as terminal hyperlinks, and show elapsed time on single-line rows.
- [x] 2.3 Schedule a lightweight local UI refresh that advances elapsed labels without activity requests and stops when the active-agent section is not shown.
- [x] 2.4 Add UI coverage for hierarchy, workflow links, missing links, elapsed-time updates, completed runs, and compact-width rendering.

## 3. Validate the change

- [x] 3.1 Run the relevant Go tests and full repository checks, then run strict OpenSpec validation and resolve any failures.
- [x] 3.2 Review the rendered Active Fullsend Agents section at standard and narrow terminal sizes, including no-color mode.
