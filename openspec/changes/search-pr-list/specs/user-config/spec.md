## ADDED Requirements

### Requirement: Search and help shortcuts cannot be overridden
The system SHALL reject custom command bindings for `/`, `?`, `n`, and `f1`, which are reserved for list search and keyboard help.

#### Scenario: Reject a reserved search or help binding
- **WHEN** a configured command binding uses `/`, `?`, `n`, or `f1`
- **THEN** configuration validation reports that the key is reserved
