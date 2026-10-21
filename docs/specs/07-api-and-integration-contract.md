# 07 — API and Integration Contract

| Field | Value |
|---|---|
| Status | Draft |
| Owner role | API / integration architect |
| Last updated | 2026-10-21 |
| Scope | The conceptual contract between Circula clients (first-party web client, later integrations) and the Platform: design principles, authentication and authorization expectations, resource and operation catalogue for the MVP, data encoding (identifiers, timestamps, amounts), error format and error-code catalogue, idempotency, optimistic concurrency, pagination, rate limiting, versioning and deprecation, machine-readable contract, and design-for rules for webhooks. Not in scope: UI behaviour ([08](08-ux-and-accessibility.md)), storage ([06](06-data-ledger-and-audit.md)), the full permission matrix ([02](02-personas-roles-and-journeys.md)), state-machine rules ([15](15-lifecycle-state-machines.md)). |
| Depends on | [00](00-conventions.md), [03](03-domain-model-and-mutual-credit-rules.md), [04](04-functional-requirements.md), [05](05-architecture-and-trust-boundaries.md), [06](06-data-ledger-and-audit.md), [09](09-security-privacy-and-abuse.md), [15](15-lifecycle-state-machines.md), [16](16-network-policy-catalog.md) |
| Open questions | Section 17 (`API-OD-1` .. `API-OD-9`), registered in [13](13-assumptions-risks-decisions-roadmap.md) |

## 1. Purpose and scope

This document defines **what a Circula API must do and how it must behave**, not which framework or protocol implements it. It gives implementation teams (a) the operation catalogue they must expose for the MVP, (b) the cross-cutting rules every operation follows (errors, idempotency, concurrency, pagination, versioning), and (c) the binding rule that **the API enforces the same ledger invariants as the user interface** (section 7), because the first-party client is just another API consumer.

Technology neutrality: the catalogue gives a *suggested HTTP binding* (method and path) because HTTP/JSON is the recommended transport (`API-002`), but the requirements are stated on observable behaviour. A different binding (for example RPC) is conformant if it preserves the semantics here. Nothing in this document implies the credit unit is money, redeemable, a security or a crypto-asset ([INV-014](03-domain-model-and-mutual-credit-rules.md)); nothing in the API exposes redemption, cash-out, interest or fiat conversion. Legal and regulatory questions about exposing financial-like APIs (for example third-party integrations touching balances) are questions for counsel; see [19](19-legal-and-regulatory-open-questions.md).

Out of scope here: a Platform Operator operations plane (infrastructure administration, tenant provisioning). It is a separate surface governed by [11](11-deployment-operations-and-support.md) and [05](05-architecture-and-trust-boundaries.md); it MUST NOT expose any operation in section 6 that posts to a ledger (INV-012; `API-017`).

## 2. Definitions used here

Domain terms (Network, Member, Member Account, Trade, Proposer, Counterparty, Provider, Recipient, Journal Entry, Posting, Adjustment, Reversal, Dispute, Audit Event, credit unit) are as in the canon ([00 section 5.1](00-conventions.md)) and [14](14-glossary.md). Roles are `member`, `network_admin`, `dispute_reviewer`, `platform_operator` ([00 section 5.1](00-conventions.md)). Lifecycle state names are exactly those of [00 section 5.3](00-conventions.md) and [15](15-lifecycle-state-machines.md). Policy parameter names are those of [00 section 5.6](00-conventions.md).

Terms local to this document:

| Term | Meaning |
|---|---|
| **Principal** | The authenticated caller: a logged-in person (session) or, post-MVP, an integration token acting for a person or for a Network. |
| **Acting Membership** | The Membership in the addressed Network under which a request is authorised. Derived server-side from the Principal and the `network_id` in the request path; never taken from a client-supplied role claim. |
| **Amount** | The JSON object defined in `API-030`: a string of integer minor units plus the Network's `unit_scale`. |
| **Problem** | An error response body per RFC 9457 with the extension members of section 8. |
| **Error code** | The stable machine-readable `code` member of a Problem (catalogue in section 8.3). |
| **Mutating operation** | Any operation other than a read (`GET`/`HEAD`-class). |
| **Posting operation** | A mutating operation that can create a Journal Entry: `accept_trade`, `create_adjustment`/`approve_adjustment`, `accept_reversal`/`approve_reversal`, and the departure settlement performed by the Platform ([15](15-lifecycle-state-machines.md)). |
| **Idempotency scope** | The tuple (`network_id`, Principal, `Idempotency-Key`) within which a key is unique (`API-081`). |

## 3. API design principles

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| API-001 | P0 | The API MUST be the only path by which any client, including the first-party web client, reads or changes Network data. The first-party client MUST NOT use privileged or undocumented operations. | One enforcement point for invariants and authorization; integrations can reach parity later. | Inspection: client network-trace review; every call appears in the OpenAPI document (`API-130`) |
| API-002 | P0 | The MVP API SHOULD be a resource-oriented HTTP API exchanging UTF-8 JSON (`application/json`; errors `application/problem+json`). Any other binding MUST preserve every semantic in this document. | Broad tooling support; matches the technology-neutral posture (D-02). | Inspection of the contract document |
| API-003 | P0 | Every resource of a Network MUST be addressed under that Network's identifier (suggested `/v1/networks/{network_id}/...`). A request MUST NOT read or write more than one Network (INV-011). | Tenant isolation is visible in every URL and every authorization check. | Test: request combining identifiers of two Networks returns `not_found` and changes nothing |
| API-004 | P0 | Operations that change state MUST be modelled as explicit commands on a resource (for example "accept" a Trade), not as generic field writes to a state attribute. Clients MUST NOT be able to set `state`, `state_version`, balances, `recorded_at`, or ledger identifiers directly. | Lifecycle transitions are only legal through the tabulated events ([15](15-lifecycle-state-machines.md), LIF-001). | Test: attempts to write `state` or `balance` via any representation are rejected `validation_failed` |
| API-005 | P0 | Ledger resources (Journal Entries, Postings) MUST be read-only through the API: no operation of any kind may update or delete them (INV-003). Corrections MUST be requested through Adjustment or Reversal operations only (INV-004). | Append-only ledger. | Contract inspection: no write method on ledger resources; fuzz test of verbs on ledger paths returns method/operation not allowed |
| API-006 | P0 | Every operation MUST evaluate in the order authenticate, authorize, check legality in current state, evaluate guards, execute (LIF-004 in [15](15-lifecycle-state-machines.md)). An unauthorised Principal MUST receive the same error whether or not the target exists or is in a legal state. | Prevents state and existence leakage; predictable precedence. | Test matrix: unauthorised actor vs legal and illegal states yields identical Problem (code, status, body shape) |
| API-007 | P0 | Request bodies MUST be validated against the contract schema. Unknown fields in a request MUST be rejected with `validation_failed` (listing the field pointers). Responses MAY gain fields in minor versions; clients MUST ignore unknown response fields (`API-121`). | Strict input catches typos in money-adjacent calls; tolerant output allows evolution. | Schema-conformance tests; negative test with an extra field |
| API-008 | P0 | The API MUST apply the least data principle: responses MUST include only fields the Acting Membership is allowed to see under `balance_visibility`, `directory_visibility` and the role matrix ([02](02-personas-roles-and-journeys.md), [09](09-security-privacy-and-abuse.md)); omitted fields MUST be absent, not `null`, when absence is itself the privacy mechanism. | Field-level privacy; avoids inference through nulls. | Test per visibility value and role |
| API-009 | P1 | Reads of one resource SHOULD be cheap and side-effect free; any `GET` MUST NOT create, change or expire state (a Trade whose `expires_at` has passed is reported with its guard consequences per LIF-007, not mutated by the read). | Safe caching, crawlers and retries. | Test: GET storm leaves ledger and audit data unchanged |
| API-010 | P0 | After a posting operation returns success, every subsequent read of balances and history by either party MUST reflect it (read-your-writes; no stale cached balance served for that Network). | INV-008: balances equal the sum of Postings; users must not see a successful trade "not happen". | Test: accept then balance read in a loop under concurrent load; never returns pre-trade value |

## 4. Authentication and authorization expectations

