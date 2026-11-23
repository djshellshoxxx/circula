# UX-047 existing unit-suffix integration qualification

Checkpoint: 2026-11-23 Pacific. The bounded suffix migration is integrated; all configured applicable source and actual-main checks completed successfully.

This is one bounded English catalogue seam. It externalizes the existing unit
suffix without changing displayed English, amount semantics or unit selection.
It is not a newly demonstrated wrong-English defect, an irregular-noun policy,
a new locale, completed unit localization, whole-UI/Core/specification completion
or release qualification. Completed #130/#131/#132 were not repeated.

## Exact source and integration

| Object | Exact revision |
| --- | --- |
| Starting qualified main | `9cb4cc0250db487d6fd4760227ce906c4ea5f58f` |
| Preserved topic | `codex/ux047-unit-suffix-20261123` |
| Regression-only source | `3d3e7e059e2a694709f99d7414322870db69814b` |
| Final #133 source | `980609562395bea7234474ba4359acc1ca145f51` |
| Qualified synthetic PR checkout | `c456418a43314d8d89e9f0e52df6ae2172155028` |
| Normal-merge actual main | `a1e424c7a4d16b1592f1b5ac978a22ffc09c1d5f` |
| Shared source/synthetic/main tree | `14ad0769187147e6988045b0629ef2218d0a81ff` |

The normal merge has starting main and final source as its two parents. The
topic ref was verified preserved; this work rewrote or deleted no historical branch. The final report/evidence
branch adds exactly two files over qualified main; that evidence-only tree is
not represented as a separately executed product qualification. Final remote
ref/tree/commit/report/binary readback is recorded in the maintained #133
description. No force push, history rewrite, branch deletion, deployment,
release or licence selection is part of this work.

## Mechanism and acceptance

Only `return name + "s"` in `unitFor` becomes
`messages.Text("unit.plural", name)`. The new ordinary-text catalogue pattern
is `other: "%[1]ss"`; the catalogue now contains 1,460 entries. Exact fixed-point
one/negative-one selection and the case-insensitive existing-s rule remain in
the helper. There is no artificial CLDR count, amount-to-float conversion,
dependency, locale negotiation or irregular English noun decision. Arbitrary
names remain string arguments rather than format strings or trusted markup.

The helper is shared by financial rows/cards, trade and reversal review,
dispute/re-review displays, administrator proposal descriptions and the template
`unit` helper. Existing rendered dashboard, marketplace, trade, dispute and
administrator regressions remain part of the full suite. There is no ledger,
SQLite/durability, stored policy, audit, mutation, authentication, tenant or
balance behavior change.

The nine-file diff comprises two product files (`internal/web/format.go` and
`internal/web/messages/en.json`), two existing test files, the focused CI regex,
and four handoff/message/acceptance documents. The focused regex adds
`TestProposalMessageIsPlainText` while retaining the full race recorder and all
other gates. No new top-level requirement/test symbol is introduced; generated
traceability is not manually edited and its freshness check passed.

| Existing test | Bounded proof |
| --- | --- |
| `TestUnitForPlural` | Nineteen table cases: precisions 0/1/2/4, both signs of exact one, zero, fractional/large integer amounts, existing-s/uppercase names and Unicode/percent/dollar/HTML-sensitive names |
| `TestProposalMessageIsPlainText` | One added named subtest, `currency_unit_inflection_stays_catalogue_plain_text`, verifies the new key, literal string interpolation and contextual template escaping |
| `TestEveryCatalogueEntryFormats` | Supplies a string to the new entry and checks catalogue formatting diagnostics |

The nineteen table cases execute inside one parent test; they are not nineteen
new named events. The added unit subtest and the three affected parent-test
events passed in the source artifact. Repeated runs are separate qualification
evidence, not additional unique coverage.

## Executed proof

| Stage | Observed result |
| --- | --- |
| Native red contract | Run `45272197441`, completed failing Linux job `138034623179`: new message subtest panicked exactly `unknown UI message: unit.plural`; the focused web package passed at 1.703s. Production files were unchanged in that source. The full race recorder was skipped after the expected focused failure. The overall red workflow was later cancelled after that completed failure; it is not green evidence. |
| Exact-head PR | Run `45272948641`, attempt 1: seven applicable successes and the expected conditional native-matrix skip. Browser 597 passed. Linux: 3,279 named passes, six skips, zero failures; sixteen passing packages plus two no-test packages; the added unit subtest and all three affected parents passed. |
| Actual main | Run `45278925063`: all nine jobs succeeded on attempt 1. Linux race recorded 3,279 named passes, six skips and zero failures; the new unit subtest and three affected parents passed. Browser: 597 passed. Native Windows non-race and macOS Intel race each passed sixteen test-bearing packages plus two no-test packages, with no cached test results. Mac retained `-race -count=1 -p 2 -timeout=45m`; Windows retained `-count=1 -timeout=20m`. Mac service/store/web/messages durations: 1683.577/959.140/636.410/1.323s; Windows: 416.866/378.784/136.956/0.123s. No applicable CI finalizer remains for this bounded slice. |
| Actual-main Scorecard | Run `45278925544`: completed successfully on attempt 1 at exact actual main. The provider reports 5.9 across eighteen checks; findings remain open. Successful execution is not approval of licence, review, branch policy or other findings. |
| Local available checks | Pinned browser message script/harness: sixteen Node tests passed, zero failures/skips in a mock DOM. Catalogue duplicate-key/JSON and source whitespace checks passed. No local Go compiler or gofmt exists; native CI supplies compilation, formatting, vet/build/race and browser execution. |

