# 07 API and Integration Contract

Status: Draft v0.1 · Owner/role: API lead · Last updated: 2026-10-21
Scope: Principles, authentication and authorisation, error format, idempotency, pagination, concurrency control, versioning and compatibility, and the conceptual MVP operations of the JSON API under `/api/v1`. Does not select a framework or wire-level library beyond the standards cited. Consistent with DEC-002 (server-rendered HTML plus versioned JSON API over one domain layer).
Dependencies: [00-conventions.md](00-conventions.md) (glossary, LED-1..LED-7, DEC-002); [03-domain-model.md](03-domain-model.md) (rules behind each operation); [05-architecture.md](05-architecture.md) (ARC-002, ARC-003, ARC-007); [06-data-ledger-audit.md](06-data-ledger-audit.md) (idempotency records, money and identifier representation); [08-ux-accessibility.md](08-ux-accessibility.md) (shared presentation of balances and errors); [09-security-privacy.md](09-security-privacy.md) (authentication, rate limits, logging); [12-quality-test-plan.md](12-quality-test-plan.md) (contract tests).
Open questions: see section 11.

External standards referenced by number (not fetched during this draft; URLs and access dates to be recorded by QA before release): RFC 9457 (Problem Details for HTTP APIs), RFC 9110 (HTTP semantics), RFC 9562 (UUIDs), RFC 3339 (date-time), RFC 7396 (JSON Merge Patch), RFC 6265bis (cookies, IETF draft), RFC 8594 (Sunset header), RFC 9745 (Deprecation header), RFC 4180 (CSV), OpenAPI Specification 3.1 (description format, optional).

---

## 1. Principles

1. **One rule set, two surfaces.** The API MUST enforce the same ledger invariants, authorisation and state transitions as the HTML interface because both call the same domain services (ARC-002, DEC-002). The API adds no extra permissions and removes none.
2. **Resources for reading, commands for changing.** Reads are side-effect-free. Ledger-affecting changes are explicit commands (POST to an action sub-resource) so that idempotency, audit, and state rules are visible.
3. **Machine-stable.** Field names, error codes, and enumerations are stable within a major version.
4. **Least exposure.** Responses contain only data the caller may see under network visibility rules (09). Cross-network and not-visible resources are indistinguishable from non-existent ones.
5. **Explicit money.** Amounts are integers in minor units and are never floating point (LED-7).

## 2. Base path, resources and representation

| Rule | Requirement |
|---|---|
| API-001 · P0 | The HTML adapter and the JSON API MUST call the same domain services. Each MVP operation in section 8 MUST have one domain command used by both surfaces where a surface exists. |
| API-002 · P0 | The JSON API MUST be served under the path prefix `/api/v1` of a network-scoped deployment. The network is identified by the deployment's network context (self-hosted) or by a path segment (hosted, OQ-API-02). |
| API-003 · P0 | Requests and responses MUST use `application/json; charset=utf-8`. A request with a body MUST declare this content type; otherwise the response is 415. Request bodies MUST be parsed strictly: unknown fields and duplicate keys are rejected with 400 (`invalid-body`). |
| API-004 · P0 | Money values MUST be JSON strings containing a canonical base-10 integer of minor units (for example `"1250"` for 12.50 credits at 2 decimals). Leading zeros, signs on positive amounts, decimal points, exponents and whitespace MUST be rejected. Values MUST fit a signed 64-bit integer. A positive amount for a trade MUST be greater than zero (LED-7). Strings avoid precision loss in JavaScript clients. |
| API-005 · P0 | Timestamps MUST be RFC 3339 in UTC with a `Z` suffix and millisecond precision. Identifiers MUST be UUIDv7 text (DAT-001). Enumerations MUST be lowercase snake_case strings. |
| API-006 · P0 | Resource names MUST be plural nouns. `GET` MUST NOT change state. `POST` creates a resource or runs an action on one. `PATCH` MUST use `application/merge-patch+json` (RFC 7396) for partial updates to policy, listings, and credit limits. `DELETE` MUST be used only for removal of revocable items (for example tokens); ledger items are never deleted. |
| API-007 · P0 | Balance views MUST include the derived field `direction` with value `in_credit` (balance > 0), `in_debt` (balance < 0) or `zero`, so that every client presents direction consistently with 08. |

