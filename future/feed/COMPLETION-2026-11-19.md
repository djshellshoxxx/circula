# FNET-204 isolated completion candidate — 2026-11-19

## Status and scope

**Implemented and independently reviewed. Disabled and integration HOLD.**

This record covers the user's isolated feed pagination/security task. It does not claim production Core integration, deployment, release readiness, browser accessibility, or completion of the broader preferences/UI roadmap. The final provider gate is the current-head status on PR #98; a local pass or a previous-head provider pass does not satisfy that gate. This record preserves the observed local evidence.

Starting commit: `8638f3d30f27f4882c4c6424d4c8b248208b6f3d` (PR #92).
Development branch: `feature/fnet-204-completion-parallel`.
Observed main at discovery: `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`.

## Isolation and branch provenance

| Reviewed source | SHA | Disposition |
|---|---|---|
| `future/fnet-204-feed-20261116` | `5d3819c3b3af031b75eceb5251da0226a4ecee77` | Original projection/spec source; already preserved in starting ancestry |
| `chatgpt/feed-signed-cursors-20261116`, PR #67 | `7a528c611bffdf92ef9f8cbddb98e8b088eb5036` | Signed-cursor fixes/tests already preserved in starting ancestry |
| `claude/fnet-204-signed-cursors`, PR #80 | `0952c94a5dbab4044d7673ce9d51cec9063ab97f` | Expiry, rotation, context binding and tests already preserved |
| `audit/future-feed-cursors-20261119`, PR #92 at start | `8638f3d30f27f4882c4c6424d4c8b248208b6f3d` | Dedicated worktree/branch base; retains all original histories |
| Same PR #92, refreshed during this task | `c4732bc95f2f1a65e642c356b04ed916b1dfa4b4`, then `0fed0480d3b129351c96cc0f62bd19479acceab7` | Core-only refreshes; `future/feed/` and its workflow byte-identical to start. Reviewed without merging the new Core/session work |

All remote feed histories were compared through their file history and ancestry. No missing original feed fix or test was found outside the starting candidate. New work from separate cursor, performance and QA worktrees was cherry-picked into the dedicated feature branch. Agent commits remain in local provenance. Initial Git-data publication `949d32907f8cc2fab30ce237d95bb361023e84ba` has tree `d2420ff49be172476024eb4e6df1c1b818394620`, byte-identical to reviewed local candidate `3c43a78c4a69444884cadfbb149f3ac5f23eabf4`. Follow-up evidence/documentation changes do not change runtime Go source. PR: https://github.com/djshellshoxxx/circula/pull/98.

The task changes only `future/feed/` and its existing dedicated workflow. No API-036, Core authentication/session code, Core migrations, unrelated future modules, root dependencies, shared Core CI, or production registration is changed. PR #90 and all original remote branch heads are untouched by this task. The later Core-only PR #92 refresh is a moving integration dependency, not a new merge parent here.

## Implemented behavior

- Indexed SQLite reference store with database-level limits and separate combined-kind/single-kind indexes.
- Immutable total ordering and insertion sequence; source identity uniqueness; durable deletion tombstones; conflicting timestamp/resurrection rejection.
- `PageFromStore`: at most 101 candidates per query, 1,000 candidates per request, 100 returned references, insertion fence excluding new/backdated inserts, live reauthorization, and finite partial/empty continuation pages.
- v3 authenticated encryption of private scan frontier, insertion fence and original issue time. Fixed-size padding prevents field-length disclosure. Category/network/viewer/preference/key bindings and cross-protocol separation remain enforced.
- Original v2 cursor compatibility with a golden fixture, stricter authenticated JSON parsing, typed-nil provider checks, and defensive key copies.
- Original traversal expiry persists across pages and rotation; malformed/oversized/expired/foreign-context tokens fail uniformly before querying the store.
- Runtime standard-library-only module; separate SQLite integration test dependencies. No automatic connection, schema initialization, or Core activation.

## Confirmed defects and hardening

| Finding | Repair and evidence |
|---|---|
| Existing full-snapshot helper cannot provide database-bounded 100k-history pagination | Added indexed `SQLStore` plus bounded `PageFromStore`; keep the 10k snapshot helper as documented compatibility API |
| Bounded filtering needs a potentially private continuation frontier | Added encrypted v3 state instead of placing unreturned IDs/counters in readable signed JSON |
| Variable v3 ciphertext length reveals private field lengths | Reproduced before padding; fixed with constant 1,024-byte plaintext and 1,052-byte encrypted body; confidentiality length/padding regressions |
| Authenticated JSON accepts duplicate/unknown/case-aliased fields and lossy Unicode normalization | Reproduced nine ambiguity cases; canonical representation check; all previously emitted v2 tokens remain compatible |
| Typed-nil custom key provider can panic | Reproduced and rejected at codec boundary |
| Returned key slices can mutate the supposedly immutable ring | Reproduced and repaired with defensive copies; concurrent key-copy tests |
| Cancellation can be ignored when a store returns an empty success | Root and independent audit identified it; regression failed on original implementation, then passed after checks immediately following both storage calls |
| Conflicting source snapshots and nil/zero codec behavior | Existing PR #92 fixes retained, with expanded property/adversarial coverage |

The parser/key-provider fixes are defensive API hardening. Review did not establish a remotely exploitable HMAC forgery or cross-viewer authorization bypass in the starting cursor implementation.

## Observed local verification

Toolchain: verified Go 1.26.11 linux/amd64, matching the repository pin. Synthetic fixtures only.

| Check | Observed result |
|---|---|
| Starting module `go test -count=1 -cover ./...` | PASS, 0.099 s, 98.8% statement coverage |
| New cursor hardening regressions on unchanged source | FAIL as intended: typed-nil panic, key aliasing, nine parser ambiguities |
| New persistent API tests before implementation | FAIL to build because the new API/types were absent |
| Cancellation regression before repair | FAIL: cancelled empty anchor returned success |
| Combined module `go test -count=1 -cover ./...` | PASS, 0.349 s, 91.3% statement coverage |
| Combined module `go test -race -count=1 -cover ./...` | PASS, 11.507 s, 91.3% statement coverage |
| Combined module `go vet ./...` | PASS |
| Independent QA snapshot properties | PASS: 300 randomized oracle trials, 100 insertion/revocation trials, 16 concurrent viewers × 40 traversals |
| Independent QA stored traversal | PASS: 3,005-row denied-history progression, malformed adapters, source changes, lifetime, 12 concurrent viewers; race repeat 2.614 s |
| Pagination property fuzz | PASS: exactly 10,000 executions, 20.134 s |
| Padded v3 cursor branch race/vet | PASS, 4.306 s, 95.3% cursor-branch coverage; this percentage is not combined-module coverage |
| Padded v3 cursor fuzz | PASS: exactly 100,000 executions, 18.766 s, two workers, two-minute hard timeout |
| Legacy v2 cursor fuzz on combined candidate | PASS: exactly 100,000 executions, 12.125 s, two workers, three-minute hard timeout |
| Repository hygiene, action pins/permissions, workflow YAML | PASS |
| Root generated traceability check | PASS before final publication; unchanged Core matrix, module-local mapping in `TRACEABILITY.md` |
| Real SQLite ordinary full suite on published runtime | PASS, 11.223 s; 10k/100k raw and signed traversals, sparse/backdated changes, restart, concurrency and indexed plans |
| Real SQLite full race suite on published runtime | PASS, 407.175 s; the complete suite, including both 100k traversals, query plans, concurrency and heap/RSS checks |
| Integration `go mod verify` and `go vet` | PASS |
| Real SQLite benchmarks on published runtime | PASS, 100 measured iterations/case; 17.391 s including excluded fixture setup; measurements below |
| Independent published-candidate verification | PASS: cancellation race regression 1.062 s; focused cursor/stored-pagination race repeat 1.747 s; tree equivalence, scope, formatting and dependencies independently checked |
| Exact-current-head provider CI | PENDING: GitHub jobs queued without assigned runners at the final local qualification checkpoint. No provider pass is claimed; inspect current PR #98 checks |

The new code adds database/error paths, so combined-module coverage is not directly comparable to the much smaller starting module. Integration-test coverage and race results are distinct from a statement-coverage percentage.

The final SQLite qualification hash-checked all five runtime Go files, both module manifests, the integration lockfile and both integration test files against published candidate `949d32907f8cc2fab30ce237d95bb361023e84ba`. Its timestamped log is `evidence/sqlite-qualification-2026-11-19.txt`. Compilation and execution were separated to bound compiler memory: ordinary and race binaries were built with `go test -p=1 -c` and `go test -p=1 -race -c`, then executed once with `-test.count=1`. This is the full Go test suite with the same race instrumentation, not a reduced substitute. Execution used `GOMAXPROCS=2`, `GOGC=100`, and `GOMEMLIMIT=400MiB`; compile used `GOGC=10`. Earlier results from an intermediate cursor build were replaced by this exact-source run. CI runs the documented direct `go test` commands; its isolated race timeout is 12 minutes and job limit 25 minutes, with all fixtures and assertions retained.

## Environment limitations and recovery

The first compile of the pinned SQLite driver was killed by the environment. The execution service then became unavailable, interrupting multiple pending runs. Results without completion were discarded. The workspace later recovered with source/worktrees intact. SQLite qualification was restarted with serial compilation, lower garbage-collection threshold and a memory limit. Shell Git read access works, but push lacks credentials; the authorized GitHub connector can publish a tree-verified draft without changing original branches.

## Independent review and integration dependencies

Independent source audit covered the bounded reader, SQL query shape, immutable/tombstoned identities, lookahead, insertion fence, v2 compatibility, padded v3 cryptography, error contracts and isolation. It found no blocking security/pagination issue beyond the cancellation repair above. The fresh audit independently approved published candidate `949d32907f8cc2fab30ce237d95bb361023e84ba` for isolated development/review after checking its tree, scope, cancellation repair and focused race evidence. Its approval is for source correctness/security/isolation. The subsequent exact-source full SQLite race gate passed; exact-current-head provider checks remain pending. This is an independent agent review, not an external human security audit.

Remaining integration work is documented in `INTEGRATION-PROPOSAL.md`: qualify the final moving Core candidate; implement an authoritative transaction-scoped source/permission adapter; choose schema/retention/restore behavior; persist keys and define rotation/issuance limits and TTL; specify preference policy; add separately reviewed routes, rate limits, CLI and no-JS/browser UI. Empty continuation counts can reveal coarse activity volume; cursor encryption does not provide traffic-analysis privacy.

## Performance measurements

Go 1.26.11 linux/amd64; AMD EPYC 7763 host; `GOMAXPROCS=2`; benchmark `GOGC=100`, `GOMEMLIMIT=400MiB`; 100 iterations per case; fixture construction excluded from ns/op. Page size is 100 returned references and 101 SQL candidates. Values are single-run measurements on a shared host, not a latency service-level objective. Equal-time cases seek into large timestamp ties. Full-page measurements include the final padded v3 cursor implementation and synthetic authorization callbacks; production Core authorization cost remains unmeasured.

| API | Events | Timestamp distribution | Latency per page | Go heap B/op | Allocations/op |
|---|---:|---|---:|---:|---:|
| Full `PageFromStore` | 10,000 | Varied | 0.495930 ms | 96,644 | 1,638 |
| Full `PageFromStore` | 10,000 | Equal timestamps | 0.631994 ms | 96,287 | 1,638 |
| Full `PageFromStore` | 100,000 | Varied | 0.666873 ms | 96,650 | 1,638 |
| Full `PageFromStore` | 100,000 | Equal timestamps | 0.753996 ms | 96,231 | 1,638 |
| SQL `QueryBatch` | 10,000 | Varied | 0.762203 ms | 44,440 | 1,357 |
| SQL `QueryBatch` | 10,000 | Equal timestamps | 0.329148 ms | 44,040 | 1,357 |
| SQL `QueryBatch` | 100,000 | Varied | 0.509034 ms | 44,441 | 1,357 |
| SQL `QueryBatch` | 100,000 | Equal timestamps | 0.364024 ms | 44,041 | 1,357 |

Raw output: `evidence/benchmarks-2026-11-19.txt`. Allocation counts remain constant across history sizes; output buffers and candidate processing have explicit hard limits. The 100k equal-timestamp query-plan checks report indexed tuple `SEARCH`, with neither a full table scan nor a temporary sort. The separate non-race heap test measured 43,704 allocated Go bytes and 1,350 allocations per 101-row query at both history sizes. Live Go heap after the repeated reads was 1,042,408 bytes (10k) and 855,120 bytes (100k). Linux process RSS after the reads was 21,069,824 bytes (10k) and 21,028,864 bytes (100k), including the runtime, driver and database caches; these are observations around the measured queries, not whole-process peak memory limits. Race-instrumented memory/latency is separately recorded in the full qualification log and is not used as the ordinary performance figure.
