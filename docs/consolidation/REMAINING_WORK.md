# Remaining Core work and integration gates

## Current candidate

1. Finish broad race/regression, contract, fuzz, browser, static and dependency
   qualification for the actual candidate tree. Preserve genuine failures and
   fix repository defects without weakening tests.
2. Complete the isolated listing/profile/directory and JavaScript catalogue
   slices, review their diffs, and merge only verified compatible changes into
   the authoritative candidate.
3. Publish the candidate through the connected GitHub provider, open/update its
   PR, and require exact-head CI before merging. The Git CLI can read the
   repository but its attempted push had no CLI credentials; the authorized
   GitHub connection supplies the write path.
4. Refresh the 114-branch snapshot for new or moved heads and give new work an
   explicit disposition. Preserve the initial exact snapshot as evidence.
5. Record any actual main merge and re-evaluate pending Core branches.

## Active independent Core work

- PR #90 owns the broad reviewed Core integration and API-036. It includes five
  other source PRs plus session, financial, authentication, and dispute fixes.
  Its head was observed moving during this continuation; always fetch again.
- PR #96 owns the active cursor-hardening follow-up. Do not duplicate unfinished
  implementation or merge a tests-only/unfinished state.
- Review current migration, authorization, bounded inventory, revocation,
  concurrency, regression and provider evidence before any eventual integration.

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
