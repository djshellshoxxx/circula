# Complete recovered federation production-source review

## Disposition and exact source

**No blocking source finding** in the isolated production implementation at `5cf90f4ea3f9bcd17d4c7eaa611759080895c4dc`, tree `efc696eca11420d14ab526433af267eb6068347c`. This is an independent read-only source review by `/root/independent_triage`, not an external cryptographic certification, runtime pass, merge approval or integration selection. No authorization, replay or SSRF bypass was demonstrated. The worktree was clean and all eight working files hash to their exact committed blobs.

The complete production read covered 1787 lines across:

- `future/federation/envelope.go` — blob `c13fcbb9597cf333c84ff448f459e11bc26dd295` (82 lines).
- `future/federation/interface.go` — blob `df59e9675225f5a2b683638a27f6606399d22553` (365 lines).
- `future/federation/members.go` — blob `8fbdf00cab5975aeed77f9d4bf50707d77176687` (110 lines).
- `future/federation/operations.go` — blob `e76d00d532e558b5dcecfaaac4df2717a198cbe5` (241 lines).
- `future/federation/outbox.go` — blob `e526e7af283f5fbfc7bbfdb23ec2510fc90a0b29` (255 lines).
- `future/federation/protocol.go` — blob `fa911231b2a5e3d7174d4f8ef3dbef3bc07dfbe9` (155 lines).
- `future/federation/store.go` — blob `46ce287d2dd6a7c1c84daf9e7097103969ac557d` (442 lines).
- `future/federation/transport.go` — blob `f0332fedd0e42707c4a21b4450e5e56e4246d3d0` (137 lines).

The recovered protocol ADR, transaction/adapter contract, current recovery document and relevant adversarial tests were read alongside the code. Historical review claims were treated as context, not as a substitute for this read. Current combined-module qualification is being handled separately; this reviewer ran no Go build or test.

## Trust, signed input and replay

The v1 `Sign`/`Verify` helpers remain separate from v2 ingress. The recovered receiver accepts bounded compact JWS only. `WireKeyID` first limits the whole wire and protected header, verifies canonical base64url segments and exact signature length, and requires exactly the canonical `alg`, `typ`, `kid` header profile. The key selector remains untrusted until it is resolved through local tenant/peer configuration. `VerifyDiscovery` pins EdDSA, verifies with the configured public key, requires valid UTF-8 and exact re-encoded typed payload bytes, and validates audience/issuer, identifier grammar, kind/content pairing, revision and expiry bounds. This rejects unknown or duplicate signed fields and alternate representations without interpreting an event-supplied key or URL.

`Store.Receive` scopes keys and replay IDs by local network and remote peer. Network/peer enablement, current key validity/revocation, event issuance/expiry relative to the key window, event hash, replay insertion and projection update are checked under one immediate SQLite transaction. An event ID with a different hash is rejected; an equal listing revision with different kind/content is rejected; lower revisions cannot overwrite a newer publication or tombstone. The replay journal is durable and not evicted to regain quota. Replaying an already-accepted event cannot refresh its projection expiry.

`BrowsePage` authorizes the local caller in the transaction and joins only enabled networks/peers, nonrevoked configured keys and unexpired publication rows. In accepted data, receive-time enforcement also guarantees a projection expires no later than its signing key's validity end; the browse query does not need a second independent key-end comparison to preserve that invariant. Corrupt noncanonical content fails validation before output. This reasoning assumes the configured database and host adapters remain trusted; the module does not claim to protect arbitrary database rewrites.

## Consent, source ownership and transaction boundary

`Store.auth` rejects malformed actor IDs and requires a host authorization callback. Admin actions use the admin flag under the same immediate transaction; member sharing has no client-supplied role. `Consent` obtains the current source projection, requires the active owner, and scopes consent to a single peer/listing/owner. Publication `Queue` additionally checks network/peer enablement, active source, revision and existing consent before signing and recording an immutable event. Idempotency is scoped by local/remote/event ID and rejects owner/listing/kind/source-revision conflicts. No account, settlement or ledger-write event exists in the protocol.

