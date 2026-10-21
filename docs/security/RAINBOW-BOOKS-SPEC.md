# Rainbow Series security requirements for Circula (RB-series)

Status: draft for review. Branch `claude/cc-r1`. Docs only.

## 1. Purpose

Derive a numbered, testable set of security requirements for Circula from the NSA/NCSC "Rainbow Series" (the DoD Trusted Computer System Evaluation Criteria and its interpretations), then map each to Circula's current code. The goal is to maximise security of the mutual-credit ledger, its JSON API and its web UI. Each requirement names its source book, the Circula status, the recommended change, a priority, and a verification test.

Scope: Go single binary, SQLite (WAL, `BEGIN IMMEDIATE` write gate), hash-chained append-only audit, per-network roles, web UI and JSON API.

Out of scope: formal TCSEC or Common Criteria evaluation. No Circula component is being certified.

## 2. Important caveats

- The Rainbow Series is historical. The TCSEC (DoD 5200.28-STD, 1985) was superseded by the Common Criteria (ISO/IEC 15408). Its classes (C1 to A1) are not a deployment target for Circula. We use the Rainbow books only as a checklist of security concerns that remain valid. The NIST catalogue describes the series as outdated and kept for historical reference.
- Source access in this pass. Outbound fetches to `irp.fas.org`, `en.wikipedia.org`, `apps.dtic.mil` and `csrc.nist.gov` were refused by the egress policy (403 CONNECT). Book content below comes from search-result snippets and titles, not from full-text reads. Anything not confirmed from a full text is marked **UNVERIFIED**.
- No section numbers are cited for the Rainbow books. Requirements cite the book and the feature category as named in the book's title or search snippet. Section-level citations need a full-text pass.
- Circula status was checked by grep and file reads in this worktree (commit `6422f87`). Files are cited by path. "Built" means present in code; it does not mean tested to the level in the verification column.

## 3. Source register

| Ref | Book (colour) | Identifier (per source) | Used for | Source | Verification |
|---|---|---|---|---|---|
| TCSEC | Orange Book | DoD 5200.28-STD, 1985 | Access control, object reuse, I&A, audit, trusted path, testing, covert channels, design docs, trusted recovery | https://apps.dtic.mil/sti/tr/pdf/ADA207905.pdf | Title and date confirmed in search snippet. Full text UNVERIFIED (not fetched). |
| Neon Orange | DAC guide | NCSC-TG-003, 30 Sep 1987 | RB-001 discretionary access control | Rainbow index per search snippet, https://en.wikipedia.org/wiki/Rainbow_Series (not fetched) | Identifier from snippet. UNVERIFIED. |
| Red | Trusted Network Interpretation | NCSC-TG-005, 31 Jul 1987 | Network I&A, network integrity, trusted path, network audit | https://irp.fas.org/nsa/rainbow/tg005.htm and https://apps.dtic.mil/sti/tr/pdf/ADA255422.pdf | Identifier and date confirmed in snippet. Foreword wording confirmed in snippet. Body UNVERIFIED. |
| Red (companion) | Environments guideline | NCSC-TG-011, 1990 | Network environment | Search snippet only | UNVERIFIED. |
| Tan | Audit in trusted systems | NCSC-TG-001 (1987 v1; 1988 v2 copies exist) | Audit content, review, protection | https://irp.fas.org/nsa/rainbow/tg001.htm and https://wiretapped.mirrorservice.org/info/reference/rainbow-books/tan-book.pdf | Title and "shall/should" convention confirmed in snippet. Body UNVERIFIED. |
| Green | DoD Password Management Guideline | CSC-STD-002-85, 12 Apr 1985 | Password management | Search snippet only | Identifier from snippet. Body UNVERIFIED. |
| Brown | Trusted Facility Management | NCSC-TG-015, 1989 | Operator separation of duties, trusted facility | Search snippet only | Identifier and title from snippet. UNVERIFIED. |
| Dark Lavender | Trusted Distribution | NCSC-TG-008, 15 Dec 1988 | Release integrity | Search snippet only | Identifier from snippet. UNVERIFIED. |
| Light Blue | I&A (TG-017), object reuse (TG-018) | NCSC-TG-017 / -018 | I&A, object reuse | Search snippets | Colour assignment of TG-018 conflicts between sources. UNVERIFIED. |
| Yellow | Technical rationale for CSC-STD-003-85 | CSC-STD-004-85 | Background only | Search snippet | UNVERIFIED. |
| Blue | Trusted Product Evaluation Questionnaire | NCSC-TG-019, v1 Oct 1989, v2 May 1992 | Self-assessment checklist (not used for requirements) | Search snippet | UNVERIFIED. |
| Forest Green | n/a | not found | Not used | Search returned nothing | Not identified. Excluded. |

