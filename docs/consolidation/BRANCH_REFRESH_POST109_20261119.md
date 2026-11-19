# Branch and PR refresh after initial PR #109 publication

This additive snapshot accounts for **128 remote branches and 109 PRs** at
**2026-11-19 12:26:52 UTC**, before any subsequent revision of PR #109. It
preserves the [initial 114-branch inventory](BRANCH_INVENTORY.md) and the
[120-branch checkpoint](BRANCH_REFRESH_20261119.md). Only new or moved heads
since the latter checkpoint were investigated; unchanged content dispositions
retain their previous provenance.

The [machine-readable evidence](evidence/branch-refresh-post109-20261119.json)
contains all 128 branch heads and dispositions, all 109 PR states and targets,
the 17 changed-head analyses, source-preservation comparisons, module/hold
metadata, review records and exact-head check receipts. The original 114-head
JSON remains byte-for-byte intact, SHA-256
`8989490d42896e49d05a105ee22dc76a7a4b7c76db47bee327aa8e8717f8b967`.

## Counts and observation boundary

| Category | Prior checkpoint | This snapshot | Change |
|---|---:|---:|---|
| Branches | 120 | 128 | Eight new, nine moved, 111 unchanged, zero deleted |
| Pull requests | 102 | 109 | Seven new drafts, #103–#109 |
| Open PRs | 20 | 25 | Seven opened; #98 and #101 merged |
| Merged PRs | 45 | 47 | #98 into isolated #92; #101 into #90 |
| Closed without merge | 37 | 37 | No additional closure |
| Known future modules | 15 | 15 | Follow-up branches add no new module identity |

The final enumeration used branch pages **100 + 28** and `state=all` PR pages
**100 + 9**, ordered by creation. All branch names are unique and PR numbers
1–109 are present. The preceding full observation at **12:20:39 UTC** had
127 branches and the same 109 PRs. During collection, one recurring-module
follow-up appeared and #90 received an eight-file documentation commit. Both
observations are recorded rather than replacing the earlier provenance.

REST pages are separate observations. This is a bounded snapshot, with
individual check timestamps, rather than a claim that concurrent branches will
stay at these heads. Standalone issues, tags, releases and deployments were not
reenumerated; their earlier counts remain historical.

## Main, integration target and initial continuation

