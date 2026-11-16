# FNET-203 Engineering Design — Optional Trade Receipts and Certificates

Status: **PROPOSED — implementation-ready design draft; not approved for implementation** · Version: 1.0 · Owner/role: Software architect, security engineer, backend engineer · Last updated: 2026-11-14
Release disposition: **Not in the current release.** This document defines a future implementation boundary and acceptance contract only.
Parent: [FNET-203 Product Specification](FNET-203-optional-trade-receipts-certificates.md).
Repository context: Circula's architecture and security requirements in [05](../05-architecture.md), [06](../06-data-ledger-audit.md), [07](../07-api-contract.md), [09](../09-security-privacy.md), [11](../11-deployment-operations.md), and [12](../12-quality-test-plan.md) remain authoritative.

## 1. Engineering objective

Define a small, replaceable subsystem that can be added after the current release without changing the ledger's correctness model or making receipts part of trade completion.

The subsystem is a read-only consumer of existing completed transaction data. Its only externally visible effects are user-requested receipt/certificate responses and security/audit events required by existing policy.

**Non-negotiable boundary:** receipt generation is never in the transaction posting path. A receipt failure cannot fail, delay, reverse, or otherwise alter a trade.

## 2. Implementation gate and release isolation

No implementation work begins until the product owner explicitly approves a separate implementation milestone and the security/privacy decisions in section 16 are resolved.

Before that approval, engineers MUST NOT:

- Add source code, routes, templates, API endpoints, jobs, migrations, tables, indexes, settings, environment variables, secrets, key material, packages, or build steps.
- Add a feature flag or disabled stub to the current release.
- Change transaction models, posting code, export code, authorization rules, or audit schemas to anticipate FNET-203.
- Mark any current requirement as built or change release-readiness status.

When approved, implement on a dedicated branch and keep the module disabled by default until all acceptance gates pass.

## 3. Target architecture and module boundaries

Follow the existing Circula architecture: HTTP/HTML and JSON adapters call domain/application services; services use repository interfaces; adapters do not access storage directly.

Proposed module layout is logical, not a mandate for exact package names:

```
Receipt HTTP/API adapter
        |
        v
Receipt application service ---- Authorization policy adapter
        |
        +---- Completed-trade read interface
        |
        +---- Receipt schema / projection
        |
        +---- Canonical serializer + digest
        |
        +---- Format renderers (TXT, JSON; later HTML/PDF)
        |
        +---- Optional certificate issuer
        |          |
        |          +---- Signing-key provider (server-side only)
        |
        +---- Optional offline verifier library / CLI
```

### 3.1 Module responsibilities

| Module | Responsibility | Must not do |
|---|---|---|
| HTTP/API adapter | Parse request, authenticate, enforce request limits, return response | Contain transaction rules, sign arbitrary client payloads, read database directly |
| Receipt application service | Orchestrate authorization, projection, rendering and optional signing | Mutate ledger or transaction state |
| Authorization adapter | Reuse current member/network/transaction access rules | Create a parallel role system |
| Completed-trade reader | Fetch one authoritative completed trade and required ledger metadata | Return pending, declined, expired, or unauthorized records |
| Receipt projector | Build minimal versioned receipt DTO | Copy whole database models or private notes |
| Canonical serializer | Produce deterministic bytes for a specified schema/version | Normalize differently across platforms or silently change existing versions |
| Digest/signature service | Hash and sign server-generated canonical bytes | Accept browser-provided facts or expose private keys |
| Format renderers | Produce TXT/JSON from the same DTO | Add fields not in the schema |
| Verification component | Verify format, digest, signature, key id and trust chain | Claim the underlying real-world trade facts are independently proven |
| Key provider | Retrieve active signing key from approved protected storage | Store private key in source control, database exports, logs, or browser |

A module MUST be replaceable through a narrow interface. The basic receipt renderer MUST operate when the certificate issuer is absent or disabled.

### 3.2 Dependency direction

- Adapters depend on the receipt application service.
- The application service depends on interfaces for authorization, completed-trade reads, renderers, and optional signing.
- Receipt code may depend on existing domain types through a projection boundary; core domain and ledger packages MUST NOT depend on receipt code.
- Signing and verification packages MUST NOT be imported by transaction posting code.
- The receipt module MUST NOT introduce a new write path to the ledger.

