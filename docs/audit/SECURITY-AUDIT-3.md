# Security audit 3: Circula (branch claude/circula-finish)

Date: 2026-10-21. Head audited: 29ef336 (SA2-10..20 fixes merged); fixes below are on later commits of the same branch.
Method: three fresh read-only reviewers (authentication and identity; money and authorization; HTTP edge, backups, CLI, CI), exploit tests run on throwaway copies of the repository. Prior rounds: SECURITY-AUDIT.md, SECURITY-AUDIT-2.md.
This is an AI-assisted review. It is not an independent professional audit and gives no guarantee of security.

## Not verified

- `govulncheck` could not run: vuln.go.dev returns 403 through the sandbox proxy (tried by two agents). Dependency vulnerabilities are NOT assessed locally; `go mod verify` passed; every module is at the latest version per the module proxy (docs/research/DEPENDENCIES.md). CI runs govulncheck.
- Browser behaviour, real TLS proxy, Windows ACLs, load, GitHub Action SHA correctness (offline), onion/I2P behaviour: not tested.
- Findings marked "trace" were confirmed by reading code only.

## High (all fixed, verified by exploit test, regression test kept)

| ID | Finding | Fix | Regression test |
|---|---|---|---|
| A3-01a (auth) | The web adapter passed the CIDR rate key (`2001:db8::/64`) to `WithClientIP`, which could not parse it, so all IPv6 clients shared one `(login,"?")` bucket: one IPv6 attacker locked a login for every IPv6 user. | `canonicalIP` accepts prefixes. | `TestA3_01_IPv6RateKeyKeepsBucketsApart`, `TestA3_01_WebIPv6CrossPrefixNoLockout` |
| A3-01b (edge) | Restore TOCTOU: the signed manifest was checked, then a pre-restore backup ran, then the file was copied without re-hashing. Anyone able to replace the backup file in that window got a forged database restored under "signature: valid". | The copy is re-hashed against the signed manifest. | `TestRestoreSignedSwapDuringPreRestoreBackup` |
| A3-01c (money) | Dual control defeated by a puppet administrator: a rogue admin appoints a second admin and that admin approves the rogue's adjustment. | Approver refused if their admin role descends (transitively) from the proposer, or was granted under 24 h ago (`NewAdminApprovalDelay`); every other admin gets an `admin.granted` notice. Founder / system grants are the trust root. | `TestA3_*` in `service/a3b_test.go` |

## Medium and Low (fixed)

| ID | Sev | Finding | Fix |
|---|---|---|---|
| A3-02a | Med | Login-wide lock renewable forever by one request at expiry. | An expired lock grants half the allowance, not a full re-arm (`TestA3_02_GlobalLockNotRenewable`). Residual: a sustained distributed attacker (100 failures per 15 min) can still hold the lock; operator `circula unlock` and the cap are the mitigations. |
| A3-02b | Med | Break-glass administrators could close the network, remove admins, anonymise members. | `requireLasting` on CloseNetwork, admin DepartMember, non-self AnonymiseMember, suspend, SetDefaultFlag. |
| A3-03a | Med | MFA enrolment needed no re-authentication; a stolen session could enrol its own authenticator. Operator reset left MFA in place with no CLI to remove it. | Password required at begin and confirm (throttled with recgen), other sessions revoked on enable, `circula reset-mfa` (audited). |
| A3-03b | Med | System account balance could overflow (8,193 maximal adjustments) and brick reads and verification. | Both sides bounded at 2^62 (`domain.MaxBalance`), triggers in migration 0015, saturating totals in verification. |
| A3-04a | Low | Argon2 ran before the throttle and code check on `/recover` (CPU DoS). | Throttle and code check first; hashing only for a valid code. |
| A3-05a | Low | `database` path with `?`, `#`, `%` changed the SQLite URI. | Rejected in config validation. |
| A3-05b | Med/Low | A network admin could erase a person's login across all networks. | Admin erasure anonymises the membership only; person row only on self-erase or operator CLI. |
| A3-06 | Low | `OnMemberErased` unwired: replay cache and export artefacts survived erasure. | Wired (in-process; `circula erase` from another process cannot reach the server's memory, TTL 24 h). |
| A3-08 | Low | Listing text accepted control/zero-width characters; no per-member cap; web edit read only 500 listings. | Rejected (Cc, Cf, Zl, Zp); cap 100 active; lookup by id. |
| A3-09 | Low | Plaintext API token cached in replay cache for 24 h. | `POST /tokens` not cached. |
| (auth) | Low | `OperatorUnlockLogin` did not clear `recgen:` rows. | Cleared. |

## Open (not fixed; owner decisions or accepted)

| ID | Sev | Finding | Note |
|---|---|---|---|
| A3-04b (money) | Med | Operator CLI commands (`erase`, `dual-control`, `reset-password`) do not call `RequireBreakGlass`, contrary to docs/dev/break-glass.md. | Owner decision, SEC-013; see SEC-011 appendix. |
| A3-07 | Low | Voluntary departure with settle-to-zero forgives debt up to the credit limit with no approval. | SEC-011 appendix. |
| A3-10, A3-11 | Info | Replay cache eviction can duplicate non-idempotent commands; audit chain unkeyed. | SEC-011 appendix. |
| E-02 | Low | Restored/unsigned database opened without `trusted_schema=OFF` or schema fingerprint. | Mitigated by the default refusal of unsigned restores. |
| E-03 | Low | `/metrics` public when a same-host proxy sends no forwarding headers and no `metrics-token`. | Set `metrics-token`; mandatory for onion deployments (docs/ops/onion-i2p.md). |
| E-04 | Low | Behind Tor/I2P/any proxy without `trusted-proxies` all clients share one address (per-IP limits collapse). | docs/research/ANONYMITY-NETWORKS.md backlog B3/B4. |
| E-06..10 | Info | No release signing or provenance; flash cookie not `__Host-`; no write deadline on health endpoints; key-file owner not checked; terminal escapes in `break-glass list`. | Backlog. |
| R-01 | Info | Recovery codes are 80-bit salted hashes (research note suggested keyed HMAC); TOTP seed key is SHA-256 of `SecretKey` (needs high entropy); no minimum backup passphrase length. | docs/research/CRYPTO-CONFORMANCE.md. |
| SA2-12 | Med | SEC-011 scope extension (limits, reversals, rolling cap). | docs/specs/addenda/SEC-011-owner-decision.md, options and recommendation. |
| SA2-21 | Info | Break-glass second approver and 24 h notice; lost-device MFA self-service. | Owner decision. |

## Verified OK (selected)

- Throttle keys cannot collide; reservation is atomic; unknown/locked/wrong look uniform; recovery codes single-use and atomic; TOTP replay blocked atomically; challenges single-use 5 min.
- `__Host-` cookies read exclusively over HTTPS; CSRF HMAC bound to session; XFF taken only from trusted peers (right-most untrusted entry); `safeNext` blocks `//`, backslash, control characters.
- CSP strict, no inline script/style; CSV neutralisation; export queries scoped by `network_id`; cursors HMAC-bound.
- Backup traversal not possible; temp files 0700/`O_EXCL`; restore refuses unsigned without `-allow-unsigned`; encrypted container AAD binds header/index/final flag.
- Workflows: actions pinned to SHAs, minimal permissions, no expression injection, tag ancestry check present.
- Ledger: LED-5 checked on payer capacity on accept/reversal/adjustment; append-only triggers; tenant isolation held across every store getter and route.
