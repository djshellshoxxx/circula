# Circula: technical standards research notes

Date: 2026-10-21. Scope: standards and practice for a mutual-credit ledger platform.

Verification legend. **[V]** = checked via web search this session (2026-10-21). **[K]** = from established background knowledge, stable, not re-fetched this session; re-check before citing externally. **[U]** = unverified or secondary-source only. Note: datatracker.ietf.org could not be fetched directly (DNS failure in sandbox), so IETF draft status comes from search-result snippets and is marked accordingly.

---

## 1. Ledger design

### Facts
- **Double-entry model.** An immutable *transaction* (journal entry) holds two or more *postings* (legs) against *accounts*; the signed amounts sum to zero per unit. Mutual credit fits naturally: every member account may go negative down to a per-account credit limit, and the sum of all balances in a unit is always zero. Formance Ledger (Numscript), Modern Treasury Ledgers and TigerBeetle all use this shape (debit/credit or source/destination). [K]
- **Append-only.** Entries are never updated or deleted; corrections are new compensating (reversing) transactions that reference the original. Modern Treasury, Square and Stripe write-ups all stress immutability + reversal as the audit foundation. [K]
- **Balances are derived.** Source of truth = postings. Cached balances (a column on the account, or a projection table) are an optimisation and must be updated in the same DB transaction as the postings, or be rebuildable from them. Pattern: `account_balance` row updated transactionally + periodic reconciliation job that recomputes `SUM(postings)` and alerts on drift. [K]
- **TigerBeetle** (purpose-built OLTP ledger DB): fixed-size 128-byte Account and Transfer records, 128-bit integer amounts, **debits_must_not_exceed_credits / credits_must_not_exceed_debits** flags that enforce balance limits atomically in the engine, `user_data` fields for linking to external IDs, `pending`/`post`/`void` two-phase transfers, `linked` flag for atomic chains, and client-supplied unique 128-bit transfer IDs which make retries idempotent (re-submission returns `exists`). Single-threaded deterministic state machine, so no isolation-level anomalies. Key lesson: *constraint enforcement inside the same atomic step as the write*, and *idempotency by client-chosen immutable ID*. [K] (docs.tigerbeetle.com)
- **Formance Ledger**: transactions built from postings, accounts addressed by hierarchical colon paths, metadata on transactions/accounts, reversible transactions, "overdraft" controls expressed in Numscript, and a hash-chained log (each log entry commits to the previous) for tamper evidence. Idempotency via `Idempotency-Key` header. [K/U on current v2 details]
- **Modern Treasury "How to Scale a Ledger" series** (Parts I–V): immutable ledger entries, ledger transactions with balanced entries, **pending / posted / archived** entry states, balance denormalisation with optimistic **version** columns, lock-free balance reads via "available" vs "posted" vs "pending" balance, and use of `lock_version` / row locks plus idempotency keys for concurrency. [K]
- **Square / Stripe**: Square's "Books" immutable double-entry service and Stripe's ledger blog posts both emphasise: append-only, event-sourced derivation of state, idempotent writes, and audit/reconciliation as first-class (Stripe: "Ledger: Stripe's system for tracking and validating money movement"). [K]
- **Concurrency for credit-limit checks.** The invariant "balance + delta >= -limit" is a read-modify-write across rows, so it is vulnerable to **write skew / lost update** under PostgreSQL's default READ COMMITTED (and even REPEATABLE READ allows write skew in some multi-row cases). Valid approaches: (a) `SELECT ... FOR UPDATE` on the affected account rows, **always locking in a deterministic order (e.g. ascending account id)** to avoid deadlocks, then check + insert; (b) conditional atomic update `UPDATE accounts SET balance = balance + $d WHERE id=$id AND balance + $d >= -credit_limit` and check rows-affected = 1; (c) SERIALIZABLE (SSI in PostgreSQL) with automatic retry on `40001`; (d) optimistic version column + retry. PostgreSQL docs: SERIALIZABLE requires application retry on serialization failure. [K] (postgresql.org/docs/current/transaction-iso.html)
- **Idempotency.** Client supplies a unique key per logical operation; server stores (key, request fingerprint, response) in a unique-constrained table in the same transaction as the ledger write; replay with same key and same payload returns the stored result; same key with different payload returns an error (Stripe semantics, and the IETF draft's 422/409 semantics). [K]

### Recommendation for Circula
1. Model: `transactions` (id UUIDv7, unit_id, kind, effective_at, recorded_at, reverses_id nullable, idempotency_key, metadata) and `postings` (id, transaction_id, account_id, amount BIGINT signed, seq). Enforce `SUM(amount)=0` per transaction with a deferred constraint trigger or in a single DB function, plus an application assertion. Revoke UPDATE/DELETE on both tables for the app role; use a trigger that raises on UPDATE/DELETE as defence in depth.
2. Keep a cached `account_balance(account_id, balance, version, updated_at)` updated in the same DB transaction, using the **conditional atomic UPDATE** form (b) for credit-limit enforcement (simple, no explicit serializable retry). Lock multi-account transfers in ascending account-id order. Add a nightly job recomputing balances from postings and failing loudly on drift; also assert global `SUM(balance)=0` per unit.
3. Run the ledger write path at READ COMMITTED with the atomic conditional update + ordered row locks; use SERIALIZABLE (with retry loop) for any rarer multi-step admin operation that reads then writes (e.g. credit-limit changes, unit-wide reconciliation fixes). Document the chosen approach in an ADR and add a concurrency test (N parallel transfers racing against a limit; invariant must hold).
4. Idempotency key mandatory on all money-moving POSTs; `UNIQUE(unit_id, idempotency_key)` on transactions; store request hash.
5. Corrections only via reversing transactions; never delete. Optionally add a hash chain (each transaction row stores `prev_hash`/`hash`) for tamper evidence, as Formance does; cheap and valuable for community trust.
6. Support `pending`/`posted` states only if a real use case emerges (e.g. two-party approval); otherwise keep v1 posted-only.
7. Do not adopt TigerBeetle/Formance as a hard dependency in v1 (operational weight, licence/ops complexity [U on Formance licence terms]); borrow their semantics so migration stays possible.

Sources: TigerBeetle docs; Formance docs; Modern Treasury "How to Scale a Ledger"; Stripe/Square engineering blogs; PostgreSQL transaction-isolation docs (URLs in Sources).

---

## 2. Identifiers

### Facts
- **RFC 9562 (May 2024)** obsoletes RFC 4122 and defines UUIDv6, **UUIDv7**, UUIDv8. UUIDv7 = 48-bit Unix-ms timestamp + version/variant bits + 74 bits of random/counter data; time-ordered, so B-tree index inserts are mostly append-like (far better locality than UUIDv4). Spec permits monotonic counters within the same millisecond (method guidance in §6.2). [K]
- UUIDv7 leaks creation time to ms precision; that is acceptable for ledger entries (they carry timestamps anyway) but note it for member-facing IDs. [K]
- **ULID**: 128-bit, 48-bit ms timestamp + 80 random bits, Crockford base32 (26 chars), lexicographically sortable. Not an IETF/ISO standard (community spec), no registered variant bits, so not a valid RFC-conformant UUID unless re-encoded. [K]
- PostgreSQL: native `uuid` type; built-in `uuidv7()` function arrived in PostgreSQL 18 (released Sept 2025). [K, re-check exact function availability on your target version]

### Recommendation for Circula
- Use **UUIDv7** (RFC 9562) as primary key for all entities, stored in native `uuid` columns; generate in app (library) or `uuidv7()` on PG18+. Enforce monotonicity within a process. Expose as canonical lowercase hyphenated strings in the API.
- Avoid ULID (no benefit over UUIDv7, ecosystem is weaker, and not a standard UUID).
- For member-facing references that humans type or read aloud (payment requests, receipts), add a separate short, checksummed, non-sequential public code; do not rely on UUIDs for that.
- Never use ID ordering as a business ordering; use a ledger sequence or `(recorded_at, id)` for deterministic ordering.

---

## 3. API design

### Facts
- **RFC 9457** (July 2024) "Problem Details for HTTP APIs" obsoletes RFC 7807. Media type `application/problem+json`; members `type`, `title`, `status`, `detail`, `instance`; extension members allowed; supports multiple-problem guidance and a registry of problem types. [K]
- **Idempotency-Key header**: `draft-ietf-httpapi-idempotency-key-header` is an HTTPAPI WG Internet-Draft, **not an RFC**. Latest official revision found: **-07 (2025-10-15), shown as expired on 2026-04-18** [V]; a greenbytes "-latest" diff suggests a July 2026 text exists, **unverified** [U]. Header value is an RFC 8941 Structured Field String; defines error handling (400 missing key if required, 422 for key reuse with different payload, 409 for concurrent in-flight request). Stripe's de-facto semantics are widely copied.
- **RateLimit headers**: `draft-ietf-httpapi-ratelimit-headers`, latest found **-11 (2026-05-23)**, still an Internet-Draft [V, via search snippet]. Modern form defines `RateLimit-Policy` and `RateLimit` fields (replacing the older `RateLimit-Limit/Remaining/Reset` triplet from drafts -00..-07). `Retry-After` (RFC 9110) remains the standard for 429/503. [V/K]
- **OpenAPI**: 3.1.x is the widely-supported baseline (full JSON Schema 2020-12 alignment); **OpenAPI 3.2.0 released September 2025** (Tag Object rework, QUERY method, streaming/sequential media types, OAuth2 device flow); patch release 3.1.2 shipped alongside. Tooling support for 3.2 lags [V]. 
- **ETag / If-Match** (RFC 9110 §8.8.3, §13.1.1): strong ETag + `If-Match` on PUT/PATCH/DELETE gives optimistic concurrency; mismatch → `412 Precondition Failed`; missing required precondition → `428 Precondition Required` (RFC 6585). [K]
- **Timestamps**: RFC 3339 (profile of ISO 8601) `date-time` with `Z` or offset; JSON Schema `format: date-time`. [K]
- **Pagination**: offset pagination is unstable under concurrent inserts and slow at depth; **keyset/cursor** pagination with an opaque cursor encoding `(sort key, id)` is stable and index-friendly. RFC 8288 `Link` header with `rel="next"` is the standard link relation mechanism. [K]
- **Versioning**: common choices are URL major version (`/v1`) or header/media-type; additive changes are non-breaking. [K]

### Recommendation for Circula
- Errors: always `application/problem+json` per RFC 9457; define stable `type` URIs under the project docs domain (e.g. `/problems/credit-limit-exceeded`) plus extension members (`account_id`, `available`, `limit`); never leak stack traces. Document each type in OpenAPI.
- Idempotency: implement `Idempotency-Key` (Structured Field string) on every POST that creates ledger or account state, following the draft semantics (400/409/422); mark in docs as "following draft-ietf-httpapi-idempotency-key-header, not yet an RFC" and keep server logic decoupled so changes in the draft are cheap.
- Pagination: cursor-based only for ledger/transaction lists (`?limit=&cursor=`, response includes `next_cursor`; cap `limit` at e.g. 200). Sort order fixed `(recorded_at, id)`.
- Versioning: URL major version `/v1`; additive changes only within a major; publish a deprecation policy using `Deprecation` and `Sunset` headers (RFC 8594 Sunset; `Deprecation` header RFC 9745 [U on RFC number, verify]).
- Spec: author in **OpenAPI 3.1.x** now; revisit 3.2 once chosen codegen/validator tooling supports it. Lint in CI (Spectral).
- Rate limiting: return 429 with `Retry-After` and emit `RateLimit`/`RateLimit-Policy` per current draft, flagged experimental; avoid hard coupling client logic to header format.
- Concurrency on mutable non-ledger resources (profile, listings, credit-limit settings): strong ETag + `If-Match`, 428 if absent, 412 on mismatch. Ledger transactions themselves use idempotency, not ETag.
- All timestamps RFC 3339 in UTC (`Z`), ms precision; separate `effective_at` from `recorded_at`.

---

## 4. AuthN / AuthZ

### Facts
- **NIST SP 800-63B-4** (Authentication and Authenticator Management) is **final, July 2025**, superseding SP 800-63B (2020) as part of SP 800-63 Rev. 4 [V]. Notable content [K, from the publication; re-read before citing clauses]: passwords: minimum 15 chars when used as sole factor (8 when part of MFA), maximum allowed at least 64, **no composition rules, no periodic forced rotation**, check against blocklists of compromised/common passwords, allow paste and password managers, Unicode accepted; syncable authenticators (synced passkeys) explicitly permitted up to AAL2; phishing-resistant authentication required at AAL3 and must be offered at AAL2; session and reauthentication timing requirements per AAL; SMS/PSTN OOB is "restricted" authenticator.
- **WebAuthn**: W3C WebAuthn Level 2 is a Recommendation (Apr 2021); **Level 3** is at least Candidate Recommendation / advanced stage [U: exact current maturity not re-verified]. Passkeys = discoverable WebAuthn credentials, optionally synced; FIDO Alliance maintains passkey guidance. [K]
- **OAuth 2.1**: `draft-ietf-oauth-v2-1` is still an **Internet-Draft, not an RFC**; revisions found: -15 (2026-03-02) and a record for -16 dated 2026-09-02 [V, via snippets; -16 date partially unverified]. It consolidates RFC 6749 + 6750 + BCP: mandatory PKCE for auth-code, no implicit grant, no ROPC grant, exact redirect URI matching, bearer tokens not in query strings. **OAuth 2.0 Security BCP is RFC 9700** (Jan 2025) [K]. **OIDC Core 1.0** remains the identity layer (OpenID Foundation). **RFC 9449 DPoP** (sender-constrained tokens) and **RFC 7636 PKCE** are published RFCs. [K]
- **Session management** (OWASP Session Management Cheat Sheet): high-entropy opaque IDs (>=64 bits), `Secure; HttpOnly; SameSite=Lax/Strict` cookies, `__Host-` prefix, rotate on login/privilege change, idle + absolute timeouts, server-side invalidation on logout. [K]
- **Account recovery** (OWASP Forgot Password cheat sheet; NIST): uniform responses (no user enumeration), single-use, short-lived, high-entropy tokens stored hashed, rate-limit, notify user on change, do not auto-login from reset link, recovery codes at enrolment or a second passkey; avoid knowledge-based questions. [K]
- **OWASP ASVS 5.0.0** released 30 May 2025 (~17 chapters, ~350 requirements; versioned requirement IDs like `v5.0.0-1.2.5`) is the latest found [V]; 4.0.3 is the prior version. Levels L1–L3. 
- **OWASP API Security Top 10 – 2023** (API1 BOLA, API2 Broken Authentication, API3 Broken Object Property Level Authorization, API4 Unrestricted Resource Consumption, API5 Broken Function Level Authorization, API6 Unrestricted Access to Sensitive Business Flows, API7 SSRF, API8 Security Misconfiguration, API9 Improper Inventory Management, API10 Unsafe Consumption of APIs) is the current edition as far as known; **verify no 2026 edition** [U].
- **CSRF**: cookie-session apps need CSRF defence: `SameSite` cookies + synchronizer token or double-submit (signed), plus `Origin`/`Sec-Fetch-Site` checking. **CORS**: explicit origin allowlist, no `*` with credentials, restrict methods/headers, short preflight cache. Bearer-token APIs are not CSRF-prone but are XSS-exposed if stored in JS-readable storage. [K]

### Recommendation for Circula
- Target **ASVS 5.0.0 Level 2** as the stated security baseline; track requirement IDs in a checklist doc.
- Authentication: passkeys (WebAuthn) as the primary and recommended method, password + TOTP as fallback; passwords follow SP 800-63B-4 (length-only policy, min 15 for password-only, blocklist check via k-anonymity HIBP-style or local list, argon2id hashing; no forced rotation). Avoid SMS OTP as sole second factor. Treat NIST AAL2 as the target for admin/treasurer roles; AAL1 acceptable for ordinary members, with step-up for credit-limit or admin actions.
- Sessions: first-party web app uses server-side sessions in `__Host-` cookies (SameSite=Lax, HttpOnly, Secure), CSRF token for state-changing requests, idle 30 min / absolute 12 h for admin, longer for members with re-auth for sensitive actions. Prefer this over JWT-in-localStorage.
- Third-party/API access (mobile apps, integrations): OAuth 2.0 authorization code + PKCE per RFC 9700 and the OAuth 2.1 draft rules, OIDC for identity; do not implement implicit/ROPC; scope tokens narrowly; plan for DPoP later. Don't claim "OAuth 2.1 compliant" in docs (draft).
- Recovery: emailed single-use hashed token (15 min TTL) plus recovery codes/second passkey; for community networks, optionally allow **admin-assisted recovery with a two-person rule** and audit entry. Always notify the member on recovery and on new passkey/credential.
- AuthZ: central policy layer, deny by default, object-level checks on every request (BOLA is API1): a member may only read accounts they own or are a party to; unit-scoped roles (member, treasurer, admin); tenant (network) isolation enforced in queries and optionally PostgreSQL RLS. Map the API tests to OWASP API Top 10 2023 items.
- CORS allowlist per deployment; CSRF as above; security headers (CSP, HSTS, `Referrer-Policy`, `X-Content-Type-Options`).

---

## 5. Accessibility

### Facts
- **WCAG 2.2** is a W3C Recommendation (Oct 2023; republished with errata/updates Dec 2024); backwards compatible with 2.1 (adds criteria, removes 4.1.1 Parsing as obsolete). **WCAG 3.0** remains a Working Draft [K/U on current stage].
- WCAG 2.2 AA criteria most relevant to a ledger UI [K]:
  - Forms: 1.3.1 Info and Relationships; 1.3.5 Identify Input Purpose (`autocomplete`); 3.3.1 Error Identification; 3.3.2 Labels or Instructions; 3.3.3 Error Suggestion; **3.3.4 Error Prevention (Legal, Financial, Data)**: for financial transactions provide reversibility, checking, or confirmation, directly relevant to payments; **3.3.7 Redundant Entry (A, new)**; **3.3.8 Accessible Authentication (Minimum) (AA, new)**: no cognitive function test (support password managers, paste, passkeys) – aligns with NIST guidance; 2.5.8 **Target Size (Minimum) 24×24 CSS px (AA, new)**; **2.4.11 Focus Not Obscured (Minimum) (AA, new)**; 2.5.7 Dragging Movements (alt to drag); 3.2.6 Consistent Help (A, new).
  - Tables: use real `<table>`, `<th scope>`, `<caption>`; 1.4.10 Reflow (320 px) so wide ledger tables need a responsive pattern (scroll container with keyboard access or stacked layout), 1.4.4 resize text.
  - Status messages: **4.1.3 Status Messages (AA)**: success/error/balance-updated messages programmatically exposed via `role="status"` / `aria-live="polite"` (errors `role="alert"`) without moving focus.
  - Also 1.4.3 contrast 4.5:1 (3:1 for large text/UI components 1.4.11), 1.4.1 don't use colour alone (debit/credit, negative balances need sign/text, not only red), 2.4.7 focus visible.
- **EN 301 549**: current legally referenced version is **V3.2.1 (2021)**, which references WCAG 2.1 AA. **V4.1.1** (aligning web requirements to WCAG 2.2 AA) was reportedly adopted 24 Aug 2026 and published 2 Sep 2026, with OJEU citation (which confers presumption of conformity) targeted ~16 Dec 2026 [U: secondary sources — Deque, law-firm note; verify on ETSI/OJEU] [V].
- **European Accessibility Act (Directive (EU) 2019/882)**: applies from **28 June 2025** to specified products/services (e-commerce, consumer banking services, e-books, etc.) provided to consumers; **microenterprises (<10 staff and <€2m turnover) providing services are exempt**. A volunteer-run community currency is likely outside scope or exempt, but a commercial operator/hosting provider serving consumers with payment/banking-like services could be in scope; also public-sector bodies fall under the Web Accessibility Directive (EU) 2016/2102. [K] **This is not legal advice; confirm applicability per deployment.**

### Recommendation for Circula
- Commit to **WCAG 2.2 AA** as the acceptance criterion for all member and admin UI (aligns with upcoming EN 301 549 V4.1.1 so no rework). Publish an accessibility statement and a feedback contact.
- Build-in: semantic HTML first; component library with labelled inputs, inline + summary error messages, `autocomplete` tokens, `role=status` toasts for "payment sent" and balance changes, confirmation/review step before posting a payment (3.3.4), 24 px minimum targets, no drag-only interactions, accessible passkey/password login.
- Ledger tables: caption + `th scope`, sortable headers as buttons with `aria-sort`, amounts right-aligned with explicit sign and units text (e.g. "−12.50 hours"), responsive reflow without information loss.
- CI: axe-core (or Pa11y) in end-to-end tests plus a manual keyboard/screen-reader checklist per release. Because automated tools catch only a subset, schedule a manual audit before 1.0.
- Document EAA applicability as a deployer's responsibility in operator docs.

---

## 6. Privacy engineering

### Facts
- GDPR principles (Art. 5): data minimisation, purpose limitation, storage limitation, integrity/confidentiality; Art. 25 data protection by design and default; Art. 32 security (pseudonymisation and encryption named as measures); Art. 17 right to erasure with exceptions (legal obligation, legal claims, Art. 17(3)); Art. 4(5) defines pseudonymisation; Recital 26: truly anonymised data is out of scope. [K]
- **Erasure vs immutable ledger.** Accepted engineering pattern: keep the ledger *free of direct personal data*. Postings reference an opaque `account_id`; personal data (name, email, address, phone) lives in a separate mutable `member_profile` store keyed by that ID. Erasure = delete/overwrite the profile (crypto-shredding is an option: per-member data key, destroy key), leaving an anonymised account stub in the ledger (display name "Former member #1234") so balances and unit conservation remain intact. Counterparty transaction descriptions are the risk: free text can embed personal data; constrain or scrub on erasure. EDPB guidance on blockchains (Guidelines 02/2025) similarly advises not storing personal data on immutable stores. [K/U on guideline number] Retention duties (e.g. accounting/tax records, typically several years) may legitimately justify retaining transaction records minus identifying profile data; this varies by jurisdiction [U].
- **Pseudonymous audit logs**: log actor as stable pseudonymous ID, not email/name; resolve through the profile store only for authorised investigations; keep IP addresses truncated or hashed with rotating salt and short retention; separate security-log retention from business-ledger retention. [K]
- Data minimisation patterns: collect only what the function needs (a mutual-credit account needs a display name and contact route, not birth date); optional fields default off; field-level visibility controls (other members see only chosen fields); aggregate/limit exports; avoid analytics SDKs that leak to third parties; DPIA for the platform template; data-export (Art. 20 portability) endpoint as JSON/CSV. [K]

### Recommendation for Circula
- **Hard architectural rule**: no direct personal data in `transactions`, `postings`, hash chain or event log. Free-text `memo` field: length-limited, labelled as visible to counterparties and admins, redactable (store memo in a separate redactable table referenced by transaction ID; the hash chain covers a hash of memo, enabling redaction with tombstone while preserving chain verification via stored memo hash, noting that stored hashes of low-entropy text may still be reversible, so salt them per-row).
- Erasure workflow: (1) require zero or settled balance policy defined by each network (a member with non-zero balance is a financial obligation; document the lawful basis for keeping the anonymised account), (2) delete profile PII, (3) tombstone account (`status=closed`, `display_name` anonymised), (4) crypto-shred per-member encrypted fields and backups-by-key-expiry, (5) write an audit event with no PII. Document backup retention so erased data ages out within a stated window.
- Pseudonymous audit log with actor UUID; IP stored truncated (/24 or /48) with 30–90 day retention [retention figures are suggestions, not legal requirements].
- Ship a data-protection template for operators: privacy notice, record of processing, DPIA starter, DPA for hosted deployments. Enforce field-level encryption for contact data at rest in addition to disk encryption.

---

## 7. Operations

### Facts
- **Backups**: untested backups are not backups; verify by regular automated restore into a scratch environment and run integrity checks (ledger invariant `SUM=0`, hash chain verify, row counts). PostgreSQL supports base backup + WAL archiving for **point-in-time recovery (PITR)**; tools: pgBackRest, WAL-G, Barman. [K]
- **RPO/RTO**: RPO = maximum tolerable data loss (time); RTO = maximum tolerable downtime. WAL archiving gives RPO of seconds-minutes; periodic dumps alone give RPO = interval. Define per deployment tier. Follow 3-2-1 backup (3 copies, 2 media, 1 offsite), encrypted, immutable/object-lock for offsite copy. [K]
- **Structured logging**: JSON lines to stdout, with `timestamp`, `level`, `message`, `trace_id`, `span_id`, request ID, actor pseudonym; no secrets/PII (OWASP Logging Cheat Sheet). [K]
- **OpenTelemetry**: CNCF project; traces, metrics, logs signals; **OTLP** protocol; traces and metrics APIs/SDKs are stable in most languages, logs signal GA in spec but SDK maturity varies by language [U: verify per language]. W3C **Trace Context** (Recommendation) for `traceparent` propagation. Semantic conventions for HTTP/DB have been stabilising (HTTP stable since 2023). [K]
- **12-factor** (12factor.net): config in environment variables, strict separation of config from code, logs as event streams, stateless processes, dev/prod parity, disposable processes, admin tasks as one-off processes. Secrets: not in the repo or image; inject via env/secret store/files (e.g. Docker/K8s secrets, SOPS-encrypted files for small deployments); rotate; scan commits (gitleaks). [K]
- **Semantic Versioning 2.0.0**: MAJOR.MINOR.PATCH; for an API product, version the public API contract and the database schema/migration set separately from release tags. Keep a Changelog format. [K]
- **DB migrations, expand/contract (parallel change)**: (1) *expand*: add nullable column/new table/index (`CREATE INDEX CONCURRENTLY` on PostgreSQL), (2) deploy code that writes both and reads old/new, (3) backfill in batches, (4) switch reads, (5) *contract*: drop old column in a later release. Avoid long `ACCESS EXCLUSIVE` locks: set `lock_timeout`; add constraints `NOT VALID` then `VALIDATE`. Migrations forward-only, idempotent, tested against a prod-sized snapshot; tools: Flyway, Liquibase, Atlas, sqitch, or framework-native. [K]

### Recommendation for Circula
- Ship reference deployment with **pgBackRest or WAL-G** (continuous WAL archive + nightly base backup), default targets **RPO ≤ 5 min, RTO ≤ 4 h** for a small community (document as adjustable defaults), and a scripted **restore-verification job** (weekly, automated) that restores to a temp instance, runs ledger invariant checks and hash-chain verification, and reports pass/fail. Include a runbook and a yearly game-day.
- Logging: JSON structured logs with request ID + trace ID; redaction middleware; log security events (login, failed auth, recovery, role change, credit-limit change) to a separate audit stream.
- Instrument with OpenTelemetry (OTLP export, vendor-neutral, off by default for self-hosters; Prometheus-compatible metrics endpoint as the simple option). Key SLIs: transfer latency p95, 5xx rate, serialization/lock-wait counts, reconciliation drift (must be 0).
- Configuration purely via environment variables or mounted secret files; ship `.env.example` without secrets; fail fast on missing config; secret scanning in CI.
- Versioning: SemVer for the software release and OpenAPI `/v1` for the API; the DB schema has its own forward-only migrations. **All migrations must follow expand/contract with N-1 compatibility** (a release can run against the schema of the previous release) so rolling upgrades and rollbacks work; add a CI check running migrations against the previous release's test suite.

---

## 8. Money and decimal handling

### Facts
- Floating point (binary) cannot represent most decimal fractions exactly; never use `float`/`double` for amounts. Options: **integer minor units** (e.g. cents) or fixed-point decimal (`NUMERIC`/`DECIMAL`). Stripe, TigerBeetle, Square all use integer minor units; TigerBeetle uses unsigned 128-bit integers with an application-defined `ledger` (unit) and decimal exponent ("asset scale") convention. [K]
- **ISO 4217** governs national/official currency codes and minor-unit exponents (0, 2, 3, or 4 decimals). It does **not** cover community or complementary currencies (unit names such as "hours" or "LETS credits"); ISO 4217 also reserves `X`-prefixed codes for non-national units (e.g. XAU, XXX test code), but there is no registry for community units, so Circula must define its own unit registry. [K]
- JavaScript `Number` is safe only to 2^53−1; JSON numbers are interoperability-limited, so large integers should be transferred as strings or kept within safe range. PostgreSQL `BIGINT` covers ±9.22×10^18. [K]
- Rounding: common modes are half-up, half-even (banker's), floor, truncate. Rounding applies only when computing derived amounts (fees, demurrage, proportional splits, hour × rate); the allocation problem is solved with "largest remainder" (distribute leftover minor units deterministically) so splits sum exactly. IEEE 754-2019 / most financial standards default to half-even for statistical neutrality; many consumer systems use half-up. [K]
- Time-based units: if a unit is *hours/minutes*, choose the smallest meaningful minor unit (e.g. 1 minute = 1/60 hour is not decimal; store minutes as integers and display hours).

### Recommendation for Circula
- Store amounts as **signed `BIGINT` in minor units** with a per-unit `scale` (number of decimal places, 0–6) and `name`/`symbol` in a `units` table (immutable once any transaction exists). Define the unit's smallest indivisible amount explicitly; for time-denominated units, use minutes (scale 0) or a 1/100-hour scale and document the choice.
- API: represent amounts as JSON **strings of integers** in minor units plus the unit id (`{"amount":"1250","unit":"<uuid>"}`) or, optionally, a decimal string with the scale applied by the server; never JSON floats. Provide display formatting server-side or via a shared library; parse user input as decimal strings, reject more fractional digits than `scale` instead of silently rounding.
- Rounding policy: explicit per operation, default **half-even**, implemented once in a shared pure function with property tests; use largest-remainder for splitting; any fee/demurrage that rounds must book the rounding remainder to a designated system account so the zero-sum invariant is exactly preserved.
- Overflow: use checked arithmetic, reject amounts above a configured maximum; set DB `CHECK (amount <> 0)` on postings and bounds on credit limits.
- Use ISO 4217 codes only if a deployment denominates in a real currency (optional `iso_4217` field); otherwise use Circula's unit identifiers.

---

## Cross-cutting top recommendations
1. Immutable double-entry journal, balances derived with transactional cache + nightly reconciliation; enforce credit limits with atomic conditional UPDATE and ordered locks; mandatory Idempotency-Key.
2. UUIDv7 IDs, integer minor-unit BIGINT amounts, RFC 3339 times, RFC 9457 errors, cursor pagination, OpenAPI 3.1.
3. PII fully outside the ledger so GDPR erasure never touches immutable data.
4. Baselines: ASVS 5.0.0 L2, NIST 800-63B-4, WCAG 2.2 AA, OWASP API Top 10 2023.
5. Treat as drafts, not standards: Idempotency-Key, RateLimit headers, OAuth 2.1; implement behind a thin layer and label in docs.

## Open items / unverified (re-check before external citation)
- IETF draft revision numbers/dates (datatracker not directly fetchable; idempotency -07 expired 2026-04-18 seen; a July 2026 "-latest" unverified; RateLimit -11 May 2026; OAuth 2.1 -15 Mar 2026, -16 dated Sep 2026 unverified).
- EN 301 549 V4.1.1 publication and OJEU citation dates (secondary sources only).
- WebAuthn Level 3 maturity; WCAG 3.0 stage; RFC number for `Deprecation` header; PostgreSQL `uuidv7()` exact version; EDPB blockchain guideline number; any OWASP API Top 10 post-2023 edition; Formance licence/current API.
- NIST 800-63B-4 clause-level numbers (15/8 char minimums, 64 max) are from background knowledge; confirm in the PDF.
- Retention periods (log IP 30–90 days; ledger retention) are suggestions, not legal advice.

## Sources (access date 2026-10-21 unless noted)
Verified this session via search ([V]):
- NIST SP 800-63B-4: https://csrc.nist.gov/pubs/sp/800/63/b/4/final ; https://pages.nist.gov/800-63-4 ; https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-63B-4.pdf
- OWASP ASVS: https://owasp.org/projects/asvs?tab=main
- OpenAPI 3.2.0: https://spec.openapis.org/oas/v3.2 ; https://spec.openapis.org/oas/latest.html
- Idempotency-Key draft: https://datatracker.ietf.org/doc/draft-ietf-httpapi-idempotency-key-header/ ; history: https://datatracker.ietf.org/doc/draft-ietf-httpapi-idempotency-key-header/history/
- RateLimit headers draft: https://datatracker.ietf.org/doc/draft-ietf-httpapi-ratelimit-headers/ ; repo: https://github.com/ietf-wg-httpapi/ratelimit-headers
- OAuth 2.1 draft: https://datatracker.ietf.org/doc/draft-ietf-oauth-v2-1/ ; history: https://datatracker.ietf.org/doc/draft-ietf-oauth-v2-1/history/
- EN 301 549 v4.1.1: https://www.deque.com/blog/en-301-549-v4-1-1-is-final-what-changed-what-it-means-and-what-you-should-do/ ; https://www.dwt.com/insights/2026/09/european-accessibility-act-ict-standards-update (secondary)

Background-knowledge references (not re-fetched; [K]):
- RFC 9562 UUIDs: https://www.rfc-editor.org/rfc/rfc9562
- RFC 9457 Problem Details: https://www.rfc-editor.org/rfc/rfc9457
- RFC 9110 HTTP Semantics: https://www.rfc-editor.org/rfc/rfc9110 ; RFC 3339: https://www.rfc-editor.org/rfc/rfc3339 ; RFC 6585: https://www.rfc-editor.org/rfc/rfc6585 ; RFC 8288: https://www.rfc-editor.org/rfc/rfc8288 ; RFC 8594: https://www.rfc-editor.org/rfc/rfc8594
- RFC 9700 OAuth Security BCP: https://www.rfc-editor.org/rfc/rfc9700 ; RFC 7636 PKCE: https://www.rfc-editor.org/rfc/rfc7636 ; RFC 9449 DPoP: https://www.rfc-editor.org/rfc/rfc9449
- OIDC Core: https://openid.net/specs/openid-connect-core-1_0.html
- WebAuthn: https://www.w3.org/TR/webauthn-3/ ; FIDO passkeys: https://fidoalliance.org/passkeys/
- OWASP API Security Top 10 2023: https://owasp.org/API-Security/editions/2023/en/0x00-header/ ; Cheat Sheets (Session Management, Forgot Password, CSRF, Logging): https://cheatsheetseries.owasp.org/
- WCAG 2.2: https://www.w3.org/TR/WCAG22/ ; What's new: https://www.w3.org/WAI/standards-guidelines/wcag/new-in-22/
- European Accessibility Act: https://eur-lex.europa.eu/eli/dir/2019/882/oj ; Web Accessibility Directive: https://eur-lex.europa.eu/eli/dir/2016/2102/oj
- GDPR: https://eur-lex.europa.eu/eli/reg/2016/679/oj ; EDPB guidelines: https://www.edpb.europa.eu/our-work-tools/general-guidance/guidelines-recommendations-best-practices_en
- TigerBeetle docs: https://docs.tigerbeetle.com/ ; Formance Ledger: https://docs.formance.com/ ; Modern Treasury "How to Scale a Ledger": https://www.moderntreasury.com/journal/how-to-scale-a-ledger-part-i ; Stripe ledger post: https://stripe.com/blog/ledger-stripe-system-for-tracking-and-validating-money-movement ; Square Books: https://developer.squareup.com/blog/books-an-immutable-double-entry-accounting-database-service/
- PostgreSQL isolation: https://www.postgresql.org/docs/current/transaction-iso.html ; PITR: https://www.postgresql.org/docs/current/continuous-archiving.html
- OpenTelemetry: https://opentelemetry.io/docs/ ; W3C Trace Context: https://www.w3.org/TR/trace-context/
- 12-factor: https://12factor.net/ ; SemVer: https://semver.org/ ; ISO 4217: https://www.iso.org/iso-4217-currency-codes.html
