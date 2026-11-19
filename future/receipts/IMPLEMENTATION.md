# FNET-203 isolated implementation and operations

Integration remains **HOLD**, disabled by default. This implementation changes only `future/receipts/` and has no dependency on another future feature or Circula internals. The newer RECEIPT-SPEC-SNAPSHOT.md and RECEIPT-ENGINEERING-SNAPSHOT.md govern: TXT/JSON baseline; HTML optional and provided; PDF optional and deferred. Baseline receipt files are ephemeral. No DB, migration, background job, automatic download, email, tracking, public lookup or mandatory external service is added.

## Read boundary and versioned content

The host supplies an `Actor` established by existing authentication and a `Reader`. Its single `ReadReceipt` method must perform current membership/field access checks and the immutable completed trade/authorized supplement read inside the existing service/store read transaction. It must deliberately permit only current parties or specifically scoped administrators. Being a dispute reviewer or having generic transaction-history access must not grant receipt access. Requesters never supply party/amount/status/entry facts. Denied and missing records share `ErrUnavailable`.

The v2 DTO contains network-scoped party references, network/transaction references, minor-unit integer amount, precision/unit, historical completed status/time, exactly two distinct immutable entry references, source revision, explanation and an optional later status supplement. The explicit schema is separate from root API/DB versions. v1 core rendering remains available for compatibility, with its amount bound corrected to the current core maximum `1<<50`. Amounts and revisions are limited to `1<<50` before canonicalization; there is no float conversion or precision loss. New sources requiring larger revisions need a separately versioned string contract rather than silent rounding.

The optional supplement is copied from authorized reader state, timestamped, and references an independent reversal/adjustment. It does not mutate previously returned documents or original entries. Dispute text and arbitrary history never enter the allowlisted DTO. A receipt represents recorded completion, not current settlement status. Already downloaded files cannot be recalled.

## Optional construction, CLI and HTTP

A zero-value `Service` is disabled. A future host explicitly sets `Enabled`, `NetworkEnabled`, and `Reader`. `NetworkEnabled` must use approved configuration. The module does not register anything. `Receipt` requires `DownloadRequest.Explicit=true`; `Download` supports txt/json/html. The only service dependency can read receipt snapshots; no trade mutation interface exists. An optional audit hook receives only fixed outcome labels. Audit failure affects the download only.

`RunCLI` is a function for later linkage into the existing authenticated CLI:

```
receipt --transaction <reference> --format txt --request-receipt
```

It uses the same service and authenticated actor; it does not implement a second login, accept snapshot files as authoritative facts, or obtain tokens. Help is available with `--help`.

The optional `Handler` is an unregistered adapter. The host implements `Host.Authorize`, `Host.RateLimit` and `Host.CSRFToken` with current Circula policy. Authorization must validate session/API authentication, active network membership and CSRF on cookie-authenticated POST. Bodies are limited before this hook can read them. GET renders only a choice form and never reads transaction facts. POST accepts JSON or ordinary form data; unknown client facts, duplicate JSON keys, unsupported formats and trailing data are rejected. Both pathways work without JavaScript. Authentication alone does not replace the reader's transaction-level reauthorization.

Proposed module-owned operation: GET `/receipt?reference=<opaque reference>` renders the opt-in form; POST the same host-selected route downloads. JSON POST: `{"reference":"opaque-reference","format":"txt","explicit":true}`. `X-Network`, if supplied, must agree with authenticated context. Form additionally carries the host CSRF token. No production route or root OpenAPI document is changed; the future integration must align this seam to `/api/v1` conventions.

Responses use private/no-store, nosniff, no-referrer and a restrictive CSP. Filenames are constant `circula-receipt.<format>`, with no party or transaction information. Invalid format/body uses 400, unsupported type 415, unavailable/denied/missing 404, method 405, rate limit 429. No sensitive contents are logged by this module. There are no remote assets, fonts, script, service workers or browser storage.