## 4. Data contract

The receipt is a purpose-built projection, never a serialized database row or full transaction aggregate.

Proposed JSON shape (illustrative; final field names and schema are versioned before implementation):

```json
{
  "schema": "circula.trade-receipt.v1",
  "network_ref": "network-scoped-reference",
  "transaction_ref": "opaque-transaction-reference",
  "completed_at": "2026-11-14T12:00:00Z",
  "payer_ref": "network-scoped-party-reference",
  "payee_ref": "network-scoped-party-reference",
  "amount": {
    "minor_units": 1250,
    "unit": "network-credit"
  },
  "status": "completed",
  "disclaimer": "Circula credits are an accounting unit and are not necessarily cash or legal tender."
}
```

This sample is not a final API commitment. Use integer minor units and the network's configured precision; do not use floating-point arithmetic. Include display names only if the product decision and network visibility policy permit them.

### 4.1 Projection rules

- The source MUST be a completed transaction and authoritative ledger records read through the application/domain layer.
- The projection MUST be built on the server after authentication and authorization.
- It MUST include only the fields defined for its schema version.
- It MUST exclude credentials, sessions, contact details, precise location, unrelated balances/history, dispute narratives, reviewer notes, and administrative notes.
- The server MUST reject attempts to supply or override receipt fields in request bodies.
- Each schema version is immutable after publication. A changed meaning or field set requires a new schema version.
- A receipt represents a point-in-time snapshot. If a reversal occurs later, the original bytes remain a record of what was issued at that time.

### 4.2 Data retention

Default design: do not persist receipt files or duplicate receipt payloads. Generate from authoritative records on an authorized user request and return the response.

If later requirements establish a need to retain an issuance record, store only the minimum metadata (for example, receipt schema version, transaction reference, issuance time, and signing key id), subject to a retention decision and privacy review. Do not store a second full transaction record or file by default.

## 5. Request and response contract

Exact paths must be aligned with the API versioning and routing conventions in [07-api-contract.md](../07-api-contract.md) during implementation planning. The following are conceptual operations, not authorization to add routes now.

| Operation | Input | Output | Side effects |
|---|---|---|---|
| Request receipt | Transaction reference; requested format | TXT or JSON receipt | Security/audit event only where required by existing policy |
| Request signed certificate | Transaction reference; receipt schema version | Receipt plus signature metadata | Key-use/issuance event if required by security policy |
| Verify downloaded certificate | Local file/payload and trusted public key set | Valid/invalid/unknown-key result with reason | None; prefer offline verification |

Rules:

- Receipt and certificate operations MUST be safe to retry.
- No operation accepts an amount, party, timestamp, completion state, or ledger reference from the client as authoritative input.
- Unauthorized and nonexistent transaction references should not create a transaction-existence oracle. Use the repository's established indistinguishable error behaviour where appropriate.
- Apply request-size limits and normal rate limiting.
- Use private/no-store cache headers and avoid shared-cache storage.
- Content-Disposition filenames MUST use sanitized, non-sensitive identifiers; do not include member names or contact details by default.
- Do not put receipt contents, signature payloads, or secrets into URLs, query strings, telemetry, or logs.
- No receipt operation may call transaction mutation methods.

## 6. Cryptographic design

### 6.1 Server authority

The server constructs the canonical receipt payload from trusted records. The server computes the digest and signs the canonical bytes. Client-side code, if any, only packages or displays returned data; it does not establish transaction truth.

### 6.2 Recommended baseline for an ADR

Evaluate and, if approved, standardize:

- Canonicalization: JSON Canonicalization Scheme (RFC 8785) for the defined JSON payload.
- Digest: SHA-256.
- Signature: Ed25519.
- Key identifier: opaque, stable key id included in the certificate.
- Signature envelope: a documented, versioned JSON structure with explicit algorithm and canonicalization identifiers.

The final choice must be recorded in an ADR after review of Go library support, implementation maturity, interoperability, key lifecycle, and independent verification tooling. No algorithm or dependency may be silently substituted. Do not invent a custom cryptographic construction.

