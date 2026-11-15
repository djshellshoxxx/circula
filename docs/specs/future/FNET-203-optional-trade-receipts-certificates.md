# FNET-203 — Optional Digital Trade Receipts and Cryptographic Certificates

Status: **PROPOSED — future development only** · Version: 1.0 · Owner/role: Product owner and software architect · Last updated: 2026-11-14
Scope: Optional, independently deployable receipt and certificate capability for completed Circula trades.
Release disposition: **EXCLUDED from the current release. Documentation only; no implementation is authorized by this specification.**
Dependencies: [00-conventions](../00-conventions.md), [03-domain-model](../03-domain-model.md), [05-architecture](../05-architecture.md), [06-data-ledger-audit](../06-data-ledger-audit.md), [07-api-contract](../07-api-contract.md), [09-security-privacy](../09-security-privacy.md), [11-deployment-operations](../11-deployment-operations.md), and [12-quality-test-plan](../12-quality-test-plan.md).
Engineering companion: [FNET-203 Engineering Design](FNET-203-engineering.md).

## 1. Purpose

FNET-203 proposes an optional way for an authorized Circula member to obtain a portable record of a completed trade. A separate, optional cryptographic certificate may allow a recipient to verify that the receipt payload was issued by Circula and has not been altered since signing.

The feature is a read-only presentation and export layer over Circula's authoritative transaction and ledger records. It is not a new ledger, payment rail, settlement mechanism, or accounting subsystem.

**A trade MUST complete whether or not either party requests a receipt or certificate.** Users MUST never be required to generate, download, retain, or share either artefact.

## 2. Status and release boundary

This is a future-facing proposal. It is not part of the current release or current MVP acceptance criteria.

Until the product owner explicitly authorizes a later implementation phase:

- No application code, routes, handlers, templates, UI controls, background jobs, feature flags, dependencies, configuration, migrations, database tables, signing keys, or deployment changes may be added for FNET-203.
- No existing transaction, ledger, export, authentication, or audit behaviour may be changed to anticipate this feature.
- No receipt or certificate may be generated automatically.
- No current release gate or completion status may be changed because this proposal exists.
- Documentation work does not imply implementation approval.

Implementation must be separately planned, reviewed, tested, and accepted. The feature must be independently removable or disabled without affecting core Circula operations.

## 3. Product principles and user choice

### FNET-203-001 — Transaction independence

The receipt/certificate subsystem MUST be downstream of a completed trade and MUST NOT participate in proposal, acceptance, posting, reversal, adjustment, or dispute decisions.

### FNET-203-002 — Entirely optional artefacts

A member MAY choose to request a receipt. A member MAY separately choose to request a cryptographic certificate. Neither is selected by default, and declining either MUST NOT affect transaction completion, account status, balances, access, or other functionality.

### FNET-203-003 — No automatic generation or retention

The system MUST NOT generate, store, email, publish, or share a receipt or certificate unless the user has explicitly requested the relevant action. A generated download may exist in the user's browser or device after the user chooses to save it; Circula MUST NOT silently retain a second copy.

### FNET-203-004 — No JavaScript requirement for Circula

JavaScript MUST NOT be required to use Circula, complete a trade, view ordinary transaction history, or choose to decline FNET-203. The feature MUST provide a usable non-JavaScript path for any capability that can reasonably be delivered by a normal server response.

### FNET-203-005 — Separate consent for browser-side processing

If a future implementation offers client-side certificate packaging or verification that requires JavaScript, it MUST disclose that requirement and its privacy/security implications before that process begins. The user must affirmatively choose it. The application MUST NOT enable JavaScript, change browser settings, or encourage a user to lower Tor Browser's security level.

### FNET-203-006 — No pressure or repeated prompting

The interface MUST provide neutral choices. Declining must be as easy as continuing. A user who declines MUST NOT be repeatedly prompted during the same transaction or on subsequent visits unless they deliberately reopen the receipt action.

## 4. Scope

### In scope for a future implementation

- A human-readable record of one completed trade.
- Portable receipt formats, initially TXT and JSON.
- Optional cryptographic signing and independent verification.
- Authorization checks using Circula's existing identity, membership, and network boundaries.
- A non-JavaScript server-generated receipt path.
- A separately consented, optional JavaScript-dependent browser workflow, if retained after security review.
- Explicit privacy controls, data minimization, and clear limits on what the receipt proves.

