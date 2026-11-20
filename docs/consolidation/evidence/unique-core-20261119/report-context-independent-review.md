# Explicit report case context: final bounded source review

## Scope

Reviewed context commit `b00706e88264ca0299c1011a9b26c5b0ac4eb8c8` on the
combined root checkout first at `5c389fe` and then at `f55ab56`. The latter
also contains the core historical-limit decoder follow-up, the historical
migration fixture adapter and the exact migration-25 rebuild registry entry.
The context runtime paths still match the atomic context commit. Consumer
coverage fields, catalogue additions and OpenAPI additions were inspected for
interaction with the context change.

**No new concrete runtime defect was identified.** This was read-only source
review. No additional Go command, disclosure scenario, guard bypass,
exploitation workflow or live-target test was executed. The already tracked
service test-fixture type correction and its pending rerun are qualification
work, not a runtime defect discovered by this review.

## Exact relation and authority

`internal/service/report_context.go` requires an active member, resolves the
case and transaction in the same network, checks party membership, and derives
the other member through the existing two-member transaction subject helper.
Listing context is offered only for the transaction's stored listing owned
by that other member. There is no inferred relation to another listing from
the same owner. A nonparty administrator/reviewer role grants no creation
capability. Current read failures propagate; only actual dispute not-found
errors are normalized.

`createReport` reuses that helper in its existing write transaction and
compares the submitted member/listing target against the derived exact target.
Transaction reports reject explicit case context and retain their established
derived relation. The pre-existing target checks, daily quota and one-open
report per reporter/target rule remain in force across cases. Case status and
age impose no new reporting policy. Creation writes the relation once;
handling updates only its original handling fields.

## Private projection and pagination

`ListDisputeReports` still opens the principal, checks current independent
authority, reads the dispute/transaction and selects reports in one view.
The retained error-propagation repair is present. The repository query adds
one parenthesized relation: explicitly attached member/listing reports for
that dispute, or transaction reports whose target is that dispute's exact
transaction. Network restriction, subject exclusion and relation filtering
all precede the cursor predicate, ordering and limit.

The service and API reviewer projection remain explicit three-field DTOs:
report ID, validated category and handling status. They do not embed the full
report or include target/context IDs, reporter identity, notes, handlers or
timestamps. Existing authority-only capability links do not read report
presence. Party creation links depend only on the authorized case relation.
Full queue/handling/audit and export-conflict decisions continue to use their
existing subject exclusions and authority; adding context does not create a
new escalation path.

## Migration and retained data

Migration `0025_report_dispute_context.sql` preserves all twelve original
report columns and copied values, defaults, nullability, the reporter-member
foreign key and all three original indexes. Existing reports receive SQL
NULL for the new relation. A non-null relation has a same-network composite
foreign key to the dispute and is restricted to member/listing targets. The
null-safe update trigger prevents attaching, detaching or redirecting it
after insertion. Legacy named-column inserts still receive NULL.

The migration runner's transaction and per-migration invariant gate remain
unchanged. The preservation fixture exercises late index failure after the
copy/rebuild, rollback, retry and reopen, compares old values and schema shape,
and checks the nullable relation and unchanged historical inserts. The registry
allows only the exact reviewed compatible rebuild, not a generic DROP/RENAME
exception. Its pinned SHA-256 was independently checked against the SQL:
`4279c438c595504ec9b37af03ad9283f8ad23ed1d76d36c4e82159b84e28b29b`.

The separate migration-23 fixture now fixes its own 22-to-23 boundary. Its
test-only prior-credit adapter refuses adopted schemas and omits only the
witness operation that did not exist in the old release; real entry, balance,
transaction, audit and idempotency operations still reach the database. This
does not change production posting or migration behavior.

## Retry compatibility and additive interface changes

The appended `ReportInput.DisputeID` has `omitempty`; when absent, the original
four-field JSON payload retains its exact order and bytes for durable request
fingerprints. Explicit context becomes part of the fingerprint, so a reused
key cannot redirect a report. Cached HTTP responses retain their original
bytes; durable reads recover the original report and its immutable context.
The source fixture independently hashes the old four-field structure to check
the historical digest.

Full report responses expose optional exact-case fields, while the existing
transaction relation field keeps its prior behavior. The HTML form labels the
case with the other party and opening date, carries its selected relation
through validation errors, and returns to the case on success. Paused/archived
listing relations remain usable, while their hidden titles are not exposed.
User text is rendered through normal escaping. Report keys and the coverage
catalogue keys are separate; their formatting cases are compatible. OpenAPI
keeps old required fields and the existing reviewer DTO shape, adding only
the optional context and optional historical-coverage objects in their own
schemas.

The historical coverage consumer is the reviewer's own implementation, so this
integration inspection is not presented as independent review of its authorship.
Parent separately reviewed it and received its exact-commit focused race receipt.

## Qualification boundaries

The context worker owns its focused migration/service/API/HTML/contract runs.
Parent reported the actual migration preservation/rollback/retry fixture passed
before pinning its digest; the broader context run and known fixture correction
were still in progress during this review. No full integrated pass is inferred
from those partial reports. The consumer's separate 23-test race qualification
does not substitute for context or repaired-core qualification.

The decoder follow-up `61ed3e7` / root `e3f2a5d` was also reviewed. It addresses
the independently reported null-policy/default-limit finding while retaining
legitimate omitted defaults. Its source-review closure and pending test
provenance are recorded in `historical_limit_core_independent_review.md`.

This report is AI source review, not GitHub approval, external security
qualification, final integrated acceptance or a deployment claim.