### 2.1 Resource overview

| Resource | Purpose |
|---|---|
| `/network` | Public-to-members network settings relevant to clients (name, decimals, currency name, join mode, policy values visible to members) |
| `/me` | The caller's person, memberships and roles in the current network |
| `/members` | Directory (members-only, per network visibility) |
| `/accounts/me` and `/accounts/me/entries` | Own balance views and own ledger history |
| `/listings` | Offers and requests |
| `/transactions` and actions | Proposals, acceptance, decline, cancellation, status history |
| `/disputes` | Dispute cases raised by members and resolved by reviewers |
| `/exports` | Export jobs and artefact download |
| `/tokens` | Personal API tokens of the caller |
| `/admin/...` | Administrator-only operations; reviewer-only operations are under `/disputes` |

Admin routes are a separate prefix so that authorisation policy, rate limits and logs can be applied and reviewed as a group (API-037).

## 3. Authentication

| Rule | Requirement |
|---|---|
| API-008 · P0 | Browser clients MUST authenticate with a session cookie issued by the sign-in flow (HTML). The cookie MUST be `HttpOnly`, `Secure` when served over TLS, `SameSite=Lax` or stricter, and scoped to the network path or host (prefix `__Host-` where possible). Session lifetime and renewal are defined in 09. |
| API-009 · P1 | Non-browser clients MUST authenticate with a personal API token presented as `Authorization: Bearer <token>`. Tokens MUST be generated by the platform with at least 256 bits of randomness, MUST be shown once at creation, and MUST be stored only as a salted or keyed hash. |
| API-010 · P1 | Each token MUST have a name, one or more scopes from the set `read` (read-only operations on own data), `write` (own commands: listings, proposals, acceptance, decline, cancel, disputes) and `admin` (administrative commands; still subject to role checks), an expiry no longer than 365 days (default; product decision), and a revocation endpoint. |
| API-011 · P1 | A token MUST NOT authorise token creation, sign-in, or changes to a token's own scopes. A token's effective permissions MUST be the intersection of its scopes and the caller's current roles and membership status. |
| API-012 · P0 | Credentials MUST NOT be accepted in query strings. A request carrying a session identifier or token in a query string MUST be rejected with 400 (`credentials-in-query`), the credential value MUST NOT be written to logs or echoed in the response, and the incident handling in 09 applies. |
| API-013 · P0 | Unauthenticated requests to non-public operations MUST return 401 with `WWW-Authenticate` (RFC 9110). Authenticated but unauthorised requests MUST return 403, except where API-017 requires 404. |
| API-014 · P0 | Cookie-authenticated requests with unsafe methods (POST, PATCH, PUT, DELETE) MUST include a per-session CSRF token in a request header; requests without a valid token MUST be rejected with 403 (`csrf-invalid`). Token-authenticated requests are exempt because they are not ambient. |
| API-015 · P0 | Cross-origin browser requests MUST NOT receive credentialed CORS responses. The API MUST NOT enable CORS for cookie-authenticated routes by default. |

## 4. Authorisation

| Rule | Requirement |
|---|---|
| API-016 · P0 | Every operation MUST check the caller's membership status and role against the operation's policy in 03 and 04 before doing any work. Suspended members MUST be able to read their own data but MUST NOT be able to create or accept transactions or listings (exact rule in 03). |
| API-017 · P0 | A request for a resource that exists in another network, or that the caller is not permitted to see by visibility rules, MUST return 404 with the same error body as for a non-existent identifier (existence not revealed). Permission failures on visible resources return 403. |
| API-018 · P0 | Admin operations MUST require the network administrator role on the caller's current membership. Reviewer operations MUST require the dispute reviewer role, and reviewers MUST NOT decide disputes they raised or are party to. |
| API-019 · P0 | The platform operator role MUST NOT have any route under `/api/v1` that mutates network data (ARC-008). |

