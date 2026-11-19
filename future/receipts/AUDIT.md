# Independent receipt/certificate audit — 2026-11-16

Reviewed baseline `26c359e651ac7c59240f56d285cb9a8063850212` and the fixes committed with this record. Reviewer was a fresh agent, separate from the implementation and candidate-design review. Scope is only this isolated `future/receipts/` module. Integration remains **HOLD**. This record is not production cryptographic approval or an external professional audit.

## Findings and fixes

1. **Certificate authorization could expire during opaque signing.** The issuer rechecked its signing key after the opaque call but did not recheck current transaction access or receipt/certificate network enablement. An actor whose membership was revoked during that call could still receive the private signed snapshot. The issuer now performs a second fresh authorized receipt projection without duplicating the original fixed-label audit event, requires its canonical digest to match the signed projection, and rechecks both enablement policies before release. A changed authorized supplement also requires retry. Regression tests revoke membership, independently disable receipt/certificate availability, and change the projection inside the opaque signing callback.
2. **Callback cancellation could be ignored.** The ordinary receipt path could continue after a Reader or Audit callback canceled the context. The service now checks cancellation after policy/read/audit boundaries and before writing; issuer checks it after key callbacks; offline verification checks it before reading and after local key lookup. Diagnostics also refuse ready status after cancellation. Tests cover cancellation within callbacks and a canceled verifier whose reader must never be invoked.
3. **Malformed DTO decode could return partial content with an error.** The offline receipt decoder now returns an empty DTO on decode failure. Callers still must check errors; no malformed DTO is qualified by a parse failure.

All findings above are fixed in the reviewed branch. No remaining blocking defect was identified in the reviewed standalone candidate during this bounded audit. That does not close the external qualification gates below.

## Requirement-to-evidence matrix

| Requirement | Review and reproduced evidence | Disposition |
|---|---|---|
| Explicit optional action; defaults disabled; ordinary receipts independent of signing | Zero-value/consent spies; separate receipt/certificate handlers and CLI; opaque failure leaves TXT path usable | Standalone tests pass |
| Current parties or explicitly scoped admins; no generic reviewer access | Narrow authoritative Reader contract; fresh actor/network/reference tests across TXT/JSON/HTML; added both-party/scoped-admin/reviewer-denial fixture; late sign reauthorization | Port behavior tested; actual core adapter pending |
| Exact allowlisted immutable historical projection | Positive minor-unit/revision cap `1<<50`, unit/precision/two distinct entry references, copied timestamped reversal supplement; no balances, dispute text, contacts, arbitrary source model or ledger writer | Standalone tests pass |
| TXT, JSON and escaped print HTML without JavaScript | Shared service/export paths, ordinary HTML forms, static escaping/private-cache/CSP/constant-filename tests; no remote assets or browser storage | Static tests pass; real browser/accessibility pending |
| Bounded unambiguous canonicalization | Pinned RFC8785 implementation; exact integer boundary, schema reconstruction, unknown/duplicate/case-variant fields, Unicode, depth16, trailing input and size rejection | Tests/fuzz pass |
| Standard JWS EdDSA and only local trusted keys | Protected alg/typ/kid allowlist, canonical base64url, fixed compact signature size, canonical signed manifest and digest, explicit expected issuer/network, exact Ed25519 public type; no jku/jwk/x5u/remote discovery; independent stdlib Ed25519 verification test | Candidate tests pass; whole external verifier pending |
| Key lifecycle and rotation | Active half-open issuance window; old retained trust; retired historical assertions; full revocation; late revocation/expiry; opaque handle signature verified with captured public key | Tests pass; production lifecycle/distribution approval pending |
| No artifact persistence, logging or trade coupling | No DB/schema, automatic download, background work, transport, automatic registration or mutation dependency; fixed audit outcome labels only; output owned by explicit requester | Source review; host audit policy pending |
| HTTP privacy and safety | Authentication/CSRF/rate hooks, tenant agreement, body cap before host hook, strict content type/form/JSON parsing, uniform unavailable response, private/no-store/no-referrer and constant filenames | Adapter tests pass; live host conformance pending |
| Restart/concurrency | Stateless reconstruction and concurrent read-only downloads; immutable offline key map with copied public slices | Module evidence passes; actual source DB restore pending |
| Licenses/dependencies | Exact JCS/go-jose pins and checksums, retained Apache licenses and documented upstream paths; `go mod verify` reproduced | Pass for module dependencies; release packaging review remains |

## Reproduced validation

Go 1.26.11 linux/amd64, `/workspace/scratch/0afb7942ff6b/toolchain/go/bin/go`:

- `GOMAXPROCS=2 go test -race -cover -timeout 90s ./...`: pass, 83.6% statements.
- `go vet ./...`: pass.
- `go mod verify`: all modules verified.
- `GOMAXPROCS=2 go test -run '^$' -fuzz '^FuzzVerifyCertificate$' -fuzztime=5s -parallel=1 -timeout 45s`: pass, 15,890 executions; 13 additional interesting inputs. This is a bounded fuzz run, not exhaustive assurance.

Earlier implementation vulnerability scan results remain in VALIDATION.md; this audit adds no dependency and does not claim to rerun that network-backed scan. No production keys or private user records were used.

## Remaining qualification and limits

- Actual Circula authentication/read-transaction adapter, ledger-reference parity, protected-key provider and current production contracts remain unimplemented integration work, expressly outside this HOLD branch.
- The trusted Reader must use current membership and authorized immutable facts in the existing service/store read transaction. Final receipt and key checks are separate trusted boundaries; they cannot provide one globally atomic transaction across arbitrary external providers. Hosts must bound callback latency, honor deadlines and define their revocation linearization policy. An opaque operation cannot be interrupted by this interface mid-call.
- Retirement relies on a signed asserted issuance time. It does not contain compromise; a compromised key requires full revocation. Authentic offline public-key distribution and support/retention policy require production approval.
- External whole-verifier interoperability, real browser/Tor, keyboard, screen-reader/WCAG and operational key/backup/incident acceptance were not performed here.
- Root/full-ledger regression, production security approval and the owner's single-feature selection remain separate gates. Nothing is merged, registered, enabled, released or published by this audit.