Modern references used for mapping:

- ISO/IEC 15408 (Common Criteria). Current edition not verified in this pass.
- NIST SP 800-53 Rev 5. Control IDs AC-x, AU-x, IA-x, SC-x, SI-x, CP-x, SR-x and SA-x are used at family/control level. AU-9 and AU-10 content came from vendor summaries (UNVERIFIED against NIST text).
- OWASP ASVS 5.0.0 (May 2025). Chapter V6 (Authentication) confirmed in snippet. V7 Session Management, V8 Authorization, V11 Cryptography, V14 Data Protection, V15 Secure Coding and V16 Security Logging are from memory and marked UNVERIFIED.

## 4. Rainbow to modern control mapping (summary)

| Rainbow concern | Modern equivalent | Notes |
|---|---|---|
| DAC / need-to-know | NIST AC-2, AC-3, AC-6; ASVS V8 | Circula is not multilevel, so MAC labels are N/A. |
| Object reuse | NIST SC-4 (information in shared resources) | Matters for deleted rows and freed pages. |
| I&A | NIST IA-2, IA-5; ASVS V6 | Plus IA-2(1) for MFA. |
| Password management (Green) | NIST IA-5(1); SP 800-63B | Modern guidance favours length and breached-password screening over forced rotation. |
| Audit (Orange, Tan) | NIST AU-2, AU-3, AU-6, AU-9, AU-10, AU-11; ASVS V16 | Hash chain covers AU-9 integrity in part. |
| Trusted path / network (Red) | NIST SC-8, SC-12, SC-13; ASVS V12 | TLS termination is an operator duty here. |
| Covert channels (Orange) | NIST SC-31 (covert channel analysis; ID from memory, UNVERIFIED) | Relevant to timing and error-oracle leaks. |
| Security testing (Orange) | NIST SA-11; ISO 15408 ADV/ATE families | |
| Design documentation (Orange) | NIST SA-4, SA-17; CC ADV | |
| Trusted recovery (Orange) | NIST CP-10; CP-9 | |
| Trusted facility / separation of duties (Brown) | NIST AC-5, AC-6; AU-9(4) | |
| Trusted distribution (Dark Lavender) | NIST SR-4, SI-7, CM-14; SLSA provenance | |
| Audit protection by separate system | NIST AU-9(2) | Per snippet. |

## 5. Requirements

Priority: P0 = fix before any public launch. P1 = next release. P2 = hardening backlog.
Swarm column: the overlap with the security swarm areas (auth, MFA, recovery, dual control, break-glass, erasure, backup encryption, release signing). "Free" means no overlap.

### RB-001: Discretionary access control, least privilege per network

- Source: Neon Orange (NCSC-TG-003, DAC guide); TCSEC DAC category (Orange).
- Requirement: every access to a ledger object must be granted by an explicit rule scoped to one network, deny by default. A subject must not reach another network's objects through any route, including API, web and CLI.
- Circula: **Partial, built**. Roles are typed in `internal/domain/types.go` (`RoleAdmin`, `RoleReviewer`, lines 90 to 95). Member visibility is enforced in `internal/service/tx_queries.go` (`canSeeTx`). Authorisation tests exist in `internal/service/authz_test.go`. No single route-to-permission table was found in this pass (UNVERIFIED).
- Recommendation: one declarative table of (route, role, network) permissions, generated or checked by a test that fails when a handler is added without an entry. Deny-by-default in the router middleware.
- Priority: P1. Swarm: Free.
- Verification: table-driven test over every registered route, every role, and a second network. Expect deny for each cross-network pair. Add a test that fails if a new route has no table entry.

### RB-002: Mandatory access control labels

- Source: TCSEC MAC category (Orange).
- Requirement: MAC labels and their enforcement are required only for multilevel systems.
- Circula: **Not applicable**. Circula is single-level per deployment and per network.
- Recommendation: record the N/A decision in `docs/specs/13-assumptions-risks-decisions.md` or a decision record. No code.
- Priority: P2 (documentation only). Swarm: Free.
- Verification: decision record exists and is referenced from the threat model.

### RB-003: Object reuse (no residual data in freed storage)