## 5. Idempotency

| Rule | Requirement |
|---|---|
| API-020 · P0 | Every command that changes ledger state, membership, or dispute state MUST accept an `Idempotency-Key` request header and MUST be idempotent for at least 24 hours (see 06 section 9 for the recommended 7-day default). The operations marked "MUST" in 06 section 6.1 are the required set. |
| API-021 · P0 | The key MUST be 1 to 255 printable ASCII characters (RFC 9110 field value rules). Clients SHOULD use UUIDv7 or another random value of at least 128 bits. Missing keys on required operations MUST return 400 (`idempotency-key-required`). |
| API-022 · P0 | A retry with the same key and an equivalent body MUST return the status and body of the original response with the header `Idempotent-Replayed: true`. The replay MUST NOT perform any posting, audit write or state change. |
| API-023 · P0 | Reusing a key with a different request body (after normalisation) MUST return 422 (`idempotency-key-reused`) and perform no state change. |
| API-024 · P0 | If a request with the same key is still in progress, the second request MUST return 409 (`idempotency-request-in-progress`) with `Retry-After` (seconds). It MUST NOT wait and then execute a second time. |
| API-025 · P0 | A command that fails without changing state (for example, validation error or insufficient credit) MUST NOT be replayed as a success. Whether its failure response is replayed for the retention period or the key may be reused immediately is decided in OQ-API-04; the chosen behaviour MUST be documented and tested. |

## 6. Concurrency control

| Rule | Requirement |
|---|---|
| API-026 · P0 | Responses for mutable resources with conflict risk (policy, listing, credit limit, membership status) MUST include a strong `ETag`. Updates to these resources MUST include `If-Match` with the current ETag. |
| API-027 · P0 | A mismatched `If-Match` MUST return 412 (`precondition-failed`) and no change. A missing `If-Match` on such a resource MUST return 428 (`precondition-required`). |
| API-028 · P0 | Transaction actions MUST be validated against the current status inside the atomic unit (06). An action invalid for the current status MUST return 409 (`invalid-transaction-state`) with the current status in the problem body. |

## 7. Errors, pagination and limits

### 7.1 Error format

| Rule | Requirement |
|---|---|
| API-029 · P0 | All error responses MUST use `application/problem+json` as defined by RFC 9457 with the members `type`, `title`, `status`, `detail`, and `instance`. |
| API-030 · P0 | Errors MUST include the extension members `code` (stable machine-readable code from the catalogue in 7.2), `request_id` (the same value as the `X-Request-Id` response header), and, for validation failures, `errors`: an array of objects with `pointer` (RFC 6901 JSON pointer into the request body), `code`, and `detail`. |
| API-031 · P0 | `type` MUST be a stable URN of the form `urn:circula:problem:<code>`, not a URL that must be fetched. `title` MUST be a short, English, human-readable summary that does not vary with detail. `detail` MUST NOT contain stack traces, SQL, internal identifiers from other networks, or personal data of other members. |
| API-032 · P0 | Server errors (5xx) MUST NOT reveal internal details. The response MUST contain `code` `internal-error` and the `request_id`, and the cause MUST be recorded in server logs (ARC-022). |
| API-033 · P1 | `detail` text MUST be in English in the MVP. Clients MUST use `code` for logic and MUST NOT parse `detail`. |

### 7.2 Error code catalogue (MVP)

The catalogue below is the stable set. New codes MAY be added in a minor revision (see API-074). Clients MUST treat unknown codes as the generic error for their status.

