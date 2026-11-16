# FUX-101 acceptance status

**Not complete. Integration HOLD.** Implementation exists in isolation; browser/platform and production-adapter qualification are not established.

| Requirement | Implemented behavior | Evidence / unmet gates |
|---|---|---|
| 1. Manifest, identity, scope, icons, navigation/help | Opt-in Go bundle, validated same-origin/base path, 192/512 original icons, native nav and install instructions | Go manifest/header/config tests pass; actual installed Android/iOS/desktop acceptance pending |
| 2. Public fingerprinted cache only | Workbox exact manifest; omitted credentials; public MIME/status/redirect rules; SHA-256; no private cache/route aliases | Node admission tests, build integrity checks, Go handler tests pass; actual CacheStorage/lifecycle suite blocked, not passed |
| 3. Offline refusal and no replay | Network-only nonpublic/unsafe requests; constant 503 unavailable navigation; no sync/queue/replay code | Node offline payload test and source review; real disconnect-before-submit and reconnect/no-replay pending |
| 4. Accessibility and no-JS | Semantic nav/help, native forms, safe areas, focus, reduced motion, wrapping layouts; no-JS synthetic flow test authored | Browser reflow/no-JS suite blocked; manual screen reader, 200–400% zoom, mobile/cross-browser pending |
| 5. Sign-out and shared-device privacy | Native sign-out adapter contract; private DOM destruction; BroadcastChannel lock; resume forces server reload | Code present and reviewable; real BFCache/account switch/cross-tab/OS-resume acceptance pending |

## Nonfunctional gates

- Go race tests/vet and Node policy tests: PASS. Library statement coverage 93.4%; demo server not covered by Go unit tests.
- Dependency audit: see VALIDATION.md. Build-only esbuild and runtime Workbox license texts retained. No Circula root license selected.
- Deterministic local build: two consecutive builds matched; SHA-256 manifest committed.
- Independent audit: pending fresh reviewer.
- Browser suite: authored in browser/test.mjs but blocked before test execution. Mocks/unit tests do not replace it.
- DB migration/rollback: not applicable; no private data persistence or schema change.
- Production routes/CSP/session/CSRF/tenant/idempotency/ledger qualification: pending later authorized integration.
- Decommission/unregister behavior: documented contract, future browser integration gate.
- Main merge, release, activation: prohibited by current HOLD.
