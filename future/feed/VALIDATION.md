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
