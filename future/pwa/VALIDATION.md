# PWA implementation qualification record

## Refreshed isolated candidate — 2026-11-19

The candidate preserves original PWA source `a6326b8725fe6d7270997b4a284cbbc129022dd1` and merges core candidate `244e1356201e6477eab4694d82947f2058f3b0d8`. The PWA implementation, lockfile, compiled public assets, feature manifest and existing browser assertions are unchanged. The new work is qualification documentation and `.github/workflows/future-pwa.yml`. `feature.json` still records `enabled: false` and `integration: HOLD`; a synthetic demo's explicit opt-in does not enable the core application.

The 2026-11-19 local module audit passed the existing Go race suite (97.5% root-package statement coverage), `go vet`, format checks and all 22 Node tests. Its real-browser attempt was blocked before assertions by a workspace UNIX-socket permission denial (`EPERM`). No browser pass is claimed for that attempt, and no host protection was changed to bypass it. The earlier 2026-11-17 browser result is preserved below as historical evidence from the original source, not a result for this refreshed candidate.

### Dedicated provider workflow

The workflow is a separate, read-only-permission job on `ubuntu-24.04`, with immutable action revisions and no persisted checkout credentials. It uses the core `go.mod` toolchain pin, Node 24.22.3, the existing npm lockfile (including Playwright 1.62.1, esbuild 0.25.12 and Workbox 7.4.1), and no production secrets or services. Dependency lifecycle scripts are disabled; optional platform packages are retained so esbuild can use its locked platform binary.

The configured gates are:

1. Reject a changed HOLD/default-disabled boundary or third-party Go module dependency; record the candidate SHA, actual checkout commit/tree and tool versions.
2. Install locked npm dependencies, rebuild the existing bundle, verify every `BUILD-SHA256SUMS` entry, and reject tracked or untracked public-bundle changes.
3. Run the existing 22 Node tests, Go format checks, `go vet`, and the complete nested-module race suite. No SQL/cgo-specific test tag applies to this module.
4. Install the Chromium revision selected by the locked Playwright package using `npx --no-install playwright install --with-deps chromium`. Obtain its executable path from that installed package, not the runner's unrelated system browser.
5. Compile the existing synthetic demo, launch it directly on `127.0.0.1:18761`, check its process and HTTP readiness within a bounded loop, and run the existing `npm run test:browser` with a five-minute hard limit. A shell trap terminates and waits for the actual demo PID. The job and browser step also have outer time limits.
6. Preserve toolchain, dependency/build, hash, test, Chromium-version and demo logs even when a gate fails. A browser launch or acceptance failure remains a failed job; there is no fallback browser or assertion bypass.

The preserved browser harness already launches Chromium with `--no-sandbox`; this candidate does not change that launch or the host security policy. Its fixed loopback origin, synthetic A/B identities and in-memory trade counter exercise application behavior on an ephemeral runner. These checks qualify the listed functional assertions only. They do **not** establish Chromium OS-sandbox isolation, production identity or ledger correctness, mobile installation, other browser engines, real BFCache restore, real device resume or the remaining manual acceptance gates below.

At preparation time, **provider execution is pending**. A future result must identify the exact PR head and workflow run, with its checked-out commit/tree and logs. This workflow does not qualify the independently maintained core, authorize integration or deployment, or remove any remaining HOLD gate. Original `feature.json` status text is historical scope, not proof of a new provider run.

## Historical implementation validation — 2026-11-17

All results below were observed in this run (Go 1.26.11 via `GOTOOLCHAIN=go1.26.11`, Node 22.23.4, npm 10.9.9, Chrome for Testing 153.0.8010.12 at `/opt/pw-browsers/chromium-1243/chrome-linux64/chrome`, Playwright 1.62.1). The module has no SQL/cgo dependency, so no cgo/SQLite test tag applies.

```
$ GOTOOLCHAIN=go1.26.11 GOMAXPROCS=2 go test -p 1 -race -count=1 -cover ./...
ok  	github.com/djshellshoxxx/circula/future/pwa	1.074s	coverage: 97.5% of statements
	github.com/djshellshoxxx/circula/future/pwa/cmd/demo		coverage: 0.0% of statements
$ go vet ./...
exit=0
$ gofmt -l .
(no output)
$ npm test
# tests 22
# suites 0
# pass 22
# fail 0
$ npm audit
found 0 vulnerabilities
$ npm run build twice; sha256sum public/* | diff - BUILD-SHA256SUMS
identical
$ CHROMIUM=/opt/pw-browsers/chromium-1243/chrome-linux64/chrome PWA_TEST_ORIGIN=http://127.0.0.1:18761 npm run test:browser
PASS: real Chromium install/CacheStorage, privacy, offline POST/no replay, account switch/back, DOM scrub (also without SW/BroadcastChannel), oversize/tampered precache rejection, 320px/400%-zoom reflow, skip link and no-JS flows
```

Prerequisite for the browser run: `npm ci --ignore-scripts` then `npm run build`, then `go run ./cmd/demo -port 18761` in another shell. A 15 s native fuzz of `Cacheable` (`go test -run xxx -fuzz FuzzCacheable -fuzztime 15s .`) executed 324,457 inputs with no failure (invariants: only GET, https, no `?`/`#`, no `..`, configured prefix).

## What the tests cover

- Go: config boundaries, default-disabled 404, headers/manifest/scope, HEAD parity, concurrent serving (8 goroutines, race detector), hostile build catalogs (17 mutation cases: size, MIME, hash, duplicate, traversal, version injection, too many/oversize assets, missing files), policy and fuzz.
- Node (22 tests): strict request/response admission; offline page; bounded streaming reads (oversize `Content-Length`, endless stream with cancel, lying/malformed lengths, stream errors, invalid bounds); large base64; verified-response rebuild; plugin URL/credential rules; privacy lifecycle on a DOM double with and without service worker / BroadcastChannel / valid base / throwing channel.
- Browser (real Chromium 153): install and CacheStorage contents, obsolete-cache cleanup preserving unrelated caches, manifest, offline POST failure and no replay, account switch/back, private DOM scrub, session-ended/pagehide, privacy lifecycle with service-worker API removed / BroadcastChannel removed / throwing / workers blocked, oversize (3 MiB) and tampered stylesheet leave worker inactive and cache empty (identical-bytes control installs 4 entries), 320 px and 320x256 reflow, keyboard skip link, no-JS flows.

## Mutation check

The pre-fix `client.mjs` was temporarily restored and rebuilt: the browser suite failed with `no service worker API: pagehide scrub`. The fixed client was restored and the build re-verified identical to `BUILD-SHA256SUMS`.

## Not passed / not established

- Real BFCache restore: with `--enable-features=BackForwardCache` and the Playwright BFCache-disabling default removed, Chromium still reloaded the page on Back (page-state marker lost), because `Cache-Control: no-store` documents are not BFCache-eligible there. The BFCache privacy path is therefore covered only by the DOM-double unit test and simulated `pagehide`; no real BFCache restore is claimed.
- Android/iOS install and reopen, OS app-switcher snapshots, Safari/Firefox, manual screen reader, WCAG 2.2 AA automation, real OS zoom, tenant/CSRF/idempotency/ledger regression against the real core: all OPEN (see AUDIT.md, COMPLETION.md).
- `cmd/demo` is covered only by the browser suite, not Go unit tests.
- npm audit is a point-in-time registry result, not a security guarantee.
