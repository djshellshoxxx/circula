# UX-047 shared-shell title qualification — 2026-11-23 Pacific

## Current integration state

PR [#137](https://github.com/djshellshoxxx/circula/pull/137) merged normally as `418d2ed4b289c183d1fbd087b8f7b7ee6451169e`.
Qualified base: `73b98ec2c5a0ca2a94e79661aa7c005c15cdf52d` (Core 45294258651, all nine jobs).
Reviewed source: `73c5afb0eeebb54c3cf7645c0300fee12cd1d733`, preserved topic `fix/catalogue-page-titles`.
Synthetic candidate: `9fddaf265dc706c8d63de046f60c1021a55abeb6`.
Source, synthetic and merge share tree `ac2c359216d12c46965c84b45b971025daae5993`; merge parents are base and source.
Tree equality verifies composition; actual-main execution remains a separate gate.

## Acceptance and changed paths

The complete public and network title formats now reside in `common.title.public` and `common.title.network`.
Default English remains `Screen - Circula` and `Screen - Network - Circula`.
Positional arguments allow catalogue-owned order and separators. Names remain plain strings with contextual HTML title escaping.
No authentication, authorization, tenant, ledger, route or persistence behavior changes.

Product/source paths: `internal/web/templates/base.html`, `internal/web/messages/en.json`,
`internal/web/messages/messages_test.go`, `internal/web/shared_pages_catalog_test.go`, `docs/dev/MESSAGES.md`.
`TestDocumentTitleCatalogue` covers public/network English and reordered titles, with percent signs,
ampersands, angle brackets and closing-title text. Fresh template instances isolate each test override.
`TestEveryCatalogueEntryFormats` covers both key signatures; `TestPageStructure` remains applicable.
This is bounded UX-047 externalization, also preserving UX-044 titles and UX-046 English document language.

## Verified source CI

[Core 45312460743](https://github.com/djshellshoxxx/circula/actions/runs/45312460743), attempt 1,
passed all seven applicable jobs; conditional native-matrix placeholder skipped as configured.
Linux artifact `14886516616`: 138,614 bytes,
SHA-256 `86f4050e29d8f88734c5a506455bd39186d3ed8466e955c66c5845abfb10fefc`.
Provider size/digest independently matched the original downloaded ZIP.
Recorded clean head `9fddaf265dc706c8d63de046f60c1021a55abeb6`, exit 0,
`go test -json -count=1 -race -timeout 30m ./...`:
3,284 named passes, six named skips, zero failures; 16 passing packages plus two no-test packages.
Title parent and all four controls passed. Recorded source digest
`ebfbe9cf9635746bdae3fc68b8924d259d9a8e17750edabcbc1f2015ff27c8f0`
is an artifact observation, not an independently recomputed digest. Browser log: 597 passed.
Independent source, acceptance and provider/merge reviews found zero bounded blockers.
Exact-pinned Node helper suites separately passed 24 tests; these are mock-DOM/helper checks.

## Verified actual-main qualification

[Core 45321695163](https://github.com/djshellshoxxx/circula/actions/runs/45321695163)
passed all nine applicable jobs on attempt 1 at exact main `418d2ed4b289c183d1fbd087b8f7b7ee6451169e`.
Linux artifact `14890858220`: 138,532 bytes, SHA-256
`d56d0646d9e05c15625a85d893c08a97fc1eda054c0c93c97426d61e7d08f54f`,
matched original ZIP bytes to provider size/digest. Recorded clean actual-main head, exit 0,
uncached full race command: 3,284 named passes, six skips, zero failures, 16 passing packages and
two no-test packages; title parent and all four controls passed. Its recorded source digest matches
the source-run artifact; this observation does not replace either execution.
Browser job 138201504074: 597 passed. Windows 138201504445:
`go test -count=1 -timeout=20m ./...`, non-race, 16 passing packages plus two no-test packages.
macOS Intel 138201504638: `go test -race -count=1 -p 2 -timeout=45m ./...`,
16 passing packages plus two no-test packages. Neither native log reports cached package results.
Mac service/store/web/messages elapsed times: 1477.509/896.722/525.733/1.384 seconds.
[Scorecard 45321695017](https://github.com/djshellshoxxx/circula/actions/runs/45321695017)
succeeded, reporting 5.9 across 18 checks; findings remain open.
The export-reader native workflow is inapplicable to the five product/source changed paths.
No source repair, rerun, test removal or budget increase was required.
Independent final artifact/native review is preserved with the evidence.

## Provenance and limits

The author reports regression-only `ba5d566` passing both English controls and failing both reordered controls,
then all four passing with implementation; both complete local web packages and Chromium 153 title checks passed.
Those remain author-provided observations. Their broader local run passed 15 test-bearing packages before
service compilation and SQLite vet hit resource limits; a larger accessibility fixture crashed during setup.
These were not full local Go/vet/browser passes. This environment has GNU Go, not the required Go compiler;
no new local Go/browser execution is claimed.

Six explicit Linux skips: `TestBackupStagingRefusesForeignOwnedParent`, `TestBackupWorkLeaseChild`,
`TestRestoreSwapCrashChild`, `TestLoadConcurrentTrades`, `TestLoginTimingStatistical`, `TestQA010CrashChild`.
Fuzz/load, reproducible-build target and packaging checks are bounded. Manual screen-reader work,
independent security review, whole-UI externalization, translation/locale scope and owner policy decisions
remain separate gates. UX-047 remains partial.

## Evidence and fresh-agent handoff

[Evidence archive](evidence/UX047-title-evidence-2026-11-23.tar.gz): 775,907 bytes,
SHA-256 `d4fcab7986f68dd6549d99bff60b15e057791a84b93e6732a776fb050e041456`,
Git blob `69a4d30d1e1ac8999e193fd005fe5afe73982d52`.
75 payloads plus SHA256SUMS; every payload checked against its manifest.
Archive contains original PR/main Linux ZIPs, provider logs/metadata, pinned inputs,
text-composition snapshots and independent source/acceptance/native/provider/documentation reviews.
Historical pending observations in those original records are preserved, superseded by this final qualification.

Audit branch `audit/ux047-title-qualification-20261123` preserves its original pending checkpoint
`6cdd910a9bf8817ab1374f7f390ae16d6bfc776c` and appends this final documentation/evidence.
It is not merged into main. Main product tree remains the qualified #137 tree.
HANDOFF.md, docs/TODO.md and the UX-047 traceability row now supersede historical #133/pending instructions.
Verify published commit/tree/ref and every modified file before relying on this record.

Title slice accepted; foreground ownership for this slice is released. Stop this run at this boundary.
Fresh-agent prompt: Read live main, current PR heads/ownership, HANDOFF.md, TODO, decisions/specs and this
final qualification. Do not redo #130–#134 or #137. #135/#136 remain the other coordinator's work;
check actual state and coordinate notification-path ownership. Then investigate only the evidenced
platform-suspension category inbox gap: reuse existing labels while preserving authored reason
`security_incident & <policy>` escaped verbatim, raw API/audit category values and legacy notices.
Write a failing current-source service/HTTP regression before any repair. Keep UX-047 partial and
all five policy PRs/fifteen future modules untouched. Do not release/deploy, select a licence,
invent owner approval, force-push, rewrite history or delete branches.
