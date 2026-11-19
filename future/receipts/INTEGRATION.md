# Deferred receipts integration contract

**Disabled; HOLD. No integration is authorized.**

The standalone module exposes Handler, CertificateHandler and CLI functions with explicit Service/Issuer construction. Nothing registers itself, connects a database or posts a ledger mutation. A later separately authorized host must supply current authenticated membership, transaction-level access, immutable read-transaction facts, CSRF/rate policy and protected signing handles. These are mandatory contracts, not verified core adapters.

## URL and body agreement

Both choice pages accept only one nonempty `reference` query field. Query bytes are capped at 8192 before host callbacks. Duplicate/unknown/malformed fields fail with 400. POST accepts no URL fields or the same `reference` as the body; retaining this field supports ordinary no-JavaScript forms submitting to the choice URL. A mismatched URL/body reference fails before receipt lookup. POST facts remain exclusively from an authenticated source read, never URL/body assertions.

## Disable and lifecycle

Keep platform/network receipt and certificate flags false and omit host registration. No module data exists to migrate or erase: generated artifacts are ephemeral and explicitly downloaded. Downloaded copies cannot be recalled. Future host audit retention and trustworthy public-key distribution require an operating policy; preserve original transaction history. Source backup/restore belongs to the existing core and must be qualified in that separate phase.

See IMPLEMENTATION.md, CANDIDATE-ADR-001.md and CERTIFICATE-OPERATIONS.md for versioned projection, cryptographic candidate and retention/revocation limitations. Browser/manual accessibility, external whole-verifier interoperability and protected-key operations remain unqualified.

URL-encoded POST bodies are capped and parsed before Host.Authorize so that host CSRF form access cannot hide the first parsing error. The host must not preconsume the body or silently discard parser errors in earlier middleware. Valid PostForm remains available to the host hook.
