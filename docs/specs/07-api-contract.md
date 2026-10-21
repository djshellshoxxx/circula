# 07 API and Integration Contract

Status: Draft v0.1 · Owner/role: API lead · Last updated: 2026-10-21
Scope: Principles, authentication and authorisation, error format, idempotency, pagination, concurrency control, versioning and compatibility, and the conceptual MVP operations of the JSON API under `/api/v1`. Does not select a framework or wire-level library beyond the standards cited. Consistent with DEC-002 (server-rendered HTML plus versioned JSON API over one domain layer).
Dependencies: [00-conventions.md](00-conventions.md) (glossary, LED-1..LED-7, DEC-002); [03-domain-model.md](03-domain-model.md) (rules behind each operation); [05-architecture.md](05-architecture.md) (ARC-002, ARC-003, ARC-007); [06-data-ledger-audit.md](06-data-ledger-audit.md) (idempotency records, money and identifier representation); [08-ux-accessibility.md](08-ux-accessibility.md) (shared presentation of balances and errors); [09-security-privacy.md](09-security-privacy.md) (authentication, rate limits, logging); [12-quality-test-plan.md](12-quality-test-plan.md) (contract tests).
Open questions: see section 11.
As-built: the machine-readable description is [docs/api/openapi.yaml](../api/openapi.yaml) (OpenAPI 3.1, 53 operations, one per route in `internal/api/routes.go`). Where the implementation differs from the requirements below, the row carries an **As built** note and section 12 lists the deviation. Requirement IDs are not renumbered.

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
| API-008 · P0 | Browser clients MUST authenticate with a session cookie issued by the sign-in flow (HTML). The cookie MUST be `HttpOnly`, `Secure` when served over TLS, `SameSite=Lax` or stricter, and scoped to the network path or host (prefix `__Host-` where possible). Session lifetime and renewal are defined in 09. **As built:** the cookie is `__Host-circula_session` over TLS and `circula_session` otherwise (`internal/httpx`); readers accept either. |
| API-009 · P1 | Non-browser clients MUST authenticate with a personal API token presented as `Authorization: Bearer <token>`. Tokens MUST be generated by the platform with at least 256 bits of randomness, MUST be shown once at creation, and MUST be stored only as a salted or keyed hash. **As built:** tokens carry the prefix `cir_` and are stored hashed. An `Authorization` header that is present but not a valid Bearer credential is 401, even when a session cookie is also sent. |
| API-010 · P1 | Each token MUST have a name, one or more scopes from the set `read` (read-only operations on own data), `write` (own commands: listings, proposals, acceptance, decline, cancel, disputes) and `admin` (administrative commands; still subject to role checks), an expiry no longer than 365 days (default; product decision), and a revocation endpoint. **As built:** scopes `read` and `write` only; requesting `admin` returns 501 `not-implemented` (deviation D-07). Default and maximum expiry are 365 days. |
| API-011 · P1 | A token MUST NOT authorise token creation, sign-in, or changes to a token's own scopes. A token's effective permissions MUST be the intersection of its scopes and the caller's current roles and membership status. |
| API-012 · P0 | Credentials MUST NOT be accepted in query strings. A request carrying a session identifier or token in a query string MUST be rejected with 400 (`credentials-in-query`), the credential value MUST NOT be written to logs or echoed in the response, and the incident handling in 09 applies. **As built:** the check covers the parameter names in the list in `internal/api/middleware.go` and any query value starting with `cir_` or `inv_`. |
| API-013 · P0 | Unauthenticated requests to non-public operations MUST return 401 with `WWW-Authenticate` (RFC 9110). Authenticated but unauthorised requests MUST return 403, except where API-017 requires 404. |
| API-014 · P0 | Cookie-authenticated requests with unsafe methods (POST, PATCH, PUT, DELETE) MUST include a per-session CSRF token in a request header; requests without a valid token MUST be rejected with 403 (`csrf-invalid`). Token-authenticated requests are exempt because they are not ambient. **As built:** the header is `X-CSRF-Token`; the token is returned in the `X-CSRF-Token` response header of every cookie-authenticated response. The check runs after authentication and before network resolution. |
| API-015 · P0 | Cross-origin browser requests MUST NOT receive credentialed CORS responses. The API MUST NOT enable CORS for cookie-authenticated routes by default. |

