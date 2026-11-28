# Session-warning expiry: corrected-head qualification and owner-gate handoff

Reviewed November 27, 2026 Pacific. **Bounded source qualification complete; integration remains owner-gated.**
[PR #143](https://github.com/djshellshoxxx/circula/pull/143) remains draft and unmerged.
No separate owner merge authorization has been provided. This is not actual-main qualification.

## Exact revisions and acceptance

Current main/base: `3faeed76c41467a75b09bc75e7185a98a92f8219`.
Branch: `fix/ux047-session-expiry-text-20261127`.
Corrected source: `bfbfb79020942394457966c1e72b329e65bb06dd`.
Tested synthetic: `a9d480304ea61d12772036ea548d420c4844542c`,
parents base and corrected source; source and synthetic share tree
`947d1e2cf6cc3d7aa9e6f9de638fe626351f5416`.
Corrected source is a normal child of initial implementation
`5523f848aab15e1d7eb8090b36e56019f5c81a63`; regression-only ancestor
`f2095da2dbeb0ec509bb0fa378f45b9c4f45dd81` is preserved.

The server supplies catalogue DateTime expiry text for the initial warning and extension.
The extension explicitly loads the saved display zone. Numeric expiry, extendability,
timers, authentication, CSRF, unsent form data and the public API remain unchanged.
Plain strings retain template attribute escaping and literal textContent replacement.
Existing nil-viewer renders use labelled UTC; older cached pages/server responses without
the new presentation value retain the legacy clock fallback. This does not complete all
shared-shell selected-zone wiring, UX-024, UX-047 or locale scope.

Source paths: internal/web/render.go, h_account.go, templates/base.html, static/app.js;
regression paths: internal/web/session_warning_zone_test.go and scripts/test_app_messages.cjs.
HANDOFF.md, docs/TODO.md and docs/qa/UX047-session-warning-2026-11-27.md describe source scope/limits.
The independent source, Go-test and repair reviews are retained in the evidence archive.

## Corrected native proof

[Core 46099555002](https://github.com/djshellshoxxx/circula/actions/runs/46099555002)
passed **all seven applicable jobs on attempt 1** at the exact corrected source:
Linux race, browser, lint/vet/gofmt/govulncheck, builds, reproducible build, API/load and fuzz smoke.
Conditional native PR matrix skipped as configured; Windows/macOS and actual-main results are not inferred.

Original Linux artifact `15249262273`: **138,833 bytes**,
SHA-256 `d8de101ea4c3023bcd539da358ff7db14d65c0dd01735e45660e8bcc2c18f439`.
The downloaded original ZIP independently matches fresh provider size/digest.
Recorder: clean exact tested synthetic, exit 0,
`go test -json -count=1 -race -timeout 30m ./...`.
**3,299 named passes, six skips, zero failures; 16 passing packages and two no-test packages.**
All 12 new timezone parent/subtest events and the previously failing
TestProgressiveCatalogueTemplate passed (13 specifically checked events).
Recorded source digest `9486d82bd728ca8e2ee02e399a45c0040ada8bff48e357d55ee4bfc71233c0ce`
was observed, not independently recomputed.
Corrected browser job `140838660748`: **597 passed**.
Both reproducible-build hashes:
`16500e626bdd5a73284a1ec4d0a150daf2eec9afe0ee2559a1640bb78e13fb2f`.
An independent final provider reviewer verified the artifact, parents/tree, jobs and browser log,
with zero bounded provider blockers. Engineering review is not external security acceptance.

The six skips remain TestBackupStagingRefusesForeignOwnedParent, TestBackupWorkLeaseChild,
TestRestoreSwapCrashChild, TestLoadConcurrentTrades, TestLoginTimingStatistical and TestQA010CrashChild.
Bounded fuzz/load/repro/build checks are not exhaustive stress, security or packaging qualification.

## Retained red/green and failure history

Native Node on original app.js: 16 pass, two intended failures (initial and extended labels ignored).
Corrected Node: 18 message tests pass under UTC/Kolkata/Vancouver and combined helper suite 26 pass.
These are mock-DOM script runs, not browser journeys; the script/tests are unchanged by the repair.

Initial native Core 46090029432 failed gofmt and the pre-existing progressive-template guard.
Its original ZIP `15242492102`, 138,799 bytes,
SHA-256 `3dd951e07ecccb79e661aeb815cd65a792ac06123c47bdcb7b18ba5ecff17d25`,
records clean `060949cea2f5f3524e0e62fa38437204cb6e894e`:
3,298 named passes, six skips and one failure (the guard); all new timezone tests passed.
The repair only restored adjacency of data-expires/data-extendable by moving the new
data-expires-text attribute after them, plus one native gofmt alignment.
No assertion, workflow, timeout or guard was weakened or removed.

Official Go 1.26.11 SDK was checksum-verified against Go release metadata; native gofmt completed.
Two local focused race attempts were killed compiling SQLite dependencies before tests ran:
first libc with -p 2, then sqlite/lib with -p 1 and constrained GC/memory.
Neither is a pass. They remain preserved alongside the succeeding hosted native run.
No new local HTTP/browser/race success is claimed.

## Durable evidence and integration gate

[Evidence archive](evidence/UX047-session-warning-provider-2026-11-27.tar.gz):
**376,245 bytes**, SHA-256
`b3b5d0a56887bfa246712863841011548fd391d92d6bf35feda6948daec36269`,
Git blob `128c4f81784017d13259338df3cc2c8e7b53fe00`.
36 payloads plus SHA256SUMS; every payload was verified.
Includes original failed/corrected Linux ZIPs and metadata, raw provider logs, local Node/failure
receipts, SDK verification receipt and independent investigation/source/test/repair/provider reviews.
The audit branch adds only this report and archive over current main. It is not merged or deployed.

The source/draft retains separate owner merge authorization. Leave #143 draft and unmerged
until authorized, then fresh-check head/base/candidate/checks, merge normally and qualify all
applicable actual-main jobs including native Windows/macOS. Do not infer main results from tree equality.
#142 and five policy-held PRs remain untouched; all fifteen future modules remain out of scope.
Manual screen-reader, independent external security review, locale/product/owner decisions
and release/deployment/licence/history gates are not closed by this source qualification.

## Fresh-agent prompt

This is the written owner-gate handoff for the bounded #143 slice; stop this run at this boundary.
Fresh investigator: fetch live refs, current owner instructions/claims, HANDOFF/TODO/specs/decisions
and #143's maintained qualification. Do not repeat this patch, repairs, reviews, tests or publication.
If owner authorization is still absent, preserve the integration gate and continue one independent
unblocked current-source UX-047 requirement outside #142/#143 owned paths; prove the gap before editing.
Treat the explicit nil-viewer UTC/legacy-cache boundary as recorded partial coverage, not a hidden
whole-feature completion claim. Do not reactivate SEC-026/GOV-021, policy PRs or future modules.
If authorization arrives, finish current integration first, with exact-head and actual-main proof.
