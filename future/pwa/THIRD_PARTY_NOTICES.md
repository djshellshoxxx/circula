# Third-party provenance

## Workbox runtime reuse

- Project: https://github.com/GoogleChrome/workbox
- Release: 7.4.1; exact reviewed release commit `62b9d8ba8eb3c1a2ab8aac9d84c90cda7865d6a3`.
- Exact npm pins: workbox-core, workbox-precaching, transitive workbox-routing and workbox-strategies, all 7.4.1. Tarball locations and SHA-512 integrity are in package-lock.json.
- License: MIT-style, Copyright 2018 Google LLC. Full text: licenses/WORKBOX-MIT.txt. Source license comments remain in the bundled worker.
- Reused APIs: PrecacheController, PrecacheStrategy and their lifecycle/plugin machinery. This is a bundled dependency, not a fork of upstream source.
- Local adaptations live in src/worker.mjs: explicit public manifest, credential omission, SHA-256 verification, strict response admission, exact URL lookup, network-only private/unsafe requests, constant offline navigation fallback and namespace-limited cache cleanup. No background-sync module is adopted.
- Compiled distributable: public/worker.js. Its hash and the other asset hashes are recorded in BUILD-SHA256SUMS. All browser assets are local; no CDN/importScripts remote URLs are used.

## Build tooling

- esbuild 0.25.12, exact npm pin and platform-package integrity in package-lock.json.
- Project: https://github.com/evanw/esbuild/tree/v0.25.12
- MIT license, copyright Evan Wallace; full text: licenses/ESBUILD-MIT.txt.
- Build-only. Its binary/node_modules are not committed or shipped with Go assets.

- Playwright 1.62.1, Apache-2.0, test-only dependency. Project: https://github.com/microsoft/playwright/tree/v1.62.1. Exact npm integrity is in the lockfile; full license in licenses/PLAYWRIGHT-APACHE-2.0.txt. Browser binaries are not committed.

## Original material and reproducibility

The Go handler, privacy policy, client lifecycle, CSS, tests and simple C-mark PNG icons are original work for this module. No GOV.UK branding, fonts or artwork were copied.

Build: `npm ci --ignore-scripts && npm run build`. esbuild's pinned platform package supplies the binary. Two consecutive builds produced identical hashes in validation. This verifies deterministic output for the tested Node/platform/lockfile, not every possible platform.

No root Circula license was selected or changed. Preserve these notices with redistributed bundled assets. Future distribution review remains a release gate.
