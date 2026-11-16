# FUX-101 upstream research and implementation handoff

Research date: 2026-11-16. Phase: **research complete; implementation and independent audit still pending**. Integration remains **HOLD**. This document does not claim the PWA feature is complete.

## Decision

Use **GoogleChrome/workbox 7.4.1** as a pinned, locally bundled service-worker dependency, concentrating reuse on its precache lifecycle and route/strategy infrastructure. Do not copy an entire PWA application, introduce Vite into the production Go application, or enable generic offline-first recipes. Retain Circula-specific privacy policy as a small independent layer.

This choice provides mature upstream implementation for the difficult service-worker lifecycle without inheriting another application's authentication, ledger, data cache or background posting behavior. It does **not** satisfy installability, privacy, accessibility or browser acceptance by itself.

The implementation agent should work exclusively inside `future/pwa/`, including any `package.json`, lockfile, bundled assets, source notices and synthetic demo/test server. No changes to main application routes, CSP, templates, config or workflows are authorized in this phase.

## Candidates verified

GitHub repository search located all three projects; shallow clones were inspected directly, including license text and source. All were public and unarchived in the GitHub connector response. Dates below are the latest inspected commit dates, not a promise of future maintenance. Package version and repository HEAD are distinguished explicitly.

| Candidate | Exact reviewed identity | Maturity and useful code | Decision and limitations |
|---|---|---|---|
| [GoogleChrome/workbox](https://github.com/GoogleChrome/workbox) | `v7.4.1` resolves to commit `62b9d8ba8eb3c1a2ab8aac9d84c90cda7865d6a3`; tag object `8674ff37178007fb2c5b225d124b224c5055145f`. Reviewed HEAD `4ea3138a120fd2c87389b54130e25f47e0fd7089`, 2026-09-02. `workbox-precaching` package version 7.4.1. | Google-maintained modular service-worker project; release tags, browser test suites, typed implementation, detailed official docs. `PrecacheController`, route matching and strategy plugins are directly reusable. | **Selected**, release pin rather than mutable HEAD. Dependencies must be bundled locally and lockfile/integrity recorded. Defaults are not Circula's security policy. |
| [vite-pwa/vite-plugin-pwa](https://github.com/vite-pwa/vite-plugin-pwa) | HEAD `355cf3dd283ca50da3ac4759501e9c3312a3da91`, 2026-10-09, package version 2.0.0. Remote tag `v2.0.0` exists as `35bfa7df1eeda2f906084833a7ff0f95689f77b3`; not selected or assumed identical to HEAD. | Active dedicated PWA project with framework integrations and `injectManifest` support. Source `src/vite-build.ts` compiles custom workers; `src/options.ts` owns default configuration. | Rejected as runtime/build adoption: Circula has server-rendered Go templates, not Vite. Its default navigation fallback is `index.html`, which is wrong for authenticated server HTML. Official website currently reports v1.2.0 while inspected package reports 2.0.0; do not assume online docs describe every HEAD behavior. |
| [alphagov/govuk-frontend](https://github.com/alphagov/govuk-frontend) | HEAD `a11bd3316363ab4702fcef81d6ce06a947c464ef`, 2026-11-12; package version 6.6.0; `v6.6.0` tag exists as `05a21ee588e0ae14e09b2754eb30b21310563459`. | Established accessible server-rendered component project. Reviewed `packages/govuk-frontend/src/govuk/components/skip-link/template.njk` and `service-navigation/template.njk`; tests and review application exist. | Design reference for progressive navigation only. Circula already has a skip link, native form controls and no-JS navigation. Copying a full Nunjucks/Sass design system would create unnecessary coupling; do not import government branding/fonts/assets. |

No upstream code has been copied by this research phase. No dependency has been installed in the module.

## License and provenance findings

- Workbox release [LICENSE](https://github.com/GoogleChrome/workbox/blob/62b9d8ba8eb3c1a2ab8aac9d84c90cda7865d6a3/LICENSE): MIT-style license, copyright 2018 Google LLC. Reviewed source files also carry per-file Google MIT notices. Preserve license/copyright with any bundled distribution or adapted substantial portions. `workbox-precaching` depends on exactly `workbox-core`, `workbox-routing`, and `workbox-strategies` 7.4.1 in the inspected release manifest; inspect every installed package/license in the lockfile, not just the top-level package.
- Vite PWA [LICENSE](https://github.com/vite-pwa/vite-plugin-pwa/blob/355cf3dd283ca50da3ac4759501e9c3312a3da91/LICENSE): MIT, copyright 2020–present Anthony Fu. Not selected; no code to attribute unless later copied.
- GOV.UK Frontend [LICENSE.txt](https://github.com/alphagov/govuk-frontend/blob/a11bd3316363ab4702fcef81d6ce06a947c464ef/LICENSE.txt): MIT, Crown Copyright/Government Digital Service. Do not assume separate branding, image or font rights follow from the software license.
- Circula's own product license remains undecided. Do not add a root license, relicense Circula, remove notices, or infer permission to redistribute branding. The owner has authorized isolated borrowing/adaptation now; preserve permissive source notices within the module and leave distribution/license review explicit before release.
- New implementation should produce `THIRD_PARTY_NOTICES.md` with package, exact version, upstream commit, license, adopted files/APIs, build command, local artifact checksum and modifications. Include the actual license text with bundled third-party assets.

## Specific upstream reuse seams

At selected release commit:

1. [`packages/workbox-precaching/src/PrecacheController.ts`](https://github.com/GoogleChrome/workbox/blob/62b9d8ba8eb3c1a2ab8aac9d84c90cda7865d6a3/packages/workbox-precaching/src/PrecacheController.ts): reuse install/activate precache processing and obsolete-entry cleanup. Supply a module-specific cache name and only build-generated public asset entries.
2. [`packages/workbox-precaching/src/PrecacheStrategy.ts`](https://github.com/GoogleChrome/workbox/blob/62b9d8ba8eb3c1a2ab8aac9d84c90cda7865d6a3/packages/workbox-precaching/src/PrecacheStrategy.ts): use plugin API for cache admission. **Default admission accepts responses below status 400 and does not enforce Circula's private/no-store boundary.** It also has redirected-response copying. A custom plugin must reject redirects before storage, and refuse private/no-store responses regardless of successful status.
3. [`packages/workbox-precaching/src/PrecacheRoute.ts`](https://github.com/GoogleChrome/workbox/blob/62b9d8ba8eb3c1a2ab8aac9d84c90cda7865d6a3/packages/workbox-precaching/src/PrecacheRoute.ts): do not use its default URL aliases blindly. Defaults ignore `utm_`/`fbclid` parameters and try directory-index/clean-URL variants. Use a strict request guard followed by exact precache lookup, or explicitly disable aliases and still reject every query/fragment-bearing request.
4. Upstream tests under `test/workbox-precaching/sw/test-PrecacheController.mjs`, `test-PrecacheRoute.mjs`, and `test-PrecacheStrategy.mjs` provide useful lifecycle cases. Adapt targeted tests with provenance if copying; do not transplant the monorepo's entire test infrastructure.

Official API references reviewed: [precaching](https://developer.chrome.com/docs/workbox/modules/workbox-precaching), [routing](https://developer.chrome.com/docs/workbox/modules/workbox-routing), and [Vite PWA injectManifest](https://vite-pwa-org.netlify.app/guide/inject-manifest). The first two document lifecycle, versioned cache entries, route ordering and custom match functions. The Vite page confirms custom worker compilation but is reference only because Vite is not selected.

## Existing Circula seams and mismatches

Read-only review at baseline `2eacf7718f5bb3da3be5a288024af879076a9597`:

- `internal/web/web.go` embeds `templates/*.html` and `static/*`; assets are served at `/static/`. Current PWA prototype's `/assets/` prefix therefore does **not** directly match the application. Make the isolated module's public path/base prefix configurable and test it, then document the future adapter.
- `internal/web/middleware.go` defaults to `Cache-Control: no-store` and special-cases `/static/` for public caching. The static names in `base.html` (`style.css`, `app.js`) are not fingerprinted. Build-fingerprinted assets or a verified manifest are required before precaching.
- Current CSP has `default-src 'none'`, `script-src 'self'`, etc., but no explicit `worker-src`, `manifest-src` or `connect-src`. Browser worker registration/manifest loading and controlled session revalidation need narrowly scoped `'self'` directives in a **later reviewed integration change**, never silent loosening in this branch.
- `internal/web/h_auth.go` logout revokes the server session, clears cookies/CSRF and redirects to login. It has no browser-tab coordination or BFCache privacy mechanism.
- `internal/web/templates/base.html` already has semantic navigation, skip link, visible POST/CSRF sign-out and no-JS forms. Extend those patterns rather than replacing them with an SPA.
- `internal/web/static/app.js` is progressive enhancement for focus and session expiry. It does not implement a PWA or back/forward-cache privacy guard.

## Implementation plan covering the full proposed scope

### A. Self-contained packaging and contracts

Create a synthetic standalone test/demo server inside this nested module, with module-owned assets, no production imports and no auto-registration. Export a small Go asset/manifest/worker handler or bundle descriptor that a later adapter can mount explicitly. Validate origin, scope, start URL and base path; prevent origins outside configured HTTPS or explicitly allowed loopback development. Render a manifest with stable identity, name, start URL, scope, display mode and original 192/512px icons. Include an accessible install-help page; do not promise browsers expose an install prompt.

Use exact Workbox versions plus lockfile and reproducible local bundling; never CDN/importScripts remote fetches. Prefer package use to code-copy maintenance. Add build integrity checks ensuring every precache entry is a fingerprinted, module/public-only asset, not a wildcard matched application output.

### B. Strict cache and offline policy

Construct a Workbox precache controller with explicit asset list and custom response plugin. Require same-origin public asset path, GET, approved request destination, no Authorization header, no query/fragment, no navigation, no response redirect, status 200 and approved static MIME. Refuse `Cache-Control: private` or `no-store`; require explicitly public response policy. During precache fetches omit credentials so server cookies cannot customize the response. Validate the received bytes against build-generated integrity metadata. Never cache authenticated HTML, API responses, exports, receipts or balances. Do not log rejected URLs containing secrets.

Use module-owned cache naming and prune only obsolete PWA caches under that namespace. Do not delete unrelated origin caches. No background sync, POST queue, replay plugins, data caching recipes or navigation app-shell fallback. A network-failed document navigation may return a **constant synthetic public unavailable page**, not cached member HTML; unsafe methods must remain network-only and fail without deferred action. A client must make a new explicit action after reconnect. Avoid `navigator.onLine` as proof of server reachability.

### C. Client lifecycle and privacy

Registration is optional progressive enhancement and requires explicit installation/feature configuration; no JS means ordinary online Circula flows continue. Keep cache assets free of user data so logout does not need to erase personal cache contents that should never exist.

Still handle private DOM and BFCache separately: scrub or conceal private views on pagehide/session end, coordinate logout across tabs, and on pageshow/resume require a successful authorized server revalidation before redisplaying private content. Design this with the future auth adapter; do not claim Cache-Control alone solves BFCache across browsers. A reload/revalidation failure must leave a generic unavailable/sign-in view. Do not store credentials or HTML in localStorage/sessionStorage/IndexedDB. Handle service-worker update without losing an in-progress trade form or automatically resubmitting it.

### D. UX, help and future integration artifacts

Provide semantic Find/Trade/Balance/Inbox links with screen-reader labels, visible focus, safe-area CSS, reduced-motion support and narrow-width reflow. Keep disabled/offline state messaging readable and explain that no trade was sent. Include installation help for browsers without a programmatic install flow. Add a future integration contract listing asset routes, CSP directives, no-store rules and logout/revalidation hooks without modifying the real server. No schema migration or authoritative DB table is needed.

### E. Acceptance tests and independent review

Automated standalone tests must cover exact origin/base-path boundaries; manifest paths; fingerprints/integrity; MIME/redirect/private/no-store rejection; query-token rejection; install/update and obsolete cache cleanup; no cache access for API/navigation/POST; offline unsafe-method failure; no replay after reconnect; public fallback contents; logout across tabs; BFCache masking/revalidation; CSP and local-only worker bundle; no-JS online navigation; keyboard and 320px/400% reflow. Inspect actual browser CacheStorage contents, not only mocks. Synthetic accounts A/B should never see each other's previous DOM after switch/back/offline/reopen.

Run Go tests/race/vet for the module, Node unit tests for policy, browser integration tests for real worker lifecycle, dependency/advisory scanning and source/license audit. Browser tests on a synthetic server do not establish production adapter safety.

Remaining manual/platform gates: Android Chromium install/reopen; iOS Safari Add to Home Screen and resume; supported desktop browsers; keyboard and screen-reader testing; 200–400% zoom; shared-device account switch and OS-app-switcher behavior. Later core integration requires full Circula session/CSRF/tenant/ledger regression and the owner's single-feature selection. Mark these **pending** until actual evidence exists.

## Completion criteria for the next agents

Implementation agent: build/adapt the standalone feature as fully as supported by the available runtime, retain notices and record concrete test results and every unmet manual/integration gate. Audit agent: independently inspect copied/dependency code boundaries, verify the bundle actually uses the selected pin, rerun negative browser tests and fix findings on this branch. Neither agent may mark the product feature complete solely because the existing Go allowlist tests pass, or enable the feature in main.
