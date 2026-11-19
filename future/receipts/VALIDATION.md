# Validation record — 2026-11-16

Toolchain: `/workspace/scratch/0afb7942ff6b/toolchain/go/bin/go`, Go 1.26.11 linux/amd64. These commands ran from the standalone `future/receipts/` module. No root application files, database, production routes, signing key or release configuration changed.

```
GOMAXPROCS=2 go test -race -cover ./...
PASS (84.1% statements)
go vet ./...
PASS (no diagnostics)
go mod verify
all modules verified
```

Added low-cardinality optional issuer health checks without key identifiers, receipt data or public registration.

The suite exercises deterministic TXT/JSON/HTML, source-minor-unit and revision exactness, fresh host/transaction authorization seams, denied/missing equivalence, separate receipt/certificate consent/default-off, no-JS form/API and CLI adapters, body/content-type/CSRF/rate hooks, output escaping/private cache headers, no automatic Reader invocation, parallel read-only requests, stateless service reconstruction, immutable reversal supplements, bounded strict JCS and local public-key parsing. It includes standard independent `crypto/ed25519` verification of the generated JWS signing input.

Certificate tests cover rotation with retained historic trust, exact Ed25519 public type, opaque handle rebinding, active/retired/revoked policy, half-open issuance window, key revocation/expiry during signing, wrong/unknown keys, malformed/oversized/deep/duplicate/unknown JSON, noncanonical base64url, extra array fields, remote-key headers, algorithm confusion, wrong digest, future/early issuance, expected scope, signer unavailability preserving basic receipts, and context cancellation before/after opaque and lookup boundaries. Private test keys are generated synthetically in memory, never persisted or production-configured.

Short fuzz runs actually performed:

- FuzzCreateReceipt: five-second run, 25,062 executions, pass.
- FuzzDecodeDocument: initial five-second run, five executions, pass; final current-code run: 54,378 executions, pass.
- FuzzVerifyCertificate: initial five-second run, 2,019 executions, pass; final current-code run: five executions, pass.

Short fuzz success is bounded evidence, not exhaustive resource/security qualification. System load limited throughput in some runs.

Dependency scan: `govulncheck ./...` using installed v1.1.4 tool, Go 1.26.11 PATH, returned **No vulnerabilities found** after adding pinned JCS and go-jose. Recheck at actual integration/release; no scan establishes absence of undisclosed vulnerabilities. Module pins/checksums and retained licenses are recorded in THIRD-PARTY.md.

Design review: orchestration agent reviewed CANDIDATE-ADR-001 and approved isolated candidate implementation with explicit depth, metadata, type, cancellation, base64 and lifecycle requirements. This is not the required fresh independent final implementation audit or final production cryptographic approval.

Still unperformed: actual Circula service/store/auth/protected-key adapter tests, full ledger/root regression, real DB restart/backup recovery (this module owns no DB), external verifier implementation interoperability, production key/trust-distribution acceptance, and browser/Tor/manual keyboard/screen-reader qualification. No claim that static no-JS HTML tests establish WCAG 2.2 AA. Integration and release remain **HOLD**.

## Independent audit continuation

Fresh review and fixes are in AUDIT.md. Post-fix race suite passes (83.6%), vet passes, and all modules verify. Independent five-second certificate fuzz run passed 15,890 executions. Review fixed late membership/enablement changes during opaque signing and cancellation inside trusted callbacks, with regression tests. Remaining external/core/browser/production gates are unchanged.

## Master continuation — 2026-11-16

Tested source `c24b654da3502e68e5e585c4203e1873ce144bb8`. Go 1.26.11, GOGC=20, GOMAXPROCS=1, GOMEMLIMIT=450MiB, shared GOCACHE.

- `go test -p 1 -race -cover -count=1 -timeout 90s ./...`: PASS1.177s,84.0% statements.
- `go vet -p 1 ./...`: PASS, no diagnostics.
- `go mod verify`: all modules verified.

Added query-cap-before-host/strict duplicate/malformed/unknown-field rejection regressions for both receipt and certificate adapters, and URL/body reference agreement with valid ordinary form submissions. No new dependency or production route. Fresh independent delta review follows this implementation milestone; historical browser/external-verifier/key-operation gates remain open.

## Independent master audit — 2026-11-16

Exact runtime/test source: `860a55135a4d7302642829a08387a000ac1132ed`. Go 1.26.11, GOGC=20, GOMAXPROCS=1, GOMEMLIMIT=450MiB, shared GOCACHE.

- Before the correction, `TestMalformedFormCannotBeHiddenByHostParsing` failed: HTTP 200, source_reads=1 after a host consumed a malformed form. This expected red run exited 1.
- Focused form/query green tests passed 0.022s.
- Fresh `go test -p 1 -race -cover -count=1 -timeout 90s ./...` passed 1.146s, 84.1% statements.
- `go vet -p 1 ./...` passed with no diagnostics.
- `go mod verify` passed: all modules verified.

Raw command/result logs and hashes are in recovery-evidence. These new tests cover cached malformed form errors and positive/mismatched certificate query submissions. Runtime Go/dependency files are unchanged by the final documentation commit. No browser, external whole-verifier, production key or real core test was executed in this audit.