- Source: TCSEC object reuse category (Orange); Light Blue NCSC-TG-018 (colour UNVERIFIED); NIST SC-4.
- Requirement: a storage object must not expose prior content when reallocated. For Circula this covers SQLite freed pages, the WAL, and the backups produced from them.
- Circula: **Not built**. `internal/store/sqlite.go` sets WAL and busy timeout but no `secure_delete` pragma (grep found none). Erasure exists in `internal/service/erasure.go` and `internal/service/purge.go`, but no evidence was found that purged bytes are overwritten on disk.
- Recommendation: set `PRAGMA secure_delete=ON` on every connection. Run `VACUUM` (or incremental vacuum plus checkpoint) after an erasure job, and checkpoint the WAL. Zero in-memory key buffers after use (`[]byte` zeroing; Go cannot guarantee this, so document the limit).
- Priority: P1. Swarm: Overlaps erasure (storage portion only).
- Verification: write a canary string into a member's PII, run erasure, checkpoint, then scan the `.db`, `-wal` and `-shm` files and any backup for the canary. Expect no match.

### RB-004: Identification and authentication before any action

- Source: TCSEC I&A category (Orange); Light Blue NCSC-TG-017 (colour UNVERIFIED); Red (network I&A).
- Requirement: every non-public operation must be attributed to an authenticated identity. No anonymous write path.
- Circula: **Built**. `internal/service/auth.go` resolves sessions to actors (`Authenticate`). Public routes are limited (UNVERIFIED for completeness). Tests: `internal/service/auth_test.go`, `internal/api/authn_test.go`.
- Recommendation: a route audit that lists every unauthenticated route and justifies it.
- Priority: P1. Swarm: **Overlaps auth**.
- Verification: enumerate routes, call each without a credential, expect 401 or 403 except the listed public set.

### RB-005: Password storage and management

- Source: Green Book (CSC-STD-002-85, DoD Password Management Guideline); NIST IA-5(1); SP 800-63B (not verified in this pass).
- Requirement: store passwords with a slow, salted, memory-hard KDF. Enforce a minimum length. Screen for compromised passwords. Do not impose forced periodic rotation.
- Circula: **Partial**. Argon2id with parameters recorded in each hash (`internal/service/password.go`, `hashPassword`, `verifyPassword`). Minimum length enforced (`password.go` line 80, `cfg.MinPasswordLen`). **No breached-password screening** (grep for pwned, hibp, breach found nothing).
- Recommendation: raise the default minimum length. Add an offline breached-password check using a k-anonymity or local bloom-filter list, so no network call is needed at login. Reject passwords equal to the account identifier.
- Priority: P1. Swarm: **Overlaps auth**.
- Verification: unit tests for length floor, for rejection of a known-breached sample, and for acceptance of a long random passphrase. Test that existing hashes with older parameters still verify and are rehashed on next login.

### RB-006: Authentication failure throttling and lockout

- Source: TCSEC I&A and audit categories (Orange); Red (network I&A).
- Requirement: bound online guessing per account and per source. Make the bound observable in audit.
- Circula: **Partial, built**. Exponential throttle after 5 failures, base 15 minutes, cap 24 hours (`internal/service/auth.go`, lines 89 to 103). Whether the counter is per account, per IP or both is UNVERIFIED in this pass.
- Recommendation: throttle on both account and source key. Emit an audit event on threshold crossing. Make the throttle state survive restart (it is in SQLite, verify).
- Priority: P1. Swarm: **Overlaps auth**.
- Verification: test that N+1 failures from one IP across several accounts is throttled, and that the throttle emits an audit event.

### RB-007: Multi-factor authentication and recovery codes

- Source: Not a Rainbow concern. Closest is TCSEC I&A strength. Map to NIST IA-2(1) and ASVS V6.
- Requirement: MFA for privileged roles. Recovery codes single-use and stored hashed.
- Circula: **Built**. TOTP in `internal/service/totp.go` and `internal/service/mfa.go`. Single-use recovery codes via `VerifyAndConsumeRecoveryCode` (`mfa.go`, line 71). Migration `0003_mfa.sql`.
- Recommendation: make MFA mandatory for admin and reviewer roles. Verify TOTP replay protection (a code must not be accepted twice inside its window).
- Priority: P0 for admin enforcement (verify first). Swarm: **Overlaps MFA and recovery**.
- Verification: admin login without MFA is refused. Reused TOTP code is refused. Recovery code is refused on second use.

### RB-008: Session lifetime, idle timeout and revocation

- Source: TCSEC I&A (Orange); ASVS V7 (UNVERIFIED chapter number).
- Requirement: idle and absolute timeouts; server-side revocation; cookies `HttpOnly`, `Secure` on HTTPS, `SameSite`.
- Circula: **Built**. `internal/service/auth.go` (`Authenticate`) rejects revoked, idle-expired and absolutely-expired sessions (SEC-003). Cookies set `HttpOnly` and `SameSite=Lax` and `Secure` when HTTPS (`internal/web/session.go`, lines 37 to 39).
- Recommendation: rotate session ID on login and on privilege change. Revoke all sessions on password change (verify).
- Priority: P1. Swarm: **Overlaps auth**.
- Verification: session fixation test (ID changes on login). Password change revokes other sessions.