Before a publication leaves `Dispatch`, current member authorization, active owner/source, source revision, consent and exact projected public content are checked again. The durable wire is bound back to the outbox's event ID, listing, revision, kind, expiry and local/remote identities. The real host must supply transaction-correct authorization and projection callbacks; these are trusted adapter seams, not client facts. The final Core adapter must share its existing transaction rather than nest this reference adapter's writer transaction. That integration is explicitly deferred.

## Leases, cancellation and withdrawal

The sender reserves a durable one-minute lease and increments a bounded attempt counter before network I/O. Completion updates require the same `sending` state and attempt number, so an old completion cannot overwrite a later attempt or cancellation. Retry count, expiry, state and scheduling checks are bounded; backoff shifts cannot grow beyond the guarded attempt range. The writer transaction is closed before transport runs. This is at-least-once delivery, with scoped receiver deduplication, rather than an exactly-once network claim.

Consent withdrawal, departure and source deletion cancel queued publications and record text-free withdrawal intents. Automatic withdrawal can be recorded without a live signer, source projection or departed owner's authorization; sealing is deferred until dispatch. Coalescing checks whether any publication needs a newer withdrawal. The ordinary 1,000-row lifetime outbox cap leaves a 2,000-row total reserve for corresponding automatic withdrawals. Maintenance removes expired content/wire bodies while preserving replay and revision safety metadata, and leaves unsealed withdrawal intents available after signer outage. Local revocation cannot recall a send already in flight or force a partitioned peer to erase data before its signed expiry; the interface and operations documentation state that boundary.

## Key lifecycle and outbound transport

Peer approval requires local admin authorization and a new key ID. Reusing an existing key ID causes the entire approval transaction to roll back, including its earlier peer upsert. Key/peer revocation disables subsequent acceptance and hides affected projections. Signing keys are injected and are not persisted by this module. Production private-key protection, distribution, rotation operations and independent peer approval remain host/operational gates.

`HTTPSDelivery` permits configured HTTPS endpoints only, without credentials, query strings, fragments or alternate ports. Literal forbidden IPs reject early. At dial time it checks the requested host/port, resolves at most16 addresses, rejects the complete answer set if any result is nonpublic, and dials a checked address directly. The standard HTTP transport retains hostname/certificate verification, enforces TLS1.2 or later, disables environment proxies and redirects, and bounds dial/handshake/header/whole-request time. Responses have an8KiB header limit and a4KiB body limit enforced with a limit-plus-one read. Error text does not include remote response bodies. The injected `Delivery`/`Resolver` interfaces remain trusted construction seams; using an arbitrary replacement transport is not the production `HTTPSDelivery` policy.

Existing transport tests inspect TLS/proxy/time settings and synthetic response behavior and reject mixed DNS answers. They do not establish live peer interoperability or a completed real-network TLS acceptance result; no such result is claimed here.

## HTTP adapters and bounds

Member/admin routes are unregistered host-constructed handlers. Queries are validated before host callbacks, form input is bounded and uses body fields rather than merged query mutation values, and the host is responsible for current authentication/CSRF. Private GET reads occur after the CSRF-token callback and a fresh same-actor lookup. Output uses escaped templates. The peer ingress has a method/media/body limit and a per-handler request window; the store additionally enforces a scoped accepted-peer rate. Lifetime peer, inbox, consent and outbox caps fail closed. These are explicit candidate limits; comprehensive production rate policy and capacity/epoch design remain separate gates.

## Remaining qualification

Source approval does not replace full current-source real-driver tests, race/vet/fuzz, native provider gates, independent live-peer/TLS interoperability, actual Core read/write/lifecycle adapters, production key operations, or manual browser/accessibility review. No folder replacement, runtime activation, retention-policy choice, migration registration or settlement work was authorized or performed. PR106's README ownership and PR110's recurring handoff remain unchanged.
