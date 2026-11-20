# Circula read-only feature review: notification transports and federation

Review date: 2026-11-19 America/Los_Angeles (2026-11-20 UTC). This is a narrow static review of two isolated future modules. Neither is enabled in the reviewed Core runtime. Both manifests explicitly retain `enabled: false` and `integration: HOLD`.

## Pinned scope and evidence

| Module | Reviewed commit | Local remote-tracking branch |
|---|---|---|
| FUX-104 notifications, including transport repair | `f831c45de9137b20abe3d31590433bda6348112b` | `origin/master/notification-transport-tests-20261119` |
| FNET-209 recovered discovery federation | `bb8d113d3a8e80597f611417410e7d78151247e2` | `origin/master/recover-federation-20261119` |

Source roots: [notifications](https://github.com/djshellshoxxx/circula/tree/f831c45de9137b20abe3d31590433bda6348112b/future/notifications) and [federation](https://github.com/djshellshoxxx/circula/tree/bb8d113d3a8e80597f611417410e7d78151247e2/future/federation).

All source was read directly from these Git objects. References below are module-relative and refer to the corresponding pinned revision. Searches of each feature branch's `cmd`, `internal`, and root `go.mod` found no imports/references to either feature module. The nested-module/disabled state was also confirmed in the manifests and implementation seams. This does not certify a deployment configuration; no deployment was contacted.

No source changes, activation, tests, fuzzing, network sends, exploit payloads, or new vulnerability reproductions were performed. Existing test source and historical qualification reports were inspected. Their reported passes remain attributed historical evidence, not fresh executions by this reviewer. Messaging/Windows was not included in this follow-up.

## Disposition

Two notification items need resolution before production integration. The first is a source-confirmed lease/fencing defect with low security severity and operational consequences. The second is a confirmed metadata exposure for privacy-policy review, also low severity. They are **pre-integration risks**, not active Core exposures.

| ID | Item | Classification | Confidence |
|---|---|---|---|
| FTR-N01 | Dispatch lease ownership is not enforced for every target send or target-result write | Low security severity; delivery reliability/integrity defect | High for missing invariant; not dynamically reproduced |
| FTR-N02 | Email action links carry readable internal network/member identifiers | Low privacy-minimization risk; integration policy decision required | High for encoding/disclosure behavior |

No additional federation signature-verification, cross-peer trust, or destination-validation bypass was established in the inspected source. Its existing production integration and operational gates remain material.

## FTR-N01 — a notification worker can continue fan-out after its lease has moved

### Source evidence

`future/notifications/dispatch.go:32–38,56–69` defaults each worker to a batch of 20, a two-minute lease, and eight abandoned claims. The stated lease constraint refers to the longest send. `claim` gives the whole batch a common lease deadline (`dispatch.go:163–190`); `RunOnce` processes the returned rows sequentially (`dispatch.go:111–127`). Expired in-flight leases are eligible for another claim (`dispatch.go:163–165`).

`authorize` checks status and owner once, before the target list is processed, but does not read `lease_until` (`dispatch.go:324–337`). The subsequent fan-out iterates multiple targets (`dispatch.go:254–271`). Each sender call gets its own 45-second timeout (`dispatch.go:538–540`). The ordinary active-subscription cap is ten (`service.go:75–76,106–108`; `targets.go:122–129`). Thus the supported fan-out and batch lifetime is not bounded by one send's timeout or by the default lease duration.

Immediately before each send, `stillAllowed` checks current preference and target revocation/bounce state (`dispatch.go:506–535`). It does **not** check current delivery owner, lease generation, or lease expiry. An old worker can therefore continue starting target sends after another worker has reclaimed the delivery. This is distinct from the unavoidable case where one provider request was already in flight when a process failed.

The terminal delivery update is guarded by current owner and state (`dispatch.go:709–735`), which is useful. However, `recordTarget` upserts each target's result without the same guard and ignores the update error (`dispatch.go:578–585`). A stale worker can replace a newer target status, undermining the successful-target suppression used at `dispatch.go:467–489`.

### Preconditions and impact

This requires multiple active dispatchers, overlapping dispatcher invocations, or another legitimate claim after a lease expires while an earlier invocation is still processing its targets. Slow but bounded provider/adapter calls can make the window relevant. It is not an unauthenticated route or a claimed source of ledger writes.

Consequences include duplicate notifications, stale per-target outcomes, unnecessary retries, inaccurate delivery accounting, and provider capacity/cost waste. The module correctly describes transport as at-least-once (`delivery.go:84–91`); this finding does not demand impossible exactly-once SMTP. It concerns avoidable new sends and state writes by a worker that no longer owns the delivery.

### Repair direction

Use a per-claim generation/fencing token, and validate it before starting each target operation and before storing each target result. Either claim work close to actual processing, renew the lease under the same ownership guard, or otherwise ensure the entire authorized fan-out fits a defined ownership interval. Stop remaining work on lost ownership or expired delivery. Prevent stale results from downgrading already recorded successful outcomes. Do not hold the SQLite writer transaction open across provider I/O.

Safe acceptance criteria: ordinary local fixtures with synthetic clocks/senders should show that a transferred lease prevents the prior claimant from starting additional targets or replacing a newer result; successful targets remain suppressed on retry; one already-started provider operation may still have the documented at-least-once outcome. Validate default configuration as well as custom lease/send limits. Existing inspected fan-out, retry, opt-out, and expiry tests do not establish this ownership invariant.

## FTR-N02 — signed email action tokens expose internal identity metadata

### Source evidence

`future/notifications/secrets.go:314–320` defines token fields for purpose, network, member, channel, target, and optional expiry. `signToken` JSON-serializes that structure, Base64URL-encodes the bytes, and authenticates the encoded body with an HMAC (`secrets.go:323–337`). This protects integrity but provides no confidentiality for those fields.

Confirmation tokens include network, member, and email-target IDs (`targets.go:419–420`). Unsubscribe tokens include network/member/channel and intentionally have no expiry (`targets.go:459–467`). The dispatcher embeds these URLs in verification and generic activity emails (`dispatch.go:541–551`). SMTP also includes the unsubscribe URL in the relevant message header (`smtpsend/smtpsend.go:158–162`). Therefore a recipient or mail provider that possesses the normal message also receives readable internal identity metadata through the link, even though the visible message text is generic.

The existing `TestEmailDoubleOptInThenGenericDelivery` checks for literal fixture identifiers in the subject/body (`dispatch_test.go:65–68`). An encoded identifier does not satisfy that literal search, so this check is not evidence that the internal identifiers are absent from the complete message.

### Impact and qualification

The additional exposure is stable application network/member identity, correlated with the delivery address; confirmation links additionally disclose target identity. This is a data-minimization issue. It does not show token forgery, plaintext trade amounts, disclosure of signing keys, or an authentication bypass. SMTP necessarily knows its recipient, and some deployments may accept pseudonymous application identifiers as part of the provider's data boundary. That decision should be explicit, particularly because the feature's stated objective is to keep sensitive details inside the authenticated application (`SPEC-SNAPSHOT.md`, functional requirement 4).

### Repair or disposition

If internal identities must remain private from transport providers and forwarded-message recipients, use opaque random action references backed by scoped server state, or a reviewed authenticated-encryption token format. Keep purpose/channel/target restrictions, double opt-in, replay/idempotency behavior, old-message unsubscribe usability, revocation, and key-rotation handling. If the owner accepts this metadata disclosure, record it in the privacy/data-flow decision and narrow test/documentation claims accordingly.

Safe acceptance criteria: inspect the complete synthetic mail representation, including headers and action URLs, for recoverable internal identities under the selected policy. Preserve valid unsubscribe/confirmation and wrong-purpose/foreign-target rejection. No real email, token extraction tool, or live recipient is needed.

## Notification controls observed

The previously reported delimiter-AAD and malformed-UTF-8 binding repairs are present. `targets.go:35–57` uses a distinct, length-prefixed authenticated-data format for tuples containing the former delimiter while preserving unambiguous legacy tuples; opening and rotation use the same derivation. `service.go:24–33`, `delivery.go:68–81`, and `secrets.go:323–330` reject malformed UTF-8 in the relevant identity paths. These should not be re-reported as unfixed.

Outbound push destinations require an explicitly allowlisted DNS host, HTTPS, and the allowed port (`egress.go:48–110`). The client validates every DNS result, refuses non-public addresses, dials checked IPs, preserves TLS hostname verification, disables inherited proxies and redirects, and bounds connection/request/header work (`egress.go:181–255`). `pushsend.SendPush` reapplies endpoint policy, caps payload size, passes the explicit HTTP client into the upstream sender, and limits retained response content (`pushsend/pushsend.go:75–105`). The transport's known successful-response typed-nil error and Retry-After overflow repairs are visible at lines 131–174.

SMTP host, sender, credentials, and trust settings are operator inputs, distinct from member email targets. TLS verification is mandatory for STARTTLS and implicit modes; the custom implicit dialer completes verified TLS before returning its connection (`smtpsend/smtpsend.go:89–132`). Member email normalization and three-per-day verification quota appear in `targets.go:262–289,326–332`. Verification is required before ordinary email delivery, and preference/target revocation is rechecked before sending. Push session validity is rechecked when the subscription is session-bound.

Targets are encrypted with AES-GCM and scoped associated data; endpoint/address indexes use keyed hashes (`secrets.go:52–133,296–305`). Secret-file reads require regular files, check permission bits, and bound size (`secrets.go:237–254`). `Secret` formatting redacts SMTP/VAPID credentials, and `SendError` exposes fixed classifications instead of provider-supplied addresses/tokens (`transport.go:71–108`). This is not an audit of every host logging path or operating-system ACL model.

Production construction must actually supply the checked egress client. `pushsend.Config.Client` is an injected interface: its non-nil check cannot establish that arbitrary host-provided implementations apply the documented network policy. This is an explicit trusted-adapter obligation, not a demonstrated member-controlled bypass.

## Federation: persisted identity, protocol and transport review

### Peer identity stays under local administrative control

`future/federation/store.go:131–165` validates the local/remote pairing, endpoint and Ed25519 key shape; requires transactional administrator authorization; scopes the peer and key records by `(local, remote)`; and refuses replacement of an existing key ID. Rotation needs a new key ID. `RevokePeer` disables the peer, cancels pending/sending outbox state, and revokes its stored keys (`store.go:167–185`). Network/peer defaults and reads require enabled configuration.

Ingress uses host-configured `Local`/`Remote`, not identities selected by an inbound event (`interface.go:25–28,64`). Receive obtains the key from the persisted local/remote/key-ID tuple, checks revocation and validity, and requires the signed event's lifetime to fit that key's validity (`store.go:284–292`). `WireKeyID` is a bounded untrusted selector and does not itself confer trust (`protocol.go:100–131`). No event-provided key download, trust-on-first-use, or automatically accepted rotation was identified.

An explicit pre-integration policy remains necessary for peer continuity. `ApprovePeer` can update the endpoint of an existing remote identifier while adding a new key (`store.go:150–161`). Owner consent and queued work are indexed by that remote identifier, not by a separate administrative trust-generation record. The host's approval procedure must distinguish a verified rotation/move of the same peer from reassigning an old peer label to a new organization. This is not a finding that an unprivileged peer can redirect a destination; it identifies where the missing production identity-distribution procedure must preserve the intended consent boundary.

### Signature, replay and content boundaries

The recovered runtime uses compact JWS EdDSA with a narrowly specified protected header, canonical Base64URL, exact payload encoding, bounded identifiers/text and Ed25519 signature length (`protocol.go:65–155`). It checks audience, issuer, publish/revoke kind, revision, expiry and clock tolerance. The public DTO is limited to listing title, description, category and area. It has no remote financial operation, balance, owner profile, or dereferenced content URL.

Replay hash, per-listing revision/content comparison, inbox record and projection update share the write transaction (`store.go:310–345`). Conflicting event IDs and equal-revision content changes fail. The inbox lifetime cap fails closed instead of discarding replay state. Browse joins current enabled network/peer and non-revoked key state and validates stored projected content before returning it (`store.go:364–395`). Legacy v1 helpers remain separate; the inspected v2 ingress has no fallback to them.

### Outbound work and withdrawal

Queue validates fresh owner/source/consent and creates the signed immutable outbox event in one transaction (`outbox.go:14–119`). Dispatch rechecks peer configuration, event-to-row binding, current source/owner/revision/consent, then reserves a bounded attempt before external I/O (`outbox.go:130–227`). Its result update checks state and attempt generation (`outbox.go:228–246`). The documented send-start boundary means an already-started request cannot be recalled; this review does not reinterpret that known boundary as a new flaw.

Source-independent withdrawal, departure cancellation, coalescing, and reserved withdrawal capacity are present in `operations.go`. Maintenance preserves replay/sequence/high-water metadata while clearing expired content. Lifetime outbox/inbox capacity and safe peer-epoch/compaction remain a documented operational gate. Deleting safety metadata merely to recover space would invalidate the current replay design.

### Network egress and ingress bounds

`transport.go:21–32` validates a fixed HTTPS endpoint, rejects query/userinfo/fragment/opaque forms, constrains port and path, and rejects disallowed literal destinations. Before dialing, `HTTPSDelivery` checks the configured hostname and every resolved address, caps DNS answers, dials a checked address directly, keeps normal TLS verification, and disables redirects/proxies (`transport.go:61–93`). Request lifetime, response headers and body are bounded (`transport.go:75–113`). Content is sent only to the configured destination; received listing content does not select a fetch URL.

Ingress has method/media/body checks and a pre-receive request-rate gate (`interface.go:31–68`); the store additionally meters accepted peer traffic. The existing integrated deployment/rate policy remains open. Member/admin form parsing, query bounds, no-store output, escaped rendering, and same-actor rechecks after token callbacks are present in the inspected HTTP adapters, consistent with the earlier corrected-delta report.

## Prior-audit reconciliation and remaining gates

The notifications `VALIDATION.md` and `TRANSPORT-QUALIFICATION-20261119.md` distinguish identity repairs, real transport tests, native workflow qualification, and absent Core integration. The federation `AUDIT.md`, `SECURITY-AUDIT.md`, `RECOVERY-20261119.md`, `INTEGRATION.md`, and protocol ADR distinguish the recovered persistent implementation from its older signing-only prototype. Fixed findings from those records were not counted again.

Federation's top-level README still describes an older prototype. `RECOVERY-20261119.md` explicitly explains its preserved bytes and separately owned README handoff. Use the recovered manifest and recovery/integration documents for scope. This known documentation discrepancy is an integration/handoff item, not evidence that the present code lacks its implemented durable store or egress controls.

Neither module yet establishes actual Core transaction coupling, current authorization under every lifecycle transition, production credential/key custody, safe backup/restore across retained replay state, live provider/peer interoperability, or all browser/accessibility behavior. Host interfaces and standalone tests cannot substitute for those concrete integration properties. Keep notification failure outside ledger retries, and couple federation withdrawal with Core departure/suspension/deletion through the actual transaction boundary before activation.

## Exact coverage and limits

Notification production source read: `egress.go`, `pushsend/pushsend.go`, `smtpsend/smtpsend.go`, `dispatch.go`, `transport.go`, `secrets.go`, `service.go`, `targets.go`, `outbox.go`, `prefs.go`, `delivery.go`, and the initial SQL schema. Existing dispatch tests and transport test sections/names were inspected for regression coverage. This was not an independent review of every upstream dependency, metrics path, or provider behavior.

Federation production source read: `protocol.go`, `envelope.go`, `store.go`, `transport.go`, `outbox.go`, `members.go`, and the security-relevant HTTP/CLI and lifecycle/withdrawal sections of `interface.go` and `operations.go`. Existing protocol/interface test references and audit/recovery notes were inspected. This was not a new cryptographic proof, exhaustive protocol interoperability study, or full re-execution of the archived audit.

No ordinary or adversarial tests ran in this follow-up. No test-pass or native-platform qualification is claimed by this worker. The only deliverable change is this review note; application source and remote branches were not modified.
