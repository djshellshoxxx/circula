# Remaining Core work and integration gates

## Current candidate

1. Complete broad race/regression, contract, fuzz, browser, static and dependency
   qualification for the actual candidate tree. The combined race attempt on
   local 5da8bc7 was interrupted by an executor restart; it is not a pass.
2. Complete and integrate the isolated trade/history and inbox/report catalogue
   continuations. Listing/profile/directory and JavaScript slices are already
   integrated in the candidate.
3. Keep PR #102 current and observe its exact-head CI. The Git CLI lacks write
   credentials; the authorized connected provider is the working publication
   path. The cross-platform label is set; the next source update must trigger
   the full intended matrix.
4. Keep the additive 120-branch/102-PR refresh current when heads move. Preserve
   the initial 114-branch evidence unchanged.
5. Merge only after qualification, record the actual main SHA, and reevaluate
   pending Core branches. Main remains ead63b5 at this checkpoint.

## Active independent Core work

- PR #90 owns API-036 and several reviewed financial/authentication repairs.
  It must not merge alone: the live-session token-mutation authority gap
  identified in #99's source review is repaired in descendant #99.
- PR #101 supersedes closed, unmerged #96. Its latest observed head is
  baedf4b6670f6bf39ec593978ce38780c5640215. It adds cursor hardening, previous
  pages, bulk owner-session revocation, MFA challenge invalidation and migration
  0021. Preserve its active branch and refresh it before integration.
- PR #99 at the observed 47446b3 combines #90 with token authority, SQLite
  rollback/migration safeguards and UUIDv7 job allocation. Its COMMENTED source
  review is not a completed CI gate or an APPROVED review.
- Reconcile the recorded catalogue/documentation conflicts and session query
  plan test conflict, then qualify the combined tree. Source-head checks do
  not qualify a later combination.

## Safely actionable work still tracked

- Remaining UX-047 handlers/templates beyond the completed catalogue flows.
- Requirement evidence that can be established by meaningful automated checks;
  source citations alone do not establish acceptance criteria.
- Further storage-failure qualification when a suitable isolated writable fault
  harness is available. The new export adapter tests do not close physical
  read/fsync/truncate, device, checkpoint-sync, or uncertain-commit coverage.

## Preserved boundaries

- All 15 future modules stay isolated. #92's feed work remains disabled/HOLD;
  dedicated module tests are not authorization for Core integration.
- SEC-026 signals, GOV-021 recovery, reviewer-reset policy and other documented
  product-policy decisions retain their existing holds.
- Manual screen-reader qualification and SEC-033 independent external review
  cannot be replaced by these automated/AI-assisted checks.
- Licence, reference hardware/release thresholds, support commitments and other
  recorded owner/external decisions remain open where not explicitly settled.
- Deployment and release publication are excluded by the current directive.

Full Core completion must not be claimed while the actionable items above remain.
