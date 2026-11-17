# FUX-101 self-audit — 2026-11-17

**Integration remains HOLD. This feature is not complete or release-qualified.** This is the implementing agent's own requirement-by-requirement pass, not an independent third-party audit; a fresh reviewer must still reproduce it.

## Recovery provenance

The earlier independent auditor's worktree (`/workspace/scratch/...`) was not recoverable and its diff was never published. Only its two findings survive, as text in the execution-recovery checkpoint. Both were **re-implemented from the description**, not restored from the original diff, and each has new regression tests that fail on the pre-fix code (the browser test was confirmed to fail against the old `client.mjs`).

| Reported finding | Fix | Evidence |
|---|---|---|
| Privacy lifecycle init depended on service worker / BroadcastChannel availability | `src/privacy.mjs` `initPrivacy` runs unconditionally from `client.mjs`; base path, channel construction/`postMessage`/`addEventListener` and `serviceWorker` are each optional and individually exception-isolated; registration moved to a separate optional `registerWorker`. Added `freeze` scrub; restore marker is `data-scrubbed` (not the generic `hidden`) | `src/privacy.test.mjs` (9 tests); `browser/test.mjs` runs sign-in, pagehide scrub, session-ended lock and sign-out with no `navigator.serviceWorker`, no `BroadcastChannel`, a throwing `BroadcastChannel`, and workers blocked |
| Unbounded precache response reads | `src/precache.mjs`: streaming `readBounded` (cancels the stream at the bound, validates `Content-Length` syntax/size/consistency), hard 1 MiB ceiling, exact per-asset `size` from the build catalog, SHA-256 over the bounded bytes, chunked base64 (no argument-spread overflow), stored response rebuilt from the verified bytes. Go loader rejects catalogs with wrong/missing size, wrong MIME for extension, oversize assets, >32 assets, duplicates, path separators, non-fingerprinted names, malformed version | `src/precache.test.mjs` (10 tests); `hardening_test.go` (hostile catalog table, boundary, fail-closed); browser: oversize (3 MiB) and tampered stylesheet leave the worker inactive with 0 cache entries, identical bytes control installs 4 entries |

## Spec checklist (`docs/future-features/01-installable-pwa.md`)

Legend: DONE = implemented and covered by tests listed; PARTIAL = implemented, a stated gate is open; OPEN = not done / outside isolated scope.

| # | Requirement | Status | Notes |
|---|---|---|---|
| F1 | Same-origin manifest, stable scope/start URL, install help, icons, home navigation Find/Trade/Balance/Inbox | PARTIAL | Manifest/scope/icons/help/nav links DONE (Go + browser tests). Navigation is on the install-help page and the synthetic harness; real home navigation is the host app's templates (adapter contract). Installed-app acceptance on Android/iOS/desktop OPEN |
| F2 | Precache only fingerprinted public assets; authenticated HTML/API/tokens/receipts/exports no-store, never cached | DONE (isolated) | Exact-URL allowlist, credentials omitted, MIME/Cache-Control/redirect checks, SHA-256 + bounded read, private/unsafe requests network-only with `no-store` forced. Real Chromium CacheStorage checks pass. Production routes/headers unverified (HOLD) |
| F3 | Offline shows 'unavailable', refuses create/accept/decline/reverse, no replay/queue/offline balance | DONE (Chromium) | Constant 503 offline page, no sync/queue code; real Chromium offline POST fails, count unchanged after reconnect. Real Android/iOS offline OPEN |
| F4 | Screen-reader labels, safe areas, reduced motion, 200–400% zoom, 320px reflow, plain-HTML workflows | PARTIAL | Landmarks/labels/skip link/focus styles, safe-area padding, reduced-motion CSS; Chromium reflow at 320x720 and 320x256 (400% of 1280x1024) and keyboard skip-link, no-JS flow pass. Manual screen reader, WCAG 2.2 AA automation (axe-core is MPL-2.0, deliberately not added), cross-browser, real zoom/touch-target review OPEN |
| F5 | Visible sign-out, scrub in-memory private views, no stale content on reopen/shared device | PARTIAL | Native POST sign-out contract; DOM scrub on pagehide/freeze/hidden; reload on resume; cross-tab lock; Chromium account-switch/back tested. Real BFCache: Chromium does not BFCache `Cache-Control: no-store` documents, so with BFCache enabled the page was reloaded, not restored (observed `__marker` lost) — BFCache restore behavior is **not exercised**; Safari/Firefox BFCache, OS app-switcher snapshots, installed-app resume OPEN |
| P1 | Persistence/migration | N/A by design | Spec forbids new authoritative tables; no private data is persisted anywhere. Optional per-member presentation preferences were deliberately **not** implemented (would add private per-member state and an unneeded SQLite adapter/migration). If the owner wants them, that needs a separate decision. Service-worker/cache versioning and obsolete-cache purge are DONE (browser-tested, unrelated caches preserved) |
| I1 | Reuse existing /api/v1, no parallel wallet/posting API; Go-served static assets with explicit cache headers; reverse-proxy/base-path compatible | DONE (isolated) | No API handled at all; explicit headers; configurable base path. openapi/contract docs not touched (HOLD) |
| S1 | HTTPS outside loopback; scope bounded | DONE | Config rejects non-HTTPS except explicit loopback; strict base path; `Service-Worker-Allowed` = base |
| S2 | No client storage of auth/private inbox/proposals/credentials | DONE | No localStorage/IndexedDB/Cache use for private data; static review plus browser cache inspection |
| S3 | Network status from successful server communication, not `navigator.onLine` | DONE | Offline page only on actual fetch failure; `onLine` never consulted |
| A1 | Install/reopen on Android Chromium + iOS Safari; login, browse, propose, accept a synthetic trade | OPEN | Needs devices |
| A2 | Disconnect before submit: no posting, no replay | DONE (Chromium) | `browser/test.mjs` |
| A3 | Switch account / sign out / Back / offline reload: no prior private data | DONE (Chromium, non-BFCache) | See F5 caveat |
| A4 | Cross-browser keyboard/screen-reader/mobile accessibility, CSP/asset integrity | PARTIAL | Chromium keyboard + asset integrity + proposed CSP header DONE; other browsers/screen readers OPEN |
| X1 | Migration rollback, restart recovery | N/A / DONE | Stateless; restart = new process serving identical embedded bytes (concurrency test) |
| X2 | `go test -race`, no-JS paths, SCA, secret scan | DONE / PARTIAL | See VALIDATION.md. `npm audit` 0 known; no secrets scanner run beyond review |
| X3 | Auth/tenant threat matrix, API compatibility, full ledger regression | OPEN | Requires the real core adapter (HOLD) |
| D1 | Definition of done: owner approval, license decision, CI/SCA/human checks, opt-in rollout | OPEN | Owner-gated; Circula has no approved product license |

## Findings still open

1. No real BFCache restore test; Chromium excludes no-store pages. Treat the pagehide-scrub + `pageshow.persisted` reload path as unit-tested (fake DOM) and Chromium-simulated only.
2. Private-view wrapper (`[data-private]`) and sign-out placement are an adapter contract, not verified against Circula's real templates.
3. The harness uses trivial A/B cookies; CSRF/tenant/idempotency are untested here by design.
4. The `Set-Cookie` response-header check in `responseAllowed` is inert in browsers (forbidden response header); credential omission is what actually protects.
5. `cmd/demo` has no Go unit coverage; it is exercised by the browser suite only.
6. The original auditor may have found more than these two items; only these two were recorded.
