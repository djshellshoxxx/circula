# Security review: beta consolidation

Date: 2026-11-11. Revision: branch `claude/circula-beta-completion-1npjw6`. The final commit SHA is in
[../releases/beta-readiness-report.md](../releases/beta-readiness-report.md).

This is an AI-assisted targeted review. It is not an independent professional audit, and it does not replace SEC-033
(the independent security review owner gate). It builds on the earlier reviews and does not repeat them:
[SECURITY-AUDIT.md](../audit/SECURITY-AUDIT.md), [SECURITY-AUDIT-2.md](../audit/SECURITY-AUDIT-2.md),
[SECURITY-AUDIT-3.md](../audit/SECURITY-AUDIT-3.md), the 2026-10-28 code audit in [../audit/2026-10-28/](../audit/2026-10-28/README.md)
with its [remediation](../audit/2026-10-28/REMEDIATION-2026-11-04.md), and the ChatGPT source-pinned audit ledger
[../audits/CHATGPT_RANDOM_AUDIT_LEDGER.md](../audits/CHATGPT_RANDOM_AUDIT_LEDGER.md).

## Assets, trust boundaries, adversaries

- **Assets:** ledger integrity (balances, entries, the history of each transaction), members' credit capacity,
  dispute rights, personal data, credentials and sessions, backups.
- **Boundaries:** browser ↔ HTML handlers (cookie session plus CSRF); API client ↔ JSON API (scoped bearer tokens plus
  idempotency keys); operator ↔ CLI (local file access, no network auth); process ↔ SQLite file; backup file ↔
  restore.
- **Adversaries:** an outside attacker without an account; a member acting against other members (IDOR, replay,
  racing the credit limit); a single administrator acting alone (dual control, break-glass, policy changes against
  members); someone holding a stolen cookie or token; anyone with read access to a backup.

## Findings from this review

| ID | Severity | Area | Affected code | Preconditions | Status | Remediation | Verification |
|---|---|---|---|---|---|---|---|
| BR-01 | Medium | Governance / financial fairness | `service/disputes.go` `OpenDispute` (before `e127db8`) | An administrator changes `dispute_window_days` | **Fixed** | The deadline is fixed at completion and stored immutably (DOM-017/DAT-051, migration 0016, triggers) | Before the fix, an administrator could shorten the window and strip members of the right to dispute trades already completed, or lengthen it and revive expired disputes. `TestDisputeDeadlineFixedAtCompletion` fails against the old logic and passes now. Backfill tests in `internal/store` |
| BR-02 | Low | API concurrency control | `api/h_policy.go` `policyETag` (before `addc229`) | Two administrators edit policy at once | **Fixed** | The validator includes the pending policy version | The stale `If-Match` reached the service; only the one-scheduled-change rule stopped a lost update (409 instead of 412). `TestPendingCurrencyNamePolicyPatchChangesETag`; contract walk updated |
| BR-03 | Low | Release supply chain | `.github/workflows/release.yml` | A tag is pushed | **Fixed** | The release workflow runs `check-release-notes.sh` and `verify-release.sh` (the scan included) | YAML parses; `check-action-pins.sh` ok; not yet exercised on a real tag |
| BR-04 | Low | Privacy | `api/erasure_hook.go:33` | An admin made a `full` export, then a member is erased | Open | Drop the network's admin-scope export artefacts on erasure | The export review found it (existing behaviour). Not fixed here |
| BR-05 | Low | Enumeration | sign-in and recovery | Timing measurement | Open | Measure the timing equivalence of unknown-login and wrong-password paths | Errors and codes are already identical (tests); timing is not tested |
| BR-06 | Medium (only if integrated) | Resource exhaustion | export spool slice (not integrated) | Parallel large exports | Not integrated | See the consolidation report; fix before integrating | Independent review 2026-11-11 |

## Checks run against the high-risk classes

| Class | Result | Evidence |
|---|---|---|
| Authentication, session fixation, logout | Pass | `TestLoginDoesNotAdoptPlantedSessionCookie`, `TestLogoutInvalidatesReplayedCookie` (new); `TestLoginSessionLifecycle` |
| Cookies (HttpOnly, SameSite, Secure, `__Host-`), HSTS | Pass | `TestHeadersAndCookies`, `TestSecureCookiesAndHSTS`, `cookie_policy_adapter_test.go` (merged from PR #28) |
| CSRF | Pass | `TestCSRFRejections`, `TestCookieSessionNeedsCSRF` |
| XSS | Pass (automated) | `html/template` everywhere; the only `template.HTML` conversion formats times (`render.go:138`), so no user text reaches the output unescaped |
| SQL injection | Pass | Parameterized queries throughout `internal/store`; table names in tests only; fuzzed API bodies (`FuzzAPICommandBody`) |
| Authorization, IDOR | Pass | `TestAuthorizationMatrix` (10 actor classes × operations); non-party gets 404 in service, web and API |
| Race on the credit limit | Pass | `BEGIN IMMEDIATE` plus process write gate; `TestParallelAcceptsNeverBreachCreditLimit` under `-race` |
| Replay, duplicate submission | Pass | Idempotency keys with body hash; replay of reversals and adjustments posts nothing (new test) |
| Partial failure | Pass | Fault injection at 8 write points each for accept, reversal and adjustment |
| Tampering | Detected | Append-only triggers; `VerifyLedger` reports a tampered entry when triggers are bypassed (new test); hash-chained audit |
| Privileged operations | Pass, one owner gate | Dual control (SEC-011), break-glass time limits, audited adjustments; A3-F02 (break-glass scope) waits on DEC-014 |
| Secrets in logs, configuration | Pass (existing) | Secrets only from env or a 0600 file; `cmd_secrets.go`; `TestConfig*` |
| HTTP security headers | Pass | `TestHeadersAndCookies` |
| Resource limits | Pass (existing) | Rate limits per address and /64; body size limits; `network-join-limit` |
| Dependencies | **Not run here** | `govulncheck` needs `vuln.go.dev`, which this environment's proxy refuses (HTTP 403). CI runs it on every push (`ci.yml`). Modules are pinned in `go.sum` and verified with `go mod verify` by `package-release.sh` |
| Static analysis | Pass | `go vet ./...`; `golangci-lint` v2.15.0 (the CI pin): 0 issues, after two fixes in integrated test code |

## Earlier findings still open

- 2026-10-28 audit: 6 Medium findings. Five are remediated with regression tests (A1-01, A1-02/A1-11, A1-03,
  A3-F01, A3-F16). A3-F02 (break-glass scope) waits on owner decision DEC-014.
- The 31 Low and 18 Info findings of that audit were not part of the 2026-11-04 remediation. Their status is not
  re-verified here.
- Owner questions listed in `HANDOFF.md`: GOV-011, DOM-031, SEC-032 and SEC-006.

## Not covered

Manual penetration testing, a timing side-channel analysis, a review of the Windows-specific code paths, container
image hardening (Docker is not available here), and an independent review (SEC-033).
