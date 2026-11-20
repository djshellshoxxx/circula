# Ordinary transaction reports — 2026-11-20

## Accepted scope

[SEC-025](../specs/09-security-privacy.md) requires a member to be able to
report a transaction to which they are a party, prevents disclosure of the
reporter's identity to the reported party, and requires a relation to related
disputes. The existing report contract in
[FR-024](../specs/04-functional-requirements.md) supplies the category, bounded
note, duplicate-open limit, daily limit, handling and reporter-only notification
behavior. This follow-up completes the ordinary two-member transaction case
using those existing mechanisms.

A transaction report identifies the transaction directly. Its subject is the
other member party. Both transaction accounts must belong to distinct
members, and the reporter must be one of them. This includes a two-member
reversal. Reporting does not require a completed transaction and does not use
the dispute filing deadline. It neither changes transaction terms or status
nor opens a dispute, posts an entry, suspends a member or hides a listing.

## Data and authorization

Migration `0023_transaction_reports.sql` widens only the report target
constraint. The table rebuild retains every report ID and all twelve columns,
their defaults and nullability, the reporter's membership foreign key, and the
existing duplicate-open, queue and rate indexes. Financial and audit rows are
unchanged.

The physical table rebuild is registered as a reviewed compatible expansion
in `internal/store/expand_contract_test.go`: SQLite cannot widen this CHECK
in place, but the logical table and every prior column, value, default, foreign
key and index remain present. The scanner still detects all DROP/RENAME forms;
only this named rebuild has an explicit preservation proof. Genuine schema
contraction still requires the documented earlier read switch and later release.
This is not an old-binary downgrade or rolling-upgrade guarantee. The migration's existing transaction encloses copy, drop, rename,
index creation and schema-version recording.

The same report subject rule applies to queue reads, audit correlations,
handling, durable replay and cached replay. The queue and audit repository
predicates exclude the subject before pagination. The service also checks
individual report access inside its current unit of work. A later administrator
role does not remove the conflict. Complete audit/full exports retain their
existing generic refusal for a member with any retained subject conflict;
independent administrators and existing System authority retain their
established access. The earlier
[privacy checkpoint](REPORT_PRIVACY_20261120.md) explains the complete-export
and all-conflicted constraints.

The subsequent [reviewer projection](DISPUTE_REPORT_PROJECTION_20261120.md)
provides independent dispute reviewers with a separate, read-only summary of
reports on that dispute's transaction. It preserves the full administrator
queue and handling roles; reporter identity and all report free text are
omitted from the reviewer projection.

## Related disputes

Transaction reports return `related_disputes_url` in the API response and
provide a related-disputes link in the administrator HTML queue. Both use the
ordinary dispute list filtered by the exact transaction ID in the same
network. The relation includes cases that existed before the report and cases
created afterward. Historical cases remain reachable through pagination.

The filter preserves ordinary party, reviewer and administrator dispute
authorization. API cursors bind the transaction filter along with the existing
status, network and principal context. HTML next-page links preserve that
filter. Neither the general dispute list nor a dispute detail acquires a
report count, a reporter field or a link that reveals report existence.
An independent administrator already authorized to read the report can follow
the linked dispute list under their existing authority.

## Remaining boundaries

- A one-member transaction involving a system account has no accepted rule
  identifying the reported party. The creating administrator, operator,
  network and original transaction counterparty are not interchangeable.
  These reports remain unsupported until that subject contract is set; the
  ordinary member-party case is complete without that decision.
- Listing and member reports have no transaction or case reference. Sharing
  a member or listing does not by itself establish which disputes relate to
  those reports, so this change derives no such relationship.
- The existing all-conflicted administrator case remains retained for the
  established operational decision. No escalation privilege, quorum,
  role-grant policy or operator report-handling command is introduced.

SEC-025 therefore remains marked partial for those remaining boundaries; its
ordinary transaction creation, subject protection and exact-transaction
dispute relation are implemented.

## Verification

Focused coverage is in `internal/service/transaction_reports_test.go`,
`internal/api/transaction_reports_test.go`,
`internal/web/transaction_reports_test.go` and
`internal/store/transaction_report_migration_test.go`.
`internal/contract/transaction_reports_test.go` validates the new target,
relation field and dispute query against the published OpenAPI document.
The existing service, API and HTML report privacy fixtures now include the
transaction subject.

The focused Go 1.26.11 race run passed **31 top-level tests and 58 subtests**
across the store, service, API, HTML, message catalogue and API contract
packages, with no failures, skips or race reports. It included both party
directions, ordinary completed/proposed/cancelled/reversal targets, reports
after the dispute window, related-case history across pagination, preserved
member/listing behavior, and the existing general dispute-list and trade-detail
link regressions. Migration coverage checks the final-index failure after
copy/drop/rename, exact rollback, successful retry and a second open.

The run used `-race -p 1 -count=1` with `GOMAXPROCS=1`, `GOGC=20`
and `GOMEMLIMIT=256MiB`. Catalogue JSON, OpenAPI YAML, Go formatting and
`git diff --check` also passed. This is focused qualification of this slice;
the consolidated head still requires its combined and provider checks.