Detailed controls (credential policy, recovery, session lifetimes, step-up thresholds) belong to [09](09-security-privacy-and-abuse.md); this section fixes only what the contract exposes.

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| API-011 | P0 | The first-party client MUST authenticate with a server-side session established by an authenticated login ([09](09-security-privacy-and-abuse.md), D-07). The session credential MUST be an opaque high-entropy value carried in a cookie that is `Secure`, `HttpOnly` and `SameSite` and not readable by page scripts. | Avoids long-lived bearer tokens in script-readable storage. | Inspection of cookie attributes; test that script context cannot read it |
| API-012 | P0 | Every mutating operation authenticated by a cookie session MUST also present a CSRF defence (an anti-forgery token or equivalent header check plus `Origin` validation); failure MUST return `csrf_failed` (403) and change nothing. | Cookie sessions are CSRF-prone. | Test: cross-origin form post is rejected |
| API-013 | P0 | A request without valid authentication MUST return `unauthenticated` (401); an expired session MUST return `session_expired` (401). Neither response may reveal whether the addressed object exists. | Uniform failure. | Test |
| API-014 | P0 | Authorization MUST be decided on the server from (Principal, Acting Membership, object, operation). Role claims, Membership identifiers or "acting as" hints in request bodies, headers or tokens MUST NOT be trusted. | OWASP API1/API5 class risks (broken object and function level authorization). | Authorization test matrix: every operation x every role x own/other object ([12](12-quality-strategy-and-test-plan.md)) |
| API-015 | P0 | Object-level authorization MUST be enforced on every read and write: a `member` MUST be able to read only Trades, Disputes, Adjustments, Reversals and ledger history where their Member Account is a party, plus data the Network's visibility policies make visible to members. | Prevents enumeration of other members' activity. | Test: member requests another member's trade by id receives `not_found` |
| API-016 | P0 | Operations whose acting role is "Counterparty only" or "Proposer only" (accept, decline, cancel of a Trade) MUST reject every other caller, including `network_admin`, `dispute_reviewer` and `platform_operator`, with `forbidden` (403) or `not_found` per API-006. There MUST be no query parameter, header or token scope that bypasses this. | INV-012; D-06; LIF-013. | Authorization matrix test including all four roles |
| API-017 | P0 | No credential, token scope or role MUST be able to cause a Trade to be proposed or accepted on behalf of a Member, nor to post to a ledger other than through the Adjustment and Reversal operations in section 6. The `platform_operator` role MUST have no ledger-posting operation and no default read access to Network content. | INV-012; canon role definition. | Inspection of every operation's actor list; test as `platform_operator` |
| API-018 | P1 | Sensitive operations (`create_policy_version`, `create_adjustment`, `approve_adjustment`, `approve_reversal`, role changes, `request_export` of Network-wide data, `begin_membership_close`) SHOULD require recent re-authentication; if the session is too old the API MUST return `reauthentication_required` (403) with a Problem field `reauth_methods`, and MUST NOT perform any part of the operation. Thresholds [DECISION NEEDED: `API-OD-1` step-up recency window]. | Limits damage from a hijacked session; admin misuse resistance ([09](09-security-privacy-and-abuse.md)). | Test with an aged session |
| API-019 | P2 | The API MUST be designed so scoped integration tokens can be added without breaking v1: authorization MUST be expressed as (Principal, Acting Membership, scope) with scopes named per operation group (for example `trades:read`, `trades:propose`, `balances:read`, `listings:write`, `exports:read`). A token MUST never exceed the grants of the person who issued it, and a token MUST NOT be able to perform `accept_trade`, `decline_trade`, `create_adjustment`, `approve_*` or policy changes unless a later Baseline decision explicitly allows it. | Integrations (post-MVP) must not become a bypass for member consent. | Design inspection now; scope-matrix test when delivered |
| API-020 | P2 | Third-party and mobile access SHOULD use the OAuth 2.0 authorization code flow with PKCE following the OAuth security best current practice (RFC 9700), with short-lived access tokens and narrow scopes, delivered with the Idempotency, error and rate-limit rules of this document unchanged. Documents MUST NOT claim conformance to OAuth 2.1, which is still an IETF Internet-Draft as of 2026-10-21 (see [research note](../notes/research-technical-standards.md)). | Avoids bespoke token flows and over-claiming a draft. | Design inspection |
| API-021 | P0 | The API MUST set no permissive cross-origin policy: allowed origins MUST be an explicit per-deployment allowlist; wildcard origins with credentials MUST NOT be used. | Standard browser-side abuse resistance. | Config inspection; test from a non-allowlisted origin |

Role summary for the catalogue below: **M** = `member` (acting as Proposer, Counterparty or other party as stated), **NA** = `network_admin`, **DR** = `dispute_reviewer`, **PO** = `platform_operator`. The authoritative permission matrix is in [02](02-personas-roles-and-journeys.md) (PER-prefixed requirements); where this document and 02 differ on who may do something, 02 wins and this table MUST be corrected.

## 5. Data conventions

### 5.1 Identifiers, timestamps, encoding

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| API-022 | P0 | All identifiers MUST be opaque, non-guessable, time-sortable strings in canonical lowercase hyphenated form (UUIDv7 recommended, D-03). Clients MUST NOT parse identifiers or rely on their ordering; ordering is defined only by the documented sort of each collection. | D-03; ID leaks creation time but no semantics. | Schema check; negative test of malformed ID returns `validation_failed` |
| API-023 | P0 | All timestamps MUST be RFC 3339 `date-time` strings in UTC with a `Z` suffix and millisecond precision (for example `2026-10-21T14:03:22.418Z`). Date-only values MUST use `YYYY-MM-DD`. The API MUST NOT accept local-time inputs without an explicit offset. | Canon writing rule: times in UTC in data. | Schema validation; negative tests |
| API-024 | P0 | Deadlines and windows (`expires_at`, dispute window end) MUST be returned as absolute UTC timestamps computed by the server; clients MUST NOT recompute them from policy values. Evaluation uses the server clock at commit (LIF-007). | Policy values may change; server is the authority. | Test: policy change does not move returned `expires_at` of existing proposals |
| API-025 | P0 | Every ledger-visible resource MUST carry `recorded_at` (server time the record committed). Journal Entries of type `trade` MUST expose `trade_id`; Reversal and Adjustment entries MUST expose the id of the entry or request they correct or implement (INV-004). | Traceability. | Schema inspection; reconciliation test |

### 5.2 Amount encoding (decision)

**Decision (recommended default, `API-OD-2`):** Amounts are transported as **a JSON string holding a base-10 integer count of minor units**, accompanied by the Network's `unit_scale`, never as JSON numbers and never as decimal strings. Rationale: JSON numbers are exchanged as IEEE-754 doubles by many parsers (JavaScript `Number` is exact only to 2^53 - 1), so large integers can silently lose precision; decimal strings invite rounding and locale bugs; an integer string with an explicit scale is exact, unambiguous and aligns with the canon ([00 section 5.2](00-conventions.md)). Display formatting (decimal separator, symbol position) is a client concern using `unit_scale`, `unit_symbol`, and the viewer's locale ([08](08-ux-and-accessibility.md)).

```json
{ "minor_units": "-9500", "unit_scale": 2 }
```

Means -95.00 of the Network's credit unit. The unit symbol and name are returned once on the Network resource (`unit_name`, `unit_symbol`, `unit_scale`) and in a `unit` block on balance and history responses, not repeated per amount.

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| API-030 | P0 | Every monetary-like field in requests and responses MUST be an Amount object `{ "minor_units": "<integer string>", "unit_scale": <0..4> }`. `minor_units` MUST match `^-?(0\|[1-9][0-9]*)$`; requests MUST NOT contain floats, exponents, separators, leading `+`, or leading zeros. | Exact arithmetic; no precision loss in JS clients. | Schema + fuzz tests of malformed values (each rejected `validation_failed`) |
| API-031 | P0 | On input, `unit_scale` MUST equal the Network's `unit_scale`; otherwise the request MUST fail `unit_scale_mismatch` (422) and no rounding or conversion may occur. The API MUST NOT silently round any value. | Prevents a client rendering 12.5 and posting 1250 vs 125. INV-013 fixes the scale for life. | Test with scale 0,2,4 Networks and mismatched requests |
| API-032 | P0 | Trade amounts in requests MUST be strictly positive (at least 1 minor unit) and within `min_trade_amount` and `max_trade_amount`; out-of-bounds MUST fail `amount_out_of_bounds` (422) with the bounds in the Problem. Balances, credit limits and Posting amounts are signed per canon: balances may be negative, `credit_limit` is >= 0 and returned as a positive magnitude. | Canon 5.2. | Boundary-value tests (0, 1, min, max, max + 1, 2^53, 2^63 - 1, 2^63) |
| API-033 | P0 | Server arithmetic on Amounts MUST use checked integer arithmetic of at least 64 bits; a value that would overflow MUST fail `amount_out_of_bounds`, never wrap or saturate. Implementations MUST accept up to at least 2^63 - 1 in `minor_units` without precision loss or reject above a documented smaller maximum. | Overflow safety. | Property tests including values above 2^53 |
| API-034 | P0 | The API MUST NOT expose any field, parameter or operation that expresses a fiat value, exchange rate, interest, redemption, cash-out or crypto-asset semantic for the credit unit (INV-014). | Canon prohibition. | Contract inspection (keyword and structure review) in [12](12-quality-strategy-and-test-plan.md) release gate |

### 5.3 Localisation of messages

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| API-035 | P0 | Clients MUST be able to rely on `code` (stable) and structured parameters, not on message text. `title` MUST be a stable English summary per error code. `detail` MAY be localised according to `Accept-Language`; the response MUST then carry `Content-Language`. The MVP ships English only (D-13); the API MUST nonetheless select a message template by (code, locale) so adding languages needs no contract change. | D-13; clients localise from `code` and `params`. | Test: unknown locale falls back to English with `Content-Language: en`; changing `Accept-Language` changes no `code` |
| API-036 | P0 | Every Problem that carries variable values MUST expose them as typed members (`params` object or the code-specific members of section 8.3), so clients can build their own localised text. Message text MUST NOT be the only carrier of an amount, id or date. | i18n-ready contract. | Schema inspection |
| API-037 | P1 | Localised `detail` text MUST NOT include personal data of other Members beyond what the caller may already see (API-008), and MUST NOT include secrets, internal paths, stack traces or SQL. | Safe error content. | Test: forced failures; scan bodies for forbidden content |

## 6. Operation catalogue (MVP)

Conventions for the tables: **Idem** = idempotency key requirement per section 9: **R** required (request without the key fails `missing_idempotency_key`), **O** optional but honoured, **-** not applicable (safe read or naturally idempotent PUT-like replacement guarded by `If-Match`). **CC** = conditional write guarded by `If-Match` (section 10). Key errors list the most specific stable codes; every operation can additionally return the generic errors of section 8.3 (`unauthenticated`, `forbidden`, `not_found`, `validation_failed`, `rate_limited`, `internal_error`, and so on). Functional requirement sources are in [04](04-functional-requirements.md); state machines in [15](15-lifecycle-state-machines.md).

`API-040` (P0): The MVP API MUST provide every operation in the tables of sections 6.1 to 6.12 with the stated actors, preconditions and idempotency behaviour, unless marked P1 or P2. Verification: contract inspection (every row has an OpenAPI operation with the same `operationId`) and one automated authorization test per row.

