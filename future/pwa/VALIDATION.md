# PWA implementation validation — 2026-11-17

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