### Out of scope

- Changing the ledger or creating a second ledger.
- Creating, accepting, posting, reversing, or adjusting trades.
- Public transaction lookup, public receipt registries, or publicly enumerable receipt URLs.
- Payment processing, cash settlement, redemption, escrow, or proof of external payment.
- Identity verification or proof of a person's legal identity.
- Legal, tax, accounting, regulatory, or financial certification.
- A guarantee that a trade is fair, dispute-free, enforceable, or accepted outside the relevant network.
- Automatic email delivery, third-party storage, analytics, tracking, or document-generation services.
- Requiring a member to disclose a receipt to another member.

## 5. Feature modules

The future feature is divided into independent modules. The basic receipt must not depend on the certificate module.

1. **Receipt data interface:** obtains a minimal, authorized, read-only representation of one completed trade.
2. **Receipt renderer:** turns that representation into a human-readable receipt.
3. **Format exporters:** provide supported output formats such as TXT and JSON; HTML/PDF are later optional formats.
4. **Certificate issuer:** signs the authoritative receipt payload using a protected server-side private key.
5. **Certificate verifier:** verifies the signature and integrity using a trusted Circula public key and documented format.
6. **Authorization adapter:** reuses Circula's existing authorization and network-isolation rules.
7. **Presentation adapter:** exposes user-initiated actions through the existing HTML and API layers, without embedding business rules in the interface.
8. **Policy/configuration adapter:** controls whether the feature is enabled for a deployment or network, without changing core transaction behaviour.

Each module MUST have a narrow interface and be testable without the others. The certificate issuer/verifier MUST be optional dependencies from the receipt renderer's perspective.

## 6. Receipt content and formats

### 6.1 Proposed minimum content

The exact schema is to be finalized in the engineering design and a versioned contract before implementation. The initial receipt may contain:

- Receipt schema version.
- Circula network identifier or network display label, using the least identifying form that remains useful.
- Transaction reference.
- Completion timestamp, including timezone/UTC representation.
- Payer and payee references scoped to the network; display names only where necessary and permitted by network visibility policy.
- Amount and the network's credit-unit label.
- Completed status at the time represented by the receipt.
- Relevant ledger-entry references, only if needed for verification and safe to disclose.
- A plain-language explanation that Circula credits are an accounting unit and are not necessarily equivalent to cash, legal tender, or a claim redeemable for national currency.
- Optional certificate metadata when the user separately requests a certificate.

The receipt MUST clearly distinguish a completed Circula ledger transaction from external delivery of goods/services or payment. Circula cannot attest to facts it does not observe.

### 6.2 Formats

- **TXT:** preferred baseline human-readable format; portable and usable without JavaScript.
- **JSON:** structured, versioned format for software verification and archival.
- **HTML:** optional, self-contained format; no remote assets or tracking.
- **PDF:** optional future format; no third-party document-generation service may be mandatory.

The same authoritative fields and semantics MUST apply across formats. Format changes MUST NOT change the underlying transaction record.

## 7. Optional cryptographic certificate

The certificate's purpose is limited: it can establish that a particular canonical receipt payload was signed by a Circula signing key and has not changed since signing, subject to trust in that key and the verification process.

Proposed signing flow:

1. Circula confirms that the trade is completed and the requester is authorized to obtain the selected fields.
2. The server constructs the authoritative, versioned receipt payload from trusted transaction and ledger records.
3. The server canonicalizes and hashes that payload using the format and algorithm selected in the engineering design.
4. The server signs the digest or canonical payload using a protected private signing key. The private key MUST remain server-side and MUST never be returned to a browser, API client, log, receipt, or export.
5. Circula returns the receipt and, only if requested, its certificate metadata/signature.
6. A verifier checks the signature using the corresponding trusted public key and confirms the payload digest.

The browser MUST NOT be treated as an authority for transaction facts, hashes, signature material, or certificate validity. Any browser-side packaging is presentation only. The server MUST validate and sign its own authoritative representation, never data supplied by the browser as proof of the trade.