## 4. Authorisation

| Rule | Requirement |
|---|---|
| API-016 · P0 | Every operation MUST check the caller's membership status and role against the operation's policy in 03 and 04 before doing any work. Suspended members MUST be able to read their own data but MUST NOT be able to create or accept transactions or listings (exact rule in 03). |
| API-017 · P0 | A request for a resource that exists in another network, or that the caller is not permitted to see by visibility rules, MUST return 404 with the same error body as for a non-existent identifier (existence not revealed). Permission failures on visible resources return 403. |
| API-018 · P0 | Admin operations MUST require the network administrator role on the caller's current membership. Reviewer operations MUST require the dispute reviewer role, and reviewers MUST NOT decide disputes they raised or are party to. **As built:** dispute decisions (review, resolve, withdraw is opener-only, retry) accept either the reviewer or the administrator role, and the independence rule (DOM-037/038) applies to both. Administrators may read any transaction of their network (see API-049). |
| API-019 · P0 | The platform operator role MUST NOT have any route under `/api/v1` that mutates network data (ARC-008). |

## 5. Idempotency

| Rule | Requirement |
|---|---|
| API-020 · P0 | Every command that changes ledger state, membership, or dispute state MUST accept an `Idempotency-Key` request header and MUST be idempotent for at least 24 hours (see 06 section 9 for the recommended 7-day default). The operations marked "MUST" in 06 section 6.1 are the required set. |
| API-021 · P0 | The key MUST be 1 to 255 printable ASCII characters (RFC 9110 field value rules). Clients SHOULD use UUIDv7 or another random value of at least 128 bits. Missing keys on required operations MUST return 400 (`idempotency-key-required`). **As built:** a malformed key returns 400 `invalid-header`. The edge and the service layer both enforce the 255-character limit (D-03 closed). |
| API-022 · P0 | A retry with the same key and an equivalent body MUST return the status and body of the original response with the header `Idempotent-Replayed: true`. The replay MUST NOT perform any posting, audit write or state change. **As built:** the replay record (status, Content-Type, Location, ETag, body) is kept per person and key in process memory for 24 hours, at most 10,000 records (`internal/api/idem.go`). It is lost on restart, so a retry after a restart re-enters the service, which has its own durable idempotency (7 days, shared with HTML). This is deviation D-04. |
| API-023 · P0 | Reusing a key with a different request body (after normalisation) MUST return 422 (`idempotency-key-reused`) and perform no state change. |
| API-024 · P0 | If a request with the same key is still in progress, the second request MUST return 409 (`idempotency-request-in-progress`) with `Retry-After` (seconds). It MUST NOT wait and then execute a second time. |
| API-025 · P0 | A command that fails without changing state (for example, validation error or insufficient credit) MUST NOT be replayed as a success. Whether its failure response is replayed for the retention period or the key may be reused immediately is decided in OQ-API-04; the chosen behaviour MUST be documented and tested. **As built (OQ-API-04 chosen):** failures are not cached; the key stays reusable. Only 2xx responses are replayed. |

## 6. Concurrency control

| Rule | Requirement |
|---|---|
| API-026 · P0 | Responses for mutable resources with conflict risk (policy, listing, credit limit, membership status) MUST include a strong `ETag`. Updates to these resources MUST include `If-Match` with the current ETag. **As built:** the ETag is returned by `GET /admin/policy`, `PATCH /admin/policy`, `GET /admin/accounts/{id}`, the credit-limit PUT, the listing POST and PATCH responses, and the member action and role responses. It is also returned by `GET /listings/{id}`, which honours `If-None-Match` with 304 (gap G-1 closed). It is not returned by `GET /members`, `GET /members/{id}`, `GET /network` or listing lists. `If-Match` is required on policy, listing and credit-limit updates, but optional on member actions (`approve`, `deny`, `suspend`, `reinstate`, `remove`) and on role changes, which do not require it (deviation D-05). |
| API-027 · P0 | A mismatched `If-Match` MUST return 412 (`precondition-failed`) and no change. A missing `If-Match` on such a resource MUST return 428 (`precondition-required`). **As built:** applies to the required resources above; see D-05 for the optional ones. |
| API-028 · P0 | Transaction actions MUST be validated against the current status inside the atomic unit (06). An action invalid for the current status MUST return 409 (`invalid-transaction-state`) with the current status in the problem body. **As built:** `current_status` is added for accept, decline, cancel and the administrative reversal; it is not added to dispute actions. |

