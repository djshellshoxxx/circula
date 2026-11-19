# Core verification and recovery follow-up

## Authority and immutable source boundaries

The owned working branch is `integration/core-verification-followup-20261119`.
Its source precursor `4069ecc` contains the complete UI candidate plus the
export-reader reconciliation, COMMIT acknowledgment regression, refreshed
branch dispositions, narrow secret-scanner repair, interrupted-test diagnostic
repair and keyboard-confirmation helper repair described below.
The published UI candidate is PR [#112](https://github.com/djshellshoxxx/circula/pull/112),
head `08d4b0078e8c308ae7f353849783913f4302882a`. Its tree
`9ae8efd0306da7de8984e239254aa6b01996f02e` exactly matches local frozen
`e0d27cc7cb73299219439919b260991b824d594a`.

The standalone `core-qualification` checkout stays at that frozen local head.
Its full race command has completed with the failure detailed below. A result below belongs to its stated source,
not automatically to a later integration commit. Normal Go VCS stamping is
retained. PR #112 targets PR #109's owned continuation branch; PR #109 at
`5db3efc084e6a807bb4702c09d3235490d42b71d` targets the active broader Core
integration #90 at `839e86935767288e594914ad711486616a604eca`.

Main was reobserved at `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625` on
2026-11-19 at 13:45–13:46 UTC (05:45–05:46 America/Los_Angeles).
No main merge, release or deployment has occurred in this continuation.
All 15 future modules remain isolated. SEC-026, GOV-021, reviewer-reset and
other recorded product-policy holds remain in place.

## Recovered changes in this follow-up

| Change | Exact contribution and evidence |
|---|---|
| Export-reader parity | Source `12cb5c5` / root `7654d25` applies PR #104's manifest consistency and strict JSON constant checks additively. The existing compressed-stream EOF guard and all three compression regressions remain byte-identical. All 11 non-script peer paths, including the native Python workflow and ten evidence files, are retained exactly. |
| Export-reader qualification | Source `3768ad5` / root `7db1fe7`: the combined parity tests fail before with 96 assertion failures and no test errors; after, all 41 tests pass in 18.657 seconds. The independent workflow review found no privilege, secret or publication expansion. Native hosted matrix results remain separate. |
| Lost COMMIT acknowledgment | Source `08003c9` / root `e89471f` adds a test-only driver wrapper that forwards a real successful COMMIT and then reports a nonretryable acknowledgment error. It verifies durable balanced state, connection recovery, idempotent retry and reopen, with a pre-COMMIT failure control. No production fault hook is added. |
| COMMIT qualification | Source `0ff8f68` / root `1054d9b`: four top-level tests and eight subtests pass with race instrumentation and default vet; package time 14.008 seconds. This qualifies the application/driver boundary, not device, fsync or physical power-loss behavior. |
| Independent reviews | Source `43c515b` / root `789f082` records bounded COMMIT-recovery and administrator-history source reviews. These are AI-assisted reviews, not GitHub APPROVED reviews or SEC-033 independent external approval. |
| Secret-scanner CI repair | Source `dd3357d` / root `9284596` adds six exact rule/path/value conjunctions for 18 independently verified nonsecret matches. No matcher, entropy threshold, default rule or scan path is disabled. All eight real-scanner controls pass, including fresh replacement values, extended values, relocated paths, adjacent generic/GitHub-shaped controls and a separate rule. |
| Keyboard confirmation fixture | Source `e0da8bb` / `bf4261e`, root `957e4b2` / `f561bfa`: the helper now requires a native checkbox when a field requests Space confirmation, while preserving Tab/Space/Enter navigation and ordinary link/text targeting. Three actual-helper controls fail before; all thirteen combined Node message/helper checks pass after. The provider artifact establishes the exposed duplicate-name link/checkbox order. |
| Actual HTTP confirmation controls | Source `1cc3e70`, root `28ed3e7`: invalid CSRF returns 403 without changing the session cookie, CSRF cookie or authenticated form token; missing confirmation returns 422 with the same-name link before the checkbox. Reload does not remove the collision. Confirmation using the original token returns 303, revokes the target (API 401) and preserves the current session (API 200). This is HTTP/HTML evidence, not a browser after-pass. |
| Interrupted-test diagnostics | Root `93f5ac3`: a real child timeout fails before because the recorder drops panic/stack output. The fixed full traceability package passes with race/default vet. Independent source review `f180c62` reports no findings. Terminal test results and provenance remain unchanged. |
| Branch and PR recovery | The additive 130-branch/112-PR refresh and six exact PR #111 audit files are in `d9ce332`. The alternate PR #111 WAL witness does not replace the candidate's existing pinned read-only connection fixture. Original discovery evidence stays preserved. The subsequent 137-branch/119-PR refresh is preserved in root `20fc647`, including the late PR #119 move and its precise rejected/superseded/recovered dispositions. |

See [export parity receipts](evidence/export-reader-reconciliation-20261119/README.md),
[COMMIT qualification](evidence/commit-acknowledgment-loss.md),
[scanner provenance and controls](evidence/gitleaks-evidence-exceptions-20261119.md),
and [the latest 137-branch refresh](BRANCH_REFRESH_POST119_20261119.md).

The complete UI source, 1,431 English catalogue entries and the singular
precision correction remain documented in
[CORE_UI_COMPLETION_20261119.md](CORE_UI_COMPLETION_20261119.md).
At `e523021`, production application Go files, web assets, `go.mod` and `go.sum`
remain byte-identical to frozen `e0d27cc`. Additional Go changes are tests and
the traceability diagnostic tool. The browser change is confined to its test helper.
The Python export reader and CI configuration are genuine additional changes
and retain their separate qualification.

## Observed qualification of frozen e0d27cc

Raw receipts and logs are retained in
[the verification evidence index](evidence/core-verification-20261119/index.json).
The complete normal terminal-event record is compressed losslessly; the index
records original and stored byte lengths and SHA-256 checksums. The recorder's
source digest is `9345085d1e2cb966e89c6334ffbceb690769d5caf061b9d105b921a8102287dc`.

| Gate | Observed result and limit |
|---|---|
| Full normal Go suite | PASS: 14 test packages, 3,080 passing test/subtest terminal events, six skipped tests; exit 0 in 277.194 seconds. HEAD, tree and clean status remained unchanged. `go test -json -count=1 -p 1 -timeout 20m ./...`, Go 1.26.11, CGO enabled. |
| Full race suite | FAIL: exit 1 after 3,241.220 seconds. Thirteen packages pass, with 2,701 passing test/subtest events and six skips. The service package fails at 1,200.255 seconds, consistent with its 20-minute limit; no individual failing-test terminal event is retained. Source remains unchanged. The recorder discarded detailed output from the unfinished test; a separate real-timeout regression reproduces and repairs that diagnostic loss. This is not a full race pass. |
| Concurrent ledger smoke | PASS: 4 workers, 20 members, 25 trades per worker; 100 completed trades / 200 committed writes, zero unavailable, retry, exhausted or unexpected errors; ledger verification clean and balances sum to zero. This is a correctness smoke run, not a release throughput claim. |
| Production binary | PASS, 2.211 seconds; ordinary VCS stamping retained. |
| Binary vulnerability scan | PASS: pinned govulncheck v1.1.4 reports no vulnerabilities, 10.540 seconds. |
| Package-level source vulnerability scan | PASS: `govulncheck -scan=package ./...`, no vulnerabilities, 13.675 seconds. This is explicitly package-level coverage. |
| Default source vulnerability scan | Did not complete: the subprocess was killed with signal 9 after 35.968 seconds, before writing diagnostics. The precise kill cause was not observed. No default symbol/call-graph pass is claimed. The unchanged CI gate still runs the default source scan. |
| Small repository gates | PASS: action pins/permissions, hygiene including four migration-checker tests, dependency licence list, gofmt, generated traceability freshness, seven Node message tests, 25 frozen-reader tests, four property-runner tests and eight threshold-checker tests. The later reconciled reader has its own 41-test receipt. |
| Static analysis | Pinned golangci-lint v2.15.0 first reports two test-source findings on frozen e0. Minimal test-only repair `e77559a` (root `ff5ee75`) preserves the deferred Rows.Next fault and catalogue assertions. Eight top-level tests / twelve subtests pass; full pinned lint then passes with zero issues and all twelve configured linters active in 31.334 seconds. Configuration and diagnostics are not weakened. |
| Fuzzing | PASS on exact `789f082`: amount parsing 1,000,000 executions; strict JSON, signed cursors, session positions and session selectors 100,000 executions each. No fuzz assertion/corpus failure. The first strict-JSON attempt failed during modernc/libc instrumentation compilation before fuzzing; its original receipt remains separate. All four remaining targets completed after recovery with one warm cache and greater available memory. |
| Browser suite | Blocked before the first page: the project-configured Chromium fails its process-singleton socket with `Operation not permitted`. The fresh run exited 1 in 2.507 seconds. No browser, keyboard, responsive or accessibility pass is claimed. |
| Supported browser alternatives | The desktop entry point returned an unavailable local MCP connection. The pinned Playwright headless-shell installer received unusable ZIP responses and exited 1. These attempts did not change browser security settings. |

The six normal-suite skips are `TestBackupStagingRefusesForeignOwnedParent`,
`TestBackupWorkLeaseChild`, `TestRestoreSwapCrashChild`,
`TestLoadConcurrentTrades`, `TestLoginTimingStatistical`, and
`TestQA010CrashChild`. Three are child-only harness entry points, one is
platform/ownership dependent, and two are explicit opt-ins. The load opt-in
was subsequently exercised as recorded above. The timing opt-in remains open.

The first continuation's failed race compilation remains recorded separately:
its modernc compiler was killed, although eight smaller packages and 467
test/subtest events passed. A later isolated race build of that dependency
passed. Neither outcome is relabelled as a full race-suite pass.

## Hosted checks and merge eligibility

At 13:45–13:46 UTC, PR #90's quality job had failed on 15 preserved audit
hash/prose matches covered by the expanded 18-match scanner repair. Its Windows
job was in progress and seven other jobs remained queued. PR #109 and #111
each had nine queued checks with no assigned runner and no executed steps.
PR #112 had seven queued checks and one skipped matrix job: its cross-platform
label was applied after its initial event. The next actual source update must
trigger the labelled matrix. These are pending or failed gates, not approvals.

Hosted jobs for #90 started after roughly 76–81 minutes of observed waiting.
REST showed runner-allocation delay; it did not expose an account-level cause.
Pending-deployment endpoints were empty. Required checks and reviews must be
observed on each actual proposed head before a normal protected merge.

## Remaining work and precise next actions

1. Resolve the service-package race failure with retained raw diagnostics and
   adequate isolated execution resources. The frozen failed run is preserved;
   no assertions, vet or race instrumentation may be removed to obtain a pass.
2. Preserve the completed workflow-sized fuzz and pinned static-analysis
   receipts. Retry the default source-vulnerability scan only with the recovered
   execution headroom, retaining its original killed attempt and actual source.
3. Preserve the completed 137-branch/119-PR refresh. PR #119's reload hypothesis
   is contradicted by provider/HTTP evidence; the actual helper collision has a
   reviewed repair. Additional acceptance-gap investigation starts on a separate
   branch while this candidate remains frozen for qualification.
4. Regenerate traceability after the combined source is final, qualify changed
   surfaces, commit the complete checkpoint and publish the exact local tree
   on the owned integration path. Preserve active peer refs and immutable
   earlier test sources.
5. Observe full required hosted checks and reviews; merge eligible changes
   normally through the integration stack, then verify resulting main and
   recalculate branch dispositions. No main qualification is inferred from
   the local candidate results.

Manual screen-reader work, SEC-033 independent external review, disclosure
contacts, owner-policy choices, release/reference-hardware thresholds and
physical storage-fault qualification remain outside the evidence above.
English externalization does not authorize new languages or establish complete
locale/date/number support. No release or deployment is authorized here.

The reconciled upstream patch includes four preserved space-only unified-diff
context lines, and the raw before-test log retains ten progress-line trailing
spaces. These byte-preserving evidence exceptions are documented in its
receipt; they are not a blanket `git diff --check` pass for the whole recovery.

## Executor recovery and diagnostic repair — 2026-11-19 14:01 UTC

The executor connection dropped after the full frozen race result had been saved.
It recovered with the repository and durable logs intact; temporary caches and
binaries were cleared. The same frozen production binary was rebuilt successfully
in 64.409 seconds with ordinary VCS stamping. One shared runtime cache is used
for recovery. The pinned linter was restored from its official prebuilt release
with both archive and checksum-file provenance verified, avoiding a repeat cold
tool installation.

The recorder timeout-output defect is fixed and its full package passes under
race/default vet; see [the real before/after regression](evidence/traceability-timeout-20261119/README.md).
This later tooling change is additional source beyond the frozen UI test tree.
The exact frozen race results are indexed alongside the normal results.

The session fixture repair is documented in
[its qualification note](../qa/session-keyboard-confirmation-qualification.md)
and [independent review](evidence/session-keyboard-independent-review-20261119.md).
PR #99's captured provider run passed 587 browser tests and failed the session
confirmation case. Those provider results belong to that older PR head; they
do not qualify this candidate or establish a browser after-pass for the repair.

The lint repair is confined to test code: a successfully opened query now
assigns its deferred-fault row wrapper before returning, and the catalogue's
zero/one/other count assertions use a tagged switch. It changes neither
production storage semantics nor the test's injected fault behavior.

## Additional acceptance review boundary

The current batch closes the confirmed catalogue, export, COMMIT-acknowledgment,
scanner, keyboard-fixture and diagnostic-tooling defects described above.
The historical built-requirement evidence inventory also lists potential gaps
in production UUIDv7/generator checks, help definitions, deferred route exclusion,
text-only dispute attachment refusal, packaged-binary mode parity, browser
pointer/motion acceptance and deeper storage fault injection. These require
comparison with current tests before new work is declared necessary. A separate
bounded review will classify already-covered cases, meaningful missing automated
regressions, and actual policy/environment barriers without expanding Core scope.

The five fuzz budgets total 1,400,000 executions across their separately recorded
commands. The first true amount pass is retained; it was not rerun to obscure
the later compile failure. The four completed retry commands use unchanged
three-minute per-package fuzz limits, `-parallel=1`, `-p=1`, two Go processors,
a 750MiB Go memory limit and the single restored cache. This materially differs
from the earlier contended one-processor/250MiB environment. All tested source
heads and package fingerprints remain in the original agent receipts.

Pinned static-analysis provenance and before/after results are retained in
[the lint qualification record](evidence/pinned-lint-20261119/README.md).
A direct Git comparison confirms the combined candidate's application and
traceability Go inputs match the lint-passing repair source `e77559a`; later
combined differences are documentation and browser-test helpers.

The complete workflow-sized fuzz qualification is retained in
[the first-attempt record](evidence/workflow-fuzz-789f082-20261119/README.md)
and [the completed retry](evidence/workflow-fuzz-789f082-20261119/attempt-2/README.md).
The original first-pass amount log and all four later pass logs remain intact.

## Publication and next frozen qualification

The reviewed follow-up will advance the owned PR #112 branch through a normal
fast-forward commit whose Git tree must equal the local frozen candidate.
The active #90 and #109 heads remain untouched during publication. Their
existing checks do not qualify the new combined head. The current cross-platform
label must produce the intended matrix on the new source event.

After publication, the combined source is frozen separately for full normal and
race testing, with raw JSON output retained to diagnose any failure. The earlier
e0 failure and passing source-bounded checks remain as historical evidence.
Newly identified acceptance regressions are developed on isolated later branches,
not inside a checkout or head undergoing qualification. No protected merge is
eligible merely because this checkpoint is committed or published.
