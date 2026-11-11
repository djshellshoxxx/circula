# Security audit: Circula (branch claude/circula-build)

Date: 2026-10-21. Scope: docs/specs/09-security-privacy.md threats T-01..T-23. Method: code review plus throwaway
exploit tests run on a COPY of the repo (scratchpad `.../scratchpad/c`, files `zz_xp_test.go`, `cmd/xp`). Repo code was
not modified; this file is the only change. Only findings confirmed by a test run or by a direct code/CLI observation are listed.
Test labels XPn below refer to those throwaway tests.

## Not verified / unavailable

- `govulncheck` could not run (vuln.go.dev returned 403 from the sandbox proxy). Dependency vulns: NOT assessed. `go mod verify`: all modules verified.
  `go list -m all` direct deps: x/crypto v0.38.0, modernc.org/sqlite v1.38.2, google/uuid v1.6.0. CI runs govulncheck@v1.1.4 (ci.yml:46); rely on it.
- Browser behaviour (SA-01 redirect resolution, cookie handling behind a real proxy) is argued from the URL spec / header output, not run in a browser.
- No real TLS proxy, no race detector run on exploit tests, no fuzzing, no load test with production Argon2 params.

## Verified OK (no finding)

- Argon2id, PHC string with params, 16-byte salt, constant-time compare, dummy hash for unknown logins (password.go, auth.go:93-135).
- Session and API tokens: 256-bit random, only SHA-256 stored, revoked on logout/password change (auth.go:138-190).
- Tenant isolation: XP12 (cross-network GET tx/member via X-Network, token scoped, export by other member, mass assignment, bearer creating token,
  member hitting admin routes) all 404/403/400. Existing authz matrix test passes. Idempotency keys are scoped per user (XP12: same key, two users, two results).
- SQL injection: every store query is parameterised; only `sumProposed` concatenates a column name from two internal constants (sqlite_account.go:64).
- XSS: html/template everywhere; only `template.HTML` is fmtTime with formatted time (render.go:117). XP10: script/img payload in listing rendered escaped. CSP `default-src 'none'; script-src 'self'; frame-ancestors 'none'` present on web.
- CSRF (web): XP11 no token 403, valid token + foreign Origin 403. Double submit is HMAC signed.
- Overspend race: XP2, 30 parallel proposals of 30.00 against limit 100.00 -> exactly 3 accepted; double parallel accepts -> no overspend. Write gate + BEGIN IMMEDIATE works.
- Rate limit spoofing: XP7, rotating X-Forwarded-For does not bypass the limiter (header is ignored). No trusted-proxy parsing anywhere.
- Export: CSV formula neutralisation, single-network queries, no credential columns (sqlite_export.go). Export download owner-bound (XP12).
- Audit immutability by triggers: UPDATE on audit_events refused (XP4). Edited event detected by VerifyLedger.
- Backup/restore path handling: names derived from timestamps; restore takes explicit operator path; no traversal from untrusted input. Lock file 0600, manifest 0600, scheduled backup dir 0700.
- Workflows: top-level `permissions` least privilege, `persist-credentials: false`, tag regex check, checksum verify before release.

## Findings

### SA-01 Medium: open redirect after sign-in via TAB / VT in `next` (T-02/T-07 area, SEC-024)
- File: internal/web/session.go:136-141 (`safeNext`), used at h_auth.go:34,44,52-56,92,96.
- Evidence (XP5): POST /login with valid credentials and `next=/%09/evil.example` returns `303 Location: "/\t/evil.example"`.
  Same for `/\v/evil.example` and `/ /evil.example`. `//x` and `/\x` are correctly dropped. `safeNext` blocks only CR/LF.
  Browsers strip TAB/LF/CR inside URLs, so `/<TAB>/evil.example` resolves to `//evil.example` (protocol-relative). GET `/login?next=` and `/register?next=` feed the same value.
