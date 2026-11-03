# fs-dash delivery order

fs-dash is a Go/Bubble Tea dashboard for choosing the next useful action across Fullsend work. Each OpenSpec change has its own proposal, behavior specs, design, and task checklist. Apply and archive them in order where dependencies exist; do not combine all features into one implementation change.

| Change | Usable result | Depends on |
| --- | --- | --- |
| [01 PR list foundation](changes/archive/2026-11-03-01-pr-list-foundation/proposal.md) | Fast navigable PR list, explicit async field lifecycle, diagnostics, config, command bindings | None |
| [TUI visual system](changes/archive/2026-11-03-tui-visual-system/proposal.md) | Semantic color, framed views, responsive layout, and no-color mode | 01 |
| [02 PR triage and reminders](changes/02-pr-triage-and-reminders/proposal.md) | Filtered ranked review list, active Fullsend agent section, stars and snoozes | 01 |
| [03 PR details and review workflow](changes/03-pr-details-and-review-workflow/proposal.md) | Details, unresolved-thread walkthrough, comments, merge/queue, close | 01 and 02 |
| [04 PR activity summaries](changes/04-pr-activity-summaries/proposal.md) | Optional cached Luna summary of recent PR updates | 03 |
| [05 Issue workspaces](changes/05-issue-workspaces/proposal.md) | GitHub and Jira issue lists and details | 01 and 03 |

The shared contract is established in change 01: every remote or derived field can carry a retained value, provenance, freshness, and an independent request state. Later changes reuse it for new sources and synthetic fields. A failed request must settle its spinner, show an error and retry policy, and preserve any usable older value.

Change 01 and the TUI visual system are implemented and archived. Change 02 is implemented and ready to archive; change 03 is next in the recommended order. Changes 04 and 05 can be scheduled independently after their prerequisites; their numbers express a recommended product order, not a technical coupling.
