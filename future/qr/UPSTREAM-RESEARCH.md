# FUX-102 upstream research and adaptation handoff

Research date: 2026-11-16. Phase: **research complete; implementation and independent review pending**. Integration remains **HOLD**. This document does not mark QR trading complete and does not change Circula's release status.

## Decision

Reuse a pinned QR encoder and locally bundled browser scanner. Retain Circula's own authorization, idempotency and SQLite transaction boundaries for one-use trade requests. No general QR library can supply those ledger-specific guarantees. Prefer importing the encoder module over copying its Reed–Solomon/QR algorithms. Do not import an OAuth framework merely to obtain token invalidation.

Selected encoder: `github.com/yeqown/go-qrcode/v2`, exact source commit `d822f086bffb0d37d6fda9191551a6fa91ed9576`. Selected scanner candidate: Nimiq `qr-scanner` version `1.4.2`, commit `abcfe1bce2703721408d8ce7ebde94a359998506`, subject to browser/CSP qualification below. The scanner has materially older maintenance evidence, so its suitability is conditional, not a claim of active maintenance.

## Search and directly verified evidence

GitHub repository searches for `go-qrcode in:name` and `qr-scanner in:name` returned the encoder candidates and Nimiq scanner. Their repository responses reported the three as not archived. Exact commits, repository LICENSE files, dependency manifests, API source and README files were then read from local clones. Fosite was also inspected specifically for the durable single-use pattern. Dates below are observed last-commit timestamps in the fetched repositories, not promises about support or a comprehensive vulnerability assessment.