- Repro: send victim `https://host/login?next=%2F%09%2Fevil.example`; after login they land on evil.example.
- Fix: parse with url.Parse, require empty scheme+host, reject any byte < 0x20 or 0x7f or `\`, and require `path.Clean` result to start with a single `/`. Add test for `\t`, `\v`, `%09`.

### SA-02 High: session cookie not guaranteed Secure; `cookie-secure` setting is dead; no start-up refusal (T-01, SEC-022, SEC-003)
- Files: cmd/circula/serve.go:155 (`web.NewHandler(svc)` with zero Options), cmd/circula/config.go:57 ("reserved until the web adapter reads it"),
  internal/web/session.go:26-28, internal/web/web.go:24-26 (ForceSecure never set), internal/api/middleware.go:42 (HSTS only if r.TLS != nil).
- Evidence (XP8): handler built as in serve.go, plain HTTP request -> `Set-Cookie cm_csrf Secure=false`; no Strict-Transport-Security on web responses.
  CLI: `CIRCULA_PUBLIC_BASE_URL=http://example.com CIRCULA_LISTEN=0.0.0.0:8080 CIRCULA_COOKIE_SECURE=false circula serve -check-config` exits 0, accepted.
  docs/ops/README.md:28 tells operators to use a TLS-terminating proxy but not to send `X-Forwarded-Proto: https`; without it the session cookie `cm_session` is sent without Secure over any HTTP hop. TLS terminates at the proxy, so r.TLS is always nil.
- Impact: session theft on any plain-HTTP request to the host (downgrade, mixed link). SEC-022 acceptance test (refuse start) fails.
- Fix: wire `Options.ForceSecure = cfg.CookieSecure`, refuse start when public-base-url is http and listen is non-loopback unless explicit override; add HSTS when PublicBaseURL is https; document X-Forwarded-Proto.