The initial implementation should evaluate a deterministic JSON canonicalization standard (for example RFC 8785), SHA-256, and a modern asymmetric signature algorithm such as Ed25519. The final algorithm, canonicalization implementation, key format, rotation rules, and verifier distribution MUST be approved in a security-reviewed architecture decision record before implementation. Algorithms and schema versions must be explicit; silent algorithm substitution is prohibited.

A certificate should identify, at minimum:

- Certificate format/version.
- Issuer identifier (Circula deployment or network, as defined by policy).
- Transaction reference.
- Receipt schema version.
- Digest algorithm and digest.
- Signature algorithm and signature.
- Signing key identifier.
- Issued-at timestamp.
- Canonicalization identifier.

A certificate MUST NOT claim legal settlement, cash payment, delivery of goods/services, legal identity, legal enforceability, or external financial certification.

A hash is not encryption and does not conceal the receipt data. A signature does not make the underlying transaction private, anonymous, true beyond the authoritative records, or legally certified.

## 8. Tor Browser compatibility and JavaScript warning

Tor Browser compatibility is a privacy requirement, not an afterthought.

The basic receipt path MUST be server-generated and usable without JavaScript. It should be delivered as a normal authenticated response or download, with no client-side cryptographic computation required.

If a later implementation retains an optional browser-side certificate packaging or verification workflow, that specific workflow may require JavaScript. It MUST NOT be the only way to obtain a receipt or to use Circula.

### Required warning for a JavaScript-dependent option

> **JavaScript required — privacy and security notice**
>
> This optional browser-based certificate process requires JavaScript.
>
> If you are using Tor Browser, enabling JavaScript or lowering your current security protections may reduce some of the privacy and security benefits of your configuration. Depending on the site code and browser settings, JavaScript can increase the browser's attack surface and may make browser behaviour more distinguishable. It can also expose you to malicious or vulnerable scripts.
>
> Enabling JavaScript does not automatically reveal your identity. However, it may reduce protections you have deliberately chosen.
>
> **You do not need to enable JavaScript to use Circula, complete trades, or obtain an available server-generated receipt.**
>
> You can decline this optional browser-based process and continue using Circula normally. Do not change Tor Browser settings unless you understand and accept the trade-off.

The warning MUST appear before the user starts the JavaScript-dependent process. It MUST identify the exact action that needs JavaScript. It MUST NOT suggest enabling JavaScript globally or lowering Tor Browser's security level as the default solution.

### User choices

The interface should provide clear, neutral actions:

- Continue without a certificate.
- Obtain an available non-JavaScript receipt.
- Cancel and return to the transaction.
- Proceed with the optional JavaScript-dependent process.

If a non-JavaScript certificate issuance path is implemented, it should be offered as a separate server-side option and must not be conflated with client-side processing.

## 9. Privacy and data minimization

The feature MUST inherit Circula's existing network isolation, authorization, privacy, and audit requirements. It MUST NOT create a new public disclosure surface.

The receipt MUST include only fields necessary for the user's chosen purpose. In particular, it MUST NOT include:

- Email addresses or telephone numbers.
- Physical addresses or precise location.
- Login credentials, session identifiers, API tokens, or recovery material.
- Account balances unrelated to the trade.
- Other transaction history.
- Private dispute statements, reviewer notes, or administrative notes.
- Internal operator identifiers or security metadata.
- Unnecessary global or cross-network member identifiers.

Additional requirements:

- No public transaction lookup or public receipt database.
- No predictable public receipt URLs or unauthenticated retrieval endpoint.
- No third-party analytics, trackers, remote fonts, remote images, or external document processors.
- No automatic receipt storage, email, or sharing.
- Private, authorized access only.
- Responses containing sensitive receipt data MUST use appropriate private/no-store cache controls and must not be cached by shared intermediaries.
- Downloads and generated files MUST NOT be logged with their sensitive contents.
- The feature MUST NOT add a new persistent copy of the receipt unless a separately approved requirement establishes a clear need, retention period, access policy, and deletion process.
- A receipt copied or saved by a user is outside Circula's ability to revoke; the interface should explain this where sharing is offered.
- Hashes, transaction references, and certificates may remain linkable or identifying. They MUST NOT be described as anonymous.

## 10. Authorization and access

