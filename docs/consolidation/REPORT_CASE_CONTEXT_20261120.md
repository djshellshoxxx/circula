# Explicit report case context — 2026-11-20

## Accepted relation

[SEC-025](../specs/09-security-privacy.md) connects abuse reports to relevant
disputes. [FR-024](../specs/04-functional-requirements.md) preserves the
administrator report record and reporter confidentiality, while
[PER-003](../specs/02-personas-journeys.md) limits independent reviewers to
the information needed for their case.

Member and listing reports can now include an optional `dispute_id` when
created. The service checks the selected case, its transaction, the reporting
member and the target in the same write snapshot:

| Report target | Required relation |
|---|---|
| Member | The target is the other member party to this dispute's transaction |
| Listing | The target is the transaction's stored listing, owned by the other member party |
| Transaction | Uses the existing derived relation to disputes on that transaction; no explicit case field is needed or accepted |

The reporter must be an active party to an ordinary two-member transaction.
A reviewer or administrator role alone does not qualify a nonparty to file
a case-linked report. Case status, case age and the original dispute filing
window impose no additional reporting restriction. A paused or archived
listing remains a valid already-recorded relation, without exposing a title
hidden by ordinary listing visibility.

Reports without a selected case continue to work. Existing reports acquire
no inferred links, and a report cannot be attached, detached or redirected
after creation. The existing one-open-report-per-reporter-and-target rule
and daily quota remain unchanged across cases.

## Interface and confidentiality

Eligible parties see actions on their authorized dispute page to report the
other member and, where applicable, the related listing. The existing report
form shows the other member, case opening date and a link back to the selected
dispute. The relation is carried with the form and retained after validation
errors; the user does not have to enter internal identifiers. Successful
submission returns to the selected dispute. Existing context-free report
entry points and their destinations remain unchanged.

The API accepts optional `dispute_id` on report creation. Full report responses
include that field and `related_dispute_url` only when explicit context exists.
The administrator queue shows a link to that exact authorized dispute.
Transaction reports retain `related_disputes_url`, which lists all authorized
disputes on their exact transaction, including cases created later.

The existing independent-reviewer projection now selects both transaction
reports on the case's transaction and member/listing reports explicitly attached
to that exact case. This selection and the existing report-subject exclusion
occur before pagination. Each page rechecks current independent authority in
the same read snapshot. The projection still contains exactly report reference,
validated category and handling status: no target type, target ID, reporter,
notes, handling identity or timestamps. The existing API capability field
`transaction_reports_url` continues to lead to this same-dispute projection.

Ordinary parties receive no report-existence count, summary or review link.
Their creation actions depend only on the authorized case relation. Existing
administrator queue, handling, audit/replay and export-conflict checks retain
their current roles and subject exclusions. No new escalation authority is
introduced.

## Migration and compatibility

Migration `0025_report_dispute_context.sql` follows the real historical-limit
migration 0024. SQLite cannot add a composite foreign key with `ADD COLUMN`,
so 0025 rebuilds the same logical `reports` table inside the migration runner's
existing transaction and verification gate. All twelve original columns,
values, defaults, nullability, IDs, the reporter membership foreign key and
the three existing indexes are preserved. Every historical row receives SQL
`NULL` for the new column. The new relation has:

- A nullable `dispute_id` constrained to member/listing report targets.
- `FOREIGN KEY (network_id, dispute_id) REFERENCES disputes(network_id, id)`;
  the parent pair already has a unique constraint. A non-null reference must
  belong to the report's network. Context-free values bind SQL `NULL`, not an
  empty string, so the composite foreign key is correctly optional.
- A case index and a null-safe update trigger that prevents attaching,
  removing or redirecting context after insertion.

Legacy named-column inserts remain valid and receive null context. Current
readers scan the nullable value explicitly. The migration-23 fixture now tests
its own fixed 22-to-23 boundary using the original report columns, so it does
not depend on fields introduced by later migrations.

The new service request field is omitted from the idempotency fingerprint when
empty. The original context-free payload therefore retains its exact digest.
Explicit context is included when present; reusing a key with another case is
refused. Existing HTTP cached responses retain their original bytes, and
durable replays reload the original report and its immutable context. Report
records were not part of the downloadable export tables; those projections
and closure export versions remain unchanged. The personal-data inventory
classifies the added case reference alongside the existing report references.

## Qualification

Focused fixtures cover matching and mismatching party/target/case relations,
exact-case reviewer pagination, subject role overlap, legacy and explicit
replays, the HTML journey and validation retention, OpenAPI compatibility,
and migration preservation, late-DDL rollback, retry, reopen, nullable FK
behavior and context immutability. Formatting and JSON/YAML parsing passed.

Qualification used Go 1.26.11 with race instrumentation, one CPU and the real
0024/0025 migration chain. The focused store, API, HTML, catalogue and OpenAPI
contract selections passed at `13e070d`. The service selection subsequently
passed 19 top-level cases and 44 subcases at `94add4a`; its remaining case
passed at `1f3c5c1` after a fixture correction. Together, all 53 selected
top-level cases and 75 subcases have passing evidence, without skips or race
reports. Migration 0025 preservation, late-DDL rollback, retry, reopen and
relation-constraint checks passed in 2.26 seconds.

Two test-only corrections are retained in the execution history: the expected
error table now uses the domain error type required by the existing assertion
helper, and the read-only snapshot assertion runs before the administrator
audit query, which intentionally records its own `audit.read` event. No
production source changed between the qualification runs. The earlier failed
attempts remain in `analysis/report_case_context_verification.json` and its
three raw logs. This is focused qualification; the parent integration owns
the final combined repository and provider gates.

The remaining distinct decisions are system-account transaction subject
semantics and the all-conflicted operational procedure. Ordinary member,
listing and two-member transaction reporting do not depend on those decisions.
