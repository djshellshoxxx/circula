# Candidate ADR 001: isolated certificate profile

Status: reviewed by orchestration agent on 2026-11-16 and approved for isolated candidate implementation only. Not production approval. Integration remains HOLD. No current Circula route, signing key or configuration is changed.

## Decision and purpose

Keep ordinary TXT/JSON/HTML receipts independent of certificate code. An explicitly requested certificate uses RFC 8785 canonical receipt JSON (gowebpki/jcs `7d24017834028a80af75c290d5b122d6346dc76b`), SHA-256 digest and standard JWS Compact EdDSA signing through go-jose v4.1.9 (`3c8a86c0b60170109f319d96b05da1e77cc8dbb4`). This is an experimental interoperability candidate, not a claim of accepted production cryptography. No browser code, HTTP key discovery or remote verification endpoint is introduced.

The signed canonical manifest contains schema `circula-certificate/v1`, issuer, network, key ID, issued-at UTC, signature algorithm `EdDSA`, canonicalization `RFC8785`, digest algorithm `SHA-256`, lowercase hexadecimal receipt digest and the versioned receipt DTO itself. The JWS protected header contains exactly `alg=EdDSA`, `typ=circula-receipt+jws`, `kid=<opaque ID>`. The compact payload is the canonical manifest. No JWT semantics, detached signatures, unprotected headers, embedded/remote keys, alternate algorithms, multiple signatures or signing user-supplied transaction facts are accepted.

## Key boundary and lifecycle

Issuer receives only an injected key provider returning active metadata and a `jose.OpaqueSigner` handle. The handle supports EdDSA and returns public information; private key export/generation/persistence is absent. It can wrap a protected server key boundary. The host remains responsible for credential/OS isolation. Issuance is allowed only during the metadata's half-open `[not_before,not_after)` window, with active state, and metadata exactly bound to the receipt network, configured issuer and opaque key ID. Current clock is injected and times are UTC. Certificates cannot be issued with retired, revoked, missing or future keys. No private key or signature payload enters logs or audit sinks.

Verifier receives a separate offline, public-only trust-store interface keyed by expected issuer + network + key ID. Caller supplies expected issuer and network; untrusted payload values cannot broaden this scope. Ed25519 public keys are accepted only from this trust store, never the JWS. Active keys verify in-window historic signatures. Retired keys verify signatures issued strictly before retirement (and inside validity window). Revoked keys return a revoked result for all certificates, including historic ones. Unknown key and retired-key-after-retirement are distinct reason codes. Rotation keeps old public keys and their original metadata; it never overwrites a key ID. Public-key distribution and compromise/revocation policy remain production approval gates.

## Verification order and bounds

Reject inputs over 64 KiB, non-compact serialization, invalid base64url, empty segments, protected headers over 2 KiB, payloads over 32 KiB, duplicate JSON keys, invalid UTF-8/surrogates, unknown fields, unsupported schema/algorithms, and unexpected headers. Parse protected headers with the pinned strict canonicalizer before go-jose and allowlist EdDSA in `ParseSignedCompact`. Unverified payload fields serve only bounded parsing and key selection; no receipt is returned until signature verification succeeds against the local trusted key. Then require canonical payload bytes, manifest/header key agreement, expected issuer/network, receipt schema/amount/revision/time constraints, issuance time no later than injected clock, lifecycle eligibility and exact digest. Signed snapshot completion cannot be later than issuance. A successful result returns a copied validated DTO and reason `valid` or `valid_retired_key`, and explicitly means signed historical snapshot integrity, not current status or cash settlement.

Stable results distinguish malformed, unsupported, unknown key, revoked key, retired key, invalid signature/digest, invalid key window and valid. Parsing errors and unknown keys disclose no transaction records because this verifier operates only on supplied files and local public keys.

## User consent and failure isolation

Certificate issuance requires separate explicit consent and independently enabled certificate configuration, then obtains its own fresh authorized receipt through the read-only service. A certificate failure affects only that optional download; ordinary receipt methods never depend on signer availability. CLI verification reads a bounded local file and injected trusted public metadata; no network, receipt retention, delivery or ledger write occurs. No signer is added to the ordinary receipt handler unless the host separately registers its optional certificate adapter after future integration approval.

## Evidence required

Tests cover exact canonical/digest values, fresh read/consent/default-off behavior, tampered payload/headers/kid/digest, wrong/unknown/revoked/retired key, rotation, future/out-of-window issue time, duplicate/unknown JSON, invalid Unicode, algorithm confusion, remote-key headers, bounds and cancellation. No production keys are used; synthetic Ed25519 test keys are generated per test and never written. Reproduce race/vet/module verification, short fuzzing and dependency vulnerability scan. Independent fresh-agent audit, interoperable external verifier, real core adapter, human accessibility and operational key distribution remain separate gates.

## Orchestration security review

Review accepted the candidate profile and required explicit maximum nesting depth 16, bounded metadata, exact Ed25519 public key type, cancellation tests, canonical base64url segments and only alg/typ/kid protected headers. Retirement relies on the signing key's asserted issued-at timestamp; an attacker with a compromised key can backdate. Therefore retirement is not compromise containment: compromise requires full revocation. This review does not replace fresh independent implementation audit or production approval.