### RB-009: Audit of all security-relevant actions

- Source: TCSEC audit category (Orange); Tan (NCSC-TG-001).
- Requirement: every security-relevant action (auth, authorisation denial, ledger write, role change, admin action, export, erase) produces an audit event.
- Circula: **Partial, built**. Action constants in `internal/domain/audit.go` and `audit_*.go` files (MFA, recovery, dual control, break-glass, erasure, setup). Coverage per handler not verified.
- Recommendation: test that enumerates action constants and asserts each has at least one call site. Add audit events for denied authorisations if missing (UNVERIFIED).
- Priority: P1. Swarm: Overlaps dual control, break-glass, erasure, recovery, MFA (the audit events themselves).
- Verification: coverage test over action constants. Negative test: a denied request produces a denial audit event.

### RB-010: Audit tamper evidence (hash chain plus external anchor)

- Source: TCSEC audit protection (Orange); Tan (NCSC-TG-001); NIST AU-9, AU-10.
- Requirement: audit records must be tamper-evident. A party with database write access must not be able to rewrite history undetected.
- Circula: **Partial**. Each audit row carries `prev_hash` and `hash` (`internal/store/sqlite_audit.go`, around line 22). Update and delete triggers abort (`internal/store/migrations/0001_init.sql`, `audit_no_update` at line 283, and `entries_no_update` / `entries_no_delete` at lines 278 to 281). Chain verifier is `verifyAuditChain` in `internal/service/verify.go` (line 153). **Gap**: the chain head is stored in the same database. An attacker with file write access can recompute the whole chain. No external anchor was found (grep for anchor, checkpoint, witness found nothing).
- Recommendation: periodically publish a signed checkpoint of each network's chain head (sequence number plus hash). Sign with a key that is not on the database host (offline key, HSM, or a cloud KMS key with separate access). Store checkpoints off-host in an append-only location. Verifier checks head against the latest checkpoint.
- Priority: P0. Swarm: Free.
- Verification: test that modifying any historic row (including recomputing the chain from that point) is detected against the external checkpoint. Test that a missing checkpoint is reported by `VerifyLedger`.

### RB-011: Audit review, scoped read access

- Source: Tan (NCSC-TG-001), Orange audit review.
- Requirement: audit records readable only by authorised reviewers, scoped to their network, with reduction tools. Personal data minimised in audit.
- Circula: **Built**. Scoped audit read (`internal/api/h_audit_scoped.go`, `internal/service/audit_read.go`). Spec `docs/specs/09-security-privacy.md` (line 292) forbids secrets and full narratives in audit events. Test coverage: `internal/web/audit_scoped_test.go`.
- Recommendation: add a redaction test that scans audit output for known secret patterns.
- Priority: P2. Swarm: Free.
- Verification: scoped read tests exist. Add the redaction scan.

### RB-012: Audit stored apart from the audited system

- Source: NIST AU-9(2) (per snippet); Tan.
- Requirement: audit records copied to a store the audited application cannot modify.
- Circula: **Not built**. Audit lives in the same SQLite file as the ledger.
- Recommendation: stream audit events (or the checkpoints from RB-010) to a remote append-only sink (object storage with object lock, or a remote syslog collector). Keep the local copy for queries.
- Priority: P1. Swarm: Free.
- Verification: test that the sink receives each event in order. Test that the sink can be verified independently of the database file.

### RB-013: Trusted path (encryption in transit, refuse plaintext)

- Source: TCSEC trusted path (Orange); Red (network security).
- Requirement: user credentials and session tokens must not cross an untrusted network in clear.
- Circula: **Partial**. The binary does not terminate TLS. `cmd/circula/serve.go` (line 185) logs a warning when listening on a non-loopback address. HSTS and secure-cookie policy live in an edge helper (`internal/web/web.go`, `Edge`).
- Recommendation: refuse to start on a non-loopback address unless TLS is configured or an explicit `--insecure-behind-proxy` flag is set. Set minimum TLS 1.2 (prefer 1.3).
- Priority: P0. Swarm: Free.
- Verification: start with a non-loopback address and no TLS. Expect a refusal. Start with TLS and check the negotiated minimum version with a test client.

### RB-014: Operator and metrics interfaces not on the public listener