| Code | Status | Meaning |
|---|---|---|
| `invalid-body` | 400 | Malformed JSON, unknown field, duplicate key |
| `invalid-query` | 400 | Unsupported query parameter value |
| `invalid-cursor` | 400 | Cursor not valid for this request |
| `member-has-open-balance` | 409 | Member departure or removal refused while a non-zero balance exists (rule in 03) |
| `credentials-in-query` | 400 | Credential supplied in the URL |
| `idempotency-key-required` | 400 | Required header absent |
| `invalid-header` | 400 | A header value is malformed (for example an `Idempotency-Key` that is not 1 to 255 printable ASCII characters) |
| `validation-failed` | 422 | One or more fields invalid (see `errors`) |
| `insufficient-available-credit` | 422 | Payer's available credit would be exceeded (LED-5) |
| `amount-exceeds-policy` | 422 | Amount above the network's maximum transaction amount |
| `counterparty-not-active` | 422 | Counterparty is not an active member |
| `idempotency-key-reused` | 422 | Key reused with a different body |
| `unauthenticated` | 401 | No valid credentials |
| `forbidden` | 403 | Authenticated but not permitted |
| `csrf-invalid` | 403 | Missing or invalid CSRF token |
| `not-found` | 404 | Resource does not exist or is not visible to the caller (also used for cross-network) |
| `invalid-transaction-state` | 409 | Action not allowed in the current state |
| `idempotency-request-in-progress` | 409 | Same key currently being processed |
| `precondition-failed` | 412 | `If-Match` does not match |
| `precondition-required` | 428 | `If-Match` required |
| `payload-too-large` | 413 | Body exceeds limit |
| `unsupported-media-type` | 415 | Wrong or missing content type |
| `rate-limited` | 429 | Rate limit exceeded; `Retry-After` present |
| `internal-error` | 500 | Unexpected failure |
| `unavailable` | 503 | Temporarily unavailable (for example maintenance); `Retry-After` present |

### 7.3 Pagination

| Rule | Requirement |
|---|---|
| API-034 · P0 | List operations MUST use cursor pagination. Requests take `limit` (integer 1 to 100, default 25) and `cursor` (opaque string). Responses use the envelope `{"data": [...], "next_cursor": "<string or null>"}`. |
| API-035 · P0 | Ordering MUST be stable and total: a unique sort key (for example `ledger_seq` or creation sequence, then id). Cursors MUST encode the position and MUST be integrity-protected so that they cannot be forged to access other rows. A cursor used with different filters MUST return 400 (`invalid-cursor`). |
| API-036 · P1 | Total counts are not required. Listing endpoints MUST NOT run unbounded queries; an endpoint without pagination MUST return at most 100 rows. |

### 7.4 Request limits and rate limiting

| Rule | Requirement |
|---|---|
| API-037 · P0 | Request bodies MUST be limited to 64 KiB by default. Oversize bodies MUST be rejected with 413 before parsing. |
| API-038 · P1 | Rate limits MUST apply per authenticated principal and per client IP for unauthenticated and sign-in routes. Exceeding them MUST return 429 with `Retry-After`. Initial limits are product decisions recorded in 09. |
| API-039 · P0 | Responses containing balances, ledger entries, transactions, disputes, exports, or token metadata MUST carry `Cache-Control: no-store`. |

## 8. Operations (MVP)

Each operation has one ID. Columns: method and path; minimum role; idempotency requirement from 06 section 6.1; priority; acceptance. Unless stated, responses are JSON, reads return `Cache-Control: no-store` for balances, entries, transactions and disputes (API-039).

