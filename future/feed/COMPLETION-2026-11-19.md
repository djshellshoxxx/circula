# FNET-204 isolated completion candidate — 2026-11-19

## Status and scope

**Implemented; qualification in progress. Disabled and integration HOLD.**

This record covers the user's isolated feed pagination/security task. It does not claim production Core integration, deployment, release readiness, browser accessibility, or completion of the broader preferences/UI roadmap. Current-head provider checks and real SQLite performance qualification are recorded separately below as they complete.

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
| Same PR #92, refreshed during this task | `c4732bc95f2f1a65e642c356b04ed916b1dfa4b4` | Core-only refresh; `future/feed/` and its workflow byte-identical to start. Reviewed without merging the new Core/session work |

All remote feed histories were compared through their file history and ancestry. No missing original feed fix or test was found outside the starting candidate. New work from separate cursor, performance and QA worktrees was cherry-picked into the dedicated feature branch. Agent commits remain in local provenance; publication may use a Git-data commit with an identical reviewed tree if shell Git cannot authenticate.

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
| Repository hygiene, action pins/permissions, workflow YAML | PASS |
| Root generated traceability check | PASS before final publication; unchanged Core matrix, module-local mapping in `TRACEABILITY.md` |
| Real SQLite integration/performance and current-head provider CI | Pending at this draft checkpoint; no result inferred from compilation or earlier revisions |

The new code adds database/error paths, so combined-module coverage is not directly comparable to the much smaller starting module. Integration-test coverage and race results are distinct from a statement-coverage percentage.

## Environment limitations and recovery

The first compile of the pinned SQLite driver was killed by the environment. The execution service then became unavailable, interrupting multiple pending runs. Results without completion were discarded. The workspace later recovered with source/worktrees intact. SQLite qualification was restarted with serial compilation, lower garbage-collection threshold and a memory limit. Shell Git read access works, but push lacks credentials; the authorized GitHub connector can publish a tree-verified draft without changing original branches.

## Independent review and integration dependencies

Independent source audit covered the bounded reader, SQL query shape, immutable/tombstoned identities, lookahead, insertion fence, v2 compatibility, padded v3 cryptography, error contracts and isolation. It found no blocking security/pagination issue beyond the cancellation repair above. Final candidate approval still requires inspecting the consolidated SHA, scope diff and full verification evidence. This is an independent agent review, not an external human security audit.

Remaining integration work is documented in `INTEGRATION-PROPOSAL.md`: qualify the final moving Core candidate; implement an authoritative transaction-scoped source/permission adapter; choose schema/retention/restore behavior; persist keys and define rotation/issuance limits and TTL; specify preference policy; add separately reviewed routes, rate limits, CLI and no-JS/browser UI. Empty continuation counts can reveal coarse activity volume; cursor encryption does not provide traffic-analysis privacy.