`API-041` (P0): An operation marked **R** MUST be rejected with `missing_idempotency_key` (400) when the key is absent, before any other business evaluation except authentication and authorization. Verification: test per row.

### 6.1 Session and identity

| Operation (`operationId`) | Binding (suggested) | Actors | Preconditions | Idem | Key errors |
|---|---|---|---|---|---|
| `get_me` | `GET /v1/me` | any authenticated | - | - | `unauthenticated` |
| `list_my_memberships` | `GET /v1/me/memberships` | any authenticated | - | - | `unauthenticated` |
| `end_session` | `DELETE /v1/session` | any authenticated | - | O | `unauthenticated` |

Login, recovery and credential management are specified in [09](09-security-privacy-and-abuse.md) and are not repeated.

### 6.2 Network and policy

| Operation | Binding | Actors | Preconditions | Idem | Key errors |
|---|---|---|---|---|---|
| `get_network` | `GET /v1/networks/{network_id}` | M, NA, DR (Memberships of that Network) | Acting Membership exists | - | `not_found` |
| `update_network_profile` | `PATCH /v1/networks/{network_id}` | NA | Fields limited to name, description, rules text; `unit_name`/`unit_symbol`/`unit_scale` rejected once any Journal Entry exists (INV-013) | CC | `stale_version`, `unit_immutable`, `forbidden` |
| `open_network` | `POST /v1/networks/{network_id}/open` | NA | Network in setup; all required parameters set; description and rules published; at least one `network_admin` ([04](04-functional-requirements.md) setup rules) | R | `setup_incomplete` (lists missing items), `illegal_transition` |
| `get_policy` | `GET /v1/networks/{network_id}/policy` | M, NA, DR | Visible parameters per role | - | `not_found` |
| `list_policy_versions` | `GET /v1/networks/{network_id}/policy/versions` | M, NA, DR | - | - | - |
| `create_policy_version` | `POST /v1/networks/{network_id}/policy/versions` | NA (quorum `admin_quorum`) | Valid parameter names and ranges ([16](16-network-policy-catalog.md)); `effective_at` at least `policy_notice_days` ahead unless flagged safety fix (D-11); reason present; `unit_scale` never changes (INV-013) | R | `policy_notice_too_short`, `policy_parameter_invalid`, `unit_immutable`, `reauthentication_required`, `approval_pending` |
| `approve_policy_version` | `POST .../policy/versions/{version_id}/approve` | NA (distinct from initiator) | Version `pending`; approvers distinct | R | `approver_not_eligible`, `illegal_transition` |

Every policy change MUST produce an Audit Event (INV-010) and notice to Members (D-11); the response includes the `audit_event_id`.

### 6.3 Membership

| Operation | Binding | Actors | Preconditions | Idem | Key errors |
|---|---|---|---|---|---|
| `create_invitation` | `POST .../invitations` | NA (or M if the Network allows) | Network open; `membership_mode` allows | R | `network_not_open`, `forbidden` |
| `apply_to_join` | `POST .../memberships` | authenticated person | Valid invitation, or `membership_mode` = `approval`/`open`; no existing Membership | R | `invitation_invalid`, `membership_exists`, `network_not_open` |
| `list_members` | `GET .../members` | M, NA, DR | `directory_visibility` / role | - | - |
| `get_member` | `GET .../members/{membership_id}` | M, NA, DR | Visible per policy | - | `not_found` |
| `decide_application` | `POST .../memberships/{membership_id}/decision` | NA | Membership `applicant`; reason on reject | R | `illegal_transition`, `reason_required` |
| `suspend_membership` / `reinstate_membership` | `POST .../memberships/{membership_id}/suspend` / `.../reinstate` | NA (quorum where policy requires) | Membership `active` / `suspended`; reason; not leaving the Network without an active `network_admin` | R | `illegal_transition`, `last_admin_required`, `reason_required` |
| `begin_membership_close` | `POST .../memberships/{membership_id}/depart` | M (own), NA (any, with reason) | Membership `active` or `suspended`; departure rules per `departure_settlement_mode` | R | `illegal_transition`, `last_admin_required` |
| `assign_role` / `revoke_role` | `PUT`/`DELETE .../members/{membership_id}/roles/{role}` | NA (quorum) | Role in {`network_admin`, `dispute_reviewer`}; Audit Event | R | `last_admin_required`, `approver_not_eligible` |
| `update_my_profile` | `PATCH .../members/me` | M (own) | Only member-editable fields | CC | `stale_version` |

### 6.4 Listings (Offers and Requests)

| Operation | Binding | Actors | Preconditions | Idem | Key errors |
|---|---|---|---|---|---|
| `create_listing` | `POST .../listings` | M (`active`) | Kind `offer` or `request`; no price expressed as money (API-034) | R | `membership_not_active`, `validation_failed` |
| `update_listing` | `PATCH .../listings/{listing_id}` | M (owner), NA (moderation fields only) | Owner or admin; state not `removed` | CC | `stale_version`, `illegal_transition` |
| `publish_listing` / `pause_listing` / `archive_listing` | `POST .../listings/{listing_id}/publish` etc. | M (owner) | State per Listing machine ([00 5.3](00-conventions.md)) | R | `illegal_transition` |
| `remove_listing` | `POST .../listings/{listing_id}/remove` | NA | Reason; Audit Event | R | `reason_required` |
| `search_listings` | `GET .../listings` | M, NA (public listings only if `directory_visibility` = `public_listings`) | Filters in section 11 | - | `invalid_cursor` |
| `get_listing` | `GET .../listings/{listing_id}` | as above | - | - | `not_found` |

Listings have no ledger effect; no listing operation touches balances.

### 6.5 Trades

| Operation | Binding | Actors | Preconditions | Idem | Key errors |
|---|---|---|---|---|---|
| `propose_trade` | `POST .../trades` | M (`active`, a party; Provider or Recipient per `initiated_by`) | Counterparty distinct and `active`; amount valid; open-proposal quota not exceeded; **no posting, no hold** (D-05) | **R** | `counterparty_not_active`, `self_trade_not_allowed`, `amount_out_of_bounds`, `unit_scale_mismatch`, `open_proposal_quota_exceeded`, `idempotency_conflict` |
| `get_trade` | `GET .../trades/{trade_id}` | M (party), NA\*, DR\* | \*only where the role matrix grants (e.g. as Dispute evidence) | - | `not_found` |
| `list_trades` | `GET .../trades` | M (own), NA\*, DR\* | Filters in section 11 | - | `invalid_cursor` |
| `accept_trade` | `POST .../trades/{trade_id}/accept` | **Counterparty only** | `proposed`; not past `expires_at`; `If-Match` matches the version the Counterparty saw; both Memberships `active` (or `departing` where permitted); INV-005 and INV-006 checked at commit | **R**, plus `If-Match` | `trade_not_proposed`, `proposal_expired`, `stale_version`, `precondition_required`, `credit_limit_exceeded`, `max_balance_exceeded`, `counterparty_not_active`, `membership_not_active`, `idempotency_conflict`, `idempotency_in_progress` |
| `decline_trade` | `POST .../trades/{trade_id}/decline` | Counterparty only | `proposed` | R | `trade_not_proposed`, `proposal_expired` |
| `cancel_trade` | `POST .../trades/{trade_id}/cancel` | Proposer only (system cancels internally) | `proposed` | R | `trade_not_proposed` |

`accept_trade` is the only operation that posts a Trade. It performs acceptance and posting as one atomic unit (INV-002) and returns the completed Trade with its `journal_entry_id`. A guard failure leaves the Trade `proposed` and unchanged ([15](15-lifecycle-state-machines.md) section 4.2).

### 6.6 Balances and history

| Operation | Binding | Actors | Preconditions | Idem | Key errors |
|---|---|---|---|---|---|
| `get_my_balance` | `GET .../members/me/balance` | M (own) | Membership not `rejected`; own access retained while `suspended`/`departing` | - | `not_found` |
| `get_member_balance` | `GET .../members/{membership_id}/balance` | M, NA, DR | Allowed by `balance_visibility` (`self` \| `admins` \| `members`) | - | `balance_not_visible` (as `not_found` to non-admin; see API-006) |
| `list_my_ledger_entries` | `GET .../members/me/ledger-entries` | M (own) | Filters, cursor | - | `invalid_cursor` |
| `get_ledger_entry` | `GET .../ledger-entries/{entry_id}` | M (party), NA\*, DR\* | - | - | `not_found` |
| `get_network_integrity` | `GET .../integrity` | NA | Returns result of the invariant checks (INV-007, INV-008) and last reconciliation time; no member-level data | - | - |

### 6.7 Disputes

| Operation | Binding | Actors | Preconditions | Idem | Key errors |
|---|---|---|---|---|---|
| `raise_dispute` | `POST .../disputes` | M (party to the Trade, `active`) | Trade `proposed` or `completed` within `dispute_window_days`; no open Dispute by same raiser on same Trade; reason present | R | `dispute_window_closed`, `duplicate_open_dispute`, `trade_not_disputable`, `reason_required` |
| `get_dispute` / `list_disputes` | `GET .../disputes[/{dispute_id}]` | M (party), DR, NA | - | - | `not_found` |
| `add_dispute_submission` | `POST .../disputes/{dispute_id}/submissions` | M (party), DR | Dispute `open` or `under_review` | R | `illegal_transition` |
| `start_dispute_review` | `POST .../disputes/{dispute_id}/start-review` | DR (not a party) | Dispute `open` | R | `reviewer_conflict`, `illegal_transition` |
| `resolve_dispute` | `POST .../disputes/{dispute_id}/resolve` | assigned DR | Dispute `under_review`; outcome in {`upheld`, `rejected`, `settled`}; written reasoning; remedy from the closed set in [03](03-domain-model-and-mutual-credit-rules.md); `settled` needs both parties' recorded consent | R | `reviewer_conflict`, `consent_missing`, `remedy_not_available`, `illegal_transition` |
| `withdraw_dispute` | `POST .../disputes/{dispute_id}/withdraw` | raiser | Non-terminal | R | `illegal_transition` |
| `escalate_dispute` | `POST .../disputes/{dispute_id}/escalate` | assigned DR, NA | Reason code | R | `illegal_transition` |

