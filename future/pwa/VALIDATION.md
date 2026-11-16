# PWA implementation validation — 2026-11-16

## Passed

- `GOTOOLCHAIN=go1.26.11 go test -race -cover ./...`: library PASS, 93.4% statements. `cmd/demo` has no Go unit test coverage and is not qualified production code.
- `go vet ./...`: exit 0, no diagnostics.
- `npm test`: 4 Node tests pass (strict request admission; public response MIME/cache policy; nonprivate inactive offline page; redirect/cookie response rejection).
- `npm run build`: pinned Workbox/esbuild build succeeds. Two consecutive builds produced byte-identical SHA-256 sums. `BUILD-SHA256SUMS` records distributed files.
- `npm audit --omit=dev`: 0 known vulnerabilities reported. Full `npm audit` after pinning Playwright also reports 0 known vulnerabilities. These are point-in-time registry advisory results, not a security guarantee.

## Blocked, not passed

Attempted real browser suite with Playwright and `/usr/bin/chromium`. Chromium exited before page creation:

```
FATAL:chrome/browser/process_singleton_posix.cc:297
socket() failed: Operation not permitted (1)
```

Attempted Playwright chromium-headless-shell fallback; all download retries returned invalid/truncated ZIP data (`End of central directory record signature not found`). No browser assertion ran successfully. No Chromium, mobile, BFCache, cross-tab, no-JS UI or accessibility pass is claimed.

The authored suite checks public CacheStorage entries, old-cache cleanup while preserving unrelated caches, manifest, 320px reflow, private response exclusion, offline POST failure/no replay, account-switch/back privacy, session/pagehide DOM scrub, and native no-JS flows. Run it against the loopback harness in a browser-capable environment; a fresh auditor must inspect and execute it. Tests disabling BFCache by browser automation defaults are not sufficient for real BFCache qualification.

## Scope and remaining tests

No production Circula file changed and no root application test/qualification was performed by this implementation agent. The private-view wrapper and sign-out placement contract needs real auth-adapter testing. Native iOS/Android install/reopen, OS task-switcher snapshots, manual screen reader/zoom, supported browsers and full tenant/session/ledger regression remain pending. See COMPLETION.md and INTEGRATION.md.
