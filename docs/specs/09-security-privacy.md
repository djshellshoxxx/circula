# 09 Security, Privacy and Abuse Resistance

Status: Draft v0.1 · Owner/role: Security reviewer (with Product architect) · Last updated: 2026-10-21 · Scope: assets, actors, trust assumptions, threat model, authentication, authorization, privacy, abuse resistance, secure logging, backups, incident response for the MVP · Dependencies: [00](00-conventions.md) (glossary, LED invariants, DEC-001..004), [03](03-domain-model.md), [05](05-architecture.md), [06](06-data-ledger-audit.md), [07](07-api-contract.md), [08](08-ux-accessibility.md), [10](10-governance-operations.md), [11](11-deployment-operations.md), [12](12-quality-test-plan.md), [13](13-assumptions-risks-decisions.md) · Open questions: listed in section 9 and in [13](13-assumptions-risks-decisions.md)

Normative terms (MUST, SHOULD, MAY) follow [00 section 1](00-conventions.md#1-normative-terms). Ledger rules (LED-1..LED-7) are defined once in [00 section 4](00-conventions.md#4-platform-wide-invariants-not-configurable) and are referenced here, not restated. Requirements in this document use the prefix `SEC`. Threat identifiers `T-nn` are local to this document and are not requirement IDs.

This document does not claim anonymity, cryptographic guarantees for ledger contents, or regulatory compliance. Where a control is a recommended default rather than a settled decision, it is marked as such and tracked in [13](13-assumptions-risks-decisions.md).

---





## Build status (as built)

Snapshot: HEAD `925f637` (branch claude/circula-build). Statuses were first read at `31b1015` and re-checked at `925f637` (see TRACEABILITY.md). **built** = behaviour exists in code; **partial** = exists with the gaps stated, or the required verification was not run; **not built**; **deferred** = out of MVP by this spec or a recorded decision. Code paths and test names: [TRACEABILITY.md](../TRACEABILITY.md). Open owner decisions and resolved questions: [DECISIONS.md](../DECISIONS.md). Status is implementation status, not a compliance claim; no release gate has been run.

Summary: built 8, deferred 1, not built 7, partial 18.

| ID | Priority | Status | Note |
|---|---|---|---|
| SEC-001 | P0 | partial | Argon2id PHC with per-password salt; benchmark for DEC-006 not recorded |
| SEC-002 | P0 | partial | Length 12 enforced; breached-password check not wired (PasswordDenied unset) |
| SEC-003 | P0 | partial | Server sessions, cookie attributes, password change revokes; role-specific timeouts not built |
| SEC-004 | P0 | partial | CSRF tokens on unsafe methods; token bound to random cookie, not to session |
| SEC-005 | P0 | partial | Login and sign-up responses uniform; recovery not built |
| SEC-006 | P0 | partial | Atomic per-login throttle and per-address limiter via trusted proxies; no per-network or invitation limits |
| SEC-007 | P1 | not built | MFA not built |
| SEC-008 | P0 | not built | Account recovery not built |
| SEC-009 | P0 | built | Default deny; authorisation matrix test |
| SEC-010 | P0 | built | Self-approval, conflicts, last-admin and operator-grant refusals |
| SEC-011 | P1 | not built | No dual control; adjustments post at once |
| SEC-012 | P0 | built | Audit in same unit for admin actions; fixed-text in-app notice to the affected member on suspend, reinstate, removal, role change, credit-limit change, adjustment and reversal |
| SEC-013 | P0 | partial | No operator role reads network data in product; break-glass not built |
| SEC-014 | P0 | partial | Minimal profile fields; data inventory document absent |
| SEC-015 | P0 | built | Network-scoped queries and cross-network tests |
| SEC-016 | P0 | partial | Members-only directory; no-index directive absent |
| SEC-017 | P0 | built | Member and admin scopes; CSV formula neutralisation; 24h download expiry |
| SEC-018 | P0 | not built | No pseudonymisation or erasure on departure |
| SEC-019 | P0 | partial | Audit excludes secrets; admin-only read path; member read in service only |
| SEC-020 | P0 | partial | RedactSecrets and route templates in logs; no log-wide canary scan |
| SEC-021 | P1 | partial | `log-retention` setting and `circula prune-logs`; documented in ops monitoring guide |
| SEC-022 | P0 | built | Refuses plain HTTP on non-loopback unless explicit; Secure cookies forced |
| SEC-023 | P0 | built | CSP, frame-ancestors, no-store, same-origin scripts |
| SEC-024 | P0 | built | Safe next parsing; html/template; integer amounts |
| SEC-025 | P1 | not built | Abuse reporting not built |
| SEC-026 | P1 | not built | Velocity and reciprocal signals not built |
| SEC-027 | P0 | partial | Secrets from env or files; rotation procedure not documented for session key |
| SEC-028 | P0 | partial | Backup manifest signed with key; backup not encrypted at rest; files 0600 |
| SEC-029 | P1 | partial | Module versions pinned; govulncheck and SBOM module list in CI; dependency checksum failure test not present |
| SEC-030 | P0 | partial | Incident contacts and failure table in docs/ops; drills not recorded |
| SEC-031 | P1 | built | SECURITY.md with private reporting channel; owner approval of contact pending (DEC-026) |
| SEC-032 | P1 | partial | Grant defaults to zero; cap and open-mode confirmation not built |
| SEC-033 | P1 | partial | Tests cover most P0 SEC items; independent review not performed |
| SEC-034 | P2 | deferred | Tamper-evident option deferred; unkeyed chain already present (DEC-044) |

## 1. Assets

| ID | Asset | Why it matters | Primary home |
|---|---|---|---|
| A-1 | Ledger entries and transaction records | Integrity of balances is the core product promise (LED-1..LED-3) | [06](06-data-ledger-audit.md) |
| A-2 | Member account balances and credit limits | Direct economic effect on members | [03](03-domain-model.md) |
| A-3 | Member credentials (password hashes, MFA secrets, recovery tokens) | Account takeover gives control of balances | This document |
| A-4 | Sessions and API tokens | Bearer capability to act as a member or admin | This document |
| A-5 | Personal data (names, contact details, listings, dispute narratives) | Privacy of members | This document, [06](06-data-ledger-audit.md) |
| A-6 | Audit events | Evidence for disputes and governance; must not leak secrets | [06](06-data-ledger-audit.md) |
| A-7 | Network configuration and governance records | Rules every member relies on | [10](10-governance-operations.md) |
| A-8 | Backups, exports and release artifacts | Copies of A-1..A-7 outside the live system | [11](11-deployment-operations.md) |
| A-9 | Service secrets (session keys, SMTP credentials, backup encryption keys) | Compromise defeats other controls | [11](11-deployment-operations.md) |

## 2. Actors

| Actor | Description | Trust level |
|---|---|---|
| Anonymous visitor | Unauthenticated internet user | Untrusted |
| Member | Authenticated person in one or more networks | Trusted only for their own actions |
| Network administrator | Member with administrative role in one network | Trusted for network rules; bound by audit and dual control where SEC-011 applies |
| Dispute reviewer | Member appointed to review disputes in one network | Trusted for case decisions; bound by recusal |
| Platform operator (staff) | Person operating a deployment (hosted or self-hosted) | Trusted to operate the software; MUST NOT have routine access to network financial data (SEC-013) |
| Malicious member | Member who attempts fraud, collusion or abuse | Untrusted |
| Compromised account | Legitimate account used by an attacker | Untrusted until re-authenticated |
| External attacker | Network-reachable adversary without an account | Untrusted |
| Backup custodian | Anyone with access to backup media | Trusted only for storage, not for reading content (SEC-028) |
| Authority with lawful request | Court, regulator or similar requester | Outside the product trust model; handled by a documented process ([10](10-governance-operations.md)); legal questions for counsel |

## 3. Trust assumptions

- TA-1: Networks are mutually distrusting tenants. A member of network X MUST NOT gain any information about network Y through the product, except information that is publicly served (none in the MVP).
- TA-2: Browsers and API clients are untrusted. Every security decision is made on the server.
- TA-3: The application host, its operating system and its storage are trusted only to the extent the platform operator secures them (see [11](11-deployment-operations.md)). The product does not protect data from a fully compromised host.
- TA-4: TLS is terminated either by the application or by a reverse proxy that the operator configures. The product does not accept responsibility for transport security that the operator disables.
- TA-5: In hosted mode the platform operator can technically read stored data. The product relies on access policy, audit and notification (SEC-013) rather than on cryptographic separation. Client-side or end-to-end encryption is not assumed for the MVP.
- TA-6: Members can lie about their identity and about the value of goods or services. The product cannot verify these claims; governance and disputes exist for this reason ([10](10-governance-operations.md)).
- TA-7: Collusion between members can create false balances that software cannot distinguish from legitimate trade. The product detects some patterns and supports disputes; it does not prevent collusion (see T-19 and RSK entries in [13](13-assumptions-risks-decisions.md)).

## 4. Threat model (STRIDE-style)

The table lists the practical threats for the MVP. Each row names the requirements that mitigate it. A threat with residual risk names it explicitly.

| ID | STRIDE | Threat | Affected assets | Mitigating requirements | Residual risk |
|---|---|---|---|---|---|
| T-01 | Spoofing | Credential stuffing against member logins | A-3, A-4 | SEC-001, SEC-006, SEC-007 | Low with MFA enabled for privileged roles; members may not use MFA |
| T-02 | Spoofing | Account takeover through account recovery | A-3, A-2 | SEC-005, SEC-008 | Medium if recovery channel (DEC-009) is weak |
| T-03 | Spoofing | Member impersonates another member in the directory or in a trade | A-5, A-2 | SEC-016 | Medium; display names are not unique identity |
| T-04 | Tampering | Direct modification or deletion of ledger rows by an operator or attacker with database access | A-1, A-2 | LED-2, SEC-013, SEC-028, [11](11-deployment-operations.md) verification command | Medium; tamper-evidence is optional (SEC-034) |
| T-05 | Tampering | Concurrent postings push an account beyond its credit limit | A-1, A-2 | LED-5, [12](12-quality-test-plan.md) concurrency tests | Low if tests pass |
| T-06 | Tampering | Duplicate or replayed request posts a trade twice | A-1 | LED-4 | Low |
| T-07 | Tampering | Cross-site request forgery triggers a trade or admin action | A-1, A-7 | SEC-004, SEC-023 | Low |
| T-08 | Repudiation | Administrator denies making an adjustment or policy change | A-6, A-7 | SEC-011, SEC-012, LED-6 | Low if audit is complete |
| T-09 | Repudiation | Member denies accepting a trade | A-1, A-6 | LED-6, [06](06-data-ledger-audit.md) acceptance record | Low |
| T-10 | Information disclosure | Broken object-level authorization exposes another member's balance or history | A-2, A-5 | SEC-009, SEC-015, SEC-016 | Low if authorization matrix tests pass |
| T-11 | Information disclosure | Secrets, tokens or personal data leak into logs, metrics, backups or error pages | A-3, A-4, A-5, A-8 | SEC-020, SEC-021, SEC-027, SEC-028, [11](11-deployment-operations.md) metrics rules | Medium |
| T-12 | Information disclosure | Export contains other networks' data, excess personal data, or spreadsheet formula injection | A-5, A-1 | SEC-015, SEC-017 | Low if export tests pass |
| T-13 | Denial of service | Login, recovery or proposal flooding | A-4, availability | SEC-006, SEC-026, [11](11-deployment-operations.md) request limits | Medium |
| T-14 | Denial of service | Disk exhaustion or write-lock contention on the embedded database | Availability, A-1 | [11](11-deployment-operations.md) failure recovery; [05](05-architecture.md) | Medium |
| T-15 | Elevation of privilege | Member changes a request parameter to act as an administrator or reviewer | A-7, A-2 | SEC-009, SEC-010 | Low |
| T-16 | Elevation of privilege | Administrator misuses power: self-dealing adjustments, removing dissenting members, changing rules to favour self | A-2, A-7 | SEC-010, SEC-011, SEC-012, [10](10-governance-operations.md) | Medium; governance and social controls |
| T-17 | Elevation of privilege | Platform operator reads or alters a network's data without authorization | A-1, A-5 | SEC-013, SEC-019, SEC-028, DEC-014 | Medium in hosted mode (TA-5) |
| T-18 | Abuse | Sybil sign-ups farm sign-up grants or join open networks | A-2 | SEC-032, DEC-011, [10](10-governance-operations.md) join modes | Medium |
| T-19 | Abuse | Collusive trades (wash trades) create false balances | A-1, A-2 | SEC-026, dispute process ([10](10-governance-operations.md)) | High; not preventable by software (TA-7) |
| T-20 | Abuse | Fraudulent or non-delivered goods and services | A-2, A-5 | SEC-025, dispute process ([10](10-governance-operations.md)) | Medium; outcome is governance, not software |
| T-21 | Abuse | Stored cross-site scripting through listing text or dispute narratives | A-5, A-3 | SEC-023, SEC-024 | Low |
| T-22 | Supply chain | Compromised dependency or tampered release artifact | All | SEC-029, DEC-003 checksums, DEC-027 (signing, deferred) | Medium |
| T-23 | Physical or host | Stolen backup media or disk image | A-1, A-3, A-5, A-8 | SEC-027, SEC-028 | Medium without encryption at rest |

---

## 5. Authentication

**SEC-001** · P0 · Password storage
- Requirement: Member and administrator passwords MUST be stored only as salted one-way hashes produced by Argon2id (preferred) or bcrypt. The hash string MUST record the algorithm and its parameters so that they can be upgraded later. Passwords MUST NOT be stored, logged, or returned in any response.
- Rationale: Password database theft is a realistic threat (T-23). A memory-hard function raises the cost of offline guessing. Recording parameters makes upgrades possible without forcing a mass reset.
- Acceptance criteria:
  - A test inserts a known password and verifies that the stored value is not equal to and does not contain the plaintext.
  - The stored value identifies the algorithm and its parameters; a test with a non-default parameter set verifies correctly.
  - Parameters are chosen by a benchmark on the minimum supported hardware and recorded as part of DEC-006 in [13](13-assumptions-risks-decisions.md) before release. Until then the release gate is not passed.
  - A search of the code base, logs and API responses for a seeded canary password finds zero occurrences.

**SEC-002** · P0 · Password policy
- Requirement: A password MUST be at least the minimum length set by DEC-006 (recommended default: 12 characters) and MUST accept at least 64 characters. The product MUST NOT require composition rules (mandatory symbols or digits) or periodic forced rotation. Passwords that appear in a breached-password list named by DEC-006 MUST be rejected when that check is enabled.
- Rationale: Length is the primary strength factor. Composition rules and forced rotation add friction without a proven security benefit. Breached-password checks reduce reuse risk.
- Acceptance criteria:
  - An 11-character password is rejected; a 12-character password is accepted (under the recommended default).
  - A 64-character password is accepted and verifies correctly.
  - No validation message requires a specific character class.
  - When the breached-password check is enabled, a seeded listed password is rejected with a message that does not reveal other account information.

**SEC-003** · P0 · Sessions
- Requirement: After successful authentication the server MUST issue a session identifier with at least 128 bits of randomness from a cryptographically secure source. Sessions MUST be server-side and revocable. The session cookie MUST be marked HttpOnly, MUST be marked Secure whenever the deployment is served over HTTPS, and MUST use a SameSite value that does not disable CSRF protection on its own (see SEC-004). The session identifier MUST be regenerated at login and at any privilege change. Sessions MUST be invalidated on logout, on password change, on account recovery, and on role removal. Idle and absolute timeouts apply as set by DEC-005.
- Rationale: Session fixation and stolen-token replay are common account takeover paths (T-01, T-02).
- Acceptance criteria:
  - The session identifier value differs before and after login (test).
  - A session cookie captured before logout is rejected after logout (test).
  - After password change, a session from another browser is rejected (test).
  - After a role is removed, that role's privileged routes return an authorization error on the next request (test).
  - Cookie attributes in a login response include HttpOnly and SameSite; Secure is present in HTTPS-configured tests (test).
  - A session idle for longer than the configured idle timeout is rejected (test with an injected clock).

**SEC-004** · P0 · CSRF protection
- Requirement: Every state-changing request that is authenticated by a cookie (form posts from the server-rendered interface and any JSON request that uses cookie authentication) MUST carry and verify an anti-CSRF token bound to the session. Requests without a valid token MUST be rejected before any state change. Read-only requests MUST NOT change state.
- Rationale: The product posts financial state changes from a browser (T-07). Relying on SameSite alone is not sufficient for this product.
- Acceptance criteria:
  - A posted trade proposal without a token is rejected with no ledger, transaction or audit change (integration test).
  - A token from one session is rejected in another session (test).
  - A cross-origin form post from a test page is rejected (end-to-end test).
  - No GET route changes ledger, membership, role or configuration state (route inventory test).

**SEC-005** · P0 · Uniform responses and anti-enumeration
- Requirement: Login failures, recovery requests, invitation checks and sign-up responses MUST NOT reveal whether an account, email address, network or invitation exists except to an authorized party. Response bodies and response timing SHOULD NOT differ materially between existing and non-existing accounts.
- Rationale: Account enumeration enables targeted attacks (T-01, T-02).
- Acceptance criteria:
  - For a known and an unknown email address, login and recovery responses have the same status code and body (test).
  - Median response time difference between existing and non-existing accounts for login is below the threshold recorded under QA-040 in [12](12-quality-test-plan.md) (test; threshold pending DEC-023).
  - A public invitation link for an unknown or expired token returns the same generic message as for a used token.

**SEC-006** · P0 · Rate limiting and backoff for authentication
- Requirement: The product MUST rate-limit login attempts per account, per client address and globally per network, and MUST rate-limit account recovery requests and invitation redemption. Exceeding a limit MUST return a throttled response and MUST NOT lock out the legitimate owner permanently. Recommended default: after five consecutive failed logins for one account, require increasing delays of at least 15 minutes before further attempts, reset on success. Limits are configurable by the platform operator within bounds in [11](11-deployment-operations.md).
- Rationale: Credential stuffing and guessing are T-01 and T-13. Permanent lockout would let an attacker deny service to a member.
- Acceptance criteria:
  - The sixth failed login for one account within the window is throttled (test with injected clock).
  - Successful login after the delay resets the failure counter (test).
  - Throttling by client address does not block a different client address with valid credentials for the same account beyond the account-level rule (test).
  - Throttled responses use a standard retry-after indication (API test).
  - An account is never disabled permanently by failed attempts alone (test).

**SEC-007** · P1 · Multi-factor authentication for privileged roles
- Requirement: Network administrators and dispute reviewers MUST be able to enable a second factor (time-based one-time passwords or an equivalent standard), and the platform MUST support it for members. Whether it is mandatory for privileged roles is decided by DEC-007 (recommended default: mandatory for network administrators and platform operators, optional for members and dispute reviewers). Every second-factor method offered MUST have at least one option that does not require transcribing or solving a cognitive challenge, as [08](08-ux-accessibility.md) UX-053 requires. Recovery codes MUST be single use and stored hashed.
- Rationale: Administrator accounts control network rules and adjustments (T-16); password-only protection is weak for them.
- Acceptance criteria:
  - An administrator with mandatory MFA cannot complete an administrative action without a valid second factor (test).
  - A used recovery code is rejected on reuse (test).
  - Second-factor secrets are not retrievable through any API or interface after enrollment (test).
  - Enrollment and removal of a second factor each write an audit event (test).

**SEC-008** · P0 · Account recovery
- Requirement: Account recovery MUST use a verified contact channel defined by DEC-009. Recovery tokens MUST be random, single use, stored hashed, and valid for a short period (recommended default: 60 minutes). Completing recovery MUST invalidate all existing sessions and MUST send a notice to the previously verified contact. When no verified contact channel exists for a member, recovery MUST be administrator-assisted and require the dual-control approval in SEC-011 for the network. A platform operator MUST NOT reset a member password directly except under the break-glass rule in SEC-013.
- Rationale: Recovery is a common takeover path (T-02). Notices let a legitimate owner detect misuse.
- Acceptance criteria:
  - A recovery token is rejected after first use, after expiry, and when altered (tests).
  - Recovery completion invalidates all prior sessions for the account (test).
  - A notice is queued to the previous contact on recovery completion (test).
  - An administrator-assisted recovery without a second approver is refused (test).
  - No operator or administrator interface displays a recovery token or a new password in plain text after issue (test).

**SEC-009** · P0 · Server-side authorization on every request
- Requirement: Every API route and every interface route MUST enforce authorization on the server. The default MUST be deny. Each decision MUST evaluate the authenticated principal, the role in the target network, the membership status of the principal, and the ownership of the target object (for example, a transaction must involve the principal before its details are shown). The client MUST NOT be trusted to supply role or network identity without verification against membership.
- Rationale: Broken object-level and function-level authorization is the most likely disclosure and elevation path (T-10, T-15).
- Acceptance criteria:
  - An authorization matrix test covers every route, each role, each membership state (member, departed, suspended, not a member) and each network; expected result is recorded per cell and any mismatch fails the build (test; see [12](12-quality-test-plan.md)).
  - A member changing a transaction identifier in a request to another member's transaction receives the same not-found response as for a missing identifier (test).
  - A route with no authorization rule fails the route inventory test (test).

**SEC-010** · P0 · Role separation
- Requirement: The following MUST hold: a network administrator MUST NOT approve their own adjustment, write-off or role grant; a dispute reviewer MUST NOT decide a dispute in which they are a party or have a recorded conflict; a member MUST NOT hold the administrator role in a network they have departed; the last active administrator of a network MUST NOT be removed or demoted without a successor, and the product MUST refuse the action when no successor exists (see [10](10-governance-operations.md)); platform operator staff roles MUST NOT grant themselves a network role.
- Rationale: Self-approval and conflicts are the main administrative abuse patterns (T-16).
- Acceptance criteria:
  - Self-approval of an adjustment is refused (test).
  - A reviewer assigned to a dispute where they are a party cannot record a decision (test).
  - Removing the last active administrator is refused (test).
  - A platform operator account cannot add itself to a network role through any route (authorization matrix test).

**SEC-011** · P1 · Dual control for value-moving administrative actions
- Requirement: Adjustments, write-offs and any administrative action that changes a member's effective credit position MUST require approval by a second authorized administrator when the network has two or more administrators, as set by DEC-013. Approval MUST be recorded as a separate audit event linked to the original request. A request MUST expire if not approved within a defined period.
- Rationale: Single-person control of balances is a high-impact repudiation and misuse risk (T-08, T-16).
- Acceptance criteria:
  - An adjustment is not posted until a second approver records approval (test).
  - The requester cannot be the approver (SEC-010; test).
  - An unapproved request past its expiry is closed with an audit event and posts nothing (test).
  - Networks with a single administrator follow the rule in DEC-013 and display the restriction to members in the change log (test).

**SEC-012** · P0 · Administrative action audit and notice
- Requirement: Every administrative action (role change, configuration change, membership decision, adjustment, export, suspension, closure, break-glass access) MUST write an audit event in the same atomic unit (LED-6) with actor, target, time, reason where required, and a before and after summary. Actions that change a member's rights, obligations or balance MUST produce a notice to the affected member.
- Rationale: Transparency deters misuse and supports disputes (T-08, T-16).
- Acceptance criteria:
  - Each administrative route, when invoked, produces exactly one audit event of the correct type (test).
  - A failure injected after the business change and before audit commit leaves no business change (atomicity test).
  - A removal of a member produces a notice to that member (test).

**SEC-013** · P0 · Platform operator access
- Requirement: Platform operator staff MUST NOT have routine access to network financial data, member personal data or dispute narratives through any product interface. Operator access to these data for support, incident response or repair MUST use a break-glass procedure that: (a) requires a documented reason and a second staff approver, (b) is recorded in an audit trail that network administrators can read, (c) notifies affected network administrators within the period set by DEC-014 unless legally prohibited, and (d) is limited to the minimum data needed. Hosted operators MUST publish their access policy. The specific technical controls are decided by DEC-014.
- Rationale: TA-5 means the operator can technically read data. Policy, audit and notice are the MVP control.
- Acceptance criteria:
  - No platform-operator role can view balances, transaction details or dispute text in the product UI or API without a break-glass record (authorization matrix test).
  - A break-glass session produces an audit event visible to the affected network administrators (test).
  - The published operator access policy exists and is linked from the operator documentation before hosted launch (release gate, [12](12-quality-test-plan.md)).

**SEC-014** · P0 · Data minimization
- Requirement: The product MUST collect only the personal data fields listed in [06](06-data-ledger-audit.md) for the MVP. It MUST NOT require national identity numbers, government identifiers, bank account numbers, card data or date of birth. Optional profile fields MUST be marked as optional in the interface and API. Free-text fields MUST have length limits.
- Rationale: Minimal collection reduces breach impact and privacy obligations (counsel question in [13](13-assumptions-risks-decisions.md)).
- Acceptance criteria:
  - The registration and profile schema contains no field from the forbidden list (schema test).
  - Every collected field has an entry in the data inventory in [06](06-data-ledger-audit.md) (traceability check).
  - Submitting an over-length free-text value is rejected with a field-level error (test).

**SEC-015** · P0 · Tenant isolation between networks
- Requirement: Every query, cache key, search, export, notification and API response MUST be scoped to exactly one network, derived from the authorized context and never from unvalidated input alone. No record, count, name or identifier from one network MUST appear in another network's responses. Hosted deployments MUST apply the storage isolation model set by DEC-008.
- Rationale: Networks are tenants (TA-1). Cross-tenant leakage is a severe failure.
- Acceptance criteria:
  - A seeded two-network dataset is queried by a member of each network; no record of the other network appears in any response, export or log line (cross-tenant test).
  - Changing a network identifier in any request is refused unless the principal is a member of that network (authorization matrix test).
  - Counts in directory and admin views include only the current network (test).

**SEC-016** · P0 · Directory visibility and identity display
- Requirement: The member directory MUST default to members-only visibility within a network, as set by [00 section 5](00-conventions.md#5-configurable-network-policies-defaults). Member pages MUST NOT be indexed by search engines (robots directives on member and transaction pages). A member MUST see a counterparty's display name and the information needed to complete a trade only after a proposal or acceptance involving them. Display names are not unique identity; the interface MUST show a stable member identifier or handle in confirmation steps.
- Rationale: Directory exposure and display-name impersonation are T-03 and privacy risks.
- Acceptance criteria:
  - An unauthenticated request for a directory or member page returns not-found or sign-in (test).
  - Response headers or markup on member and transaction pages include a no-index directive (test).
  - A member without an involving transaction cannot retrieve another member's contact details (authorization test).
  - Trade confirmation screens show the handle (test; see [08](08-ux-accessibility.md)).

**SEC-017** · P0 · Export controls
- Requirement: Exports MUST be scoped to a single network, MUST be generated only for authorized principals, MUST be logged as audit events (SEC-012), MUST exclude credentials, tokens, password hashes, MFA secrets and recovery material, and MUST be delivered only through an access-controlled download that expires (recommended default: 24 hours). Member self-exports MUST contain only that member's own data and the transactions in which they are a party. Spreadsheet-format exports MUST neutralise cells that begin with formula-triggering characters. Export formats and contents are defined in [07](07-api-contract.md) and [10](10-governance-operations.md).
- Rationale: Exports are a common over-disclosure path (T-12).
- Acceptance criteria:
  - An export for network X contains zero records of network Y (test).
  - An export file contains none of the seeded canary credentials (test).
  - A CSV cell seeded as a formula string is written with neutralisation (test).
  - A download link is refused after expiry (test).
  - A member's self-export contains no transaction in which the member is not a party (test).

**SEC-018** · P0 · Retention and erasure
- Requirement: Personal data that is not needed for ledger integrity or legal retention MUST be deleted or irreversibly pseudonymised when a member's retention period ends or when an erasure request is approved under the process in [10](10-governance-operations.md). Ledger entries and transaction records are append-only (LED-2) and MUST be retained with the member reference replaced by an opaque account reference. Retention periods are set by DEC-020, with recommended defaults in [06](06-data-ledger-audit.md) section 9, and are reviewed by counsel (RSK entries in [13](13-assumptions-risks-decisions.md)).
- Rationale: Erasure obligations and append-only ledger integrity conflict. The product must make the trade-off explicit rather than hide it.
- Acceptance criteria:
  - After pseudonymisation, a search for the member's name, email and phone in the database, exports and audit events finds zero matches in personal-data fields (test).
  - Ledger totals and balances are unchanged after pseudonymisation (test).
  - Erasure of a member with an open dispute is deferred and the deferral is recorded (test).

**SEC-019** · P0 · Audit event access and content
- Requirement: Audit events MUST NOT contain passwords, tokens, MFA secrets, recovery material or full free-text narratives beyond a reference to the dispute record. Audit events MUST be readable only by network administrators and dispute reviewers of the same network for events that concern that network. A member MAY see audit events that concern their own transactions and accounts, with actor identities reduced as [06](06-data-ledger-audit.md) DAT-033 requires. Platform operators MAY read audit events only through break-glass (SEC-013). Audit events MUST NOT be editable or deletable through any product interface.
- Rationale: Audit is needed for repudiation controls (T-08, T-09) but must not become a secrets store.
- Acceptance criteria:
  - Seeded canaries (password, token, recovery code) do not appear in audit event storage or views (test).
  - No route can update or delete an audit event (route inventory and database permission test).
  - A member without an administrative role cannot read the audit trail of the network (authorization test).

**SEC-020** · P0 · Secure logging
- Requirement: Application logs MUST NOT contain passwords, session identifiers, CSRF tokens, recovery tokens, API tokens, MFA secrets, or secret service configuration. Email addresses and phone numbers MUST be masked or omitted. Monetary amounts and balances SHOULD NOT appear in application logs; they belong in audit events. Log entries MUST include a request identifier, timestamp (UTC), severity and event name. User-controlled strings MUST be encoded so that newlines and control characters cannot forge log entries.
- Rationale: Logs are widely copied and rarely protected (T-11).
- Acceptance criteria:
  - Seeded canary values for each forbidden category are sent through login, recovery, trade and export flows; a scan of all log output finds zero occurrences (test; see [12](12-quality-test-plan.md)).
  - A display name containing a newline and a fake log line produces a single escaped log entry (test).
  - Every log entry has the required fields (schema test).

**SEC-021** · P1 · Log retention and access
- Requirement: Application logs MUST have a defined retention period (recommended default: 90 days, DEC-019) and access restricted to platform operator staff who need them. Log storage MUST be included in backup and deletion procedures.
- Rationale: Logs grow without bounds and become a data liability.
- Acceptance criteria:
  - Log rotation or pruning removes entries older than the configured retention in a test run (test).
  - The retention setting is documented in [11](11-deployment-operations.md) (documentation check).

**SEC-022** · P0 · Transport security
- Requirement: Hosted deployments MUST be served only over HTTPS. Self-hosted deployments MUST serve session cookies only over HTTPS, except on loopback addresses for local development, which MUST be labelled as such in the operator documentation. The application MUST refuse to start in production mode when it is configured to serve session cookies over plain HTTP outside loopback, unless the operator sets an explicit override that is logged as a warning.
- Rationale: Cookies sent over plain HTTP expose sessions (T-01).
- Acceptance criteria:
  - Plain HTTP requests to a hosted deployment are redirected or refused (test).
  - Start-up with the insecure configuration fails with a clear message unless the override is set (configuration test; see [11](11-deployment-operations.md)).

**SEC-023** · P0 · Browser security headers
- Requirement: Server-rendered responses MUST include a content security policy that disallows inline script and unlisted external sources, a frame-ancestors directive that forbids framing, a content-type no-sniff directive, and a referrer policy that does not leak member identifiers or tokens to third parties. The product MUST NOT load third-party scripts or fonts in MVP. Progressive enhancement scripts MUST be served from the same origin.
- Rationale: CSP and framing controls reduce cross-site scripting and clickjacking impact (T-07, T-21). [00 section 6](00-conventions.md#6-shared-technical-decisions-recommended-defaults-see-dec-log) requires the core flows to work without JavaScript, which keeps the policy strict.
- Acceptance criteria:
  - Every HTML response includes the required headers (integration test).
  - Core flows (registration, trade proposal, acceptance, balance view) complete with scripting disabled (end-to-end test).
  - A seeded inline script in a listing description does not execute in a browser test (end-to-end test).

**SEC-024** · P0 · Output encoding and input validation
- Requirement: All user-supplied text MUST be encoded for its output context. Amounts MUST be accepted only as integer minor units within the bounds set by the network and LED-7; floating-point input MUST be rejected with a field-level error. Identifiers MUST be validated for format before database access. Maximum lengths MUST be enforced on the server for every text field.
- Rationale: Injection and money-handling errors are both high impact (T-21, LED-7).
- Acceptance criteria:
  - An amount of "1.5" in an input is rejected if decimals are zero or if the value is not an integer in minor units (test).
  - A listing description containing markup renders as text (test).
  - An identifier with injection characters is rejected before any database call (test).
  - Server-side length limits are enforced even if client-side limits are removed (test).

**SEC-025** · P1 · Abuse reporting
- Requirement: Members MUST be able to report a transaction, listing or member. A report MUST create a record visible only to the network's administrators and dispute reviewers, MUST NOT reveal the reporter's identity to the reported party, and MUST be linked to any related dispute. False or abusive reports are handled under the governance rules in [10](10-governance-operations.md).
- Rationale: Fraud and abuse need a channel that does not depend on the platform operator (T-20).
- Acceptance criteria:
  - A report can be created on a transaction the reporter is party to (test).
  - The reported member's views and notices do not show the reporter's identity (test).
  - A report on an unrelated transaction is refused (authorization test).

**SEC-026** · P1 · Basic velocity and pattern signals
- Requirement: The product MUST raise a signal to network administrators and dispute reviewers when a configurable velocity threshold is exceeded (for example, many proposals from one member in a short period) and when a reciprocal trading pattern between the same two members exceeds a configurable count in a configurable window. Signals MUST NOT block a trade by themselves. Thresholds are network-configurable with recommended defaults set in [12](12-quality-test-plan.md). Advanced fraud analytics is deferred (P2 in [13](13-assumptions-risks-decisions.md)).
- Rationale: Collusion cannot be prevented in software (TA-7) but can be surfaced for review (T-19).
- Acceptance criteria:
  - A seeded burst above threshold creates one signal per window (test with injected clock).
  - A reciprocal pattern above threshold creates a signal and does not change any ledger entry (test).
  - Signals are visible only to authorized roles in the same network (authorization test).

**SEC-027** · P0 · Secrets management
- Requirement: Service secrets (session signing or encryption keys, SMTP credentials, backup encryption keys, any API keys) MUST be supplied at run time through environment variables or a secret file referenced by configuration, MUST NOT be committed to the repository, MUST NOT appear in logs, configuration dumps, error pages, metrics or the administrative interface, and MUST have a documented rotation procedure. Rotating a session key MUST invalidate existing sessions and MUST NOT alter ledger data. See [11](11-deployment-operations.md).
- Rationale: Secrets in source or logs are a frequent cause of compromise (T-11, T-23).
- Acceptance criteria:
  - A repository secret scan finds no secret pattern in committed files (CI check; [12](12-quality-test-plan.md)).
  - A configuration dump command redacts secret values (test).
  - Session key rotation test: sessions are invalidated and ledger checksums are unchanged (test).

**SEC-028** · P0 · Backup security
- Requirement: Backups MUST be encrypted at rest with a key that is not stored alongside the backup. Backup access MUST be limited to named platform operator roles. Backups MUST be included in SEC-018 deletion processing within the backup retention period, and the retention period MUST be stated to members in the privacy information (DEC-020). Restored data MUST pass the ledger verification defined in [11](11-deployment-operations.md) before the restored instance accepts postings. Backup content is governed by the same access rules as live data (SEC-013).
- Rationale: Backups copy every asset (T-23). Erasure can be defeated by long-lived backups unless retention is stated.
- Acceptance criteria:
  - A backup file cannot be read without the key (test).
  - Backup key and backup file are stored in separate locations in the documented procedure (documentation and operator checklist).
  - A restore drill fails and blocks posting when a seeded ledger inconsistency is present in the backup (test; [12](12-quality-test-plan.md)).

**SEC-029** · P1 · Dependency and supply-chain hygiene
- Requirement: The build MUST pin dependency versions through the repository's lock mechanism and verify module checksums. CI MUST scan dependencies for known vulnerabilities and fail the build on findings above a severity threshold set in [12](12-quality-test-plan.md). Each release MUST publish a software bill of materials and SHA-256 checksums for every artifact listed in DEC-003. The product MUST NOT depend on third-party content delivery at run time.
- Rationale: Supply-chain compromise is a practical threat (T-22).
- Acceptance criteria:
  - A build with a modified dependency checksum fails (test).
  - A seeded known-vulnerable dependency at or above the threshold fails the CI job (test).
  - Release checklist includes the bill of materials and checksums for each target platform (release gate).

**SEC-030** · P0 · Incident response
- Requirement: The platform operator MUST maintain a written incident response procedure that defines severity levels, roles, detection sources, containment steps (including freezing postings for an affected network), evidence preservation, notification of affected network administrators, notification of members through network administrators, recovery verification, and a post-incident review. The notification deadline for affected parties is set by DEC-014 and counsel review (RSK in [13](13-assumptions-risks-decisions.md)). Incident drills MUST be run before hosted launch and at least once per year.
- Rationale: Response quality determines damage in a breach or ledger incident. The notification duty is a legal question, not asserted here.
- Acceptance criteria:
  - The procedure document exists and contains each listed element (documentation check).
  - A tabletop drill is recorded with date, participants and findings before hosted launch (release gate).
  - A simulated ledger verification failure freezes postings for the affected network only and records an audit event (test in staging).

**SEC-031** · P1 · Vulnerability disclosure
- Requirement: The repository MUST publish a security policy describing how to report a vulnerability, the expected response time (recommended default: acknowledgement within 5 business days), and what is in scope. The contact channel is decided by DEC-026 and MUST NOT be a personal address published without the owner's approval.
- Rationale: Reporters need a path that is not public disclosure (T-22). The repository currently has no LICENSE or contact file (see section 9).
- Acceptance criteria:
  - A security policy file exists at the repository root before the first public release (release gate).
  - The policy names a contact channel approved by the owner (documentation check).

**SEC-032** · P1 · Sign-up grant and Sybil controls
- Requirement: When sign-up grants are enabled (DEC-011), the product MUST default the grant to zero, MUST cap the grant amount per network, MUST restrict grant eligibility to members who passed the join mode required by the network (approval or invite), and MUST record each grant as an adjustment-class audit event linked to the approving process. Open join mode MUST NOT be combined with an enabled grant unless the network administrator confirms the risk in an audited action.
- Rationale: Free balances attract Sybil accounts (T-18).
- Acceptance criteria:
  - The default network configuration has grant zero (test).
  - Enabling a grant in open join mode without the confirmation step is refused (test).
  - A grant above the cap is refused (test).

**SEC-033** · P1 · Security test obligations
- Requirement: Each P0 SEC requirement MUST have at least one automated test or a documented inspection procedure, traceable in [12](12-quality-test-plan.md). Security tests MUST run in the merge gate and before every release. Penetration testing or independent security review MUST be performed before hosted launch; scope and date are recorded in [12](12-quality-test-plan.md).
- Rationale: Security claims without evidence are not accepted under the project rules.
- Acceptance criteria:
  - Traceability matrix shows a test or inspection for each P0 SEC requirement (documentation check).
  - Independent review report is recorded before hosted launch (release gate).

**SEC-034** · P2 · Tamper-evident audit chain (deferred option)
- Requirement: The product MAY, after MVP, link audit events of each network with a cryptographic chain so that alteration or removal is detectable. Decision DEC-015 records the choice. The MVP data model MUST NOT preclude adding this later.
- Rationale: Tamper evidence strengthens T-04 mitigation but adds complexity not needed for MVP.
- Acceptance criteria:
  - The DAT design review confirms that audit events carry a stable sequence per network (documentation check against [06](06-data-ledger-audit.md)).

---

## 6. Privacy statement at requirements level

- The product MUST present member-facing privacy information that lists the categories of personal data collected, the purposes, the network administrators and platform operators who can see them, the export and deletion options, and the retention periods from DEC-020. The text itself is drafted for counsel review and is not a compliance claim.
- The product MUST NOT use personal data for purposes outside running the network, unless a separate specification approves the purpose.
- Credit balances are accounting records and MUST NOT be presented as personal financial data for analytics or advertising.

## 7. Secure logging and audit boundary

Application logs (operational diagnostics, SEC-020) and audit events (business record, [06](06-data-ledger-audit.md), SEC-012, SEC-019) are different stores with different retention and access rules. An operational failure MUST NOT drop an audit event, and an application log MUST NOT be the only record of a business change.

## 8. Incident response summary

The full procedure is SEC-030. In summary: detect (monitoring in [11](11-deployment-operations.md), ledger verification, abuse reports), contain (freeze affected network postings, revoke sessions, rotate secrets), preserve evidence (audit and log snapshot before recovery), notify (affected network administrators first, then members through them, as set by DEC-014 and counsel), recover (restore from verified backup or correct through new ledger transactions, never by editing entries), and review (blameless post-incident review recorded in the governance log).

## 9. Open questions

1. Password hash parameters and breached-password data source (DEC-006).
2. Session timeout values (DEC-005) and whether members and administrators share timeouts.
3. Mandatory MFA scope (DEC-007).
4. Hosted storage isolation model (DEC-008).
5. Recovery channel when no verified email exists (DEC-009).
6. Platform operator break-glass technical controls and notice period (DEC-014).
7. Retention periods for logs, backups and personal data (DEC-019, DEC-020); counsel review of the erasure versus append-only ledger trade-off (RSK entries).
8. Tamper-evident audit chain timing (DEC-015). As built in part: unkeyed chain and head (DEC-044); keyed chain still open under DEC-015.
9. Vulnerability disclosure contact (DEC-026).
10. Whether the threshold for velocity signals (SEC-026) is a product decision or a test-engineering default.
11. Relationship to [05](05-architecture.md) and [06](06-data-ledger-audit.md) identifiers once those specs are finalised: this document references them by file and prefix, and cross-ID bindings will be checked at integration.
