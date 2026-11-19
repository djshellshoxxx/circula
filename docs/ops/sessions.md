# Account session management

Members can review and end their own sign-ins from **Security → Manage active sessions**. This account-wide page does not need a network selection or an administrator role. It requires a live session cookie; an API token cannot list or revoke sessions.

## Member actions

The inventory identifies the current browser as **This session** and shows sign-in, last-activity and effective-expiry times. Effective expiry uses the same current role-specific idle and absolute lifetimes as authentication. Device labels and IP history are not collected by this feature.

**Sign out this session** opens a confirmation page. Opening it changes nothing. Submitting the checked confirmation ends the selected sign-in immediately. **Sign out here** ends the current browser and redirects to sign-in. A session that expired or ended between opening and submitting a confirmation may produce a not-found response; refresh the inventory.

**Sign out other sessions** has its own confirmation. It preserves the current session, revokes the person's other retained sign-ins, and cancels pending second-factor sign-in challenges. Those interrupted sign-ins must start again. Other people's sessions and API tokens are unaffected. A repeated request succeeds. Every successful bulk command records an audit event, including a zero-row repeat, so operator records distinguish a successful command from no request.

API clients use these cookie-authenticated operations:

| Operation | Result |
|---|---|
| `GET /api/v1/me/sessions?limit=25` | Active-session page and signed `next_cursor` |
| `DELETE /api/v1/me/sessions/{session}` | `204` after ending the owned selector; retained already-revoked rows also return `204` |
| `POST /api/v1/me/sessions/revoke-others` | `204` after ending other sign-ins while retaining the caller |

Both writes require the existing CSRF token and reject a pending forced password change. Unknown and foreign selectors have the same not-found response. Authorization is rechecked inside the write transaction, so a retained request actor cannot perform another revocation after its current session has ended.

## Pagination and work bounds

The API defaults to 25 rows, with an explicit maximum of 100. HTML uses 25. Session creation time and the independent public selector form the stable total ordering; ordinary activity updates do not move a row. HTML supports Previous, Next and First page without JavaScript or an accumulating cursor history. Concurrent creation, revocation or expiry can change what a page contains; First page refreshes the newest view. If every retained newer row behind a Previous link has disappeared, the request refreshes the newest page within its remaining scan budget. This preserves the surviving boundary row instead of skipping it through a Next link. An empty forward continuation remains terminal, so API traversal never wraps back to earlier rows.

Each direction examines at most 20 indexed batches of 100 retained rows. This bound includes expired and revoked rows. A filtered page can therefore be empty while continuation remains. API clients must follow `next_cursor` until it is null, including after an empty response. HTML still offers navigation after an empty scan window. No total count query or complete owner-session materialization is used. Individual confirmation and revocation use direct owner-and-selector lookups.

Signed API cursors are bounded to 4096 characters and use canonical base64url components. Raw session keyset positions are bounded to 85 bytes before timestamp parsing. Invalid cursor errors must not be logged with session credentials or credential hashes. Public selectors are not credentials and do not authorize access by themselves.

## Schema changes and upgrade handling

Migration `0020_session_public_id.sql` assigns independent random public selectors to existing sessions and creates uniqueness and owner-order indexes. Migration `0021_session_unrevoked_index.sql` adds the partial index used by other-session revocation. Both are forward migrations; existing credentials and configured lifetimes remain unchanged.

Before an upgrade, take and verify a backup using the normal [upgrade procedure](upgrade.md). Do not edit or reverse applied migration files. Any rollback requiring an older schema uses a compatible verified backup and the existing restore procedure; authentication state restored from a backup needs the incident-response review described by the [operator runbook](runbook.md).

Links containing the former derived selectors no longer identify sessions after migration. Reload the inventory to obtain current public selectors. Signed cursors for the old selector ordering return `400 invalid-cursor`; restart without a cursor. An unsupported or malformed signed position also returns `400 invalid-cursor`, rather than a different timestamp-validation response.

## Audit and scope

Individual revocation records the person and whether the selected session was current. Bulk revocation records aggregate results, including the number of previously unrevoked retained other rows marked revoked and confirmation that the current session was retained. That count may include retained expired rows; it is not a live-session count. Neither operation records credential values, stored credential hashes or public selectors in its audit payload.

The operator's `circula revoke-sessions` emergency command remains separate. It can invalidate every session or every session of a specified person and does not preserve the caller's browser. Session management does not change ledger entries, balances, monetary rules or governance policy.
