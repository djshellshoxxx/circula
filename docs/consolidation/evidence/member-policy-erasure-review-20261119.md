# Independent member, policy, network and erasure presentation review

Review date: 2026-11-19. This is a read-only source review of the immutable
commits below. No compilation, test execution, remote writes or product-policy
changes were performed for this review.

## Result

No actionable presentation, escaping, authorization or form regression was
identified in the reviewed changes. No corrective patch is proposed. This
finding is limited to these diffs and their source/test context; it is not a new
runtime, full-suite or browser qualification.

| Slice | Implementation commit | Related review input | New catalogue keys |
| --- | --- | --- | ---: |
| Member actions, invitations and inactivity | `f522e013a29acd1ed4d2a80f01f35200e24ca832` | Documentation follow-up `5f64e92b9c22a96bad8123afccf87051a0a1c6d0` | 88 |
| Policy and network status | `82ee9eb752858087100b7fd76d270cb3d28e05ab` | Test-only commit `b19f7012e20caebc3c64e501845f1b6c10798e55` | 96 |
| Personal-data erasure | `8bd40428a769849eca9a00bcfa88791020e8e7bb` | Its accompanying catalogue and production-template tests | 20 |

Each implementation was compared with its own parent. All existing catalogue
key values were retained; JSON object-pair inspection found no duplicate keys.
The test-only policy commit requires the corresponding `82ee9eb` production
changes and is not independently a runnable implementation.

## Behavior and escaping evidence

- **Members and invitations:** `internal/web/h_admin.go` at `f522e013`, functions
  beginning at lines 44, 52 and 60, maps known status, role and removal values to
  plain labels and preserves unknown values as strings. `actionsFor` at line 97
  retains domain-state action selection. `invitationStatus` at line 299 keeps
  Used, Revoked, Expired, Unused precedence and the strict `ExpiresAt.Before(now)`
  boundary. The separate `CanRevoke` flag prevents display wording from deciding
  whether revocation is offered. `invitationRow` at line 312 uses the same viewer
  timezone and display format for its visible and accessible expiration labels.
- **Policy and network:** `internal/web/h_admin2.go` at `82ee9eb`, `policyPost`
  at line 57, retains amount/integer parsing, submitted fields, service arguments,
  idempotency, status handling and redirects. `internal/web/messages/messages.go`
  adds integer-typed `AroundCount` at line 83 and `AroundTwoVersion` at line 111;
  the latter formats versions as ungrouped decimal strings. Neither helper
  introduces trusted HTML. `internal/web/h_admin_network.go` maps categories at
  line 48 while preserving unknown strings; closure handling at line 128 retains
  exact-slug confirmation and the existing service call. User data remains in
  ordinary `html/template` string slots.
- **Erasure:** `internal/web/h_erasure.go` at `8bd4042`, admin handlers beginning
  at lines 29 and 39 and self-service handlers at lines 60 and 68, retains departed
  membership checks, literal `confirm=yes`, service arguments and redirects.
  `internal/web/templates/erase_confirm.html` retains the required administrator
  reason, optional self-service reason, initially unchecked consent, CSRF field,
  target URLs and error associations. No service/domain retention rules change.

Static comparisons of all seven changed templates (`admin_members`,
`admin_member_act`, `admin_invites`, `admin_inactive`, `admin_policy`,
`admin_network`, `erase_confirm`) found identical form opening tags, literal input
controls, option values and CSRF call counts. Both policy idempotency calls remain
present. Field-macro names, requirements, modes and limits were also inspected in
the diffs. The action-link accessible names preserve the visible action label.

## Tests inspected and limits

The review inspected `member_invitation_catalog_test.go`,
`policy_network_catalog_test.go`, `erasure_catalog_test.go`, their message tests,
the new helper tests, and the real templates they load. The controls cover unsafe
names/reasons/labels, unknown enums, exact large amounts and versions, invitation
expiry precedence, control visibility, and consent/error associations. These are
source observations, not newly executed test results. The branch's separately
recorded test receipts remain the execution evidence; integration and external
browser gates retain their existing status.
