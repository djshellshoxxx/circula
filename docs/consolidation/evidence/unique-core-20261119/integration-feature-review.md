# Independent review: accepted feature completion

## Result and provenance

No actionable blocking regression was found in the three bounded scopes below. This is an independent AI source review, not a GitHub approval, external security qualification, or an execution claim for the full product acceptance suite.

Reviewed on 2026-11-20, using the root checkout at `e8c8f6abf52fc9a56dcc5246d0d9eb0d83b1b57d`:

1. Migration integrity gate: `d65f5a0b6de0b3c6dc10c1ebfdc0407a0f72b707`, including the extracted integrity evaluator, service delegation, migration transaction sequencing, and regression tests.
2. Progressive loading, authentication messages, and network About: `e8c8f6abf52fc9a56dcc5246d0d9eb0d83b1b57d`, including production templates/scripts, route authority, catalogue formatting, and test sources.
3. Posting outcome metrics: the stable uncommitted change in `/workspace/scratch/4d30fa819181/circula-posting-metrics`, based on `01685005573bf36bcbec07f06a4c14c8bb4a73fa`. The production snapshot is identified by the hashes below. Both new untracked service test files and the modified metrics test were read.

No production files were edited. No Go, browser, or vulnerability reproduction tests were run by this reviewer. The reviewer ran read-only source inspections and `git diff --check`, which reported no whitespace errors for either the committed migration/UI range or the metrics working diff. Earlier review provenance remains in `integration_independent_review.md` and `integration_second_review.md`.

## Migration integrity gate

The shared `internal/integrity` evaluator preserves the prior ledger checks, audit traversal, finding text/order, and audit hash construction. `service.AuditHash` remains available and delegates to the same canonical byte construction: sorted copied subjects, the same field ordering and separators, millisecond timestamp, and SHA-256 hex. The fixed-hash compatibility test pins a historical value independently of fixtures that use the new helper. Service authorization, read snapshots, timestamps, and result persistence stay with the service layer; ordinary verification uses zero-valued compatibility options.

Each pending migration evaluates the baseline, executes its SQL, and evaluates the result on the same `sql.Conn` inside its existing `BEGIN IMMEDIATE` transaction. New findings or incomplete verification refuse the unit before recording the migration version. Existing rollback/discard behavior handles refusal and COMMIT failure. Findings are compared as `(scope, message)` pairs, so a finding in one network cannot excuse a newly introduced finding in another. The comparison deliberately permits an unchanged pre-existing inconsistency; it does not relabel that database as healthy or authorize service writes.

The schema-1 exception skips only the unavailable recorded-head comparison after the chain itself has been checked. Migration 0002 immediately enables head checking, including platform scope; a missing heads table in a later schema is an error. Empty fresh databases and platform-only history follow the same checks. Migration checks use unfiltered repository reads, so report-view confidentiality filters do not remove evidence from this system-level verification.

Test sources cover newly introduced balance, reversal-total and platform-head findings; a verification query failure; unchanged versus additional existing findings; baseline read failure; schema-1 head transition; rollback persistence after reopen; and ordinary retry. These provide relevant coverage for the new admission boundary. The actual 10,000-transaction migration and historical upgrade suites provide additional execution evidence listed below.

## UI completion

The loading listener observes native click and submit events without cancelling submission, disabling successful controls, moving focus, or reconstructing request bodies. It uses the actual submitter overrides for method, target and action. A deferred check respects another listener's cancellation. Native invalid submission does not emit the submit event, and pageshow clears both pending and displayed state after history restoration.

All three existing attachment forms (member history export, leave-page export, administrator export) explicitly opt out. Fragment-only navigation, download links, modified clicks, other windows, external destinations and non-navigation methods are excluded. The live status is outside the region marked busy, uses text content, and starts empty for the no-script page. The new tests check the status independently of existing session-warning behavior.

The About endpoint is wrapped in the existing `netH` path, retaining authentication, forced-password-change, current membership and network-read authority. Its footer link uses the current network slug, and its content reuses the approved acceptance statement plus the existing rules link. The route inventory is updated. Tests include anonymous/non-member refusal, suspended/departed read access, closure read access and erasure-related refusal.

Authentication sentence changes use existing escaped `msgAround`/`msgAroundCount` boundaries. Trusted template links and code elements remain template-owned. Counted recovery sentences keep singular/plural selection separate from localized numeric display, including the 1,000 case; no removed fragment key remains referenced by the updated production templates.

## Posting outcome metrics

