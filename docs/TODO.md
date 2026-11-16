# Circula current work

Updated 2026-11-16 on `codex/circula-core-completion-20261116`. This list supersedes the old
unmerged-slice checklist. Scope: core Circula; the 15 future modules remain isolated.

## Engineering in progress

- Final current-head race, lint, contract, browser and dependency checks; then merge the consolidation.
- Review the remaining historical Low/Info audit findings against current source. Earlier reports
  are evidence of their recorded revisions, not a reliable list of current defects.

## Completed engineering in this consolidation

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
- CI already avoids duplicate branch/PR runs and reserves the platform matrix for main/nightly/dispatch.
  Container evidence is recorded in `docs/ops/container-verification-2026-11-12.md`.

## Remaining specification and qualification work

- QA-001 long property-run budget/thresholds (DEC-023); short invariant sequences exist.
- QA-042 source/test citations and per-requirement dated result reporting remain incomplete. Generated
  traceability detects stale inventory but is not proof that every requirement is qualified.
- ARC-002 dual-adapter equivalence coverage, ARC-004 driver separation and ARC-008 operator surface audit.
- L9 real SQLite I/O-failure coverage beyond interface fault injection and process-crash tests.
- UX-017 optional original-transaction link for adjustments requires a defined data/API contract.
- SEC-014/018 retention choices for time zone, last activity and challenge reasons remain documented questions.
- Refresh historical spec status tables only against specific implementation and test evidence.
- First tagged release pipeline and deployment qualification; release thresholds and supported pilot jurisdictions.

## Owner and external gates

- Choose the project licence; confirm outstanding policy decisions in `docs/DECISIONS.md` and relevant addenda.
  Several descriptions there predate built mechanisms: MFA, recovery, dual control, break-glass, grants,
  role-specific session lifetimes, policy notices and objections exist. A stale decision row is not a missing feature.
- Break-glass scope (A3-F02 / DEC-014), reviewer-versus-administrator decisions (DEC-072), locale scope (UX-047),
  support commitments and notice defaults require owner policy confirmation where records remain unresolved.
- Manual screen-reader qualification (UX-056 / QA-035) and independent security review (SEC-033).
- Hosted wiki publishing remains separate from the versioned `docs/wiki` source.

Do not mark the entire specification complete from the 45-row beta feature matrix. Do not treat automated
accessibility checks or AI-assisted security review as completion of the manual/external gates.
