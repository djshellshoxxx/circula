# UX-047 native Windows qualification continuation

Checkpoint: 2026-11-23 Pacific. Source/acceptance record; passing results follow in the maintained [integration handoff](https://github.com/djshellshoxxx/circula/pull/131#issuecomment-7109225318).

- Starting actual main: `d3c5ce67a252a9111e98729c7b967ecb9cc7cdc8`, tree `004ff13caaf1532a30d488432110f2499e443a68`.
- PR #131 product source/head: `037f10a7d8729ab7e3add4c06e724e30b8e8cd77` / `a2b4229031cb18ff3517ad429b86e54c9edc329a`. All seven applicable PR jobs passed; its named Linux results were 3,278 passes, six skips, zero failures, and browser results were 597 passes.
- Follow-up [#132](https://github.com/djshellshoxxx/circula/pull/132), branch `codex/ux047-windows-qualification-20261123`, CI source `089c3a196d436e8c3fb4c71d8949e57529b0382a`.

## Observed failure and bounded remedy

Actual-main Core run [45237650449](https://github.com/djshellshoxxx/circula/actions/runs/45237650449), Windows job 137913540541, ran the complete uncached non-race suite. At 18:33:31 UTC the service package reached Go's implicit ten-minute alarm: 600.066s. The reported running test, TestErasurePurgesReplayRecordsAndExports, was 0s old and in fixture cleanup: DB.Close, SQLite WAL checkpoint and Windows FlushFileBuffers. No assertion failure was reported. The store package passed at597.914s; web at188.730s and messages at0.219s also passed. Those observations support a cumulative runner budget limit; they do not prove that all I/O can always complete.

The only execution change is `go test -count=1 -timeout=20m ./...` on native Windows. The 30-minute job limit, every package/test, uncached count and existing non-race limitation remain. No product/system timeout, storage implementation, durability setting, authentication, ledger, policy, privacy boundary or future module is changed. This is qualification of the same UX-047 slice, not a new feature or permission to bypass a failing test.

Independent review inspected the original stack and approved this bounded candidate. The original failure is preserved; do not mark it resolved until full native Windows execution at the final corrected head passes. The cross-platform label opts the final PR head into both native jobs. Required gates retain the full Linux race suite, browser/axe/keyboard suite, vet/gofmt/lint/vulnerability and traceability checks, builds, reproducibility, fuzz and API/load smoke. Actual-main checks remain separate integration evidence.

## Traceability and continuation

No Go test/requirement symbols change. The existing UX-047 citations and generated inventory remain intact; CI freshness is required. No new unit test is added for a command-only runner budget; real complete native execution is the proof. Local Go is unavailable. No local native runtime or passing rerun is fabricated.

Finish #132, verify its exact current head and applicable native/default checks, merge normally only after qualification, then verify actual main and publish exact source/Actions/artifact evidence. Keep the original #131 and follow-up branches. If a test assertion or timeout persists, inspect it rather than weakening the workload. Only after this same slice is qualified should a fresh run investigate a different current-source UX-047 gap. UX-047 remains partial; translations, policy/manual/external gates, held PRs #83/#85/#86/#94/#107 and all fifteen future modules stay untouched. No deployment, release, licence choice or history rewriting.
