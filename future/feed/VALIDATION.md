# Independent consolidation and repair — 2026-11-18 Pacific / 2026-11-19 UTC

## Final core refresh

The publication candidate merges final core audit head
`4140bc135479077f6297d53ed2e9dfcd85c056d6`. The module's Go source remains
byte-identical to the previously tested repair. The generated traceability
conflict is resolved by rerunning the checked-in generator on the combined
source with published core base `4140bc1`; no generated row is selected by hand.
The dedicated workflow now requests 100,000 cursor-fuzz executions with a
three-minute hard timeout, matching the core's workaround for the pinned Go
fuzz timer defect. That new provider run remains pending; the measured
20-second run below is retained as historical execution evidence.

## Consolidation and repair evidence

Local integration branch `audit/future-feed-cursors-20261119` starts at module
branch `5d3819c3b3af031b75eceb5251da0226a4ecee77`. It merges PR #67 source
`7a528c611bffdf92ef9f8cbddb98e8b088eb5036` and PR #80 source
`0952c94a5dbab4044d7673ce9d51cec9063ab97f`, preserving their histories. The
combined source before new repairs is `9ad3237d7948392896857aac2c939bad0f61a389`.
The isolated source changes remain within `future/feed/`. The branch then
merges published core audit candidate PR #90 at
`9b1fc6a0bb4ae4ad95c4f505929ce8088f10bf2c`; that candidate supplies the core
refresh and remains an independently unqualified dependency. Relative to that
candidate, the only core source change is the reviewed monetary-guard test
follow-up `d627af48ff4042db73adac9a72f6208db268a924` and its audit record;
no core runtime code, schema or root dependency differs. A dedicated
`.github/workflows/future-feed.yml` now
runs the nested module's checks on relevant pushes and pull requests. The
existing core workflows remain unchanged. Integration remains disabled/HOLD.

The independent review reproduced and repaired two defects:

1. Conflicting duplicate source snapshots silently selected whichever row came
   first. Reversing the input changed the returned creation time or owner, and
   a stale visible/active copy survived a hidden/inactive duplicate. The new
   `TestPageRejectsConflictingSourceSnapshots` fails on the combined source in
   all four cases and both row orders. The implementation now refuses
   contradictory snapshots, while equivalent timestamps in different time
   zones still de-duplicate and other networks remain isolated.
2. A zero-value exported `CursorCodec` panicked during `Seal` and `Open`, while
   a last-page request could return results with an unconfigured codec.
   `TestZeroValueCursorCodecFailsClosed` reproduces all three cases before the
   repair. Nil and zero-value codecs now refuse every operation through the
   existing error contracts.

Observed checks with Go 1.26.11 linux/amd64, `GOTOOLCHAIN=local`, CGO enabled,
and `GOMAXPROCS=2`:

| Check | Observed result |
|---|---|
| Combined source baseline: `go test -race -count=1 -cover ./...` | PASS; 98.8% statement coverage |
| New regressions on unchanged combined source | FAIL: both defect groups reproduced |
| Repaired source: `go test -race -count=1 -cover ./...` | PASS; 28 tests plus fuzz seeds; 98.8% statement coverage |
| `go vet ./...` | PASS, no diagnostics |
| `gofmt -l .` and `git diff --check` | PASS, no output |
| `go test -run='^$' -fuzz=FuzzOpenCursor -fuzztime=20s -parallel=2` | PASS; 66,869 executions in 21.074 s; 37 seed inputs and 52 new interesting inputs |
| Independent security review and repeat `go test -race -p=1 -count=1 -cover ./...` | Approved for isolated development; PASS in 1.899 s, 98.8% coverage |
| Dedicated workflow configuration | YAML and shell syntax, pinned actions, read-only permissions, and the live disabled/HOLD module-boundary check pass locally; remote Actions execution is pending publication |
| After PR #90 refresh: `go test -race -p=1 -count=1 -cover ./...` | PASS; 98.8% coverage; nested module only |
| Initial PR #90 traceability check | FAIL; the added feed pagination citations changed the generated inventory; no requirement completion was inferred |
| Regenerate inventory and repeat `go run -p=1 ./scripts/traceability -check` from the root | PASS; checked-in generator adds API-034/API-035 feed citations; the latest caller-supplied base is published core candidate `9b1fc6a`; runtime execution evidence remains separate |
| Reviewed monetary-guard follow-up: focused race test for `TestNoFloatsInMoneyPackages` and `TestIntegerOnlyMoneyArithmeticGuard` | PASS in 2.450 s; exact integer aggregation remains permitted and floating-point/rational money APIs remain rejected |

These results qualify the isolated projection/cursor code. They do not qualify
core integration, a persistent snapshot adapter, key custody, browser workflows,
or any of the remaining feature work. Existing historical results below keep
their original toolchain and source attribution.

# Validation record — 2026-11-18 (signed cursors, PR #67 follow-up)

Branch `claude/fnet-204-signed-cursors`, from PR #67 head `7a528c61`. Toolchain: Go 1.25.0 linux/amd64 (selected by `go.mod`), standard library only. The root application was not built or tested; no file outside `future/feed/` changed.