| Reference | Exact head | Tree or scope |
|---|---|---|
| `main` | `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625` | Unchanged from both preserved inventories |
| [#90](https://github.com/djshellshoxxx/circula/pull/90), `integration/comprehensive-audit-20261119` | `839e86935767288e594914ad711486616a604eca` | Tree `9d23b7b81d60b07066444232dee65b0fda071b0d`; documentation-only child of Core992 |
| #90 runtime composition | `992478594e5902f57a39f3772739bee6ab495d24` | Tree `3725d720ce7fc41f148b560e5f06c8f9bf5ab221` |
| [#109](https://github.com/djshellshoxxx/circula/pull/109), initial publication | `afb0da286098cd1b64a7324170ebb7d178e5e1e1` | Tree `8a1013069819d6e81d7f6d9c5c863020a2b32c08`; sole parent Core992 |
| Local source used for this report | `3ecf76a63d24e2051bed151e431aeb6daf5ec63f` | Exact full tree equals initial #109 |

Initial #109 changes 75 paths over Core992. Its catalogue has **982 entries**:
all 525 Core992 values unchanged, plus 457 entries for the completed bounded
inbox/report, trade/history, network-picker, audit/export, administrator-amount,
time-zone and initial dispute slices. It also includes the Python compressed
stream-completion repair, its Go acceptance regression, and the WAL fault
fixture's separate query-only witness. This inventory does not rerun their
tests or transfer earlier receipts to a later resulting tree.

The separate dispute-confirmation 40-entry and terms/rules/leave 65-entry
slices were being integrated locally after this initial publication. They are
outside the recorded `afb0da2` tree; this snapshot makes no claim about their
subsequent publication or qualification.

### The late #90 documentation move

At **12:25:56 UTC**, #90 had moved from Core992 to `839e869`. Immutable Git
comparison verifies one additional commit and exactly eight paths, all under
`docs/audits/`: six reports and two new
`evidence/closing-publication-20261119/` JSON files. Runtime, tests, workflows
and `docs/TRACEABILITY.md` are identical to Core992. The eight paths have no
overlap with initial #109's 75 changed paths. Preserve this peer audit evidence
when composing the next candidate; it introduces no missing runtime/test work
into the already frozen continuation.

PR #109's list metadata still returned `base_sha=9924785` after the target
branch moved. The evidence retains that response and the separately verified
target branch head; it does not claim #109 already contains `839e869`.

## Eight new branches

Full SHAs, trees, parent relationships and changed-path evidence are in the
JSON. The short SHAs below refer to those same immutable objects.

| Branch | Head | PR / target | Disposition |
|---|---|---|---|
| `master/notification-transport-tests-20261119` | `13669485b641` | [#103](https://github.com/djshellshoxxx/circula/pull/103) → notifications topic | Four module files repair true-nil Web Push success, bounded Retry-After, and verified custom-dial SMTP TLS, with tests. Already preserved in #100; disabled/HOLD remains. |
| `integration/circula-master-consolidation-20261119` | `c3c991aa3c12` | [#104](https://github.com/djshellshoxxx/circula/pull/104) → main | Five Core export-verifier files. This exact source is ancestral to #90 and #109; preserve provenance and avoid duplicate implementation. |
| `master/fnet209-utf8-20261119` | `059b0e4508a6` | [#105](https://github.com/djshellshoxxx/circula/pull/105) → federation topic | Three paths add UTF-8 signing-input rejection, regression/fuzz source and a dedicated workflow. Federation stays disabled/HOLD. |
| `audit/federation-readme-20261119` | `d55554408ef5` | [#106](https://github.com/djshellshoxxx/circula/pull/106) → #105 topic | One README line acknowledges that dedicated workflow. All other blobs match #105; no runtime change. |
| `audit/gov021-final-core-refresh-20261119` | `b8cb63bfc6d2` | [#107](https://github.com/djshellshoxxx/circula/pull/107) → existing GOV-021 topic | Technical refresh of held recovery authority. Preserve separate policy decision and migration boundary. |
| `master/recover-analytics-20261119` | `29a0312c5c4d` | [#108](https://github.com/djshellshoxxx/circula/pull/108) → analytics topic | Thirty-six module/workflow paths recover archived analytics with source and qualification records. Disabled/HOLD and production acceptance gaps remain. |
| `feature/core-catalogue-continuation-20261119` | `afb0da286098` | [#109](https://github.com/djshellshoxxx/circula/pull/109) → #90 | Active bounded Core continuation described above. Initial publication is recorded separately from its next revision. |
| `fix/fnet207-ci-proven-20261119` | `aeb70db0255d` | No PR in this enumeration; child of #97 `73a911a9bec3` | Three-file isolated recurring CI repair. Two Core files exactly match main and Core992; the recurring test preserves its fixed digest and comparisons. No missing Core recovery; disabled/HOLD remains. |

The recurring follow-up appeared at the last enumeration and had zero
check-runs when read. Its name and commit message are not proof of provider
qualification. Direct blob comparison verifies that `cmd/circula/auditanchor.go`
and `internal/service/departure_pagination_regression_test.go` already match
main/Core992. The remaining change splits a non-secret golden digest into
literal parts without changing the expected value or removing assertions.

Federation's UTF-8 repair concerns `Sign` returning success after lossy JSON
encoding. The PR's fail-before evidence says `Verify` already rejected those
envelopes; this report does not promote that finding into a reproduced
signature-verification or replay bypass. Analytics' historical completion
percentage and tested-source metadata remain provenance, with the current
feature metadata explicitly retaining its integration and acceptance gaps.

## Nine moved branches and source inclusion

| Branch / PR | Prior → recorded head | Verified disposition |
|---|---|---|
| Core integration #90 | `f25d10b6b408` → `839e86935767` | Core992 composes #99, final #101, frozen #102, #104 and admin-reset evidence; the final child adds only audit documents. |
| Core consolidation #99 | `47446b33b01f` → `0a73ee42b962` | Includes earlier cursor/session/recovery streams and is ancestral to current #90 and initial #109. PR remains open against main. |
| API-036 #101 | `baedf4b6670f` → `f7ac6d22a1bd` | Adds the committed keyset-insert regression plus traceability. GitHub records merge into #90 at **11:39:56 UTC**. |
| Core recovery #102 | `c97233fb5b79` → `f29ebb01ff68` | Fifteen-path follow-up includes the real-client committed export-interruption test, export-authorization regression correction and evidence. Source is ancestral to #90/#109; PR remains open against main. |
| Isolated feed #92 | `0fed0480d3b1` → `db80ed34af09` | Composes #98 with Core992; all 33 #98 module files and its dedicated workflow match exactly. |
| Feed completion #98 | `949d32907f8c` → `a8aecfe8782d` | Nine paths update workflow/qualification/HOLD evidence. GitHub records merge into #92 at **11:57:10 UTC**; no merge into main. |
| Isolated notifications #100 | `be0c071ebb3d` → `98852afbd1cd` | Composes #103 with Core992; all 38 transport/module source files match #103, and its dedicated workflow matches prior #100. |
| Isolated PWA #95 | `f153bf5b18f9` → `0b0d70624591` | Core refresh preserves all 44 module files, dedicated workflow, and existing setup-node registration exceptions. |
| Held SEC-026 #94 | `26f350b8aa43` → `45b596f12876` | Core refresh retains independent, default-off signals and repairs its feature-specific fixtures. Policy hold remains. |

The large file counts in isolated refreshes include inherited Core history.
Direct tree comparison shows that #92 and #100 preserve **all 1,469
non-traceability paths of Core992**; their only changed existing Core path is
generated traceability. The remaining additions are
their isolated module/workflow and composition evidence. #95's only differing
existing Core paths are the two previously reviewed setup-node registration
files; both are unchanged from its preceding head. These are not unmerged
hundreds-file Core feature contributions.

The prior checkpoint's #99 review identified a live-token authority defect in
the older `f25d10b` parent and warned against merging that parent alone. Current
#90 and #109 contain the repair. Direct blob checks confirm that their token
authority, migration cleanup, SQLite transaction handling and composed session
store files equal #99's `0a73ee4` versions. The old review concern is preserved
as provenance rather than repeated as a missing repair in current #90.

### Policy-held branches remain separate

GOV-021 #107 and SEC-026 #94 both preserve the 21 Core migrations and all 525
Core992 catalogue values. They add different feature migrations at **0022**,
with 45 GOV or 24 signal messages respectively. Their topic-specific schema and
policy states do not form a combined adoption plan. GOV-021 still requires
decisions on permanent/temporary authority, independent authenticated approval,
waiting/objection and expiry. SEC-026 still requires subject visibility,
threshold/disable authority and retention decisions. Retain both holds and
their distinct ownership, even if technical checks later pass.

All inspected future-module `feature.json` files retain `enabled:false`,
`integration:HOLD` and no dependency on another future module. The JSON
retains their explicit remaining scope. No new branch in this snapshot changes
that adoption boundary.

## Exact-head checks and review evidence

Check collections were read at their recorded timestamps, from **12:22:08 UTC**
through the late target/recurring observations. All check counts fit within a
100-record page, and every returned check's `head_sha` matched its recorded
commit. These observations are independent of local or historical source
qualification reported in PR descriptions.

| Head / PR | Observed check result | Interpretation |
|---|---|---|
| Main `ead63b5` | 12 success, 1 failure, 9 queued | Historical fuzz failure remains recorded; nine newer checks are pending. |
| Earlier #90 Core992 | 9 cancelled | Historical snapshot retained; no passing qualification inferred. |
| Latest #90 `839e869` | 9 queued | New docs-only head requires its own provider results. |
| Initial #109 `afb0da2` | 7 queued, 1 skipped | Initial optional native job skipped; `ci:cross-platform` label is present at observation. |
| #99, #101, #102 | 9 queued each | Source inclusion and #101's merge state do not make these checks passes. |
| #104 | 7 queued, 2 in progress | Export source is included, but this source-head run is incomplete. |
| #92, #98, #100 | 9 queued, 1 skipped each | Isolated composed/module qualification remains incomplete. |
| #94, #106, #107 | 7 queued, 1 skipped each | No technical completion or policy approval inferred. |
| #95 | 8 queued, 1 skipped | Earlier source/browser passes remain separate. |
| #103 | 8 queued, 1 skipped, 1 success | One successful module check is not the complete set. |
| #105, #108 | 10 queued, 1 skipped each | Dedicated native/fuzz qualification remains incomplete. |
| Recurring follow-up `aeb70db` | Zero check-runs | No provider result observed. |

The evidence contains each job ID, name, state, timestamp and URL. Relevant
runs include [latest #90 run 44523175368](https://github.com/djshellshoxxx/circula/actions/runs/44523175368),
[initial #109 run 44522656889](https://github.com/djshellshoxxx/circula/actions/runs/44522656889),
and the existing [main fuzz failure](https://github.com/djshellshoxxx/circula/actions/runs/44296161788/job/134875883683).
The separate commit-status endpoint returned `pending` with zero status
contexts for main and both focus heads; check-runs provide the actual job
evidence above.

Review collections for the 16 investigated PRs contained two submitted
engineering reviews: #99's earlier review and #105's source review. Both are
`COMMENTED`; none of these collected records is `APPROVED`. PR body statements
about independent source review are attributed source claims and are not
converted into GitHub approvals or provider results.

## Consequence for Core consolidation

This delta reveals **no additional unowned Core runtime recovery** outside the
active #109 continuation. The independently useful #104 verifier work and
final #101/#102 regressions are already included in the Core ancestry. The
remaining unique runtime changes belong to isolated future modules or the two
explicit policy holds. The latest #90 addition contributes audit evidence only.

Preserve the eight new #90 documentation paths, finish the already assigned
Core continuation, and qualify the exact resulting candidate. All original
and subsequent branch histories remain available. This refresh makes no
remote mutation, source edit, build, deployment, release, module activation or
policy decision, and leaves the original inventories unchanged.
