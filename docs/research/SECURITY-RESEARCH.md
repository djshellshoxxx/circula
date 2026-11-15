# Circula security research

Date: 2026-10-21. Repo: /home/user/circula (Go, SQLite WAL, html/template + JSON API, session cookies and bearer tokens, Argon2id).

Method: read the code in internal/ and cmd/circula/, the workflows in .github/workflows/, and the two existing audit files in docs/audit/. Web research was done for each topic. No code was edited, built or run, and nothing was committed.

## 0. Read this first

### 0.1 Source status

Primary sources were NOT fetched directly in this run. Egress from the session blocked them: WebFetch returned DNS failures (`getaddrinfo ENOTFOUND`) for rfc-editor.org, sqlite.org, cheatsheetseries.owasp.org and pages.nist.gov. curl through the agent proxy returned `CONNECT tunnel failed, response 403` for the same hosts (see /root/.ccr/README.md: policy denial, not retried). Where a claim below is backed by a primary URL, it comes from WebSearch result text (titles, snippets, mirrors), and is labelled "snippet" in the source table. Where no source could be reached, the text says "not confirmed". All accesses are dated 2026-10-21.

### 0.2 docs/audit/SECURITY-AUDIT.md and CODE-AUDIT.md are partly stale

Both audit files pin HEAD `94c77c9` on branch `claude/circula-build`. The working tree is newer: many findings they list are already fixed in code. Re-checked this run by reading code:

| Audit claim | Current code | Evidence |
|---|---|---|
| SA-01 `next` open redirect via TAB/VT | Fixed | `httpx.SafeNext` rejects control chars, space, backslash, `//`, and percent-decoded forms (internal/httpx/httpx.go) |
| SA-02 cookie Secure not forced, no start-up refusal | Fixed | `cookie-secure` default true, wired to `httpx.Edge.ForceSecure` (cmd/circula/config.go, serve.go:234); `checkTransport` refuses http:// on non-loopback unless `allow-insecure-http` (config.go:411-418) |
| SA-03 network creation open, SetupSecret not wired | Fixed | `setup-secret` setting, one-time use via `setupUsed` (web/h_auth.go:152-154) |
| SA-04 parallel login guesses bypass throttle | Fixed | Attempt is counted inside the same write unit before verification (service/auth.go `Login`) |
| SA-05 rate limiter ignores proxies | Fixed | `httpx.Edge.ClientIP`, `trusted-proxies` CIDRs, right-most untrusted XFF (httpx.go) |
| SA-06 DB/backup world-readable | Fixed | `restrictUmask` 077 on unix (cmd/circula/umask_unix.go); backups created 0600 (snapshot.go:187-202, 316) |
| SA-07 forged backup swaps credentials | Partly fixed | Optional `backup-key` signs manifest; restore refuses unsigned manifest when key set (snapshot.go `checkManifestAuth`). Without a key, restore only warns (commands.go:332). |
| SA-09 audit tail truncation undetected | Partly fixed | Migration 0002 adds `audit_heads` with advance-only trigger. Still same DB, still unkeyed (see section 5). |
| SA-12 no purge | Fixed | `service/purge.go` (sessions, throttles, idempotency) |
| SA-13 invitation token in logs | Fixed | `route.go:57` logs `r.Pattern` |
| SA-14 actions pinned to tags | Fixed | All `uses:` pinned to 40-hex SHA; `check-action-pins.sh` enforces it; release cache off |
| SA-15 /metrics gate | Fixed | Forwarding headers refused, optional bearer `metrics-token` (cmd/circula/ops_http.go `metricsAccess`) |
| CODE-AUDIT: View() can write (P1) | Fixed | `View` sets `PRAGMA query_only = ON` (store/sqlite.go) |
| CODE-AUDIT: API idempotency record stuck after panic | Fixed | `defer` releases key (api/idem.go `runIdempotent`) |
| CODE-AUDIT: no purge job | Fixed | service/purge.go |
| SA-10 registration reveals logins | Fixed | Uniform decoy response (service/auth.go `RegisterPerson`) |
| SEC-002 breached-password check | NOT fixed | `PasswordDenied` field exists (service/service.go:26) and is used (password.go:84) but nothing assigns it. grep finds no assignment. |
| SEC-007 MFA / TOTP, SEC-008 recovery | NOT implemented | grep for totp, mfa, recovery code finds nothing |
| SA-08 request-proposal freezing | Not re-checked | |
| SA-11 export artefact exhaustion | Not re-checked | |
| SA-14 release provenance, SBOM, signing | NOT fixed | see section 8 |

This research uses the current code as the baseline. Items marked "not re-checked" are carried over from the audit without re-verification. The audit suggests running `govulncheck` and the race suite; neither was run here.

### 0.3 Summary

| # | Area | Overall | Biggest gap |
|---|---|---|---|
| 1 | Ledger integrity, SQLite | Strong | WAL + synchronous=NORMAL can lose acknowledged commits on power loss |
| 2 | Session, cookie, CSRF | Good | Session lifetime 30 d idle / 90 d absolute; plain-name cookie accepted in secure mode |
| 3 | Passwords, throttling, enumeration | Good hashing, weak lockout | Per-login lockout can be held shut indefinitely; breached-password check unwired |
| 4 | TOTP MFA, recovery codes | Missing | No MFA, no recovery |
| 5 | Tamper-evident audit | Partial | Unkeyed chain, same-DB head, no external anchor |
| 6 | Backup, restore, encryption | Partial | No encryption at rest; restore unauthenticated unless key set |
| 7 | Multi-tenant isolation, IDOR | Strong | Composite FKs in schema; not all store methods audited |
| 8 | Supply chain | Partial | No signing, no provenance, no approval gate on publish |
| 9 | Rate limiting behind proxies | Good design, unsafe default | Default `trusted-proxies none` makes all proxied users share one budget |

