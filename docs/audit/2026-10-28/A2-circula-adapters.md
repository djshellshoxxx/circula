# A2 audit: Circula HTTP adapters and access control at every entry point

Auditor: A2 (read-only code auditor, one of six). Date: 2026-10-28.
Repository: `djshellshoxxx/circula`. SHA: `bb5a36f9e6912d73fa6989f1ae00fc2017f3f18f` (detached worktree `/home/user/audit/circula-a2`, `git rev-parse HEAD` verified).
This is an AI-assisted review with executed proof-of-concept tests on synthetic data. It is not an independent professional audit.

## 1. Scope, SHA and what was reviewed

In scope (read in full unless noted):

- `internal/api`: `api.go`, `auth.go`, `middleware.go`, `route.go`, `routes.go`, `idem.go`, `cursor.go`, `decode.go`, `respond.go`, `jobs.go`, `erasure_hook.go`, `dto.go`, every `h_*.go` (accounts, admin, audit_scoped, closure, core, dispute_review, disputes, dualcontrol, export, f2, gov, histbal, listings, notify, ops, password, policy, reports, suspend, tokens, tx).
- `internal/web`: `web.go`, `session.go`, `middleware.go`, `render.go`, `refusal.go`, `ratelimit.go`, `format.go`, `histbal_view.go`, `extra.go`, every `h_*.go`, all 63 templates (URL/attribute contexts, `template.HTML` uses, forms), `static/app.js`.
- `internal/httpx/httpx.go` (cookie names, CSRF scheme, trusted-proxy client IP, rate key, Secure/HSTS, `SafeNext`, `IsBusy`).
- `docs/api/openapi.yaml` against the runtime route table, and `internal/contract/contract_test.go`.
- Wiring of adapter options only: `cmd/circula/serve.go` `newHandlers` and `ops_http.go` (to confirm what the adapters receive; cmd itself is not in scope).
- Service functions were read only where needed to judge whether an adapter passes the authenticated actor and the resolved network and does not bypass service authorisation (principal resolution, token/session auth, GetMember/GetListing visibility, audit scoping, MFA/break-glass role application).

Specs checked: 07-api-contract (API-001..API-083 incl. section 12 deviations), 08-ux-accessibility (functional/error/form rows marked built; visual rows not judged), 09-security-privacy (web/API controls), addenda SEC-007, SEC-008, SEC-012, SEC-018 (adapter parts). Prior audits read: SECURITY-AUDIT.md, -2, -3, CODE-AUDIT.md, COMPLETENESS.md, RESUME-2026-10-24.md (2026-10-28 section), CONTINUATION.md.

Not in scope and not reported: service money logic (A1); MFA/recovery/break-glass/erasure/backup/CLI internals (A3). Accepted rules (A3-01, append-only triggers) and owner-pending items (A3-04, A3-07, A3-10, A3-11, SEC-011 questions) are noted only where an adapter touches them.

## 2. Method

