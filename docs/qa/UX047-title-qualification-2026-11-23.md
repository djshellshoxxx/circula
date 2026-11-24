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

## Actual-main qualification still open

At 2026-11-24 04:42 UTC (November 23 Pacific),
[Core 45321695163](https://github.com/djshellshoxxx/circula/actions/runs/45321695163) attempt 1
had seven successful jobs; Linux and macOS Intel race were still running.
Windows job 138201504445 passed `go test -count=1 -timeout=20m ./...`:
16 passing packages plus two no-test packages, without cached package results.
Browser job 138201504074 recorded 597 passed.
[Scorecard 45321695017](https://github.com/djshellshoxxx/circula/actions/runs/45321695017)
completed successfully; execution success does not resolve its policy/licence findings.
No actual-main full pass is claimed. The title integration must remain active until the remaining native
jobs, Linux artifact and final durable evidence are verified.

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

## Exact continuation

Fresh-check main and Core 45321695163 for the merged revision. Inspect all nine applicable jobs, original
Linux artifact and native logs; fix only a real failure, then publish final qualification and verify readback.
Do not repeat #137's patch, source CI or merge. After this title slice is qualified, stop this run at the boundary.
A fresh investigator may then verify the suspension-category inbox gap, after coordinating #136's
notification-path ownership. Preserve authored escaped reason, raw API/audit category values and legacy data.
#135/#136 retain their coordinator. Five held policy PRs, all fifteen future modules, licence selection,
release/deployment and history/branch deletion are out of scope.