| Candidate | Exact inspected source | License and maintenance evidence | Reusable surface / disposition |
|---|---|---|---|
| yeqown/go-qrcode | [`d822f086bffb0d37d6fda9191551a6fa91ed9576`](https://github.com/yeqown/go-qrcode/tree/d822f086bffb0d37d6fda9191551a6fa91ed9576), last commit 2026-08-06 | [MIT](https://github.com/yeqown/go-qrcode/blob/d822f086bffb0d37d6fda9191551a6fa91ed9576/LICENSE), copyright 2018 yeqown; Go 1.19 module, tests, examples, dependency-update commit | **Selected** encoder. `NewWith`, encoding/ECC options, `QRCode.Save(Writer)`, `Matrix.Iterate`, width/height. Original small PNG writer can avoid importing its more elaborate image writer. |
| skip2/go-qrcode | [`da1b6568686e89143e94f980a98bc2dbd5537f13`](https://github.com/skip2/go-qrcode/tree/da1b6568686e89143e94f980a98bc2dbd5537f13), last commit 2020-06-17 | [MIT](https://github.com/skip2/go-qrcode/blob/da1b6568686e89143e94f980a98bc2dbd5537f13/LICENSE), copyright 2014 Tom Harwood; compact implementation with tests, but old last-change evidence | **Fallback only**. `Encode(content, level, size)` and `QRCode.PNG(size)` are simpler, but newer maintenance evidence favors yeqown. Do not claim skip2 is actively maintained. |
| nimiq/qr-scanner | [`abcfe1bce2703721408d8ce7ebde94a359998506`](https://github.com/nimiq/qr-scanner/tree/abcfe1bce2703721408d8ce7ebde94a359998506), package 1.4.2; last commit 2022-11-22 | [MIT](https://github.com/nimiq/qr-scanner/blob/abcfe1bce2703721408d8ce7ebde94a359998506/LICENSE), copyright 2017 Nimiq, danimoh. Has TS source, typed API, local worker fallback, camera and image scanning. Old activity is a qualification risk. | **Selected conditional scanner candidate**. Local `qr-scanner.min.js`, `qr-scanner-worker.min.js`, `new QrScanner(video, callback, options)`, `start`, `stop`, `destroy`, `scanImage`. Requires transitive Apache notice below. |
| ory/fosite | [`a5f0b09bf31c17297b25637bb3fec2ff7a55b159`](https://github.com/ory/fosite/tree/a5f0b09bf31c17297b25637bb3fec2ff7a55b159), last commit 2025-07-03 | [Apache-2.0](https://github.com/ory/fosite/blob/a5f0b09bf31c17297b25637bb3fec2ff7a55b159/LICENSE); substantial OAuth library and transaction/error-path tests | **Pattern reference only; do not import/copy whole framework**. Invalidate one-time code and create resulting sessions inside the same transactional storage boundary. Its OAuth storage, error and requester types do not fit Circula's mutual-credit service. |

Maturity here means established implementations with source/tests and usable narrow APIs. No popularity number is substituted for security review. Advisory scanning, a full dependency license inventory, and execution of the pinned upstream tests remain implementation-phase tasks.

## Critical transitive scanner licensing detail

Nimiq's worker includes `jsqr-es6` code; MIT for the wrapper is not the whole bundle's license inventory.

- [`src/worker.ts`](https://github.com/nimiq/qr-scanner/blob/abcfe1bce2703721408d8ce7ebde94a359998506/src/worker.ts) imports `jsqr-es6/dist/jsQR.js`.
- [`yarn.lock`](https://github.com/nimiq/qr-scanner/blob/abcfe1bce2703721408d8ce7ebde94a359998506/yarn.lock) resolves `jsqr-es6` **1.4.0-1**, tarball SHA-1 `e72d94a0ed794c6f19d87682ac89fab2a0f8496b`.
- The [package registry metadata](https://registry.npmjs.org/jsqr-es6/1.4.0-1) reports source `danimoh/jsQR`, git commit `42dbfd55f35db625119e8f33247b6e9d073902ec`, license **Apache-2.0**, and integrity `sha512-LPWZJLI+3LLOy9k3/s/MeXlkfNOs3bYBX5O+fp4N0XuxbgO8H7Uc/nYZeNwo13nSZXRW9xWFKmZdy9591+PyAg==`.
- The actual [published tarball](https://registry.npmjs.org/jsqr-es6/-/jsqr-es6-1.4.0-1.tgz) was fetched and its `package/LICENSE` read; it contains the Apache 2.0 text. The tar listing did not contain a separate NOTICE file. Preserve relevant source copyright/attribution and the complete license when bundling; inspect upstream headers for additional notices rather than assuming there are none.
- If copying the scanner distribution, record source SHA, copied paths, SHA-256 of each copied asset, MIT license and Apache license in `third_party/` and `THIRD-PARTY-NOTICES.md`. Do not label the combined bundle “MIT only.” Do not replace Circula's undecided project license.

No upstream source is copied by this research commit.

## Exact encoder adaptation

Official inspected interfaces:

- [`qrcode.go`](https://github.com/yeqown/go-qrcode/blob/d822f086bffb0d37d6fda9191551a6fa91ed9576/qrcode.go): `NewWith` produces a code, `Save` delegates to a writer.
- [`writer.go`](https://github.com/yeqown/go-qrcode/blob/d822f086bffb0d37d6fda9191551a6fa91ed9576/writer.go): writer contract is `Write(Matrix) error` and `Close() error`.
- [`matrix.go`](https://github.com/yeqown/go-qrcode/blob/d822f086bffb0d37d6fda9191551a6fa91ed9576/matrix.go): exported matrix size and iteration APIs.
- [`go.mod`](https://github.com/yeqown/go-qrcode/blob/d822f086bffb0d37d6fda9191551a6fa91ed9576/go.mod): core dependencies include `yeqown/reedsolomon v1.0.0` and `x/text v0.22.0`; review transitive licenses and advisories. The separate standard writer has a larger graph (`gg`, `freetype`, `x/image`, etc.).

Add the exact core commit to this nested module only and record Go's resolved pseudo-version plus `go.sum`. Write an original bounded PNG adapter with `image/png` and integer module scaling, black modules, white background, and four-module quiet zone. No logos, gradients or remote resources. Keep rendering independent of persistence and HTTP. Limit URL input, output dimensions and allocation before encoding; do not accept arbitrary file paths. Encode only a successful trusted-origin URL construction, never client-supplied credential/payment URLs. Default correction level should be explicit and documented; test at least medium and high.

## Scanner adaptation

Use [upstream README API documentation](https://github.com/nimiq/qr-scanner/blob/abcfe1bce2703721408d8ce7ebde94a359998506/README.md). Use the detailed result API, not the deprecated bare-string overload. Bundle all executable assets locally; do not load a CDN or send images to a service.

- A user explicitly starts the camera. On a result, stop scanning, bound input, and submit the scanned text to the same server parser as manual entry.
- Always stop/destroy on leave, hidden screen, logout, permission failure and successful result. Do not store frames or token strings in local/session storage or analytics.
- Decode success is only “code read,” never trade success. Existing server-authorized confirmation shows network, parties, fixed amount and available-credit estimate; submission then creates a proposal, with completed state only after ordinary acceptance.
- Clipboard paste and a normal HTML manual-code form remain available with no JavaScript. Camera controls are progressive enhancement.
- The inspected minified worker constructs a Blob-backed Worker, so CSP must be tested with the actual bundle. Do not silently loosen application CSP or add `unsafe-eval`. If a narrowly scoped local worker adapter cannot satisfy the app policy, keep scanner qualification open and use manual entry while selecting another scanner; do not mark the camera acceptance complete.

## Existing Circula seams and the atomicity trap

Inspected at the feature baseline `2eacf7718f5bb3da3be5a288024af879076a9597`:

- `internal/service/transactions.go`: public `ProposeTransaction` calls `s.update`, authenticates with `s.open`, requires active membership and accepted terms, then invokes `idempotent` around private `s.propose(tx, principal, input)`. The latter performs current network/party/limit checks and creates only a proposed transaction.
- `internal/service/idempotency.go`: scoped key and fingerprint record is stored in that same transaction; identical replay loads the prior transaction, differing input fails.
- `internal/store/sqlite.go`: `SQLite.Update` uses `BEGIN IMMEDIATE`, a context-aware in-process writer gate and rollback on callback error.
- `internal/store/iface.go`: store transaction capabilities are explicit. They do not currently include QR aliases/requests.
- Existing nested `future/qr/intent.go` has random token generation, canonical hashing and URL validation but no durable request store or claim authorization.

**Do not** consume a QR token in one transaction then call public `ProposeTransaction` in another. **Do not** call public `ProposeTransaction` from a QR database callback: it starts a separate update rather than sharing the current write unit. Either can burn tokens on failed proposals, duplicate proposals, or deadlock a writer gate. Fosite's [transactional invalidation path](https://github.com/ory/fosite/blob/a5f0b09bf31c17297b25637bb3fec2ff7a55b159/handler/oauth2/flow_authorize_code_token.go) reinforces the same-boundary requirement, but its storage abstraction cannot implement Circula's posting rules.

Develop a transaction-scoped QR repository/claim contract with an injected **same-write-unit** proposal operation, never an arbitrary post-commit callback. A final Circula service adapter must call private `s.propose` under the existing `s.update` after authenticating fresh state. Keep that adapter as an unregistered, isolated integration artifact/test harness under `future/qr/` until individual integration is authorized. The full production two-entry ledger acceptance cannot be certified without running the actual eventual adapter.

## Full-spec implementation sequence for the fresh phase-2 agent

1. **Encoding and scanner assets:** pin imports/assets, retain notices, add deterministic PNG tests and independent decode checks, implement optional scanner and no-JS form assets. Keep all under `future/qr/`.
2. **Aliases:** add cryptographically random network-scoped alias generation, hashed storage, owner-only rotation/revocation, unique constraints and rate/quantity limits. Alias carries no amount. Retain revoked hashes as required to prevent old QR reuse; define cleanup separately.
3. **Request lifecycle:** immutable fixed positive integer amount and bounded note from an authorized recipient; hashed token only at rest; explicit creation, read, expiry and revocation. Scope every lookup by network. Do not expose whether a foreign token exists. Return raw tokens only at explicit creation/download, never logs.
4. **SQLite persistence component:** ship additive schema as module-owned SQL, not root migrations. Test against a real temporary file, close/reopen, and backup/restore. Use uniqueness on network+hash and claimed transaction reference; separate raw token from persisted records. Respect existing one-process-per-file operational model.
5. **Atomic claim service:** in one write transaction re-check active membership, recipient, expiry and revocation; bind the authenticated payer, request fingerprint and stable intent idempotency key; create the ordinary proposal through the same-write-unit capability; record claimed transaction. Same payer+same intent replay returns the prior result; competing payer or changed payload fails. Errors rollback both writes. A claimed request whose transaction later completes remains claimed.
6. **Contracts and presentation:** module-local proposed OpenAPI fragment, RFC9457 errors, CSRF/no-store/header/body-limit requirements and accessible template set. Detail routes separately from registration. Confirmation does not trust QR-supplied amount or user identity; it re-reads the server request and current available-credit estimate. No autoaccept, auto-post or offline replay.
7. **CLI/help:** explicit create/show/revoke/render interfaces and documented token-handling policy; diagnostics redact hashes/tokens where unnecessary. Include manual-entry, print and camera instructions and pending/completed distinctions.
8. **Audit evidence:** keep a requirement-to-code/test table. Have a fresh phase-3 agent independently inspect licenses, data exposure, atomicity, parser fuzz regressions, image decoding and no-JS flows. Record unresolved manual/browser or core-adapter gates honestly.

## Required test matrix

- Parser fuzzing: trusted host/scheme only; userinfo, paths, encoded separators, whitespace, query/fragment, wrong version, overlong input and noncanonical base64; never panic.
- Encode/decode: independent decoder recovers exact generated alias/request URL; white quiet zone and high contrast; bounded dimensions; printable and screen samples.
- Alias authorization: own-member rotate only, old alias revoked, cross-network aliases fail, no amount embedded.
- Atomic lifecycle: two payers racing yield one proposal; same actor retry returns same proposal; changed amount/note/key rejected; request amount immutable; proposal failure rolls back claim; DB error after proposal rolls back both; restart after commit returns original proposal.
- Current-state checks: expiry at exact boundary, revoke concurrent with claim, suspended/departed payer/payee, changed terms/credit limit, inactive network and insufficient capacity fail through service rules.
- Privacy: no raw token in persistent DB/log/export; response/cache/referrer policy; no camera frames persist; no token in telemetry or external URL. Explicitly test filename/download handling.
- UI: manual form without JS, keyboard/screen-reader errors, camera denied/unavailable, secure-context requirement, scan stop/cleanup, iOS Safari and Android Chromium, local CSP-compatible worker; scan never auto-submits a payment.
- Final integration: actual Circula proposal and acceptance produce exactly two sum-zero ledger entries, one completed transaction and normal audit/idempotency records. Run root regression, race, migration/restore, CSRF and tenant matrix only when the integration adapter is built against the qualified core.

## Reproducibility

Temporary inspected source clones are `/tmp/circula-qr-upstream-yeqown`, `/tmp/circula-qr-upstream-encoder`, `/tmp/circula-qr-upstream-scanner`, and `/tmp/circula-qr-upstream-fosite`. Their commit IDs above are authoritative; these paths are disposable. No application tests were run for this research-only change. Phase 2 must fetch/verify pins afresh if needed rather than depending on temporary directories.
