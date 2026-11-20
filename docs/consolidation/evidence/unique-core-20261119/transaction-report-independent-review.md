# Independent review: ordinary transaction reports

## Result and scope

No actionable blocking regression was found in the reviewed transaction-report change or its source integration with the migration integrity gate and UI completion.

Reviewed on 2026-11-20:

- Worker commit `d885af837b7d5b1a10c7b0c885265a6504b684dd` against `01685005573bf36bcbec07f06a4c14c8bb4a73fa` in `/workspace/scratch/4d30fa819181/circula-transaction-report`.
- Integration commit `f4fce9b62e812da756bb3c0078f264f7409e557b` on root `e8c8f6abf52fc9a56dcc5246d0d9eb0d83b1b57d`, including migration gate `d65f5a0` and the progressive-loading/authentication/About changes.
- The existing report privacy, audit correlation, export authorization, replay authorization and dispute pagination boundaries used by the new target.

This was a source review and regression-test assessment. No production files were edited; no Go, browser, live-target or vulnerability reproduction tests were run. Read-only `git diff --check` succeeded for both the worker range and integrated transaction-report commit. This review is not a GitHub approval or an external security qualification.

## Party membership and subject resolution

Creation requires the existing active member and writable-network rules. A transaction target goes through `loadParty`, which uses the current network and requires the caller to be one of the transaction parties. An unrelated member or nonparty administrator gets the ordinary not-found result. The new helper accepts only two distinct member parties and selects the member opposite the reporter; it does not infer a subject from `CreatedBy`, an original transaction or a system account. Both payer-reporting and payee-reporting directions agree with the repository predicate.

The existing category, bounded note, daily quota, duplicate-open constraint, audit and idempotency mechanisms are reused. Creation does not change transaction status, entries, balances or disputes. Reporting is deliberately independent of completion state and dispute filing deadlines, matching the documented ordinary-report scope. Two-member reversals have the same unambiguous subject rule.

The HTML report action is shown to active parties with two distinct member accounts. Form resolution rechecks that condition, and the service repeats the authoritative check inside the write transaction. Administrative visibility of a transaction alone does not confer permission to report it as a party.

## Existing privacy boundaries

`reportIndependentOfMemberSQL` now derives the transaction's subject through network-scoped transaction, account and member joins. Missing member parties, a reporter who is not either party, and unsupported same-member/system-account subjects produce no independent-subject match. The existing member and listing branches are unchanged.

Queue reads apply that predicate before their keyset boundary and LIMIT. Audit reads use the same predicate in the existing correlated `EXISTS`, scoped by both report ID and network ID, before the audit sequence boundary/limit. The service independently checks each returned report in the same read snapshot. Individual handling and durable replay use `loadIndependentReport`; cached handling replay continues to recheck current administrator authority and that same helper. A later role assignment cannot make the subject independent.

Audit/full export admission, the eventual export snapshot, retained download authorization and cached export replay retain their shared conflict check. Adding the transaction branch extends that existing check to transaction-report subjects, including handled reports. Independent administrators and the existing System principal continue to receive complete audit-bearing exports; no filtering of immutable export rows, new reader role, or escalation privilege is introduced. Member/ledger exports keep their established scope.

Report audit subjects and handling notifications remain limited to the reporter. The new dispute projection contains no report ID, reporter identity, note or report-existence marker. Existing privacy fixtures were extended to transaction targets at service, API and HTML boundaries, including both directions of subject resolution.

## Dispute relation, authorization and pagination

The report API adds an optional `related_disputes_url` only for transaction targets. The HTML administrator queue adds the equivalent link. Both point to the ordinary dispute list with the exact transaction ID in the current network; neither stores a stale snapshot of current cases nor selects only the newest dispute.

