# Independent discovery-federation audit — 2026-11-16

Scope: the isolated `future/federation` module, experimental Phase-A discovery protocol and reference adapter. Review started at `8ddfd51`. This review does not approve production protocol/key custody, integrate a feature, or approve Phase-B settlement. Integration remains **HOLD**.

## Findings fixed

1. **Withdrawal saturation could retain consent on departure.** A reserve of 100 rows could not withdraw 1,000 ordinary publications; the all-or-nothing cleanup would roll back with consent still enabled. Automatic withdrawals now coalesce by publication revision and do not allocate for a never-published listing. The lifetime ceiling is 2,000 total rows, reserving one safety withdrawal for each possible ordinary publication. A 1,000-publication departure and repeat departure pass. An index bounds per-listing lookup cost. Ordinary creation still stops at 1,000 total existing rows; safety metadata is never evicted to make room.
2. **Silent SQLite writes could be acknowledged.** `RAISE(IGNORE)` can return no SQL error and affect zero rows. Critical inbox/projection/outbox reservation/sealing/sequence and policy/audit inserts now require exactly one affected row; otherwise the transaction rolls back. Regression tests show neither an accepted inbox journal without a projection nor a network send without a durable lease can succeed.
3. **Consent could allocate rows for arbitrary unapproved peers.** Enabling consent now requires the configured enabled peer/network and a maximum of 1,000 consent rows per peer. Opting out absent consent allocates nothing. This closes an unbounded remote-ID allocation path; repeated allow/deny for an unpublished listing consumes no withdrawal reserve.
4. **`IF NOT EXISTS` concealed incompatible prior schemas.** Migration now validates existing candidate table/index declarations and object types before accepting them, in the same transaction. A table lacking required uniqueness and a same-name view fail closed, without partially creating the rest of the module schema. This first candidate does not silently upgrade an unknown prior schema.
5. **Corrupt persisted data was insufficiently checked at egress/read.** Remote projection JSON must match the exact allowlisted encoding and valid content/identity/revision shape. Outgoing records are checked against their protocol ID, issuer/audience, listing, revision, kind, expiry and attempt bounds before sending, including revocations. A mismatched stored event cannot reach the delivery interface. Queue returns an empty wire on failure.
6. **Strict profile gaps.** All compact-JWS segments require canonical base64url encodings and a 64-byte Ed25519 signature. Fixed HTTPS endpoints reject even an empty query marker. Existing SSRF, DNS pinning, redirect and response bounds remain enforced.
7. **Deletion required a still-readable source.** Added the admin/system `WithdrawListing` lifecycle hook and CLI seam. It can cancel publications and enqueue withdrawals after a source has been deleted, without a signer, while preserving tenant isolation, repeat safety and atomic audit. It is a standalone adapter seam; the actual core deletion transaction is not wired.

## Requirement and evidence matrix

| Requirement | Evidence reviewed/reproduced |
|---|---|
| Signed exact allowlist, no settlement authority | `TestV2ProtocolFailClosed`, `TestSignedButMalformedPayloadAndHeaders`, `TestAuditCanonicalBase64Required`, wire fuzz; no core/ledger imports or operations |
| Configured identity, active trust and bounded lifetime | Issuer/audience/kid/algorithm/type/expiry negative fixtures; rotation/revocation tests; local configured scope cannot be selected by the event |
| Durable idempotency and replay ordering | Concurrent inbox/reopen, trigger-abort and trigger-ignore rollback, ID/revision equivocation, stale-after-revoke and maintenance high-water tests |
| Consent, tenant privacy and data minimization | Fresh transaction callbacks; explicit publish consent; source mismatch tests; unapproved-peer and consent-cap regressions; no member/URL/balance fields in the wire DTO |
| Withdrawal despite signer/source outage | Existing signer-outage test now first queues a real publication; 1,000-publication departure, never-published toggles, deleted-source withdrawal and repeat tests |
| Delivery lifecycle | Partition/backoff/lease tests; ignored reservation cannot send; corrupt stored event binding; no network delivery inside SQLite writer transaction |
| Bounded capacity | Inbox 10,000, peer 100, consent 1,000, owner 120/hour and ordinary outbox 1,000 boundary fixtures; coalesced safety reserve at maximum ordinary capacity |
| Schema, corruption and recovery | Existing consistent closed-DB restore; incompatible uniqueness schema/view rollback; corrupted projection and outgoing record rejection |
| SSRF and transport | Private/mixed/reserved address tests; actual validated-IP dial with original TLS hostname; proxies/redirects disabled; synthetic HTTP timeout/oversize/error tests; fixed endpoint profile |
| Interfaces | Fresh host-auth/CSRF/no-store/body-bound member/admin fixtures; escaped no-JS rendering; CLI reuses Store, including lifecycle withdrawal |
| Dependencies | Exact pinned module graph and notices inspected; `go mod verify`; independent `govulncheck` pass |

## Observed final validation

Toolchain Go 1.26.11 linux/amd64.

- `GOGC=20 GOMEMLIMIT=450MiB GOMAXPROCS=1 go test -p 1 -race -cover -timeout 90s ./...`: PASS, **71.3% statements**, 17.962s.
- Constrained `go vet -p 1 ./...`: PASS.
- `go mod verify`: all modules verified.
- `go test -p 1 -run '^$' -fuzz FuzzDiscoveryWire -fuzztime=3s -parallel=1 -timeout 45s ./...`: PASS, **9,504 executions** after the audit changes.
- `govulncheck ./...` using the installed v1.1.4 tool: **No vulnerabilities found**. This is a point-in-time advisory result.

An accidental root-scoped test invocation exposed the old baseline's unrelated `cmd/circula/auditanchor.go:189:12: undefined: os` and a constrained core API scale-test timeout. No core files were changed. These results do not qualify or regress the isolated module and are not represented as core acceptance.

## Remaining gates

Actual Circula transaction/auth/suspension/deletion/departure adapters and unchanged-ledger regression are still required. The host signing-key provider must supply an authorized current key whose remote trust validity covers the event lifetime; production key custody/distribution is not implemented by this callback. Live external-peer TLS/protocol interoperability, reviewed final protocol ADR, integrated ingress/rate-limit deployment, and manual browser/screen-reader/WCAG qualification remain open. The strict candidate Go-encoding profile has not been demonstrated with an independent external implementation.

The candidate deliberately retains replay/high-water/request metadata and stops at lifetime quotas. An operational peer epoch/reset or compaction design needs a separate reviewed protocol; this audit does not introduce an unsafe reset. Outbox reservation defines the start of an in-flight send, which a later local revocation cannot retract. Already shared content can remain at a partitioned peer until expiry. No assertion of full feature completion, production readiness or cryptographic certification is made.