- Source: Red (network security); Brown (trusted facility management).
- Requirement: operator endpoints (metrics, ops HTTP) isolated from the public network.
- Circula: **Partial**. `SECURITY.md` and `docs/ops/configuration.md` tell the operator to keep `/metrics` private. The default is not enforced in code (UNVERIFIED for defaults).
- Recommendation: serve ops endpoints on a separate loopback or Unix-socket listener by default. Require a token for `/metrics` even on loopback.
- Priority: P1. Swarm: Free.
- Verification: a request to `/metrics` from a non-loopback peer is refused by default.

### RB-015: Authentication of API tokens (scopes, expiry, storage)

- Source: Red (network I&A); TCSEC I&A.
- Requirement: API tokens scoped to least privilege, with expiry and revocation. Stored hashed.
- Circula: **Partial**. Scopes are typed in `internal/domain/types.go` (`Scope`). Token handlers in `internal/api/h_tokens.go` and `internal/service/tokens.go`. Storage hashing and expiry not verified in this pass (UNVERIFIED).
- Recommendation: confirm tokens are stored as hashes with a separator and a short prefix for lookup. Enforce expiry. Test that a token with scope X cannot call a scope-Y route.
- Priority: P1. Swarm: **Overlaps auth**.
- Verification: scope matrix test. Token at rest scan for plaintext.

### RB-016: Input validation at trust boundaries

- Source: TCSEC integrity; Red (system integrity); ASVS V2 and V15 (UNVERIFIED numbers).
- Requirement: every boundary decodes into a typed value with bounds checks. Money arithmetic must not overflow.
- Circula: **Built, partial evidence**. Decoding in `internal/api/decode.go`. Validation in `internal/service/validate.go`. Money uses integer minor units (`internal/domain/money.go`). Opaque cursors (`internal/api/cursor.go`).
- Recommendation: add overflow tests on all money-accepting routes. See RB-017 for fuzzing.
- Priority: P1. Swarm: Free.
- Verification: boundary tests at max int64 and at max policy amount.

### RB-017: Security testing: fuzzing and adversarial tests

- Source: TCSEC security testing category (Orange); NIST SA-11; ISO 15408 ATE family.
- Requirement: security functions tested for bypass, and parsers fuzzed.
- Circula: **Partial, built**. Property tests `internal/service/property_test.go`. Concurrency tests `internal/service/concurrency_test.go`. Hardening tests `internal/api/hardening_test.go`, `internal/store/hardening_test.go`. E2E security spec `e2e/tests/security.spec.js`. **No Go fuzz targets** (`func Fuzz` count is 0 in the repository).
- Recommendation: add Go native fuzz targets for decoders (`internal/api/decode.go`), cursors (`cursor.go`), backup header parsing (`cmd/circula/backupcrypt.go`, `parseBackupSecret`), idempotency keys (`internal/service/idempotency.go`) and SQL-backed filters. Run seeds in CI and a longer fuzz job nightly.
- Priority: P1. Swarm: Free (backup header parsing touches backup encryption; the fuzz target itself is free).
- Verification: CI runs fuzz seed corpora on every push. Nightly job reports crashes.

### RB-018: Covert channels (timing and error oracles)

- Source: TCSEC covert channel category (Orange).
- Requirement: identify storage and timing channels that leak data across a security boundary, and bound their bandwidth.
- Circula: **Not analysed**. Candidate channels: login timing (user enumeration if unknown accounts skip hashing), differing error text, balance-inference through repeated transfers, and throttle state visible to other accounts.
- Recommendation: written covert channel analysis in `docs/security/`. Run Argon2 on a dummy hash for unknown accounts. Use uniform error messages on login and recovery. Make throttle responses independent of account existence.
- Priority: P2 (P1 if user enumeration is confirmed). Swarm: **Overlaps auth and recovery**.
- Verification: timing test comparing unknown-user and wrong-password responses, with a stated tolerance. Test identical response bodies for both.

### RB-019: Design documentation and traceability

- Source: TCSEC design documentation (Orange); NIST SA-4, SA-17; CC ADV.
- Requirement: a current design and a mapping from requirements to code and tests.
- Circula: **Built**. `docs/specs/00-conventions.md` to `13-assumptions-risks-decisions.md`, `docs/TRACEABILITY.md`, `docs/DECISIONS.md`.
- Recommendation: a CI check that every requirement ID in specs appears in TRACEABILITY.md. Add this RB series to traceability when merged.
- Priority: P2. Swarm: Free.
- Verification: CI script fails on orphan requirement IDs.

### RB-020: Trusted recovery (restore verifies integrity)