Raising a Dispute never changes a balance and never blocks a Trade (D-12). Resolution with a ledger remedy creates a pending Reversal or Adjustment request (6.8, 6.9); the Dispute reaches its resolved state when that request posts ([15](15-lifecycle-state-machines.md) section 5.2), and the `resolve_dispute` response says so via `remedy_request_id` and `dispute_state`.

### 6.8 Adjustments

| Operation | Binding | Actors | Preconditions | Idem | Key errors |
|---|---|---|---|---|---|
| `create_adjustment` | `POST .../adjustments` | NA (not the target member) | Reason code and text; target Membership not `closed`; amount at least 1 minor unit and direction stated; no Platform Operator | **R** | `reason_required`, `membership_closed`, `approver_not_eligible`, `reauthentication_required`, `override_reason_required` |
| `approve_adjustment` / `reject_adjustment` | `POST .../adjustments/{adjustment_id}/approve` or `/reject` | NA distinct from earlier approvers and target | Adjustment `pending`; quorum rules | **R** | `approver_not_eligible`, `illegal_transition`, `membership_closed` |
| `withdraw_adjustment` | `POST .../adjustments/{adjustment_id}/withdraw` | initiating NA | `pending` | R | `illegal_transition` |
| `get_adjustment` / `list_adjustments` | `GET .../adjustments[/{adjustment_id}]` | NA; affected M (visible with reason) | - | - | `not_found` |

An Adjustment is visible to the affected Member with its reason and cannot be hidden (INV-010, INV-012). When `admin_quorum` is 1 the create call posts atomically and the response status is `posted`.

### 6.9 Reversals

| Operation | Binding | Actors | Preconditions | Idem | Key errors |
|---|---|---|---|---|---|
| `request_reversal` | `POST .../trades/{trade_id}/reversals` | M (party, joint path, within `dispute_window_days`); NA (as Dispute remedy path) | Trade `completed`; no other pending/posted Reversal; neither Membership `closed` | **R** | `trade_not_completed`, `reversal_already_exists`, `dispute_window_closed`, `membership_closed` |
| `accept_reversal` / `decline_reversal` | `POST .../reversals/{reversal_id}/accept` or `/decline` | the other party (joint path) | Reversal `pending`; actor is not the requester | **R** | `approver_not_eligible`, `illegal_transition` |
| `approve_reversal` / `reject_reversal` | `POST .../reversals/{reversal_id}/approve` or `/reject` | NA (distinct, not party to the Trade; admin path) | `pending`; quorum | **R** | `approver_not_eligible`, `illegal_transition` |
| `withdraw_reversal` | `POST .../reversals/{reversal_id}/withdraw` | requester | `pending` | R | `illegal_transition` |
| `get_reversal` / `list_reversals` | `GET .../reversals[/{reversal_id}]` | parties, NA, DR | - | - | `not_found` |

A posted Reversal creates a new Journal Entry linked to the original; the original is not edited (INV-003, INV-004). Reversals are not subject to credit-limit rejection but are flagged when the outcome is out of limits ([03](03-domain-model-and-mutual-credit-rules.md)).

### 6.10 Audit

| Operation | Binding | Actors | Preconditions | Idem | Key errors |
|---|---|---|---|---|---|
| `list_audit_events` | `GET .../audit-events` | NA; M (events naming them as target, limited fields) | Filters: `actor`, `target_type`, `target_id`, `event_type`, time range | - | `invalid_cursor` |
| `get_audit_event` | `GET .../audit-events/{audit_event_id}` | as above | - | - | `not_found` |

Audit Events name actor, time, reason and target (INV-010). They are read-only; there is no write operation, because the Platform writes them as part of the operation that causes them. Actor display is pseudonymous unless the viewer is authorised to resolve identity ([09](09-security-privacy-and-abuse.md)).

### 6.11 Exports

| Operation | Binding | Actors | Preconditions | Idem | Key errors |
|---|---|---|---|---|---|
| `request_export` | `POST .../exports` | M (own data), NA (Network-scope sets allowed by [18](18-data-export-and-portability.md)) | Export kind and format valid; per-principal concurrency limit | **R** | `export_kind_invalid`, `export_limit_reached`, `reauthentication_required` |
| `get_export` | `GET .../exports/{export_id}` | requester | - | - | `not_found` |
| `download_export` | `GET .../exports/{export_id}/content` | requester | Export `ready`; link or session still valid | - | `export_not_ready` (409), `export_expired` (410) |

`request_export` is asynchronous: it returns `202 Accepted` with the export resource and its status; large content is never returned inline. Exports are bounded by the same visibility rules as live reads (API-008).

### 6.12 Notification preferences

| Operation | Binding | Actors | Preconditions | Idem | Key errors |
|---|---|---|---|---|---|
| `get_notification_preferences` | `GET .../members/me/notification-preferences` | M (own) | - | - | - |
| `update_notification_preferences` | `PUT .../members/me/notification-preferences` | M (own) | Mandatory notices (those [17](17-notifications-and-communications.md) marks non-optional, such as policy-change and adjustment notices) cannot be disabled | CC | `stale_version`, `notice_not_optional` |

## 7. Invariant parity: API enforces what the UI enforces

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| API-050 | P0 | Every ledger invariant (INV-001 to INV-014) MUST be enforced **server-side, inside the operation that could violate it**, independently of any client check. The first-party UI MUST NOT contain business rules the API lacks, and the API MUST NOT offer a path (flag, header, admin parameter, import, "dry run that commits") that skips an invariant. | Single enforcement point; integrations and UI behave identically. | Parity test: the same scenario via UI and via raw API produces identical ledger results; negative tests per invariant below |
| API-051 | P0 | The invariant behaviours below MUST hold at the API boundary. | See table. | See table |

| Invariant | Observable API behaviour | Test |
|---|---|---|
| INV-001, INV-002 | `accept_trade` creates one Journal Entry whose Postings are `-amount` (Recipient) and `+amount` (Provider) in one atomic unit; no endpoint, failure mode or timeout can yield a half-posted Trade (a failed call leaves zero Postings). | Failure injection at each step of `accept_trade`; sum of Postings = 0 |
| INV-003, INV-004 | No operation edits or deletes Journal Entries or Postings (API-005); corrections only via `request_reversal`/`create_adjustment`, each producing a new linked entry and Audit Event. | Verb fuzz on ledger paths; link-integrity query |
| INV-005 | `accept_trade` (and any other Member-account debit) is rejected with `credit_limit_exceeded` if the Recipient's resulting balance would be below `-credit_limit`, evaluated at commit on committed balances. | Boundary test at exactly the floor (allowed) and floor - 1 minor unit (rejected); 100-way concurrent accept test |
| INV-006 | Rejected with `max_balance_exceeded` if the Provider's resulting balance would exceed `max_balance` (when set). | Same, for cap |
| INV-007, INV-008 | `get_network_integrity` and balance reads agree with the sum of Postings; periodic reconciliation reports drift (target: zero). | Recompute-from-ledger comparison |
| INV-009 | Posting and state-changing operations honour `Idempotency-Key` (section 9). | Replay tests |
| INV-010 | Adjustment, Reversal, policy-change, role-change and dispute-resolution operations return the `audit_event_id` of the Audit Event written in the same atomic unit. | Every such response has an Audit Event naming actor, time, reason, target |
| INV-011 | No operation takes two `network_id`s; all referenced identifiers MUST belong to the path Network. | Cross-network id test |
| INV-012 | Propose/accept/decline/cancel only by the parties; admin value changes only via Adjustments visible to the affected Member (API-016, API-017). | Role matrix test |
| INV-013 | `update_network_profile` and policy operations reject changes to `unit_scale` and unit identity once any Journal Entry exists. | Test with a Network with one entry |
| INV-014 | No redemption, cash-out, interest, or fiat fields or operations (API-034). | Contract inspection |

## 8. Errors

### 8.1 Format