```
$ gofmt -l .
(no output; at 7a528c61 it listed cursor_auth.go and cursor_auth_test.go)
$ go vet ./...
(exit 0, no diagnostics)
$ go test -count=1 ./...
ok  	github.com/djshellshoxxx/circula/future/feed	0.022s
$ CGO_ENABLED=1 go test -race -count=1 -cover ./...
ok  	github.com/djshellshoxxx/circula/future/feed	1.276s	coverage: 98.8% of statements
$ go test -run='^$' -fuzz=FuzzOpenCursor -fuzztime=60s
fuzz: elapsed: 1m0s, execs: 1361653 (13239/sec), new interesting: 178 (total: 215)
PASS
ok  	github.com/djshellshoxxx/circula/future/feed	60.934s
```

25 tests plus `FuzzOpenCursor` (37 seed inputs). Coverage was 84.5% at `7a528c61`. The only uncovered statements are two unreachable guards in `sealWith` (a `json.Marshal` error, and a token over 2,048 bytes, which current bounds rule out).

## PR #67 review findings

| Finding | Status | Evidence |
|---|---|---|
| Medium: no expiry (`t` is the row time) | Fixed. Payload carries issue time `iat` from an injected clock; TTL configurable, default 1 h **provisional, owner to confirm** | `TestCursorExpiry`, `TestNewCursorCodecOptions`, `TestPageExpiredCursor` |
| Medium: no key ID or rotation | Fixed (mechanism). Key ID in the token and in the MAC; `KeySet` signs with the current key and verifies by ID. Custody, persistence and cadence are documented owner decisions | `TestCursorKeyRotation`, `TestPageRotatedKeyContinuesPaging`, `TestNewKeyRing` |
| Medium: no actor, route or filter binding | Fixed. MAC binds network, viewer and a canonical filter digest (kinds and preference revision); the label binds the feed route | `TestCursorBinding`, `TestPageCursorBoundToNetworkViewerAndFilter`, `TestFilterCanonical` |
| Low: no domain separation | Fixed. Length-prefixed MAC input under label `circula/future/feed-cursor/v2` | `TestCursorDomainSeparation` |
| Low: Seal/Open not wired into `Page()`; no round trip | Fixed. `Page` takes and returns opaque tokens only | `TestCursorRoundTrip`, `TestPageWalksWholeFeedWithSignedCursors` |
| Low: tamper test may fail on length, not MAC | Fixed. Every MAC byte flipped at equal length and alphabet; plus truncation, garbage, invalid and non-canonical base64, padding, line breaks, oversize, and authentic MACs over bad payloads | `TestCursorTamperedMACSameLength`, `TestCursorMalformedInputs`, `TestNonCanonicalHelperKeepsBytes` |
| Low: `gofmt` flags both files | Fixed | `gofmt -l .` above |
| Low: network ID readable in the token | Fixed. Network, viewer and filter are only MAC input; the token shows key ID, issue time and position | `TestCursorTokenOmitsBoundIdentifiers`; fuzz property: readable segments do not depend on the binding |
| Low: README and VALIDATION.md not updated | Fixed | This file, README.md |
| Uniform error and constant-time compare | Kept. Every refusal is exactly `ErrInvalidCursor` with a zero `Cursor`; `hmac.Equal` before parsing | `TestCursorUniformRefusal` |

`FuzzOpenCursor` checks that no input panics, every refusal is exactly `ErrInvalidCursor`, an accepted token is one the fixture minted for an equivalent binding (a forgery would fail the run), and a token minted for fuzzed inputs round-trips, still verifies after rotation, and is refused under another network, viewer, swapped network/viewer, filter, preference revision, retired key, other key or after its TTL.

An ad-hoc mutation check (not committed) applied 23 single mutations to `cursor_auth.go` and `feed.go`, such as dropping each MAC field, the length prefix, the expiry, future-issue, version or alphabet checks, strict base64, the key copy and the next-cursor condition. The suite failed on all 23.

Coverage is not proof of security or feature completion. Owner decisions (key custody and rotation cadence, TTL value, mute/follow semantics, persistent source joins) and the remaining feature work in README.md are open. No claim of production qualification or release readiness is made.

# Validation record — 2026-11-16

Toolchain: Go 1.25.4 linux/amd64, standard library only. The root application requires Go 1.26.11; it was not built or tested here. Its files remain unchanged.

```
$ go test -race -cover ./...
ok  	github.com/djshellshoxxx/circula/future/feed	1.011s	coverage: 82.1% of statements
$ go vet ./...
PASS (exit 0, no diagnostics)
```

Manual source review checked: no app/internal imports, no cross-feature imports, no init/registration or background jobs, no external dependencies, no network transport or production DB writes, and no ledger posting API. All feature inputs requiring trusted current state are documented as adapter responsibilities.

The suite tests this initial core milestone only. Coverage is not proof of security or feature completion. Durable stores, fresh authorization adapters, full acceptance tests, browser/CLI experience and independent security review remain unfinished as detailed in README.md. No claim of race-safe DB persistence, production qualification or release readiness is made.
