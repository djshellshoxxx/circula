# UX-047 platform-notice provider qualification — November 27, 2026 Pacific

## Status and scope

PR [#142](https://github.com/djshellshoxxx/circula/pull/142) already implements the requested bounded suspension-category inbox correction.
It remains **draft and unmerged**, retaining its documented separate owner merge authorization.
This run reviewed and qualified existing work; it did not duplicate the patch or change its source head.
UX-047 remains partial. #135/#136 are now recorded merged through the completed #138 integration.

Current main/base: `3faeed76c41467a75b09bc75e7185a98a92f8219`.
Source: `78e254f4f1b8283c124274e0a6a1ba2b82959a0f`, branch `fix/ux047-platform-notice-categories-20261126`.
Tested synthetic: `392b2022359995d95b0b73d8e25c268ee14e402f`, parents base and source.
Source and synthetic share tree `2cfb643bcd5ee5e9239be9716cd9abdf2d5fdf91`.
This matches composition, not a claim of actual-main integration.

## Verification

[Core 45836478829](https://github.com/djshellshoxxx/circula/actions/runs/45836478829) passed all seven applicable jobs on attempt 1:
Linux race, browser, lint/vet/gofmt/govulncheck, target builds, reproducible build, fuzz smoke, API/load smoke.
The conditional native PR matrix is skipped as configured.
Original Linux artifact `15131292538` matched provider size/digest: **139,018 bytes**,
SHA-256 `61e0792685b1accffd10d20cbb862a0761775ee26efc494724814d85079b55e2`.
Recorded clean tested synthetic head, exit zero, command
`go test -json -count=1 -race -timeout 30m ./...`.
**3,304 named passes, six skips, zero failures; 16 passing packages and two no-test packages.**
Both new integration parents and all 13 fallback subtests passed (15 events).
Browser job `139966576982` records **597 passed**.

Fresh current-main Core `45996511685` has all nine successful jobs, including native Windows and macOS.
Those baseline results do not certify a future #142 merged main.
Source record digest `4d9422d2b7be4f427a55d04e663c46e43320df11c7537ea8b7916036af33fb23`
was read from the recorder; it was not independently recomputed.

A fresh independent source review found no actionable blockers. Exact producer-envelope recognition,
ordinary string/template escaping, historical record-only category selection and row-local presentation
preserve raw stored/API values, operator-authored reason and existing member/network scoping.
Unknown/modified/erased/linked/event-dated records retain their original wording.
There is no service/storage/schema/authorization/suspension-policy change.

The source qualification note's regression-only/local Go results remain author-provided evidence.
This environment has GNU Go rather than the required compiler; no new local Go or browser execution is claimed.
Hosted checks supply the repository-wide results above. Engineering review is not external security acceptance.

## Durable evidence and remaining gates

[Evidence archive](evidence/UX047-platform-notices-provider-2026-11-27.tar.gz):
173,050 bytes, SHA-256 `ebc3a44473ddff2824358dd2f263a1cdb72fe2a114d3a5724c47ed9d9610c76e`,
Git blob `cfd1eb0c8f41cdbc51577876602960afecd0df3c`.
Eight payloads plus SHA256SUMS contain original Linux ZIP, provider run/jobs/artifact metadata,
tested commit, extracted summary, raw browser log and independent source review.
This audit branch adds only this report/archive over main and is not a product integration.

The source/draft explicitly requires separate owner authorization for main merge.
Leave it draft and unmerged until that gate is resolved; do not invent approval.
After authorization, recheck exact head/base/candidate and applicable checks, merge normally and verify actual-main CI.
No merge is performed by this qualification.

Six skips, bounded fuzz/load/build/reproducibility, Windows non-race, manual screen-reader, independent external
security review and translation/locale/product decisions remain separate limits.
The five policy PRs, SEC-026/GOV-021 holds, all fifteen future modules, licence/release/deployment and history gates stay untouched.

## Fresh-agent continuation

Do not repeat #130–#139 or the #142 patch/tests. Read live refs and the maintained #142 provider record.
If owner merge authorization is still absent, retain that integration gate and investigate one independent
unblocked current-source UX-047 requirement outside #142's owned inbox paths. Confirm a real gap before editing,
write its failing regression, keep scope modular, and independently verify exact-head CI.
Do not turn the beta matrix or one page slice into whole-feature completion.
