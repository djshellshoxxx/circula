# Third-party notices for future/qr (FUX-102)

Circula currently has no approved product license. Nothing below changes that; owner/license review must precede any release or redistribution (see README "Open gates"). No third-party source is copied except the two Nimiq browser assets and one test-only decoder listed in section 2. Everything else is imported as a pinned Go module (`go.mod`/`go.sum`). The `go` directive is 1.26.0 because modernc.org/sqlite v1.60.1 requires it.

## 1. Go modules linked into the library

| Module | Pin | License | Use |
|---|---|---|---|
| `github.com/yeqown/go-qrcode/v2` | `v2.3.1-0.20260806012459-d822f086bffb` = commit `d822f086bffb0d37d6fda9191551a6fa91ed9576` (2026-08-06) | MIT, Copyright (c) 2018 yeqown | QR matrix encoder only (`NewWith`, `Matrix.Bitmap`). Its image writers, fonts and `x/image` graph are NOT imported; `png.go` is an original writer using `image/png`. |
| `github.com/yeqown/reedsolomon` | `v1.0.1-0.20260308105121-c5f4bc9af094` = commit `c5f4bc9af094852b52e593a5f964647c43028c51` | MIT, Copyright (c) 2026 yeqown (`LICENSE` present at this commit) | Reed-Solomon codes for the encoder. **Why not v1.0.0:** the `v1.0.0` module that go-qrcode requires contains no LICENSE file and no license headers. The pinned later commit adds the MIT LICENSE and its Go sources are byte-identical to v1.0.0 (checked by diffing all `*.go` files). Upstream's stated terms for v1.0.0 were not independently confirmed beyond that; owner license review should confirm. |
| `golang.org/x/text` | `v0.22.0` (`3b64043c9e8fa8cd61a019df17dc729630915fa9`) | BSD-3-Clause, Copyright 2009 The Go Authors | Transitive dependency of go-qrcode. |

The upstream MIT notice text must accompany redistribution of the encoder: "Permission is hereby granted, free of charge ... The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software." (full text in each module's LICENSE file in the module cache).

## 2. Copied files

### Nimiq `qr-scanner` 1.4.2 (browser scanner), MIT

- Upstream: https://github.com/nimiq/qr-scanner, package `qr-scanner@1.4.2` (npm tarball SHA-1 `bc4fb88022a8c9be95c49527a1c8fb8724b47dc4`, integrity `sha512-kV1yQUe2FENvn59tMZW6mOVfpq9mGxGf8l6+EGaXUOd4RBOLg7tRC83OrirM5AtDvZRpdjdlXURsHreAOSPOUw==`). The research pin is source commit `abcfe1bce2703721408d8ce7ebde94a359998506`; the npm tarball carries no gitHead, so the binding between that commit and the tarball is upstream's published release, not independently verified here.
- Copyright (c) 2017 Nimiq, danimoh. License text: `web/static/vendor/qr-scanner/LICENSE`.
- Files (unmodified), SHA-256:
  - `web/static/vendor/qr-scanner/qr-scanner.min.js` `0564e4cf84b94cac80213b4f779bd215bb0876a91c19df28baf97766f4fd5484`
  - `web/static/vendor/qr-scanner/qr-scanner-worker.min.js` `f4d8445f5a15c4e5f71a8c8c062c6443f09db370b8d3fe1fbdc7fe7889630d14`
  - `web/static/vendor/qr-scanner/LICENSE` `f9e6b44d80d0eb44442c7e66af8a756709a61e4d9adf0a54b8951a92ece734e2`
- The `//# sourceMappingURL` comments point to `.map` files that are intentionally not shipped (they 404, harmless).

### Bundled transitive: `jsqr-es6` 1.4.0-1 (Apache-2.0)

- The worker bundle contains `jsqr-es6` (fork of cozmo/jsQR by danimoh, repository https://github.com/danimoh/jsQR, commit `42dbfd55f35db625119e8f33247b6e9d073902ec`). Tarball SHA-1 `e72d94a0ed794c6f19d87682ac89fab2a0f8496b`, integrity `sha512-LPWZJLI+3LLOy9k3/s/MeXlkfNOs3bYBX5O+fp4N0XuxbgO8H7Uc/nYZeNwo13nSZXRW9xWFKmZdy9591+PyAg==`.
- **The combined scanner bundle is therefore not "MIT only"**: it includes Apache-2.0 code. Full license text: `web/static/vendor/qr-scanner/LICENSE-jsqr-es6-Apache-2.0` (SHA-256 `c6596eb7be8581c18be736c846fb9173b69eccf6ef94c5135893ec56bd92ba08`). The tarball has no NOTICE file and `jsQR.js` carries no copyright header; none was removed. Copyright holders per package metadata: Cosmo Wolfe, Jefff Nelson, danimoh.
- Test-only copy (never served, not embedded): `testdata/jsqr-es6/jsQR.js` (SHA-256 `f3ed14f9486e011b740a12ebab9ccd536969c99495d3f22a37063875212271ec`), same package version, used by `png_test.go` to decode generated PNGs independently of the encoder. `testdata/jsqr-es6/package.json` (`{"type":"module"}`) and `testdata/decode.mjs` are original test glue.

## 3. Test-only dependency

- `github.com/mattn/go-sqlite3` `v1.14.53` (commit `93c7e0a2ec6762a38ab371755d49a2b3f4a12e67`), MIT, Copyright (c) 2014 Yasuhiro Matsumoto; it embeds the public-domain SQLite amalgamation. It is imported only from `_test.go` files and the `tools/browser-qualification` main under build tag `sqlite_cgo_test`; it is never linked into a normal build. It exercises real SQLite but does not qualify Circula's production driver.

### Test-only pure-Go driver graph (build tag `sqlite_modernc_test`)

`modernc.org/sqlite v1.60.1` is the SQLite driver Circula's core uses. It is imported only from `_test.go` files under the tag and never linked into a normal build; it is listed in `go.mod` so the tagged tests are reproducible. Licenses were read from each module's LICENSE in the module cache (heads only; full SCA is an open gate):

| Module | Pin | License |
|---|---|---|
| `modernc.org/sqlite` | v1.60.1 | BSD-3-Clause, Copyright (c) 2017 The Sqlite Authors |
| `modernc.org/libc` | v1.77.1 | BSD-3-Clause, The Libc Authors |
| `modernc.org/mathutil` | v1.7.1 | BSD-3-Clause, The mathutil Authors |
| `modernc.org/memory` | v1.12.1 | BSD-3-Clause, The Memory Authors |
| `github.com/dustin/go-humanize` | v1.0.1 | MIT |
| `github.com/google/uuid` | v1.6.0 | BSD-3-Clause, Google Inc. |
| `github.com/mattn/go-isatty` | v0.0.24 | MIT |
| `github.com/ncruces/go-strftime` | v1.0.0 | MIT |
| `github.com/remyoudompheng/bigfft` | v0.0.0-20230129092748-24d4a6f8daec | BSD-3-Clause, The Go Authors |
| `golang.org/x/sys` | v0.48.0 | BSD-3-Clause, The Go Authors |

## 4. Pattern reference and tooling that are NOT included

- `ory/fosite` (Apache-2.0): read only for the "invalidate the one-time code in the same transaction as the resulting grant" pattern. No code, types or dependency.
- `skip2/go-qrcode` (MIT): evaluated fallback, not used.
- Playwright, Chromium and `axe-core` (MPL-2.0) are used by `tools/browser-qualification/run.js` from outside the repository and are not vendored or distributed.
