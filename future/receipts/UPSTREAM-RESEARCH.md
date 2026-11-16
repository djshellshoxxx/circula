# FNET-203 upstream research and implementation handoff

Date: 2026-11-16. Phase: research only. Integration remains **HOLD**. The owner's current instruction authorizes isolated development and upstream reuse, not merging or enabling this feature. The two newer receipt specification/engineering snapshots govern the scope; no signature design in this report is a production-approved cryptographic ADR.

## Decision

Reuse maintained standards implementations as pinned dependencies: **gowebpki/jcs** for canonicalization and **go-jose/go-jose/v4** for JWS signing/verification. Keep receipt projection, consent, authorization, key policy and operational semantics native to Circula. No mature receipt library can safely supply those project-specific rules. Reusing entire JCS/JWS implementations is more code reuse with less maintenance than copying crypto files into this module.

The dependency selection is suitable for an isolated candidate, not a claim of audit completion. The maintained JCS commit includes recent parser fixes and must be independently exercised against malicious inputs. Do not select its older release just because a stable tag is more convenient.

## Verified candidate comparison

| Candidate | Exact revision and license | Maintenance/maturity evidence inspected | Decision and reusable surface |
|---|---|---|---|
| [gowebpki/jcs](https://github.com/gowebpki/jcs) | `7d24017834028a80af75c290d5b122d6346dc76b`; Apache-2.0; Go 1.21 | HEAD commit 2026-11-10, recent invalid-input regression changes, Go CI/race/fuzz workflow, derived with permission from Anders Rundgren's reference implementation. Latest listed tag is v1.0.2, but pinned HEAD contains newer validation work. | Select `jcs.Transform([]byte)`. Use as dependency for immutable schema canonical bytes; borrow upstream test vectors with notices if copied. Do not fork its number/Unicode parser casually. |
| [go-jose/go-jose](https://github.com/go-jose/go-jose) | v4.1.9 = `3c8a86c0b60170109f319d96b05da1e77cc8dbb4`; Apache-2.0; Go 1.24 | Release tag and HEAD match, commit 2026-11-05; README identifies v4 as stable, v5 forthcoming; explicit SECURITY.md, threat model, extensive JWS/JWK tests/fuzz corpus. | Select standard JWS signing/verifying with **only EdDSA** allowed. Use `NewSigner`, `SignerOptions`, `ParseSignedCompact` and `Verify`; use its JWK support only for public trusted keys if adopted. No JWT claims, encryption, embedded remote-key discovery or all-algorithm parser. |
| [cyberphone/json-canonicalization](https://github.com/cyberphone/json-canonicalization) | `32a48195b9b20fe67b1795e39e4509f14d860b45`; Apache-2.0 | Commit 2026-10-11 rejecting invalid Unicode; established multi-language reference and testdata tree; canonicalization concept authored before its RFC publication. | Select as independent interoperability/test-vector reference, not a second runtime parser. Avoid maintaining two copies of substantially related Go code. |

Evidence was gathered from GitHub repository searches, exact Git clones, `git ls-remote --tags`, pinned source/license/go.mod/README files, and module download metadata. GitHub star counts were not used as a security score. No third-party source code was copied in this research phase.

### Reproducible pins

`go mod download -json` resolved and downloaded both exact candidates without modifying this feature's go.mod:

- `github.com/gowebpki/jcs v1.0.3-0.20261110202536-7d2401783402`, source hash `7d24017834028a80af75c290d5b122d6346dc76b`, module sum `h1:jYkkqXsMx6hQ7h1KEmBA5qreiW7jtiqUqJdka31XZM0=`.
- `github.com/go-jose/go-jose/v4 v4.1.9`, source hash `3c8a86c0b60170109f319d96b05da1e77cc8dbb4`, module sum `h1:pKfeQkig1NmgK4hJrZYXLEYURXfoJB61Larw0uIW4gY=`.

The installed Go launcher is `/tmp/circula-toolchain/go/bin/go`; `GOTOOLCHAIN=go1.26.11` is available and matches the root toolchain. Both proposed dependencies fit the current nested module's Go 1.25 minimum. Record resulting exact module/checksum graph, vulnerability scan result and bundled notices when implementing.

## Pinned official sources

- [JCS LICENSE](https://github.com/gowebpki/jcs/blob/7d24017834028a80af75c290d5b122d6346dc76b/LICENSE), [README](https://github.com/gowebpki/jcs/blob/7d24017834028a80af75c290d5b122d6346dc76b/README.md), [go.mod](https://github.com/gowebpki/jcs/blob/7d24017834028a80af75c290d5b122d6346dc76b/go.mod).
- [JCS Transform/parser](https://github.com/gowebpki/jcs/blob/7d24017834028a80af75c290d5b122d6346dc76b/jcs.go), [invalid-input tests](https://github.com/gowebpki/jcs/blob/7d24017834028a80af75c290d5b122d6346dc76b/invalid_input_test.go), [CI](https://github.com/gowebpki/jcs/blob/7d24017834028a80af75c290d5b122d6346dc76b/.github/workflows/ci.yml).
- [go-jose LICENSE](https://github.com/go-jose/go-jose/blob/3c8a86c0b60170109f319d96b05da1e77cc8dbb4/LICENSE), [README and supported versions](https://github.com/go-jose/go-jose/blob/3c8a86c0b60170109f319d96b05da1e77cc8dbb4/README.md), [SECURITY](https://github.com/go-jose/go-jose/blob/3c8a86c0b60170109f319d96b05da1e77cc8dbb4/SECURITY.md).
- [JWS parse APIs](https://github.com/go-jose/go-jose/blob/3c8a86c0b60170109f319d96b05da1e77cc8dbb4/jws.go), [signing APIs](https://github.com/go-jose/go-jose/blob/3c8a86c0b60170109f319d96b05da1e77cc8dbb4/signing.go), [duplicate-key-rejecting JSON decoder](https://github.com/go-jose/go-jose/blob/3c8a86c0b60170109f319d96b05da1e77cc8dbb4/json/decode.go).
- [Reference LICENSE](https://github.com/cyberphone/json-canonicalization/blob/32a48195b9b20fe67b1795e39e4509f14d860b45/LICENSE), [reference test data](https://github.com/cyberphone/json-canonicalization/tree/32a48195b9b20fe67b1795e39e4509f14d860b45/testdata).

If source or tests are copied, retain their Apache license/copyright notices and identify changed files and source SHAs. Preserve any applicable NOTICE content discovered during packaging. This does not select or change Circula's own unresolved product license. No GPL/AGPL code is proposed.

## Critical compatibility findings

1. **Amounts:** JCS serializes IEEE-754 numbers (`strconv.ParseFloat` in its parser). The core's `internal/domain/policy.go` caps transaction amounts at `1 << 50`, safely below the exact-integer boundary; this isolated prototype currently accepts arbitrary positive int64. Add a bound consistent with the authoritative core before signing. Do not silently round a larger int64. A future decimal-string amount needs a separately versioned schema/ADR; never alter the meaning of an already issued schema. Test exact cap, cap+1, 2^53 boundaries, and MaxInt64.
2. **Authorization:** `internal/service/tx_queries.go:GetTransaction` returns a `TxView` to parties, administrators, and reviewers of an existing dispute. The module prototype is active-party-only. A future receipt reader must deliberately narrow this through the service policy, not treat every successful general-history read as receipt permission. Resolve suspended/departed access explicitly; current ordinary history allows some such reads. No automatic admin/export bypass.
3. **No whole-model serialization:** `TxView` includes entries and history; only the receipt allowlist may escape. The current DTO lacks an explicit status field and precision; settle an isolated versioned contract before signing. Never add private notes, reviewer information or balances because the reader provides them.
4. **JWS API:** v4.1.9 deprecates generic `ParseSigned`. Use `ParseSignedCompact` for an explicitly compact format and allowlist exactly `jose.EdDSA`. Parsing is not verification; obtain authoritative payload only after `Verify` succeeds. Treat `kid` as untrusted selection input until verification, never a URL.
5. **Do not invent signature plumbing:** Prefer JWS over a handwritten signature envelope. An isolated ADR may define a canonical certificate manifest containing schema, issuer/network, transaction reference, issued-at, receipt, SHA-256 digest and canonicalization identifier, signed with standard JWS EdDSA. Bind all security-relevant metadata inside the signed content/protected header. Do not mix plain Ed25519 and Ed25519ph or silently substitute formats.
6. **Strict input handling:** Bound certificate/receipt input before parsing, reject duplicate/unknown fields, invalid UTF-8/surrogates, trailing JSON, unexpected schema/algorithm, multi-signature/unprotected-header variants and extra remote-key fields. Canonicalization is not schema validation; validate the allowlisted DTO separately. Keep limits on depth, field lengths and decoded/base64 size.
7. **Trusted key distribution:** Never fetch `jku`, `x5u`, embedded keys, arbitrary URLs or a network receipt registry. A separate local trust-store input maps issuer/network + key ID to public key and explicit lifecycle metadata. Keys must not be trusted merely because they arrive in the receipt.

## Fresh implementation agent plan

Implement only under `future/receipts/`. Keep HOLD and disabled default. No changes to core service/store/routes, no production keys, no current-release changes.

| Work package | Native adaptation and upstream reuse | Acceptance evidence |
|---|---|---|
| Versioned projection | Validate authoritative snapshot with core-compatible amount cap, network-scoped opaque references, UTC completion time, precision/unit, completed status, explicit request. Keep request DTO free of facts. | TXT/JSON semantic equivalence; forbidden-field absence; controls/Unicode/overflow/invalid status; deterministic bytes. |
| Read-only service | Inject a narrow `AuthorizedCompletedTradeReader` or combined authorization-and-snapshot interface; reader must perform authorization and snapshot under the core's existing read transaction at later integration. Optional issuer is a separate dependency. | Spy adapter proves no reader/issuer call on absent consent or disabled feature, no mutation interface reachable, context/cancellation honored, same external error for denied/missing references. |
| Canonical serializer | Pin JCS exact commit, wrap `Transform`, validate numeric/schema constraints first; SHA-256 canonical receipt bytes. Retain provenance and license. | Official vectors plus independent vectors; ordering/Unicode/exponents/duplicates fuzz; exact minor-unit tests. |
| Optional certificate issuer | Experimental ADR selects JCS + SHA-256 + JWS EdDSA; use pinned go-jose `NewSigner`, explicit protected type/key ID, documented canonical manifest. Signing failure must leave TXT/JSON available. | Separate certificate consent; wrong-key/metadata/digest tampering; unavailable signer; no key/body logging; no automatic signing. |
| Key provider and trust store | Inject protected server-side signer/key provider; separate public-only verifier store. Model active, retired and revoked keys. Do not add production key generation/persistence implicitly. | Rotation retains historic public keys; retired keys cannot issue but prior valid issuance can verify under policy; revoked/unknown outcomes distinct; future-dated/out-of-window issuance rejected. |
| Offline verifier/CLI | Strict bounded parser + `ParseSignedCompact(..., []jose.SignatureAlgorithm{jose.EdDSA})` + trusted-key lookup + `Verify` + schema/digest checks. CLI accepts local file/key set and emits reason-coded result; no network. | Unknown/revoked/retired/malformed/unsupported/invalid statuses; algorithm confusion and substituted kid; receipt snapshot semantics; resource-bound fuzz. |
| Optional HTTP adapter | Explicit constructor/registration only; inject host authentication, rate-limit and CSRF hooks, no independent roles. Request only reference/format/action; no-store headers, safe filenames. Plain HTML form/download only; omit optional browser JS for candidate. | `httptest` no-JS flows, unauthorized/nonexistent equivalence, bad content types/extra fact fields, size limits, correct content type/disposition, fail-closed absent hooks. |
| Operations and observability | No artifact retention or new DB. Low-cardinality optional counters, no private content. Write disabled/enablement, trust-distribution, rotation/retirement/compromise runbooks and data-retention statement. | No private keys in source/logs/fixtures except explicitly synthetic test keys; no receipt persistence or outbound traffic; disable affects module alone. |
| Qualification | Run tests/race/vet/fuzz and dependency audit, then fresh-agent security review. Record requirement matrix and evidence honestly. | Spec-by-spec automated results; manual keyboard/screen-reader/Tor/no-JS qualification remains explicitly pending until exercised. Core adapter/end-to-end ledger immutability remains an integration gate. |

## Provisional isolated design defaults and unresolved approval gates

A coherent candidate can use opaque party references, no artifact persistence, separate certificate request, per-network disabled default, TXT/JSON only, offline verification only, and no optional browser JavaScript. These minimize new assumptions and fit the spec's recommended baseline. Label these **candidate ADR decisions**, not final production approvals.

Production key storage, trustworthy key-set distribution, exact retirement/revocation policy and external independent security approval cannot be supplied by a borrowed library. Record their interfaces/runbooks and tested candidate behavior; do not mark the feature release-ready while these and real-core integration/manual acceptance remain open. User authorization permits continuing isolated implementation without pausing for each routine choice.

## Research completion record

Both newer specification snapshots, module code, core transaction-query visibility, and core monetary cap were inspected. Three candidates were compared; two exact dependencies are recommended and one source supplies interoperability references. Dependency pins were downloaded successfully, but no runtime dependency, application code, copied upstream source, route, key or migration was added. A new agent must implement and another new agent must audit.