## Canonicalization and independent modules

The isolated candidate canonical serializer pins `github.com/gowebpki/jcs` to `7d24017834028a80af75c290d5b122d6346dc76b`. Its Apache-2.0 license is in `third_party/jcs-LICENSE`. No source was transplanted or modified; runtime use is its exported `Transform` implementation. It has only standard-library runtime dependencies. Tests added here are original, not copied upstream tests. The local `THIRD-PARTY.md` records provenance.

`Document.Canonical`, `Digest` and bounded offline `DecodeDocument` operate only on the explicit DTO. JSON decoding rejects duplicates, invalid Unicode, unknown fields, unsupported schema, invalid integer bounds and trailing data. A SHA-256 digest alone is not a signature, authenticity proof or anonymity protection. Optional certificate behavior is independently described in CANDIDATE-ADR-001.md and must not become a dependency of ordinary downloads.

## Disable, retention, backup and recovery

Do not register the adapter in Circula now. At a future separately selected integration, keep platform and network enablement off until accepted. Disable by setting `Enabled=false` and removing this adapter's host registration; no source trade, ledger entry or history is removed. Certificate signing must be independently disableable. Renderer failures do not participate in trade posting or acceptance.

No module data exists to migrate, back up or restore. Existing core transaction and audit retention policies remain authoritative. Restart/recovery reuses the host reader and recomputes a fresh projection. Source backup/restore conformance is a later actual core adapter gate. Tests demonstrate deterministic output after service recreation and current access revocation; they do not pretend an in-memory fixture proves Circula DB recovery. No issuance metadata is persisted without a separately reviewed purpose/retention policy.

## Completion matrix

| Requirement | Isolated evidence | Remaining gate |
|---|---|---|
| Explicit optional action, default disabled | Spy reader tests; zero-value enablement; CLI explicit flag; choice form | Real core registration remains prohibited |
| Read-only authoritative allowlist | One narrow reader method; projection bounds/status/entry/reference tests; no mutation dependency | Qualify actual service/store adapter and ledger read transaction |
| TXT/JSON/print HTML, no JS | Byte-stable exports, ordinary form/JSON handler, escaping tests | Keyboard/screen-reader/Tor browser human acceptance |
| Fresh tenant/member access | Reader per download, host auth/CSRF/rate policy hooks, denied/missing equivalence | Host hook conformance against live core |
| Privacy/cache/log controls | Fixed audit labels, no retained files, constant filenames, CSP/cache headers, strict parser | Existing audit policy and operator acceptance |
| Reversals and immutability | Separate copied timestamped supplement; earlier bytes unchanged | Real authoritative entry/reference parity |
| Restart/concurrency | Parallel read-only downloads, service reconstruction, revocation test | Core restart/backup qualification |
| Canonicalization/digest | Exact 2^50 vectors, strict parser, Unicode/duplicate/size/unknown fuzz | External interoperability review |
| Optional certificate | Reviewed candidate standard JWS EdDSA issuer/offline verifier; separate HTTP/CLI; strict local public trust; lifecycle/rotation/revocation/adversarial tests | Independent audit fixes recorded in AUDIT.md; external interoperability and production ADR/key operations approval |
| Release | No root changes, off by default, HOLD | Owner single-feature selection, full core regression/security/accessibility gates |

No passing module test advances core Circula release status. The feasible standalone candidate is implemented; this feature remains unqualified for integration/release while real adapters, external interoperability, production approval and human acceptance gates remain open.

## Independent audit continuation

AUDIT.md records the fresh implementation review and fixes. Certificate release now reauthorizes the exact receipt projection after the opaque signing operation, compares its canonical digest, rechecks enablement/key eligibility and honors callback cancellation. This second read does not create a second fixed-label audit event. Host adapters must provide fresh policy decisions and define atomicity/deadlines across their own read/key boundaries.