Errors use **Problem Details for HTTP APIs, RFC 9457** (July 2023, obsoletes RFC 7807; media type `application/problem+json`; standard members `type`, `title`, `status`, `detail`, `instance`, extension members permitted). Source: <https://www.rfc-editor.org/rfc/rfc9457.html> (search-result confirmation of status, date and obsoletes relation, accessed 2026-10-21; the RFC Editor host was not directly fetchable from the authoring sandbox, so re-check wording before external citation).

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| API-060 | P0 | Every non-2xx response generated by the Platform MUST be a Problem with `Content-Type: application/problem+json`, with members `type`, `title`, `status`, `detail`, `instance` and the extension members `code`, `request_id`, and (where applicable) `errors`, `params` and the code-specific members of 8.3. `status` MUST equal the HTTP status. | One parseable error shape. | Schema test across all catalogued errors; negative test for any HTML or plain-text error |
| API-061 | P0 | `type` MUST be a stable URI identifying the problem type, resolving (where the deployment publishes docs) to a human-readable description of that code (suggested `{docs_base}/problems/{code}` with `code` in kebab-case). The `type` and `code` for an error MUST NOT change within a major API version. | Documented, stable error identity. | Contract diff check (`API-123`) |
| API-062 | P0 | `code` MUST be a snake_case value from the catalogue (8.3). New codes MAY be added in minor versions; removing or repurposing a code is a breaking change. Clients MUST treat unknown codes by falling back to the HTTP status class. | Evolvability. | Catalogue test: every code returned by tests exists in the catalogue and the OpenAPI document |
| API-063 | P0 | `request_id` MUST be present on every Problem and in a response header, equal to the identifier used in server logs and traces, so a user can quote it to support ([11](11-deployment-operations-and-support.md)). | Supportability without exposing internals. | Test |
| API-064 | P0 | Validation failures MUST list every failing field in `errors[]`, each entry `{ "pointer": "<RFC 6901 JSON Pointer>", "code": "<field-level code>", "detail": "..." }` (RFC 6901 JSON Pointer), not stop at the first. | Clear validation messaging ([08](08-ux-and-accessibility.md)). | Test with multiple invalid fields |
| API-065 | P0 | Problems MUST NOT contain stack traces, internal hostnames, SQL, secrets, or another Member's balance or private data. For limit failures involving the *other* party's account, only the fact (`failing_party`: `counterparty`) MAY be disclosed, not amounts, unless the viewer is permitted to see that balance by `balance_visibility`. | Privacy and security ([09](09-security-privacy-and-abuse.md)). | Test: provider-cap failure seen by recipient reveals no provider balance when `balance_visibility` = `self` |
| API-066 | P0 | A guard failure on a state-changing operation MUST leave the target object and the ledger unchanged and MUST name the specific failing guard through `code` (LIF-014). When several guards fail, the one earliest in the table order of [15](15-lifecycle-state-machines.md) is reported, with the rest MAY listed in `errors[]`. | Deterministic, actionable errors. | Test with multiple simultaneous failures |
| API-067 | P1 | 5xx responses MUST use `internal_error` or `service_unavailable`; for retryable 503s the response SHOULD include `Retry-After`. A 5xx on a posting operation means the outcome is unknown to the client; the client MAY retry with the same `Idempotency-Key` and the server MUST resolve it per INV-009. | Safe retry. | Fault-injection test: crash after commit, before response; retry returns the original result without a second posting |

### 8.2 Status mapping principles

400 = malformed or missing input; 401 = not authenticated; 403 = authenticated but not allowed; 404 = unknown or not visible object; 409 = request conflicts with the current state of the object (lifecycle, concurrent in-flight request); 412/428 = optimistic-concurrency precondition; 422 = syntactically valid request rejected by a domain rule (including ledger limits); 429 = throttled; 5xx = Platform fault. 422 is used for domain-rule rejections because the request was understood but cannot be processed in the current business context.

### 8.3 Error-code catalogue

`API-070` (P0): The Platform MUST return exactly these codes (and statuses) for the conditions listed; additional codes MAY be added per `API-062`. Verification: one automated test per row asserting status, `code`, and unchanged state.

| Code | HTTP | Condition | Code-specific members |
|---|---|---|---|
| `malformed_request` | 400 | Body not parseable, wrong content type for body | - |
| `validation_failed` | 400 | Schema or field validation failed (including unknown fields) | `errors[]` |
| `missing_idempotency_key` | 400 | Operation marked R without key | - |
| `invalid_idempotency_key` | 400 | Key malformed or longer than the limit (`API-080`) | - |
| `invalid_cursor` | 400 | Cursor expired, tampered or from another query | - |
| `unknown_parameter` | 400 | Unsupported filter or sort parameter | `parameter` |
| `unauthenticated` | 401 | No or invalid credentials | - |
| `session_expired` | 401 | Session ended or timed out | - |
| `csrf_failed` | 403 | CSRF defence failed | - |
| `forbidden` | 403 | Authenticated but not permitted (when existence is not secret) | - |
| `reauthentication_required` | 403 | Step-up needed (`API-018`) | `reauth_methods` |
| `approver_not_eligible` | 403 | Approver is initiator, target, a party, or the `platform_operator` role (distinct-approver rules) | - |
| `reviewer_conflict` | 403 | Dispute reviewer is a party to the Dispute or Trade | - |
| `not_found` | 404 | Unknown, other-Network, or not visible object | - |
| `balance_not_visible` | 404 | Returned as `not_found` to members; see API-006. Listed so docs name the internal reason | - |
| `trade_not_proposed` | 409 | Trade operation requires `proposed`; Trade is in another state | `current_state`, `state_version` |
| `trade_not_completed` | 409 | Reversal requested for a Trade not `completed` | `current_state` |
| `illegal_transition` | 409 | Event not legal from the object's current state (generic, other machines) | `object_type`, `current_state`, `state_version` |
| `proposal_expired` | 409 | Trade still `proposed` but `now >= expires_at` | `expires_at` |
| `idempotency_in_progress` | 409 | A request with this key is still executing | `retry_after_seconds` |
| `duplicate_open_dispute` | 409 | Same raiser already has a non-terminal Dispute on the Trade | `dispute_id` |
| `reversal_already_exists` | 409 | A pending or posted Reversal exists for the Trade | `reversal_id` |
| `membership_exists` | 409 | Person already has a Membership in this Network | - |
| `network_not_open` | 409 | Network is in setup or closed | `network_state` |
| `last_admin_required` | 409 | Action would leave the Network with no active `network_admin` | - |
| `export_not_ready` | 409 | Export not yet `ready` | `status` |
| `export_expired` | 410 | Export content no longer available | - |
| `stale_version` | 412 | `If-Match` does not match the current version | `current_version` |
| `payload_too_large` | 413 | Body over limit | `max_bytes` |
| `unsupported_media_type` | 415 | Content type not accepted | - |
| `credit_limit_exceeded` | 422 | Posting would take the Recipient below `-credit_limit` (INV-005) | `failing_party`, and for the caller's own account: `balance`, `credit_limit`, `available_to_spend`, `shortfall` (Amounts) |
| `max_balance_exceeded` | 422 | Posting would take the Provider above `max_balance` (INV-006) | `failing_party`, and for the caller's own account: `balance`, `max_balance`, `headroom`, `excess` |
| `amount_out_of_bounds` | 422 | Amount < 1, outside `min_trade_amount`/`max_trade_amount`, or overflow | `min`, `max` (Amounts) |
| `unit_scale_mismatch` | 422 | Request `unit_scale` differs from the Network's | `expected_unit_scale` |
| `counterparty_not_active` | 422 | Other party's Membership not `active` (or not permitted while `departing`) | - |
| `membership_not_active` | 403 | Acting Membership not `active` for this operation (`suspended`, `departing`, `closed`, `applicant`) | `membership_state` |
| `membership_closed` | 422 | Target Membership of an Adjustment or Reversal is `closed` | - |
| `self_trade_not_allowed` | 422 | Counterparty is the caller | - |
| `open_proposal_quota_exceeded` | 422 | Proposer already has the maximum open `proposed` Trades | `limit` |
| `trade_not_disputable` | 422 | Trade state cannot be disputed (`declined`, `cancelled`, `expired`, `reversed`) | `current_state` |
| `dispute_window_closed` | 422 | Past `dispute_window_days` | `window_ended_at` |
| `reason_required` | 422 | Mandatory reason missing | - |
| `override_reason_required` | 422 | Adjustment outcome outside limits without override reason | - |
| `consent_missing` | 422 | Settlement terms lack a party's recorded consent | `missing_parties` |
| `remedy_not_available` | 422 | Remedy precondition fails (for example Reversal for a non-`completed` Trade) | `remedy` |
| `setup_incomplete` | 422 | Network not ready to open | `missing[]` |
| `policy_notice_too_short` | 422 | `effective_at` earlier than `policy_notice_days` allows (D-11) | `earliest_effective_at` |
| `policy_parameter_invalid` | 422 | Unknown parameter, or value outside catalogue range | `errors[]` |
| `unit_immutable` | 422 | Attempt to change unit identity or `unit_scale` after first posting (INV-013) | - |
| `invitation_invalid` | 422 | Invitation unknown, used, expired or for another Network | - |
| `export_kind_invalid` | 422 | Unsupported export kind or format | - |
| `export_limit_reached` | 429 | Too many concurrent exports | `retry_after_seconds` |
| `notice_not_optional` | 422 | Attempt to disable a mandatory notice | - |
| `idempotency_conflict` | 422 | Same `Idempotency-Key` previously used with a different payload (INV-009) | `original_request_id` |
| `approval_pending` | 202 (not an error) | Operation accepted but awaiting approvals under `admin_quorum`; the 2xx body carries `status: pending` | - |
| `precondition_required` | 428 | Conditional operation without `If-Match` | - |
| `rate_limited` | 429 | Request rate exceeded | `retry_after_seconds` |
| `internal_error` | 500 | Unexpected Platform fault | - |
| `service_unavailable` | 503 | Temporary unavailability or maintenance | `retry_after_seconds` |

`approval_pending` is a 2xx outcome, not an error; it is listed here only so the code vocabulary is complete and clients can distinguish "queued for quorum" from "posted".

## 9. Idempotency

The `Idempotency-Key` request header follows the semantics of the IETF HTTPAPI working group draft *The Idempotency-Key HTTP Header Field* (`draft-ietf-httpapi-idempotency-key-header`). **Status as verified 2026-10-21:** latest revision found is -07 dated 2025-10-15, shown as expired on 2026-04-18; it is **not an RFC**, and the draft itself warns against citing drafts as anything but work in progress. The Platform therefore specifies the behaviour in this document and treats the draft as informative; the semantics are also the de facto practice of widely used payment APIs. Source: <https://datatracker.ietf.org/doc/draft-ietf-httpapi-idempotency-key-header/> (via search results; the host was not directly fetchable, accessed 2026-10-21).

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server
    C->>S: POST accept (Idempotency-Key K, payload P)
    S->>S: store K+fingerprint(P); execute atomically
    S-->>C: 200 result R  (connection lost)
    C->>S: retry POST accept (K, P)
    S-->>C: 200 result R (Idempotent-Replayed: true; no second posting)
    C->>S: POST accept (K, payload P2)
    S-->>C: 422 idempotency_conflict
