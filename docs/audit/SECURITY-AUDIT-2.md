# Security audit 2: Circula (branch claude/circula-security-features)

Date: 2026-10-21. Scope: whole repository at the head of PR #6 (new features SEC-007, 008, 011, 012, 013, 018, OPS-019/SEC-028, SEC-027 plus the existing code).
Method: three independent read-only reviews (auth and new identity features; money, authorization, dual control, break-glass, erasure; HTTP edge, backups, CLI, CI), exploit tests run on throwaway copies of the repository, code tracing where noted.
This is an AI-assisted review. It is not an independent professional audit and gives no guarantee of security.

## Not verified

- `govulncheck` could not run: vuln.go.dev returns 403 through the sandbox proxy. Dependency vulnerabilities are NOT assessed (x/crypto 0.58.0, modernc.org/sqlite 1.60.1, google/uuid 1.6.0 reviewed by version only). CI runs govulncheck.
- Browser behaviour (cookie planting, redirects), a real TLS proxy, load, fuzzing, Windows file-permission semantics: not tested.
- Findings marked "trace" were confirmed by reading code, not by running an exploit.

## Fixed in this PR (High, verified by exploit test)

| ID | Sev | Finding | Fix | Regression test |
|---|---|---|---|---|
| SA2-01a | High | Admin password reset handed an administrator of any network full takeover of a person's global identity, including their admin roles in other networks (exploit: AdminResetPassword in network one, login as the target, reset them again in network two). | `AdminResetPassword` refuses targets that hold the admin role here or have a live membership in another network; those need `circula reset-password` (operator). | `TestAdminResetRefusesMultiNetworkTarget`, updated `TestAdminResetGuards` |
| SA2-01b | High | One administrator could set the dual-control threshold to 0 (or raise it) through `UpdatePolicy` and then post any adjustment alone. | Administrators may only enable or lower the threshold. Loosening or disabling needs the operator: `circula dual-control -network SLUG -threshold-minor N -reason TEXT` (`OperatorSetDualControl`, audited). | `TestPolicyCannotLoosenDualControl` |
| SA2-02 | High | Departure with `settle_to_zero` let one administrator move any balance to or from the system account without approval. | Admin-initiated departure whose settlement exceeds the threshold is refused with `approval-required`; settle through an approved adjustment first. Self-departure unchanged. | `TestDepartureSettlementRespectsDualControl` |

## Fixed in this PR (Medium or Low, cheap and safe)

| ID | Sev | Finding | Fix |
|---|---|---|---|
| SA2-03 | Medium | No security notice after account recovery or admin/operator reset. | `RecoverPassword` and `resetPerson` write a notice. |
| SA2-04 | Medium | A break-glass administrator could create permanent administrators (role grant outlives the session). | Principal flag `bgAdmin`; `AssignRole`, `RevokeRole` and `UpdatePolicy` refuse it (`TestBreakGlassCannotMakeLastingChanges`). |
| SA2-05 | Low | Erasure left `person_mfa`, recovery-code rows and security notices (which can carry API token names). | `erasePersonIfUnused` deletes them (`TestErasureRemovesMFAAndNotices`). |
| SA2-06 | Info | Windows CI: POSIX mode assertions in new backup tests. | Skipped on Windows. |

## Open findings (not fixed here)

