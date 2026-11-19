# Security finding disposition — consolidation review

Reviewed source: `163fe1596bd7cfb8c2a22891629fc4459e6b639c` (PR #69). PR #70 head reviewed: `1ee699f4e41c2779f32bd27063b14a4ff22e3a7d`.

The 55 findings are historical findings at `bb5a36f`, not 55 presently exploitable defects. Current source has fixes and named regression tests for 49. Six residual findings remain below. Status `fixed` denotes inspected current code and existing regressions; aggregate execution must be linked from the root validation record before claiming all tests passed. No finding is dismissed as a false positive merely because later code repaired it.

Independent checks in this workstream: PR #69’s four new tests pass; three guard mutations fail as expected. Re-running original residual PoCs confirms break-glass scope, challenge-text retention, WAL remnants and immediate per-member limit cuts. Clean/forbidden ZIP fixtures prove the packaging scanner handles ZIPs. Migration-guard Python tests pass.

| Finding | Severity | Current disposition | Current evidence and decision |
|---|---|---|---|
| A1-01 | Medium | fixed | Network-state gate refuses settlement postings after closure, during suspension and when an administrator is required. `internal/service/departure_test.go` |
| A1-02 | Medium | fixed | Scale changes refused at creation boundary; scheduled-change bypass regression covered. Historical databases with already-scheduled scale changes still require migration policy. `internal/service/decimals_test.go` |
| A1-03 | Medium | fixed | Re-review posting refused while another live dispute owns the transaction. `internal/service/dispute_rereview_test.go` |
| A1-04 | Low | fixed | networkOpenErr gates reverse; PR69 adds entry-point regression, independently mutation-tested in this review. `internal/service/ledger_auth_integration_regression_test.go` |
| A1-05 | Low | fixed | Frozen networks excluded before the bounded sweep; healthy expiry does not starve. `internal/service/expiry_fairness_test.go` |
| A1-06 | Low | fixed | Departure cancels proposals beyond the first 500. `internal/service/departure_pagination_regression_test.go` |
| A1-07 | Low | fixed | Closure with an absent final checksum remains eligible for retry. `internal/service/closure_retry_test.go` |
| A1-08 | Info | fixed | Operator changes preserve scheduled policy values and allocate the next revision. `internal/service/operator_policy_regression_test.go` |
| A1-09 | Info | fixed | Reversal-target audit events and ledger postings share the transaction. `internal/service/dispute_reversal_audit_test.go` |
| A1-10 | Info | fixed | Frozen pending adjustments excluded before bounded selection. `internal/service/adjustment_expiry_fairness_test.go` |
| A1-11 | Low | fixed | Current tests enforce scale immutability; remediation record amended FR-005/GOV-004. DOM-031/DAT-049 remain separately qualified. `internal/service/decimals_test.go` |
| A2-01 | Low | fixed | Canonical JSON field names reject case aliases; strict body tests remain. `internal/api/canonical_fields_test.go` |
| A2-02 | Low | fixed | Network-bound tokens cannot manage person-wide tokens/preferences. `internal/service/token_person_scope_test.go` |
| A2-03 | Low | fixed | Replay reauthorizes scope, role, MFA, break-glass expiry and current membership. `internal/api/replay_permissions_test.go`; `internal/api/replay_authority_edges_test.go`; `internal/api/replay_inventory_test.go` |
| A2-04 | Low | fixed | Adapters use live effective admin authority including authorized break-glass grants. `internal/api/adapter_auth_test.go`; `internal/web/adapter_auth_test.go` |
| A2-05 | Low | fixed | Administrator reads/navigation enforce the currently required MFA. `internal/api/adapter_auth_test.go`; `internal/web/adapter_auth_test.go` |
| A2-06 | Low | fixed | Failed server revocation does not clear retryable session or claim success. `internal/web/logout_failure_test.go` |
| A2-07 | Low | fixed | Conditional writes include effective, pending, override and expiry state, with atomic service recheck. `internal/api/credit_limit_revision_test.go`; `internal/api/policy_revision_test.go`; `internal/service/credit_limit_revision_test.go`; `internal/service/policy_revision_test.go` |
| A2-08 | Low | fixed | Declared runtime errors are validated against actual OpenAPI responses. `internal/contract/middleware_responses_test.go` |
| A2-09 | Low | fixed | Bounded signed pagination for policy logs and challenges. `internal/api/governance_pagination_test.go` |
| A2-10 | Low | fixed | Invitation-only networks remain opaque to outsiders; public join modes retained. `internal/web/join_privacy_test.go` |
| A2-11 | Low | fixed | Navigation reflects readable suspended/departed sections. `internal/web/navigation_access_test.go` |
| A2-12 | Low | fixed | Re-review and dual-control approval use signed reviewed confirmation, rechecking live state. `internal/web/money_confirmation_test.go` |
| A2-13 | Info | fixed | Notice times render as time elements rather than escaped markup. `internal/web/notice_times_test.go` |
| A2-14 | Low | fixed | Rendered invitation/login/register links retain one correctly escaped local return path. `internal/web/next_link_test.go` |
| A2-15 | Info | fixed | Cacheable public assets issue no CSRF cookies. `internal/web/static_cookie_test.go` |
| A2-16 | Info | fixed | Record identifiers rejected before repository calls after authorization, preserving opaque failures. `internal/service/identifier_boundary_test.go`; `internal/service/identifier_http_test.go`; `internal/api/path_id_test.go` |
| A2-17 | Info | fixed | Home and invitation entry points enforce forced password change. `internal/web/forced_password_pages_test.go` |
| A2-18 | Info | fixed | Decoder messages hide Go internals; unknown/malformed queries refused. `internal/api/canonical_fields_test.go`; `internal/api/get_query_validation_test.go`; `internal/api/raw_query_validation_test.go` |
| A2-19 | Info | fixed | POST/tokens correctly documents no Idempotency-Key support; token secrets are never cached. Canonical API-066 SHOULD remains design guidance, not a runtime replay promise. `internal/api/a3b_test.go`; `docs/api/openapi.yaml` |
| A2-20 | Info | fixed | Edge and adapter request identifiers are correlated; untrusted client identifiers do not replace the server audit identifier. `cmd/circula/log_safety_test.go` |
| A2-21 | Info | deferred design | hashToken still uses plain SHA-256, so API-009 salted/keyed-hash wording is unmet. Tokens contain 256 random bits; credential format migration and key/salt lookup compatibility need explicit design. No unapproved token revocation or format change. `internal/service/auth.go`; `docs/specs/07-api-contract.md` |
| A3-F01 | Medium | fixed | Encrypted pre-migration backups decrypted before database integrity verification. `cmd/circula/premigrate_encrypted_test.go` |
| A3-F02 | Medium | owner policy | DEC-014 remains unresolved. Current original PoC re-run separately; no break-glass/governance policy changed. `docs/specs/addenda/SEC-013.md`; `docs/audit/2026-10-28/REMEDIATION-2026-11-04.md` |
| A3-F16 | Medium | fixed | Session-side operations cannot renew/consume the owner sign-in MFA budget. `internal/service/mfa_throttle_test.go` |
| A3-F03 | Low | fixed | requireDecider enforces admin MFA; PR69 service-entry regression independently mutation-tested. `internal/service/ledger_auth_integration_regression_test.go` |
| A3-F05 | Low | partly fixed policy | Recovery and sign-in throttle families erased, including Unicode/IP variants. Challenge reasons still survive; SEC-014/018 retention policy unresolved. `internal/service/erasure_throttle_test.go`; `internal/service/erasure.go` |
| A3-F06 | Low | fixed | SQLite character-length cleanup replaces mismatched Go-byte length and respects literal prefixes. `internal/store/throttle_family_test.go` |
| A3-F09 | Low | fixed | Restore invalidates bearer sessions/tokens and break-glass credentials before swap. Password/recovery-code/invitation state still reflects the backup; separate operational limitation. `cmd/circula/restore_credentials_test.go`; `internal/service/restore_credentials_test.go` |
| A3-F10 | Low | fixed | Future-dated backup names excluded from cadence; subsecond unique names and legacy retention supported. `cmd/circula/jobs_regress_test.go`; `cmd/circula/backup_test.go` |
| A3-F11 | Low | fixed | Same correction as A1-08. `internal/service/operator_policy_regression_test.go` |
| A3-F13 | Low | fixed | Explicit audited revoke-sessions command and corrected key-rotation documentation; no claim that key rotation alone revokes opaque sessions. `cmd/circula/revoke_sessions_test.go`; `internal/service/sessions_review_test.go`; `docs/ops/configuration.md` |
| A3-F14 | Low | fixed | Protected append-only UPDATE/REPLACE/UPSERT and DROP TRIGGER checked, with lexical/comment/exemption regressions. `scripts/check_migrations.py`; `scripts/test_check_migrations.py` |
| A3-F15 | Low | fixed | Scheduled ledger verification is read-only while maintaining in-memory freeze/thaw behavior. `cmd/circula/scheduled_verify_test.go` |
| A3-F17 | Low | fixed | Manifest/record/anchor writers use unpredictable exclusively-created files. Plaintext snapshot staging residual is NEW-01, handled separately. `cmd/circula/util_atomic_test.go`; `cmd/circula/util.go` |
| A3-F18 | Low | owner policy | Per-member limit cuts still apply immediately. GOV-003/default-policy notice versus per-member notice requires owner decision; no financial-rule change. `internal/service/membership.go`; `docs/TODO.md` |
| A3-F19 | Low | fixed | Operator authentication settings bounded, validated and wired into service. `internal/service/throttle_config_test.go`; `cmd/circula/config_throttle_test.go` |
| A3-F25 | Low | fixed | Notification erasure targets exact resource relationships, preserving unrelated name substrings. `internal/service/notification_erasure_test.go` |
| A3-F07 | Info | deferred design | PASSIVE checkpoint can retain erased plaintext in WAL frames. Forced TRUNCATE policy and concurrency/availability consequences unresolved; secure_delete alone does not prove historical-page removal. `docs/security/security-review.md`; `internal/store/sqlite.go` |
| A3-F12 | Info | fixed | Nonpositive migration metadata refuses cleanly instead of indexing out of bounds. `internal/store/migrate_version_test.go` |
| A3-F20 | Info | fixed | ZIP-specific listing/extraction implemented. This review independently ran clean and forbidden-name ZIP fixtures; exits 0 and 1 respectively. `scripts/scan-release.sh` |
| A3-F21 | Info | documentation open | Line 14 still says a shared counter doubles to 24 hours; line 15 amendment correctly describes split counters and fixed 15 minute default. Correct historical wording; do not change runtime policy. `docs/specs/addenda/SEC-007.md` |
| A3-F22 | Info | fixed | Break-glass target login normalized consistently with sign-in. `internal/service/breakglass_login_regression_test.go` |
| A3-F23 | Info | fixed | Operator registration audited as the operator, rather than self-registration. `internal/service/operator_registration_test.go`; `cmd/circula/cli_regress_test.go` |
| A3-F24 | Info | fixed | Erasure invalidates affected archives and running exports; admitted downloads pinned/revoked safely. `internal/api/export_lifecycle_test.go`; `internal/api/export_spool_test.go` |

## Additional findings and boundaries

| Item | Disposition | Evidence / reason |
|---|---|---|
| NEW-01 | Fix prepared separately | Private temporary directories for all snapshot and verification files; exclusive copy creation; cleanup restricted to owned paths. Focused failing-before tests retained. |
| PRIV-01 / PRIV-02 | Owner retention decision | `person_activity.last_seen_at`, `person_prefs.time_zone`, and challenge free text remain; no policy silently altered. |
| PRIV-03 | Deferred design; same issue as A3-F07 | Original current-source PoC finds erased display-name bytes in WAL after PASSIVE checkpoint. A forced TRUNCATE can contend with active readers; policy/design must be explicit. |
| PRIV-04 / PRIV-05 | Policy/process | DEC-020 retention and free-text audit-reason retention need owner policy. Existing immutable audit history is not rewritten. |
| E-02 | Deferred hardening | SQLite connection does not explicitly disable trusted_schema. Unsigned restore is refused by default; deliberate defense-in-depth change needs compatibility tests across schema functions/triggers and connections. |
| A3-F09 residual | Documented restore limit | Bearer credentials are invalidated; passwords, recovery codes and invitation state still reflect the historical snapshot. Their migration/invalidation policy is distinct. |
| A1-02 residual | Owner migration decision | Old databases can already contain a future scale change scheduled by an older binary. Current writes prevent new ones; remediation of existing policy data cannot be silently invented. |
| SEC-033 | External qualification remains open | Agent review and automated tests are not the required independent professional audit. |

## Reproduction record

`current-policy-residuals.log`: all four lasting break-glass actions and immediate dual-control approval reproduce; challenge text survives while erased-login throttle rows no longer do; a canary survives in the WAL; a per-member limit cut applies immediately. These are explicit unresolved policies/designs, not changes in this consolidation.

`a3-f20-zip-scan.json`: clean ZIP exits 0; ZIP containing `credential.key` exits 1 with the forbidden-name finding.

`pr69-focused.log`: four tests pass. `pr69-mutation-gate.log`, `pr69-mutation-mfa.log`, and `pr69-mutation-reversal.log`: each deliberately weakened guard fails the corresponding regression.

`new01-before.log`: existing files and symlink targets are overwritten by the old copy helper; predictable staging paths are used/deleted. `new01-cleanup-before.log`: stale plaintext directories remain and failed copies leave new files. The final patch covers both.