### 6.3 Key lifecycle

- Private signing keys MUST be generated and used within a protected server-side key boundary approved by operations/security.
- Private keys MUST NOT be committed, exposed to browsers, included in backups or application exports by default, written to logs, or returned by any endpoint.
- Key access MUST be restricted to the certificate issuer process/identity.
- Key generation, activation, rotation, retirement, compromise response, and backup/recovery policy MUST be documented before production use.
- Public keys and their key identifiers may be distributed for offline verification. The trust root and authenticity of public-key distribution must be documented.
- A verifier MUST distinguish valid signature, invalid signature, unknown key, unsupported schema/algorithm, malformed input, and revoked/retired key states.
- Rotation MUST preserve verification of previously issued certificates for the supported retention period. Do not silently replace historical public keys.
- If key compromise is suspected, suspend certificate issuance and publish a clear revocation/status process. Do not alter or delete the underlying transaction.

### 6.4 Verification semantics

A successful verification means only: the supplied canonical payload matches the signed digest and the signature verifies under the selected trusted public key. It does not prove identity of the human parties, delivery of goods/services, legal enforceability, cash payment, or the absence of later reversals or disputes.

## 7. Browser and Tor compatibility

### 7.1 Baseline path

TXT and JSON receipts MUST be generated server-side and downloaded through standard browser navigation/form submission. This path MUST work with JavaScript disabled.

No client-side hashing, signing, service worker, browser storage, or third-party script is required for the baseline receipt.

### 7.2 Optional JavaScript path

If client-side packaging or verification is retained, isolate it as a separate enhancement. Do not make the server's authoritative receipt generation depend on it.

Before starting, show the warning in the parent FNET-203 specification and require an affirmative choice. The warning must be available as ordinary server-rendered content and understandable with assistive technology.

The application MUST NOT:
- Detect Tor Browser and silently change its settings.
- Enable JavaScript or alter security settings.
- Require users to lower Tor Browser's security level.
- Block receipt-only or transaction workflows when JavaScript is disabled.
- repeatedly prompt a user after they decline.

A JavaScript-dependent enhancement must degrade cleanly: cancel, unsupported browser, script failure, or disabled JavaScript returns the user to the non-JavaScript path without loss of transaction state.

## 8. Privacy and threat model

### 8.1 Assets

- Member identities and network membership.
- Transaction parties, amount, timestamps, and transaction references.
- Network separation and authorization state.
- Signing private keys and trusted public-key metadata.
- Receipt files after download and issuance metadata, if any.
- Browser privacy settings and user choice.

### 8.2 Threats and controls

| Threat | Required control |
|---|---|
| IDOR / cross-network receipt access | Reauthorize every request against actor, transaction and network; test cross-network references |
| Forged client receipt fields | Ignore client-supplied transaction facts; project from authoritative server records |
| Private-key compromise | Protected key boundary, least privilege, rotation, incident process, no key in browser or logs |
| Receipt tampering | Canonical payload, cryptographic digest and signature; independent verifier |
| Sensitive data leakage | Allowlisted projection, no full model serialization, response cache controls, redacted logs |
| Receipt enumeration | High-entropy or opaque references, rate limits, indistinguishable unauthorized/not-found responses |
| Browser script compromise | No JS for baseline; minimal optional script; no remote assets; consent warning |
| Cross-site request abuse | Existing CSRF protections for cookie-authenticated unsafe methods; receipt generation itself should be read-only |
| Timing/metadata leakage | Avoid public verification lookup; no third-party trackers; minimize externally observable distinctions |
| Misleading proof claims | Explicit certificate semantics and disclaimer; verifier reports integrity/issuer only |
| Stale or misleading status | Timestamped snapshot semantics; separate reversal references; no mutation of historical receipt |
| Abuse by privileged operator | Follow existing operator access, break-glass and audit requirements; no new bypass |

### 8.3 Logging and telemetry

Log only operational metadata required by existing policy: request id, actor/network context as allowed, operation outcome, schema version, and key id where relevant. Do not log receipt bodies, party names, amount, signature private material, or downloaded file contents.

No analytics or telemetry event may reveal that a specific member requested a receipt unless this is already explicitly authorized by Circula's security/audit policy and documented to the member.