- Source: TCSEC trusted recovery (Orange); NIST CP-9, CP-10.
- Requirement: a restored system must be shown to be consistent before it is used. Recovery must not silently drop audit or ledger state.
- Circula: **Built, partial evidence**. Restore and recovery code in `cmd/circula/recover.go`. Restore drill test `cmd/circula/restore_drill_test.go`. Backup tests in `cmd/circula/backup_test.go`. Not verified: whether the drill calls `VerifyLedger` and checks the audit chain.
- Recommendation: restore procedure must run `VerifyLedger` for every network and compare the head with the latest external checkpoint (RB-010).
- Priority: P1. Swarm: **Overlaps recovery (backup side)**; the verification step itself is free.
- Verification: drill fails if the restored chain does not match the checkpoint, or if one row is corrupted in the backup copy.

### RB-021: Backup confidentiality and key handling

- Source: Brown (trusted facility management); Red; NIST SC-12, SC-28, CP-9.
- Requirement: backups encrypted with authenticated encryption; keys separated from backup media; key use controlled.
- Circula: **Built**. Chunked AES-256-GCM with Argon2id derivation in `cmd/circula/backupcrypt.go` (`deriveBackupKey`, `newEncryptWriter`, `chunkNonce`, `chunkAAD`). Tests `cmd/circula/backupcrypt_test.go`.
- Recommendation: key rotation procedure, and key escrow under dual control. Confirm the header is authenticated (`chunkAAD` covers header; verify).
- Priority: P1. Swarm: **Overlaps backup encryption**.
- Verification: tamper test flips one bit in each chunk and in the header. Expect decryption refusal.

### RB-022: Separation of duties and dual control

- Source: Brown (trusted facility management); NIST AC-5.
- Requirement: no single operator can complete a high-risk action alone. Approver is never the requester.
- Circula: **Built**. Dual control for large adjustments: `DualControlThreshold` in `internal/domain/policy.go` (line 41). Service in `internal/service/dualcontrol.go` and `dualcontrol_operator.go`. Tests `internal/api/dualcontrol_test.go`, `internal/web/dualcontrol_test.go`.
- Recommendation: confirm self-approval is refused in the service layer, not only in the UI. Time-bound approval requests.
- Priority: P0 to verify, P1 to fix gaps. Swarm: **Overlaps dual control**.
- Verification: self-approval attempt returns refusal. Approval after expiry returns refusal.

### RB-023: Break-glass access with audit and time limit

- Source: Brown (trusted facility management); NIST AC-6, AU-2.
- Requirement: emergency access is possible, time-limited, approved or justified, and audited with a reason.
- Circula: **Built**. `internal/service/breakglass.go`, `internal/domain/audit_breakglass.go`, migration `0006_breakglass.sql`, tests `internal/service/breakglass_test.go`.
- Recommendation: alert on each break-glass grant to a second channel (RB-012 sink).
- Priority: P1. Swarm: **Overlaps break-glass**.
- Verification: expired grant is refused. Each grant emits an audit event and a notice.

### RB-024: Trusted distribution and release integrity

- Source: Dark Lavender (NCSC-TG-008, Trusted Distribution; identifier per snippet, body UNVERIFIED); NIST SR-4, SI-7, CM-14; SLSA provenance (external).
- Requirement: released binaries and checksums are authenticated by a signature that users can verify independently.
- Circula: **Not built**. `.github/workflows/release.yml` (around line 87 to 90) verifies `SHA256SUMS` but publishes no signature, no `cosign` or attestation (grep found none). Checksums in the same release channel offer no authenticity against a compromised release.
- Recommendation: sign `SHA256SUMS` with a key held outside the CI runner (or keyless signing with a recorded identity). Publish SLSA build provenance. Document the verification command in `docs/ops`.
- Priority: P0. Swarm: **Overlaps release signing**.
- Verification: CI job downloads the release and verifies the signature against a pinned identity. A modified artifact fails.

### RB-025: Dependency and vulnerability management

- Source: TCSEC testing and configuration (Orange); NIST SI-2, SR-3, SA-11.
- Requirement: known vulnerable dependencies detected and patched; build tools pinned.
- Circula: **Built**. `govulncheck` in `.github/workflows/ci.yml` (lines 46 to 47), `golangci-lint` with `gosec` (`.golangci.yml`), Dependabot (`.github/dependabot.yml`), GitHub Actions pinned by SHA (`ci.yml`, golangci action).
- Recommendation: add SBOM generation to the release job. Gate on fixed `govulncheck` output.
- Priority: P2. Swarm: Overlaps release signing (SBOM only).
- Verification: CI fails on a seeded vulnerable dependency in a test branch.

### RB-026: Erasure with audit and verifiable completion

