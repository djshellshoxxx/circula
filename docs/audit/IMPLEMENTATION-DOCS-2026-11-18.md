# Factual implementation documentation reconciliation

Reviewed code baseline: `ef54238ee9ebc2184919b6a8ec30c3398df55ed5` (PR #69 merge).
This follows the A3-F21 documentation correction at
`f2954c80181c958818b11d9e1e7a9f19d83dd881`. It updates implementation descriptions,
preserves their historical wording in dated correction notes, and makes no code,
migration or policy change.

## Evidence and limits

| Documentation correction | Source evidence | Existing regression evidence |
|---|---|---|
| DEC-007: MFA exists; mandatory-role policy is still open | `internal/service/mfa.go`; `cmd/circula/config.go` sets `require-mfa-admins` to false by default | `TestMFAEnrolFlow`, `TestRequireMFAForAdmins`, `TestRequireMFADefaultOff` |
| DEC-014: break-glass mechanisms exist, with open scope, approval and notice policy | `internal/service/breakglass.go`: temporary grants, clamped TTL, audit at platform and network scopes | `TestBreakGlassTTLClamp`, `TestBreakGlassExpiryBoundary`, `TestBreakGlassAuditRows` |
| DEC-019: retention setting and pruning command exist; the policy decision stays open | `cmd/circula/config.go`: `log-retention` defaults to `2160h`; `cmd/circula/log_retention.go`: pruning of qualifying old log files | `TestPruneLogs` |
| SEC-018: person erasure removes MFA and both recovery-code sets, within existing authority limits | `internal/service/erasure.go`: only self/System paths call `erasePersonIfUnused`; that method retains persons with another non-departed membership, otherwise calls `DeleteMFA`, `ReplaceRecoveryCodes(..., nil)` and `DeleteAccountRecoveryCodes`; deletion statements are in `internal/store/sqlite_mfa.go` and `sqlite_recovery.go` | `TestEraseMyData`, `TestA3_05_AdminErasureKeepsLogin`, `TestErasureRemovesMFAAndNotices` |
| SEC-013: operator password reset already exists and does not require a break-glass session | `cmd/circula/recover.go` calls `internal/service/recovery.go`'s `OperatorResetPassword`, which checks System authority and reason but does not call `RequireBreakGlass` | `TestOperatorReset`, `TestResetPasswordCommand` |
| SEC-013: existing restrictions are partial | `internal/service/principal.go` implements `requireLasting`; `membership.go`'s `changeRole` invokes it; remaining powers are the unresolved A3-F02 scope question | `TestBreakGlassCannotMakeLastingChanges`, `TestA3_02_BreakGlassAdminCannotMakeLastingChanges` |
| SEC-006: fixed-lock behavior differs from the increasing-delay recommendation | `internal/service/throttle.go`; A3-02 in `SECURITY-AUDIT-3.md`; dated A3-F16 remediation in `2026-10-28/REMEDIATION-2026-11-04.md` | `TestA3_02_GlobalLockNotRenewable`; MFA counter/expiry regressions already recorded in the A3-F21 note |

Test scope matters: `TestErasureRemovesMFAAndNotices` seeds pending MFA and a
security notice, then checks that the MFA, notice and both recovery-code tables
have no rows for the person. It does not seed both recovery-code tables, so the
deletion of previously populated code rows is established here by source
inspection, not claimed as independently exercised by that test.

## Verification

All 15 tests named in the table passed with `go test -count=1`, filtered to
those names: 13 tests in `./internal/service` (0.261s) and two in `./cmd/circula`
(0.946s). `git diff --check` passed. This documentation-only change adds no
regression test or executable behavior.

## Decisions preserved

DEC-007, DEC-014 and DEC-019 remain open. A3-F02 is unresolved. The wider
SEC-011 scope, recovery-channel decision and personal-data retention/legal-hold
decisions remain unchanged. SEC-006's normative requirement and recommended
default are unchanged; the added note identifies the implemented deviation and
outstanding policy confirmation. No completed security review or release
qualification is claimed.
