# Security audit, 2026-11-18 consolidation pass (AUD4)

This is an AI-assisted review, not an independent professional audit. SEC-033 remains an external gate.

## Scope and method

Two independent reviewer agents worked against `main@ffa2758`:

- Financial integrity plus authentication and authorization.
- Data, SQLite, backup, privacy and the HTTP surface. This one also covered #71 and #72 merged on top.

Each candidate finding needed an executed probe, or was marked "code-read". Probes were adapted into regression
tests where a fix landed. A parallel integration pass reviewed every fix PR independently on GitHub; its findings
G03 and G04 on #81 were reproduced and fixed. Standards used: OWASP ASVS categories V2/V3/V4/V8/V12, with CWE
identifiers per finding.

## Verified original-ledger update: 2026-11-18T10:37:58Z

PR #84 is now [merged as `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`](https://github.com/djshellshoxxx/circula/commit/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625),
tree `717b32995b82772e3d6ff644297b7d05a74464b5`. It completes the previously outstanding portions of
**A2-08** (the problem-code catalogue) and **A3-F10** (future-dated backup retention and clock-anomaly
logging). Both are addressed on `main`; neither remains classified as a partial implementation/documentation
fix in the original ledger. That original **55-finding ledger now has 50 addressed and five policy/design
items remaining**. The additional AUD4 findings below are separate from that denominator.