The API now delegates to the existing `FindDisputes` implementation. Its authority is equivalent to the prior list operation: current administrators/reviewers can see network disputes, and ordinary members are constrained to transactions in which they are parties. The transaction predicate narrows those rows before LIMIT. An unrelated ordinary member obtains no related cases. Following a relation link does not confer report access or additional dispute authority.

API pagination binds the transaction filter alongside the existing status, path, network and principal context. The filter's UUID syntax and the validated status vocabulary keep the added delimiter unambiguous. With no transaction filter, the old fingerprint is unchanged. HTML next-page links retain the transaction ID, and the shared service validates the page boundary. Existing case history and later-created cases remain reachable through the relation.

The API field is omitted for existing member/listing reports, retaining their existing JSON field set. The published request/response enums and dispute query schema are updated. The new contract test validates create, queue, related-dispute list and handled-report responses against OpenAPI.

## Migration 0023 and the integrated gate

The table rebuild copies all twelve report columns, preserves their defaults/nullability and the composite reporter-membership foreign key, and recreates the duplicate-open, queue and rate indexes. Comparison with migration 0008 shows the intended target-type CHECK expansion is the only table contract change. There are no existing inbound report foreign keys or report-table triggers that would introduce a new cascade or trigger effect during this rebuild. Report queue ordering and audit correlation use stable report IDs, not the table's internal rowids.

In the integrated source, the existing migration transaction encloses baseline verification, copy/drop/rename/index creation, after-verification and schema-version recording. The baseline sees schema 22; the after-check sees schema 23. Both use the same locked connection. The gate's ledger/audit evaluator reads complete audit history with an empty report exclusion filter, so report table rebuilding and the extended subject predicate do not remove evidence or require a schema-specific verification exception. No financial or audit table is rewritten by migration 0023.

The migration regression fixture deliberately causes the final index statement to fail after copy/drop/rename. It checks rollback of durable rows and schema metadata, absence of the temporary table, restored table SQL/constraints/indexes, successful retry, a second open, duplicate and foreign-key enforcement, and successful creation of the new report type. This is relevant coverage for late DDL failure. The worker execution preceded integration of the integrity gate, so combined schema-23 execution remains part of root qualification rather than an execution claim of this source review.

## UI merge and test provenance

The integrated production paths match the reviewed worker source except for the expected catalogue merge. The root catalogue retains the loading, authentication and About messages from `e8c8f6a` alongside the transaction-report messages. Its catalogue-format test retains both sets of argument cases. The report links and forms use native navigation/submission and require no special interaction with the loading listener; no attachment behavior is changed.

Test sources reviewed include transaction creation/replay and immutable ledger/history checks; proposed, cancelled, past-window and reversal cases; unrelated/cross-network/system-account refusals; subject privacy in queue/audit/handling/replay/exports; API cursor binding and multi-page dispute history; the HTML report journey and 26-case pagination; migration rollback/retry; and the API contract test.

The reviewer inspected `analysis/transaction_reports_verification.json` and independently parsed `analysis/transaction-report-focused.jsonl`. The raw log contains **31 top-level passes and 58 subtest passes**, with passing package outcomes for store, service, API, web, messages and contract, no failed/skipped tests, and no race-report marker. Its SHA-256 is `d54af130b4a9ed2622b2ec5ced4b0a5946bc6847c99310a3712a6b6145bc7b53`, matching the receipt. The receipt SHA-256 is `8d99a3345f070cbabc9d99a71846fea357d9c5703c2572be64a16b1c04f1b599`.

The receipt records Go 1.26.11 with `-race -p 1 -count=1`, `GOMAXPROCS=1`, `GOGC=20`, and `GOMEMLIMIT=256MiB`, and identifies worker commit `d885af8` and its source hashes. All 31 recorded file hashes matched the reviewed worker checkout. These are inspected author-run results, not tests rerun by this reviewer or evidence of a full consolidated-head pass. The ordinary transaction slice remains distinct from the explicitly documented system-account subject semantics, reviewer report projection, and all-conflicted operational decision.
