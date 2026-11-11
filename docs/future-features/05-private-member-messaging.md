# FUX-105: Private member-to-member messaging

Status: **PROPOSED — future only; NOT implemented or beta-qualified**  
Priority: Medium · Recommended phase: 2 · Origin: future-features planning (2026-11-10 America/Vancouver)  
Source baseline: draft integration PR #31 (`claude/circula-beta-completion-1npjw6`, `bffcede8c7dc`).

## Goal and existing behavior

Members can arrange a trade from inside the network with retention controls and clear privacy claims.

**Current baseline:** Circula has trade proposals and notifications, not an audited private direct-message channel.

This document is an incremental, non-authoritative planning specification. It does **not** amend `docs/specs/00-conventions.md`, `docs/specs/03-domain-model.md`, or existing P0/P1 requirements until reviewed and adopted by the repository owner. Keep the current beta branch release gates unchanged.

## Functional requirements

1. Conversation is network-scoped, permitted only between active participants or for existing trade counterparties as policy specifies.
2. Support text messages, blocked users, report/abuse handling, unread indicators, optional listing/transaction references and bounded attachments deferred.
3. Default retention 30 days after last activity; network may shorten. Members can hide/delete own view but cannot silently alter another participant's copy; enforce legal-hold/dispute policy separately.
4. Start with TLS-protected server-side messages and explicitly label them accessible to the hosting operator; end-to-end encryption is a separate assurance milestone.
5. Realtime delivery can use WebSocket/SSE but persistent SQLite remains authoritative, with monotonic sequence per thread and restart consistency.
6. No message text is ever used to post a trade; all trade proposals follow the existing authenticated review/consent service.

## Proposed persistence and migration

- message_threads(id,network_id,participant1,participant2,linked_listing_id,created_at,closed_at).
- message_entries(id,thread_id,sender_id,sequence,body_cipher_or_text,created_at,expires_at,moderation_state) with bounded body and nonmutable sequence.
- message_read_positions(thread_id,member_id,last_seen_seq) and blocks/reports.

Apply additive forward-only SQLite migrations; use the existing service/store transaction boundaries and one-process-per-file guarantee. New fields may be null/default for all historical records. Specify index/retention and backup behavior before merge.

## Interface and API contract

- Proposed GET/POST /api/v1/conversations, GET /api/v1/conversations/{id}/messages (cursor), POST /api/v1/conversations/{id}/messages (idempotent).
- Proposed POST /api/v1/members/{id}/block and report path integration; member and actor checks within service.
- Optional same-origin WebSocket endpoint for member-scoped unread hints only, with origin/session checks.

All future endpoints are **proposals**, not existing routes. Before implementation, update `docs/api/openapi.yaml`, `docs/specs/07-api-contract.md`, and service contract tests. Preserve Circula's authenticated `/api/v1` contract, X-Network scoping, 64 KiB API body policy unless an explicitly reviewed media-upload exception is adopted, RFC 9457 errors, cursor paging, cookie CSRF, no-store responses and idempotency.

## Security and privacy boundaries

- No false E2EE claim; message content readable by server unless separately audited end-to-end encryption is designed and implemented.
- HTML output escapes, URL previews disabled by default, no remote images; request, message and per-member quotas.
- Use network-specific access and no-store headers; after suspension or membership removal, deny future access.

None of the above may bypass the existing transaction lifecycle, credit limits, two-entry sum-zero invariant, audit, tenant isolation or member authorization checks.

## Acceptance and negative tests

1. Cross-member and cross-network message reads/writes fail; thread participants only.
2. Concurrent send and restart preserve order without duplication; rate limits and spam reports work.
3. Messages cannot execute HTML/JS; deleted/expired content not exposed in exports or logs beyond retention policy.
4. No message event accidentally creates ledger entries or bypasses trade review.

In addition: check migration rollback, restart recovery, `go test -race ./...`, no-JavaScript essential paths, WCAG 2.2 AA automation, manual screen-reader qualification, auth/tenant threat matrix, API compatibility, secret scanning and full ledger regression.

## Upstream reuse assessment

- coder/websocket — MIT Go transport; business logic/storage remains native.
- centrifugal/centrifuge — MIT Go richer realtime transport; more complexity than MVP needs.
- simplex-chat/simplex-chat — AGPL-3.0 and separate protocol; architecture reference only, not direct copy.

**No upstream source is copied by this spec.** Before porting: pin a full upstream commit and file path, review its exact per-file license and dependencies, record ownership/attribution, run advisory scanning and build/API tests. A permissive license permits reuse subject to its conditions, but never means that the other project's business logic is safe for Circula. Circula currently has no approved product license; owner/license review must precede third-party copying or redistribution.

## Explicit non-goals

- Claims of anonymous, untraceable or end-to-end-encrypted messaging in version one.
- External federation or encrypted attachments.

## Definition of done

- [ ] Owner approves scope, decisions, privacy and license/attribution.
- [ ] Upstream dependency or reused files pinned and traced to original license.
- [ ] SQL migration + rollback/restore acceptance and least-privilege service access.
- [ ] New API contract, CLI/UI/help/wiki and user/developer docs updated.
- [ ] Positive, negative, concurrency, restart, cross-network and accessibility tests pass.
- [ ] CI, SCA, human checks and release disposition recorded.
- [ ] Feature enabled only behind explicit network/owner configuration; no silent rollout to beta.