The original two-method `Observer` interface is unchanged; `PostingObserver` is an optional extension. Existing observers therefore retain source compatibility and committed/replay behavior. The central posting function records its returned outcome with a named return and deferred callback; successful posting work is counted as committed only when the enclosing write unit succeeds. A later audit, idempotency or COMMIT failure is attributed as an observed posting failure. The documented uncertainty of a failed COMMIT is preserved.

The reversal capacity preflight records a failed attempt before returning. This covers the existing dispute path that commits a `posting_pending` decision after a credit refusal without counting a committed posting. A successful capacity preflight proceeds to the normal central posting boundary, so it does not double-count. Durable replays bypass the posting boundary and add no attempt.

Unit statistics are reset and copied inside the existing serialized write callback, then reported after `Store.Update` returns. The per-unit failure map is replaced for subsequent units, so a completed unit's reporting does not share mutable failure counts with a later unit. With no observer, the prior direct update path remains unchanged. The original committed-posting and replay callbacks still run only after successful units.

Failure labels use a fixed nine-value vocabulary; both the service and metrics implementation constrain unexpected reasons to `internal`. The counter output carries no transaction, account, member, network, amount, or error-message labels. Nonpositive counts are ignored by the added metrics callbacks. Documentation correctly limits attempts to the central financial posting boundary plus the explicit reversal preflight, leaving earlier HTTP refusals and service-only authentication coverage separate.

The new service test source covers ordinary commit/replay/credit refusal, an enclosing-unit rollback followed by successful retry, and a committed posting-pending dispute. The metrics test exercises actual service events and bounded label output. These tests had not yet been executed when this source review was completed; they are not reported here as passes.

### Metrics snapshot hashes

| Path relative to the metrics checkout | SHA-256 |
| --- | --- |
| `cmd/circula/metrics.go` | `d8c67f7de57cb393e252d18b200253b5a90dfeebdae4b19d44365419947a22d0` |
| `internal/service/observe.go` | `57e65a50a9da820896dab5cf52db90f45d55aa27c15dc1646977243cebb204cb` |
| `internal/service/posting.go` | `64026a19adddb3bcf89a8c5239bb11295fcc88acd9d3870e691354709387b309` |
| `internal/service/reversal.go` | `20d99de6ee015ee1b0af162f47bffb1abab980cc543f69c9ffea746c707746fe` |
| `internal/service/service.go` | `a18b37f7e0afa6fcf08ce3506adcdf53815e7ecc94c2542aadbb8d7475abb1ac` |
| `internal/service/posting_observe_internal_test.go` | `f2fe1a1f51e06f411b6dffdda123c96acafaccc3530e45123d94d46d5428e268` |
| `internal/service/posting_observe_test.go` | `e40a76a0d93dcb744f47ef22054ad974a3574e94724f2d078da0b3db753ce8e7` |
| `cmd/circula/metrics_hooks_test.go` | `071720c4758eaf257024b16675b74311bc559a307baa8f798b4d5f0b752d6623` |

## Execution evidence inspected, not rerun

All paths below are under `/workspace/scratch/4d30fa819181/analysis`. Parent/author provenance identifies the Go runs as focused race runs; this reviewer read their recorded output rather than launching them.

| Evidence | What the retained output establishes |
| --- | --- |
| `migration-integrity-before.log` | The four new integrity-gate test groups failed against the earlier behavior. |
| `migration-integrity-after.log` | 24 top-level store/migration tests passed, including rollback/reopen, all historical schema versions, closure-checkpoint migration, and head backfill. |
| `migration-service-compatibility.log` | 14 top-level service tests passed, covering ordinary/platform verification, hash format, audit behavior, and restore authorization/atomicity. |
| `migration-audit-hash-before.log` | The fixed historical hash test passed against the previous hash implementation. |
| `migration-integrity-golden.log`, `migration-integrity-timing.log` | Both golden tests passed. The actual v1-to-v22 upgrade retained 10,000 transactions and 61 accounts; recorded migration time was 2m42.441s, verification 3.823s, fixture build 29.502s. This is the measured single-CPU race fixture, not a general production throughput claim. |
| `ui-loading-after.log` | 23 combined Node checks passed, including loading-event and existing keyboard/helper checks. |
| `ui-web-final.log` | Eight focused web tests passed for About authority, authentication templates and progressive shell formatting/escaping. |
| `ui-route-authority.log` | Route inventory and authorization matrix tests passed. |
| `ui-messages.log` | The messages package passed. |
| `ui-playwright-list.log` | Six browser cases were discovered in two files. This is test discovery only; it does not establish browser execution or accessibility qualification. |

The pending transaction-report work and any later changes to the uncommitted metrics snapshot are outside this review and need their own bounded follow-up. Full frozen-source qualification remains owned by the integration workflow.
