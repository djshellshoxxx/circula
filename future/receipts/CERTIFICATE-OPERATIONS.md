# Candidate certificate operations

**HOLD. Isolated candidate only.** CANDIDATE-ADR-001 records the orchestration design review, not production authorization or an independent security audit.

## Enable and disable

Both receipt service and certificate issuer require independent explicit enablement and network configuration. A zero value of either is disabled. The host must explicitly register `CertificateHandler`; the ordinary `Handler` neither signs nor depends on signing keys. Never register these adapters in Circula from this branch. The no-JavaScript certificate choice is a distinct user action. Closing it or returning to transaction history has no persistent side effect. No JavaScript-dependent path exists.

After a future single-feature integration is accepted, a host can register the server-side certificate adapter only after selecting a protected `SigningKeys` provider. Disable certificate issuance by setting issuer Enabled false and removing only certificate routes/CLI registration. Keep ordinary receipts and core trades available. Do not erase historical ledger/transaction data or downloaded files.

## Private key boundary

The module has no private key generation, serialization, disk format, environment-variable loading, HTTP discovery or recovery mechanism. The injected provider returns `jose.OpaqueSigner`; Public must contain an exact `ed25519.PublicKey`, never private material. The host protects signing credentials and device/process access. The opaque handle must stay bound to one key for an entire call. Captured public-key verification detects a rotated/misbound handle before return. Final provider and clock checks reject key revocation, rotation or expiry during signing.

The opaque interface cannot interrupt a hardware operation mid-call. The provider must enforce deadlines itself. Context is checked before/after each relevant boundary; a cancelled operation returns no certificate. No private material, manifest, receipt body or token is logged. Module audit sinks receive only fixed receipt outcome labels. `Issuer.Health` provides only enabled/ready/reason for future authorized operational diagnostics and never returns key IDs, transaction data or receipt contents. No health endpoint is registered. Production operators must decide low-cardinality key-failure and volume monitoring under existing policy; do not label metrics by member or transaction.

Production signing-device setup, operator permissions, key backups/recovery and trust bootstrap require accepted operations/security decisions. They are not satisfied by the synthetic in-memory test signer. A certificate cannot attest to trustworthy issuance time beyond the issuer's signed clock assertion.

## Offline public-key trust

Verification requires configured expected issuer and network, plus an independent public-only `TrustStore`. It never trusts a JWS-embedded public key, URL, kid-as-URL, certificate-provided trust file or network key registry. `LoadTrustStore` provides a bounded immutable offline configuration reader with schema `circula-trusted-keys/v1`, at most 64 entries and at most 64 KiB. Each entry contains issuer, network, key_id, state, UTC not_before/not_after, optional retired_at, and a canonical base64url 32-byte public_key_base64url. Only locally authenticated operator configuration may supply this file. Keep it separate from downloads.

The caller must authenticate public-key distribution independently, for example through an already trusted operator-controlled offline configuration channel. Loading a file does not prove it is authentic. The library makes no freshness claim about the caller's key set. A valid result means valid under that supplied local policy; it cannot discover a later compromise in an outdated offline file.

## Rotation and retirement

Generate/activate a new protected key only through the host's approved key operations. Assign a new opaque key ID scoped by issuer and network; never reuse or overwrite an existing key ID with another public key or altered original validity window. Publish its public record through the authenticated channel before issuing certificates. Reload an entirely new immutable trust store at an explicit administrative boundary, retaining original historic keys and metadata for the supported period. An in-flight issue detects changed current key identity and fails safely; retry can use the new key.

Only active keys in `[not_before,not_after)` issue. Retired keys never issue but verify historical signatures asserted to precede retired_at, within their original issuance window. Retired-at comparison is strict. Retirement is not compromise containment: a compromised private key can backdate its signed issued_at. Use full revocation for compromise. Normal verifier clock checking rejects future issuance and completion-after-issuance; there is no external timestamp authority.

## Compromise and revocation

1. Disable certificate issuance and isolate the affected host signing boundary.
2. Mark the affected issuer/network/key ID revoked in the authenticated local key policy; all signatures under that key then return revoked_key, including old ones.
3. Distribute the replacement trusted key set through the approved channel. Offline users with stale policy will not learn revocation until updated. Previously downloaded files remain on their devices and cannot be remotely recalled.
4. Preserve ledger and audit history. A signature/key event does not mutate a completed trade.
5. Restore issuance only with a newly approved protected key/ID and operational review. Basic receipt downloads remain independent throughout.

## CLI and results

Future host linkage exposes these functions after existing authentication and network selection:

```
certificate --transaction <reference> --request-certificate
verify-certificate --file <local-certificate.jws>
```

`RunCertificateCLI` requires separate explicit consent and writes compact JWS without newline. `RunVerifierCLI` opens only the user-selected local file, uses configured expected issuer/network and injected offline trust, and prints a reason plus limited interpretation. It does not reveal the receipt body. It never fetches keys or current transaction status. Human assistance may request ordinary receipt output instead; users need not enable browser scripts.

Verification results distinguish valid, valid_retired_key, malformed, unsupported, unknown_key, key_unavailable, revoked_key, retired_key, invalid_key, invalid_key_window, invalid_signature, invalid_digest, invalid_time, invalid_scope, invalid_configuration and cancelled. Only valid/valid_retired_key results contain a parsed manifest. Validity means a supplied signed historical snapshot matches the trusted key; it does not prove current status, identity, delivery, cash payment or external legal certification.

## Recovery and retention

No receipt or certificate artefact is retained by the module, and no migration or application DB exists here. A process restart reconstructs service/issuer adapters and reloads only approved public policy through host configuration. Private key backup/recovery stays inside the protected key operations boundary. Receipt/certificate output is on-demand and does not participate in trade transaction commit. Public key retention must preserve offline verification for the supported period; removing a historic key changes results to unknown_key.

## Remaining qualification

The implemented profile and unit/HTTP/CLI/fuzz tests are an isolated candidate. A fresh independent implementation audit, external verifier interoperability, actual Circula authorization/read-transaction/key adapter qualification, operational trust distribution acceptance, browser/keyboard/screen-reader/Tor acceptance, full ledger regression and owner-selected integration remain required. No local test turns this candidate into production crypto approval.
