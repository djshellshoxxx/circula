# FNET-209 upstream research and fresh-agent handoff

Date: 2026-11-16. **Research complete; implementation and independent audit pending. Integration HOLD.** This research authorizes no main-branch registration, deployment or settlement behavior. Work remains confined to `future/federation/`.

## Selected reuse

Use **go-jose/go-jose v4.1.9**, exact commit `3c8a86c0b60170109f319d96b05da1e77cc8dbb4`, as a pinned library for a provisional JWS Compact/EdDSA discovery wire format. Reuse its tested signing, signature parsing, algorithm allowlist and verification implementation rather than expanding the prototype's custom signature framing. Keep the Circula-specific peer trust, schema, audience, expiry, replay and publication policy separate.

This is a deliberate experimental wire-format revision, not a claim that v1 prototype bytes are interoperable. Record an ADR and explicit format/version/type; do not silently accept both custom and JWS formats. The package is independently usable without any other future feature. Nothing here adopts cross-network settlement.

## Candidates and evidence

GitHub repository searches located CloudEvents and current go-jose; go-ap vocabulary was inspected directly at its official repository after search results favored unrelated forks. Shallow clones were inspected, including source and license text. All commit dates are observed snapshots, not future maintenance assurances.

| Candidate | Exact identity and maintenance evidence | Suitability and decision |
|---|---|---|
| [go-jose/go-jose](https://github.com/go-jose/go-jose) | `v4.1.9` -> `3c8a86c0b60170109f319d96b05da1e77cc8dbb4`, commit 2026-11-05. Current unarchived successor to archived square/go-jose. Go 1.24 minimum. [SECURITY.md](https://github.com/go-jose/go-jose/blob/3c8a86c0b60170109f319d96b05da1e77cc8dbb4/SECURITY.md) declares >=v3 supported and a security reporting process. Source includes extensive signing/verification tests. | **Selected** standard cryptographic envelope implementation. It does not establish trusted peers or implement an inbox/outbox. Reject deprecated broad parsing APIs and constrain algorithms. |
| [cloudevents/sdk-go](https://github.com/cloudevents/sdk-go) | Stable `v2.16.2` -> `af3e8599b3316ab6b4b73ff69aa8ec0efddbb5bb`, release commit 2025-09-04, Go 1.23. Repository HEAD inspected `d673df6fd72e420ae0098eb2b7c29648302ff9e9`, 2026-11-13, now requires Go 1.26. Official SDK, conformance suite, event tests and active repository. Distinguish maintained HEAD from older stable tag. | Mature event vocabulary/protocol library, **reference only initially**. Its `Event.Validate` checks CloudEvents structure, not signatures, audiences, replay, bilateral policy or SSRF. Adding its full transport stack would not satisfy these controls. A future explicit CloudEvents adapter may reuse `v2/event` after wire-contract approval; not needed for the minimal strict listing DTO. |
| [go-ap/activitypub](https://github.com/go-ap/activitypub) | Exact HEAD `b164c2575fd9011248507b1db4c388dd4349df4c`, 2026-11-14. Go 1.26. Extensive vocabulary tests; official README describes ActivityStreams/ActivityPub data types, not a complete federation server. | **Not selected** for this discovery-only protocol. Broad actor/object/collection vocabulary creates unnecessary fields and semantics. It supplies neither durable delivery nor local trust policy. No public actor discovery, social graph or foreign inbox URLs are required. |

## License and provenance

- go-jose [LICENSE](https://github.com/go-jose/go-jose/blob/3c8a86c0b60170109f319d96b05da1e77cc8dbb4/LICENSE): Apache-2.0. Preserve the license and applicable source copyright notices; preserve any upstream NOTICE if a later selected distribution includes one. The inspected root contains LICENSE and no NOTICE file. Do not import archived square/go-jose instead.
- CloudEvents [LICENSE](https://github.com/cloudevents/sdk-go/blob/af3e8599b3316ab6b4b73ff69aa8ec0efddbb5bb/LICENSE): Apache-2.0. Reviewed `v2/event/event_validation.go` carries The CloudEvents Authors copyright and SPDX Apache-2.0.
- ActivityPub [LICENSE](https://github.com/go-ap/activitypub/blob/b164c2575fd9011248507b1db4c388dd4349df4c/LICENSE): MIT, copyright 2017 Golang ActitvityPub (spelling as upstream).
- No upstream source is copied or dependency added in this research phase. Implementation should record package version, commit, source paths, adopted APIs, transitive license inventory and any changed/copied files in THIRD_PARTY_NOTICES.md. Retain full license text for copied/bundled portions. Circula's own license remains undecided; do not add or infer a root project license. No AGPL/GPL server transplant is proposed.

## Specific reusable APIs and documented limits

At go-jose's selected commit:

- [`signing.go`](https://github.com/go-jose/go-jose/blob/3c8a86c0b60170109f319d96b05da1e77cc8dbb4/signing.go): `NewSigner(SigningKey{Algorithm: EdDSA, Key: ed25519.PrivateKey}, options)`, `SignerOptions.WithType`, `Signer.Sign`, and `JSONWebSignature.Verify` provide the crypto machinery. An explicit protected type separates federation from receipt/JWT use. Key ID must select only an already trusted local peer key.
- [`jws.go`](https://github.com/go-jose/go-jose/blob/3c8a86c0b60170109f319d96b05da1e77cc8dbb4/jws.go): `ParseSignedCompact(raw, []SignatureAlgorithm{EdDSA})` and `CompactSerialize`. Parsing does **not** verify a signature: always call Verify with locally pinned key before interpreting payload. Never use UnsafePayloadWithoutVerification as an accepted event. Reject embedded key/URL discovery headers and unexpected type/critical extensions; do not permit general JWS JSON/multiple signatures unless separately designed.
- [`asymmetric_test.go`](https://github.com/go-jose/go-jose/blob/3c8a86c0b60170109f319d96b05da1e77cc8dbb4/asymmetric_test.go) and [`signing_test.go`](https://github.com/go-jose/go-jose/blob/3c8a86c0b60170109f319d96b05da1e77cc8dbb4/signing_test.go): useful interoperability, malformed signature and algorithm/key mismatch test patterns. Adapt targeted cases with provenance, not the whole upstream test harness.
- Official API documentation: [go-jose v4](https://pkg.go.dev/github.com/go-jose/go-jose/v4), [CloudEvents event](https://pkg.go.dev/github.com/cloudevents/sdk-go/v2/event), [ActivityPub vocabulary](https://pkg.go.dev/github.com/go-ap/activitypub). Version-pinned cloned source is authoritative if current docs differ.

Do not copy CloudEvents' README `NewClientHTTP` example as a production transport. It accepts a context target and has no Circula allowlist/DNS/private-address policy. Likewise ActivityPub object URLs are data, never an instruction to fetch a remote URL.

## Existing Circula seams and prototype gaps

Inspected `internal/service/listings.go`, `internal/store/iface.go`, root configuration and `future/federation/envelope.go`.

- `ListListings` authenticates active local membership, but an owner querying their own listings can see nonactive states. Export projection must explicitly require active listing **and active owner**, fresh owner opt-in, network opt-in and peer-scoped sharing. A local-visible listing is not automatically authorized for foreign disclosure.
- `store.Store.Update` uses a serialized immediate write transaction. Inbox replay recording and listing projection/tombstone mutation belong in one transaction. An in-memory seen callback or check-before-write is insufficient.
- No production peer trust/inbox/outbox seam exists in the inspected core. Implement standalone domain/service/store interfaces and SQLite adapter under this module; integrate through a later core adapter. Do not expose the ledger or import transaction services.
- The prototype authenticates a listing ID but carries no shareable content, key ID, sequence, consent provenance, durable replay state or revocation ordering. Its helpers alone cannot constitute federation.
- Existing public-base-url configuration is for the local application's address; it is not a remote SSRF egress validator. Do not reuse it as one.

## Full Phase-A adaptation sequence

1. **Contract and trust:** write an experimental versioned JWS/EdDSA ADR and strict listing DTO. Specify local network, issuer peer, audience, event ID, key ID, listing ID, monotonically increasing per-listing revision, kind, issued/expiry timestamps and bounded minimal shared content. No balances, private profiles, exact location, credentials or arbitrary callback URLs. Reject unknown and duplicate JSON members as well as oversized strings/arrays, invalid UTF-8 and integer overflow. Do not confuse an SDK-valid event with an authorized one.
2. **Bilateral opt-in:** build administrator-authorized network enable/disable, explicit peer approve/revoke, fixed HTTPS endpoint, scoped publication permissions and pinned public keys. Keys are never learned from untrusted events. Rotation uses locally authorized new key IDs/validity windows; compromised keys stop immediately. Persist trust revision; recheck inside acceptance/dispatch. Dual-control hook must support policy-selected approval without granting remote administrative power.
3. **Durable store:** independent additive SQLite schema for peers/keys, listing consent, inbox events, imported listing projections/tombstones, bounded outbox, attempts and audit metadata. Use root-compatible modernc.org/sqlite v1.60.1 and Go1.26.11 if adopting that driver; do not introduce a second competing driver. No core migrations are applied now. Scope keys by local network and peer as well as event ID. Same ID/same hash is an idempotent replay; same ID/different hash is rejected equivocation. Persist acceptance and projection atomically.
4. **Ordering and retention:** signed revisions must prevent an old publish from resurrecting a revoked listing. Higher-revision revoke creates a durable tombstone; lower revisions are ignored/rejected deterministically. Equal revision with conflicting content is rejected. Retain high-water marks/tombstones beyond every accepted event's maximum lifetime plus clock skew, and define peer-reset epochs as an explicit admin action. Expiry hides content at read time and GC must not reopen replay windows. An expired publish is not authorization to delete another peer's listing.
5. **Outgoing publication:** explicit listing-owner consent plus local-admin peer policy produces a minimal snapshot into a bounded outbox. Atomically connect the approved snapshot/revision to the event in the eventual core adapter. Withdraw/depart/suspend/peer-revoke cancels queued publishes and produces appropriate revocation where authorized. No automatic public directory or DNS discovery. Sharing with one peer does not imply permission for onward relay.
6. **Transport:** construct a dedicated HTTPS client with fixed configured endpoint, redirects disabled, environment proxies disabled unless separately approved, strict TLS validation, timeouts, response/body limits and no remote asset fetching. Resolve DNS, reject every nonpublic/reserved result (including mapped IPv6, loopback, private, link-local, unspecified, multicast and CGNAT), then dial only a validated address while retaining the original TLS server name. Validate at each new connection to defeat rebinding; checking the URL once is insufficient. Use explicit injected test transports rather than a production insecure toggle. Bound retries with backoff/expiry/dead-letter outcome; HTTP delivery is at-least-once, acceptance is idempotent.
7. **Inbound boundary:** bounded raw body before parsing/signature verification, global pre-auth resource limits and per-trusted-peer rate limits, strict content type/version/type, verified key/issuer/audience/time/schema/scope, then transactional replay/projection. Stop immediately when network/peer disabled. Discard private details from logs. Do not register a production endpoint or background worker automatically.
8. **Read/admin adapters:** injected authorization interface; remote listings distinctly labeled, sanitized/escaped, with no local trade/ledger authority. Admin CLI and no-JS HTML can approve/revoke peers, inspect redacted queue state and retry explicitly. Never fetch an event's links during display. API/errors/paging and audit schema are separate module contracts until later adoption.
9. **Operational qualification:** document backup/restore, retention, quotas, key rotation/revocation, partition recovery, queue inspection, disabled behavior and failure isolation. Run dependency vulnerability checks and retain evidence. Disable removes network ingress/egress without changing core trade behavior.

## Acceptance and negative test matrix

| Requirement | Required evidence before Phase-A completion |
|---|---|
| Signed scoped publication/revocation | Two synthetic peers exchange an explicitly consented bounded listing; verify both network and peer enablement; revoke disappears immediately. Wrong scope, missing consent or suspended owner cannot publish. |
| Foreign data has no local authority | All settlement/ledger kinds and balance fields rejected; module dependency/import/API review contains no posting capability. Core fixture ledger hash unchanged around all federation operations. |
| Peer identity and schema | Wrong key, wrong type/algorithm, embedded-key spoofing, wrong issuer/audience, malformed/duplicate/unknown JSON, tampered payload, future/expired timestamps and size limits fail closed; fuzz untrusted decoder. |
| Replay/equivocation/durability | Concurrent identical deliveries yield one accepted event record; changed payload under same ID rejected; crash between inbox insert/projection rolls back; restart/retry preserves result. |
| Ordering and revocation | Revoke-before-publish, stale publish-after-revoke, equal-revision conflict, expiry/GC/replay and key rotation races cannot resurrect data. Scope identical IDs across different networks/peers safely. |
| Safe bounded egress | DNS rebinding, mixed public/private DNS, mapped IPv6, redirects, proxy variables, metadata/private addresses, TLS failure, oversized/slow responses and queue saturation tested without contacting real private targets. |
| Key/peer lifecycle | Disable/revoke during queued send and acceptance rechecked; old key rejected after window; rotation never accepted merely because a remote event requests it. |
| No-JS and isolation | Accessible peer administration/listing views, explicit remote label, no automatic link fetch, no-JS path, auth/CSRF tests; adapter remains disconnected from main. |
| Partition and capacity | Retry bounded by attempts/expiry; dedupe survives disconnect; outbox/inbox quotas and rate limits do not block local trade services. |

Run module `go test -race ./...`, `go vet ./...`, focused fuzzing and migration/restart/transport tests. Record exact results and unresolved findings; unit coverage is not security or release qualification. Manual accessibility and final core-adapter compatibility remain explicit separate evidence if not performed.

## Excluded and held

Phase B settlement, bridge accounts, FX/netting and remote balances remain excluded until a separate accounting/protocol design is approved and independently audited. Do not create placeholder ledger tables or silently narrow that approval gate. Main integration, release, production keys, external peer setup and rollout remain HOLD. A fresh implementation agent consumes this research; a different fresh audit agent reviews its result.