- Source: TCSEC object reuse (Orange); NIST SI-12 (information handling), SC-4.
- Requirement: erasure is authorised, audited, and verifiable. Erasure does not break the ledger chain.
- Circula: **Built, with storage gap**. `internal/service/erasure.go`, `internal/service/purge.go`, `internal/api/h_*`, migration `0007_erasure.sql`, test `internal/web/breakglass_erasure_test.go`. Storage gap is RB-003.
- Recommendation: after erasure, prove it: the chain verifies, and the canary check from RB-003 passes.
- Priority: P1. Swarm: **Overlaps erasure**.
- Verification: erasure test checks the chain, the canary scan and the audit event.

### RB-027: Ledger arithmetic integrity cross-check

- Source: TCSEC integrity; Orange design documentation.
- Requirement: the balance projection equals the fold of the immutable entries at all times.
- Circula: **Built**. `internal/domain/fold.go`, `internal/service/verify.go` (`VerifyLedger`, balance corruption detection; test `internal/service/audit_test.go`, `TestVerifyDetectsBalanceCorruptionAndBalanceAt`). Read-only verifier tests in `hardening_test.go`.
- Recommendation: run `VerifyAndRecordLedger` on a schedule and alert on mismatch.
- Priority: P2. Swarm: Free.
- Verification: scheduled job test and alert test.

### RB-028: Single writer and write serialisation

- Source: TCSEC integrity and concurrency; Red (system integrity).
- Requirement: writes serialised so that concurrent requests cannot interleave ledger postings or break the audit chain.
- Circula: **Built**. `internal/store/sqlite.go` runs writes in `BEGIN IMMEDIATE` (line 102) under a process write gate. Connection pool capped at 8 (line 77). Concurrency tests `internal/service/concurrency_test.go`, `race_on_test.go`.
- Recommendation: document that the process must be a single instance, or add a lock file for multi-process (`cmd/circula/lock_unix.go` exists; confirm it guards the database).
- Priority: P2. Swarm: Free.
- Verification: race test with the race detector on in CI. Second process start refused by the lock.

### RB-029: Resource limits (denial of service)

- Source: Red (availability of network services); TCSEC is silent. Mapped to NIST SC-5.
- Requirement: bounded request bodies, job queues and notification volume.
- Circula: **Built, partial**. `cmd/circula/jobs_limits.go`, `internal/service/notify_limits.go`, `internal/api/idem.go`. Body limits in `internal/httpx/httpx.go` (UNVERIFIED values).
- Recommendation: a table of limits in `docs/ops/configuration.md`, each with a test.
- Priority: P2. Swarm: Free.
- Verification: oversize body refused. Job queue at limit refuses.

### RB-030: Log and error content free of secrets

- Source: Tan (audit content); Orange audit.
- Requirement: logs and error responses carry no passwords, tokens, MFA secrets or recovery codes.
- Circula: **Partial**. `docs/specs/09-security-privacy.md` (line 292) forbids this in audit events. No scan test was found.
- Recommendation: a redaction test that feeds known secret values through login, MFA, token and recovery paths and asserts absence in logs, audit and error bodies.
- Priority: P1. Swarm: **Overlaps MFA and recovery** (for the secret values).
- Verification: the redaction test above.

## 6. Gap summary

| ID | Topic | Status | Priority | Swarm overlap |
|---|---|---|---|---|
| RB-001 | DAC per network | Partial | P1 | Free |
| RB-002 | MAC | N/A | P2 | Free |
| RB-003 | Object reuse / secure delete | Not built | P1 | Erasure (storage) |
| RB-004 | I&A before action | Built | P1 | Auth |
| RB-005 | Password storage and policy | Partial | P1 | Auth |
| RB-006 | Throttling and lockout | Partial | P1 | Auth |
| RB-007 | MFA and recovery codes | Built | P0 (enforce for admins) | MFA, recovery |
| RB-008 | Session lifetime | Built | P1 | Auth |
| RB-009 | Audit coverage | Partial | P1 | Dual control, break-glass, erasure, recovery |
| RB-010 | Audit tamper evidence with external anchor | Partial | **P0** | Free |
| RB-011 | Scoped audit read | Built | P2 | Free |
| RB-012 | Audit stored apart | Not built | P1 | Free |
| RB-013 | Trusted path / TLS refusal | Partial | **P0** | Free |
| RB-014 | Operator interfaces isolated | Partial | P1 | Free |
| RB-015 | API token scope and storage | Partial | P1 | Auth |
| RB-016 | Input validation at boundaries | Built | P1 | Free |
| RB-017 | Fuzzing and adversarial tests | Partial | P1 | Free (backup parser touches backup) |
| RB-018 | Covert channels | Not analysed | P2 | Auth, recovery |
| RB-019 | Design traceability | Built | P2 | Free |
| RB-020 | Trusted restore with verify | Partial | P1 | Recovery |
| RB-021 | Backup encryption and keys | Built | P1 | Backup encryption |
| RB-022 | Dual control | Built | P0 (verify) | Dual control |
| RB-023 | Break-glass | Built | P1 | Break-glass |
| RB-024 | Release signing | Not built | **P0** | Release signing |
| RB-025 | Dependency management | Built | P2 | Release signing (SBOM) |
| RB-026 | Erasure proof | Built, gap | P1 | Erasure |
| RB-027 | Ledger cross-check | Built | P2 | Free |
| RB-028 | Single writer | Built | P2 | Free |
| RB-029 | Resource limits | Built, partial | P2 | Free |
| RB-030 | Secret-free logs | Partial | P1 | MFA, recovery |