```

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| API-080 | P0 | Every Posting operation (`accept_trade`, `create_adjustment`, `approve_adjustment`, `accept_reversal`, `approve_reversal`) and every operation marked **R** in section 6 MUST require an `Idempotency-Key` header. The value is an opaque string of 16 to 128 visible ASCII characters; clients SHOULD generate a random UUID per logical operation. Operations marked **O** MUST honour a key when supplied. | INV-009. | Test per operation: missing key gives `missing_idempotency_key`; malformed gives `invalid_idempotency_key` |
| API-081 | P0 | A key MUST be scoped to the **Idempotency scope** (`network_id`, Principal, key). The same key from another Principal or Network MUST be treated as unrelated. Keys MUST NOT be reusable across different operations by the same Principal: reuse on a different method or path MUST fail `idempotency_conflict`. | Prevents cross-user key probing and accidental cross-operation replay. | Test with two Principals, two paths |
| API-082 | P0 | The server MUST compute a request fingerprint over the method, the path, and the canonicalised body (key order and insignificant whitespace ignored; headers other than `If-Match` excluded). The same key with an equal fingerprint MUST return the original outcome (`API-083`); the same key with a different fingerprint MUST fail `idempotency_conflict` (422) and MUST NOT execute. | INV-009: same key and payload gives the same outcome; same key with a different payload is rejected. | Replay and mutation tests, including key-order permutation of the same body |
| API-083 | P0 | A replay of a request that completed successfully MUST return the same status code and body as the first response (including the same `journal_entry_id`), plus the response header `Idempotent-Replayed: true`, and MUST NOT create a second Journal Entry, Audit Event or notification. | Safe retries. | Test: 50 retries, one Journal Entry; fault injection after commit (`API-067`) |
| API-084 | P0 | If a request with the same key is still executing, the server MUST respond `idempotency_in_progress` (409) with `Retry-After`, never run it twice. Atomicity: the key record, the state change and the Journal Entry MUST commit in the same atomic unit, so a key is never recorded as used without its effect, nor the effect without its key. | Concurrent double-submit (double click, two tabs). | Concurrency test: 20 parallel identical requests give one effect and 19 replays or 409s |
| API-085 | P0 | For rejected requests, a response that is a domain-rule rejection (4xx guard failures such as `credit_limit_exceeded`) MUST NOT consume the key: the same key MAY be retried with the same payload after the cause is fixed and MUST then be evaluated fresh. Validation failures (400) likewise. A 2xx outcome and the unchanged object state are what the key protects. [DECISION NEEDED: `API-OD-3` store-all versus store-success-only; recommended default: success-only for rejections, since a rejected accept leaves the Trade `proposed` and the Counterparty must be able to retry once funds change] | Rejections have no side effect to protect; retries after top-up are the normal path. | Test: reject, change balance, retry with same key, succeeds exactly once |
| API-086 | P0 | Retention of keys of Posting operations MUST be at least as long as the Journal Entry exists (the key is stored with the entry), so a very late retry still cannot double post; keys of other operations MUST be retained at least 24 hours [DECISION NEEDED: `API-OD-4` retention for non-posting keys; recommended 24 hours, 7 days for exports]. | A retry after weeks must not post twice. | Test with controlled clock |
| API-087 | P1 | State transitions that are naturally idempotent (decline of an already `declined` Trade by the same actor) MUST follow LIF-006 in [15](15-lifecycle-state-machines.md): the same actor repeating an event that already took effect receives the original outcome, whether or not the same key is used; a different actor or a different event follows `illegal_transition` / `trade_not_proposed`. | Consistent with the lifecycle spec. | Test |
| API-088 | P0 | Idempotency-covered operations MUST be safe against client-supplied clock or ID manipulation: the server, not the client, assigns identifiers, `recorded_at` and `expires_at`. | Prevents forged history. | Test |

## 10. Optimistic concurrency

Mutable resources that people edit or decide on carry a **strong ETag** (RFC 9110) derived from the object's version (`state_version` for lifecycle objects, per [15](15-lifecycle-state-machines.md) LIF-003, combined with `proposal_version` for Trades). Ledger entries are immutable and need no ETag.

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| API-090 | P0 | Every `GET` of a Trade, Listing, Policy version, Membership, Dispute, Adjustment, Reversal and notification-preference resource MUST return an `ETag` header and the same value in the body as `version`. | Clients need the token. | Test |
| API-091 | P0 | `accept_trade` MUST require `If-Match` carrying the ETag of the Trade the Counterparty viewed; the guard on `state_version` and `proposal_version` (TR2) is evaluated at commit. Missing header gives `precondition_required` (428); mismatch gives `stale_version` (412) with the current version, and the Trade MUST remain unchanged. | The Counterparty must accept what they saw (for example a changed amount invalidates their view). | Test: modify then accept with old ETag |
| API-092 | P0 | Conditional write operations marked **CC** in section 6 MUST require `If-Match` and MUST follow the same 428/412 behaviour. A weak ETag MUST NOT be used for `If-Match`. | RFC 9110 precondition semantics. | Test per **CC** operation |
| API-093 | P1 | `decline_trade`, `cancel_trade` and other state events SHOULD accept an optional `If-Match`; when absent they are decided by the state machine alone (LIF-003: exactly one of racing events commits, others fail `trade_not_proposed` naming the new state). | Avoids spurious 412 for cancel/decline while retaining determinism. | Race test: accept vs cancel vs decline vs expire |
| API-094 | P0 | Balance and history reads MUST return a `ledger_position` (monotonic per-account sequence or `as_of` timestamp) so clients can detect staleness; a balance read MUST NOT be served from a cache older than the last committed posting for that account (API-010). | Prevents stale-balance UX errors. | Test |

## 11. Pagination, filtering and sorting

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| API-100 | P0 | All collection reads MUST be cursor-paginated. Request: `limit` (default 50, maximum 200 [DECISION NEEDED: `API-OD-5` page size defaults]) and `cursor`. Response: `{ "data": [...], "page": { "next_cursor": "<opaque>"\|null, "has_more": bool } }`. Offset/page-number pagination MUST NOT be offered. | Offset pagination is unstable under concurrent inserts and slow at depth; ledger lists grow without bound. | Test: insertion during paging yields no duplicates or omissions of items present throughout |
| API-101 | P0 | Cursors MUST be opaque, tamper-evident (signed or encrypted) and bound to the query, Principal and Network; an invalid, expired or mismatched cursor MUST fail `invalid_cursor`. Clients MUST NOT parse cursors. | Prevents cursor-based enumeration. | Test: altered cursor; cursor reused for another filter |
| API-102 | P0 | Each collection MUST define a **total order** with a unique tie-breaker; for ledger entries and Trades the default order is `recorded_at` descending then identifier. Clients MAY request `sort=<field>` and `order=asc|desc` only on documented fields. | Deterministic paging. | Schema inspection; test |
| API-103 | P0 | Filters MUST be an allowlist per collection, documented in OpenAPI. Minimum filters: Trades (`state`, `role` (`provider`\|`recipient`), `counterparty_id`, `created_from`/`created_to`, `amount_min`/`amount_max`); ledger entries (`type`: `trade`\|`reversal`\|`adjustment`\|`departure_settlement`, `from`/`to`, `counterparty_id`); Listings (`kind`, `state`, `category`, `q` text search, `owner_id`); Members (`state`, `role`, `q`); Disputes (`state`, `trade_id`, `assigned_to_me`); Audit Events (`event_type`, `actor`, `target_type`, `target_id`, `from`/`to`). Unsupported parameters MUST fail `unknown_parameter`. | Predictable queries; typo detection. | Per-collection filter tests |
| API-104 | P1 | Collection responses SHOULD NOT include total counts by default; if `include_total=true` is offered it MAY be approximate and MUST be rate-limited. | Cost and inference-privacy (counting hidden members). | Inspection |
| API-105 | P0 | Paging and filtering MUST be applied *after* authorization: items the caller may not see MUST NOT be counted, cursored or leaked through gaps. | OWASP API1/API3 class leaks. | Test |
| API-106 | P1 | Responses MAY include RFC 8288 `Link` headers (`rel="next"`) in addition to `page.next_cursor`. | Standard navigation. | Inspection |

## 12. Rate limiting and abuse controls

Policy thresholds and abuse-case analysis are in [09](09-security-privacy-and-abuse.md); this section defines the contract.

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| API-110 | P0 | The API MUST enforce rate limits per Principal, per client network address, and per Network, with stricter limits on authentication, recovery, `propose_trade`, `create_invitation`, `request_export`, and search operations. Numeric limits MUST be configuration, documented in the deployment guide [DECISION NEEDED: `API-OD-6` default limits; recommended starting points: 600 reads/min and 60 mutations/min per Principal, 20 `propose_trade`/hour, 5 `request_export`/day]. | Resource-consumption and sensitive-business-flow abuse (OWASP API4, API6). | Load test crossing each limit |
| API-111 | P0 | A throttled request MUST return `rate_limited` (429) with `Retry-After` (seconds). Throttling MUST occur before any state change; a throttled request MUST NOT consume an idempotency key. | Safe, predictable back-off. | Test |
| API-112 | P1 | Responses SHOULD carry rate-limit state in the `RateLimit` and `RateLimit-Policy` fields of the IETF HTTPAPI draft (`draft-ietf-httpapi-ratelimit-headers`; at the time of research, revision -11 dated 2026-05-23, an Internet-Draft and not an RFC; see the [research note](../notes/research-technical-standards.md)). Clients MUST NOT depend on these headers for correctness; `Retry-After` is the contract. | Useful but unstable standard. | Inspection |
| API-113 | P0 | Per-principal open-proposal quota (`open_proposal_quota_exceeded`) is a domain rule, not a rate limit, and is configured as in [15](15-lifecycle-state-machines.md) TR1. | Spam control. | Test |
| API-114 | P0 | Request bodies MUST be size-limited (default 64 KiB for JSON bodies, `payload_too_large` otherwise), strings length-limited per field in the schema, and list sizes bounded. | Abuse resistance. | Fuzz test |
| API-115 | P1 | Authentication failures and `forbidden` outcomes MUST be counted and MAY trigger progressive delays or temporary lock-out per [09](09-security-privacy-and-abuse.md) without revealing whether an account exists. | Credential-stuffing resistance. | Test |
| API-116 | P1 | The API SHOULD log rate-limit and authorization denials as security events without request bodies or personal data ([09](09-security-privacy-and-abuse.md)). | Detection. | Log inspection |

## 13. Versioning, compatibility and deprecation

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| API-120 | P0 | The API MUST carry a major version in the path (`/v1/...`). The version identifies the contract only; it is independent of the software release number and of the data-schema migrations ([11](11-deployment-operations-and-support.md)). | Visible, cache-friendly versioning. | Inspection |
| API-121 | P0 | Within a major version only **backward-compatible** changes are permitted: new optional request fields, new response fields, new operations, new filters, new error codes in existing status classes, and new values of open (non-state) enumerations. Clients MUST ignore unknown response fields and unknown non-state enumeration values and MUST handle unknown error codes via the HTTP status class. | Allows evolution without client breakage. | Contract diff in CI (`API-123`); client conformance test with injected extra fields |
| API-122 | P0 | The following are **breaking** and require a new major version: removing or renaming an operation, field, filter or error code; changing a field's type, format, unit, sign or scale semantics; making an optional field required; tightening validation; changing a status code for a documented condition; adding a state to a lifecycle machine in [00 section 5.3](00-conventions.md) or removing/renaming one; changing idempotency, pagination or authorization semantics; changing the default sort. | Defines "breaking" objectively. | Contract diff rules |
| API-123 | P0 | CI MUST run an automated comparison of the OpenAPI document against the previous released version and fail the build on any breaking difference not accompanied by a new major version. | Prevents accidental breaks. | CI gate in [12](12-quality-strategy-and-test-plan.md) |
| API-124 | P0 | The amount encoding (`API-030`), signs and the state vocabulary MUST NOT change within a major version. | Money-adjacent safety. | Contract diff |
| API-125 | P1 | Deprecation MUST be announced at least [DECISION NEEDED: `API-OD-7` minimum notice; recommended 12 months for hosted, and N and N-1 major versions simultaneously supported for self-hosted] before removal, through: the `Deprecation` response header (RFC 9745, March 2025, an Item Structured Field date), the `Sunset` header (RFC 8594), a `Link` header with `rel="deprecation"` pointing to migration notes, a changelog entry, and notice to Network Administrators ([17](17-notifications-and-communications.md)). Sources: <https://www.rfc-editor.org/rfc/rfc9745.html> (search-confirmed, accessed 2026-10-21), <https://www.rfc-editor.org/rfc/rfc8594> (accessed 2026-10-21, background knowledge not re-fetched). | Predictable, machine-detectable lifecycle. | Test: deprecated operation emits headers |
| API-126 | P1 | Security fixes MAY change behaviour inside a major version only when the old behaviour is itself a vulnerability; such changes MUST be documented in the changelog with the reason. | Escape hatch with accountability. | Changelog review |
| API-127 | P0 | The server MUST support rolling upgrades with one release back-compat: version N+1 servers serve the same `/v1` contract as N, so a deployment mixing both never returns different shapes ([11](11-deployment-operations-and-support.md)). | Zero-downtime upgrades. | Mixed-version test |

## 14. Machine-readable contract

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| API-130 | P0 | The API MUST be described by an **OpenAPI Specification** document that is the source of truth for the HTTP binding, versioned in the repository, covering every operation in section 6 with `operationId` equal to the operation name, request/response schemas, all error codes per operation, security schemes, headers (`Idempotency-Key`, `If-Match`, `ETag`, `Retry-After`) and examples. Baseline: OpenAPI 3.1.x (full JSON Schema 2020-12 alignment, broad tooling). OpenAPI 3.2.0 was released September 2025; adopt when the chosen generator and validator support it (`API-OD-8`). Sources: <https://spec.openapis.org/oas/latest.html> (accessed 2026-10-21, per [research note](../notes/research-technical-standards.md)). | Enables codegen, docs, contract tests and diff checks. | Lint in CI; every operation present |
| API-131 | P0 | The server MUST validate requests and (in test environments) responses against the OpenAPI schemas; the first-party client SHOULD use a client generated from the same document. | Prevents drift between spec and behaviour. | Contract test suite |
| API-132 | P1 | The OpenAPI document SHOULD publish the error-code catalogue as reusable Problem schemas, and the repository SHOULD ship a `problems/` documentation page per `type` URI (`API-061`). | Developer experience. | Inspection |
| API-133 | P1 | Example requests and responses for propose trade, accept trade (success and `credit_limit_exceeded`), and balance read (section 16) MUST be included in the OpenAPI document and be exercised by contract tests. | Examples that cannot rot. | Test |

## 15. Webhooks and events (P2, designed for)

Not delivered in the MVP. The MVP design MUST not preclude the following, and the internal event record produced for notifications ([17](17-notifications-and-communications.md)) SHOULD be reusable as the source.

| ID | Pri | Requirement | Rationale | Verification |
|---|---|---|---|---|
| API-140 | P2 | The Platform SHOULD expose Network-scoped webhook subscriptions managed by `network_admin` (and per-person subscriptions only if the Network allows), delivering JSON events with envelope `{ "id", "type", "occurred_at", "network_id", "api_version", "data" }`; `data` references resources by id and by state, never includes balances of other Members. | Integration without polling. | Design inspection |
| API-141 | P2 | Event types SHOULD include at least `trade.proposed`, `trade.completed`, `trade.declined`, `trade.cancelled`, `trade.expired`, `trade.reversed`, `dispute.raised`, `dispute.resolved`, `adjustment.posted`, `membership.state_changed`, `policy.version_scheduled`. Names MUST be derived from the canon state names. | Stable vocabulary. | Design inspection |
| API-142 | P2 | Delivery MUST be at least once, with event ids unique and stable so receivers can deduplicate; ordering MUST NOT be guaranteed across events, and each event MUST carry the resource's `version` so receivers can order per object. | Matches idempotent consumption. | Test |
| API-143 | P2 | Each delivery MUST be signed with a per-subscription secret (HMAC over timestamp and body) with a timestamp tolerance for replay defence; endpoints MUST be HTTPS and outbound requests MUST be protected against server-side request forgery (no private or link-local targets, bounded time and size). | Receiver authenticity; SSRF (OWASP API7). | Design review; SSRF test |
| API-144 | P2 | Events MUST be written to an outbox in the same atomic unit as the state change they report (so there is never an event without its effect, nor an effect without its event), and retried with backoff for at least 72 hours; subscriptions that persistently fail MUST be disabled with notice. | Reliable, consistent events. | Fault-injection test |
| API-145 | P2 | Events are informational: a webhook MUST NOT be able to approve, accept, post or alter anything by its response. | Consent integrity (INV-012). | Design inspection |

## 16. Examples (non-normative)

Network unit "hours" (symbol `h`), `unit_scale` 2, so `minor_units` "800" is 8.00 h. Ids are illustrative.

### 16.1 Propose trade (Provider proposes: "I provided X, please accept a charge of 8.00 h")

```http
POST /v1/networks/0198a3c0-1111-7000-8000-000000000001/trades HTTP/1.1
Content-Type: application/json
Idempotency-Key: 6f6a8d3e-5e1a-4c0d-9a54-2b7e2f0c1a11
X-CSRF-Token: <token>

