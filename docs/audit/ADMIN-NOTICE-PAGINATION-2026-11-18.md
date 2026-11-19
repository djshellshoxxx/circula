# Administrator grant notices beyond 500 members

Reviewed baseline: `163fe1596bd7cfb8c2a22891629fc4459e6b639c`, the source head of merged PR #69.

## Finding and reproduction

`internal/service/membership.go:noticeAdminGranted` requested only the first 500 active members and then filtered those members for the administrator role. Existing administrators whose member IDs sorted after that page received no `admin.granted` security notice when another administrator was appointed.

This is a notification coverage defect in the existing A3-01 control. It does not change who may grant or approve a role, and the reproduction does not demonstrate a bypass of the independent-approver checks.

The failing-before regression seeds 501 low-sorted active members. It includes an administrator on the first page and the original founder after the boundary. Granting the administrator role to a real member produces one first-page notice but zero notices for the founder; the regression requires one for each.

## Change

The function now pages `ListMembers` with its existing ascending-ID cursor until the result is exhausted. Each request is capped at 500 rows. The loop uses the same transaction supplied by `changeRole`, so the role assignment, notifications and audit event retain their common commit or rollback boundary.

Recipient rules are unchanged: notify active administrators in the specified network, except the grantee. No financial rule, break-glass policy, migration or future module changes.

## Verification

`TestAdminGrantNoticesBeyondFirstMemberPage` fails before the fix and passes afterward. It verifies exactly one new notice for both first-page and later-page administrators, and zero for an ordinary member, the grantee, an inactive administrator and an administrator of another network.

The new regression and existing A3-01 tests pass twice. The new regression and `TestA3_01_AdminGrantNotifiesOtherAdmins` also pass twice under the race detector. Aggregate integration qualification is recorded in the consolidation execution record and pull request.

A3-F02 / DEC-014 remains unresolved and unchanged.
