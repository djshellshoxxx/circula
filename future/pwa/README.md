# FUX-101: Installable PWA

**HOLD — isolated development, incomplete feature, not approved for integration or release.**

The owner authorized independent future-feature development on 2026-11-16, while explicitly deferring integration until Circula and the feature are complete. This authorizes this branch's standalone development; it does not authorize enabling, merging or deploying the feature. Historical implementation prohibitions in the copied planning documents are superseded only for this isolated development scope.

## Isolation contract

- Branch: `future/fux-101-pwa-20261116`. Baseline: `2eacf7718f5bb3da3be5a288024af879076a9597`.
- Only `future/pwa/` is added. No application imports, routes, migrations, root dependencies, feature flags, workflows, jobs or runtime configuration are changed.
- This nested Go module has no third-party Go dependencies and no dependency on another future module. It is excluded from the root module's normal `go test ./...` and build traversal.
- No `init`, automatic registration, background execution, DB connection, external service or ledger write is present.
- Inputs describing authorization, active membership, visibility or transaction facts are **trusted adapter snapshots**, never client JSON. This package is not an HTTP authorization boundary.
- No production persistence is provided. Pure preflight checks do not guarantee atomicity, idempotency, authorization freshness or restart durability; future adapters must supply these within existing service/store transactions.
- Spec copies are historical reference snapshots; their original relative links refer to the source branch layout. No third-party source was copied.

## Implemented isolated feature

- Pinned Workbox 7.4.1 bundled locally; exact request allowlist, explicit public/MIME response policy, credential omission, SHA-256 content integrity, scoped cache lifecycle and no background replay.
- Original fingerprinted CSS/client JS and 192/512 PNG icons; deterministic build, lockfile and full third-party notices.
- Explicitly opt-in Go handler for manifest, worker, assets and accessible install help. Default configuration serves 404. Configurable base path and same-origin scope; loopback HTTP requires an explicit development option.
- Constant public offline-unavailable navigation response. Private requests and unsafe methods use the network without storage/queueing.
- Private DOM scrub on pagehide/visibility loss; authenticated reload on resume; session-ended/POST sign-out locking and cross-tab BroadcastChannel. No credential or private HTML local storage.
- Native navigation and install guidance, safe-area/reduced-motion styles and no-JS synthetic online flows.
- Audit fixes (re-implemented 2026-11-17 from the recovery notes): private-view lifecycle initializes independently of service-worker/BroadcastChannel support; precache responses are read through a streaming size bound with exact build-recorded sizes and a 1 MiB ceiling, and the Go loader validates catalog size/MIME/names. See `AUDIT.md`.
- Loopback-only synthetic demo plus Go, Node, and browser acceptance suites. The browser suite passed on real Chromium 153 on 2026-11-17 (see `VALIDATION.md`).

## Build and validate

```sh
npm ci --ignore-scripts
npm run build
npm test
go test -race -cover ./...
go vet ./...
npm audit
```

The nested Go module remains dependency-free; browser dependencies are npm-pinned and bundled. No node_modules or package caches are committed. `BUILD-SHA256SUMS` records distributable bytes; `THIRD_PARTY_NOTICES.md` records reuse and licenses.

Run the synthetic harness with `go run ./cmd/demo -port 18761`, then open `http://127.0.0.1:18761/app/`. With Playwright available: `CHROMIUM=/path/to/chrome PWA_TEST_ORIGIN=http://127.0.0.1:18761 npm run test:browser` (default `/usr/bin/chromium`). The A/B demo cookies are fixtures, never production authentication.

## Remaining acceptance gates

See `AUDIT.md` (requirement checklist) and `COMPLETION.md`. Chromium-only browser acceptance passes. Still open: Android/iOS installation and resume, real BFCache restore (Chromium does not BFCache no-store pages, so it was not exercised), Safari/Firefox, OS task-switcher snapshots, manual screen-reader and real zoom qualification, WCAG 2.2 AA automation, an independent third-party audit, and production adapter/tenant/CSRF/idempotency/ledger regression. No persistence is implemented by design (the spec allows none beyond optional presentation preferences, which are intentionally omitted). **The feature is not complete or release-qualified.**

The future integration contract and actual core seams are documented in `INTEGRATION.md` and `UPSTREAM-RESEARCH.md`. No root application behavior or core release status changed.

## Later integration, one feature at a time

1. Finish this module's remaining scope and acceptance tests. Audit its own code and record unresolved findings.
2. Qualify the then-current Circula core separately. Do not infer completion from this module's tests.
3. Rebase this branch on the qualified core; review the baseline diff and any API/schema drift. No promise of a conflict-free future merge is made.
4. Add a narrow adapter, reviewed API contracts, additive migrations (if needed), explicit per-network opt-in, CLI and accessible/no-JS UI in a separate integration change. Keep all other future features absent.
5. Recheck tenant/auth/CSRF/idempotency behavior, concurrency/restart, backup/restore, full ledger regression and relevant manual/browser acceptance.
6. Obtain the owner's explicit selection of this feature for integration. Only then merge that single integration change. Never auto-merge this hold branch or batch-enable the backlog.
7. Disable by removing routes/jobs and adapter registration; do not roll back ledger history. Any feature-data retention/migration procedure must be documented before integration.

No work on this branch changes Circula's core completion or release status.