{
  "initiated_by": "provider",
  "counterparty_membership_id": "0198a3c0-2222-7000-8000-00000000000a",
  "amount": { "minor_units": "800", "unit_scale": 2 },
  "description": "Two hours of bike repair, 14 Oct",
  "listing_id": "0198a3c0-3333-7000-8000-00000000000c"
}
```

```http
HTTP/1.1 201 Created
Content-Type: application/json
ETag: "tr-0198a3c1-4444-v1"
Location: /v1/networks/0198a3c0-1111-7000-8000-000000000001/trades/0198a3c1-4444-7000-8000-00000000000d

{
  "id": "0198a3c1-4444-7000-8000-00000000000d",
  "state": "proposed",
  "version": "tr-0198a3c1-4444-v1",
  "initiated_by": "provider",
  "provider_membership_id": "0198a3c0-2222-7000-8000-00000000000b",
  "recipient_membership_id": "0198a3c0-2222-7000-8000-00000000000a",
  "amount": { "minor_units": "800", "unit_scale": 2 },
  "description": "Two hours of bike repair, 14 Oct",
  "created_at": "2026-10-21T14:03:22.418Z",
  "expires_at": "2026-11-04T14:03:22.418Z",
  "journal_entry_id": null,
  "advisories": []
}
```

`journal_entry_id` is null because a proposal has no ledger effect; `available_to_spend` is unchanged (D-05).

### 16.2 Accept trade: success (Counterparty = Recipient)

```http
POST /v1/networks/0198a3c0-1111-7000-8000-000000000001/trades/0198a3c1-4444-7000-8000-00000000000d/accept HTTP/1.1
Idempotency-Key: 0c7b1e0a-9b21-4a67-8f0e-61d2ab3f9d02
If-Match: "tr-0198a3c1-4444-v1"
```

```http
HTTP/1.1 200 OK
Content-Type: application/json
ETag: "tr-0198a3c1-4444-v2"