### 8.4 Retention and deletion

The default is no receipt-file retention. Existing transaction and audit retention rules remain authoritative. Any new issuance metadata requires a documented purpose, access controls, retention period, and deletion behaviour. A receipt already downloaded or shared by a member cannot be remotely recalled; disclose this before any sharing workflow.

## 9. Reversal, dispute, and historical semantics

- Receipt issuance is permitted only for a completed transaction, subject to current access policy.
- A receipt is a snapshot, not a live view.
- A later reversal, adjustment, or dispute does not rewrite the original receipt.
- If the requester is authorized, the UI may offer a separately generated status supplement referencing the later event. It must have its own timestamp and identifier.
- Receipt issuance MUST NOT open, resolve, or alter a dispute.
- The verifier should state that it verifies the signed snapshot, not current transaction status. A status check, if ever introduced, is a separate privacy-reviewed feature.

## 10. Error handling and failure isolation

Define stable, non-sensitive error classes consistent with the API contract:

- invalid request/format;
- not found or not authorized (combined where needed to prevent enumeration);
- transaction not completed;
- unsupported receipt schema;
- signing unavailable;
- signing key unavailable/retired;
- internal rendering failure;
- verification malformed/unsupported/invalid/unknown-key.

A failure to render, sign, or download a receipt MUST NOT modify any transaction, balance, ledger entry, dispute, or audit history except the security/operational records required by existing policy. Retry must be safe and must not create duplicate ledger effects.

## 11. Observability and operations

- Provide operational health for the optional module without exposing receipt contents.
- Alert on signing-key failures, unexpected signing volume, verification failures above an agreed threshold, and key-rotation/revocation events.
- Do not include transaction identifiers or personal data in high-cardinality metrics.
- Document operator procedures for enabling/disabling the module, key rotation, key compromise, incident response, and verifier key distribution.
- Feature enablement is off by default and must be reversible without data migration.
- Disabling the module must not affect core Circula routes, transaction history, or ledger operations.

## 12. Test plan and verification gates

### Unit tests

- Deterministic projection from authoritative fixtures.
- Explicit allowlist and rejection/absence of forbidden fields.
- Stable canonical bytes for a given schema version.
- Stable digest for canonical bytes.
- Signature valid/invalid cases, wrong key, unknown key, retired/revoked key, malformed certificate.
- TXT/JSON semantic equivalence.
- Unsupported schema and algorithm handling.

### Service/integration tests

- Authorized member can request only permitted completed-trade receipts.
- Pending, declined, expired, reversed-only, and otherwise ineligible transaction states are handled according to product rules.
- Cross-network and cross-member access is refused.
- Client-supplied amount, party, status, timestamp, and ledger fields are ignored or rejected.
- Receipt generation does not write to ledger or transaction tables.
- Certificate module disabled/unavailable does not disable receipt-only output.
- Receipt renderer disabled/unavailable does not affect trade completion.
- No receipt artefact is generated without explicit user action.
- Cache headers and response content type/disposition are correct.
- Logs contain no receipt body or forbidden fields.

### Browser/UX tests

- Full receipt path works with JavaScript disabled.
- User can decline certificate and continue.
- Warning is shown before any optional JavaScript-dependent operation.
- Declining does not trigger repeated prompts.
- Script failure/cancellation returns to the non-JavaScript path.
- Keyboard and screen-reader navigation work for the choice and warning.
- No remote assets, third-party trackers, or unexpected browser storage are used.

### Security tests

- IDOR and tenant-isolation tests across every route/format.
- Enumeration resistance and rate limiting.
- CSRF and content-type validation consistent with existing adapter rules.
- Signing key never appears in browser bundles, source maps, logs, API responses, exports, or backups.
- Tampered payload, substituted key id, algorithm confusion, duplicate/ambiguous JSON keys, and canonicalization edge cases are rejected.
- Key rotation preserves validation of prior certificates within the supported policy.
- Threat model and independent security review completed.

### Release acceptance

No rollout until all required tests pass, the security/privacy review is approved, operational key procedures exist, and the product owner explicitly accepts the open decisions. The current release is not to be modified to satisfy this future acceptance list.

