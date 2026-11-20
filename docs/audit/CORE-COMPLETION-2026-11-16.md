# Core completion and consolidation — 2026-11-16

Branch: `codex/circula-core-completion-20261116`. Continuation base: `dc97f7d`, which contains
`claude/quirky-einstein-l9ni8m@4a8db9e` and main `2eacf77`. Documentation upkeep
`claude/cool-gauss-vdxlod@64fa4f6` is merged, preserving newer implementation evidence.
Merged by PR #56 (`304600c`) and follow-up #57 (`7cdb966`). The 15 future modules remain on independent branches and are not enabled here.

This is an implementation and AI-assisted review record, not an independent security certification.
Historical audit reports retain their original findings; the table below records specific remediations.

| Finding / requirement | Result | Regression evidence |
|---|---|---|
| A1-05 / FR-034 | Frozen networks cannot starve other proposals' expiry | `service/expiry_fairness_test.go` |
| A1-07 / GOV-025 | Failed closure final exports retry; erasure waits for recorded checksum | `service/closure_retry_test.go` |
| A1-08 / A3-F11 | Operator threshold edits preserve future policy values/notice times | `service/operator_policy_regression_test.go` |
| A1-10 | Pending adjustments skip frozen networks before bounded selection | `service/adjustment_expiry_fairness_test.go` |
| A2-01 / A2-18 | Canonical field names; no internal decoder-type disclosure | `api/canonical_fields_test.go`, `strict_test.go` |
| A2-02 | Restricted tokens cannot list/revoke global tokens or change global preferences | `service/token_person_scope_test.go` |
| A2-03 | Cached responses reauthorize current scope/role/MFA; stale finishers cannot replace newer records | `api/replay_*test.go` |
| A2-04 / A2-05 | Effective break-glass and MFA authorization reaches both adapters | `api/adapter_auth_test.go`, `web/adapter_auth_test.go` |
| A2-06 | Logout reports revocation failures and keeps retry possible | `web/logout_failure_test.go` |
| A2-07 (credit limits) | Configuration-sensitive ETag and atomic write precondition | `api/credit_limit_revision_test.go`, `service/credit_limit_revision_test.go` |
| A2-07 (policy) | Effective and pending policy revisions rechecked inside the write transaction | `api/policy_revision_test.go`, `service/policy_revision_test.go` |
| A2-08 | Common runtime error statuses match OpenAPI; authentication preserves retryable storage failures | `internal/contract` runtime error regressions |
| A2-09 | Signed cursor pages for policy history and challenges, bounded at 100 | `api/governance_pagination_test.go` |
| A2-12 / UX-018 | Signed review/back/confirm for re-review and adjustment approval | `web/money_confirmation_test.go`, affected Playwright flows |
| A2-15 | Public assets do not issue browser cookies | `web/static_cookie_test.go` |
| A3-F06 | Unicode throttle-family cleanup uses database character length | `store/throttle_family_test.go` |
| A3-F09 | Restored bearer authority invalidated and audited before swap | `cmd/circula/restore_credentials_test.go`, `service/restore_credentials_test.go` |
| A3-F10 | Future-dated backups cannot suppress cadence; precise names coexist with legacy names | `cmd/circula/jobs_regress_test.go`, `backup_test.go` |
| A3-F12 | Invalid nonpositive migration versions refuse cleanly | `store/migrate_version_test.go` |
| A3-F22 | Break-glass login normalization matches sign-in | `service/breakglass_login_regression_test.go` |
| A3-F24 / BR-04 | Erasure drops affected archives, rejects stale running exports and revokes downloads | `api/export_lifecycle_test.go`, `export_spool_test.go` |
| A3-F25 | Erasure uses exact notification relationships, preserving unrelated content | `service/notification_erasure_test.go` |
| History ordering | Immutable insertion order survives backwards clocks, equal timestamps, VACUUM and migration | `store/event_order_test.go` |
| Export compatibility | Pre-upgrade export bytes and closure checksum remain unchanged | `store/event_order_test.go` |
| QA-019 | Decline/expiry status, balances, reservations and notices exercised through API/HTML forms | `api/qa019_expiry_test.go`, `web/qa019_expiry_test.go` |
| Export reconciliation | Standalone CSV/JSON reader validates summary/account/transaction integrity | `scripts/test_read_export.py`, `service/export_schema_test.go` |

## Further current-source review

- Dispute resolution, retry and re-review now audit the reversal transaction itself; prior reversal
  authors remain conflicted even after more than 500 newer reversals (`dispute_reversal_audit_test.go`,
  `dispute_independence_history_test.go`).
- Recovery throttle rows are erased with the final person login; operator-provisioned identities and
  CLI restore records identify the platform operator (`erasure_throttle_test.go`,
  `operator_registration_test.go`, `restore_audit_test.go`).
- Suspended/departed navigation includes permitted reads, rendered timestamps remain structured markup,
  and forced password rotation covers home and invitation pages (`navigation_access_test.go`,
  `notice_times_test.go`, `forced_password_pages_test.go`).
- Unsupported and malformed API queries reject before handlers or cached replay. Damaged historical
  balance queries cannot silently become current-balance queries (`get_query_validation_test.go`,
  `raw_query_validation_test.go`).
- UUID reference and paging validation covers 61 service entry points and both HTTP adapters after
  authorization. Invalid targets do not reach repository lookups; scope/role precedence, opaque
  missing records and invitation/slug/numeric-cursor formats remain covered (`identifier_boundary_test.go`).