### SA-03 Medium: any self-registered account can create a network; SetupSecret never wired (SEC-014/T-18, FR-002)
- Files: internal/web/web.go:21-23, internal/web/h_auth.go:136,142,147, cmd/circula/serve.go:155. `/register` is open (h_auth.go:15-16).
- Evidence (XP9): fresh `/register` then `/setup` (open join mode, signup_grant 1000.00) -> network created, user is admin, `/n/mine/` served. No config key sets the secret.
- Impact: unlimited network creation by anonymous internet users (storage, Sybil operation, hosting of abusive content on the operator's domain). Not a cross-tenant breach.
- Fix: add `setup-secret` (from file/env, SEC-027) and pass it; or gate network creation on an operator-created platform role. Compare with subtle.ConstantTimeCompare (h_auth.go:147 uses `!=`).

### SA-04 Medium: login lockout bypass by parallel guesses (T-01, SEC-006)
- File: internal/service/auth.go:96-128. Failure count is read in one View tx (line 99), password verified outside any tx (113-117), then written as `failures+1` from the stale read (126).
- Evidence (XP1): 40 parallel wrong guesses on one account -> 40 evaluated, 0 throttled (sequential baseline: 5 then throttled). XP1b: 5 bursts of 40 = counter reaches only 5 and ~80 guesses beyond lock were still evaluated; each burst adds +1 to the counter regardless of size.
- Impact: the per-account limit does not bound guesses; only the per-IP web limiter (10/min, see SA-05) remains, and that is per proxy address.
- Fix: do read-check-increment atomically in one Update tx before verifying (reserve an attempt), or increment first then reset on success. Add a parallel test.

### SA-05 Medium: rate limiters key on RemoteAddr only; behind a reverse proxy one client throttles everyone (T-13, SEC-006)
- Files: internal/web/ratelimit.go:50-55 (`remoteIP`), web.go:51 (`newLimiter(10, 60)`), internal/api/middleware.go:92-98 (`clientIP`), api.go:62-64 (1200/min).
- Evidence (XP7): 12 wrong logins from one connection -> 429; a legitimate correct login for another user from the same RemoteAddr then also gets 429 (all clients look like the proxy IP). XFF is ignored (so no spoof bypass), but no trusted-proxy option exists.
  The limiter also counts successful logins.
- Impact: any internet user can lock all logins (10/min total) and, with 1200 requests/min, exhaust the API IP bucket for everyone. Real client IPs are never separated in the documented deployment.
- Fix: add a `trusted-proxies` CIDR setting and take the client IP from the right-most untrusted X-Forwarded-For entry only when RemoteAddr is trusted; count only failures for the login limiter.

### SA-06 Medium: database and manual backups are plaintext and world-readable (T-23, SEC-028)
- Files: internal/store/sqlite.go:33 (driver creates file with default umask), cmd/circula/snapshot.go:98 (`VACUUM INTO`), commands.go:150-160 (`backup <dest>`); no encryption anywhere.
- Evidence (CLI, umask 022): `circula migrate` -> `data/circula.db` mode 644; `circula backup $PWD/bk.db` -> `bk.db` mode 644 (manifest 600). Scheduled backups are only protected because the app creates `backups/` as 0700 (serve.go:91); if that dir pre-exists with wider mode the files are 644.
  Backup contains password hashes, session/token hashes, all PII and ledgers.
- Impact: any local user or backup custodian can read all tenants' data and offline-crack Argon2 hashes. SEC-028 acceptance "backup cannot be read without the key" fails.
- Fix: create DB and backups with 0600 (chmod after VACUUM INTO, set umask 077 in process), refuse a group/world-readable database dir, add age/AES-GCM encryption with key from env/file (SEC-027).

### SA-07 Medium: restore trusts a self-attested manifest; forged backup swaps credentials (T-04, T-22/T-23, SEC-028)
- Files: cmd/circula/snapshot.go:227-258 (manifest sha256 is stored next to the file), commands.go:189-262 (restore). Ledger verification covers only ledger tables (service/verify.go:21-30).
- Evidence (CLI): seed victim+attacker, backup, edit `persons.password_hash` of victim in bk.db to the attacker's hash, recompute sha256 in manifest, `circula restore -overwrite bk.db` -> exit 0, "no networks to verify"; then `Login(victim, attackers-known-password)` succeeds, victim's real password fails.
- Impact: anyone able to write to backup storage (custodian, shared bucket) gets full account takeover and can insert roles/sessions, with no detection. Checksum protects only against corruption.
- Fix: HMAC/sign manifest with an operator key kept outside the backup location; restore requires valid tag; also have restore refuse sessions and API tokens present in the backup (revoke all on restore) and print a diff of persons/roles counts.

### SA-08 Medium: unsolicited "request" proposals let a member freeze another member's credit (T-20 abuse, T-13)
- Files: internal/service/transactions.go:84-105 (payer capacity checked and reserved at proposal time, counterparty never consented), store/sqlite_account.go:62-68 (sum of proposed).
- Evidence (XP3): attacker with a plain account sends `Direction=request` proposals to a victim until the victim's reserved sum equals the victim's whole limit (4 requests of 25.00 vs 100.00). Victim Available=0; the victim's own 1.00 payment fails `insufficient-available-credit`. No cap on open proposals, no cost to attacker; lasts ProposalExpiryDays.
- Impact: denial of trading for any member at scale (script over directory).
- Fix: cap open outgoing proposals per proposer and per counterparty pair; reserve payer capacity only after the payer accepts for `request` direction; allow payer-side block list.

### SA-09 Low: audit chain does not detect tail truncation; chain is unkeyed (T-04, T-08, SEC-019)
- Files: internal/service/audit.go:93-100 (plain SHA-256 chain), service/verify.go:84- (`verifyAuditChain`).
- Evidence (XP4): with DB write access, DROP TRIGGER audit_no_delete, delete the newest audit event -> `VerifyLedger` returns OK=true, no findings. Mid-chain edits are detected; but an attacker can recompute the whole chain (no secret). Triggers are droppable by anyone with file access.
- Impact: tamper-evidence is partial; matches the stated limit in DAT-030 / SEC-034 (optional) but T-04 residual is understated for tail loss.
- Fix: store and externally publish the head hash/seq (metrics, backup manifest, operator log); verify against last published head; optionally HMAC with an operator key.

### SA-10 Low: registration reveals which logins exist (T-01/T-02, SEC-005)
- Files: internal/web/h_auth.go:122-127, internal/service/auth.go:52-55.
- Evidence (XP6): `/register` with an existing login -> 422 "That login is not available"; new login -> 200. Limited only by the 10/min IP limiter (SA-05 makes that weaker or stronger depending on proxy).
- Fix: generic response plus out-of-band confirmation, or accept as documented risk and rate-limit per IP+global; spec SEC-005 says sign-up responses must not reveal existence.

### SA-11 Low: export artefact store can be exhausted by any member (T-13/T-14)
- Files: internal/api/jobs.go:19-23,67-77; api/h_export.go:90-96.
- Evidence (XP13): one ordinary member creates 50 `mine` exports -> every subsequent export by any user in any network returns 503 for 24 h (the cap is global to the process, entries are kept in memory, TTL 24 h). Each attempt also writes an audit event (service/export.go:98-101).
- Fix: per-person and per-network quotas; evict by per-person LRU; do not audit before the artefact is accepted.

### SA-12 Low: expired sessions, throttle rows and idempotency rows are never purged (T-14)
- Evidence: no purge/delete job exists (`grep -rni "purge\|DeleteExpired"` finds none; jobs are expiry of proposals, verify, backup only: serve.go:206-221). `Login` inserts a session per login (auth.go:131) and a throttle row for every unknown login string (auth.go:126).
- Impact: unbounded DB growth by an attacker with many IPs; slow write-lock contention (T-14).
- Fix: scheduled delete of expired/revoked sessions, throttle rows older than 24 h, expired idempotency records; cap sessions per person.

### SA-13 Low: invitation token written to logs on internal error (T-11, SEC-020)
- File: internal/api/route.go:56 logs `r.URL.Path`; route `POST /api/v1/invitations/{token}/accept` (api/routes.go:57) carries a live credential in the path. Web `GET /join/{token}` is protected only because ops access log uses route templates (ops_http.go:88).
- Not triggered in a test (needs an unexpected error); finding is by code inspection.
- Fix: log the route pattern (`r.Pattern`), never the raw path.

### SA-14 Medium: supply chain hardening gaps (T-22, SEC-029)
- Files: .github/workflows/ci.yml:20-23,40,55-58,75-78,85; release.yml:20-23,52-56,63,80-83; scripts/check-action-pins.sh:30-34 (enforces `^v[0-9]+$`).
- Evidence: all actions pinned to mutable major tags, including third-party `golangci/golangci-lint-action@v8`; the pin script forbids SHAs by design (accepts only `@vN`). Release job uses `setup-go` `cache: true` (release.yml:56-59) so a cache from CI can feed the release build. Pushing any `v*` tag auto-publishes a non-draft release (release.yml:80-100) with no environment approval. SBOM is only a module list (package-release.sh:3); no provenance or signature (DEC-027 deferred); no CI test that a seeded vulnerable dep fails (SEC-029 criterion).
- Fix: pin every action to a full commit SHA (update script to require 40-hex), disable cache in release, protect tags and put the publish job behind an environment with required reviewers, generate CycloneDX/SPDX SBOM and `actions/attest-build-provenance`.

### SA-15 Low: /metrics gate relies on absence of forwarding headers (T-11)
- File: cmd/circula/ops_http.go:153-160. Default `metrics-allow` is loopback. A reverse proxy on the same host (nginx `proxy_pass` does not add X-Forwarded-For unless configured) that forwards `/metrics` presents RemoteAddr 127.0.0.1 with no XFF, so the check passes and the endpoint is public. Docs only say "do not forward" (monitoring.md:18). By code inspection; not run against a real proxy.
- Fix: serve /metrics on a separate listener or require a bearer token (secret file).

## Spec controls not implemented (verified by grep for recovery/totp/mfa/approval; no code paths)

| Req | Gap |
|---|---|
| SEC-008 (P0) | No account recovery flow at all (`grep -ri "recover\|totp\|mfa"` finds only panic recovery). Lost password = operator DB edit. |
| SEC-007 (P1) | No MFA/TOTP/recovery codes. |
| SEC-011 (P1) | No dual control: `CreateAdjustment` posts immediately (service/adjustments.go:36-70); a lone admin can credit any member up to MaxPolicyAmount. Roles grant also single-step (membership.go:145-). |
| SEC-006 | No rate limit on invitation redemption or join requests (web h_auth.go:19-20). |
| SEC-002 | Breached-password check not wired: `newService` never sets `PasswordDenied` (serve.go:38-45). |
| SEC-013 | No break-glass or operator audit trail; operator CLI reads DB directly (inherent to TA-5). |
| SEC-022 | Start-up refusal missing (see SA-02). |

## Notes (not findings)

- API session cookie names (`__Host-circula_session`, api/auth.go:15) differ from the web cookie `cm_session` (web/session.go:18), so a browser login cannot authenticate to the API cookie path and `POST /tokens` (cookieOnly) is unreachable from the product UI. Functional issue, noted because token creation is the only way to get a bearer token.
- Web CSRF token is bound to a random cookie, not to the session (session.go:44-51); an attacker who can plant cookies (subdomain, HTTP MITM) can pair their own cookie+token. Mitigated by SameSite=Lax and Origin check; not exploited.
- Password change does not revoke API tokens (documented, auth.go ChangePassword doc comment).
- Argon2 default m=64MiB t=3 p=2 exceeds OWASP minimum; DEC-006 benchmark not recorded.

## Priority order

1. SA-02 (cookie Secure/HSTS/refuse start), 2. SA-04 + SA-05 (login throttling), 3. SA-01 (redirect), 4. SA-06/SA-07 (backup perms, signed manifest),
5. SA-03 (network creation), 6. SA-08 (reservation squatting), 7. SA-14, rest.
