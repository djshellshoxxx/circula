# Circula current work

Updated 2026-11-16 after merging PRs #56 and #57. This list supersedes the old
unmerged-slice checklist. Scope: core Circula; the 15 future modules remain isolated.

## Integration evidence

- [PR #56](https://github.com/djshellshoxxx/circula/pull/56) merged core completion after seven passing CI jobs.
- [PR #57](https://github.com/djshellshoxxx/circula/pull/57) merged native-platform fixes and real WAL-write recovery qualification after all nine CI jobs passed, including 579 browser checks.
- [QA-001 execution evidence](qa/QA001-2026-11-16.md): 100,000 distinct sequences and 2,800,700 generated operations passed with no failures, gaps or overlaps.
- Fourteen superseded PRs were closed after verifying their heads are ancestors of main; #55 was automatically marked merged. All branches remain preserved.
- Historical reports remain evidence of their recorded revisions; current remediations and
  qualification limits are in the core audit record.

## Completed engineering in this consolidation

- PER-022 financial-page snapshot timestamps are present without JavaScript; offline or restored-history notices preserve the earlier time and never clear merely because connectivity returns.

- FR-067 stale-proposal attention link and paginated administrator queue, with matching count/age/expiry predicates, authorization and neutral party wording.
- Real Linux checkpoint database-write failure preserves an already committed trade and WAL through restart, idempotent replay, successful checkpoint retry and reopen.

- Optional adjustment links, including dual-control approval, immutable metadata, visible history and
  API/export support; migration 0018 preserves existing export bytes when no links exist.
- Driver-neutral repository contracts with old-store aliases; 43 paired MVP HTML/API operations and
  six explicitly API-only lifecycle/token operations have ledger/audit equivalence coverage.
- Atomic policy revisions, bounded operator-configurable authentication throttles, audited operator
  provisioning and rendered reversal balance reads, and read-only scheduled ledger verification.
- Record UUID validation across HTML/JSON/filter inputs after authorization; strict raw and unknown
  API query validation; required password-change navigation and suspended/departed read links.
- Complete reversal audit targets and independence checks beyond 500 reversals; recovery-throttle
  erasure; structured timestamp markup and destructive migration guards.
- Real SQLite disk-full and Linux WAL-commit write-failure rollback/retry/reopen tests, with explicit
  qualification limits. QA-042 records test layers and real dated per-test outcomes separately from
  deterministic requirement citations.
- QA-001 isolated seed replay, strict disjoint ranges, complete failure fixtures and an explicit manual
  long-run workflow. The full 100,000-sequence non-race run passed on the recorded source; thresholds remain unconfirmed.
- Restore interruption guards and recovery instructions, HMAC-v2 authenticated backup summaries with legacy compatibility,
  complete reversal histories beyond 500 entries, live temporary-proposer approval, and invitation-only join privacy.
- Freeze-safe bounded proposal/adjustment expiry, governance cursor pagination, and signed confirmation
  flows for re-review decisions and adjustment approval.
- Effective administrator read/navigation authorization, scoped-token confinement, exact JSON field names,
  private decoder errors and cookie-free public assets.
- Atomic credit-limit conditional updates, pending-policy-preserving operator edits and login normalization.
- Credential invalidation before restore swap; original closure-export checksum compatibility across migration 0017.
- Integrated the beta follow-up and documentation upkeep. The preserved WIP slices are already in the beta history.
- Export disk spooling, capacity reservations, active expiry, download pinning/revocation and shutdown cleanup
  (`internal/api/export_*test.go`); erasure invalidates affected member and administrator archives.
- Current authorization on cached API responses (`internal/api/replay_*test.go`).
- Immutable history insertion ordering and migration guards (`internal/store/event_order_test.go`,
  `migrate_version_test.go`); no dependence on wall-clock ordering.
- Rapid backup naming, old-name retention and scheduling under clock skew (`cmd/circula/backup_test.go`,
  `jobs_regress_test.go`).
- Independent reconciliation reader (`scripts/test_read_export.py`, actual generated member archives
  in `internal/service/export_schema_test.go`).
- Logout error handling, Unicode throttle cleanup, precise notification erasure, and retryable closure
  final exports (the named regression tests in `internal/web`, `internal/store`, `internal/service`).
- Same-network visibility and equal password-hash work; timestamp-based historical balances are implemented.
- QA-019 decline/expiry through API and HTML forms; optional listing labels; traceability generation/check in CI.
- CI avoids duplicate branch/PR runs; native platforms run on main/nightly/dispatch and opted-in PRs.
  Container evidence is recorded in `docs/ops/container-verification-2026-11-12.md`.

## Remaining specification and qualification work

- SEC-026 is an outstanding implementation: network-configurable proposal-velocity and reciprocal-trade signals, one signal per window, visible only to same-network administrators/reviewers, with no automatic trade blocking. Existing throttles are not this feature.
- UX-047 message catalogues and catalogue-based plural/number formatting remain outstanding engineering work; translation/locale scope still needs product confirmation.
- GOV-021 administrator-unavailable recovery needs its specified dual approval, member-identity verification, network-wide notice and no-balance-change qualification, aligned with the unresolved break-glass scope. Do not count generic operator break-glass as this complete recovery flow.

- Confirm QA-001 release budgets/thresholds (DEC-023) and reference-hardware performance criteria.
  The 100,000-sequence execution itself is complete; it does not approve a release gate.
- QA-042 remaining inspection/manual/external citations and full acceptance-criterion qualification.
  Source/test mappings and matching dated execution evidence do not prove untested criteria.
- L9 read/fsync/truncate, other device/write failures, checkpoint sync/truncate and uncertain commit outcomes
  beyond the covered disk-full and pre-commit-marker WAL write failure; see the storage qualification note.
- SEC-014/018 retention choices for time zone, last activity and challenge reasons remain documented questions.
- Refresh historical spec status tables only against specific implementation and test evidence.
- First tagged release and deployment qualification; release thresholds and supported pilot jurisdictions.

## Owner and external gates

- Choose the project licence; confirm outstanding policy decisions in `docs/DECISIONS.md` and relevant addenda.
  Several descriptions there predate built mechanisms: MFA, recovery, dual control, break-glass, grants,
  role-specific session lifetimes, policy notices and objections exist. A stale decision row is not a missing feature.
- Break-glass scope (A3-F02 / DEC-014), reviewer-versus-administrator decisions (DEC-072), locale scope (UX-047),
  operator dual-control exceptions (ARC-008), member-limit notice timing (A3-F18), support commitments and
  notice defaults require owner policy confirmation where records remain unresolved.
- Manual screen-reader qualification (UX-056 / QA-035) and independent security review (SEC-033).
- Hosted wiki publishing remains separate from the versioned `docs/wiki` source.

Do not mark the entire specification complete from the 45-row beta feature matrix. Do not treat automated
accessibility checks or AI-assisted security review as completion of the manual/external gates.