## 7. Errors, pagination and limits

### 7.1 Error format

| Rule | Requirement |
|---|---|
| API-029 · P0 | All error responses MUST use `application/problem+json` as defined by RFC 9457 with the members `type`, `title`, `status`, `detail`, and `instance`. |
| API-030 · P0 | Errors MUST include the extension members `code` (stable machine-readable code from the catalogue in 7.2), `request_id` (the same value as the `X-Request-Id` response header), and, for validation failures, `errors`: an array of objects with `pointer` (RFC 6901 JSON pointer into the request body), `code`, and `detail`. **As built:** `errors[].code` is an open set of field codes (for example `invalid`, `invalid-enum`, `invalid-amount`, `invalid-type`, `invalid-timestamp`, `out-of-range`, `required`, `incorrect`); clients MUST NOT rely on it beyond the presence of the member. Pointers use the API field names (`amount_minor`, not `amount`). Blocked refusals carry `blockers` (string array). |
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
| `member-has-open-balance` | 409 | Reserved. Member departure or removal refused while a non-zero balance exists (rule in 03). **Not emitted as built**; the refusal uses `blocked` (deviation D-06). |
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
| `method-not-allowed` | 405 | Added as built (API-074 minor addition). Path exists with other methods; `Allow` header lists them |
| `not-implemented` | 501 | Added as built (API-074). Feature not provided by the service layer: token scope `admin` (API-010), dispute outcome `adjustment` (API-055) |
| `network-required` | 400 | Added as built. Caller belongs to several networks and sent no `X-Network` header |
| `blocked` | 409 | Added as built. Action refused by a rule with a `blockers` list (departure or removal with a disputed transaction or as last active administrator; see D-06) |
| `conflict` | 409 | Added as built. Store-level duplicate or conflicting state that the service maps to 409 |

### 7.3 Pagination

| Rule | Requirement |
|---|---|
| API-034 · P0 | List operations MUST use cursor pagination. Requests take `limit` (integer 1 to 100, default 25) and `cursor` (opaque string). Responses use the envelope `{"data": [...], "next_cursor": "<string or null>"}`. **As built:** when a filtered scan reaches its work cap (20 rounds of 100 rows) before the page is full, a page may hold fewer than `limit` items with a non-null `next_cursor`. Clients MUST follow `next_cursor` until it is null, not stop on a short page. |
| API-035 · P0 | Ordering MUST be stable and total: a unique sort key (for example `ledger_seq` or creation sequence, then id). Cursors MUST encode the position and MUST be integrity-protected so that they cannot be forged to access other rows. A cursor used with different filters MUST return 400 (`invalid-cursor`). |
| API-036 · P1 | Total counts are not required. Listing endpoints MUST NOT run unbounded queries; an endpoint without pagination MUST return at most 100 rows. |

### 7.4 Request limits and rate limiting

| Rule | Requirement |
|---|---|
| API-037 · P0 | Request bodies MUST be limited to 64 KiB by default. Oversize bodies MUST be rejected with 413 before parsing. |
| API-038 · P1 | Rate limits MUST apply per authenticated principal and per client IP for unauthenticated and sign-in routes. Exceeding them MUST return 429 with `Retry-After`. Initial limits are product decisions recorded in 09. **As built:** fixed one-minute windows in process memory: 600 requests per minute per principal and 1,200 per minute per client IP, applied to every `/api/v1` route (not only unauthenticated ones). Password change has a further cap of 5 per minute per person (API-067a). Defaults are settable (`Options`). Deviation D-08 (IP limit scope; in-memory, not shared between instances). |
| API-039 · P0 | Responses containing balances, ledger entries, transactions, disputes, exports, or token metadata MUST carry `Cache-Control: no-store`. |

## 8. Operations (MVP)

Each operation has one ID. Columns: method and path; minimum role; idempotency requirement from 06 section 6.1; priority; acceptance. Unless stated, responses are JSON, reads return `Cache-Control: no-store` for balances, entries, transactions and disputes (API-039).

