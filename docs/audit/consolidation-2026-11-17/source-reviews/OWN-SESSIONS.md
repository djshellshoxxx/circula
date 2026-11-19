# Account active-session flow

## Accepted scope

The account-and-security screen inventory in `docs/specs/08-ux-accessibility.md` requires a session list with revoke actions. `docs/dev/MESSAGES.md` explicitly recorded that no page existed. The existing session repository already stores owner, creation, last activity, expiry and revocation state. This work completes that flow without a migration, device/IP collection, a timeout change or a new recovery policy.

## Behavior and boundaries

- `ListOwnSessions` requires a live session and scopes every query to its person. Dead sessions are omitted using the same idle, absolute and current-role lifetimes as authentication.
- A domain-separated one-way public selector is returned for revocation. Neither bearer values nor stored credential hashes are returned in HTML, API responses or cursors.
- `RevokeOwnSession` rechecks the caller's live session inside the write transaction, checks target ownership again in SQL and records the audit atomically. Unknown and foreign selectors are indistinguishable. The audit identifies the person and whether this was the current session; it contains no session identifier (DAT-031).
- `/account/sessions` is linked from account security. A read-only confirmation page identifies the sign-in and describes the consequence. The confirmed POST needs CSRF. Ending the current sign-in clears the browser session and rotates CSRF before returning to sign-in.
- `GET /api/v1/me/sessions` and `DELETE /api/v1/me/sessions/{session}` are cookie-only owner operations. The API uses the existing signed cursor format, binds it to the person and route, caps response pages at 100 and continues correctly when the boundary session is revoked.
- Password-change requirements retain their existing semantics: the HTML authentication gate redirects first; API reads remain available, while revocation refuses until the required password change. Other sessions and API tokens are not revoked by a single-session action.

## Verification

New repository contract, service, API, HTML and catalogue regressions cover ownership, credential secrecy, stale caller rejection, exact expiry boundaries and role limits, transactional audit rollback, current-browser sign-out, CSRF, explicit confirmation and paginated continuation after a boundary revocation. The machine-readable OpenAPI contract and route authorization inventory include both API operations and all HTML routes. The contract harness exercises the documented list and revoke operations.

Local production builds of `internal/service`, `internal/api`, `internal/web` and the driver-neutral `internal/storetest` pass with Go 1.26.11. The catalogue race suite passes (1.055 seconds). The actual embedded session templates and catalogue also pass their four rendering states under race instrumentation with all production web files (1.292 seconds), without a database substitute. Catalogue JSON, OpenAPI YAML and the changed browser JavaScript parse successfully. Both new templates are registered in the shared browser axe/target-size inventory and Go page-structure suite.

Independent source review approved ownership, live-caller authorization, cookie/CSRF enforcement, role/expiry boundaries, atomic audit, selector secrecy, repeat semantics and boundary-revocation pagination. The reviewer identified the missing browser template inventory entries and requested explicit audit-payload secrecy coverage; both are included. The full SQLite behavioral, browser and contract suites still require CI qualification because local `modernc.org/libc` compilation exceeded the environment memory limit. No runtime pass is claimed for those suites until their CI result is recorded.

## Scale limitation

The repository scopes its query to one owner but currently materializes all retained sessions for that owner. API pagination bounds the response, not the query, scan or memory. Existing idle/absolute expiry and cleanup limit how long records remain; they do not impose a per-person session-count bound, because successful sign-ins have no session cap. Large histories therefore have linear read/selector cost. A later indexed public-selector or encrypted-cursor design can bound database work without exposing credential hashes. That storage/key-format change is separate from this completion; the current implementation adds no cap or timeout policy. The owner-isolation and stable page-boundary regressions cover its present behavior.