---

## 1. Ledger integrity, double-spend, races (SQLite)

### 1.1 Top risks

- Overspend through concurrent postings (check-then-debit race).
- Lost updates on the cached `accounts.balance`.
- Acknowledged commit lost after power failure (durability, not corruption).
- Partial posting (one entry without its pair).
- Duplicate submission (client retry, double click) posting twice.
- Long full-table verification scans blocking all postings (availability).
- Multi-process writers (CLI backup/verify vs `serve`) producing `SQLITE_BUSY` or corrupting assumptions.

### 1.2 Recommended controls

- One write transaction per command, started with `BEGIN IMMEDIATE` so the write lock is taken at the start. A deferred transaction that reads and then writes can fail with SQLITE_BUSY even with a busy timeout, because the lock upgrade bypasses the timeout. (Snippets: SQLite transaction docs mirror https://system.data.sqlite.org/home/raw/124adce1a34b456ca46ca6839bf145ce55753b3a; Willison summary https://www.simonwillison.net/2025/Feb/17/sqlite-busy. Official https://www.sqlite.org/lang_transaction.html not fetched.)
- Check limits and debit in the same transaction that reads the balance.
- Serialise writers in-process as well (a gate), so the busy timeout only matters for other processes.
- WAL mode for concurrent readers. Snippet: under WAL, readers are not blocked by a writer (https://www.sqlite.org/wal.html, not fetched).
- `synchronous=FULL` when acknowledged commits must survive power loss. Snippet of official text (https://www.sqlite.org/docsrc/info/bcf687da9c172292): in WAL mode with synchronous=NORMAL, "a transaction committed ... might roll back following a power loss or system crash". The database stays consistent; the commit can be lost.
- Idempotency keys stored in the same transaction as the change, so commit or rollback are atomic together.
- DB-level invariants as CHECK and composite foreign keys, not just code.

### 1.3 What Circula does

- `store/sqlite.go` `OpenWith`: `busy_timeout` 5 s (`DefaultBusyTimeout`), `journal_mode(WAL)`, `synchronous(NORMAL)`, `foreign_keys(1)`, pool of 8.
- `Update` takes an in-process channel gate and then `BEGIN IMMEDIATE`. `View` runs `BEGIN` with `query_only = ON`, so reads cannot write. Migrations also use `BEGIN IMMEDIATE` (store/migrate.go).
- `service/posting.go` `postTx`: payer and payee read inside the write unit, `checkCapacity` (LED-5) before any entry, two equal and opposite entries, cache updated with `AddBalance`, status event, all in one unit.
- Schema (store/migrations/0001_init.sql): `CHECK (amount > 0)`, `CHECK (reversed_total >= 0 AND reversed_total <= amount)`, `CHECK (payer_account_id <> payee_account_id)`, append-only triggers on entries, audit, status events, consistency runs, and transaction status-flow trigger (`tx_status_flow`), `tx_complete_needs_entries`, `entries_only_for_open_tx`.
- Idempotency: service `idempotent()` (service/idempotency.go) records `(network, actor, operation, key)` plus a fingerprint in the same unit, and replays the stored result. Used by transactions (3 call sites), adjustments, reversal, disputes, listings and networks. Membership and role changes do not use it.
- `serve` takes a lock file (cmd/circula/lock_unix.go), so one serving process per database.
- Tests named in the repo: service/race_on_test.go, race_off_test.go, concurrency_test.go, atomicity_test.go, idempotency_test.go. CODE-AUDIT ran `go test -race` clean at its HEAD. Not re-run here.

### 1.4 Gaps, ranked

1. **Durability (Medium-High for a ledger).** `synchronous(NORMAL)` in WAL means a 200 response for a posting can be lost on power loss or kernel crash. The ledger stays internally consistent, but a completed trade disappears. The fix is a one-line change to `synchronous(FULL)`, with a benchmark, since each commit then fsyncs.
2. **`accounts.balance` cache has no direct-write guard (Medium).** CODE-AUDIT LED-2 notes this. Only `VerifyLedger` (hourly job, serve.go) detects drift. A process with raw SQL access can change the cache without touching entries. `BalanceMismatches` in store/sqlite_verify.go is the detector.
3. **Verification holds the only write slot (Medium, availability).** Full-table checks run inside `s.update` (service/verify.go). Every posting waits behind a long hourly scan. Chunk the scan or run it on a snapshot and verify against that.
4. **Global cap, not per-network serialisation (Low).** One write gate serves all networks. Fine at current scale; note for scaling.
5. **API replay cache is per process and 24 h (Low).** `api/idem.go` keeps the first response in memory. After a restart a retry falls back to the service-level record, which is durable for 7 days (`idempotency-ttl` 168 h). Confirm every money-moving command has a service-level key. Not verified for membership and join paths, which are not money-moving.
6. **Key reuse after TTL (Low).** A retry more than 7 days later is treated as new and can post again. Document, or keep the key until the transaction is final.

### 1.5 Cheapest high-value fix

Set `synchronous(FULL)` in `store/sqlite.go` for the ledger file (or at least for `Update` transactions), measure the cost on the target hardware, and record the decision as a DEC. Second: move the hourly verification into chunks.

---

## 2. Sessions, cookies, CSRF

### 2.1 Top risks

- Session theft over HTTP or through a cookie planted by a sibling subdomain.
- Session fixation via a planted plain-name cookie.
- CSRF on state-changing POST, especially money-moving forms.
- Long-lived sessions that survive device loss.
- Login CSRF (attacker signs victim into attacker account).

### 2.2 Recommended controls

- Cookie attributes: `HttpOnly`, `Secure`, `SameSite=Lax` (or Strict for admin), `Path=/`, no `Domain`. Use the `__Host-` prefix. RFC 6265bis defines the prefix rule: a `__Host-` cookie must be set with Secure, Path=/ and no Domain (snippet from a tough-cookie README quoting the draft, https://github.com/salesforce/tough-cookie; IETF draft text not fetched; the draft's SameSite section is 8.8 per the same snippet).
- Opaque 256-bit random session tokens, server stores only a hash.
- Session ID rotation on login and on privilege change.
- Synchronizer or signed double-submit CSRF token. OWASP recommends the signed double-submit pattern with the token bound to the session, and HMAC for signing (snippet, https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html). Unsigned double-submit is considered weak (same source).
- Fetch Metadata (`Sec-Fetch-Site`) as an extra cross-site check (same OWASP snippet).
- Idle and absolute timeouts appropriate to the risk. The OWASP Session Management cheat sheet was not retrieved, so no sourced number is given here.

### 2.3 What Circula does

- Session token: `randToken(32)` (256 bits), stored as `SHA-256` (service/auth.go `Login`, `hashToken`). Revoked on logout and on password change (`ChangePassword` calls `RevokeSessionsOf`).
- New session on every successful login. Login is not fixed to a prior token.
- Cookie writer (`web/session.go` `setCookie`): `HttpOnly`, `SameSite=Lax`, `Path=/`, `Secure = s.secure(r)` where `secure` follows `Edge.Secure` (ForceSecure from `cookie-secure`, default true, or TLS, or trusted `X-Forwarded-Proto`).
- Name: `__Host-circula_session` when secure, else `circula_session` (`httpx.SessionCookieName`). `httpx.SessionToken` reads `__Host-` first, then the plain name.
- CSRF: `httpx.CSRF` is `HMAC-SHA256(process key, "circula-csrf-v2" || binding)`. Binding is the session token when signed in, else the `cm_csrf` cookie value. Shared by HTML and JSON adapters (API-014). Constant-time compare.
- Web `csrfOK`: `sameOrigin` (refuses `Sec-Fetch-Site: cross-site` and `Origin: null`; accepts missing Origin) plus token check.
- API `checkCSRF` (api/auth.go) applies to cookie-authenticated unsafe methods only. Bearer requests skip CSRF, which is correct for bearer auth.
- Login CSRF: login form has a CSRF token from `ensureCSRF` (pre-login binding).
- Logout rotates `cm_csrf`; `clearSession` clears both names.
- Idle timeout 720 h (30 days) and absolute 2160 h (90 days) by default (cmd/circula/config.go).
- HSTS set when `PublicHTTPS` or TLS or secure (`httpx.Edge.HSTS`), `max-age=31536000; includeSubDomains`.

### 2.4 Gaps, ranked

1. **Secure mode still accepts the plain cookie name (Medium).** `httpx.SessionToken` reads `circula_session` even when the deployment is secure. A sibling subdomain (or an HTTP-level attacker) can set `circula_session` for the host, because non-prefixed cookies are not protected from that. The server takes `__Host-circula_session` first, so the planted value matters whenever no `__Host-` session cookie is present (before sign-in, or after logout clears it). The planted token then makes the victim's browser act as the attacker's account. Fix: when `ForceSecure` is true, read only `__Host-circula_session`.
2. **Long sessions for a ledger app (Medium).** 30 days idle and 90 days absolute is long for a system that moves credit. Recommend a shorter web idle timeout and a shorter absolute lifetime, plus step-up re-authentication before accepting a proposal, creating an adjustment, or changing roles. These are engineering recommendations, not sourced.
3. **CSRF cookie not `__Host-` (Low).** `cm_csrf` is a plain name. The audit notes that an attacker who can plant cookies can pair their own cookie and token. Mitigated by SameSite=Lax and the Origin check. Rename to `__Host-cm_csrf` in secure mode.
4. **Missing Origin accepted (Low).** `sameOrigin` returns true when `Origin` is absent. Acceptable with `SameSite=Lax` and `Sec-Fetch-Site`, but note it.
5. **API tokens survive password change (Low).** Documented in `ChangePassword`. A stolen bearer token outlives the password reset. Offer a "revoke tokens too" option, or revoke by default.
6. **Login and logout not audited (Low).** Session creation and failed logins are not in the audit chain (LED-6 note, CODE-AUDIT).

### 2.5 Cheapest high-value fix

In `httpx.SessionToken`, accept only `SessionCookieSecure` when secure is configured (one function change plus a test). Then rename `cm_csrf` to `__Host-cm_csrf` in secure mode. Lower `session-idle` default for the web UI.

---

## 3. Password storage, login throttling, enumeration, breached passwords

### 3.1 Top risks

- Offline cracking of a stolen database (password hashes, the backup in particular).
- Online guessing, including parallel guessing.
- Account lockout used as denial of service against a named user.
- Account enumeration through login, sign-up, invitation or timing.
- Breached or common passwords accepted.

### 3.2 Recommended controls

- Argon2id with parameters at or above the OWASP minimum, and as RFC 9106 recommends. OWASP's current Argon2id minimum is 19 MiB memory, 2 iterations, 1 degree of parallelism (snippet, https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html; older copies quoted 15 MiB, superseded). RFC 9106 first recommended option is t=1, p=4, m=2 GiB; second is t=3, p=4, m=64 MiB (values from secondary snippets, e.g. argon2-cffi profiles and OWASP CheatSheetSeries issue #1183, https://github.com/OWASP/CheatSheetSeries/issues/1183; RFC 9106 text https://www.rfc-editor.org/rfc/rfc9106.html not fetched).
- Per-hash salt, parameters stored in the hash string, constant-time compare, dummy hash on unknown login.
- Passwords: long minimum, long maximum (ASVS 4.0 V2.1.2 requires that passwords of 64+ characters be accepted; snippet, https://owasp-aasvs4.readthedocs.io/en/latest/2.1.2.html). No composition rules.
- Breached-password screening. HIBP range API: SHA-1 locally, send only the first 5 hex characters, compare suffixes locally. Add-Padding is optional (snippets from secondary docs; official https://haveibeenpwned.com/API/v3 not fetched).
- NIST SP 800-63B: rate-limit failed authentication, check against breached lists, no periodic rotation without compromise. The 800-63B-4 text (https://pages.nist.gov/800-63-4/sp800-63b.html) was NOT reachable. Search returned the 800-63-3 implementation notes only, so the 800-63B-4 minimum length figure is not confirmed here. Do not rely on any number from this file for 800-63B-4 without reading it.
- Throttling that is not a DoS. Lock policies should slow guessing without letting an attacker hold a named account shut indefinitely. Use per-(IP, login) budgets, a global per-login budget with a cap, and an operator or self-service unlock path.

### 3.3 What Circula does

- Hash: Argon2id, `DefaultArgon2Params` = m 64 MiB, t 3, p 2, salt 16 B, key 32 B (service/password.go). Meets the OWASP minimum with margin. Parallelism p=2 is below the RFC 9106 second option (p=4); memory is the dominant cost, so this is minor.
- Stored PHC string with params; `verifyPassword` refuses memory above 2^21 KiB (2 GiB) and checks PHC shape. Constant-time compare.
- Unknown login: `verifyPassword(password, s.dummyHash)`, so timing is equalised (service/auth.go `Login`).
- Password rules: minimum `min-password-length` (default 12, not below 12), maximum 1024 bytes, no composition rules (`validatePassword`).
- Service throttle (per login string): five failures, then 15 min, doubling per failure to 24 h. Counter reserved in the same write unit before verification (parallel guesses counted; SA-04 fixed).
- Web throttle: `ipFail` 20/min per IP, `acctFail` 5/min per (IP, login) on sign-in (web/web.go, h_auth.go). Password-change attempts have their own `pwFail` (5 per 10 min, h_account.go).
- Enumeration: login returns one message for unknown login and wrong password. Sign-up for an existing login returns the same success path with a decoy person and a platform audit event (`ActRegisterDuplicate`); the CLI may set `RevealDuplicate`.
- Breached check: `PasswordDenied` hook exists but is never assigned (verified by grep). NOT active.
- Account recovery: none. Operator has no unlock command for `login_throttle` (grep for unlock or throttle outside the service finds only purge).

### 3.4 Gaps, ranked

1. **Per-login lockout can hold a named account shut (High).** `Login` refuses before counting while locked. When the lock expires, one wrong guess increments the counter and sets the next lock (doubling up to 24 h). So one request per window keeps the victim locked, from any IP, with no unlock path. Verified by reading service/auth.go (`lockFor`, `Login`). This is a new finding not in SECURITY-AUDIT.md. Fix options: key the lock on (IP, login) for the increasing part and keep a much smaller global per-login cap (for example 15 min flat); add a CAPTCHA or step-up after N failures; add an operator unlock CLI (`circula admin unlock <login>`) and self-service recovery (section 4).
2. **Breached-password check unwired (Medium).** Add a HIBP range client or an offline denylist, set `cfg.PasswordDenied`, and decide explicitly whether an outage fails open or closed. Offline denylist is simpler to operate.
3. **No recovery path (High, shared with section 4).** A forgotten password means an operator edit of the database.
4. **Argon2 cost is undocumented as benchmarked (Low).** DEC-006 benchmark is not recorded in the repo (SECURITY-AUDIT note). Record it.
5. **Hash parameters accepted from storage without upper bound check beyond 2 GiB (Low).** Only a DB-write attacker can change them; acceptable, but a DoS vector if the DB is writable by another tenant process.
6. **Registration and invitation rate limits (Low).** `reg` is 10/min per IP (web). SEC-006 asks for limits on invitation redemption and join requests; not re-checked.

### 3.5 Cheapest high-value fix

Replace the flat exponential-to-24 h login lock with: lock keyed on (IP, login) for increasing delays, and a global per-login cap of 15 minutes that cannot grow while locked. Add `circula admin unlock <login>`. Assign `PasswordDenied` from an embedded denylist of the most common passwords (for example, top 10k), reviewed in the release notes.

---

## 4. TOTP MFA (RFC 6238) and recovery codes

### 4.1 Top risks

- Phishing and password reuse against a money-moving account.
- Replay of a valid one-time code within its time window.
- Seeds stolen from the database, letting an attacker generate codes forever.
- Recovery codes bypassing MFA or surviving reset.
- Clock drift causing lockouts, pushing operators to widen the window.

### 4.2 Recommended controls

- RFC 6238: TOTP is HOTP over a time counter, default step 30 s (RFC 6238 text not fetched; secondary: https://oneuptime.com/blog/post/2026-08-29-how-to-block-reuse-of-a-totp-code-during-its-30-second-validity-window/view). RFC 4226 (HOTP) is the base; its recommended key length of 160 bits is from memory, not checked against the RFC in this run.
- Replay: a verifier must not accept the same OTP twice. Store the last accepted time step per credential and reject any step less than or equal to it (secondary snippet; RFC 6238 section 5.2 per the same blog, not checked).
- Window: accept the current step and at most one step either side. Every extra step doubles the online guess surface (secondary).
- Seed: encrypt at rest, with the key outside the database. Otherwise a DB read gives every seed.
- Rate limit verification attempts per account and per IP.
- Enrolment: show seed once, require a successful code before enabling, and show it again only through re-authentication.
- Recovery codes: generate 8 to 10 random codes of at least 80 bits each. Store only keyed hashes (HMAC-SHA-256 with a server key, or Argon2id). Consume atomically (`UPDATE ... WHERE used_at IS NULL` and check rows affected). Regenerate invalidates the old set. Each use is audited and notified. Using a recovery code must not let the same session disable MFA without a fresh TOTP or second code.
- MFA reset by an operator must follow dual control (the spec's SEC-011 concept; not implemented).

### 4.3 What Circula does

- Nothing. grep for totp, mfa, recovery code and otp finds no code path. SECURITY-AUDIT.md confirms (SEC-007, P1).
- Tokens (API bearer) are not an MFA substitute and are not described as one.

### 4.4 Gaps, ranked

1. **No second factor on a credit ledger (High).** Any password compromise gives full control of a member's account, including proposing and accepting trades.
2. **No recovery at all (High).** Shared with section 3.
3. **No step-up for sensitive actions (Medium).** Admin role grants, adjustments, policy changes need re-auth even without MFA.

### 4.5 Cheapest high-value fix

Implement TOTP with the replay check stored per credential, encrypted seed, and one-time recovery codes as above. Start with admin accounts (roles, adjustments, policy). Use `github.com/pquerna/otp` or equivalent only after reviewing it; not searched in this run.

---

## 5. Tamper-evident audit logs

### 5.1 Top risks

- Editing history by someone with DB write access.
- Truncating the newest events (tail truncation).
- Dropping triggers, then rewriting the chain.
- Audit records lost in a restore or a backup swap.
- Missing events for sensitive actions (failed logins, lockouts, MFA changes).

### 5.2 Recommended controls

- Hash chain over canonical event content and the previous hash (already done).
- Keyed chain (HMAC) with the key outside the database. Without it, anyone who can write the DB can recompute the whole chain. Chaining alone detects only mid-chain edits by parties who do not recompute.
- External anchoring of the head: publish the latest sequence and hash to storage the DB writer cannot change (append-only object storage with retention lock, a remote syslog, or an operator log stream), and check the head against it at startup and in verification.
- Append-only enforcement in the DB (triggers) as a defence, not as the control, because a DB-write attacker can drop triggers.
- Time-stamping (RFC 3161) is an option if independent time evidence is needed. Not searched.

### 5.3 What Circula does

- `service/audit.go` `AuditHash`: SHA-256 over prev hash, id, network, seq, occurred-at ms, actor, action, target, request id, idem ref, summaries, reason and sorted subjects. Separators are unit separators (0x1f). Unkeyed.
- Migration 0001: UNIQUE `(network_id, seq)`; triggers forbid UPDATE and DELETE on `audit_events`, `audit_subjects`, status events, dispute events, consistency runs, policies, and role deletes.
- Migration 0002: `audit_heads` mirrors the newest sequence and hash per scope in the same unit. Triggers forbid deleting heads and forbid moving a head backwards or changing network.
- `VerifyLedger` (service/verify.go) checks the chain, including head against chain end. Runs at start-up, hourly, on demand, and on backup.
- Backups carry the ledger summary in the manifest (`LedgerSummary`).

### 5.4 Gaps, ranked

1. **Unkeyed chain and same-DB head (Medium).** An attacker with file write can: drop triggers, delete the tail, update `audit_heads`, and recompute hashes. `audit_heads` raises the cost but does not prevent it. This is the SA-09 residual; the head fix is partial.
2. **No external anchoring (Medium).** The head is never published. Nothing outside the DB records the last hash.
3. **Backups are plaintext (Medium).** Restoring a tampered backup yields a chain that verifies unless `backup-key` is set (section 6). A backup swap is therefore an undetected rollback unless the head is anchored.
4. **Coverage (Low).** Login success and failure, lockout, session revocation, MFA (absent), and API token creation or revocation are not in the audit chain (LED-6 notes; verify token events not re-checked).

### 5.5 Cheapest high-value fix

At each `VerifyLedger` run (and at each backup), write the head `(network, seq, hash)` to a sink the DB writer cannot change: a structured log line shipped to a remote collector, or a file in a directory with a different owner and append-only attribute. Then key the chain with an HMAC secret read from `backup-key-file` style storage, applied to new events only, with a version marker.

---

## 6. Backup, restore integrity, encryption

### 6.1 Top risks

- Backup file read by someone with storage access: password hashes, session and token hashes, all personal data, all ledgers.
- Backup substituted (credential swap, role insertion, session restoration).
- Restore of a stale backup silently rolling back the ledger.
- Backup key leaked along with backups.
- Restore run against the wrong database, destroying data (operator error).

### 6.2 Recommended controls

- Encryption at rest with authenticated encryption. For files, `age` (X25519 recipients; ChaCha20-Poly1305 payload) is a good fit: the C2SP spec describes X25519 recipient stanzas, HKDF-SHA-256 and ChaCha20-Poly1305 (snippet, https://github.com/C2SP/C2SP/blob/main/age.md; the scrypt passphrase section was not confirmed in the snippet). AES-256-GCM is the alternative; nonces must never repeat under one key, so use random 96-bit nonces with a counter-limit, or use `age`.
- Signed manifest (HMAC or signature) over file digest, size, schema and time, key held outside backup storage.
- Restore checks: digest, signature, schema version, `PRAGMA integrity_check`, ledger verification, and a dry-run into a scratch file.
- Restore revokes sessions and API tokens found in the backup, or requires an explicit flag to keep them.
- Retention with separate retention for offsite copies, and restore drills.

### 6.3 What Circula does

- `VACUUM INTO` snapshot (cmd/circula/snapshot.go, the same mechanism the SA-06 audit note cites at snapshot.go:98).
- Manifest: format, time, app and schema version, size, SHA-256, `integrity_check` result, ledger summary, optional `hmac_sha256` (snapshot.go). `manifestMAC` covers format, time, schema, size and digest, not the file name.
- Restore (cmd/circula/commands.go `cmdRestore`): reads manifest, verifies digest, `checkManifestAuth`, refuses newer schema, takes pre-restore backup, requires `-overwrite`. Without `backup-key`, prints an "unauthenticated" warning (commands.go:332).
- Files 0600; backup directory 0700 (`ensurePrivateDir`); umask 077 for the process.
- Scheduled backups in serve.go with retention (`backup-retain-daily`, `backup-retain-monthly`).
- No encryption of backup files anywhere (grep for age, aes, gcm, cipher finds none in product code).

### 6.4 Gaps, ranked

1. **No encryption at rest for backups or the live DB (High for multi-tenant custody).** Permissions are the only protection. Anyone who copies the file gets password hashes to crack offline.
2. **Backup authentication is optional (Medium-High).** Without `backup-key` a substituted backup restores with a warning. Refuse scheduled backups when no key is configured, and refuse restore of an unsigned manifest by default (flag to override).
3. **Restore does not revoke carried sessions and tokens (Medium).** Not verified in code this run; SECURITY-AUDIT SA-07 recommended it.
4. **Restore drill not automated (Low).**

### 6.5 Cheapest high-value fix

Make `backup-key-file` mandatory when `backup-interval` is not 0 (refuse start, like `checkTransport`). Then encrypt each backup with `age` to an operator recipient (public key in config), and keep the HMAC over the encrypted file. Restore requires the private identity.

---

## 7. Multi-tenant isolation and IDOR

### 7.1 Top risks

- Member reads or writes another network's accounts, transactions, members or exports by changing an identifier.
- Mass assignment of `network_id` or `person_id`.
- Token scope escalation: a bearer token for one network used on another.
- Enumeration of networks or members through distinct responses.
- Idempotency key collision across tenants or users.

### 7.2 Recommended controls

- Every tenant row carries `network_id`. Every query filters on it. Every foreign key is composite `(network_id, id)`, so cross-tenant references fail at the DB level.
- Authorization on each request, server-side, from membership, role and status. Return 404 for non-members, the same as for unknown IDs.
- Ownership checks on revoke and read (`WHERE id = ? AND person_id = ?`).
- Idempotency keys scoped by principal and network.
- Tests: an authorization matrix across roles and networks.

### 7.3 What Circula does

- Schema: composite foreign keys across tenant tables (e.g. `FOREIGN KEY (network_id, payer_account_id) REFERENCES accounts(network_id, id)`, transactions, entries, disputes, members, listings). Strong.
- Web: `netH` resolves the slug, requires membership, returns 404 (`domain.NotFound("network")`) for non-members (web/session.go).
- API: `resolveNetwork` (api/auth.go) filters memberships by the token's `NetworkScope`, requires `X-Network` when a person belongs to several networks, returns 404 for non-membership.
- Tokens: the revoke statement (store/sqlite_person.go) filters on `person_id` as well as `id`. Token scopes and optional network binding (`api_tokens.network_id`).
- Service methods take `netID` (`GetAccount(net.ID, ...)`, `CreateAdjustment(ctx, actor, netID, ...)`).
- Idempotency keys: service scope is `(network, actor, op, key)`; API scope is `actor + key` plus method, path, network and body fingerprint.
- Tests: service/authz_test.go, api/authn_test.go, api/hardening_test.go, web/security_test.go.
- Not verified this run: every store method on tenant tables takes `netID`. Some `WHERE id = ?` queries exist for global tables (persons, sessions, api tokens, invitations). Invitations and sessions look correct by design.

### 7.4 Gaps, ranked

1. **No static guard that tenant queries include `network_id` (Medium).** Relies on review and tests. Add a lint or a test that scans `store/*.go` for `WHERE id = ?` on tenant tables.
2. **Global-table ID lookups by bearer ID (Low).** `api_tokens`, `sessions`, `invitations` look up by unscoped ID or hash. Acceptable (IDs are random), but document.
3. **Enumeration of network slugs (Low).** Slugs are guessable and visible in URLs. Non-member and unknown look identical, so the risk is limited to slug discovery.

### 7.5 Cheapest high-value fix

Add a store test that enumerates tenant table names and fails if a query on them lacks `network_id`. Keep the composite FKs as the main line of defence.

---

## 8. Supply chain

### 8.1 Top risks

- Compromised action or tag executed in CI with a write token.
- Malicious or vulnerable Go module.
- Release artefact swapped after build, or built from a poisoned cache.
- Publishing a release without human approval after a tag push.
- No way for users to verify provenance.

### 8.2 Recommended controls

- Pin every action to a full commit SHA with a version comment. GitHub recommends this and offers org policy to enforce it (GitHub changelog, https://github.blog/changelog/2025-08-15-github-actions-policy-now-supports-blocking-and-sha-pinning-actions/, snippet). Pair with Dependabot for the github-actions ecosystem, otherwise pins go stale.
- Least-privilege `permissions`, `persist-credentials: false`.
- Release gated by an environment with required reviewers.
- Signing of release artefacts (Sigstore keyless via GitHub OIDC, `cosign sign-blob`, identity pinned to the workflow). Verification pins the OIDC issuer and workflow identity (secondary guides only; sigstore.dev and slsa.dev docs not reached).
- Build provenance attestation (`actions/attest-build-provenance`), verified with `gh attestation verify`. Secondary sources only.
- SBOM in a standard format (CycloneDX or SPDX). `cyclonedx-gomod` produces CycloneDX for Go modules (name from memory; cyclonedx-gomod README not fetched).
- Vulnerability scan on release inputs (`govulncheck`).

### 8.3 What Circula does

- Actions pinned to 40-hex SHAs with version comments, enforced by `scripts/check-action-pins.sh` (allowlist of five actions).
- Top-level `permissions: contents: read` (CI) and `permissions: {}` (release), job-level `contents: write` only in publish.
- `persist-credentials: false` on checkout everywhere.
- Go build cache disabled in release (`cache: false`).
- Tag format check, release notes gate (`docs/releases/<tag>.md`).
- Release build: `CGO_ENABLED=0`, `-trimpath`, ldflags for version, commit and date (`scripts/package-release.sh`).
- `SHA256SUMS` generated and verified in publish job before `gh release create`.
- SBOM: `go list -m all` text file only (`go-modules-<version>.txt`). Not CycloneDX or SPDX; no hashes.
- Dependabot weekly for actions and gomod (.github/dependabot.yml).
- CI runs `govulncheck@v1.1.4` in the lint job (ci.yml).
- `go mod verify` and default checksum database: not re-run here; SECURITY-AUDIT said verified.

### 8.4 Gaps, ranked

1. **Publish job has no approval gate (High).** `release.yml` publish job runs on any `v*` tag push with `contents: write`, no `environment`, no reviewer. Anyone who can push a tag can publish a release. Tag protection rules are a repo setting, not verifiable from the repo.
2. **No signing or provenance (Medium-High for a downloadable binary).** `SHA256SUMS` is in the same release it protects. Users cannot verify who built it.
3. **SBOM is not a standard SBOM (Medium).**
4. **Govulncheck only in lint job on push or PR (Low).** Not run on the release artefacts or scheduled.
5. **Go toolchain taken from `go.mod` without a hash pin (Low).** Release build cache is off (fixed); the toolchain itself is not verified by digest.

### 8.5 Cheapest high-value fix

Put the publish job behind an `environment: release` with required reviewers. Add `actions/attest-build-provenance` (with `id-token: write`, `attestations: write` only in that job) over `dist/*`. Replace the module list with `cyclonedx-gomod app -json` output. Sign `SHA256SUMS` with `cosign sign-blob --bundle`.

---

## 9. Rate limiting behind proxies

### 9.1 Top risks

- Every real client behind one reverse-proxy address shares one budget (self-DoS on login).
- Spoofed `X-Forwarded-For` bypassing limits.
- Attacker rotating IPv6 addresses to evade per-IP limits.
- Limiter memory growth under attack.
- Metrics or admin endpoints exposed through proxy misconfiguration.

### 9.2 Recommended controls

- Trust `X-Forwarded-For` only from configured proxy CIDRs. Walk the chain from the right and take the first address that is not a trusted proxy. Treat a malformed chain as untrusted (de facto practice; RFC 7239 `Forwarded` not fetched).
- Key IPv6 limits on /64 or wider, not on the full address.
- Separate budgets: per IP, per account, per (IP, account), and global.
- Count failures for login, not successes (SA-05 note).
- Metrics on a separate listener or with a token.

### 9.3 What Circula does

- `httpx.Edge.ClientIP`: XFF only when the TCP peer is in `trusted-proxies`; right-most untrusted entry; malformed chain returns the peer. Unit tests in httpx_test.go (not read this run).
- Default `trusted-proxies none` (config.go), so headers are ignored: safe against spoofing, but behind a proxy every client is the proxy IP.
- Web limiters: `ipFail` 20/min, `acctFail` 5/min per (IP, login), `pwFail` 5/10 min per account, `reg` 10/min per IP, `setup` 10/10 min (web/web.go). Failure-only counting for login (`fail` and `blocked`).
- API: `IPPerMinute` (per client IP) and `PrincipalPerMinute` (per person), in-memory per process (api/middleware.go, route.go). Default values in api/api.go and config.
- `metricsAccess` refuses forwarding headers and trusted proxies, optional bearer token (ops_http.go).
- Limiters are in memory and per process. Single-process by design (serve lock).

### 9.4 Gaps, ranked

1. **Unsafe default behind a proxy (Medium-High, operational).** With default `trusted-proxies none` and the documented TLS-proxy deployment, all users share the proxy's IP and therefore the 20/min login IP budget and the 10/min registration budget. A few attackers can lock every login. Fix: refuse start when `public-base-url` is https and `trusted-proxies` is empty, or warn loudly, and document the proxy CIDR setting in docs/ops/configuration.md.
2. **Broad `trusted-proxies` ranges enable spoofing (Medium if misconfigured).** If an operator trusts a whole private range that includes clients, XFF is believed. Document that CIDRs must contain only proxies.
3. **IPv6 per-address limits (Low-Medium).** Key on /64.
4. **Unbounded limiter map growth (Low).** Sweeps exist (web limiter per window; API per minute), but growth under many distinct keys is not capped.
5. **Login per-account budget is per (IP, login) (Low).** Combined with section 3.4, a distributed attacker can still hit the service-level per-login lock. This is the real DoS path, not the IP limiter.

### 9.5 Cheapest high-value fix

Startup check: if `public-base-url` is https and `trusted-proxies` is `none`, log a warning (or refuse, with `allow-direct` override). Add `/64` keying for IPv6 in `ClientIP` callers. Document `trusted-proxies` as proxy-only.

---

## 10. Prioritised action list

Ranked by risk reduced per unit of effort. Each item is a single change, except where noted.

1. **Fix per-login lockout DoS (section 3.4).** Key increasing delay on (IP, login), cap the per-login lock at 15 min, do not let it grow while locked. Add `circula admin unlock <login>`. Test: sustained wrong guesses from one IP do not extend the lock past the cap.
2. **Set `synchronous(FULL)` for the ledger (section 1.4).** One pragma; benchmark; record DEC.
3. **Read only `__Host-circula_session` in secure mode (section 2.4).** One function change plus test. Rename `cm_csrf` to `__Host-cm_csrf` in secure mode.
4. **Make backup signing mandatory for scheduled backups and restore (section 6.4).** Refuse start without `backup-key-file` when backups are on; refuse unsigned restore unless overridden.
5. **Encrypt backups with age (section 6.5).** Operator recipient in config; keep HMAC over the ciphertext.
6. **Implement TOTP with replay state, encrypted seed and recovery codes, starting with admin roles (section 4.5).** Largest single gap for a credit ledger.
7. **Account recovery path (sections 3.3, 4.4).** Operator-assisted reset with dual control first; self-service later.
8. **Wire the breached-password check (section 3.4 item 2).** Offline denylist first; HIBP range later if outbound access is acceptable.
9. **Externally anchor the audit head (section 5.5).** Ship `(network, seq, hash)` at each verify and backup to a sink the DB writer cannot change.
10. **Proxy safety default (section 9.5).** Warn or refuse on https public URL with no `trusted-proxies`; document. Key IPv6 on /64.
11. **Environment approval on the release publish job (section 8.5).** Required reviewer; tag protection in repo settings (outside the repo, check manually).
12. **Provenance and signed checksums for releases (section 8.5).** `attest-build-provenance`, `cosign sign-blob` on `SHA256SUMS`.
13. **Shorten web session lifetime and add step-up for admin and money-moving actions (section 2.4).** Config change plus re-auth prompt.
14. **Move full-table verification off the write slot (section 1.4 item 3).** Chunk the scan; keep hourly cadence.
15. **Store test that tenant queries include `network_id` (section 7.5), and CycloneDX SBOM (section 8.5).** Low effort, guards future regressions.

Items 1-5 can each be done without new dependencies. Item 6 and 7 need design work.

---

## 11. Sources

All accessed 2026-10-21. "Snippet" means text returned by WebSearch; the page itself was not fetched. "Not reachable" means WebFetch or curl failed.

| Topic | URL | Status |
|---|---|---|
| OWASP Password Storage CS | https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html | Snippet (19 MiB, t=2, p=1). Direct fetch not reachable (DNS). |
| RFC 9106 Argon2 | https://www.rfc-editor.org/rfc/rfc9106.html | Not reachable (DNS). Values from secondary snippets only. |
| OWASP CheatSheetSeries issue on Argon2 params | https://github.com/OWASP/CheatSheetSeries/issues/1183 | Snippet, secondary. |
| NIST SP 800-63B-4 | https://pages.nist.gov/800-63-4/sp800-63b.html | Not reachable (DNS). Search returned 800-63-3 implementation notes only (https://pages.nist.gov/800-63-3-Implementation-Resources/63B/Authenticators/, snippet). 800-63B-4 figures NOT confirmed. |
| OWASP ASVS 4.0 V2.1.2 | https://owasp-aasvs4.readthedocs.io/en/latest/2.1.2.html | Snippet, 4.0 text. ASVS 5.0 numbering not confirmed. |
| RFC 6238 TOTP | https://www.rfc-editor.org/rfc/rfc6238.html | Not reachable. Secondary: https://oneuptime.com/blog/post/2026-08-29-how-to-block-reuse-of-a-totp-code-during-its-30-second-validity-window/view (snippet). |
| RFC 4226 HOTP | https://www.rfc-editor.org/rfc/rfc4226.html | Not fetched. 160-bit key recommendation from memory, unverified. |
| RFC 6265bis cookie prefixes | https://datatracker.ietf.org/doc/draft-ietf-httpbis-rfc6265bis/ | Not fetched. Prefix rule via tough-cookie README snippet (https://github.com/salesforce/tough-cookie). |
| OWASP CSRF Prevention CS | https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html | Snippet. |
| OWASP Session Management CS | https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html | Not reached; no sourced numbers used. |
| SQLite WAL | https://www.sqlite.org/wal.html | Not reachable (DNS). |
| SQLite synchronous pragma source text | https://www.sqlite.org/docsrc/info/bcf687da9c172292 | Snippet of official docsrc text. |
| SQLite transactions | https://www.sqlite.org/lang_transaction.html | Not reachable. Mirror: https://system.data.sqlite.org/home/raw/124adce1a34b456ca46ca6839bf145ce55753b3a (snippet). |
| SQLite busy handling, lock upgrade | https://www.simonwillison.net/2025/Feb/17/sqlite-busy | Snippet, secondary. |
| HIBP Pwned Passwords range API | https://haveibeenpwned.com/API/v3 | Not fetched. Secondary: https://orthogonal.info/check-breached-password-without-sending-hibp-k-anonymity/ (snippet). |
| C2SP age spec | https://github.com/C2SP/C2SP/blob/main/age.md | Snippet, partial (X25519, HKDF-SHA-256, ChaCha20-Poly1305). scrypt section not confirmed. |
| SLSA v1.0 | https://slsa.dev/spec/v1.0/ | Not reached. Secondary guide only. |
| Sigstore cosign | https://docs.sigstore.dev/cosign/signing/overview | Not reached. Secondary guide only. |
| GitHub Actions SHA pinning policy | https://github.blog/changelog/2025-08-15-github-actions-policy-now-supports-blocking-and-sha-pinning-actions/ | Snippet, official GitHub changelog. |
| cyclonedx-gomod | https://github.com/CycloneDX/cyclonedx-gomod | Not fetched. Name from memory. |
| Go security and module verification | https://go.dev/doc/security/vuln/ and https://go.dev/ref/mod#checksum-database | Not searched in this run. Not cited. |
| RFC 7239 Forwarded | https://www.rfc-editor.org/rfc/rfc7239 | Not reachable. |

Repo files cited: internal/store/sqlite.go, internal/store/migrate.go, internal/store/migrations/0001_init.sql, 0002_hardening.sql, internal/service/{auth,password,posting,idempotency,audit,verify,purge,service,transactions}.go, internal/web/{session,h_auth,web,ratelimit,middleware,h_account}.go, internal/api/{auth,middleware,route,idem,api,h_password}.go, internal/httpx/httpx.go, cmd/circula/{config,serve,commands,snapshot,ops_http,umask_unix,lock_unix}.go, .github/workflows/{ci,release}.yml, .github/dependabot.yml, scripts/{package-release,check-action-pins}.sh, docs/audit/{SECURITY,CODE}-AUDIT.md.