| ID | Operation | Role | Idem. | Pri. | Acceptance (testable) |
|---|---|---|---|---|---|
| API-040 | `GET /network` | Member | No | P0 | Returns name, decimals, currency name, join mode, dispute window, proposal expiry. Does not return other members' data. **As built:** requires authentication (no public route; OQ-API-06 resolved as members-only). Returns no ETag. |
| API-041 | `GET /me` | Member | No | P0 | Returns person id, memberships in this network, status and roles. Equal across HTML and API for the same session. **As built (D-01):** returns the person's memberships across all networks visible to the credential, not only the current network; the route does not resolve a network. |
| API-042 | `GET /members` | Member | No | P0 | Paginated directory respecting visibility. Each item has id, display name, status `active` only. Other networks' members never appear (ARC-007). **As built:** `status` is always `active`; the directory has no status filter. |
| API-043 | `GET /members/{id}` | Member | No | P0 | Visible member detail; not-visible or other network returns 404 (API-017). |
| API-044 | `GET /accounts/me` | Member | No | P0 | Returns `balance_minor`, `credit_limit_minor`, `reserved_minor`, `available_minor` (DAT-018), `direction` (API-007). Values equal the domain computation in tests. **As built:** also returns `member_id`, `account_state` (active, frozen, closed), `awaiting_from_others_minor`, `decimals`, `ledger_seq`. |
| API-045 | `GET /accounts/me/entries` | Member | No | P0 | Paginated entries ordered by `ledger_seq` descending; each entry has signed amount, counterparty display name, transaction id, kind, status, `ledger_seq`. Running balance is optional (P2). |
| API-046 | `GET /listings`, `POST /listings` | Member (create requires active) | Create: SHOULD | P0 | List filters by kind and text; create validates kind (`offer` or `request`), title length, and description length. Response includes ETag. **As built:** create requires `Idempotency-Key` (400 `idempotency-key-required` without it), which is stricter than SHOULD (D-09). List filters are `kind`, `q` (max 200 characters) and `member_id`. Title is 1 to 120 characters, description at most 2,000. |
| API-047 | `PATCH /listings/{id}` | Owner member | No | P1 | Merge-patch with `If-Match` (API-026/027). Non-owner gets 403 or 404 per API-017. **As built:** a non-owner gets 404. Patchable fields are `title`, `description`, `price_minor` (null clears) and `status` (`active`, `paused`, `archived`). |
| API-048 | `POST /transactions` | Active member | MUST | P0 | Body: `counterparty_member_id`, `direction` (`pay` or `receive`, from proposer's view), `amount_minor` (string), optional `listing_id`, `description`. Creates `proposed` transaction with `expires_at` per policy. Refused with `insufficient-available-credit` when the payer's available credit would be exceeded, with no state change. Returns 201 with `Location`. |
| API-049 | `GET /transactions`, `GET /transactions/{id}` | Member (party to transaction); administrator of the network for detail | No | P0 | List filters by `status` and `role` (`payer`, `payee`). Detail includes `my_direction`, current status and status history (API-050 link). Non-party gets 404. **As built (aligned):** on the detail read, a network administrator may read any transaction in their own network, and a dispute reviewer may read a transaction on which a dispute exists; other non-parties get 404. The list is always the caller's own transactions. Transactions from other networks are 404 for everyone. |
| API-050 | `GET /transactions/{id}/events` | Party | No | P0 | Append-only status history with `from`, `to`, `actor_type`, `occurred_at`, paginated. |
| API-051 | `POST /transactions/{id}/accept` | Counterparty (not proposer) | MUST | P0 | Allowed only from `proposed`. Atomically posts both entries and returns 200 with status `completed`, `posted_ledger_seq`, and updated own balance view. A failed limit check leaves status `proposed` and returns `insufficient-available-credit`. Proposer accepting own proposal returns 403. |
| API-052 | `POST /transactions/{id}/decline` | Counterparty | MUST | P0 | Allowed only from `proposed`; status becomes `declined`; no entries written; audit event written. |
| API-053 | `POST /transactions/{id}/cancel` | Proposer | MUST | P0 | Allowed only from `proposed`; status becomes `cancelled`; no entries written. From `completed`, cancellation is not allowed; correction uses reversal (API-056 scope). |
| API-054 | `POST /disputes`, `GET /disputes`, `GET /disputes/{id}` | Party to transaction (create); party, reviewer or administrator (read) | Create: MUST | P1 | Create only for a `completed` transaction within the dispute window (policy). Second open dispute on the same transaction returns 409 (`invalid-transaction-state`). Non-party gets 404. **As built:** body `transaction_id`, `category` (`not_delivered`, `quality`, `wrong_amount`, `unauthorised`, `other`), `statement`. Only trade transactions can be disputed. The detail includes the transaction, both display names and the event history. |
| API-055 | `POST /disputes/{id}/resolve` | Dispute decider: administrator or dispute reviewer, not a party (DOM-037, DOM-038) | MUST | P1 | Body: `outcome`, `reason`, optional `amount_minor`. **As built (aligned):** outcome vocabulary is `no_action` (same as `dismissed`), `reversal` (same as `upheld`), `partially_upheld` (requires `amount_minor`), with the canonical `dismissed` and `upheld` also accepted. A reversal posts through the same ledger command as the admin reversal, with audit. The response records the decider. A decider who is a party is refused with 403. Outcome `adjustment` is not implemented and returns 501 `not-implemented` (deviation D-02). |
| API-056 | `POST /admin/transactions/{id}/reversal` | Network administrator (**as built**; the spec wording "or reviewer (via dispute)" is not implemented, deviation D-10) | MUST | P1 | Creates kind `reversal` (DAT-022). Second reversal of the same transaction returns 409. Requires a reason. **As built:** only completed transactions qualify (disputed ones go through the dispute process); an administrator cannot reverse a transaction they are party to (403); the response is the new reversal transaction with `Location`. |
| API-057 | `POST /admin/adjustments` | Network administrator | MUST | P1 | Body: `member_id`, `direction`, `amount_minor`, `reason` (mandatory, DAT-013). Posts against system account. Refused with `insufficient-available-credit` for debits beyond limit unless policy override is requested with reason (DAT-024). |
| API-058 | `GET /admin/policy`, `PATCH /admin/policy` | Network administrator | No (PATCH MUST be idempotent by ETag) | P0 | GET returns the current policy version and ETag. PATCH with `If-Match` creates a new policy version with a required `reason`; changes take effect for new transactions only (rule in 00 and 03). Response includes `effective_at`. **As built:** the service requires a non-empty `reason` (refused with 422 when empty); PATCH is a merge patch over the flat policy object; `null` values are 422. Without an `Idempotency-Key`, the server derives the key from the ETag and the body, which makes a repeat with the same ETag idempotent. |
| API-059 | `GET /admin/members`, `POST /admin/members/{id}/approve`, `/deny`, `/suspend`, `/reinstate`, `/remove` | Network administrator | MUST on POST | P0 | Lists members by status. Each action writes an audit event. `remove` on a member with a non-zero balance returns 409 (`invalid-transaction-state` is not used; code `member-has-open-balance` is reserved for the departure rule in 03) until 03's departure rule is applied. **As built (aligned):** `deny` is implemented (FR-011), as are all six actions. `remove` returns 409 `blocked` with a `blockers` list when the member has a disputed transaction or is the last active administrator; a non-zero balance does not by itself refuse removal (deviation D-06). `If-Match` is optional on these actions (D-05). |
| API-060 | `PUT /admin/accounts/{id}/credit-limit` | Network administrator | No (ETag) | P0 | Body: `credit_limit_minor` (>= 0 string). Requires `If-Match`, `reason`, and writes an audit event with before/after. Lowering below the current debt does not change balances; it blocks new debits (03). **As built:** `credit_limit_minor` is required and may be `null` to remove the member override so the policy default applies; `expires_at` (RFC 3339, optional) sets an expiry of the override. The ETag comes from `GET /admin/accounts/{id}`. |
| API-061 | `POST /admin/invitations`, `POST /invitations/{token}/accept` | Administrator; invitee | MUST | P1 | Invitation token shown once, stored hashed, expires per policy; accepting creates membership in the state required by join mode. |
| API-062 | `GET /admin/audit-events` | Network administrator | No | P0 | Paginated, filterable by `action`, `target_id`, `actor_id`, date range. Access is itself audited (DAT-034). Returns only fields in 06 section 7.1. **As built:** filters `action`, `target_id`, `actor_id`, `from`, `to` (RFC 3339); the cursor is the audit `seq`. |
| API-063 | `POST /admin/consistency-checks`, `GET /admin/consistency-checks/{id}` | Network administrator | MUST on POST | P0 | Runs ARC-015 checks and returns a result with a summary; no amounts for named members in the summary beyond what 06 section 7.3 allows. **As built:** the result (`ok`, `findings`, `ledger_seq`, `audit_events`) is kept in process memory for 24 hours, at most 200 results, scoped to the network (any administrator of that network can read it) (D-11). The result id is the consistency run id when one is recorded. |
| API-064 | `POST /admin/announcements`, `GET /admin/announcements` (and member read via `/network/announcements`) | Administrator (write); member (read) | MUST on POST | P1 | Announcements are append-only and carry `effective_at`. Policy changes SHOULD be linked to an announcement (10). **As built:** implemented. `POST /admin/announcements` (Idempotency-Key required; title 1 to 120, body 1 to 4000, optional `effective_at` not in the past and at most a year ahead), `GET /admin/announcements` (live ones, scheduled included), `DELETE /admin/announcements/{id}` (retracts; the row is kept for the audit trail) and `GET /network/announcements` (members; only those in force). Create and delete are audited. There is no edit and no supersede link (deviation D-12 narrowed). |
| API-065 | `POST /exports`, `GET /exports/{id}`, `GET /exports/{id}/artefact` | Member (own data) or administrator (network) | SHOULD on POST | P1 | Creates a job; status `queued`, `running`, `complete`, `failed`. Artefact download is available only when complete and checksum-verified (DAT-040); expires per 06 section 9. **As built (D-13):** export runs synchronously; the 201 response already has status `complete` with `sha256`, `bytes` and `artefact_url`. No `queued`, `running` or `failed` state is exposed. The artefact is a ZIP of CSV or JSON files (`application/zip`), kept in memory for 24 hours. Scope `mine` is for every member; the scopes `ledger`, `members`, `audit` and `full` are for administrators only. Limits: 5 live exports per person (429 `rate-limited`, `Retry-After` 3600), 50 live exports and 64 MiB in total (503 `unavailable`, `Retry-After` 60). The artefact is served only when its bytes match the recorded SHA-256 (DAT-040). Export ids are UUIDv4, not UUIDv7 (D-14). |
| API-066 | `GET /tokens`, `POST /tokens`, `DELETE /tokens/{id}` | Authenticated person (session auth only for create) | SHOULD on POST | P1 | Create requires session authentication (API-011). Token value appears once in the 201 response. Delete revokes immediately; revoked tokens fail with 401. **As built:** create with a bearer token returns 403 `forbidden`; scope `admin` returns 501 (D-07); `expires_at` defaults and is capped at 365 days; the list is not paginated and returns at most 100 rows; `DELETE` needs write scope and returns 204. |
| API-067a | `POST /me/password` | Signed-in person, session cookie only | MAY | P0 | Body `current_password`, `new_password`. `204` on success; every session of the person is revoked, API tokens are not. A wrong current password is `422 validation-failed` at `/current_password` (not a `401`), limited to 5 attempts per minute per person. Implements AUTH-03 / SEC-003. **As built:** a bearer token gets 403 `forbidden`; the 429 `rate-limited` answer comes when the 5-per-minute cap is exceeded; a new-password policy failure is reported at `/new_password`. Path `/me/password` is registered as `POST /api/v1/me/password`. |
| API-067 | `POST /me/membership/departure` | Member | MUST | P1 | Starts departure per 03 and 10; response states the settlement status. Does not delete ledger entries (LED principles, 06). **As built:** `settlement` is `settled` when the balance is zero and `retained` otherwise; departure is refused with 409 `blocked` (blockers) for a disputed transaction or for the last active administrator (D-06). Idempotency-Key is required. |

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
| API-077 · P1 | The API MUST publish a machine-readable description (OpenAPI 3.1 recommended) per release, and contract tests MUST validate responses against it. **As built:** the description is published as [docs/api/openapi.yaml](../api/openapi.yaml) (OpenAPI 3.1, parses, all references resolve, route set equals `internal/api/routes.go`). No contract test yet validates live responses against it (gap G-2). Not yet released as a versioned artefact. |
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
- **OQ-API-03 (decision needed):** Default token expiry and rate-limit numbers (API-010, API-038). **As built:** token expiry 365 days; rate limits 600 per minute per principal and 1,200 per minute per IP (see D-08). Decision still open.
- **OQ-API-04 (decision needed):** Replay policy for failed commands (API-025): recommended default is to replay failures for the retention period so retries are deterministic, but to allow a fresh attempt if the key was not stored. **As built:** failures are not replayed; the key stays reusable. Decision still open; the as-built choice is documented in API-025.
- **OQ-API-05 (assumption):** A proposal is a single `proposed` transaction; there is no counter-proposal chain in MVP. A counter-proposal is a new proposal.
- **OQ-API-06 (decision needed):** Whether `GET /network` is visible to unauthenticated visitors (join page) or members only. Recommended default: a minimal public subset, with join mode visible. **As built:** no route is public; `GET /network` requires authentication (members-only). Decision still open.

## 12. As-built alignment (2026-10-21)

Source of truth for this section: `internal/api` on branch `claude/circula-build` (`routes.go`, `dto.go`, `respond.go`, `auth.go`, `idem.go`, `cursor.go`, `jobs.go`, `middleware.go`, `h_*.go`). The machine-readable description is [docs/api/openapi.yaml](../api/openapi.yaml). Requirement IDs above are unchanged. Items marked "aligned" changed wording to match the code; items marked D-nn or G-nn are open deviations or gaps to decide or fix. Nothing here changes the ledger rules in 03 or 06.

### 12.1 Route coverage

- Routes in the spec and in code: every operation in section 8 is registered. Announcement routes (API-064) are implemented. The OpenAPI file has 56 operations on 47 paths.
- Routes in code and not in section 8 (extensions; no new requirement IDs):

| Route | As built |
|---|---|
| `POST /api/v1/disputes/{id}/review` | Open to under_review; records the decider. Administrator or non-party reviewer. |
| `POST /api/v1/disputes/{id}/withdraw` | Opener withdraws. A non-opener gets 404. |
| `POST /api/v1/disputes/{id}/retry-posting` | Retries a posting_pending dispute (DOM-021). |
| `PUT` and `DELETE /api/v1/admin/members/{id}/roles/{role}` | Grant or revoke `admin` or `reviewer`. Other role values are 404. No If-Match. |
| `GET /api/v1/admin/accounts/{id}` | Administrator balance view; source of the credit-limit ETag. Audited (account.viewed). |
| `GET /api/v1/listings/{id}` | One listing with ETag; `If-None-Match` gets 304. Members see active listings of active owners and all their own. |
| `PATCH /api/v1/me` | Merge patch `{"time_zone": "<IANA name>" or ""}`: the person's display zone for web pages. `GET /me` returns `time_zone` (string or null). The API stays UTC. Audited. |
| `DELETE /api/v1/admin/announcements/{id}` | Retracts an announcement (API-064). |

- Routes in section 8 and not in code: none.

### 12.2 Aligned in this revision

- **API-049 (admin read).** Administrators may read any transaction in their own network. Reviewers may read a transaction with a dispute on it. Others get 404.
- **API-018 and API-055 (decider).** Dispute decisions accept the administrator or the reviewer role, not a party (DOM-037, DOM-038). Withdrawal is by the opener only.
- **API-055 (outcome vocabulary).** `no_action` and `reversal` are accepted as aliases of `dismissed` and `upheld`. `partially_upheld` takes `amount_minor`.
- **API-059 (`deny`).** Implemented as `POST /admin/members/{id}/deny` (FR-011). All six member actions exist.
- **API-067a (`POST /me/password`).** Implemented as specified: 204 on success, 422 at `/current_password`, cookie session only, 5 attempts per minute per person.
- **API-021 (`invalid-header`).** Malformed `Idempotency-Key` returns 400 `invalid-header`.
- **API-010 (501).** The `admin` token scope returns 501 `not-implemented`. Announcements (API-064) are implemented.
- **API-077.** OpenAPI 3.1 description published (see G-2 for validation).
- **Error catalogue (7.2).** Added as-built codes `method-not-allowed`, `not-implemented`, `network-required`, `blocked`, `conflict`. `member-has-open-balance` is reserved and not emitted.

### 12.3 Deviations (open)

| ID | Requirement | As built | Where | Proposed action |
|---|---|---|---|---|
| D-01 | API-041 | `GET /me` lists memberships across all networks in the credential's scope; the route does not resolve a network. | `internal/api/h_core.go` `getMe` | Decide: keep (person view) and amend API-041, or filter to the current network. |
| D-02 | API-055 | Outcome `adjustment` returns 501. | `h_disputes.go` `resolveDispute` | Implement through the adjustment command, or keep 501 and mark API-055 as partial. |
| D-03 | API-021 | Closed: edge and service both accept 1 to 255 characters (`service/idempotency.go` `maxKeyLen` = 255). | `idem.go` `validKey`; `service/idempotency.go` | None. |
| D-04 | API-022, API-020 | Replay cache is in process (24 h, 10,000 records) and lost on restart. Service idempotency (7 days) backs the commands that have it. | `idem.go` | Persist the replay record in the store, or state the restart behaviour in 06. |
| D-05 | API-026, API-027 | `If-Match` is optional on member actions and not used on role changes; `GET /members`, `GET /members/{id}` and `GET /network` return no ETag. | `h_admin.go` `memberAction`, `roleAction` | Require `If-Match` on member actions, or amend API-026 to list the required resources. |
| D-06 | API-059, API-067 | Removal and departure refuse with 409 `blocked` and a `blockers` array (disputed transaction, last active administrator). `member-has-open-balance` is not emitted, and a non-zero balance does not refuse. | `service/departure.go` | Decide the departure rule in 03; then use `member-has-open-balance` or amend the catalogue. |
| D-07 | API-010, API-066 | Token scope `admin` returns 501. | `h_tokens.go` `createToken` | Keep (admin commands use the caller's role) and amend API-010, or implement. |
| D-08 | API-038 | Rate limits apply to every route, per principal (600/min) and per IP (1,200/min), in process memory and not shared between instances. | `api.go` defaults; `middleware.go` `edge` | Decide limits (OQ-API-03); add shared storage for multi-instance deployments. |
| D-09 | API-046 | `POST /listings` requires `Idempotency-Key` (SHOULD in the spec). | `h_listings.go` route flags | Keep and amend to MUST, or relax. |
| D-10 | API-056 | Administrative reversal is administrator-only. Reviewers reverse only through dispute resolution. | `service/reversal.go` `requireAdmin` | Keep and amend the Role column (done as built), or add reviewer access. |
| D-11 | API-063 | Consistency results are in memory for 24 h (at most 200) and do not survive a restart. | `jobs.go` | Store results with the run in the database, or accept as ephemeral and amend. |
| D-12 | API-064 | Announcements are append-only without edit or a supersede link; delete retracts. | `h_f2.go` | Add supersede links if governance needs them (10). |
| D-13 | API-065 | Exports are synchronous; `queued`, `running` and `failed` are not exposed; artefacts expire after 24 h in memory. | `h_export.go` `createExport` | Keep for MVP and amend API-065, or add the job states. |
| D-14 | API-005 | Export ids are UUIDv4 (`uuid.NewString`). Other ids are UUIDv7. | `h_export.go`, `h_ops.go` | Use UUIDv7 for consistency. |

### 12.4 Gaps

| ID | Gap | Note |
|---|---|---|
| G-1 | Closed: `GET /listings/{id}` returns the listing with its ETag and answers `If-None-Match` with 304. | `h_f2.go` | None. |
| G-2 | No contract test validates live responses against `docs/api/openapi.yaml` (API-077) and no fixtures cover the previous release (API-078). | Add a schema check to the contract test set in 12. |
| G-3 | Error `errors[].code` is an open set in the code; section 7.1 implied a closed set. | Keep open and amend API-030 (done as built in 7.1). |

Cross-references: domain rules [03-domain-model.md](03-domain-model.md); ledger data [06-data-ledger-audit.md](06-data-ledger-audit.md); UX [08-ux-accessibility.md](08-ux-accessibility.md); security [09-security-privacy.md](09-security-privacy.md); operations [11-deployment-operations.md](11-deployment-operations.md); test plan [12-quality-test-plan.md](12-quality-test-plan.md); decisions [13-assumptions-risks-decisions.md](13-assumptions-risks-decisions.md).
