## 1. Schedule freshness-driven refreshes

- [x] 1.1 Add a generation-guarded timer based on the earliest list-section or selected/visible enrichment freshness deadline, with a 20-second lead and list-mode lifecycle.
- [x] 1.2 Automatically refresh due list sections and enqueue due selected/visible enrichment through the existing priority, pacing, and shared cooldown paths.
- [x] 1.3 Sync the `async-field-state` and `pr-triage` specs and run strict OpenSpec validation.
