# FNET-209: Opt-in cross-network federation

Status: **PROPOSED — future only; NOT implemented or beta-qualified**  
Priority: Future · Recommended phase: 4 · Origin: future-features planning (2026-11-10 America/Vancouver)  
Source baseline: draft integration PR #31 (`claude/circula-beta-completion-1npjw6`, `bffcede8c7dc`).

## Goal and existing behavior

Independent Circula networks exchange signed listings and optional inter-network trade proposals without allowing foreign peers to mutate a local ledger.

**Current baseline:** Network isolation is a hard invariant. Cross-network trade/credit posting is explicitly deferred by current MVP specs.

This document is an incremental, non-authoritative planning specification. It does **not** amend `docs/specs/00-conventions.md`, `docs/specs/03-domain-model.md`, or existing P0/P1 requirements until reviewed and adopted by the repository owner. Keep the current beta branch release gates unchanged.

## Functional requirements

1. Phase A discovery-only: publish selected listings via signed versioned envelopes, bilateral trust agreements, scoped sharing, expiry/revocation and per-peer rate limits.
2. Phase B optional inter-network settlement only after a separate approved accounting model defines bridge accounts, credit exposure limits, FX/unit rules if needed, disputes, netting and rollback/compensation.
3. Each network retains independent administration, revocation and data retention control. A network never imports remote balances as authoritative local balances.
4. Remote events are untrusted: verify network identity, signatures, timestamp tolerance, event IDs, schemas, and content permissions.
5. Federation must be opt-in per peer and per network; no public directory or DNS discovery required by default.
6. Use bounded outboxes/inboxes with idempotent message processing and deterministic replay/rotation/revocation procedures.

## Proposed persistence and migration

- federation_peers(local_network_id,peer_id,public_key,endpoint,scope,enabled,capabilities,created_at,revoked_at).
- federation_events(peer_id,event_id,kind,body_hash,seen_at,processing_state) UNIQUE(peer_id,event_id).
- Optional bridge accounts and settlement obligations reserved for separately approved Phase B ADR; do not create them in discovery-only milestone.

Apply additive forward-only SQLite migrations; use the existing service/store transaction boundaries and one-process-per-file guarantee. New fields may be null/default for all historical records. Specify index/retention and backup behavior before merge.

## Interface and API contract

- Separate peer-facing federation namespace, e.g. /federation/v1/events; versioned, signed and unrelated to member session auth.
- Member-facing browse API must label remote listings and never allow remote events to call local ledger posting directly.
- Administrative peer approve/revoke routes require dual control if selected by policy.

All future endpoints are **proposals**, not existing routes. Before implementation, update `docs/api/openapi.yaml`, `docs/specs/07-api-contract.md`, and service contract tests. Preserve Circula's authenticated `/api/v1` contract, X-Network scoping, 64 KiB API body policy unless an explicitly reviewed media-upload exception is adopted, RFC 9457 errors, cursor paging, cookie CSRF, no-store responses and idempotency.

## Security and privacy boundaries

- mTLS or verified signing keys, SSRF-safe peer endpoints, no private address targets and bounded payload/fetch behavior.
- No weakening of LED-1 through LED-7, cross-tenant isolation or current tenant export rules.
- Simulate adversarial remote peer, replay, equivocation, key rotation, partition and forged inter-network settlement.

None of the above may bypass the existing transaction lifecycle, credit limits, two-entry sum-zero invariant, audit, tenant isolation or member authorization checks.

## Acceptance and negative tests

1. Discovery-only peer can publish and revoke selected listing; no ledger data changes.
2. Foreign event cannot authorize local trade or read member private data.
3. Duplicate/out-of-order events and peer restart preserve exactly one accepted event record.
4. Any money movement waits for explicitly approved Phase B accounting/protocol tests and independent audit.

In addition: check migration rollback, restart recovery, `go test -race ./...`, no-JavaScript essential paths, WCAG 2.2 AA automation, manual screen-reader qualification, auth/tenant threat matrix, API compatibility, secret scanning and full ledger regression.

## Upstream reuse assessment

- cloudevents/sdk-go — Apache-2.0 typed event envelopes and transport helpers, not settlement.
- go-ap/activitypub — MIT vocabulary for discoverable social content, not a ledger protocol.
- Medusa/Decidim are patterns only; neither supplies cross-network mutual-credit accounting.

**No upstream source is copied by this spec.** Before porting: pin a full upstream commit and file path, review its exact per-file license and dependencies, record ownership/attribution, run advisory scanning and build/API tests. A permissive license permits reuse subject to its conditions, but never means that the other project's business logic is safe for Circula. Circula currently has no approved product license; owner/license review must precede third-party copying or redistribution.

## Explicit non-goals

- Immediately fungible credits between arbitrary networks; autonomous exchange rates; remote administrative power.

## Definition of done

- [ ] Owner approves scope, decisions, privacy and license/attribution.
- [ ] Upstream dependency or reused files pinned and traced to original license.
- [ ] SQL migration + rollback/restore acceptance and least-privilege service access.
- [ ] New API contract, CLI/UI/help/wiki and user/developer docs updated.
- [ ] Positive, negative, concurrency, restart, cross-network and accessibility tests pass.
- [ ] CI, SCA, human checks and release disposition recorded.
- [ ] Feature enabled only behind explicit network/owner configuration; no silent rollout to beta.
