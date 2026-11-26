# Circula handoff

## 2026-11-26 Pacific: UX-047 platform-suspension inbox categories

The bounded continuation on `fix/ux047-platform-notice-categories-20261126`
starts from `main@3faeed76c41467a75b09bc75e7185a98a92f8219`. Platform-suspension
notices now use the existing readable category labels in the HTML inbox.
Only the exact known stored format is recognized; operator reasons, historical
records and API responses remain unchanged. Unknown and erased records keep
their stored wording. No migration or suspension-policy change is introduced.

The [qualification record](docs/qa/UX047-platform-notices-2026-11-26.md)
preserves the failing regression source, passing local checks and independent
review. The resulting PR's current-head checks remain the integration gate.
UX-047 is still partial. Main merges require separate owner authorization;
the five policy-held PRs, future modules and release/deployment gates remain
unchanged. The completed #130–#139 work below is historical context.

## 2026-11-24 Pacific: UX-047 notification/dashboard integration complete (#134–#138)

Every non-held Core branch is merged and main is qualified. Current main is
`f4243e317e01b7a95b6deff137c707f00b2e5761` (merge of
[#138](https://github.com/djshellshoxxx/circula/pull/138)). Do not repeat any
slice below.

| PR | Scope | Main merge | Actual-main qualification |
| --- | --- | --- | --- |
| #133 | Catalogue-owned currency-unit suffix | `a1e424c` | push run 45278925063 |
| #134 | Account/network timestamps in the member's time zone | `73b98ec` | push run 45294258651 |
| #137 | Catalogue browser titles for public/network pages | `418d2ed` | push run 45321695163, nightly 45405536221 |
| #135, #136 via #138 | Dashboard amount/meter messages; policy, closure and dispute notice dates in the recipient's current time zone (migration 0026, nullable `event_date`, cleared with the body on erasure) | `f4243e3` | PR run 45540044015 (7/7 exact-head); push run 45543080887 (all nine jobs, attempt 1, incl. Linux/macOS race and native Windows) |

#135 (`fa696eb`) is an ancestor of #136 (`efea7f9`); #138 composed #136 with
`418d2ed` at tree `f9607f06c28b99d7b2d63a4031f065c436dd0abf`. GitHub recorded
#135 and #136 as merged through #138. Their source branches are preserved.
Before merge, ancestry and tree were re-verified, the store/erasure/handler diff
was re-reviewed (only erasure path clears `event_date`; no other reader of the
table), and local Go 1.26.11 `go vet ./internal/...` plus non-race domain, web,
messages, api, store and service tests passed.

Only policy-held work stays open: #83 (AUD4-F02 owner decision), #85/#107
(GOV-021), #86/#94 (SEC-026). Their decisions, all fifteen future modules and
release/deployment/licence/history gates remain untouched. UX-047 remains
partial: a fresh investigator should pick one new bounded current-source gap.

## 2026-11-23 Pacific: UX-047 unit-suffix continuation (#133, historical)

The completed #130 precision and #131/#132 quantity/native qualification slices
are not repeated. Starting main is `9cb4cc0250db487d6fd4760227ce906c4ea5f58f`,
qualified by Core run 45259862932 (all nine jobs, attempt 1). Its final evidence
is on preserved commit `547d24b62a6b23ab2c8d9851e23d4a4e152182f8`.

The single active partial feature remains UX-047. [Draft #133](https://github.com/djshellshoxxx/circula/pull/133),
`codex/ux047-unit-suffix-20261123`, externalizes only the existing English
currency-unit suffix. Exact fixed-point singular selection, existing-s handling,
amounts, names and contextual escaping remain unchanged; no new morphology or
locale policy is selected. Native regression-only source
`3d3e7e059e2a694709f99d7414322870db69814b`, run 45272197441, job 138034623179,
failed the new catalogue contract with `unknown UI message: unit.plural`.
The full race step did not run after that focused failure.

The corrected source requires focused and full exact-head CI, independent diff
review and separate actual-main qualification before completion. Read the
[current acceptance/traceability record](docs/qa/UX047-unit-suffix-2026-11-23.md)
and #133's maintained PR description before acting. If #133 is incomplete,
finish this same slice; once merged and fully qualified, do not repeat it and
use a fresh investigator for another bounded UX-047 gap. UX-047 is not complete.
Held policy PRs, SEC-026/GOV-021 decisions and all fifteen future modules remain
untouched. No release/deployment/licence or history changes are authorized.

## 2026-11-23 Pacific: prior native qualification source checkpoint (#132, historical)

PR #131 is merged as `d3c5ce67a252a9111e98729c7b967ecb9cc7cdc8` after
all seven exact-head jobs passed (3,278 named Linux passes, six explicit skips,
597 browser passes). Actual-main run 45237650449 then exposed the Windows
package-wide default ten-minute timeout; this is not complete qualification.
The newly started service test was closing SQLite at the alarm; store passed
597.914s, and web/messages passed. Preserve that failure as evidence.

The one active partial feature remains UX-047. [PR #132](https://github.com/djshellshoxxx/circula/pull/132),
`codex/ux047-windows-qualification-20261123`, proposes only an explicit bounded
Windows package budget. Source `089c3a196d436e8c3fb4c71d8949e57529b0382a`
is based on the freshly verified merge above. Every test, `-count=1`, the
non-race Windows limitation and the 30-minute job bound are retained. Independent
evidence review approved the mechanism, conditional on an uncached full native
Windows pass and all applicable exact-head checks. Product code is unchanged.
See the [failure/acceptance record](docs/qa/UX047-native-windows-qualification-2026-11-23.md)
and maintained [integration handoff](https://github.com/djshellshoxxx/circula/pull/131#issuecomment-7109225318).

The maintained integration handoff records subsequent exact-head and actual-main
outcomes. If #132 or its integration qualification is incomplete, finish this
same slice. If it is merged and fully qualified, do not repeat its finalization;
let a fresh run investigate a different current-source UX-047 gap. Do not repeat #130/#131 product patches,
touch held policy PRs or inspect/run any future module. UX-047 remains partial.
Older dated checkpoints below are historical where superseded.

## 2026-11-23 Pacific: UX-047 rules-log quantity source checkpoint

Main was freshly verified at `fff81fc7f64cb4858c09d4a3a6f1ff0a55a1a337`.
PR #130 setup precision and its integration qualification are complete; do not
repeat that patch or finalizer. Its durable final record is on
[evidence commit 048b519](https://github.com/djshellshoxxx/circula/blob/048b51940f4533d746e48dc3acf69e548ee61f09/docs/qa/UX047-setup-precision-integration-2026-11-20.md).

The active partial Core feature remains UX-047. This bounded slice is [PR #131](https://github.com/djshellshoxxx/circula/pull/131),
`codex/ux047-rule-quantities-20261123`. Corrected product source is
`037f10a7d8729ab7e3add4c06e724e30b8e8cd77` over regression-only source
`15664ad3341428b24d1ccf8c9078b9f2f12fb9cc`. Its five-file presentation change
uses catalogue integer formatting for policy quantities while retaining exact
amounts and ungrouped identifiers. See the [current acceptance/traceability
and continuation record](docs/qa/UX047-rule-quantities-2026-11-23.md).
Independent source reviews approved; local browser-message tests passed 16/16.
Go execution is unavailable locally. Native regression-only run 45232559048
failed exactly the new raw-quantity and rendered-row assertions. The maintained
[PR qualification/handoff](https://github.com/djshellshoxxx/circula/pull/131#issuecomment-7109225318)
records subsequent corrected final-head CI, merge and actual-main results.
This checked-in source checkpoint makes no subsequent green-result claim.

UX-047 remains partial. SEC-026 and GOV-021 policy holds, held PRs
#83/#85/#86/#94/#107, translation scope, manual/external gates and all fifteen
future modules remain untouched. If #131 is open, finish its current-head gates;
if merged, verify its maintained final record and let a fresh run choose another
bounded UX-047 investigation.
The older dated sections below are historical where superseded.

## 2026-11-20 Pacific: prior Core continuation (historical)

Core consolidation and finalization are recorded at `main@87651d36558a8e421e41fbd76c514045d88c74ed`. The [post-merge continuation record](https://github.com/djshellshoxxx/circula/blob/8f936b6eb4eefbe85250ee5cf44a839ad361e4ce/docs/consolidation/CORE_POSTMERGE_CONTINUATION_20261120.md) preserves the exact main qualification and historical PR dispositions.

The current UX-047 slice corrects setup amount-validation wording when the selected precision is one decimal place. Its source, regression evidence and independent reviews are recorded in [the setup precision qualification note](docs/qa/UX047-setup-precision-2026-11-20.md). That note records local qualification of this bounded change; the resulting PR and actual-main checks qualify its integration. UX-047 remains partial. The dated checkpoints below describe their original state and are not current merge instructions.

## 2026-11-18 Pacific: comprehensive audit checkpoint (historical)

The audit beginning 2026-11-18 Pacific starts at `main@ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`. [PR #90](https://github.com/djshellshoxxx/circula/pull/90) holds the reviewed core consolidation, new ledger/authentication/dispute/policy/setup repairs, account-session work and onboarding catalogue continuation. Verify its current head and gates before merging; source-PR passes and local pure-package tests are not combined-main qualification.

Read [the comprehensive audit](docs/audits/COMPREHENSIVE_REPOSITORY_AUDIT.md), [execution report](docs/audits/CONSOLIDATION_FINAL_REPORT.md), [all 107 branch dispositions](docs/audits/BRANCH_CONSOLIDATION_MATRIX.md), [feature status](docs/audits/FEATURE_COMPLETION_STATUS.md), [security review](docs/audits/SECURITY_REVIEW.md), and [remaining blockers](docs/audits/REMAINING_BLOCKERS.md). The [15-module qualification](docs/audits/FUTURE_MODULE_QUALIFICATION.md) includes default and explicit SQLite suites and portable evidence. Future modules remain isolated and disabled/HOLD.

GOV-021 fixture repair [#91](https://github.com/djshellshoxxx/circula/pull/91) targets its original topic branch. Reviewer recovery, permanent operator appointment and signal subject-visibility decisions remain unresolved. All original branches are retained; no deployment is part of this work. The dated sections below remain historical evidence.

<!-- executed-consolidation:start -->
## 2026-11-18 Pacific: earlier consolidation checkpoint (historical)

This checkpoint predates `main@ead63b5` and the comprehensive audit integration. Its pending states describe that earlier time. PR #84 is included in `ead63b5`; later audit results are recorded under `docs/audits/`.

PRs #69, #70, #71, #72, #73, #74, #75, #76, #77, #79, #81 are merged. The [execution record](docs/audit/consolidation-2026-11-17/README.md) contains exact heads, merge commits, CI results, initial/final branch inventories, all historical PR dispositions and the security classification.

The PR #69 consolidation, PR #70 audit preservation and eight historical PR closures are complete; reviewed concurrent PRs #78, #80, #82, #83, #84, #85, #86, #87, #88 remain open under their recorded ownership and dispositions. Their unfinished work is not counted as integrated. The remaining open PRs are #67, #78, #80, #82, #83, #84, #85, #86, #87, #88.

- Recovered startup migration and administrator MFA/dispute reversal entry-point regressions, historical specifications and audit evidence. PR #70 is documentation preservation, not executable remediation.
- Fixed SQLite schema trust on all connections, private backup staging/exclusive copies with lease-aware orphan cleanup, and administrator notices beyond 500 members. Corrected the release-threshold validator, factual security documentation and the probabilistic privacy-erasure test. Corrected the browser freshness assertion across clock boundaries.
- Closed #1, #2, #13, #16, #20, #29, #35 and #42 after verifying preservation. The earlier ten superseded PRs stay closed; #67 stays draft and future-only. All branches remain preserved.
- The combined candidate has one successful full race run for all 14 test-bearing packages. It includes unmerged #82, #84, #87; their current-head provider gates remain incomplete at the recorded checkpoint (4 passed, 2 running and 15 queued required jobs). Their local pass is not a main-branch or provider-gate pass. Merged PRs have the required current-head CI and native qualification recorded individually.
- A3-F02 / DEC-014 remains unresolved. Retention, token-hash design, WAL sanitization, limit-notice and external release gates are not silently approved. The 15 future modules remain isolated. No deployment, tag, release, branch deletion or remote history rewrite occurred.

The sections below are dated historical records; their old pending-merge statements are superseded by the execution record above.
<!-- executed-consolidation:end -->

## 2026-11-18: PR and branch consolidation (historical preparation)

Branch `claude/circula-pr-consolidation-q6q18n` from `main@b53f757`. It gives every PR and remote branch a
verified disposition. Records:
[PR ledger](docs/plans/PR-CONSOLIDATION-2026-11-17.md) and [branch ledger](docs/plans/BRANCH-CONSOLIDATION-2026-11-17.md).

- 68 PRs and 88 remote branches checked by ancestry, patch IDs and content. Of the 19 open PRs, 10 hold no
  work missing from `main` and were closed as superseded; 8 (#1, #2, #13, #16, #20, #29, #35, #42) had their
  unique material recovered or archived in PR #69 and close when it merges; #67 (future module FNET-204) stays
  open. Branches are preserved; nothing was merged into `main`.
- Core recovery: `cb79676` and its review follow-up pin the OPS-025 start-up migration gate (from #35
  `ebb4e298`) at helper and `runServe` level, and correct docs that said `migrate` must run before a first start. `91a26f2` adds service entry-point regressions for the
  #51/#52 fixes (A1-04, A3-F03).
- Archives: the competing spec drafts of #1, #2 and #29 are under `docs/archive/specs-historical/` with an
  "IDs not citable" banner, and #38 is indexed there; canonical spec content is unchanged (the specs index gains
  a pointer). Two research notes moved to `docs/research/`.
  PR #42's audit notes are in `docs/audits/`; each finding's status on `main` is in
  `docs/security/security-review.md` (open: A3-F02, PRIV-01..05; E-02 and NEW-01 are fixed).
- `docs/future-features/README.md` lists the 15 isolated module branches (all `enabled:false`, HOLD).
- Clone pitfall: cloud clones can be shallow. Run `git fetch --unshallow` before any ancestry analysis.
- Next: owner reviews and merges the consolidation PR; FNET-204 engineering items in the PR ledger's #67 review;
  owner decisions listed at the end of the PR ledger.

## 2026-11-17: consolidation pickup (merged as #68, `b53f757`)

Branch `claude/compassionate-cannon-txdlz1` from `main@b7c27b4`. Folds three open items into one PR:
- PR #63 dashboard fix (`d53196c`): per-amount unit inflection and debt meter clamped to credit limit before scaling.
- Dependabot #48/#49/#50 (x/text 0.44.0, modernc sqlite 1.62.0 + libc 1.80.1, x/crypto 0.59.0), applied with `go get` + `go mod tidy` because the three bumps conflict in `go.mod`.
- Local `go build`, `go vet`, `gofmt -l` clean; `go test -count=1 ./...` passes in every package. Race, e2e, govulncheck left to CI.
Merging this PR supersedes #63, #48, #49, #50. No future-module work. No new features.

## 2026-11-16: core completion and consolidation

PR #61 merged as `7e97504c84209027972561b96c8b89084d9cc898` after all seven PR jobs passed at `cf5abf5c` in [CI run 43952719310](https://github.com/djshellshoxxx/circula/actions/runs/43952719310). The common-UI continuation on `codex/circula-common-ui-catalog-20261116` migrates shared shell/dashboard labels, main navigation, persistent network notices and unread/new counts to the catalogue. Dashboard amounts reuse the already-inflected card unit, fixing `Hourss` in visible values as well as speech. This is still partial UX-047; dynamic timestamp-rich notices, remaining templates/handlers and JavaScript are not fully migrated. Candidate validation belongs to its own PR checks.

PR #60 merged as `8afe90dcb422a25ce021cacb8a11cfd0a160d7ed` after all seven PR jobs passed at `96059bb4b8ae23ad6e4e45428bf69288bec4c182` in [CI run 43942206927](https://github.com/djshellshoxxx/circula/actions/runs/43942206927). The financial-message continuation on `codex/circula-financial-messages-20261116` migrates balance labels/spoken text, credit-limit usage, transaction statuses/explanations and complete per-kind row sentences into the English catalogue. It fixes the zero-balance spoken suffix for already-plural unit names and preserves exact amount strings and ungrouped ledger identifiers. New regression tests cover these contracts, but were not executed locally: the execution environment became unavailable. CI must validate this candidate, including formatting, test and traceability freshness, before merge. No local or browser pass is claimed for it.

PR #59 merged as `672cddb` after all seven PR checks passed at `12175ec`; source: [CI run 43930580504](https://github.com/djshellshoxxx/circula/actions/runs/43930580504). The catalogue continuation on `codex/circula-message-catalogs-20261116` adds English complete-sentence messages with CLDR plural/count formatting for the administrator overview and proposal rows. Focused catalogue/web race regressions pass. This is a partial UX-047 migration, with no new language or owner policy decision; see `docs/dev/MESSAGES.md`. Full candidate validation is recorded by its own PR checks.

Continuation also adds PER-022 network-page snapshot timestamps and persistent offline/restored-history notices. Server rendering and focused race checks pass; current browser evidence is supplied by PR checks because local Chromium cannot open its required socket.

Continuation: FR-067 now links aged proposals to a bounded administrator queue with current role checks, exact age/expiry predicates, keyset paging and neutral party descriptions. A real Linux checkpoint database-write failure regression preserves the committed WAL and proves restart, replay, checkpoint retry and reopen. Detailed remaining storage boundaries are in the storage qualification note.

Merged into main by PR #56 (`304600c`) and qualification follow-up #57 (`7cdb966`).
Fifteen included historical PR heads are accounted for: fourteen superseded PRs were closed and #55
was automatically marked merged; branches remain preserved.

Branch `codex/circula-core-completion-20261116` continues `claude/happy-babbage-dcumwz@dc97f7d`,
which already joined the beta follow-up (#33) with main. Documentation upkeep (#55) is merged,
with the newer export, logging and QA evidence preserved. The 15 `future/*` modules remain
isolated; this consolidation does not enable them.

Implemented and regression-tested in this pass:
- Export expiry actively releases disk space. Admitted downloads pin resources until completion;
  erasure aborts downloads and invalidates every affected network export, including member archives.
  Running exports cannot commit snapshots taken before an erasure; shutdown cleans up spools.
- Cached API retries recheck current scopes, membership, roles and MFA without repeating commands.
  Network erasure invalidates cached responses that could name counterparties.
- Migration 0017 preserves insertion order for immutable trade/dispute histories across equal or
  backward timestamps and VACUUM. Guards also reject REPLACE; invalid schema versions fail safely.
- Nanosecond backup names support rapid successive snapshots and retain old filename compatibility.
  Future-dated backups cannot suppress the scheduled cadence.
- The standalone export reader validates reconciliation for CSV/JSON and all exported scopes.
- Logout failures are reported without falsely clearing a still-valid session. Unicode login
  throttles clear correctly. Erasure removes counterpart notification content without broad text matches.
- Closure final-export failures are retried before erasure; completion/audit events are not duplicated.
- QA-019 now covers service, JSON API and HTML forms with a controlled clock; browser coverage is separate.
- Listing descriptions are labelled optional. The race target enables CGO. CI checks traceability
  freshness and runs the standalone reference-reader tests.

Additional audit completion:
- Frozen networks no longer starve proposal/adjustment expiry in other networks; batches stay bounded.
- Policy logs and challenges have signed, bounded cursor pagination and query validation.
- Re-review and pending adjustment approval require signed review/back/confirm flows, with current
  service checks and reviewer balance privacy retained.
- Administrator reads/navigation use effective roles and current MFA, including expiring break-glass grants.
- Network-scoped tokens cannot manage global credentials/preferences. JSON field spellings are exact;
  decoder failures no longer reveal Go types. Public static responses do not set CSRF cookies.
- OpenAPI covers common runtime statuses; temporary authentication storage failures preserve retryable 503 responses.
- Credit-limit ETags represent override/expiry configuration and are compared again inside the write transaction.
- Restore invalidates copied sessions, API tokens, MFA login challenges and break-glass sessions on the
  verified staging copy before swapping; failed invalidation leaves the original database untouched.
- Operator dual-control edits preserve pending policy notice timing and unrelated future values.
  Break-glass login lookup uses the same normalization as sign-in.
- Export byte order remains backwards-compatible with pre-migration closure checksums. Only history
  presentation uses migration 0017's insertion sequence.

Final historical-branch follow-up: restored the tested >500 reversal-history pagination from
`claude/cool-ramanujan-n2peag`; fixed RESTORE-01 with durable interruption markers and seven
subprocess crash tests; added versioned HMAC coverage for backup verification summaries (CRYPTO-02),
keeping legacy manifests readable. Temporary-admin adjustment proposals remain approvable while
the grant is live. Invitation-only join pages hide network existence from nonmembers.

Further completion: optional adjustment links (migration 0018) now flow through pending requests,
approval, audit, history, API and exports while preserving legacy export bytes. Repository contracts
are driver-neutral with old-store aliases. The MVP equivalence suite covers 43 paired HTML/API
operations and six explicit API-only lifecycle/token operations. Policy revisions are atomic;
throttle settings are bounded; scheduled verification is read-only; operator provisioning and displayed
reversal balances are correctly audited. UUID and query validation, forced-password navigation,
recovery-throttle erasure, reversal independence/audit history, timestamps and migration guards have
current regressions. Real SQLite disk-full tests verify rollback, retry and reopen. Traceability now
separates source citations from actual dated test evidence and uploads CI evidence artifacts.
The property harness now isolates each seed, records complete failures and provides a bounded manual
long-run workflow. The 100,000-sequence non-race run passed with 2,800,700 generated operations,
no failed seeds and no gaps/overlaps; see [the execution evidence](docs/qa/QA001-2026-11-16.md).
Release thresholds are still owner decisions.

Validation: PR #56 passed seven CI jobs. PR #57 passed all nine jobs at `dc89212`, including
native Windows, Linux/macOS race, all 579 browser/keyboard/axe tests, static/security checks,
contract/load, fuzz, target builds and reproducibility/deployment smoke. Windows document and
closed-file test assumptions were corrected, and every affected fixture now closes its resources.
The new Linux WAL regression forces an actual OS write failure during COMMIT, validates the
preserved partial WAL and verifies rollback, retry, replay and reopen. Focused normal and race checks
also passed; read/fsync/checkpoint and uncertain-commit qualification remain open.
No manual screen-reader pass, independent security review, tagged release or deployment is claimed.

See [docs/TODO.md](docs/TODO.md) for current work. All dated sections below are historical records.
They can contain superseded next steps, failures and merge instructions.

## 2026-11-16: beta follow-up

Branch `claude/quirky-einstein-l9ni8m`, draft PR #33 stacked on `claude/circula-beta-completion-1npjw6` (PR #31).
It works through the "Next" list of the 2026-11-11 section below. Every preserved WIP slice is now integrated.

Merged and reviewed (each slice reviewed, findings fixed with regression tests):
- **CI (O4):** one run per commit; macOS 15/Windows tests on `main`, nightly and dispatch only. See [docs/ops/ci.md](docs/ops/ci.md).
- **Go 1.26.11** (GO-2026-6859, ported from PR #30) and the QA-037 marker on migration 0016's temporary-table drop.
- **Export (BR-06, BR-04):** disk spool with reserved bytes and running-export slots, `export-spool-dir`, manifest
  `reconciliation` documented; erasure drops the network's admin-scope exports. QA-030 built.
- **web-ux:** UX-021, UX-049, SEC-016, GOV-010 built. UX-017 partial (adjustments cannot link to an original).
- **arch-guards:** DAT-045 built; ARC-002/ARC-004 adapter and store-contract guards; QA-012 matrix extended.
- **sec-ops-docs:** SEC-020, OPS-030 built (redacting log, request ids aligned); SEC-014, SEC-030, OPS-015 partial.
- **qa-robustness:** QA-007, QA-010, QA-024, QA-026, QA-028 built; QA-001, QA-019 partial. Bug fixed: expired
  proposals now notify both parties.
- **BR-05:** equal Argon2id work for an unusable stored hash; same-network visibility tests (A4).
- **Container (O2):** image and compose run for the first time: [docs/ops/container-verification-2026-11-12.md](docs/ops/container-verification-2026-11-12.md).
- **Traceability (QA-042, partial):** generated by `make traceability`; `make traceability-check` is in `make check`.

Verified at `927d2ca` (later commits change only docs), Go 1.26.11:
`gofmt`, `go vet`, golangci-lint v2.15.0 (0 issues), `check-hygiene.sh`, `check-action-pins.sh`, `go test -count=1 ./...`
and `go test -race -timeout 40m ./...` all pass in every package; `make e2e` 578 passed, 0 failed. CI passed on the PR.
govulncheck ran only in CI.

Next:
1. Owner: merge PR #31, then this PR.
2. Owner decisions raised in this pass: an original link for adjustments (UX-017, schema change); the timing threshold
   (DEC-023, provisional 10 %); QA-001 sequence count in CI; erasure of `person_prefs.time_zone`,
   `person_activity.last_seen_at`, `challenges.reason`; BR-04 scope (all admin scopes, not only `full`).
3. Engineering: backup names have one-second resolution (second backup in the same second is refused); status history
   orders by `occurred_at` (wrong if the clock steps back); 68 built requirements still have no test citation
   (`docs/TRACEABILITY.md`); ARC-002 dual-route test, ARC-004 driver split, ARC-008 operator audit; `read-export.py`
   does not check `reconciliation`; SEC-014 "(optional)" labels on listing forms; run `traceability-check` in CI.
4. Branch protection: drop `test (macos-15-intel)` and `test (windows-2022)` from required checks if they are set.
5. Owner gates below are unchanged: licence, DEC-*, SEC-033, manual screen-reader run.

## 2026-11-11: beta consolidation

## 2026-11-16: repo upkeep

- `docs/wiki` source pages refreshed in the repo. The GitHub wiki is NOT synced yet: the wiki git repo does not accept clones (likely no first page saved in the web UI). Owner action needed, then sync.
- `docs/specs` status refresh in progress (matrix is the reference).
- [docs/TODO.md](docs/TODO.md) is the single open-work list. The "Next" and "Paused" lists in older sections below are history.
- PR triage done. The bot closed no PRs; the owner decides which to close. Checked against the GitHub REST API on 2026-11-16:
  - 0 open issues.
  - 37 open PRs: 5 Dependabot (#46 to #50), #3 (not a draft), and 31 draft chained PRs.
  - PR #31 (head `claude/circula-beta-completion-1npjw6`, base `main`) merged 2026-11-16T01:12:49Z (merge commit `bffcede`). The 2026-11-11 section describes that branch as the draft PR into `main`. No open PR has that head now.
  - PR #19 (named as a draft in the 2026-11-04 section) merged to `main` on 2026-11-16 (`34ccb81`).
- Repo health checks run on 2026-11-16 with go1.26.11 (`go.mod`, `deploy/Dockerfile` and the CI workflows agree on 1.26.11):
  - `gofmt -l .`, `go build ./...`, `go vet ./...`, `scripts/check-hygiene.sh`, `scripts/check-action-pins.sh`: clean.
  - `go test -count=1 ./...`: fails in `internal/service`. `TestDepartureCancelsMoreThanFiveHundredProposals` panics with `rate-limited: too many open proposals` (`testutil.Must`). All other packages pass. Commit `2eacf77` is the last commit that touched it. Owner or engineering to triage.
  - Not run: race detector, e2e, load test, govulncheck (needs vuln.go.dev), release gate.
- Docs corrected in this pass: the release archive contents (`README.md`), the PR #19 status (`docs/ops/README.md`), the container build status (`docs/ops/hardening.md`), the lockout description (fixed 15 minutes, not doubling; `docs/dev/recovery.md`, `docs/security/RAINBOW-BOOKS-SPEC.md`), the release-gate notes (`docs/qa/EVIDENCE.md`, `docs/qa/RELEASE-GATE.md`), a stale unit path (`docs/security/GOV-STANDARDS-MAPPING.md`), and the QA-042 pending note in `docs/TRACEABILITY.md`.
- Not changed: owner decision rows in `docs/DECISIONS.md`. DEC-014 still says "Break-glass not built", which no longer matches `cmd/circula/breakglass.go`. Owner to update the status.


Branch `claude/circula-beta-completion-1npjw6`, a draft PR into `main`. It is the integration branch: it starts from
`claude/gracious-ptolemy-dscc7v@3c2a577` (PR #30) and merges PRs #21–#28, part of #13 and four preserved WIP
slices. Read these first:

- [docs/plans/branch-consolidation-report.md](docs/plans/branch-consolidation-report.md): every branch and the decision on it.
- [docs/plans/circula-completion-matrix.md](docs/plans/circula-completion-matrix.md): feature states with evidence, and the dependency graph.
- [docs/security/security-review.md](docs/security/security-review.md): findings BR-01..BR-06.
- [docs/releases/beta-readiness-report.md](docs/releases/beta-readiness-report.md): exact SHA, commands and results, blockers.

Done: C-09 policy ETag fix; DOM-017/DAT-051 dispute deadline fixed at completion (migration 0016); beta ledger and
session tests; release workflow gates; e2e WCAG/contrast/flow coverage (578 tests); lint at 0 issues.

Next, in order:
1. Owner: review and merge this PR into `main`, then close the PRs it supersedes (see the consolidation report).
2. Export slice: fix review findings BR-06 (bound in-flight bytes, configurable spool directory, document the
   manifest field), then integrate.
3. Rebase web-ux, arch-guards and sec-ops-docs onto this branch, review them, integrate.
4. CI: cut duplicate runs; move the macOS/Windows matrix to main or nightly (measure first).
5. BR-04: drop admin full-export artefacts on member erasure. BR-05: timing check for enumeration.
6. Owner gates below; independent security review (SEC-033); manual screen-reader run.

---

## Earlier handoff (2026-11-04, kept for history)

Updated 2026-11-04. Owned branch: `claude/happy-noether-444cum`, draft PR
https://github.com/djshellshoxxx/circula/pull/19 (base `main`, which holds prompts only).
Started from the product tip `codex/circula-noscript-qa@ba57512`, which contains the #17 → #18
chain and the responsive/no-script QA branches. Earlier handoffs are preserved in Git history
and under `docs/audit/`.

## Done in this pass
- `94728b1` DEF-001: the invitation return link (`next`) was double-escaped, so the return path
  was lost after Create an account / Sign in. This was the only failing e2e case at `b231dc5`.
  Fixed in the templates. Regression test: `TestReturnLinksDecodeToTheOriginalPath`.
- `c34d947`: remediates audit 2026-10-28 Medium findings A1-01, A1-02/A1-11, A1-03, A3-F01 and
  A3-F16. Each PoC is kept as a regression test, and those tests fail with the fix reverse-applied.
  A3-F02 is confirmed but waits on owner decision DEC-014. Report:
  [docs/audit/2026-10-28/REMEDIATION-2026-11-04.md](docs/audit/2026-10-28/REMEDIATION-2026-11-04.md).

- `ce7dcf0`, `d98734c`: merged reviewed slices `wf/auth-sessions` and `wf/gov-policy`. They add
  SEC-027/OPS-010 `circula revoke-sessions` (with `-rotate-web-key`), SEC-003 role session
  lifetimes, SEC-006 configurable web/API limits and `network-join-limit`, SEC-032 grant cap plus
  audited open-mode confirmation, GOV-011 listed removal reasons and conflicted-admin refusal, and
  DOM-031 stored scale-change detection. Defaults keep the earlier behaviour.
- `a2d20b1`: fixes the four blocking findings from an independent review of those merges, each
  with a regression test that fails without the fix.

## Verified locally (Go 1.26.10, this container)
- At `ba57512`: `go build ./...`, `go vet ./...` and `go test -count=1 ./...` pass in all 9 packages.
- At `94728b1`: the full Playwright suite passes, 325 of 325.
- At `c34d947`: `go vet ./...` passes; `go test` passes for service, store, cmd/circula, web, api and domain.

## Open owner questions raised this pass
- GOV-011: should `transaction_id` be mandatory for fraud or breach removals? The conflict check
  only sees the trade named in the request plus undecided disputes and re-reviews.
- DOM-031: should a scale change stored by an earlier build be neutralised once it is in force?
  It is currently detected, audited and logged as a warning only.
- SEC-032: should open self-joins be excluded from the grant even after confirmation? Currently
  they receive it, per DOM-027/DEC-051. Also: how should a lowered cap affect stored grants?
- SEC-006: per-login throttle values, numeric bounds for the rate settings (OPS-008) and a
  per-network login limit.
- DEC-005, DEC-006 and DEC-011 values. DECISIONS.md rows DEC-051, DEC-064 and DEC-065 are stale
  against the built mechanisms; they are owner records and were not edited.

## Paused 2026-11-11 (owner instruction)
- Work stopped at the owner's request. No agents or workflows are running.
- Unmerged, unreviewed slice work is preserved as patch series in
  [docs/audit/wip-slices-2026-11-11/](docs/audit/wip-slices-2026-11-11/README.md), with base, head,
  IDs and state for each.
- Next steps to resume:
  1. Review each implemented slice, then fix and merge it: web-ux, e2e-axe, e2e-flows, arch-guards, sec-ops-docs.
  2. Finish boundary-fuzz, export and release-chain.
  3. Implement qa-robustness.
  4. Cut duplicate CI runs and move the macOS/Windows test matrix to main only, measured as in
     the scratch notes: macOS race 7.9 min, Ubuntu race 6.8 min, e2e 4.7 min.
  5. Regenerate docs/TRACEABILITY.md (QA-042).
- Last CI result: `76e2c89` passed every job (push and PR runs). CI for later heads had not
  finished when work was paused.

## Owner gates (not decided here)
- Licence; DEC-005, DEC-011, DEC-013, DEC-014, DEC-021, DEC-022, DEC-023 and DEC-025; UX-047 locale scope.
- Release thresholds; pilot jurisdictions; the manual screen-reader run (UX-056, QA-035).
- Independent security review (SEC-033); A3-F02 break-glass scope.
- Merging to `main`, releases and deployments.