| ID | Sev | Finding | Evidence | Suggested fix |
|---|---|---|---|---|
| SA2-10 | Medium | Login lock DoS. The throttle is keyed by login only and counted before the password check; five bad guesses lock a victim (even with the right password) for 15 min, doubling to 24 h, from one IP in about 12 requests. Recovery has the same shape (`recover: LOGIN`). | exploit `TestAuditA_LockDoS`; confirmed independently by all three reviews | key the lock by (login, client IP) plus a high global per-login cap; cap the lock at 15 min; operator unlock. Needs a design decision (owner). |
| SA2-11 | Medium | Restore accepts a plaintext, forged backup when an encryption key is set but `backup-require-encryption` is false and `backup-key` (manifest HMAC) is unset; a rolled-back older encrypted backup is accepted too. | trace (commands.go checkEncryptionPolicy, snapshot.go manifest) | default `backup-require-encryption=true` when a key is set; refuse unsigned restore unless `-allow-unsigned`; warn at startup if scheduled backups run without `backup-key`. |
| SA2-12 | Medium | Credit-position actions other than adjustments bypass dual control: `SetCreditLimit`, default limit and grant policy, `ReverseTransaction` of an approved adjustment, dispute resolution; sub-threshold adjustments can be split without a rolling cap. | exploit `TestB_ReverseAdjustmentBypass`, `TestB_Splits` | route limit raises, reversals of adjustments and grant increases above a threshold through the pending flow; add per-admin rolling cap. Owner decision on scope of SEC-011. |
| SA2-13 | Low | Admin reset and recovery do not delete pending MFA challenges, so a TOTP challenge issued before a reset can still complete within 5 min. | trace | delete the person's challenges in those paths. |
| SA2-14 | Low | Session and CSRF cookies: plain `circula_session` is accepted on HTTPS when no `__Host-` cookie is present; `cm_csrf` is not `__Host-` prefixed (planting from a sibling subdomain). | trace | when Secure, read only `__Host-` names. |
| SA2-15 | Low | IPv6 clients rotate within a /64 to defeat per-IP limiters; no startup warning when behind a proxy without `trusted-proxies`. | trace | key IPv6 on /64; warn. |
| SA2-16 | Low | Setup secret is marked spent in a second transaction after `CreateNetwork`; a crash between them leaves the secret valid. | trace | create and mark in one service transaction. |
| SA2-17 | Low | Export artefacts and API replay cache are not re-authorised at download (demoted or erased admin keeps a 24 h artefact). | trace | re-check role at download; purge on erasure. |
| SA2-18 | Low | `ProposeTransaction` accepts any listing id of the network (paused or other member's). No data leak. | trace | validate listing state and owner. |
| SA2-19 | Low | Release workflow does not require the tag to be on the default branch; govulncheck is version-pinned, not hash-pinned. | trace | add ancestry check; protected tags/environment. |
| SA2-20 | Info | `PRAGMA synchronous=NORMAL` can lose the most recent commits (including audit rows) on power loss in WAL mode. | trace | benchmark FULL for ledger writes; document. |
| SA2-21 | Info | Break-glass has no second approver or 24 h notice (SEC-013 a, c); existing operator commands (restore, migrate) are not gated; MFA has no lost-device self-service path. | design | owner decisions, see addenda. |

## Verified OK (selected)

- MFA: only two session-issuing paths (`LoginStep` for non-MFA people, `CompleteMFA`); challenge single-use, 5 min, person-bound; TOTP replay guard, constant-time compare, sealed seed bound to person id; recovery-code consumption atomic (parallel use succeeds once).
- Account recovery: uniform errors and equal work for unknown login, malformed and used codes; throttled before evaluation; revokes sessions, tokens, codes; does not bypass MFA. Forced change enforced for writes on service, API and web.
- Dual control: self-, target- and proposer-reject blocked; re-validation at approval; single-post under concurrency; CHECK `decided_by <> proposer`.
- Break-glass: TTL clamp in code and DB, exact expiry, append-only grants, cross-tenant guarded.
- Erasure: admin or self only, departed and no blockers, one-shot, ledger and audit chain untouched; login string absent from every table afterwards.
- Backup encryption: unique nonce per chunk (random prefix plus counter), AAD binds header, index and final flag, truncation, reorder, append and header tampering fail; keys never printed; 0600 files, plaintext work dir wiped.
- Tenant isolation: every store getter takes network_id; route enumeration of API and web found no IDOR; strict JSON decoding; parameterised SQL; token scopes cannot widen; idempotency keys scoped by network, person, operation.
- Edge: CSP strict, CSRF HMAC bound to session, safeNext rejects control characters, body and header limits, server timeouts, recovered panics, metrics token constant-time.
