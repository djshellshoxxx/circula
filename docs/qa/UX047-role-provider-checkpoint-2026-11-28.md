# UX-047 role-list provider/source checkpoint — 2026-11-28 Pacific

This is a partial qualification record. PR #144 remains open; current-head CI and actual-main qualification are pending.

## Fresh source and ownership

Live main/base is `3faeed76c41467a75b09bc75e7185a98a92f8219`. Another worker advanced the topic from original source `8c3ea2d91783dfcd4e951a92ab867fa0b3a9292a` to corrected source `9f7d98d4917bb7243ea7cefc79033e34053d8e3a` and marked it ready for review. We did not change its product files, draft state, checks or merge state.

Corrected source and synthetic candidate `7c6c3c874d64787aaa6affeefcfa214af57ed6cf` share tree `896a76b30e4081741245b01a0bfa02b99d900db1`; candidate parents are the current base and corrected source. Tree equality is not test execution.

Read current PR body, changed-file composition, HANDOFF/TODO, UX specification, decision log and exact acceptance note before acting. Prior topic work was preserved; no stale checkout was used. Held policies, #142/#143, future modules and their owner gates remain untouched.

## Original head: independently verified hosted evidence

[Core 46127440071](https://github.com/djshellshoxxx/circula/actions/runs/46127440071), attempt 1, passed seven applicable jobs; the conditional native PR placeholder skipped as configured.

Original Linux artifact `15256338574`: 138,679 bytes, SHA256
`6ed0e9dd80848688c5f2157b3a9ca102188fc81e2fed4c9a2a982d8be5ac54e3`.
Downloaded original ZIP matched provider size/digest. Its record shows:
- Clean tested synthetic `d260c4590d63025cfb04971903ebbb542da2d5df`, exit zero.
- `go test -json -count=1 -race -timeout 30m ./...`.
- 3,292 named passes, six explicit skips, zero failures; 16 passing packages plus two no-test packages.
- All five new role-template parent/subtest events and existing real-web `TestMemberCatalogueRowsAndActions` passed.

The original Linux job log confirms that checkout, command, artifact ID and digest. Original browser job log records **597 passed**. This is hosted execution at the ORIGINAL tree, not local execution or current-head acceptance.

GitHub run payloads contain a nested pull-request head that can update after a run. Qualification was bound to the run's own original head and artifact's tested synthetic, not that mutable nested field.

## Corrected delta: independent source review

Another worker found narrow-screen action targets below 24 CSS pixels and added a scoped member-actions class, logical minimum sizes and a real-page browser regression. Our independent reviewer fetched the exact corrected files/fixtures and found no actionable source blocker. Seeded Dana/Admin, Carol/Reviewer and Alice/ordinary-member expectations match fixture creation. The test uses the authenticated member list at 320px, requires action links to exist and checks all their measured bounds. The axe helper covers both colour schemes. Existing wrapping, escaping, routing and authorization remain unchanged.

The new browser test checks single-role rows and ordinary fallback; it does not itself check two simultaneous roles. The earlier template test and author-reported runtime exercise are distinct coverage. Reviewer ran no browser, Go, HTTP or stress test.

The acceptance note's 125 repeated tests, 240 authenticated reads/eight clients, grant/removal round trip and original 320px axe failure are **author observations**, not independently rerun here. They are not substituted for corrected-head CI.

## Pending gates and continuation

[Corrected Core 46131171576](https://github.com/djshellshoxxx/circula/actions/runs/46131171576), attempt 1, was queued on source `9f7d98d4917bb7243ea7cefc79033e34053d8e3a`: all seven applicable jobs queued, conditional placeholder skipped. No actual failure was observed, no code edit/rerun warranted, and no passing result inferred.

Next run must finish THIS role-list/action-target slice using fresh refs and native results, not reimplement either worker's patch. Verify corrected original artifacts/logs and all applicable jobs, preserve current base composition, then any authorized normal merge and all applicable actual-main native checks. Coordinate active foreground ownership; do not race another worker's merge. Do not revert ready-for-review status from a stale draft handoff.

UX-047 remains partial. Preserve six skips, Windows non-race/platform boundaries, bounded fuzz/load/repro/packaging, manual screen-reader, external-review and locale/owner limits. #142/#143 separate owner merge authorizations and shared-catalogue coordination remain absent unless later owner messages change them. No held-policy/future-module/licence/deploy/release/history/branch-deletion action is authorized here.

## Durable evidence

Archive: `evidence/UX047-role-provider-checkpoint-2026-11-28.tar.gz`, 173415 bytes,
SHA256 `8d06ced84a6ebbd3893ec7df464a0cdbee28e46337141c611c7afeca378cab86`.
Seven payloads plus digest manifest preserve the original ZIP, independent parsed summary, original jobs, corrected queued checkpoint, Linux/browser logs and independent corrected-delta review. This archive does not contain a corrected-head passing artifact.