{
  "id": "0198a3c1-4444-7000-8000-00000000000d",
  "state": "completed",
  "version": "tr-0198a3c1-4444-v2",
  "amount": { "minor_units": "800", "unit_scale": 2 },
  "completed_at": "2026-10-21T15:40:09.032Z",
  "journal_entry_id": "0198a3d2-5555-7000-8000-0000000000e1",
  "postings": [
    { "membership_id": "0198a3c0-2222-7000-8000-00000000000a", "amount": { "minor_units": "-800", "unit_scale": 2 } },
    { "membership_id": "0198a3c0-2222-7000-8000-00000000000b", "amount": { "minor_units": "800", "unit_scale": 2 } }
  ],
  "my_balance_after": { "minor_units": "400", "unit_scale": 2 }
}
```

Postings sum to zero (INV-001). `my_balance_after` is the caller's balance only; the counterparty's balance is not disclosed here (API-008, API-065). A retry with the same `Idempotency-Key` returns this body with `Idempotent-Replayed: true`.

### 16.3 Accept trade: failure `credit_limit_exceeded`

The Recipient's balance is -95.00 h with a credit limit of 100.00 h; the Trade is for 8.00 h.

```http
HTTP/1.1 422 Unprocessable Content
Content-Type: application/problem+json
Content-Language: en

{
  "type": "https://circula.example/problems/credit-limit-exceeded",
  "title": "Credit limit would be exceeded",
  "status": 422,
  "detail": "Accepting this trade would take your balance below your credit limit. You can spend up to 5.00 h more.",
  "instance": "/v1/networks/0198a3c0-1111-7000-8000-000000000001/trades/0198a3c1-6666-7000-8000-00000000000e/accept",
  "code": "credit_limit_exceeded",
  "request_id": "req_0198a3d9c1f27b",
  "failing_party": "caller",
  "balance": { "minor_units": "-9500", "unit_scale": 2 },
  "credit_limit": { "minor_units": "10000", "unit_scale": 2 },
  "available_to_spend": { "minor_units": "500", "unit_scale": 2 },
  "amount": { "minor_units": "800", "unit_scale": 2 },
  "shortfall": { "minor_units": "300", "unit_scale": 2 },
  "trade_state": "proposed"
}
```

The Trade stays `proposed` and unchanged, no Journal Entry exists, and the idempotency key is not consumed (`API-085`). The `type` host is a placeholder; deployments publish their own base. If the failing party were the Provider's `max_balance`, the Recipient would see `"failing_party": "counterparty"` and no amounts (API-065).

### 16.4 Balance read

```http
GET /v1/networks/0198a3c0-1111-7000-8000-000000000001/members/me/balance HTTP/1.1
```

```http
HTTP/1.1 200 OK
ETag: "bal-0198a3c0-2222-7000-8000-00000000000a-2931"

{
  "membership_id": "0198a3c0-2222-7000-8000-00000000000a",
  "account_id": "0198a3c0-7777-7000-8000-0000000000a1",
  "unit": { "name": "hours", "symbol": "h", "scale": 2 },
  "balance": { "minor_units": "-9500", "unit_scale": 2 },
  "standing": "net_received",
  "credit_limit": { "minor_units": "10000", "unit_scale": 2 },
  "max_balance": null,
  "available_to_spend": { "minor_units": "500", "unit_scale": 2 },
  "pending_informational": {
    "incoming_proposals": { "count": 1, "total": { "minor_units": "800", "unit_scale": 2 } },
    "outgoing_proposals": { "count": 0, "total": { "minor_units": "0", "unit_scale": 2 } },
    "reduces_available_to_spend": false
  },
  "ledger_position": 2931,
  "as_of": "2026-10-21T15:41:00.000Z"
}
```

Fields: `balance` is the signed sum of Postings (canon 5.2); `standing` is `net_provided` (balance > 0), `net_received` (< 0) or `even` (0), a convenience so clients do not infer meaning from the sign alone ([08](08-ux-and-accessibility.md)); `available_to_spend = balance + credit_limit`, never below 0; `pending_informational` is explicit that pending proposals do not reduce it in the MVP (D-05).

## 17. Assumptions and open decisions

Registered in [13](13-assumptions-risks-decisions-roadmap.md); local labels `API-OD-n` are used until 13 assigns `DEC-` numbers.

| Label | Decision needed | Options | Recommended default |
|---|---|---|---|
| API-OD-1 | Step-up re-authentication recency for sensitive operations (`API-018`) | 5 / 15 / 60 min; per operation | 15 minutes for policy, adjustment, role and export operations |
| API-OD-2 | Amount wire format (section 5.2) | (a) integer string + scale object; (b) integer string only, scale on Network; (c) decimal string | (a): self-describing; mismatch detection (`unit_scale_mismatch`) |
| API-OD-3 | Whether rejected requests consume an idempotency key (`API-085`) | store all final responses; store success only | Success only (retry after fixing the cause is the normal flow) |
| API-OD-4 | Retention of non-posting idempotency keys (`API-086`) | 24 h; 7 days; 30 days | 24 hours; 7 days for exports |
| API-OD-5 | Page-size default and maximum (`API-100`) | 25/100; 50/200; 100/500 | 50 default, 200 maximum |
| API-OD-6 | Default rate limits (`API-110`) | tighter or looser than the starting points given | Starting points in `API-110`, tuned in the first beta |
| API-OD-7 | Minimum deprecation notice (`API-125`) | 6 / 12 / 24 months | 12 months hosted; N and N-1 majors for self-hosted |
| API-OD-8 | OpenAPI baseline version (`API-130`) | 3.1.x; 3.2.0 | 3.1.x now, revisit when tooling supports 3.2 |
| API-OD-9 | Whether `propose_trade` returns `advisories` about the caller's own limits (TR1 early feedback) | none; caller's own account only; both parties | Caller's own account only (never the counterparty's) |

Further assumptions:

- A1: The canon's Trade lifecycle exposes `state_version` and `proposal_version` ([15](15-lifecycle-state-machines.md) TR2); the single ETag in this document combines them. If 15 changes the versioning attributes, `API-091` follows.
- A2: [15](15-lifecycle-state-machines.md) LIF-018 suggests the TR2 history record stores both parties' balances for dispute review. This API exposes only the caller's `my_balance_after` on `accept_trade`; the dispute reviewer evidence view, subject to `balance_visibility`, is specified with disputes in [04](04-functional-requirements.md). Reconcile if reviewer access differs.
- A3: Operation preconditions that refer to the closed remedy set, `admin_quorum` approvals and notice periods are owned by [03](03-domain-model-and-mutual-credit-rules.md), [10](10-governance-and-network-operations.md) and [16](16-network-policy-catalog.md); this document follows them and does not redefine them.
- A4: No canon problem found. The canon names no HTTP error codes, so the catalogue (8.3) is proposed here; names `trade_not_proposed`, `illegal_transition`, `credit_limit_exceeded` and `max_balance_exceeded` align with [15](15-lifecycle-state-machines.md) usage.
- A5: Whether third parties may ever see balances through integrations is a question for counsel and for [09](09-security-privacy-and-abuse.md) before P2 token scopes ship ([19](19-legal-and-regulatory-open-questions.md)).

## 18. References

Internal: [00 conventions](00-conventions.md); [02 personas and roles](02-personas-roles-and-journeys.md); [03 domain model](03-domain-model-and-mutual-credit-rules.md); [04 functional requirements](04-functional-requirements.md); [05 architecture](05-architecture-and-trust-boundaries.md); [06 data, ledger and audit](06-data-ledger-and-audit.md); [08 UX](08-ux-and-accessibility.md); [09 security](09-security-privacy-and-abuse.md); [10 governance](10-governance-and-network-operations.md); [11 deployment and operations](11-deployment-operations-and-support.md); [12 quality strategy](12-quality-strategy-and-test-plan.md); [13 assumptions and decisions](13-assumptions-risks-decisions-roadmap.md); [14 glossary](14-glossary.md); [15 lifecycle state machines](15-lifecycle-state-machines.md); [16 policy catalog](16-network-policy-catalog.md); [17 notifications](17-notifications-and-communications.md); [18 export and portability](18-data-export-and-portability.md); [19 legal questions](19-legal-and-regulatory-open-questions.md); [technical standards research note](../notes/research-technical-standards.md).

External (access date 2026-10-21; the authoring sandbox could not fetch standards hosts directly, so statuses below were confirmed through search-result summaries and the research note, marked accordingly):

- RFC 9457, Problem Details for HTTP APIs (July 2023, obsoletes RFC 7807): <https://www.rfc-editor.org/rfc/rfc9457.html>
- IETF Internet-Draft `draft-ietf-httpapi-idempotency-key-header`, latest -07 (2025-10-15), expired 2026-04-18, not an RFC: <https://datatracker.ietf.org/doc/draft-ietf-httpapi-idempotency-key-header/>
- RFC 9745, The Deprecation HTTP Response Header Field (March 2025): <https://www.rfc-editor.org/rfc/rfc9745.html>
- RFC 8594, The Sunset HTTP Header Field: <https://www.rfc-editor.org/rfc/rfc8594> (not re-fetched)
- RFC 9110, HTTP Semantics (ETag, If-Match, 412): <https://www.rfc-editor.org/rfc/rfc9110> (not re-fetched); RFC 6585 (428): <https://www.rfc-editor.org/rfc/rfc6585> (not re-fetched); RFC 8288 (Link): <https://www.rfc-editor.org/rfc/rfc8288> (not re-fetched); RFC 6901 (JSON Pointer): <https://www.rfc-editor.org/rfc/rfc6901> (not re-fetched); RFC 3339 (timestamps): <https://www.rfc-editor.org/rfc/rfc3339> (not re-fetched)
- RFC 9700, OAuth 2.0 Security Best Current Practice: <https://www.rfc-editor.org/rfc/rfc9700> (not re-fetched)
- IETF draft `draft-ietf-httpapi-ratelimit-headers` (research note: -11, 2026-05-23, still a draft)
- OpenAPI Specification: <https://spec.openapis.org/oas/latest.html>
- OWASP API Security Top 10 (2023): <https://owasp.org/API-Security/editions/2023/en/0x00-header/>