P0 totals: four items. Free P0: RB-010, RB-013. Swarm-overlap P0: RB-007 (admin MFA enforcement), RB-022 (verify self-approval refusal), RB-024 (release signing).

## 7. Roadmap

Phase 0, before launch (P0):
1. RB-010: external signed checkpoint of audit chain heads. Free.
2. RB-013: refuse non-loopback start without TLS. Free.
3. RB-007, RB-022: verify admin MFA enforcement and self-approval refusal. Swarm.
4. RB-024: sign release checksums and publish verification steps. Swarm.

Phase 1, next release (P1):
1. RB-003 and RB-026: `secure_delete`, post-erasure VACUUM, canary test. Storage portion is free; erasure logic is swarm.
2. RB-012: remote append-only audit sink (depends on RB-010).
3. RB-017: fuzz targets and CI seed run.
4. RB-020: restore drill verifies chain against checkpoint (depends on RB-010).
5. RB-005, RB-006, RB-008, RB-015, RB-004: auth hardening. Swarm.
6. RB-009, RB-030: audit coverage test and secret-redaction test.
7. RB-014: ops listener isolation.
8. RB-001: route-permission table and deny-by-default test.
9. RB-016: overflow tests on money routes.

Phase 2, hardening backlog (P2):
1. RB-018: covert channel analysis document and timing test.
2. RB-002: MAC N/A decision record.
3. RB-011, RB-019, RB-025, RB-027, RB-028, RB-029: tests and documentation.

## 8. Swarm overlap summary

- Overlaps swarm areas: RB-004, RB-005, RB-006, RB-007, RB-008, RB-009 (events for dual control, break-glass, erasure, recovery), RB-015, RB-018 (auth and recovery), RB-020 (backup side of recovery), RB-021, RB-022, RB-023, RB-024, RB-025 (SBOM only), RB-026, RB-030 (MFA and recovery secrets), RB-003 (erasure storage portion), RB-017 (backup header parser only).
- Free (no swarm overlap): RB-001, RB-002, RB-010, RB-011, RB-012, RB-013, RB-014, RB-016, RB-019, RB-027, RB-028, RB-029, and the verification step of RB-020.

## 9. Sources

- DoD 5200.28-STD (TCSEC, 1985): https://apps.dtic.mil/sti/tr/pdf/ADA207905.pdf (title and date confirmed in search snippet; full text UNVERIFIED)
- Rainbow Series index: https://en.wikipedia.org/wiki/Rainbow_Series (search snippet; fetch refused by egress policy)
- Red Book, NCSC-TG-005: https://irp.fas.org/nsa/rainbow/tg005.htm and https://apps.dtic.mil/sti/tr/pdf/ADA255422.pdf (identifier and date confirmed in snippet; body UNVERIFIED)
- Tan Book, NCSC-TG-001: https://irp.fas.org/nsa/rainbow/tg001.htm and https://wiretapped.mirrorservice.org/info/reference/rainbow-books/tan-book.pdf (snippet only)
- NIST Rainbow Series page: https://csrc.nist.gov/pubs/other/1985/12/26/dod-rainbow-series/final (listed in search; fetch refused by egress policy)
- NIST SP 800-53 Rev 5 AU-9, AU-10: vendor summaries only in this pass (UNVERIFIED against NIST text). Catalogue: https://csrc.nist.gov
- OWASP ASVS 5.0.0: https://github.com/OWASP/ASVS (V6 confirmed via snippet; other chapter numbers from memory, UNVERIFIED)
- Green Book CSC-STD-002-85, Brown Book NCSC-TG-015, Dark Lavender NCSC-TG-008, Blue Book NCSC-TG-019, Yellow CSC-STD-004-85: identifiers from search snippets; no URL verified. UNVERIFIED.
- Circula code: paths cited inline, worktree commit `6422f87`.
