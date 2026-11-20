# Independent reviewer report context — 2026-11-20

## Contract and projection

[SEC-025](../specs/09-security-privacy.md) includes dispute reviewers in the
audience for abuse reports. [FR-024](../specs/04-functional-requirements.md)
keeps reporter identity in the administrator queue, while
[PER-003](../specs/02-personas-journeys.md) limits reviewers to information
needed for their dispute. The ordinary transaction report already identifies
its transaction and its two member parties.

This follow-up adds a separate read-only projection containing exactly:

| Field | Meaning |
|---|---|
| `id` | Stable report reference |
| `category` | One of the accepted report categories |
| `status` | Open or handled |

Reporter IDs and names, report notes, handling notes, handler identity and
timestamps are omitted. All free text is omitted because it can identify its
author or another member; the implementation does not guess whether a sentence
is identifying. Unknown stored categories or invalid projection fields yield
a generic unavailable-summary error without returning the stored text.

The projection has a dedicated service type and API DTO. It does not embed or
convert the full report DTO. Existing administrator queue responses and report
handling retain their established data and authority.

## Authorization and relation

`GET /api/v1/disputes/{id}/reports` and
`GET /n/{slug}/disputes/{id}/reports` require current active reviewer or
administrator authority and the existing dispute independence check. A party,
proposer, acceptor or reversal participant cannot use this projection merely
by gaining a reviewer or administrator role. The existing administrator
fallback in dispute authority and existing trusted System read authority are
preserved. There is no new assignment, escalation or handling privilege.

The service resolves the dispute, its transaction, the current role and all
independence checks in the same read snapshot as the report query. The caller
cannot select a report target: it is derived from the dispute. Only transaction
reports for that exact transaction and network are selected, with the existing
unresolved-subject exclusion retained for member authority.

Report filtering precedes pagination. Every page repeats authorization.
API cursors bind the dispute path, network and principal; a report-context
cursor cannot resume another dispute or the ordinary dispute list. HTML next
links preserve the dispute path.

An eligible reader receives a `transaction_reports_url` capability link in
the API dispute detail and an equivalent HTML link. These links depend only
on current independent authority, not whether reports exist. Ordinary parties
receive neither a report field nor a report count or link. General dispute
lists and their cursors carry no report context. Opening the linked page
performs a fresh authorization check.

## Boundaries

The projection covers the already accepted relation between a transaction
report and disputes on that exact transaction. The full report queue,
handling, report creation, replay, notifications, audit correlations and export
authorization remain governed by their existing paths. This read introduces
no database write or migration.

System-account transaction subject semantics and the all-conflicted
operational procedure remain outside this implementation. Member and listing
reports still need explicit case context before a reviewer can receive them;
sharing a member or listing alone does not establish that relation. These
remaining boundaries do not defer the implemented transaction context.

## Verification

Focused service, API and HTML fixtures cover the separate projection,
exact-transaction filtering before pagination, ordinary parties gaining
reviewer/admin roles, role revocation and suspension, reversal participation,
cross-network cases, invalid stored categories and unchanged full administrator
details. The API contract fixture and reviewed route authorization inventory
include both new read routes.

The focused Go 1.26.11 run used `-race -p 1 -count=1`, one CPU and a 256 MiB
runtime memory target. Five packages passed: store (2.654 s), API (26.409 s),
web (22.498 s), messages (1.045 s) and contract (8.820 s). These contain
26 top-level tests and 31 subtests, with no executed-test failure, skip or
reported race. They include the new API and HTML projection fixtures, the
existing report/privacy behavior, the route authorization matrix and the full
OpenAPI operation fixture. Migration 0023's preservation/rollback/retry fixture
passed against the integrated pre-commit verification gate.

The separate `service_test` compiler was killed before executing its tests in
both the initial 256 MiB run and a service-only retry at 128 MiB. Those attempts
remain recorded as inconclusive. After relocating the Circula caches from the
memory-backed temporary filesystem and using a workspace `GOTMPDIR`, the
unchanged service selection passed under race at the original 256 MiB setting
in 35.018 s: 15 top-level tests and 27 subtests, including all four new direct
service fixtures. No source or test coverage was changed for that retry.

The combined focused result is therefore **41 top-level tests and 58 subtests
passed across six packages**, with no executed-test failure, skip or reported
race. The raw JSON logs and their hashes are retained in the workspace
verification receipt. Integrated full-suite and provider qualification remain
separate. Formatting, catalogue JSON parsing, OpenAPI YAML parsing and
`git diff --check` passed.
