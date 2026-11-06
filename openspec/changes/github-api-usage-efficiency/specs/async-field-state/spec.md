## MODIFIED Requirements

### Requirement: Failed fetches settle visibly
Every fetch attempt SHALL end in success, retry cooldown, cancellation, or terminal failure. A failed list or field fetch SHALL never leave its loading indicator active after the attempt ends. A GitHub rate-limit response SHALL establish one shared cooldown for all GitHub requests in the application, including requests for other repositories and manual refreshes.

#### Scenario: Transient failure
- **WHEN** a request receives a retryable network, rate-limit, or server error
- **THEN** its state shows the error, next retry time, and any retained value while the spinner stops during cooldown

#### Scenario: Terminal failure
- **WHEN** a request cannot be retried without changed credentials, permissions, or input
- **THEN** its state shows a terminal failure and a manual retry path

#### Scenario: Primary GitHub quota is exhausted
- **WHEN** a GitHub response reports that the primary quota is exhausted and provides a reset time
- **THEN** all queued and newly requested GitHub work waits until that reset time before sending another request

#### Scenario: Secondary rate limit provides Retry-After
- **WHEN** a GitHub response reports a secondary rate limit with a Retry-After duration
- **THEN** all GitHub work waits for that duration before sending another request

#### Scenario: Secondary rate limit omits Retry-After
- **WHEN** a GitHub response reports a secondary rate limit without a Retry-After duration or an exhausted primary quota
- **THEN** the application applies a shared cooldown of at least one minute and increases subsequent retry delays with backoff

#### Scenario: Manual refresh during a shared cooldown
- **WHEN** the user requests a manual refresh while a GitHub cooldown is active
- **THEN** the refresh remains queued and sends no GitHub request before the shared cooldown expires