## 13. Performance and availability

- Receipt generation is on-demand and bounded by existing request/body/time limits.
- No synchronous receipt work is added to trade posting.
- No background job is needed for the baseline.
- If future volume requires asynchronous generation, that requires a separate design because job queues, stored artefacts, and notification channels add new data-retention and privacy risks.
- Signing unavailability must disable only certificate issuance, not basic receipt generation or Circula transactions.

## 14. Migration and compatibility

The preferred design requires no schema migration. It reads existing authoritative records and produces an ephemeral projection.

If implementation proves that persistent issuance metadata is necessary, propose a separate migration with:
- purpose and data inventory;
- retention and deletion policy;
- tenant-scoped uniqueness and indexes;
- rollback/forward-compatibility plan;
- backup and restore implications;
- security review;
- explicit owner approval.

Receipt schema versions are independent of database schema versions and API versions. Previously issued files must remain verifiable for the documented support period.

## 15. Definition of done

- [ ] Product owner authorizes the implementation milestone.
- [ ] Open product/security decisions are resolved or explicitly accepted.
- [ ] Architecture decision record selects canonicalization, digest, signature, key format and rotation.
- [ ] Receipt DTO is allowlisted, versioned and independent of persistence models.
- [ ] Authorization reuses Circula's existing network and member policy.
- [ ] Read-only service has no ledger mutation dependency.
- [ ] TXT and JSON work without JavaScript.
- [ ] Certificate is a separate opt-in action.
- [ ] Private keys remain inside the approved server-side key boundary.
- [ ] Offline verification tool/library and public-key trust instructions are available.
- [ ] Privacy, logging, retention, cache and threat-model reviews pass.
- [ ] Reversal/dispute snapshot semantics are tested.
- [ ] All unit, integration, browser and security tests pass.
- [ ] Operational runbooks and incident/key-compromise procedures are accepted.
- [ ] Feature is disabled by default and can be disabled without affecting core Circula.
- [ ] No current-release requirements or release status were changed.

## 16. Required decisions before implementation

| ID | Decision | Options to assess | Required approver |
|---|---|---|---|
| FNET-203-OD-01 | Certificate issuance path | Server-side signature + normal download; optional client-side packaging; both | Product owner + security |
| FNET-203-OD-02 | Cryptographic suite | RFC 8785 + SHA-256 + Ed25519 recommended; alternatives only with documented rationale | Security/architecture |
| FNET-203-OD-03 | Party identifiers | Network-scoped opaque references; display names; user-selectable representation | Product + privacy |
| FNET-203-OD-04 | Public-key trust distribution | Static published key set; signed key manifest; other offline method | Security/operations |
| FNET-203-OD-05 | Key rotation/revocation | Retention of old public keys; compromise handling; verifier status semantics | Security/operations |
| FNET-203-OD-06 | Persistence | No persisted artefact by default; minimal issuance metadata only if justified | Product + privacy |
| FNET-203-OD-07 | Network-level enablement | Platform-wide availability; per-network opt-in/opt-out; hosted operator control | Product + governance |
| FNET-203-OD-08 | JavaScript-dependent enhancement | Omit entirely; optional packaging/verification with warning | Product + security |
| FNET-203-OD-09 | Formats | TXT/JSON baseline; HTML/PDF later | Product + accessibility |
| FNET-203-OD-10 | Status verification | Offline signature-only verification; separate online status feature after privacy review | Product + security |

## 17. Implementation handoff

When implementation is authorized, the engineering lead should:

1. Re-read the then-current versions of the parent and linked core specs; do not rely on this draft as a substitute for current architecture.
2. Record the approved decisions as ADRs and update the repository decision log.
3. Confirm that existing transaction/ledger and authorization interfaces provide the required read-only fields.
4. Implement the projection and TXT/JSON receipt path first, with JavaScript disabled in tests.
5. Add certificate issuance as a separate optional module after key management and cryptographic review.
6. Add an independent verifier and documented public-key trust process.
7. Add the UI consent/warning only for a separately approved JavaScript-dependent option.
8. Keep the module disabled by default until the full definition of done is satisfied.

**This document is a future engineering handoff, not an instruction to implement FNET-203 in the current release.**