| ID | Operation | Role | Idem. | Pri. | Acceptance (testable) |
|---|---|---|---|---|---|
| API-040 | `GET /network` | Member | No | P0 | Returns name, decimals, currency name, join mode, dispute window, proposal expiry. Does not return other members' data. |
| API-041 | `GET /me` | Member | No | P0 | Returns person id, memberships in this network, status and roles. Equal across HTML and API for the same session. |
| API-042 | `GET /members` | Member | No | P0 | Paginated directory respecting visibility. Each item has id, display name, status `active` only. Other networks' members never appear (ARC-007). |
| API-043 | `GET /members/{id}` | Member | No | P0 | Visible member detail; not-visible or other network returns 404 (API-017). |
| API-044 | `GET /accounts/me` | Member | No | P0 | Returns `balance_minor`, `credit_limit_minor`, `reserved_minor`, `available_minor` (DAT-018), `direction` (API-007). Values equal the domain computation in tests. |
| API-045 | `GET /accounts/me/entries` | Member | No | P0 | Paginated entries ordered by `ledger_seq` descending; each entry has signed amount, counterparty display name, transaction id, kind, status, `ledger_seq`. Running balance is optional (P2). |
| API-046 | `GET /listings`, `POST /listings` | Member (create requires active) | Create: SHOULD | P0 | List filters by kind and text; create validates kind (`offer` or `request`), title length, and description length. Response includes ETag. |
| API-047 | `PATCH /listings/{id}` | Owner member | No | P1 | Merge-patch with `If-Match` (API-026/027). Non-owner gets 403 or 404 per API-017. |
| API-048 | `POST /transactions` | Active member | MUST | P0 | Body: `counterparty_member_id`, `direction` (`pay` or `receive`, from proposer's view), `amount_minor` (string), optional `listing_id`, `description`. Creates `proposed` transaction with `expires_at` per policy. Refused with `insufficient-available-credit` when the payer's available credit would be exceeded, with no state change. Returns 201 with `Location`. |
| API-049 | `GET /transactions`, `GET /transactions/{id}` | Member (party to transaction) | No | P0 | List filters by `status` and `role` (`payer`, `payee`). Detail includes `my_direction`, current status and status history (API-050 link). Non-party gets 404. |
| API-050 | `GET /transactions/{id}/events` | Party | No | P0 | Append-only status history with `from`, `to`, `actor_type`, `occurred_at`, paginated. |
| API-051 | `POST /transactions/{id}/accept` | Counterparty (not proposer) | MUST | P0 | Allowed only from `proposed`. Atomically posts both entries and returns 200 with status `completed`, `posted_ledger_seq`, and updated own balance view. A failed limit check leaves status `proposed` and returns `insufficient-available-credit`. Proposer accepting own proposal returns 403. |
| API-052 | `POST /transactions/{id}/decline` | Counterparty | MUST | P0 | Allowed only from `proposed`; status becomes `declined`; no entries written; audit event written. |
| API-053 | `POST /transactions/{id}/cancel` | Proposer | MUST | P0 | Allowed only from `proposed`; status becomes `cancelled`; no entries written. From `completed`, cancellation is not allowed; correction uses reversal (API-056 scope). |
| API-054 | `POST /disputes`, `GET /disputes`, `GET /disputes/{id}` | Party to transaction (create); party or reviewer (read) | Create: MUST | P1 | Create only for a `completed` transaction within the dispute window (policy). Second open dispute on the same transaction returns 409 (`invalid-transaction-state`). Non-party gets 404. |
| API-055 | `POST /disputes/{id}/resolve` | Dispute reviewer (not a party) | MUST | P1 | Body: `outcome` (`no_action`, `reversal`, `adjustment`), `reason`. Reversal or adjustment created through the same ledger command as the admin operations, with audit. Reviewer who is a party gets 403. |
| API-056 | `POST /admin/transactions/{id}/reversal` | Network administrator or reviewer (via dispute) | MUST | P1 | Creates kind `reversal` (DAT-022). Second reversal of the same transaction returns 409. Requires a reason. |
| API-057 | `POST /admin/adjustments` | Network administrator | MUST | P1 | Body: `member_id`, `direction`, `amount_minor`, `reason` (mandatory, DAT-013). Posts against system account. Refused with `insufficient-available-credit` for debits beyond limit unless policy override is requested with reason (DAT-024). |
| API-058 | `GET /admin/policy`, `PATCH /admin/policy` | Network administrator | No (PATCH MUST be idempotent by ETag) | P0 | GET returns the current policy version and ETag. PATCH with `If-Match` creates a new policy version with a required `reason`; changes take effect for new transactions only (rule in 00 and 03). Response includes `effective_at`. |
| API-059 | `GET /admin/members`, `POST /admin/members/{id}/approve`, `/deny`, `/suspend`, `/reinstate`, `/remove` | Network administrator | MUST on POST | P0 | Lists members by status. Each action writes an audit event. `remove` on a member with a non-zero balance returns 409 (`invalid-transaction-state` is not used; code `member-has-open-balance` is reserved for the departure rule in 03) until 03's departure rule is applied. |
| API-060 | `PUT /admin/accounts/{id}/credit-limit` | Network administrator | No (ETag) | P0 | Body: `credit_limit_minor` (>= 0 string). Requires `If-Match`, `reason`, and writes an audit event with before/after. Lowering below the current debt does not change balances; it blocks new debits (03). |
| API-061 | `POST /admin/invitations`, `POST /invitations/{token}/accept` | Administrator; invitee | MUST | P1 | Invitation token shown once, stored hashed, expires per policy; accepting creates membership in the state required by join mode. |
| API-062 | `GET /admin/audit-events` | Network administrator | No | P0 | Paginated, filterable by `action`, `target_id`, `actor_id`, date range. Access is itself audited (DAT-034). Returns only fields in 06 section 7.1. |
| API-063 | `POST /admin/consistency-checks`, `GET /admin/consistency-checks/{id}` | Network administrator | MUST on POST | P0 | Runs ARC-015 checks and returns a result with a summary; no amounts for named members in the summary beyond what 06 section 7.3 allows. |
| API-064 | `POST /admin/announcements`, `GET /admin/announcements` (and member read via `/network/announcements`) | Administrator (write); member (read) | MUST on POST | P1 | Announcements are append-only and carry `effective_at`. Policy changes SHOULD be linked to an announcement (10). |
| API-065 | `POST /exports`, `GET /exports/{id}`, `GET /exports/{id}/artefact` | Member (own data) or administrator (network) | SHOULD on POST | P1 | Creates a job; status `queued`, `running`, `complete`, `failed`. Artefact download is available only when complete and checksum-verified (DAT-040); expires per 06 section 9. |
| API-066 | `GET /tokens`, `POST /tokens`, `DELETE /tokens/{id}` | Authenticated person (session auth only for create) | SHOULD on POST | P1 | Create requires session authentication (API-011). Token value appears once in the 201 response. Delete revokes immediately; revoked tokens fail with 401. |
| API-067a | `POST /me/password` | Signed-in person, session cookie only | MAY | P0 | Body `current_password`, `new_password`. `204` on success; every session of the person is revoked, API tokens are not. A wrong current password is `422 validation-failed` at `/current_password` (not a `401`), limited to 5 attempts per minute per person. Implements AUTH-03 / SEC-003. |
| API-067 | `POST /me/membership/departure` | Member | MUST | P1 | Starts departure per 03 and 10; response states the settlement status. Does not delete ledger entries (LED principles, 06). |

**Operation count note.** Operations API-040 to API-067 are the MVP conceptual set. Identifiers in the 040-067 range are contract identifiers; the paths above are a conceptual shape for the contract and MAY be adjusted during detailed design without changing the acceptance criteria.

## 9. Ledger invariants through the API

| Rule | Requirement |
|---|---|
| API-068 · P0 | No API response MAY report a successful state change that did not commit. A success response (2xx) MUST occur only after the storage unit commits (LED-4). |
| API-069 · P0 | Every write response that changes balances MUST return the resulting account view or the posting result (`posted_ledger_seq`) so that clients need not compute balances. |
| API-070 · P0 | The API MUST NOT accept client-supplied balances, ledger entries, cached totals, `ledger_seq` values, or audit events. Such fields in a request MUST be rejected (`invalid-body`). |
| API-071 · P0 | Contract tests MUST run the same scenario set through the HTML and API surfaces and compare ledger outcomes (entries, balances, audit events, status). Scenarios include: successful trade, declined trade, insufficient credit at acceptance, concurrent acceptances on one account, retry with the same key, key reuse with a different body, and reversal. |
| API-072 · P0 | An invariant violation found during a request MUST return 500 (`internal-error`), roll back the request, and raise an alert; it MUST NOT return a partial success. |

## 10. Versioning and compatibility

| Rule | Requirement |
|---|---|
| API-073 · P0 | The major version appears in the path (`/api/v1`). A breaking change requires a new major version. |
| API-074 · P0 | Non-breaking changes within v1 are: adding optional request fields, adding response fields, adding endpoints, adding error codes, adding enumeration values that clients are documented to ignore when unknown. Clients MUST ignore unknown response fields. |
| API-075 · P0 | Breaking changes include: removing or renaming fields, changing types or the meaning of a field, changing ledger semantics, changing money representation, changing idempotency or status rules in a way that alters outcomes, and removing enum values. |
| API-076 · P1 | When a v1 operation is deprecated, responses MUST include `Deprecation` (RFC 9745) and `Sunset` (RFC 8594) headers. Deprecation MUST be announced with the notice period in 10 (recommended: at least 6 months for self-hosted operators, product decision). |
| API-077 · P1 | The API MUST publish a machine-readable description (OpenAPI 3.1 recommended) per release, and contract tests MUST validate responses against it. |
| API-078 · P1 | Backward-compatibility tests MUST run each release against recorded requests from the previous release (fixtures from 12). |

## 11. Out of MVP scope and open questions

| ID | Item | Priority |
|---|---|---|
| API-079 | Outbound webhooks or event streams | P2 (deferred; see 01) |
| API-080 | Third-party integrations and OAuth provider role | P2 (deferred) |
| API-081 | Language negotiation (`Accept-Language`) for error details | P2 |
| API-082 | GraphQL or other query-style interfaces | Not planned for MVP; any proposal requires a change to this document |

Open questions:

- **OQ-API-01 (decision needed):** Which party proposes and which accepts for `direction`. This spec assumes any member may propose; the counterparty alone may accept. Confirm in 03.
- **OQ-API-02 (decision needed):** Hosted URL structure (network in path vs host name). Recommended default: deployment-level context; decided with 05 OQ-ARC-02.
- **OQ-API-03 (decision needed):** Default token expiry and rate-limit numbers (API-010, API-038).
- **OQ-API-04 (decision needed):** Replay policy for failed commands (API-025): recommended default is to replay failures for the retention period so retries are deterministic, but to allow a fresh attempt if the key was not stored.
- **OQ-API-05 (assumption):** A proposal is a single `proposed` transaction; there is no counter-proposal chain in MVP. A counter-proposal is a new proposal.
- **OQ-API-06 (decision needed):** Whether `GET /network` is visible to unauthenticated visitors (join page) or members only. Recommended default: a minimal public subset, with join mode visible.

Cross-references: domain rules [03-domain-model.md](03-domain-model.md); ledger data [06-data-ledger-audit.md](06-data-ledger-audit.md); UX [08-ux-accessibility.md](08-ux-accessibility.md); security [09-security-privacy.md](09-security-privacy.md); operations [11-deployment-operations.md](11-deployment-operations.md); test plan [12-quality-test-plan.md](12-quality-test-plan.md); decisions [13-assumptions-risks-decisions.md](13-assumptions-risks-decisions.md).