Verification is pinned to reviewed source `eb4bb08b15d41295aa2c4b9f93040124ce4d66e9`,
[all seven required jobs in run 44272452596](https://github.com/djshellshoxxx/circula/actions/runs/44272452596),
[review 5954934747](https://github.com/djshellshoxxx/circula/pull/84#pullrequestreview-5954934747), and the
expected merge-tree check in [MERGE-MANIFEST](MERGE-MANIFEST.md). The merged
[remediation record](https://github.com/djshellshoxxx/circula/blob/ead63b5ce24b2bcf8352d75fd0d1ff7be711f625/docs/audit/2026-10-28/REMEDIATION-2026-11-18.md)
provides the implementation context. The separate full local combined candidate `17152dad` retains its
successful race/lint qualification; it is not the same tree as this merge.

PRs #82 and #87 remain unmerged with provider CI pending at this update. **AUD4-F03 remains partial**,
including its same-millisecond and host-clock-rollback residuals. **A3-F02 / DEC-014 remains unresolved**;
the five original policy/design dispositions were not approved by this merge.

## Findings and dispositions

| ID | Severity | CWE | State | Evidence or fix |
|---|---|---|---|---|
| E-02 | Low | 1188 | Fixed and verified ([#71](https://github.com/djshellshoxxx/circula/pull/71), `28ed716`) | `trusted_schema(0)` on every normal and raw connection; `TestSQLiteDistrustsSchemaOnEveryConnection`, `TestSQLiteRefusesSchemaFunctionCalls` |
| NEW-01 | Low | 377 | Fixed and verified ([#72](https://github.com/djshellshoxxx/circula/pull/72), `a62071d`) | Leased private staging and `O_EXCL` copies |
| AUD4-D01 | Low | 116 | Fixed and merged in [#81](https://github.com/djshellshoxxx/circula/pull/81), `b439807`; [exact-head CI](https://github.com/djshellshoxxx/circula/actions/runs/44264463917) passed 9/9 required jobs | SQLite URI path escaping; [AUD4-D01-D02](AUD4-D01-D02-BACKUP-PATHS.md); head `09ef8a5`, including native Windows and macOS qualification |
| AUD4-D02 | Low | 367/59 | Fixed and merged in [#81](https://github.com/djshellshoxxx/circula/pull/81), `b439807`; [exact-head CI](https://github.com/djshellshoxxx/circula/actions/runs/44264463917) passed 9/9 required jobs | Identity re-checks, owner/sticky parent rule, Unix-scoped no-follow; G03/G04 resolved. The foreign-owner test requires root and was skipped by CI and the parallel reviewer; its unretained author run is attributed separately in [TEST-EVIDENCE](TEST-EVIDENCE.md) |
| AUD4-F06 | Medium | 367 | Fixed; verification pending in [#82](https://github.com/djshellshoxxx/circula/pull/82) CI | Sign-in re-checks the hash inside the session-issuing unit; `ChangePassword` uses compare-and-swap; [AUD4-AUTH-FIXES](AUD4-AUTH-FIXES.md) |
| AUD4-F04 | Low | 307 | Fixed; verification pending in [#82](https://github.com/djshellshoxxx/circula/pull/82) CI | Durable `recgen` throttle on change-password |
| AUD4-F03 | Medium | 840 | Mitigated with documented residual risk; verification pending ([#87](https://github.com/djshellshoxxx/circula/pull/87) CI) | The response and re-review periods are now read from the policy in force at opening. Residual: a same-millisecond policy version, or one written after a host clock rollback, still applies (reproduced by the independent review); a durable binding at opening is outstanding |
| AUD4-F02 | Medium | 284/639 | Blocked on owner decision ([#83](https://github.com/djshellshoxxx/circula/pull/83) draft) | An administrator can take over a reviewer by resetting their password; the fix is a recovery-policy choice, and a role-only guard can be bypassed |
| AUD4-F01 | Medium | 654/284 | Blocked on owner decision | Appointer approves a puppet's request; the SEC-011 appendix accepts this as a side effect; options include quorum, an operator-rooted approver, or a proposer delay |
| AUD4-F07 | Info | 613 | Blocked on owner decision (spec or code) | No session rotation on role change; no practical effect because roles are re-read on every request |
| A3-F02 | Medium | 284 | Still reproducible; blocked on DEC-014 | A break-glass administrator can also reset passwords, reverse trades and create 90-day invitations |
| PRIV-01 | Low | 359 | Still reproducible; owner decision (SEC-014/018) | `person_activity.last_seen_at` is not erased; `sessions.last_seen_at` stays until purge |
| PRIV-02 | Low | 359 | Still reproducible; owner decision | `person_prefs.time_zone`, `challenges.reason`/`rationale` are not erased |
| PRIV-03 | Info/Low | 226 | Mitigated with residual risk (design) | No WAL checkpoint after erasure; deleted text can stay in the WAL until checkpoint |
| PRIV-04 | Policy | n/a | Blocked on owner decision (DEC-020) | Retention periods |
| PRIV-05 | Policy | n/a | Blocked on owner decision | Free-text audit reasons are kept in the hash chain |

The later password-proof follow-up in [CODE-AUDIT, C2](CODE-AUDIT.md) was reproduced and repaired in open
PR #82 at `e127acc`: 12 baseline cases fail and pass after the fix, and all 21 selected top-level race tests
pass. Independent review is recorded in [#82](https://github.com/djshellshoxxx/circula/pull/82#pullrequestreview-5954151240).
This additional finding does not change the original 55-finding ledger. The full local combined race suite
passed at 2026-11-18T10:07:00.462200Z on candidate `17152dadad91d4e1c994ea807e7d6e2497513aa8`,
tree `1e47c9c30165ca27866887533586c90323e1630e`; pinned lint also passed with zero issues. This candidate
includes #82/#84/#87. At 10:07:24Z all three PRs remained unmerged, with their seven required provider
jobs each queued, so the repaired code is not yet on `main`. See [TEST-EVIDENCE](TEST-EVIDENCE.md).

## Withdrawn after attempted falsification

- Member audit actor oracle: the service accepts an `actor_id` filter for members, but neither adapter passes it.
  Defence in depth only; not reachable over HTTP.
- Repeat disputes on one trade: by design, and only one can be live at a time (`dispute_live` index).

## Checked and sound (decisive locations)

- **Credit limits, double spend, TOCTOU.** Capacity is checked inside `BEGIN IMMEDIATE` plus the process gate
  (`internal/service/posting.go`, `internal/store/sqlite.go`). Parallel accept tests pass under `-race`.
- **Idempotency and replay.** Keys are scoped by (network, person, op, key) with a payload fingerprint; replays
  are re-authorized (`internal/service/idempotency.go`, `replay_auth.go`, `internal/api/idem.go`).
- **Reversals.** A reversal cannot itself be reversed, is bounded by the remaining total, and goes through the
  dispute flow when the trade is disputed (`internal/service/reversal.go`).
- **Overflow.** `ParseAmount`, `parseMinor`, `AddBounded` (limit 2^62) and `MaxPolicyAmount` (limit 2^50).
- **Frozen, suspended, closed or admin-less networks.** Checked by `networkOpenErr` on every posting path.
- **IDOR.** Reads are scoped by network; exports are bound to (network, person) and re-authorized at download.
- **Scoped tokens, CSRF, MFA for administrators, token revocation, recovery throttle.** No defect found.
- **Cursor signing.** Signed with HMAC over path, network, person and filter; they do not expire (Info).
- **Proxy trust.** `X-Forwarded-For` is honoured only from trusted peers.
- **Templates.** The only `template.HTML` is `fmtTime`.
- **SQL.** No string-built SQL with input in `internal/store`.
- **Supply chain.** Action pins and permissions pass `scripts/check-action-pins.sh`; `govulncheck` reports no
  vulnerabilities. CI gitleaks reports no leaks.

## Not covered in this pass

- Cache headers on authenticated pages.
- Duplicate or oversized JSON beyond the existing fuzz targets.
- `next` redirects beyond `safeNext` and the existing tests.
- Authentication timing beyond BR-05.
- Cross-tenant search and notification text.
- `logx` redaction.
- `deploy/Dockerfile` hardening.

These are candidates for the next pass or for SEC-033.