1. Baseline: ran the adapter test packages at the SHA before adding anything; the `-race` run was made once the first PoC files existed (all packages, including those PoCs, passed under `-race`).
2. Line-by-line read of every adapter file, building a route inventory for both surfaces (85 API operations, about 100 web routes) and recording for each route: authentication wrapper, CSRF path, network resolution, idempotency mode, and which service call carries the authorisation decision.
3. Targeted review of the FOCUS list: cookie flags/expiry/fixation/logout; token hashing, scope and network binding; authZ and cross-network ids; X-Network; CSRF on every POST and cookie-auth API call; idempotency fingerprint, scoping and replay; strict JSON; query allowlists; cursor integrity; ETag/If-Match; problem bodies and logs; template contexts and `template.HTML`; redirects; rate-limit keys; headers/CSP; OpenAPI parity.
4. Mechanical parity checks: route table vs `openapi.yaml` (script diff), documented response codes per operation (script tally), idempotency declarations vs route flags.
5. Every reported defect has a throwaway PoC test that asserts the current (defective) behaviour, so it passes while the defect exists, except A2-12 (evidence is the repository's own test `TestDisputeResponseAndReReviewPages`), A2-20 and A2-21 (code trace). PoC files (left in place as evidence):
   - `/home/user/audit/circula-a2/internal/api/zz_a2_poc_test.go` (12 tests)
   - `/home/user/audit/circula-a2/internal/web/zz_a2_poc_test.go` (9 tests, one is a negative XSS check)
   - `/home/user/audit/circula-a2/internal/contract/zz_a2_poc_test.go` (1 test)
6. Cross-referenced each finding with prior audit IDs.

## 3. Findings table

Counts: Critical 0, High 0, Medium 0, Low 13, Info 8. Confidence: CONFIRMED 20, LIKELY 1.

| ID | Sev | Confidence | Category | Spec IDs | file:line | One-line summary |
|---|---|---|---|---|---|---|
| A2-01 | Low | CONFIRMED | input validation / conformance | API-003, API-070 | internal/api/decode.go:76-83, 103-142 | Case-variant duplicate keys pass the duplicate check and the last spelling wins; wrongly-cased field names are accepted instead of refused as unknown |
| A2-02 | Low | CONFIRMED | authZ (token network binding) | API-011, SEC-015 | internal/api/h_tokens.go:30, 92; internal/api/h_f2.go:119; internal/service/tokens.go:104, 127 | A token restricted to one network lists the person's other tokens (with other network ids), revokes them, and changes the person-wide time zone |
| A2-03 | Low | CONFIRMED | authZ / idempotency | API-011, API-022 | internal/api/idem.go:161-178 | Replay cache keyed only by person+key: a read-only token (or a demoted admin) replays a cached write response, e.g. a plaintext invitation token. KNOWN SA2-17 (partly fixed) |
| A2-04 | Low | CONFIRMED | authZ consistency (fail-closed) | SEC-013 (addendum decision 4), API-018 | internal/web/session.go:148, 198-233, 236; internal/api/h_policy.go:78-83 | Break-glass `grant-admin` admin is refused by every web admin page and by API `adminOnly` pre-checks, while the service accepts them |
| A2-05 | Low | CONFIRMED | authZ (adapter-only decision) | SEC-007 addendum (RequireMFAForAdmins), API-018, SEC-009 | internal/api/h_ops.go:121-130; internal/api/h_policy.go:85-105 | GET /admin/consistency-checks/{id} and GET /admin/policy are authorised only by the adapter role check, so an admin without the mandatory second factor reads them |
| A2-06 | Low | CONFIRMED | session management | SEC-003 | internal/web/h_auth.go:91-99 | POST /logout ignores a failed revocation, says "You are signed out." and clears the cookie, but the server session stays valid |
| A2-07 | Low | CONFIRMED | concurrency (ETag) | API-026, API-027 | internal/api/h_accounts.go:10-12, 60-69; internal/api/h_policy.go:64-66, 145-154 | ETags omit the credit-limit override/expiry and scheduled (pending) policy changes, so a stale If-Match succeeds and silently undoes another admin's change. Related KNOWN API-02 (not fixed) |
| A2-08 | Low | CONFIRMED | OpenAPI vs runtime | API-077, API-029, API-030 | docs/api/openapi.yaml (all 85 ops); internal/api/respond.go:103-111; internal/api/middleware.go:52-57 | Runtime 423 network-frozen (all 52 writes), 429 (46 ops), 503 busy (81 ops), 413/415 on some writes are not documented; kin-openapi rejects them |
| A2-09 | Low | CONFIRMED | conformance (pagination) | API-034, API-036 | internal/api/h_gov.go:81-91; internal/api/h_closure.go:127-137; internal/service/challenges.go:210-227 | GET /network/policy-log returns every version (105 rows in PoC) and GET /challenges up to 200, both with next_cursor null; limit, cursor and status are ignored |
| A2-10 | Low | CONFIRMED | privacy / enumeration | SEC-005 | internal/web/h_auth.go:314-321; internal/service/networks.go:169-183 | /n/{slug}/join tells any signed-in person whether a slug exists and shows the name of an invitation-only network; every other /n/{slug} page answers 404 |
| A2-11 | Low | CONFIRMED | conformance (navigation) | UX-001 | internal/web/render.go:208-233 | Sections hidden from a suspended or departed member's navigation still answer 200 (suspended: /disputes, /policy; departed: /trades, /disputes, /policy) |
| A2-12 | Low | CONFIRMED | conformance (review step) | UX-018 | internal/web/h_disputes.go:334-346; templates/dispute_view.html (rereview-decide form); templates/admin_adjust_pending.html:26 | A re-review decision that posts a reversal is recorded on first submit with no review step; dual-control "Approve and post" also posts without a review screen |
| A2-13 | Info | CONFIRMED | UX defect | UX-031, FR-067 | internal/web/render.go:345; internal/web/h_admin_f2.go:79 | Frozen-network notice and the admin overview's last-check line show escaped `&lt;time datetime=...&gt;` markup as text |
| A2-14 | Low | CONFIRMED | functional (web flow) | FR-010 (web), UX-030 | templates/join_anon.html:3, login.html:9, register.html:10, login_mfa.html:10; internal/web/render.go:86 | Custom `qs` (url.QueryEscape) inside href is escaped again by html/template; the double-encoded `next` fails SafeNext, so a person following the invitation page's own Sign in/Create account link lands on "/" not on the invitation |
| A2-15 | Info | CONFIRMED | hardening (caching) | SEC-023 | internal/web/middleware.go:52-56; internal/web/web.go:85-93 | /static/*, /robots.txt, /favicon.ico are `Cache-Control: public` and carry a fresh `Set-Cookie: cm_csrf=...` (pre-login CSRF secret) |
| A2-16 | Info | CONFIRMED | conformance (input validation) | SEC-024 | internal/api/h_tx.go:102-104; internal/service/tx_queries.go:42 (pattern repeats on every id) | Path ids reach the store unvalidated (`x' OR '1'='1` arrives at GetTransaction); SQL is parameterised, so not exploitable |
| A2-17 | Info | CONFIRMED | conformance (forced change) | SEC-008 addendum decision 5 | internal/web/h_home.go:34-61; internal/web/h_auth.go:228-235 | A person who must change their password can still load "/" (network list) and "/join/{token}"; other pages redirect |
| A2-18 | Info | CONFIRMED | error hygiene | API-031, API-034 | internal/api/decode.go:83, 94-100 | invalid-body detail echoes Go internals ("Go struct field proposeReq.amount_minor of type string"); several reads ignore unknown query parameters |
| A2-19 | Info | CONFIRMED | OpenAPI vs runtime | API-066, API-020 | internal/api/routes.go:110; openapi.yaml POST /api/v1/tokens | OpenAPI declares an optional Idempotency-Key on POST /tokens; the route ignores it, so a retry with the same key mints a second credential |
| A2-20 | Info | LIKELY | logging / traceability | SEC-020, UX-025, ARC-022 | internal/web/middleware.go:34-42; internal/api/middleware.go:30-33; internal/service/audit.go:37 | Web request reference shown to users is a second random id unrelated to the access-log request_id; the API stores a client-chosen X-Request-Id in audit events |
| A2-21 | Info | CONFIRMED (code) | conformance (credential storage) | API-009 | internal/service/auth.go:166-169 | API tokens (and session ids) are stored as plain, unsalted, unkeyed SHA-256; spec says "salted or keyed hash" (256-bit tokens make the practical risk negligible) |

No finding (verified OK, details in section 4.22): stored XSS on 11 main pages, CSRF coverage of every web POST and cookie-auth API write, open redirect (`SafeNext` + `http.Redirect` cleaning), cursor forgery, cross-network IDOR on both surfaces, X-Network handling, cookie flags and `__Host-` use, route parity (85 = 85).

## 4. Finding details

### A2-01 Case-variant duplicate keys and non-canonical field names accepted (Low, CONFIRMED)

- Where: `internal/api/decode.go:76-83` (`strictUnmarshal`: `checkDuplicateKeys` then `json.Decoder.DisallowUnknownFields`), `:103-142` (`checkDuplicateKeys` compares raw key strings).
- Scenario: `POST /api/v1/transactions` with `{"counterparty_member_id":"<bob>","direction":"pay","amount_minor":"100","AMOUNT_MINOR":"7"}`. The duplicate check sees two different strings; `encoding/json` matches field names case-insensitively, assigns both to `AmountMinor`, last wins. Result 201 with amount 7. The exact duplicate (`amount_minor` twice) is correctly 400. `{"KIND":"offer","Title":"bike"}` creates a listing although neither name is a documented field.
- Impact: API-003 says duplicate keys and unknown fields are rejected with 400. A proxy, WAF, logging or approval layer that reads the first `amount_minor` sees 100 while the ledger posts 7 (parser differential). The idempotency fingerprint is computed on the raw body, so it does not help.
- Evidence: `go test ./internal/api -run TestA2_CaseVariantDuplicateKeysAccepted -v` -> `case-variant duplicate -> 201 {... "amount_minor":"7" ...}`, `exact duplicate -> 400`, `wrongly cased names -> 201`.
- Fix: in `checkDuplicateKeys` compare `strings.ToLower(k)` (or reject any key whose lower-case form collides), and reject keys that do not exactly match a struct tag (e.g. decode into `map[string]json.RawMessage` first and check exact names against the target's tags, or use `encoding/json/v2` with case-sensitive matching). Add the case-variant cases to `TestStrictBodyHandling`.
- KNOWN: none (prior audits recorded "strict JSON decoding" as OK).

### A2-02 Network-restricted token acts on person-wide resources (Low, CONFIRMED)

- Where: `internal/api/h_tokens.go:30` (`listTokens`), `:92` (`revokeToken`); `internal/api/h_f2.go:119` (`patchMe`); service `ListAPITokens` (`internal/service/tokens.go:127`) and `RevokeAPIToken` (`:104`) ignore `actor.NetworkScope`.
- Scenario: alice is in networks acme and other. She gives an integration a write token restricted to acme (`TokenInput.NetworkID`). With that token: `GET /tokens` lists her token "integration-for-other" with `network_id` of the other network; `DELETE /tokens/{id}` revokes it (204); `PATCH /me` changes her display time zone for all networks (200). For network data the restriction holds (`X-Network: other` -> 404; `GET /me` filters memberships; joins are refused by `joinableActor`).
- Impact: a leaked or misbehaving single-network credential reveals which other networks the person uses (other-tenant identifiers) and can disable the person's other integrations. Inconsistent with the confinement applied to `GET /me` and joins (API-011 intersection of scope and context; SEC-015 "no identifier from one network appears in another network's responses").
- Evidence: `TestA2_NetworkScopedTokenActsOnWholePerson` -> `scoped GET /tokens -> 200, mentions other network id: true`, `scoped DELETE other network's token -> 204`, `scoped PATCH /me ... -> 200`.
- Fix: in `ListAPITokens`/`RevokeAPIToken` restrict to tokens with the same `NetworkID` when `actor.NetworkScope != ""` (or refuse token management for scoped tokens with 403); refuse `SetTimeZone` for scoped tokens. Add a test.
- KNOWN: none.

### A2-03 Replay ignores the replaying credential's scope and role (Low, CONFIRMED; KNOWN SA2-17, partly fixed)

- Where: `internal/api/idem.go:161-178`. The cache key is `PersonID + key`; `idemReplay` returns the stored response before the handler (and thus before any service scope or role check) runs.
- Scenario: admin creates an invitation with a write token and `Idempotency-Key: inv-key-0001-abcdef` (201 with plaintext `token`). The same person's read-only token sends the same request: a fresh key is refused (403 forbidden), but the same key returns `201`, `Idempotent-Replayed: true`, with the invitation token. By code, the same holds for an admin who lost the role within 24 h and for a person whose password must change: the replay precedes every service check. <!-- gitleaks:allow: synthetic idempotency key, not a secret -->
- Impact: API-011 says a token's effective permissions are the intersection of its scopes and current roles. A read-only integration that knows or guesses the write client's key (keys are client-chosen; the server accepts any 1..255 printable characters) obtains credentials or data from write responses. No state change happens.
- Evidence: `TestA2_ReplayIgnoresScopeAndRole` -> `read-only replay -> 201 Idempotent-Replayed="true"` with the same invitation token.
- Fix: store the credential scope set (and token id or session id) with the record and refuse replay (`403`) when the replaying actor lacks write scope or is a different credential; alternatively re-run a cheap service authorisation check (e.g. `open(..., ScopeWrite)` plus the route's role) before replaying. Do not cache responses that contain credentials (invitations), as A3-09 did for tokens.
- KNOWN: SA2-17 ("export artefacts and API replay cache are not re-authorised"). Artefact download is fixed at this SHA (`h_export.go:119` re-checks); the replay half is not. The read-only-token angle is new.

### A2-04 Break-glass administrators are refused by adapter pre-checks (Low, CONFIRMED)

- Where: web `adminH` (`internal/web/session.go:236`) uses `rc.isAdmin()` (`:148`) on `c.Me`, which `netH` takes from `MyMemberships` (`:205`) -> `ListMembershipsOfPerson` (stored roles only). API `reqCtx.adminOnly()` (`internal/api/h_policy.go:78`) uses `c.member` from the same list. Break-glass roles are applied only inside `service.open` (`applyBreakGlassRoles`).
- Scenario: operator runs `break-glass open` and `grant-admin` for alice (the documented "lost admin" path; SEC-013 addendum decision 4: "A break-glass administrator can use all administrator powers"). The service accepts alice as admin (`ListInvitations` succeeds; API `POST /admin/invitations` 201, `GET /admin/members` 200), but every web admin page (`/n/town/admin`, `/admin/members`, `/admin/invitations`, ...) answers 403 and API `GET /admin/policy` and `GET /admin/audit-events` answer 403. Admin-only buttons on trade and dispute pages are also hidden.
- Impact: fail-closed, no bypass. The break-glass grant is unusable through the product UI, which is the operator's documented remedy for a network without a usable administrator.
- Evidence: `TestA2_BreakGlassAdminRefusedByWebAdminPages` -> `GET /n/town/admin -> 403` (x3); `TestA2_BreakGlassAdminRefusedByAPIPrechecks` -> `POST /admin/invitations 201, GET /admin/members 200, GET /admin/policy 403, GET /admin/audit-events 403`.
- Fix: expose a service read that returns the caller's effective membership (stored roles plus live break-glass roles, plus an "MFA missing" flag) and use it in `netH` and `resolveNetwork`; or drop the adapter pre-checks and let the service decide (see A2-05).
- KNOWN: none.

### A2-05 Adapter-only authorisation ignores mandatory MFA for administrators (Low, CONFIRMED)

- Where: `getCheck` (`internal/api/h_ops.go:121-130`) calls only `c.adminOnly()` and the in-memory job store; `getPolicy` (`internal/api/h_policy.go:85-105`) calls `adminOnly()` then `GetNetwork`, which needs only membership. Neither reaches `principal.requireAdmin`, which enforces `RequireMFAForAdmins` (`mfaMissing`).
- Scenario: platform setting `require-mfa-admins` on. admin (with TOTP) runs a consistency check. admin2 (admin role, no second factor): `POST /admin/consistency-checks` 403 and `GET /admin/members` 403 (service), but `GET /admin/consistency-checks/{id}` 200 (findings, ledger_seq, audit count) and `GET /admin/policy` 200 (full policy incl. pending change).
- Impact: SEC-007 addendum says an administrator without an enabled factor gets `forbidden` (`mfa-enrolment-required`) on administrator operations. Two admin reads bypass it. Data is modest (check results can include member references; policy is mostly visible to members). Same root cause as A2-04: the adapter makes its own role decision. On the web, an MFA-less admin hitting a service-checked admin page gets the generic "No access" page; the `mfa-enrolment-required` reason is not shown.
- Evidence: `TestA2_AdminOnlyPrecheckIgnoresMandatoryMFA` -> `admin without MFA: POST check 403, GET /admin/members 403, GET check 200, GET /admin/policy 200`.
- Fix: route both reads through service methods that call `requireAdmin` (e.g. `Service.AdminGetPolicy`, and store/read check results through a service call or at least call a `Service.RequireAdmin(actor, netID)` before reading the job store). Add these routes to the authorisation matrix.
- KNOWN: none.

### A2-06 Logout reports success when the session was not revoked (Low, CONFIRMED)

- Where: `internal/web/h_auth.go:91-99`: `_ = s.svc.Logout(r.Context(), tok)`.
- Scenario: the database is busy or failing when the person signs out (`Service.Logout` is a write unit; SQLITE_BUSY after the busy timeout, disk error). The page shows "You are signed out." and deletes the cookie in this browser, but the session row is not revoked; a copy of the cookie (shared computer, another tab's cookie export, a captured cookie) keeps working until idle/absolute expiry.
- Impact: SEC-003 "A session cookie captured before logout is rejected after logout"; the person is told the opposite of what happened.
- Evidence: `TestA2_LogoutFailureLeavesSessionValid` (store wrapper makes `RevokeSession` return `SQLITE_BUSY: database is locked`) -> `after 'You are signed out.', replayed cookie GET /n/town -> 200 at /n/town/`.
- Fix: on error, render a 503 page ("Not signed out. Try again.") and keep the cookie, or retry; log the error. Add a fault-injection test.
- KNOWN: none.

### A2-07 ETags miss parts of the resource; stale If-Match passes (Low, CONFIRMED; related KNOWN API-02 not fixed)

- Where: `limitETag` (`internal/api/h_accounts.go:10-12`) = hash(member id, effective limit); `policyETag` (`internal/api/h_policy.go:64-66`) = hash(network id, in-force `PolicyVersion`). Scheduling an adverse change (`UpdatePolicy`, `internal/service/networks.go:259-279`) appends a later version but does not change `PolicyVersion`.
- Scenario 1 (credit limit): ETag before = ETag after turning the default 100.00 into a member override 100.00 with `expires_at` (both `"e7bd5c3d6a300f5f238fce80"`). A second admin holding the old ETag sends `credit_limit_minor: null` and silently removes the override (200).
- Scenario 2 (policy): admin A reads the policy (ETag E). Admin B schedules "lower the default limit" with 14 days' notice; members are notified; `GET /admin/policy` now shows `"pending":{...}` but the ETag is still E. Admin A's no-op PATCH (`{"reason":"unrelated note"}`) with E returns 200 and cancels B's announced change.
- Impact: API-026 requires a strong ETag and API-027 a 412 on mismatch; lost updates of other administrators' decisions without warning.
- Evidence: `TestA2_CreditLimitETagMissesOverrideAndExpiry` -> `ETag before "e7bd...", after override+expiry "e7bd..."`, `stale If-Match PUT -> 200`; `TestA2_PolicyETagMissesScheduledChange` -> same ETag before and after scheduling, `stale If-Match no-op PATCH -> 200; pending still shown: false`.
- Fix: include override presence, override value and expiry in the limit ETag; derive the policy ETag from the latest policy version row (including pending/withdrawn state). Also carry the expected version into the service write unit (KNOWN API-02: today the ETag check and the write are separate units, `casMu` is process-local and web edits bypass it; still true at this SHA, e.g. `h_listings.go:104-151`, `h_accounts.go:60-70`).
- Note: `If-Match: *` is accepted (`decode.go:216-219`), RFC-conformant but it disables the protection (spec gap, section 6).

### A2-08 Runtime statuses not documented in openapi.yaml (Low, CONFIRMED)

- Where: `docs/api/openapi.yaml`; `codeStatus` (`internal/api/respond.go:103-111`) maps `network-frozen` to 423 and busy to 503 for any operation; the edge limiter (`middleware.go:52-57`) and the principal limiter (`route.go:73-76`) return 429 on any route.
- Evidence (script tally over 85 operations): 423 documented on 0 operations (all 52 writes can return it); 429 missing on 46; 503 missing on 81; 413 missing on 22 writes; 415 missing on 18 writes. `TestA2_UndocumentedRuntimeStatuses` (kin-openapi with `IncludeResponseStatus`) -> `frozen write: POST /listings -> 423; openapi validation error: status is not supported`, `rate limited: GET /network -> 429; ... status is not supported`. The 07 error catalogue (7.2) also lacks four codes the runtime emits: `network-frozen` (423), `approval-required` (403), `password-change-required` (403), `terms-acceptance-required` (409); OpenAPI lists them only in the code enum.
- Impact: generated clients and the contract test treat legitimate responses as contract violations; API-077 says contract tests MUST validate responses against the description, and the existing test only passes because these paths are never exercised.
- Fix: add shared responses (423, 429 with Retry-After, 503 with Retry-After, 413, 415) to every operation (or as default responses); add the four codes to 07 section 7.2; extend the contract test with a frozen network, a busy store and a rate-limited principal.
- Route parity itself is exact: 85 runtime operations = 85 documented operations (script diff, section 8).
- KNOWN: none (07 G-2 "no contract test validates live responses" is stale; such a test exists).

### A2-09 Unpaginated, unbounded list endpoints (Low, CONFIRMED)

- Where: `policyLog` (`internal/api/h_gov.go:81-91`) returns `PolicyLog` in full with `next_cursor: null`; `listChallenges` (`internal/api/h_closure.go:127-137`) returns up to 200 rows (`internal/service/challenges.go:223`); neither calls `c.only` or `c.page`, so `limit`, `cursor` and unknown or invalid `status` are ignored (`?status=bogus` -> 200 empty list, although OpenAPI enumerates `requested|decided`).
- Scenario: after 104 policy updates, `GET /network/policy-log?limit=5` returns 105 rows, `next_cursor=<nil>`.
- Impact: API-034 (list operations MUST use cursor pagination) and API-036 (no unbounded queries; an endpoint without pagination returns at most 100 rows).
- Evidence: `TestA2_PolicyLogUnbounded` -> `GET /network/policy-log?limit=5 -> 105 rows, next_cursor=<nil>`; `TestA2_DetailEchoesGoTypesAndUnknownQueryIgnored` -> `GET /network/policy-log?limit=abc -> 200`, `GET /challenges?bogus=1 -> 200`, `GET /challenges?status=bogus -> 200`.
- Fix: keyset-page both (version / challenge id), validate `status`, call `c.only(...)`; or cap at 100 and record a deviation in 07 section 12.

### A2-10 Join page discloses invitation-only networks (Low, CONFIRMED)

- Where: `joinNetForm` (`internal/web/h_auth.go:314-321`) calls `NetworkBySlug` for any signed-in person and renders `joinnet.html`, which for invite mode prints "<Name> is invitation only". `joinNetPost` returns a distinct 403 message for invite networks.
- Scenario: eve registers (registration is open) and probes slugs: `/n/town` -> 404 (non-member), `/n/no-such-net/join` -> 404, `/n/town/join` -> 200 "Net town is invitation only".
- Impact: SEC-005 says sign-up and invitation responses must not reveal whether a network exists except to an authorised party; the rest of the product deliberately hides network existence from non-members (API-017, FR-006). Names of private communities can be learned by guessing slugs (3..40 lower-case characters, often words).
- Evidence: `TestA2_JoinPageDisclosesInviteOnlyNetwork` -> `outsider: /n/town -> 404, /n/town/join -> 200, /n/no-such-net/join -> 404`.
- Fix: for join mode `invite` (and closed/setup networks) answer exactly like an unknown slug (404 page); show the join page only for `approval` and `open` networks. Decide in the spec whether open/approval networks are public by slug (section 6).

### A2-11 Hidden navigation sections reachable by URL (Low, CONFIRMED)

- Where: `rc.nav` (`internal/web/render.go:208-233`) hides Directory, Listings, Disputes and Rules from suspended members and everything except Home and History from departed members; the handlers do not apply the same rule.
- Evidence: `TestA2_HiddenSectionsReachable` -> `hidden sections answering 200: map[alice@example.org:[/disputes /policy] bob@example.org:[/trades /disputes /policy]]` (alice suspended, bob departed; Directory and Listings are refused).
- Impact: UX-001 "A hidden section MUST NOT be reachable by URL with a different result than a 403 or 404 page." The reachable pages are useful (a suspended party can still respond to a dispute), so the likely fix is to show them in the navigation rather than to block them.
- Fix: align `nav` with what the service allows (show Trades/Disputes/Rules where readable), or refuse in the handlers. Add a nav-versus-reachability test per membership state.

### A2-12 Value-moving decisions without a review step (Low, CONFIRMED)

- Where: `disputeAct` case `rereview-decide` (`internal/web/h_disputes.go:334-346`) calls `DecideReReview` directly; `dispute_view.html` posts the form with button "Record re-review decision". Contrast `resolve`, which goes through `decisionReview`. Dual-control approval: `admin_adjust_pending.html:26` "Approve and post" posts straight to `pendingAct` (`internal/web/h_dualcontrol.go:105`).
- Evidence: the repository's own test `TestDisputeResponseAndReReviewPages` (`internal/web/feedback_test.go:140-147`) submits `outcome=upheld` once and the reversal is posted (alice's balance changes) with no review page.
- Impact: UX-018 requires a review step showing direction, amounts, names and resulting balances before "creating a reversal or adjustment". Approval of a pending adjustment is arguably "creating" the posting (spec ambiguity, section 6).
- Fix: route `rereview-decide` through the same review/confirm pattern as `resolve` (step=review/confirm, `dispute_decide_review.html`); add a confirm screen for "Approve and post" that shows the resulting balance.

### A2-13 Escaped `<time>` markup shown in notices (Info, CONFIRMED)

- Where: `frozenNotice` (`internal/web/render.go:345`) and the overview check line (`internal/web/h_admin_f2.go:79`) concatenate `string(c.when(...))` (a `template.HTML` `<time>` element) into plain strings that the templates escape.
- Evidence: `TestA2_FrozenNoticeAndCheckLineShowEscapedMarkup` -> admin overview `The last ledger check passed. &lt;time datetime=&#34;2026-03-01T12:00:00Z&#34;&gt;1 Mar 2026, 12:00 UTC&lt;/time&gt;`; every member page of a frozen network shows the same raw markup in the notice. Existing tests miss it because `mustContain` unescapes the body.
- Fix: pass the time as a separate `template.HTML` field (as `FrozenSince` already does) or format plain text.

### A2-14 Invitation and sign-in links lose `next` (Low, CONFIRMED)

- Where: `"qs": url.QueryEscape` (`internal/web/render.go:86`) used inside `href` in `join_anon.html:3`, `login.html:9`, `register.html:10`, `login_mfa.html:10`. html/template escapes the query value again (custom functions are not recognised as escapers), producing `next=%252Fjoin%252Finv_...`; after one decode the value starts with `%`, `SafeNext` rejects it.
- Scenario: a person opens an invitation link while signed out, clicks the page's "Sign in" (or "Create an account"), signs in, and lands on "/" with no trace of the invitation; the page promised "You will come back to this page afterwards". Same loss on login->register, register->login and MFA "Start again".
- Evidence: `TestA2_InviteLinkNextDoubleEncoded` -> `sign-in link on the invitation page: /login?next=%252Fjoin%252Finv_...`; `hidden next on the login form = ""; after sign-in landed on /`. `TestJoinByInvitationLink` does not catch it because it builds `/login?next=` by hand.
- Fix: use `{{.Next}}` directly in the href (html/template query-escapes it) or the built-in `urlquery`; add a test that follows the rendered link.

### A2-15 Public-cacheable responses carry Set-Cookie (Info, CONFIRMED)

- Where: `wrap` calls `ensureCSRF` for every request (`internal/web/middleware.go:52`) and then sets `Cache-Control: public, max-age=3600` for `/static/` (`:55`); `/robots.txt` and `/favicon.ico` handlers do the same (`web.go:85-93`).
- Evidence: `TestA2_StaticPublicCacheWithSetCookie` -> `/static/style.css: Cache-Control="public, max-age=3600" Set-Cookie=["cm_csrf=...; Path=/; HttpOnly; SameSite=Lax"]`.
- Impact: a shared cache that stores responses with Set-Cookie would hand one pre-login CSRF secret to many clients. Login CSRF is still blocked by the Origin check, so the practical impact is small.
- Fix: skip `ensureCSRF` for static/robots/favicon, or mark them `Cache-Control: private`.

### A2-16 Identifiers not validated before database access (Info, CONFIRMED)

- Where: every `{id}` path value is passed straight to the service and store, e.g. `getTx` (`internal/api/h_tx.go:102`) -> `GetTransaction` -> `tx.GetTransaction(netID, txID)` (`internal/service/tx_queries.go:42`). No UUID format check exists in `internal/api`, `internal/web` or `internal/service`.
- Evidence: `TestA2_IdentifierReachesStoreUnvalidated` (store wrapper records ids) -> `GET /transactions/x'%20OR%20'1'='1 -> 404; ids passed to store.GetTransaction: ["x' OR '1'='1"]`.
- Impact: none exploitable (all queries are parameterised); SEC-024 acceptance "An identifier with injection characters is rejected before any database call (test)" is not met.
- Fix: one `validID` (UUID text) check in the adapters' path handling (both surfaces) returning the uniform 404; add the test.

### A2-17 Forced password change not enforced on "/" and "/join/{token}" (Info, CONFIRMED)

- Where: `home` (`internal/web/h_home.go:34`) and `joinInviteForm` (`internal/web/h_auth.go:228`) call `s.authenticate` directly instead of `s.auth`.
- Evidence: `TestA2_ForcedChangeNotEnforcedOnHomeAndJoin` -> `/ -> 200 at /`, `/join/x -> 200`, `/n/town/history -> 200 at /account/password`.
- Impact: read-only pages only; SEC-008 addendum decision 5 says every authenticated page except `/account/password` redirects.
- Fix: apply the `MustChangePassword` redirect in both handlers.

### A2-18 Problem detail echoes Go internals; unknown query parameters ignored (Info, CONFIRMED)

- `strictUnmarshal` puts the decoder message in `detail` (`internal/api/decode.go:83`): `"cannot unmarshal number into Go struct field proposeReq.amount_minor of type string"`. Not prohibited by API-031's list, but it exposes implementation names and type names; prefer "amount_minor: must be a string".
- `GET /network?bogus=1`, `/challenges?bogus=1`, `/accounts/me?foo=1`, `/network/policy-log?limit=abc` all return 200, while list endpoints reject unknown parameters with `invalid-query`. Evidence: `TestA2_DetailEchoesGoTypesAndUnknownQueryIgnored`.

### A2-19 POST /tokens: documented idempotency is ignored (Info, CONFIRMED)

- `openapi.yaml` declares `IdempotencyKeyOptional` on `POST /api/v1/tokens`; the route has no idempotency (`routes.go:110`, deliberately uncached since A3-09). A client that retries with the same key (the documented behaviour) gets a second token. Evidence: `TestA2_TokenCreateIgnoresIdempotencyKey` -> two 201 responses with different ids, no `Idempotent-Replayed`. Fix: document "no idempotency" (or implement replay of metadata without the secret, returning 409 for a repeated key). Script check found this is the only idempotency declaration that differs from the route flags.

### A2-20 Request-id handling (Info, LIKELY)

- The edge (`cmd/circula/ops_http.go` `observe`) assigns or accepts `X-Request-Id` and logs it; the web `wrap` (`internal/web/middleware.go:34-42`) then generates a different random id, overwrites the response header and shows it as the page "Reference". The access-log line for that request carries the other id, so support cannot match a user's reference to the access log (UX-025, ARC-022); only `log.Printf` internal-error lines (unstructured, no severity field, SEC-020) carry it.
- The API (`internal/api/middleware.go:30-33`) keeps a client-supplied `X-Request-Id` matching `^[A-Za-z0-9._-]{8,64}$` and the service writes it into audit events (`internal/service/audit.go:37`), so a client can choose (and duplicate) the request ids recorded in the audit chain.
- Fix: web should reuse the edge request id; generate server-side ids for audit and keep the client id as a separate correlation field.

### A2-21 Token hashes are unsalted, unkeyed SHA-256 (Info, CONFIRMED by code)

- `hashToken` (`internal/service/auth.go:166-169`) stores `sha256(token)` for API tokens and sessions. API-009 says "stored only as a salted or keyed hash". With 256-bit random tokens this is not practically attackable; a keyed hash (HMAC with a server key) would meet the text and protect against offline confirmation of a leaked token list.

### 4.22 Verified OK (no finding)

- Authentication: one cookie name/CSRF scheme for both surfaces (`httpx`); `HttpOnly`, `SameSite=Lax`, `Secure` and `__Host-` over HTTPS; only `__Host-` names read over HTTPS; session id issued fresh at login and CSRF rotated; bearer requires `cir_` prefix (a session id cannot be used as a bearer); an invalid `Authorization` header is 401 even with a cookie.
- CSRF: every web POST goes through `wrap` -> `csrfOK` (Origin/Sec-Fetch-Site plus HMAC bound to the session, or to the pre-login cookie); all registered web routes are GET or POST (the method-less catch-all `/` only renders the 404 page or the read-only home); GET forms carry no token; API cookie-auth unsafe methods require `X-CSRF-Token` before network resolution; no CORS.
- Authorisation: every `/n/{slug}/...` web route is wrapped in `netH`/`adminH`; every API route is authenticated (none is public); all service calls receive the authenticated actor and the network resolved server-side (slug -> membership, `X-Network` -> membership, token `NetworkScope` honoured in `open`); no adapter passes a client-supplied actor, role or network to the service as authority. Cross-network ids return the uniform 404 on both surfaces.
- Idempotency: records are scoped per person (no cross-principal poisoning), the fingerprint includes method, path, network and body; in-progress 409 with Retry-After; failures not cached; panic releases the key.
- Cursors: HMAC-SHA256 over position and a filter fingerprint (path, network, person, filters), `hmac.Equal`; tampering or reuse with other filters -> 400.
- Redirects: `SafeNext` blocks scheme/host, `//`, backslash, control characters and their percent-encoded forms; `http.Redirect` additionally cleans the path; every redirect target is a fixed prefix plus server ids or a `SafeNext` value.
- Templates: html/template everywhere; the only `template.HTML` values are server-formatted times (`fmtTime`); hostile display names, listing text, announcements, terms, dispute statements and report notes are escaped on 11 main pages (`TestA2_NoStoredXSSOnMainPages`, negative check).
- Headers: strict CSP without unsafe sources, `frame-ancestors 'none'`, XFO DENY, nosniff, referrer policy, `no-store`, `X-Robots-Tag`, HSTS when HTTPS.
- Rate-limit keys: client address from trusted proxies only (right-most untrusted XFF entry), IPv6 truncated to /64, per-login keys combined with the address.
- Problem bodies: `instance` redacts `inv_`/`cir_` tokens; internal errors log the route pattern, not the raw path.

## 5. Conformance table

Legend: OK, partial, not-implemented, untested, mismatch. "Doc-drift" marks status-table rows whose text no longer matches the code in either direction. Code paths are relative to the repository; tests named where they exist.

### 07 API contract (rows carry "As built" notes; COMPLETENESS lists them as built)

| Spec ID | Status | Code / test |
|---|---|---|
| API-001 | OK (dual-surface scenario test missing, see API-071) | both adapters call `*service.Service`; `internal/contract/contract_test.go` |
| API-002 | OK | `internal/api/routes.go` |
| API-003 | mismatch (A2-01) | `decode.go:76-142`; `strict_test.go:TestStrictBodyHandling` (no case-variant case) |
| API-004 | OK | `decode.go:parseMinor`; `TestStrictBodyHandling` |
| API-005 | OK with documented D-14 (UUIDv4 export ids; check id fallback `h_ops.go:112`) | `dto.go:ts` |
| API-006 | OK | merge-patch on policy/listings; credit limit is PUT as documented |
| API-007 | OK | `dto.go:direction` |
| API-008 | OK | `httpx.go`; `hardening_test.go:TestWebSessionWorksForAPI`, `TestSecureSessionCookieWorksForAPI`; web `TestHeadersAndCookies` |
| API-009 | mismatch, Info (A2-21) | `service/auth.go:166` |
| API-010 | partial (documented D-07) | `h_tokens.go:62-66` |
| API-011 | partial (A2-02, A2-03) | `routes.go:110` cookieOnly; replay `idem.go:169` |
| API-012 | partial (documented: name list + `cir_`/`inv_` prefixes; session ids have no prefix) | `middleware.go:23-24,72-87`; `TestCredentialsInQueryRejected` |
| API-013 / 014 / 015 | OK | `respond.go:155`; `auth.go:47-55`; no CORS; `TestCookieSessionNeedsCSRF` |
| API-016 / 017 | OK | service `open`; `TestAuthorisation`, service `TestCrossNetworkIdsAreNotFound` |
| API-018 | partial (A2-04, A2-05: adapter pre-checks diverge from service) | `h_policy.go:78`, `h_ops.go:121` |
| API-019 | OK | no operator route |
| API-020 / 021 / 023 / 024 / 025 | OK (A3-10 eviction owner-pending) | `idem.go`; `TestIdempotencyKeyHeaderCode`, `TestInProgressIdempotencyKeyIs409UntilFinished`, `TestPanicReleasesIdempotencyKey` |
| API-022 | partial (A2-03; "equivalent body" = whitespace-compacted bytes, reordered members -> 422, A2 spec gap) | `idem.go:152-160`; `TestA2_ReorderedRetryIsKeyReuse` |
| API-026 / 027 | mismatch (A2-07) plus documented D-05; KNOWN API-02 race not fixed | `h_accounts.go:10`, `h_policy.go:64`; `TestPolicyAndListingETags`, `TestCreditLimitAndMemberAdmin` |
| API-028 | OK (documented partial for dispute actions) | `h_tx.go:txActionError` |
| API-029 / 030 / 031 / 032 / 033 | OK (Info A2-18; catalogue lacks 4 emitted codes, A2-08) | `respond.go`; contract test |
| API-034 | partial (A2-09) | `cursor.go`; `TestCursorPagination` |
| API-035 | OK | `cursor.go:22-49`; `TestEntriesOrderedByLedgerSeq` |
| API-036 | mismatch (A2-09) | `h_gov.go:81`, `h_closure.go:127` |
| API-037 / 038 / 039 | OK (D-08 documented) | `middleware.go`; `TestBodySizeLimit`, `TestRateLimit`, `TestIPRateLimitUsesTrustedProxyClientIP` |
| API-040 .. API-067a | OK per contract test (85 ops exercised and schema-validated), with: API-061 web join flow loses `next` (A2-14); API-063 read authorised in adapter only (A2-05); API-066 no web UI to create/revoke tokens and OpenAPI idempotency mismatch (A2-19) | `internal/contract/contract_test.go:TestAPIMatchesOpenAPI` |
| API-068 / 069 / 070 / 072 | OK (API-070 except case variants, A2-01) | `trade_test.go`, `TestStrictBodyHandling` |
| API-071 | partial (no shared HTML+API scenario contract test; TRACEABILITY agrees) | none |
| API-077 | partial: parity exact, exercised responses validated, but undocumented runtime statuses (A2-08); 07 text "no contract test validates live responses" is doc-drift | `contract_test.go`, `TestA2_UndocumentedRuntimeStatuses` |
| API-083 | OK | `h_histbal.go`; `h_histbal_test.go`, `histbal_route_test.go`, contract test |

### 08 UX (rows marked built in the 08 table; functional behaviour only)

| Spec ID | Status | Code / test |
|---|---|---|
| UX-001 | mismatch (A2-11) | `render.go:208`; `TestA2_HiddenSectionsReachable` |
| UX-003, 004, 006, 008, 009, 010, 011, 013, 014, 015, 016 | OK | `format.go`, `render.go`, `dashboard.html`, `history.html`; `flow_test.go:TestLoginTradeFlow` |
| UX-012 | OK | `h_notify.go:limitNotice`; `TestLimitNoticePages` |
| UX-018 | partial (A2-12) | `h_disputes.go:334`; `feedback_test.go:140` shows the gap |
| UX-019 | partial / spec gap (trade acceptance and adjustment flashes state the balance; reversal, dispute decision, re-review and dual-control approval do not) | `h_trade_act.go:129-132`, `h_admin2.go:227`, `h_admin_f2.go:229` |
| UX-020 | OK with minor gap (forbidden/not-found on a POST show generic pages without "nothing changed") | `refusal.go`, `render.go:264-291` |
| UX-025 | OK (reference correlation Info A2-20) | `render.go:285-287`; `TestErrorPagesHideInternals` |
| UX-029 | OK | `TestAuthorisationAndIsolation` |
| UX-030 / UX-059 | OK (warning, extend, return via next on GET; A2-14 loses next on secondary links) | `h_account.go:70-98`, `app.js`; `TestSessionExpiryWarning` |
| UX-031 | OK (A2-13 rendering defect in the frozen variant) | `render.go:235-261` |
| UX-032 | partial: remove member, cancel trade, close network confirmed; "revoke token" has no web UI at all | `admin_member_act.html`, `trade_confirm.html`, `h_admin_network.go:112` |
| UX-041 .. 046, 048, 051 | OK (structural tests only; no visual judgement) | `a11y_test.go`, `a11y_security_test.go` |
| UX-061 | OK | `histbal_view.go`; `histbal_page_test.go`, `histbal_view_test.go` |
| 08 table generally | doc-drift: UX-022, 023, 024, 052, 053, 037 marked partial/not built but implemented (COMPLETENESS agrees); the table is a historical 925f637 snapshot | |

### 09 Security and privacy (web/API controls) and addenda

| Spec ID | Status | Code / test |
|---|---|---|
| SEC-003 | partial: cookie flags, idle/absolute expiry, revoke on password change/recovery OK; logout failure not surfaced (A2-06); no session regeneration at privilege change and no invalidation on role removal (acceptance met only through per-request role checks; spec gap) | `session.go`, `service/auth.go:184-205`; `TestPasswordChange`, `TestSecureCookiesAndHSTS` |
| SEC-004 | OK; untested: no route-inventory test ("No GET route changes ... state"); doc-drift (table says token bound to a random cookie; it is session-bound) | `httpx.go:65-92`, `session.go:93-107`; `TestCSRFRejections`, `TestCookieSessionNeedsCSRF` |
| SEC-005 | partial (A2-10) | `h_auth.go:314`; `TestRegisterIsUniform`, `TestWebRecoveryFlow` |
| SEC-006 | OK for adapter limiters (per-login service throttle out of scope) | `ratelimit.go`, `httpx.go:129-178`; `TestLoginLimitPerClientBehindProxy`, `TestA3_01_WebIPv6CrossPrefixNoLockout` |
| SEC-007 (addendum, web) | OK on web; API: no MFA endpoints (documented gap); A2-05 bypass of mandatory MFA on two adapter-authorised reads | `h_security.go`; `TestWebMFAEnrolAndLogin`, `TestWebMFAChallengeCSRFAndGet` |
| SEC-008 (addendum, web) | OK with Info A2-17 | `h_recover.go`; `TestWebRecoveryFlow`, `TestWebAdminReset`, `TestForcedPasswordChangeOnAPI` |
| SEC-009 | partial / untested: service matrix `TestAuthorizationMatrix` covers 21 operations x 10 principal types, HTTP-level checks cover a handful of routes; no route inventory test; adapter-only decisions (A2-05) are outside the matrix | `service/authz_test.go`, `api/authn_test.go:TestAuthorisation`, `web/security_test.go:TestAuthorisationAndIsolation` |
| SEC-012 (addendum, web) | OK | `h_security.go`; `TestWebSecurityNoticesAndDisable` |
| SEC-013 (adapter part) | partial (A2-04) | `h_breakglass.go`; `TestBreakGlassPage` |
| SEC-015 | OK for network data; A2-02 for person-level token endpoints | `TestCrossNetworkIdsAreNotFound`, `TestAuthorisation` |
| SEC-016 (noindex part) | OK (doc-drift: table says no-index absent) | `middleware.go:41`, `base.html:6`; `TestNoIndex` |
| SEC-017 (adapter part) | OK | `h_export.go:114-131` re-checks scope; `TestExportCapPerPerson`, contract test |
| SEC-018 (addendum, web) | OK | `h_erasure.go`; `TestErasurePages`, `TestWebBreakGlassErasureNegative` |
| SEC-019 (read path) | OK (member/reviewer reads via `/audit-events` and `/n/{slug}/audit`); doc-drift (table says member read in service only) | `h_audit_scoped.go`; `TestMemberAuditPage` |
| SEC-020 (adapter logs) | partial (A2-20) | `route.go:191`, `render.go:324` |
| SEC-022 (cookie part) | OK | `httpx.go`, wiring `serve.go:286-295`; `TestSecureCookiesAndHSTS`, `TestSecureDeploymentIgnoresPlainCookies` |
| SEC-023 | OK (Info A2-15) | `middleware.go` (web and API); `TestHeadersAndCookies`, `TestSecurityHeadersAndRequestID` |
| SEC-024 | partial (A2-16; SafeNext and escaping OK) | `httpx.go:221`; `TestSafeNextRejectsBrowserTricks`, `TestA2_IdentifierReachesStoreUnvalidated` |

## 6. Spec gaps (not bugs; no policy proposed)

1. API-022/023 "equivalent body (after normalisation)": normalisation is not defined. The API layer treats reordered JSON members as a different body (422); the durable service layer compares its own input structs, so the same retry is a replay after a restart. Define the normalisation.
2. API-026/027: whether `If-Match: *` is acceptable for the required-precondition resources (RFC 9110 allows it; it disables the lost-update protection).
3. UX-018: whether approving a pending dual-control adjustment counts as "creating ... an adjustment" that needs its own review step.
4. UX-019: "after a successful posting ... state the new balance": whose balance when an administrator or reviewer posts on behalf of others (reversal, dispute decision, approval)?
5. SEC-005 vs FR-012: whether open/approval networks are discoverable by slug for any signed-in person, and whether invite-only networks must be indistinguishable from unknown slugs.
6. SEC-003: "regenerated at any privilege change" and "invalidated on role removal" are not implemented; the as-built design re-evaluates roles on every request. Record the decision or require session rotation.
7. Token network restriction (`TokenInput.NetworkID`) is an as-built extension not described in 07 (API-010/011): its intended reach over person-level endpoints (token list/revoke, time zone, password, GET /me) is unspecified.
8. 07 section 7.2 catalogue does not list `network-frozen`, `approval-required`, `password-change-required`, `terms-acceptance-required` (API-074 allows additions; they should be recorded).
9. 08 section 3 "Account and security" lists API tokens and a session list; neither has a web screen (tokens can only be created through `POST /api/v1/tokens` with a browser cookie and a manually copied CSRF header). Noted earlier in SECURITY-AUDIT.md "Notes"; still true.

## 7. Not reviewed / coverage limits

- Service internals behind the adapters (money rules, MFA, recovery, break-glass, erasure, backup, CLI) except where needed to judge the adapter call; those belong to A1/A3.
- No browser was run: Playwright e2e suites (`e2e/tests/*.spec.js`), real cookie behaviour in browsers, axe checks, screen-reader and keyboard behaviour, visual/UX rows (UX-033..040, 054..057) were not exercised.
- No real reverse proxy/TLS terminator; trusted-proxy behaviour judged from code and existing tests.
- The KNOWN API-02 race (ETag check and write in separate units; web bypasses `casMu`) was traced, not reproduced with an interleaving test.
- A2-20 (request-id correlation) traced through `cmd/circula/ops_http.go` and the web/API middleware; not executed end to end.
- Performance/DoS of per-request full-directory loads on the propose and adjust forms (`h_trades.go:84-96`, `h_admin2.go:127-146`; KNOWN WEB-03, now uncapped) not measured.
- govulncheck and dependency review not run (out of scope; known proxy limits).
- Speculative appendix (not findings): (S1) one principal can fill the 10,000-record replay cache and evict others' completed records (A3-10, owner-pending); (S2) the global export store (50 artefacts) can still be filled by ten members in an open-join network (KNOWN SA-11, partly fixed by the per-person cap of 5); (S3) the `cm_sstart` MAC is compared with `==` (`h_account.go:78`), affecting only the client-side expiry warning; (S4) pager links in `history.html:21` and `admin_audit.html:11` are built with `printf` from raw query values (self-only parameter pollution); (S5) `cmd` accepts request ids up to 128 printable characters while the API only keeps `[A-Za-z0-9._-]{8,64}`, so some requests get different ids in the access log and the API response.

### KNOWN cross-reference (prior IDs in this scope, status at bb5a36f)

| Prior ID | Status here |
|---|---|
| CODE-AUDIT API-01 (cookie names), API-03, API-04, API-06, API-07, API-08, API-09 | fixed |
| CODE-AUDIT API-02 (If-Match optional; check-then-write race; web bypasses `casMu`) | not fixed (D-05 documents the optional part); see A2-07 |
| CODE-AUDIT API-05 / SA-11 (export store) | partly fixed (per-person cap 5; in-memory D-13 remains); S2 |
| CODE-AUDIT WEB-01, WEB-02, WEB-04 | fixed |
| CODE-AUDIT WEB-03 (full directory loads) | not fixed (now uncapped paging per request) |
| SA-01 (open redirect), SA-02 (Secure cookies), SA-05 (proxy IP), SA-10 (registration enumeration), SA-13 (token in logs) | fixed |
| SA-03 (anyone can create a network) | mitigated: `setup-secret` wired, start-up warning when unset; open by default |
| SA2-14 (`__Host-` cookies), SA2-15 (IPv6 /64) | fixed |
| SA2-17 (replay cache and artefacts not re-authorised) | partly fixed: artefact download re-checks; replay does not (A2-03) |
| A3-06 (erasure hook), A3-09 (token not cached) | fixed (A2-19 is a doc mismatch left by A3-09) |
| A3-10 (replay eviction) | owner-pending, unchanged |
| E-06..10 (flash cookie not `__Host-`) | unchanged |
| SECURITY-AUDIT.md note: POST /tokens unreachable from the product UI | still true (section 6 item 9) |

## 8. Commands run

All in `/home/user/audit/circula-a2` with `go version go1.26.10 linux/amd64`.

| Command | Result |
|---|---|
| `git rev-parse HEAD` | `bb5a36f9e6912d73fa6989f1ae00fc2017f3f18f` |
| `go test -count=1 ./internal/api/... ./internal/web/... ./internal/httpx/... ./internal/contract/...` (baseline, before PoCs) | pass (api 14.2 s, web 18.2 s, httpx, contract 1.4 s) |
| `CGO_ENABLED=1 go test -race -count=1 ./internal/api/... ./internal/web/... ./internal/httpx/... ./internal/contract/...` (run with the first 14 PoC tests present) | pass (api 243 s, web 203 s, httpx 1 s, contract 10 s) |
| route parity: `grep` of `routes.go`/`h_dualcontrol.go` vs Python parse of `openapi.yaml` paths, `diff` | identical, 85 operations (74 paths) |
| response-code tally per operation (Python over `openapi.yaml`) | 423 on 0/85; 429 missing on 46; 503 missing on 81; 413 missing on 22 writes; 415 missing on 18 writes |
| idempotency declaration vs route flags (Python) | only `POST /tokens` differs (documented optional, runtime none) |
| `go vet ./internal/api/... ./internal/web/... ./internal/httpx/... ./internal/contract/...` (with PoCs) | pass |
| `gofmt -l` on the three PoC files | clean |
| `go test -count=1 -run 'TestA2' -v ./internal/api ./internal/web ./internal/contract` | pass: 12 API, 9 web, 1 contract PoC tests (each asserts the defect is present; the XSS test is a negative check) |
| `go test -count=1 ./internal/api/... ./internal/web/... ./internal/httpx/... ./internal/contract/...` (with PoCs) | pass (api 27.8 s, web 31.4 s, httpx, contract 3.3 s) |

PoC test index (file -> tests):

- `internal/api/zz_a2_poc_test.go`: `TestA2_CaseVariantDuplicateKeysAccepted` (A2-01), `TestA2_NetworkScopedTokenActsOnWholePerson` (A2-02), `TestA2_ReplayIgnoresScopeAndRole` (A2-03), `TestA2_BreakGlassAdminRefusedByAPIPrechecks` (A2-04), `TestA2_AdminOnlyPrecheckIgnoresMandatoryMFA` (A2-05), `TestA2_CreditLimitETagMissesOverrideAndExpiry` and `TestA2_PolicyETagMissesScheduledChange` (A2-07), `TestA2_PolicyLogUnbounded` (A2-09), `TestA2_IdentifierReachesStoreUnvalidated` (A2-16), `TestA2_DetailEchoesGoTypesAndUnknownQueryIgnored` (A2-18, A2-09), `TestA2_TokenCreateIgnoresIdempotencyKey` (A2-19), `TestA2_ReorderedRetryIsKeyReuse` (spec gap 1).
- `internal/web/zz_a2_poc_test.go`: `TestA2_BreakGlassAdminRefusedByWebAdminPages` (A2-04), `TestA2_LogoutFailureLeavesSessionValid` (A2-06), `TestA2_JoinPageDisclosesInviteOnlyNetwork` (A2-10), `TestA2_HiddenSectionsReachable` (A2-11), `TestA2_FrozenNoticeAndCheckLineShowEscapedMarkup` (A2-13), `TestA2_InviteLinkNextDoubleEncoded` (A2-14), `TestA2_StaticPublicCacheWithSetCookie` (A2-15), `TestA2_ForcedChangeNotEnforcedOnHomeAndJoin` (A2-17), `TestA2_NoStoredXSSOnMainPages` (negative check).
- `internal/contract/zz_a2_poc_test.go`: `TestA2_UndocumentedRuntimeStatuses` (A2-08).

Nothing was committed or pushed; no other directory was modified.
