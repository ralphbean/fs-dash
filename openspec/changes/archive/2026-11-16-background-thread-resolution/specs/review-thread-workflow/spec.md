## MODIFIED Requirements

### Requirement: Selected review thread can be resolved
The system SHALL let the developer resolve the selected unresolved review thread with `r`. After `r` is pressed, the walkthrough SHALL immediately advance to the next unresolved thread and temporarily omit the selected thread while its resolution is in progress. The mutation and a fresh complete thread-list readback SHALL run in the background, and the developer SHALL be able to start resolving subsequent threads while earlier resolutions are pending. A thread SHALL remain omitted only after a complete readback confirms it is resolved. If readback finds it unresolved or cannot verify its state, the thread SHALL return to the walkthrough and the UI SHALL explain the outcome. The UI SHALL report success only after persisted resolution is verified.

#### Scenario: Advance while resolution runs
- **WHEN** the developer presses `r` on an unresolved review thread
- **THEN** the walkthrough immediately advances past that thread while its mutation and readback continue in the background

#### Scenario: Resolve subsequent threads without waiting
- **WHEN** a prior thread resolution is still pending and another unresolved thread is selected
- **THEN** the developer can press `r` to advance again and queue that thread's resolution

#### Scenario: Resolve a thread
- **WHEN** a complete fresh thread-list readback confirms a thread is resolved
- **THEN** the thread remains removed from the unresolved walkthrough and the UI reports verified success

#### Scenario: Resolution is not confirmed
- **WHEN** a complete fresh readback still contains the thread
- **THEN** the thread returns to the walkthrough and the UI explains that it remains unresolved

#### Scenario: Resolution cannot be verified
- **WHEN** the fresh readback fails or is incomplete
- **THEN** the UI does not report success, restores the thread to the walkthrough, and explains that its state could not be verified

#### Scenario: Mutation error is resolved by readback
- **WHEN** the resolve mutation reports an error but a complete fresh readback no longer contains the thread
- **THEN** the UI reports verified success because the persisted thread state confirms resolution