- A requester MUST be authenticated using Circula's existing mechanisms.
- The requester MUST be authorized for the relevant transaction and fields under the current network policy.
- Network scoping MUST be enforced server-side on every request.
- A member MUST NOT obtain another member's receipt merely by changing an identifier or URL.
- The API and HTML paths MUST apply equivalent authorization checks.
- The certificate must not provide a bypass around transaction access controls. A signed file can be shared by its holder, so the data included in it must be safe for that intended disclosure.
- No public verification endpoint may reveal whether a private transaction exists. Offline verification of a file is preferred. Any online key-discovery or revocation mechanism requires a separate privacy review.

## 11. Reversals, disputes, and corrections

A receipt is an immutable snapshot of the transaction state represented when it was issued.

- A receipt MUST NOT rewrite or mutate the original transaction.
- A reversal or adjustment MUST retain its own transaction/ledger references and history.
- A later reversal MUST NOT invalidate the historical fact that the original transaction was completed and then reversed.
- A receipt MAY indicate that a later reversal exists only if the requester is authorized to see that information and the status is clearly timestamped.
- Dispute content and private reviewer notes MUST NOT be included.
- Re-issuing a receipt after a material status change must create a new version or a clearly identified supplemental record; it must not silently overwrite a previously issued file.

## 12. Integration constraints

- Reuse Circula's authoritative transaction, ledger, identity, and authorization services.
- Read through the existing domain/service layer; do not read or write database tables directly from an adapter.
- Do not duplicate accounting logic or create a receipt ledger.
- Do not add a transaction mutation or modify ledger invariants.
- Keep receipt data contracts versioned and independently testable.
- Keep receipt rendering, format export, signing, and verification independently replaceable.
- The feature MUST fail closed for unauthorized or inconsistent data and MUST fail open with respect to the underlying trade: receipt failure must never roll back or block a completed trade.
- Do not require a new database table unless a reviewed, documented need is demonstrated.
- Feature enablement must be off by default until the implementation is separately approved.

## 13. Future acceptance criteria

Before release approval, the implementation team must demonstrate:

- [ ] Feature is absent/disabled in the current release.
- [ ] A trade completes normally when the feature is disabled or unavailable.
- [ ] Receipts are opt-in and are never automatically generated or stored.
- [ ] Certificates are a separate opt-in action.
- [ ] TXT and JSON output follow a versioned schema.
- [ ] Receipt fields are sourced from authoritative server-side records.
- [ ] Authorization and network isolation are enforced on every path.
- [ ] Private signing keys never leave the server-side key boundary.
- [ ] A certificate can be independently verified against a trusted public key.
- [ ] Canonicalization and algorithm identifiers are explicit and versioned.
- [ ] The non-JavaScript receipt path works.
- [ ] Any JavaScript-dependent option displays the warning before execution.
- [ ] Users can decline without repeated prompts or loss of functionality.
- [ ] No public transaction lookup, public receipt registry, tracking, or third-party document service is introduced.
- [ ] Data minimization, cache controls, logging, retention, and deletion are reviewed.
- [ ] Reversals and disputes preserve the original receipt's historical meaning.
- [ ] Failure of the receipt subsystem cannot affect ledger state.
- [ ] Security review, privacy review, threat model, and operational key-management review are approved.

## 14. Open product decisions

The following are intentionally unresolved until the implementation phase:

1. Whether the first implementation offers receipt-only, server-signed certificates, optional browser-side packaging, or more than one path.
2. Whether both parties can request the same receipt fields, or whether the requester sees a party-specific representation.
3. Whether network display names appear in receipts or only network-scoped opaque references.
4. Whether HTML/PDF are needed beyond TXT/JSON.
5. Whether certificate verification is entirely offline or includes a privacy-preserving key-discovery/revocation mechanism.
6. Whether a network administrator may disable receipt issuance, and whether the platform operator may do so in hosted mode.
7. Whether any signed receipt metadata is persisted, and for how long.
8. Which canonicalization, digest, signature, key-rotation, and revocation standards are adopted.

No default in this list is an approved product decision. Record decisions in the repository's decision log before implementation.

## 15. Current disposition

**FNET-203 is a proposal for a later release.** It is included in the repository as planning documentation only. It does not authorize code, schema, migration, UI, endpoint, dependency, configuration, key, or release changes.

The governing principle is simple: users decide whether to create a receipt, whether to obtain a certificate, and whether to use any optional browser process that changes their security posture. Circula remains fully usable when they decline.