PR Linux artifact `14867144686`, 139,282 bytes: SHA-256
`7f891b2c403dc4652c6b07cd05d7a7fed35df49e048866bebb9a8bae0be0eba9`,
matched to provider metadata. Original ZIP contains only
`traceability-results.json` and `TRACEABILITY-results.md`. It records clean
synthetic head `c456418a43314d8d89e9f0e52df6ae2172155028`, exit zero and source
digest `8ff2362a89abdbd6b4fc7ba61dfdda59a2e66d9b27b2a69cf2ab07f01c665fcb`.
Actual-main Linux artifact `14870930568`, 138,290 bytes: SHA-256
`040cf62b3a411450a19368a462abe6246c724fb42c1cf9f4ef259ed57093dde0`,
independently matched to the provider digest. The original ZIP records clean
exact actual main `a1e424c7a4d16b1592f1b5ac978a22ffc09c1d5f`, exit zero,
and the same source fingerprint as its tree-equivalent PR checkout. Both
original ZIPs independently record 3,279 named passes, six skips, zero failures,
sixteen package passes and two no-test packages. The added unit subtest and all
three affected parent tests passed in each distinct run.

The six explicit default-suite skips remain skips:
`TestBackupStagingRefusesForeignOwnedParent`, `TestBackupWorkLeaseChild`,
`TestRestoreSwapCrashChild`, `TestLoadConcurrentTrades`,
`TestLoginTimingStatistical` and `TestQA010CrashChild`. The two no-test packages
are separate package records, not those six named skips. Native non-verbose
package results do not create per-test event evidence.
The separate native export-reader workflow was inapplicable. Its actual-main
path filters are `scripts/read-export.py`, `scripts/test_read_export.py` and
`.github/workflows/export-reader.yml`; none of the nine modified paths match.
Its absence is not an outstanding applicable finalizer.

## Review, durable evidence and limits

Independent browser investigation found no confirmed current browser-message
UX-047 defect; the separately known clock-zone concern was not selected or
changed. Independent unit/source review approved the narrow suffix migration
before publication. Native red proof precedes implementation. Independent final source/acceptance and provider reviews approved exact source and actual-main qualification with zero blockers. The archived completed-main review supersedes its earlier pending checkpoint.

The [evidence archive](evidence/UX047-unit-suffix-integration-2026-11-23.tar.gz)
retains provider metadata/logs, both original Linux ZIPs, source/readback data,
reviews and local-check boundaries. Its manifest covers exact payload paths,
sizes and SHA-256. Archive SHA-256 `3dafdb42bca31173acde595d5563d4b01f0b96ed074f1b9bec84bee8c447ce96`, 552,145 bytes,
68 payloads plus manifest. A separately authored read-only validator passed exact revision/run/job checks, both original ZIP digests/counts/skips/regressions, full safe manifest inventory, every payload hash/size and byte equality with the frozen directory. No compiled executables or Python bytecode are included. Final remote publication readback is recorded in the maintained PR description.

These assistant-worker reviews are not the independent external security
assessment. No manual desktop/mobile screen-reader pass is fabricated. Six
named skips, bounded fuzz/load and packaging/reproducibility/sandbox-smoke
limits remain explicit. This presentation-only work does not close storage
read/fsync/truncate or uncertain-commit qualification. Scorecard execution
success does not approve its findings or licence/branch/review policy.
SEC-026 visibility/threshold/retention, GOV-021 authority/identity,
translation/locale scope and other owner/manual/external gates remain open.
Held PRs #83/#85/#86/#94/#107 and all fifteen future modules were untouched.

## Fresh continuation

After publication and readback, ownership of this bounded slice is released.
UX-047 remains the single active partial Core feature; this run stops at that
boundary and leaves the next investigation to a fresh run.
Read actual live main, newest Core PR heads/checks, HANDOFF.md, docs/TODO.md,
specifications, decisions, this final record and #133's maintained description.
Continue the single active partial feature UX-047 with a different bounded
current-source investigation and fresh independent workers. Do not repeat
#130 precision, #131 quantities, #132 native finalization or #133 suffix wiring;
revive historical missing-flow lists; choose held policy work; or inspect/run
future modules. Write a real failing regression before a confirmed correction,
preserve financial/privacy boundaries, run focused and required exact-head CI,
publish a precise handoff and stop at the bounded boundary. Do not deploy,
release, select a licence, invent owner approvals or use paid coding/API services.