- Scheduled verification is read-only while retaining freeze/thaw, metrics and logs
  (`cmd/circula/scheduled_verify_test.go`). Authentication throttle settings are bounded and wired
  through file/environment/flags to password, recovery and MFA checks (`throttle_config_test.go`,
  `cmd/circula/config_throttle_test.go`).
- The migration guard recognizes protected updates, replacement/upsert writes, trigger drops and
  shorthand column drops, including multiline and quoted forms (`scripts/test_check_migrations.py`).
- ARC-004 contracts live in `internal/repository`; all 53 existing contract types retain old-store
  aliases. Service and reusable contract tests have no concrete database dependency. Architecture,
  storage capability and SQLite/wrapped contract tests enforce that boundary.
- ARC-002/API-071 integration tests compare 43 paired MVP operations and six explicit API-only
  lifecycle/token operations. Review and confirm phases compare ledger and complete audit content;
  displayed reversal balances now create the required read-audit records.
- UX-017 adds an optional related transaction on adjustments, including dual-control requests,
  review, approval, API, history and exports. Migration 0018 enforces same-network posted-party
  relationships and immutability, including REPLACE protection. Reversal references and posting
  semantics are unchanged. No-link legacy exports retain identical bytes and closure checksums.
- Real SQLite `SQLITE_FULL` after staged posting now tests acceptance, reversal and adjustment:
  rollback, same-connection recovery, replay and reopen all pass. The later Linux WAL-write test
  separately covers a real commit-write failure; read/fsync/checkpoint and other I/O faults remain open.
- QA-042 now separates deterministic citations from recorded per-test evidence. Fifty missing
  citation rows were mapped to existing tests; remaining inspection/manual/external criteria stay
  explicit. CI records actual terminal test results, timestamps, commit and source fingerprint.
- QA-001 gives each seed an independent database, clock and ID generator, captures failing steps and
  helper failures, and supports strict disjoint seed ranges. A manual runner preserves logs, failure
  fixtures and checkout/toolchain metadata. The complete 100,000-sequence run passed with 2,800,700 generated operations; see
  `docs/qa/QA001-2026-11-16.md`. Release thresholds remain unconfirmed.

## Historical-branch follow-up

- Selectively ported reversal-history pagination and its 501-entry regression from
  `claude/cool-ramanujan-n2peag` (`888b4f2` / `6631d8b`). Older incompatible export configuration
  and duplicate implementations were not merged over the newer code.
- PR #42's newer audit notes identified RESTORE-01 and CRYPTO-02. Restore now uses a durable marker;
  every command-owned database open refuses interrupted/legacy restore remnants before SQLite can
  create an empty database. Seven subprocess interruption points preserve recovery evidence.
  A scoped, expiring validation exception exists only during the swap callback. Windows test-binary
  cross-compilation and focused race tests passed; see `restore_swap_crash_test.go` and the recovery runbook.
- New HMAC-v2 manifests sign verification summaries and other attestation fields; historical signatures
  remain verifiable. Unknown versions and version-stripping downgrade attempts reject. See
  `manifest_signature_version_test.go`. Renaming a backup remains supported.
- Temporary administrator proposers use current effective authority inside approval; expired/closed
  grants and grants from other networks cannot post (`dualcontrol_proposer_grant_test.go`).
- Invitation-only join GET/POST hide network details from nonmembers while preserving token invitations
  and public join modes (`web/join_privacy_test.go`).
- The token-creation contract no longer advertises idempotency that the one-time-secret route does not provide.
- Linux race tests have an explicit 20-minute per-package budget, within the existing 30-minute CI job budget.

## Validation status

PR #56 passed all seven PR jobs at `e3c60a6` ([run 43907084340](https://github.com/djshellshoxxx/circula/actions/runs/43907084340)).
PR #57 passed all nine jobs at `dc89212` ([run 43914378342](https://github.com/djshellshoxxx/circula/actions/runs/43914378342)),
including native Windows, Linux/macOS race, 579 browser/keyboard/axe checks, lint/vulnerability/secret
scans, API contract/load, fuzz, release targets and reproducibility/deployment smoke. Runtime requirement
evidence is attached to the Linux test job. Citation presence alone does not prove an entire requirement.

The Windows follow-up fixed CRLF assumptions, portable closed-file assertions and fixture resource
cleanup. The Linux WAL test uses a child-only file-size limit to cause actual `SQLITE_IOERR_WRITE`
during COMMIT after a complete posting has been staged. It verifies the original state, preserved
partial WAL, successful retry/replay and reopen. Focused normal and race runs passed; broader I/O
qualification remains open in the [storage evidence note](../qa/sqlite-storage-failure-qualification.md).

The [QA-001 long-run record](../qa/QA001-2026-11-16.md) independently verifies 100,000 contiguous,
distinct non-race sequences, 2,800,700 generated operations and no failures or fixtures. Both batches
used clean `e3c60a6`; PR #57 leaves production files and that harness unchanged. This is execution
evidence, not approval of DEC-023 thresholds or performance criteria on reference hardware.

Manual screen-reader and independent security review remain separate gates. No release tag or deployment was made.

## Remaining boundaries

See `docs/TODO.md` for qualification and owner decisions. This work does not claim closure of every
historical policy/retention question or every requirement in the larger specification. A2-07's
credit-limit and policy races are covered here; other mutable resources' semantics retain their
separate regression and contract evidence.
Passwords, enrolled MFA/recovery secrets and memberships retain snapshot semantics after restore;
operators still reconcile post-snapshot changes. Existing export format ordering is deliberately
preserved for final closure checksums; UI/API histories use the new immutable sequence.
