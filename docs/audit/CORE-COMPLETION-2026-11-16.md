# Core completion and consolidation — 2026-11-16

Branch: `codex/circula-core-completion-20261116`. Continuation base: `dc97f7d`, which contains
`claude/quirky-einstein-l9ni8m@4a8db9e` and main `2eacf77`. Documentation upkeep
`claude/cool-gauss-vdxlod@64fa4f6` is merged, preserving newer implementation evidence.
The 15 future modules remain on independent branches and are not enabled here.

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

Focused tests for the changes above passed, including negative controls for several historical defects.
The earlier full Go suite passed after the first core fixes. The final combined race, contract and CI
checks are pending; do not treat that earlier run as evidence for later changes.

Local Chromium launch is blocked by the execution environment's socket restriction. Playwright tests
parse, but current browser execution must be verified in GitHub Actions. Manual screen-reader and
independent security review are still separate gates. No new release tag or deployment was made.

## Remaining boundaries

See `docs/TODO.md` for qualification and owner decisions. This work does not claim closure of every
historical Low/Info finding or every requirement in the larger specification. A2-07's credit-limit
race is covered here; other mutable resources' concurrency semantics need separate qualification.
Passwords, enrolled MFA/recovery secrets and memberships retain snapshot semantics after restore;
operators still reconcile post-snapshot changes. Existing export format ordering is deliberately
preserved for final closure checksums; UI/API histories use the new immutable sequence.
